package service

import (
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func buildGeminiPassthroughForwardResult(input GeminiPublicPassthroughInput, requestedModel string, headers http.Header, body []byte, duration time.Duration, stream bool) *ForwardResult {
	usage := extractOpenAICompatUsage(body)
	if parsed := extractGeminiUsage(body); parsed != nil {
		usage = *parsed
	}
	requestID := strings.TrimSpace(firstNonEmptyString(
		headers.Get("x-request-id"),
		headers.Get("X-Request-Id"),
		gjson.GetBytes(body, "responseId").String(),
		gjson.GetBytes(body, "id").String(),
		gjson.GetBytes(body, "name").String(),
	))
	upstreamModel := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "modelVersion").String(),
		gjson.GetBytes(body, "model").String(),
		requestedModel,
	))
	mediaType := detectGeminiPassthroughMediaType(input.Path, input.Body, body)
	imageCount := detectGeminiPassthroughImageCount(input.Path, body)
	imageSize := detectGeminiPassthroughImageSize(input.Body)
	return &ForwardResult{
		RequestID:     requestID,
		Usage:         usage,
		Model:         requestedModel,
		UpstreamModel: upstreamModel,
		ServiceTier:   extractGeminiRequestedServiceTierFromBody(input.Body),
		Stream:        stream,
		Duration:      duration,
		MediaType:     mediaType,
		ImageCount:    imageCount,
		ImageSize:     imageSize,
	}
}

func extractOpenAICompatUsage(body []byte) ClaudeUsage {
	if len(body) == 0 {
		return ClaudeUsage{}
	}
	return ClaudeUsage{
		InputTokens:  int(gjson.GetBytes(body, "usage.prompt_tokens").Int()),
		OutputTokens: int(gjson.GetBytes(body, "usage.completion_tokens").Int()),
	}
}

func detectGeminiPassthroughRequestedModel(path string, body []byte) string {
	if model := strings.TrimSpace(gjson.GetBytes(body, "model").String()); model != "" {
		return model
	}
	trimmed := strings.TrimSpace(path)
	if idx := strings.Index(trimmed, "/models/"); idx >= 0 {
		modelPart := trimmed[idx+len("/models/"):]
		for _, sep := range []string{":", "?", "/"} {
			if cut := strings.Index(modelPart, sep); cut >= 0 {
				modelPart = modelPart[:cut]
			}
		}
		return strings.TrimSpace(modelPart)
	}
	return ""
}

func detectGeminiPassthroughMediaType(path string, requestBody []byte, responseBody []byte) string {
	lowerPath := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.Contains(lowerPath, "/images/generations"):
		return "image"
	case strings.Contains(lowerPath, "/videos"):
		return "video"
	}
	outputModality := inferBodyOutputModality(requestBody)
	if outputModality != "" && outputModality != "text" {
		return outputModality
	}
	if len(responseBody) > 0 {
		if gjson.GetBytes(responseBody, "data.0.b64_json").Exists() || gjson.GetBytes(responseBody, "data.0.url").Exists() {
			return "image"
		}
	}
	return ""
}

func detectGeminiPassthroughImageCount(path string, responseBody []byte) int {
	if !strings.Contains(strings.ToLower(strings.TrimSpace(path)), "/images/generations") {
		return 0
	}
	if len(responseBody) == 0 {
		return 1
	}
	if data := gjson.GetBytes(responseBody, "data"); data.Exists() && data.IsArray() {
		return len(data.Array())
	}
	return 1
}

func detectGeminiPassthroughImageSize(requestBody []byte) string {
	size := strings.TrimSpace(gjson.GetBytes(requestBody, "size").String())
	if size == "" {
		return ""
	}
	switch size {
	case "1024x1024":
		return "1K"
	case "1536x1536", "2048x2048":
		return "2K"
	case "4096x4096":
		return "4K"
	default:
		return size
	}
}
