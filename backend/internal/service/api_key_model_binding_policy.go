package service

import (
	"context"
	"strings"
)

func (s *APIKeyService) validateUserAPIKeyModelBindings(
	ctx context.Context,
	user *User,
	bindings []APIKeyGroupBinding,
) error {
	if user == nil {
		return ErrAPIKeyModelSelectionRequired
	}
	mode := user.EffectiveAPIKeyModelBindingMode()
	for _, binding := range bindings {
		if mode == APIKeyModelBindingModeModelRequired && len(binding.ModelPatterns) == 0 {
			return ErrAPIKeyModelSelectionRequired
		}
		if len(binding.ModelPatterns) == 0 {
			continue
		}
		visible, err := s.visibleModelSetForAPIKeyGroupBinding(ctx, binding)
		if err != nil {
			return err
		}
		for _, pattern := range binding.ModelPatterns {
			modelID := NormalizeModelCatalogModelID(pattern)
			if modelID == "" || apiKeyModelPatternHasWildcard(pattern) {
				return ErrAPIKeyModelPatternForbidden
			}
			if _, ok := visible[modelID]; !ok {
				return ErrAPIKeyModelNotVisible
			}
		}
	}
	return nil
}

func (s *APIKeyService) visibleModelSetForAPIKeyGroupBinding(
	ctx context.Context,
	binding APIKeyGroupBinding,
) (map[string]struct{}, error) {
	if s == nil || s.gatewayService == nil || binding.Group == nil {
		return nil, ErrAPIKeyModelNotVisible
	}
	projection, err := s.gatewayService.ListGroupPublicModelProjection(ctx, binding.Group, nil)
	if err != nil {
		return nil, err
	}
	visible := make(map[string]struct{}, len(projection))
	for _, entry := range projection {
		publicID := NormalizeModelCatalogModelID(entry.PublicID)
		if publicID == "" {
			continue
		}
		visible[publicID] = struct{}{}
	}
	return visible, nil
}

func apiKeyModelPatternHasWildcard(value string) bool {
	return strings.ContainsAny(strings.TrimSpace(value), "*?[]")
}
