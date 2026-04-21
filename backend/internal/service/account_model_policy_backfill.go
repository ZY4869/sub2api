package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const AccountModelProbeSnapshotSourcePolicyBackfill = "policy_backfill"

type AccountModelPolicyBackfillRepository interface {
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) ([]Account, *pagination.PaginationResult, error)
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
}

type AccountModelPolicyBackfillResult struct {
	Scanned           int `json:"scanned"`
	Updated           int `json:"updated"`
	ScopeNormalized   int `json:"scope_normalized"`
	SnapshotRefreshed int `json:"snapshot_refreshed"`
}

func BackfillAccountModelPolicies(ctx context.Context, repo AccountModelPolicyBackfillRepository, registry *ModelRegistryService, pageSize int) (*AccountModelPolicyBackfillResult, error) {
	if repo == nil {
		return &AccountModelPolicyBackfillResult{}, nil
	}
	if pageSize <= 0 {
		pageSize = 100
	}

	result := &AccountModelPolicyBackfillResult{}
	for page := 1; ; page++ {
		accounts, paginationResult, err := repo.ListWithFilters(
			ctx,
			pagination.PaginationParams{Page: page, PageSize: pageSize},
			"",
			"",
			"",
			"",
			0,
			AccountLifecycleAll,
			"",
		)
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			return result, nil
		}

		for i := range accounts {
			account := &accounts[i]
			result.Scanned++
			updates, scopeChanged, snapshotChanged := BuildAccountModelPolicyBackfillUpdates(ctx, account, registry, time.Now().UTC())
			if len(updates) == 0 {
				continue
			}
			if err := repo.UpdateExtra(ctx, account.ID, updates); err != nil {
				return nil, err
			}
			result.Updated++
			if scopeChanged {
				result.ScopeNormalized++
			}
			if snapshotChanged {
				result.SnapshotRefreshed++
			}
		}

		if paginationResult == nil || paginationResult.Pages <= page {
			return result, nil
		}
	}
}

func BuildAccountModelPolicyBackfillUpdates(ctx context.Context, account *Account, registry *ModelRegistryService, now time.Time) (map[string]any, bool, bool) {
	if account == nil {
		return nil, false, false
	}

	updates := map[string]any{}
	scopeChanged := false
	snapshotChanged := false

	projection := BuildAccountModelProjection(ctx, account, registry)
	if normalizedScope := buildAccountModelPolicyBackfillScopeFromProjection(projection); normalizedScope != nil {
		normalizedScopeMap := normalizedScope.ToMap()
		if !jsonValuesEqual(account.Extra["model_scope_v2"], normalizedScopeMap) {
			updates["model_scope_v2"] = normalizedScopeMap
			scopeChanged = true
		}
	}

	if projection != nil && len(projection.Entries) > 0 {
		snapshotUpdatedAt := now
		snapshotSource := AccountModelProbeSnapshotSourcePolicyBackfill
		probeSource := AccountModelProbeSnapshotSourcePolicyBackfill
		if snapshot, ok := AccountModelProbeSnapshotFromExtra(account.Extra); ok && snapshot != nil {
			if updatedAt, parsed := parseSnapshotUpdatedAt(snapshot.UpdatedAt); parsed {
				snapshotUpdatedAt = updatedAt
			}
			snapshotSource = firstNonEmptyString(snapshot.Source, snapshotSource)
			probeSource = firstNonEmptyString(snapshot.ProbeSource, probeSource)
		}

		snapshotUpdates := BuildAccountModelAvailabilitySnapshotExtra(
			projection,
			AccountSavedModelIDs(account),
			snapshotUpdatedAt,
			snapshotSource,
			probeSource,
		)
		if snapshotUpdates != nil && !jsonValuesEqual(account.Extra[accountModelProbeSnapshotExtraKey], snapshotUpdates[accountModelProbeSnapshotExtraKey]) {
			for key, value := range snapshotUpdates {
				updates[key] = value
			}
			snapshotChanged = true
		}
	}

	if len(updates) == 0 {
		return nil, false, false
	}
	return updates, scopeChanged, snapshotChanged
}

func buildAccountModelPolicyBackfillScopeFromProjection(projection *AccountModelProjection) *AccountModelScopeV2 {
	if projection == nil || !projection.Explicit {
		return nil
	}

	scope := &AccountModelScopeV2{
		PolicyMode: projection.PolicyMode,
		Entries:    make([]AccountModelScopeEntry, 0, len(projection.Entries)),
	}
	for _, entry := range projection.Entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(firstNonEmptyString(entry.TargetModelID, entry.RouteModelID, entry.CanonicalID, displayModelID))
		if displayModelID == "" {
			continue
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		scope.Entries = append(scope.Entries, AccountModelScopeEntry{
			DisplayModelID: displayModelID,
			TargetModelID:  targetModelID,
			Provider:       NormalizeModelProvider(entry.Provider),
			SourceProtocol: NormalizeGatewayProtocol(entry.SourceProtocol),
			VisibilityMode: normalizeAccountModelVisibilityMode(entry.VisibilityMode, displayModelID, targetModelID),
		})
	}
	scope.normalize()
	return scope
}

func jsonValuesEqual(left any, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}
