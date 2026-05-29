package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

func getContextInt64(c *gin.Context, key string) (int64, bool) {
	if c == nil || key == "" {
		return 0, false
	}
	v, ok := c.Get(key)
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case int32:
		return int64(t), true
	case float64:
		return int64(t), true
	default:
		return 0, false
	}
}

func setResponsesImagegenCompatTracePayload(
	c *gin.Context,
	model string,
	contentType string,
	metadata service.OpenAIResponsesCompatMetadata,
	tool map[string]any,
) {
	if c == nil {
		return
	}
	compatPayload := map[string]any{
		"model":        strings.TrimSpace(model),
		"content_type": strings.TrimSpace(contentType),
		"compat": map[string]any{
			"enabled":               metadata.Enabled,
			"rejected":              metadata.Rejected,
			"source":                strings.TrimSpace(metadata.Source),
			"source_guess":          strings.TrimSpace(metadata.SourceGuess),
			"reject_code":           strings.TrimSpace(metadata.RejectCode),
			"reference_image_count": metadata.ReferenceImageCount,
			"normalized":            metadata.ReferenceImagesNormalized,
			"size":                  strings.TrimSpace(metadata.ImageGenerationSize),
		},
	}
	if len(tool) > 0 {
		compatPayload["tools"] = []any{tool}
		compatPayload["tool_choice"] = map[string]any{"type": "image_generation"}
	}
	service.SetOpsTraceNormalizedRequest(c, "openai_responses_imagegen_compat", compatPayload)
}

func detectOpenAIResponsesCompatRequestModel(body []byte, contentType string) string {
	model, err := service.DetectOpenAIImageRequestModel(body, contentType)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(model)
}

func detectOpenAIResponsesCompatSourceGuess(body []byte, contentType string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/form-data") {
		return service.OpenAIResponsesImagegenCompatSourceMultipart
	}
	if !gjson.ValidBytes(body) {
		return service.OpenAIResponsesImagegenCompatSourceJSONShorthand
	}
	input := gjson.GetBytes(body, "input")
	if input.IsArray() {
		return service.OpenAIResponsesImagegenCompatSourceStructured
	}
	return service.OpenAIResponsesImagegenCompatSourceJSONShorthand
}

func openAIResponsesCompatRequestID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return strings.TrimSpace(requestID)
	}
	return strings.TrimSpace(c.GetHeader("X-Request-ID"))
}

func openAIResponsesCompatCorrelationID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if correlationID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(correlationID) != "" {
		return strings.TrimSpace(correlationID)
	}
	return ""
}

func ensureOpenAIPoolModeSessionHash(sessionHash string, account *service.Account) string {
	if sessionHash != "" || account == nil || !account.IsPoolMode() {
		return sessionHash
	}
	// 为当前请求生成一次性粘性会话键，确保同账号重试不会重新负载均衡到其他账号。
	return "openai-pool-retry-" + uuid.NewString()
}
