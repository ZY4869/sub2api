package service

import (
	"strings"
	"unicode/utf8"
)

func normalizeImageBatchProvider(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "gemini", ImageBatchProviderGeminiAPI, "google", "aistudio", "ai_studio":
		return ImageBatchProviderGeminiAPI
	case ImageBatchProviderVertex, "vertex_ai", "vertexai":
		return ImageBatchProviderVertex
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func normalizeImageBatchSize(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "1024x1024"
	}
	return value
}

func imageBatchGroupAllows(settings *ImageBatchGroupSettings, provider string, model string, itemCount int) bool {
	if settings == nil || !settings.Enabled || itemCount <= 0 {
		return false
	}
	if settings.MaxItems > 0 && itemCount > settings.MaxItems {
		return false
	}
	provider = normalizeImageBatchProvider(provider)
	model = strings.TrimSpace(model)
	if len(settings.AllowedProviders) > 0 && !stringListAllows(settings.AllowedProviders, provider) {
		return false
	}
	if len(settings.AllowedModels) > 0 && !stringListAllows(settings.AllowedModels, model) {
		return false
	}
	return true
}

func imageBatchProviderPlatform(provider string) string {
	switch normalizeImageBatchProvider(provider) {
	case ImageBatchProviderGeminiAPI, ImageBatchProviderVertex:
		return PlatformGemini
	default:
		return ""
	}
}

func imageBatchProviderModelAllowed(settings *ImageBatchGroupSettings, provider string, model string) bool {
	provider = normalizeImageBatchProvider(provider)
	if settings == nil || !settings.Enabled {
		return false
	}
	if len(settings.AllowedProviders) > 0 && !stringListAllows(settings.AllowedProviders, provider) {
		return false
	}
	if len(settings.AllowedModels) > 0 && !stringListAllows(settings.AllowedModels, strings.TrimSpace(model)) {
		return false
	}
	return true
}

func stringListAllows(patterns []string, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, raw := range patterns {
		pattern := strings.TrimSpace(raw)
		switch {
		case pattern == "":
			continue
		case pattern == "*":
			return true
		case strings.EqualFold(pattern, value):
			return true
		case strings.HasSuffix(pattern, "*") && strings.HasPrefix(strings.ToLower(value), strings.ToLower(strings.TrimSuffix(pattern, "*"))):
			return true
		}
	}
	return false
}

func normalizeImageBatchItems(req ImageBatchSubmitRequest) ([]ImageBatchItem, error) {
	if len(req.Items) == 0 {
		return nil, ErrImageBatchInvalidRequest
	}
	seen := make(map[string]struct{}, len(req.Items))
	items := make([]ImageBatchItem, 0, len(req.Items))
	for _, raw := range req.Items {
		customID := strings.TrimSpace(raw.CustomID)
		prompt := strings.TrimSpace(raw.Prompt)
		if customID == "" || prompt == "" || !utf8.ValidString(customID) || !utf8.ValidString(prompt) {
			return nil, ErrImageBatchInvalidRequest
		}
		if _, exists := seen[customID]; exists {
			return nil, ErrImageBatchInvalidRequest
		}
		seen[customID] = struct{}{}
		n := raw.N
		if n <= 0 {
			n = 1
		}
		if n > 4 {
			return nil, ErrImageBatchInvalidRequest
		}
		items = append(items, ImageBatchItem{
			CustomID: customID,
			Prompt:   prompt,
			N:        n,
			Size:     normalizeImageBatchSize(firstNonEmptyString(raw.Size, req.Size)),
			Status:   ImageBatchItemPending,
			Metadata: copyStringAnyMap(raw.Metadata),
		})
	}
	return items, nil
}

func copyStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[strings.TrimSpace(key)] = value
	}
	return out
}

func ImageBatchJobToResponse(job *ImageBatchJob) *ImageBatchJobResponse {
	if job == nil {
		return nil
	}
	return &ImageBatchJobResponse{
		ID:             job.ID,
		Object:         "image_batch.job",
		Status:         job.Status,
		Provider:       job.Provider,
		DisplayModelID: job.DisplayModelID,
		Size:           job.Size,
		Counts: ImageBatchJobCounts{
			Items:     job.ItemCount,
			Succeeded: job.SuccessCount,
			Failed:    job.FailedCount,
			Cancelled: job.CancelledCount,
		},
		FriendlyError: job.FriendlyError,
		ErrorID:       job.ErrorID,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
		SubmittedAt:   job.SubmittedAt,
		CompletedAt:   job.CompletedAt,
		CancelledAt:   job.CancelledAt,
	}
}

func ImageBatchItemToResponse(item *ImageBatchItem) *ImageBatchItemResponse {
	if item == nil {
		return nil
	}
	return &ImageBatchItemResponse{
		CustomID:      item.CustomID,
		Status:        item.Status,
		OutputCount:   item.OutputCount,
		FriendlyError: item.FriendlyError,
		ErrorID:       item.ErrorID,
		Metadata:      copyStringAnyMap(item.Metadata),
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}
