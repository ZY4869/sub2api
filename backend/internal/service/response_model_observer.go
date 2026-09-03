package service

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const maxObservedResponseModelLength = 200

type responseModelObservation struct {
	model    string
	conflict bool
}

func observeOpenAIResponseModel(ctx context.Context, payload []byte) {
	if ctx == nil || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return
	}
	observeResponseModelInContext(ctx, responseModelFromJSON(payload))
}

func observeResponseModelInContext(ctx context.Context, model string) {
	if ctx == nil {
		return
	}
	model = normalizeObservedResponseModel(model)
	if model == "" {
		return
	}
	if previous, ok := UpstreamResponseModelMetadataFromContext(ctx); ok &&
		previous != "" && !strings.EqualFold(previous, model) {
		SetUpstreamModelMismatchMetadata(ctx, true)
		return
	}
	SetUpstreamResponseModelMetadata(ctx, model)
}

func firstResponseModelValue(values ...string) string {
	for _, value := range values {
		if normalized := normalizeObservedResponseModel(value); normalized != "" {
			return normalized
		}
	}
	return ""
}

func normalizeObservedResponseModel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > maxObservedResponseModelLength {
		value = value[:maxObservedResponseModelLength]
	}
	return value
}

func responseModelFromJSON(payload []byte) string {
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return ""
	}
	return firstResponseModelValue(
		gjson.GetBytes(payload, "response.model").String(),
		gjson.GetBytes(payload, "model").String(),
	)
}

func responseModelObservationFromSSE(body string) responseModelObservation {
	var observation responseModelObservation
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" || !gjson.Valid(data) {
			continue
		}
		model := responseModelFromJSON([]byte(data))
		if model == "" {
			continue
		}
		if observation.model != "" && !strings.EqualFold(observation.model, model) {
			observation.conflict = true
			continue
		}
		observation.model = model
	}
	return observation
}

func setOpenAIResponseModelObservation(ctx context.Context, observation responseModelObservation, expectedModel string) {
	if ctx == nil || observation.model == "" {
		return
	}
	observeResponseModelInContext(ctx, observation.model)
	if observation.conflict || !strings.EqualFold(observation.model, strings.TrimSpace(expectedModel)) {
		SetUpstreamModelMismatchMetadata(ctx, true)
	}
}

func setOpenAIResponseModelObservationOnRequest(c *gin.Context, observation responseModelObservation, expectedModel string) {
	if c == nil || c.Request == nil {
		return
	}
	ctx := EnsureRequestMetadata(c.Request.Context())
	setOpenAIResponseModelObservation(ctx, observation, expectedModel)
	c.Request = c.Request.WithContext(ctx)
}

func applyOpenAIResponseModelObservation(result *OpenAIForwardResult, ctx context.Context, expectedModel string) {
	if result == nil || ctx == nil {
		return
	}
	model, ok := UpstreamResponseModelMetadataFromContext(ctx)
	if !ok || model == "" {
		return
	}
	result.UpstreamResponseModel = model
	mismatch, mismatchSet := UpstreamModelMismatchMetadataFromContext(ctx)
	if !mismatchSet {
		mismatch = !strings.EqualFold(model, strings.TrimSpace(expectedModel))
	}
	result.UpstreamModelMismatch = &mismatch
}
