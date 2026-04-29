package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
)

type PublicModelProjectionEntry struct {
	PublicID          string   `json:"public_id"`
	DisplayName       string   `json:"display_name"`
	Platform          string   `json:"platform"`
	AvailabilityState string   `json:"availability_state,omitempty"`
	StaleState        string   `json:"stale_state,omitempty"`
	LifecycleStatus   string   `json:"lifecycle_status,omitempty"`
	AliasIDs          []string `json:"alias_ids,omitempty"`
	SourceIDs         []string `json:"source_ids,omitempty"`
}

func (s *GatewayService) ListGroupPublicModelProjection(
	ctx context.Context,
	group *Group,
	modelPatterns []string,
) ([]PublicModelProjectionEntry, error) {
	if s == nil || s.accountRepo == nil || group == nil || !group.IsActive() {
		return nil, nil
	}

	bindingPlatform := strings.TrimSpace(group.Platform)
	if bindingPlatform == "" {
		return nil, nil
	}

	queryPlatforms := QueryPlatformsForGroupPlatform(bindingPlatform, false)
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, group.ID, queryPlatforms)
	if err != nil {
		return nil, err
	}

	projectionByID := make(map[string]PublicModelProjectionEntry)
	for i := range accounts {
		account := &accounts[i]
		if account == nil || !account.IsSchedulable() {
			continue
		}
		entries, err := s.publicModelEntriesForAccount(
			ctx,
			account,
			APIKeyModelDisplayModeAliasOnly,
			bindingPlatform,
			modelPatterns,
			account.GetModelMapping(),
		)
		if err != nil {
			return nil, err
		}
		entries = s.filterPublicEntriesByActiveChannel(ctx, group.ID, bindingPlatform, entries)
		for _, entry := range entries {
			appendPublicModelProjectionEntry(projectionByID, entry)
		}
	}

	return sortPublicModelProjectionEntries(projectionByID), nil
}

func (s *GatewayService) ListActivePublicModelProjection(ctx context.Context) ([]PublicModelProjectionEntry, error) {
	if s == nil || s.groupRepo == nil {
		return nil, nil
	}

	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	projectionByID := make(map[string]PublicModelProjectionEntry)
	var firstErr error
	for i := range groups {
		group := &groups[i]
		if group == nil || !group.IsActive() {
			continue
		}
		entries, err := s.ListGroupPublicModelProjection(ctx, group, nil)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, entry := range entries {
			appendPublicModelProjectionAggregate(projectionByID, entry)
		}
	}
	if len(projectionByID) == 0 && firstErr != nil {
		return nil, firstErr
	}
	result := sortPublicModelProjectionEntries(projectionByID)
	slog.Info(
		"public_model_projection_active_groups_built",
		"group_count", len(groups),
		"model_count", len(result),
		"has_error", firstErr != nil,
	)
	return result, nil
}

func appendPublicModelProjectionEntry(target map[string]PublicModelProjectionEntry, entry APIKeyPublicModelEntry) {
	publicID := strings.TrimSpace(entry.PublicID)
	if publicID == "" {
		return
	}

	current := target[publicID]
	replaceRepresentative := strings.TrimSpace(current.PublicID) == "" || isBetterPublicModelRepresentative(
		entry.AvailabilityState,
		entry.StaleState,
		entry.LifecycleStatus,
		current.AvailabilityState,
		current.StaleState,
		current.LifecycleStatus,
	)
	current.PublicID = publicID
	if replaceRepresentative || strings.TrimSpace(current.DisplayName) == "" {
		current.DisplayName = strings.TrimSpace(entry.DisplayName)
	}
	if replaceRepresentative || strings.TrimSpace(current.Platform) == "" {
		current.Platform = strings.TrimSpace(entry.Platform)
	}
	if replaceRepresentative || strings.TrimSpace(current.AvailabilityState) == "" {
		current.AvailabilityState = firstNonEmptyTrimmed(entry.AvailabilityState, AccountModelAvailabilityUnknown)
		current.StaleState = firstNonEmptyTrimmed(entry.StaleState, AccountModelStaleStateUnverified)
		current.LifecycleStatus = normalizePublicModelLifecycleStatus(entry.LifecycleStatus, entry.DisplayName, entry.PublicID, entry.SourceID)
	}
	current.AliasIDs = mergePublicModelProjectionStrings(current.AliasIDs, entry.AliasID)
	current.SourceIDs = mergePublicModelProjectionStrings(current.SourceIDs, entry.SourceID)
	target[publicID] = current
}

func appendPublicModelProjectionAggregate(target map[string]PublicModelProjectionEntry, entry PublicModelProjectionEntry) {
	publicID := strings.TrimSpace(entry.PublicID)
	if publicID == "" {
		return
	}
	current := target[publicID]
	replaceRepresentative := strings.TrimSpace(current.PublicID) == "" || isBetterPublicModelRepresentative(
		entry.AvailabilityState,
		entry.StaleState,
		entry.LifecycleStatus,
		current.AvailabilityState,
		current.StaleState,
		current.LifecycleStatus,
	)
	current.PublicID = publicID
	if replaceRepresentative || strings.TrimSpace(current.DisplayName) == "" {
		current.DisplayName = strings.TrimSpace(entry.DisplayName)
	}
	if replaceRepresentative || strings.TrimSpace(current.Platform) == "" {
		current.Platform = strings.TrimSpace(entry.Platform)
	}
	if replaceRepresentative || strings.TrimSpace(current.AvailabilityState) == "" {
		current.AvailabilityState = firstNonEmptyTrimmed(entry.AvailabilityState, AccountModelAvailabilityUnknown)
		current.StaleState = firstNonEmptyTrimmed(entry.StaleState, AccountModelStaleStateUnverified)
		current.LifecycleStatus = normalizePublicModelLifecycleStatus(entry.LifecycleStatus, entry.DisplayName, entry.PublicID)
	}
	current.AliasIDs = mergePublicModelProjectionStrings(current.AliasIDs, entry.AliasIDs...)
	current.SourceIDs = mergePublicModelProjectionStrings(current.SourceIDs, entry.SourceIDs...)
	target[publicID] = current
}

func mergePublicModelProjectionStrings(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(values))
	merged := make([]string, 0, len(existing)+len(values))
	for _, item := range existing {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		merged = append(merged, trimmed)
	}
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		merged = append(merged, trimmed)
	}
	sort.Strings(merged)
	return merged
}

func sortPublicModelProjectionEntries(items map[string]PublicModelProjectionEntry) []PublicModelProjectionEntry {
	if len(items) == 0 {
		return nil
	}
	result := make([]PublicModelProjectionEntry, 0, len(items))
	for _, item := range items {
		item.AliasIDs = mergePublicModelProjectionStrings(nil, item.AliasIDs...)
		item.SourceIDs = mergePublicModelProjectionStrings(nil, item.SourceIDs...)
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(result[i].DisplayName))
		rightName := strings.ToLower(strings.TrimSpace(result[j].DisplayName))
		switch {
		case leftName != "" && rightName != "" && leftName != rightName:
			return leftName < rightName
		case leftName == "" && rightName != "":
			return false
		case leftName != "" && rightName == "":
			return true
		default:
			return result[i].PublicID < result[j].PublicID
		}
	})
	return result
}
