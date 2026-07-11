package service

import "context"

func (s *ImageBatchService) ListModels(ctx context.Context, apiKey *APIKey) ([]ImageBatchModelResponse, error) {
	if s == nil || s.gatewayService == nil || s.repo == nil || apiKey == nil {
		return []ImageBatchModelResponse{}, nil
	}
	if !s.imageBatchGloballyEnabled(ctx) {
		return []ImageBatchModelResponse{}, nil
	}
	groupID := derefInt64(apiKey.GroupID)
	if groupID <= 0 {
		return []ImageBatchModelResponse{}, nil
	}
	settings, err := s.repo.GetGroupSettings(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if settings == nil || !settings.Enabled {
		return []ImageBatchModelResponse{}, nil
	}
	entries, err := s.gatewayService.GetAPIKeyPublicModels(ctx, apiKey, PlatformGemini)
	if err != nil {
		return nil, err
	}
	providers := []string{ImageBatchProviderGeminiAPI, ImageBatchProviderVertex}
	out := make([]ImageBatchModelResponse, 0, len(entries)*len(providers))
	for i := range entries {
		entry := entries[i]
		native, _ := s.gatewayService.resolvePublicImageCapability(ctx, &entry)
		if !native {
			continue
		}
		for _, provider := range providers {
			if !imageBatchProviderModelAllowed(settings, provider, entry.PublicID) {
				continue
			}
			out = append(out, ImageBatchModelResponse{
				ID:          entry.PublicID,
				Object:      "image_batch.model",
				DisplayName: entry.DisplayName,
				Provider:    provider,
			})
		}
	}
	return out, nil
}
