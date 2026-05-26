package service

import (
	"context"
	"strings"
)

func (s *GatewayService) ResolveGeminiPublicModelMetadataAccount(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	modelID string,
) (*Account, bool, error) {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil, false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(platform), PlatformGemini) {
		return nil, false, nil
	}

	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, false, nil
	}

	mode := apiKey.EffectiveModelDisplayMode()
	normalizedModelID := strings.TrimSpace(modelID)
	seenAccounts := make(map[int64]struct{})
	candidates := make([]*Account, 0, 2)
	var firstErr error

	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		bindingPlatform := strings.TrimSpace(binding.Group.Platform)
		projectionPlatform := apiKeyPublicProjectionPlatform(bindingPlatform, PlatformGemini)
		if projectionPlatform == "" {
			continue
		}

		accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, binding.GroupID, QueryPlatformsForGroupPlatform(bindingPlatform, false))
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account == nil || !account.IsSchedulable() || !MatchesGroupPlatform(account, PlatformGemini) {
				continue
			}
			accountForProjection := ResolveProtocolGatewayInboundAccount(account, PlatformGemini)
			if !accountForProjection.IsGemini() {
				continue
			}
			if normalizedModelID != "" {
				entries, err := s.publicModelEntriesForAccount(
					ctx,
					accountForProjection,
					mode,
					projectionPlatform,
					binding.ModelPatterns,
					accountForProjection.GetModelMapping(),
				)
				if err != nil {
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				entries = s.filterPublicEntriesByActiveChannel(ctx, binding.GroupID, projectionPlatform, entries)
				if !publicModelEntriesContainID(entries, normalizedModelID) {
					continue
				}
			}
			if _, ok := seenAccounts[account.ID]; ok {
				continue
			}
			seenAccounts[account.ID] = struct{}{}
			accountCopy := *accountForProjection
			candidates = append(candidates, &accountCopy)
		}
	}

	if len(candidates) == 1 {
		return candidates[0], true, nil
	}
	if normalizedModelID != "" && len(candidates) > 0 {
		return candidates[0], false, nil
	}
	if len(candidates) == 0 && firstErr != nil {
		return nil, false, firstErr
	}
	return nil, false, nil
}

func publicModelEntriesContainID(entries []APIKeyPublicModelEntry, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	for _, entry := range entries {
		for _, candidate := range []string{entry.PublicID, entry.AliasID} {
			if strings.TrimSpace(candidate) == modelID {
				return true
			}
		}
	}
	return false
}
