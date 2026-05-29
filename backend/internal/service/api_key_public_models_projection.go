package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
)

const apiKeyPublicModelsSourcePolicyProjection = "policy_projection"

func (s *GatewayService) publicModelEntriesForAccount(
	ctx context.Context,
	account *Account,
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
) ([]APIKeyPublicModelEntry, error) {
	if account == nil {
		return nil, nil
	}
	_ = mode
	_ = mapping

	projection := BuildAccountModelProjection(ctx, account, s.modelRegistryService)
	entries := projectAccountModelProjectionToPublicEntries(platform, modelPatterns, projection)
	recordPublicModelProjectionSource(apiKeyPublicModelsSourcePolicyProjection)
	slog.Info(
		"api_key_public_models_policy_projection",
		"account_id", account.ID,
		"platform", platform,
		"source", apiKeyPublicModelsSourcePolicyProjection,
		"policy_mode", firstNonEmptyString(projectionPolicyMode(projection), AccountModelPolicyModeWhitelist),
		"projection_source", projectionSource(projection),
		"count", len(entries),
		"alias_only_count", countAliasOnlyPublicEntries(entries),
	)
	return entries, nil
}

func projectAccountModelProjectionToPublicEntries(
	platform string,
	modelPatterns []string,
	projection *AccountModelProjection,
) []APIKeyPublicModelEntry {
	if projection == nil || len(projection.Entries) == 0 {
		return nil
	}

	projected := make(map[string]APIKeyPublicModelEntry, len(projection.Entries))
	for _, candidate := range projection.Entries {
		publicID := normalizeRegistryID(candidate.DisplayModelID)
		if publicID == "" {
			continue
		}
		targetID := normalizeRegistryID(firstNonEmptyString(candidate.TargetModelID, candidate.RouteModelID))
		if !bindingAllowsProjectedPublicModel(modelPatterns, publicID, targetID) {
			continue
		}
		if _, exists := projected[publicID]; exists {
			continue
		}
		displayName := strings.TrimSpace(candidate.DisplayModelID)
		if candidate.VisibilityMode != AccountModelVisibilityModeAlias {
			displayName = firstNonEmptyString(strings.TrimSpace(candidate.DisplayName), displayName)
		}
		projected[publicID] = APIKeyPublicModelEntry{
			PublicID:          publicID,
			AliasID:           publicID,
			SourceID:          targetID,
			DisplayName:       displayName,
			Platform:          platform,
			AvailabilityState: firstNonEmptyTrimmed(candidate.AvailabilityState, AccountModelAvailabilityUnknown),
			StaleState:        firstNonEmptyTrimmed(candidate.StaleState, AccountModelStaleStateUnverified),
			LifecycleStatus: normalizePublicModelLifecycleStatus(
				candidate.Status,
				candidate.DisplayName,
				candidate.DisplayModelID,
				candidate.TargetModelID,
				candidate.RouteModelID,
			),
		}
	}

	result := make([]APIKeyPublicModelEntry, 0, len(projected))
	for _, entry := range projected {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublicID < result[j].PublicID
	})
	return result
}

func bindingAllowsProjectedPublicModel(modelPatterns []string, publicID string, targetID string) bool {
	for _, candidate := range []string{publicID, targetID} {
		if _, matched := bindingMatchesModel(modelPatterns, candidate); matched {
			return true
		}
	}
	return false
}

func projectionPolicyMode(projection *AccountModelProjection) string {
	if projection == nil {
		return ""
	}
	return strings.TrimSpace(projection.PolicyMode)
}

func projectionSource(projection *AccountModelProjection) string {
	if projection == nil {
		return ""
	}
	return strings.TrimSpace(projection.Source)
}
