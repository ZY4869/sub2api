package service

import (
	"context"
	"strings"
)

// IsNativeImageModel returns true when the model is a native image generation model
// (capability=image_generation). Tool-only image models (image_generation_tool) return false.
//
// It uses the local model registry snapshot only and must not trigger upstream probing.
func (s *GatewayService) IsNativeImageModel(ctx context.Context, modelID string) bool {
	if s == nil {
		return false
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	entry := &APIKeyPublicModelEntry{
		PublicID: modelID,
		AliasID:  modelID,
		SourceID: modelID,
	}
	native, _ := s.resolvePublicImageCapability(ctx, entry)
	return native
}

