package service

import (
	"context"
	"time"
)

type GrokBuildModelScopeUpdateRepository interface {
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
}

type GrokBuildModelScopeEnsureResult struct {
	Updated           bool
	ScopeNormalized   bool
	SnapshotRefreshed bool
}

func EnsureGrokBuildModelScopeUpdates(ctx context.Context, account *Account, registry *ModelRegistryService, now time.Time) (map[string]any, bool, bool) {
	if account == nil || !account.IsGrokOAuth() {
		return nil, false, false
	}
	if hasStructuredGrokAccountModelScopeEntries(account.Extra) {
		return nil, false, false
	}
	scope := DefaultGrokBuildTextModelScope()
	updates := map[string]any{
		"model_scope_v2": scope.ToMap(),
	}
	clone := *account
	clone.Extra = MergeStringAnyMap(account.Extra, updates)
	projection := BuildAccountModelProjection(ctx, &clone, registry)
	if projection != nil && len(projection.Entries) > 0 {
		snapshotUpdates := BuildAccountModelAvailabilitySnapshotExtra(
			projection,
			GrokBuildTextModelIDs(),
			now,
			AccountModelProbeSnapshotSourcePolicyBackfill,
			AccountModelProbeSnapshotSourcePolicyBackfill,
		)
		for key, value := range snapshotUpdates {
			updates[key] = value
		}
		return updates, true, true
	}
	return updates, true, false
}

func EnsureGrokBuildModelScopePersisted(ctx context.Context, repo GrokBuildModelScopeUpdateRepository, account *Account, registry *ModelRegistryService, now time.Time) (*GrokBuildModelScopeEnsureResult, error) {
	result := &GrokBuildModelScopeEnsureResult{}
	if repo == nil || account == nil {
		return result, nil
	}
	updates, scopeChanged, snapshotChanged := EnsureGrokBuildModelScopeUpdates(ctx, account, registry, now)
	if len(updates) == 0 {
		return result, nil
	}
	if err := repo.UpdateExtra(ctx, account.ID, updates); err != nil {
		return nil, err
	}
	account.Extra = MergeStringAnyMap(account.Extra, updates)
	result.Updated = true
	result.ScopeNormalized = scopeChanged
	result.SnapshotRefreshed = snapshotChanged
	return result, nil
}

func hasStructuredGrokAccountModelScopeEntries(extra map[string]any) bool {
	if !accountModelScopeUsesStructuredEntries(extra) {
		return false
	}
	scope, ok := ExtractAccountModelScopeV2(extra)
	return ok && scope != nil && len(scope.Entries) > 0
}
