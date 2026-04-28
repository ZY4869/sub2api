package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrImageOnlyModelOptionsUnavailable = infraerrors.ServiceUnavailable(
		"IMAGE_ONLY_MODEL_OPTIONS_UNAVAILABLE",
		"image model options are unavailable",
	)
	ErrImageOnlyGroupHasNoImageModels = infraerrors.BadRequest(
		"IMAGE_ONLY_GROUP_HAS_NO_IMAGE_MODELS",
		"target group has no available image generation models",
	)
	ErrImageOnlySelectedModelsInvalid = infraerrors.BadRequest(
		"IMAGE_ONLY_SELECTED_MODELS_INVALID",
		"image-only key must select at least one image generation model",
	)
	ErrImageOnlyGroupRequired = infraerrors.BadRequest(
		"IMAGE_ONLY_GROUP_REQUIRED",
		"image-only key must bind at least one group",
	)
)

func (s *APIKeyService) normalizeImageOnlyGroupBindings(
	ctx context.Context,
	bindings []APIKeyGroupBinding,
) ([]APIKeyGroupBinding, error) {
	if len(bindings) == 0 {
		return nil, ErrImageOnlyGroupRequired
	}
	if s == nil || s.gatewayService == nil {
		return nil, ErrImageOnlyModelOptionsUnavailable
	}

	normalized := make([]APIKeyGroupBinding, 0, len(bindings))
	for _, binding := range bindings {
		imageModels, err := s.imageOnlyModelsForGroupBinding(ctx, binding)
		if err != nil {
			return nil, err
		}
		if len(imageModels) == 0 {
			return nil, ErrImageOnlyGroupHasNoImageModels
		}
		modelPatterns := imageOnlyBindingModelPatterns(binding.ModelPatterns, imageModels)
		if len(modelPatterns) == 0 {
			return nil, ErrImageOnlySelectedModelsInvalid
		}
		binding.ModelPatterns = modelPatterns
		normalized = append(normalized, binding)
	}
	return normalized, nil
}

func (s *APIKeyService) imageOnlyModelsForGroupBinding(
	ctx context.Context,
	binding APIKeyGroupBinding,
) ([]PublicModelProjectionEntry, error) {
	if binding.Group == nil {
		return nil, ErrImageOnlyModelOptionsUnavailable
	}
	projection, err := s.gatewayService.ListGroupPublicModelProjection(ctx, binding.Group, nil)
	if err != nil {
		return nil, err
	}

	imageModels := make([]PublicModelProjectionEntry, 0, len(projection))
	for _, entry := range projection {
		if s.publicProjectionEntryHasNativeImageCapability(ctx, entry) {
			imageModels = append(imageModels, entry)
		}
	}
	return imageModels, nil
}

func (s *APIKeyService) publicProjectionEntryHasNativeImageCapability(
	ctx context.Context,
	entry PublicModelProjectionEntry,
) bool {
	publicID := strings.TrimSpace(entry.PublicID)
	if publicID == "" || s == nil || s.gatewayService == nil {
		return false
	}
	sourceIDs := append([]string(nil), entry.SourceIDs...)
	if len(sourceIDs) == 0 {
		sourceIDs = append(sourceIDs, publicID)
	}
	for _, sourceID := range sourceIDs {
		candidate := &APIKeyPublicModelEntry{
			PublicID: publicID,
			AliasID:  publicID,
			SourceID: strings.TrimSpace(sourceID),
			Platform: entry.Platform,
		}
		native, _ := s.gatewayService.resolvePublicImageCapability(ctx, candidate)
		if native {
			return true
		}
	}
	return false
}

func APIKeyAllowsConfiguredModel(apiKey *APIKey, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if apiKey == nil || modelID == "" {
		return false
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return false
	}
	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		if _, matched := bindingMatchesModel(binding.ModelPatterns, modelID); matched {
			return true
		}
	}
	return false
}

func imageOnlyBindingModelPatterns(
	currentPatterns []string,
	imageModels []PublicModelProjectionEntry,
) []string {
	result := make([]string, 0, len(imageModels))
	seen := make(map[string]struct{}, len(imageModels))
	for _, entry := range imageModels {
		publicID := NormalizeModelCatalogModelID(entry.PublicID)
		if publicID == "" {
			continue
		}
		if !imageOnlyBindingAllowsProjection(currentPatterns, entry) {
			continue
		}
		if _, exists := seen[publicID]; exists {
			continue
		}
		seen[publicID] = struct{}{}
		result = append(result, publicID)
	}
	return result
}

func imageOnlyBindingAllowsProjection(
	patterns []string,
	entry PublicModelProjectionEntry,
) bool {
	if len(patterns) == 0 {
		return true
	}
	publicID := NormalizeModelCatalogModelID(entry.PublicID)
	if publicID == "" {
		return false
	}
	if bindingAllowsProjectedPublicModel(patterns, publicID, publicID) {
		return true
	}
	for _, sourceID := range entry.SourceIDs {
		if bindingAllowsProjectedPublicModel(patterns, publicID, sourceID) {
			return true
		}
	}
	for _, aliasID := range entry.AliasIDs {
		if bindingAllowsProjectedPublicModel(patterns, publicID, aliasID) {
			return true
		}
	}
	return false
}
