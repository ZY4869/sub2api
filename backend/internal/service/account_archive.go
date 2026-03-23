package service

import (
	"strconv"
	"strings"
	"time"
)

const (
	AccountExtraKeyArchiveRestoreGroupIDs = "_archive_restore_group_ids"
	AccountExtraKeyArchiveRestoreGroups   = "_archive_restore_groups"
)

type ArchivedAccountGroupFilters struct {
	Platform    string
	AccountType string
	Status      string
	Search      string
	GroupID     int64
}

type ArchivedAccountGroupSummary struct {
	GroupID         int64
	GroupName       string
	TotalCount      int
	AvailableCount  int
	InvalidCount    int
	LatestUpdatedAt time.Time
}

type UnarchiveAccountsInput struct {
	AccountIDs []int64
}

type UnarchiveAccountResult struct {
	AccountID                int64   `json:"account_id"`
	Success                  bool    `json:"success"`
	RestoredGroupIDs         []int64 `json:"restored_group_ids,omitempty"`
	UsedFallbackCurrentGroup bool    `json:"used_fallback_current_group"`
	ErrorMessage             string  `json:"error_message,omitempty"`
}

type UnarchiveAccountsResult struct {
	RestoredCount                int                      `json:"restored_count"`
	FailedCount                  int                      `json:"failed_count"`
	RestoredToOriginalGroupCount int                      `json:"restored_to_original_group_count"`
	RestoredInPlaceCount         int                      `json:"restored_in_place_count"`
	Results                      []UnarchiveAccountResult `json:"results"`
}

func BuildArchiveRestoreSnapshot(account *Account) map[string]any {
	if account == nil {
		return nil
	}

	groupIDs := make([]int64, 0, len(account.GroupIDs))
	seenGroupIDs := make(map[int64]struct{}, len(account.GroupIDs))
	for _, groupID := range account.GroupIDs {
		if groupID <= 0 {
			continue
		}
		if _, exists := seenGroupIDs[groupID]; exists {
			continue
		}
		seenGroupIDs[groupID] = struct{}{}
		groupIDs = append(groupIDs, groupID)
	}

	groups := make([]map[string]any, 0, len(account.Groups))
	for _, group := range account.Groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		groups = append(groups, map[string]any{
			"id":   group.ID,
			"name": strings.TrimSpace(group.Name),
		})
	}

	return map[string]any{
		AccountExtraKeyArchiveRestoreGroupIDs: groupIDs,
		AccountExtraKeyArchiveRestoreGroups:   groups,
	}
}

func ParseArchiveRestoreGroupIDs(extra map[string]any) ([]int64, bool) {
	if len(extra) == 0 {
		return nil, false
	}

	raw, exists := extra[AccountExtraKeyArchiveRestoreGroupIDs]
	if !exists {
		return nil, false
	}

	switch values := raw.(type) {
	case []int64:
		return append([]int64(nil), values...), true
	case []int:
		out := make([]int64, 0, len(values))
		for _, value := range values {
			if value > 0 {
				out = append(out, int64(value))
			}
		}
		return out, true
	case []any:
		out := make([]int64, 0, len(values))
		for _, value := range values {
			parsed := parseArchiveRestoreGroupID(value)
			if parsed > 0 {
				out = append(out, parsed)
			}
		}
		return out, true
	default:
		parsed := parseArchiveRestoreGroupID(values)
		if parsed > 0 {
			return []int64{parsed}, true
		}
		return []int64{}, true
	}
}

func parseArchiveRestoreGroupID(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
