package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func SetOpenAIImageNormalizedTracePayload(c *gin.Context, source string, req *NormalizedImageRequest, capabilityProfile string) {
	if c == nil || req == nil {
		return
	}
	payload := map[string]any{
		"operation":                normalizeOpenAIImageOperation(req.Operation),
		"display_model_id":         strings.TrimSpace(req.DisplayModelID),
		"target_model_id":          strings.TrimSpace(req.TargetModelID),
		"image_count":              len(req.Images),
		"has_mask":                 strings.TrimSpace(req.Mask) != "",
		"size":                     strings.TrimSpace(req.Size),
		"quality":                  strings.TrimSpace(req.Quality),
		"background":               strings.TrimSpace(req.Background),
		"output_format":            strings.TrimSpace(req.OutputFormat),
		"moderation":               strings.TrimSpace(req.Moderation),
		"input_fidelity":           strings.TrimSpace(req.InputFidelity),
		"stream":                   req.Stream,
		"image_capability_profile": strings.TrimSpace(capabilityProfile),
	}
	if req.OutputCompression != nil {
		payload["output_compression"] = *req.OutputCompression
	}
	if req.PartialImages != nil {
		payload["partial_images"] = *req.PartialImages
	}
	if req.N != nil {
		payload["n"] = *req.N
	}
	SetOpsTraceNormalizedRequest(c, source, payload)
}
