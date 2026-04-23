package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type openAIImagesCompatResponse struct {
	Created      int64            `json:"created"`
	Background   string           `json:"background,omitempty"`
	Data         []map[string]any `json:"data,omitempty"`
	OutputFormat string           `json:"output_format,omitempty"`
	Quality      string           `json:"quality,omitempty"`
	Size         string           `json:"size,omitempty"`
	Usage        map[string]any   `json:"usage,omitempty"`
}

func (s *OpenAIGatewayService) ForwardCompatImages(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	contentType string,
	action string,
	displayModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	normalizedRequest, err := NormalizeOpenAIImageRequest(body, contentType, action)
	if err != nil {
		return nil, newOpenAIImageRequestError("image_request_invalid", err.Error())
	}
	normalizedRequest.DisplayModelID = firstNonEmptyString(strings.TrimSpace(displayModel), normalizedRequest.DisplayModelID, OpenAICompatImageTargetModel)
	normalizedRequest.TargetModelID = OpenAICompatImageTargetModel
	capabilityProfile, err := ValidateOpenAIImageCapabilities(normalizedRequest, OpenAIImageProtocolModeCompat, normalizedRequest.TargetModelID)
	if err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		ctx := EnsureRequestMetadata(c.Request.Context())
		SetImageCapabilityProfileMetadata(ctx, capabilityProfile.ID)
		c.Request = c.Request.WithContext(ctx)
	}
	SetOpenAIImageNormalizedTracePayload(c, "openai_compat_images_normalized_request", normalizedRequest, capabilityProfile.ID)

	responsesBody, err := BuildOpenAIImageCompatResponsesBody(normalizedRequest, OpenAICompatImageHostModel, normalizedRequest.TargetModelID)
	if err != nil {
		return nil, err
	}
	if isChatGPTOpenAIOAuthAccount(account) {
		responsesBody, err = applyCodexOAuthTransformToJSON(responsesBody, false, false)
		if err != nil {
			return nil, err
		}
	}
	setOpsUpstreamRequestBody(c, responsesBody)
	if normalizedRequest.Stream {
		return s.forwardCompatImagesStream(ctx, c, account, normalizedRequest, responsesBody, startTime)
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	upstreamReq, err := s.buildUpstreamRequest(ctx, c, account, responsesBody, token, false, "", false)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		return s.handleCompatErrorResponse(resp, c, account, func(c *gin.Context, statusCode int, errType, message, _ string) {
			c.JSON(statusCode, gin.H{"error": gin.H{"type": errType, "message": message}})
		})
	}

	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	upstreamBody, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		return nil, err
	}
	SetOpsTraceUpstreamResponse(c, "openai_compat_images_upstream_response", upstreamBody, resp.Header.Get("Content-Type"), false)
	finalBody, usage, err := finalizeOpenAICompatImageResponseBody(upstreamBody, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	imagesResponse, imageCount := buildOpenAIImagesCompatResponse(finalBody, normalizedRequest)
	if c != nil && c.Request != nil {
		SetImageOutputCountMetadata(c.Request.Context(), imageCount)
	}
	responseBody, err := json.Marshal(imagesResponse)
	if err != nil {
		return nil, fmt.Errorf("marshal compat images response: %w", err)
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	SetOpsTraceGatewayResponse(c, "openai_compat_images_gateway_response", responseBody, "application/json", false)
	c.Data(resp.StatusCode, "application/json", responseBody)

	return &OpenAIForwardResult{
		RequestID:     resp.Header.Get("x-request-id"),
		Usage:         usage,
		Model:         normalizedRequest.DisplayModelID,
		BillingModel:  normalizedRequest.TargetModelID,
		UpstreamModel: normalizedRequest.TargetModelID,
		ImageCount:    imageCount,
		ImageSize:     ResolveOpenAIImageSizeTier(normalizedRequest.Size),
		MediaType:     "image",
		Duration:      time.Since(startTime),
	}, nil
}

func finalizeOpenAICompatImageResponseBody(body []byte, contentType string) ([]byte, OpenAIUsage, error) {
	if isEventStreamResponse(http.Header{"Content-Type": []string{contentType}}) || looksLikeEventStreamBody(body) {
		bodyText := string(body)
		finalResponse, ok := extractCodexFinalResponse(bodyText)
		if !ok {
			return nil, OpenAIUsage{}, fmt.Errorf("failed to parse compat image response")
		}
		if supplemented, rebuilt := supplementOpenAIResponseOutputFromSSE(finalResponse, bodyText); rebuilt {
			finalResponse = supplemented
		}
		usage, _ := extractOpenAIUsageFromJSONBytes(finalResponse)
		return finalResponse, usage, nil
	}
	if !gjson.ValidBytes(body) {
		return nil, OpenAIUsage{}, fmt.Errorf("failed to parse compat image response")
	}
	usage, _ := extractOpenAIUsageFromJSONBytes(body)
	return body, usage, nil
}

func buildOpenAIImagesCompatResponse(body []byte, request *NormalizedImageRequest) (*openAIImagesCompatResponse, int) {
	response := &openAIImagesCompatResponse{
		Created:      time.Now().Unix(),
		Background:   strings.TrimSpace(request.Background),
		OutputFormat: strings.TrimSpace(request.OutputFormat),
		Quality:      strings.TrimSpace(request.Quality),
		Size:         strings.TrimSpace(request.Size),
	}
	if usageValue := gjson.GetBytes(body, "usage"); usageValue.Exists() && usageValue.Type == gjson.JSON {
		var usage map[string]any
		if err := json.Unmarshal([]byte(usageValue.Raw), &usage); err == nil && usage != nil {
			response.Usage = usage
		}
	}
	images := extractOpenAIResponsesOutputImages(body)
	response.Data = make([]map[string]any, 0, len(images))
	for _, imageURL := range images {
		item := map[string]any{}
		if b64 := stripDataURLBase64(imageURL); b64 != "" {
			item["b64_json"] = b64
		} else if strings.TrimSpace(imageURL) != "" {
			item["url"] = strings.TrimSpace(imageURL)
		}
		if len(item) > 0 {
			response.Data = append(response.Data, item)
		}
	}
	if response.OutputFormat == "" && len(images) > 0 {
		response.OutputFormat = detectImageOutputFormat(images[0])
	}
	return response, len(response.Data)
}

func extractOpenAIResponsesOutputImages(body []byte) []string {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return nil
	}
	result := make([]string, 0, 4)
	for _, output := range gjson.GetBytes(body, "output").Array() {
		for _, content := range output.Get("content").Array() {
			if strings.TrimSpace(content.Get("type").String()) != "output_image" {
				continue
			}
			if imageURL := strings.TrimSpace(content.Get("image_url").String()); imageURL != "" {
				result = append(result, imageURL)
			}
		}
	}
	return result
}

func stripDataURLBase64(value string) string {
	trimmed := strings.TrimSpace(value)
	index := strings.Index(trimmed, ",")
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:image/") || index <= 0 {
		return ""
	}
	return trimmed[index+1:]
}

func detectImageOutputFormat(value string) string {
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(lowerValue, "data:image/png"):
		return "png"
	case strings.HasPrefix(lowerValue, "data:image/webp"):
		return "webp"
	case strings.HasPrefix(lowerValue, "data:image/jpeg"):
		return "jpeg"
	default:
		return ""
	}
}
