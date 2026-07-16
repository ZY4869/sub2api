package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	duplicateAccountOperationIDExtraKey = "account_duplicate_operation_id"
	duplicateAccountSourceIDExtraKey    = "account_duplicate_source_id"
	accountNameMaxRunes                 = 100
)

var duplicateAccountRuntimeExtraKeys = map[string]struct{}{
	"quota_used":                         {},
	"quota_daily_used":                   {},
	"quota_daily_start":                  {},
	"quota_weekly_used":                  {},
	"quota_weekly_start":                 {},
	"quota_monthly_used":                 {},
	"quota_monthly_start":                {},
	"model_rate_limits":                  {},
	"codex_5h_used_percent":              {},
	"codex_5h_reset_at":                  {},
	"codex_5h_reset_after_seconds":       {},
	"codex_5h_window_minutes":            {},
	"codex_7d_used_percent":              {},
	"codex_7d_reset_at":                  {},
	"codex_7d_reset_after_seconds":       {},
	"codex_7d_window_minutes":            {},
	"codex_usage_updated_at":             {},
	"codex_spark_usage_updated_at":       {},
	"codex_spark_5h_used_percent":        {},
	"codex_spark_5h_reset_at":            {},
	"codex_spark_5h_reset_after_seconds": {},
	"codex_spark_5h_window_minutes":      {},
	"codex_spark_7d_used_percent":        {},
	"codex_spark_7d_reset_at":            {},
	"codex_spark_7d_reset_after_seconds": {},
	"codex_spark_7d_window_minutes":      {},
	"grok_usage_snapshot":                {},
	"grok_billing_snapshot":              {},
	"openai_responses_supported":         {},
	"openai_compact_checked_at":          {},
	"session_window_utilization":         {},
	"passive_usage_sampled_at":           {},
	"antigravity_force_token_refresh":    {},
	"antigravity_credits_overages":       {},
	"crs_account_id":                     {},
	"crs_kind":                           {},
	"crs_synced_at":                      {},
}

func (s *adminServiceImpl) DuplicateAccount(ctx context.Context, id int64, actorScope, operationKey string) (*Account, error) {
	if existing, err := s.RecoverDuplicateAccount(ctx, id, actorScope, operationKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	source, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrAccountNotFound
	}
	if !isDuplicateAccountTypeSupported(source.Type) {
		return nil, infraerrors.BadRequest("ACCOUNT_DUPLICATE_CREDENTIAL_TYPE_UNSUPPORTED", "this account type cannot be duplicated safely")
	}
	if source.GetExtraString("parent_account_id") != "" {
		return nil, infraerrors.BadRequest("ACCOUNT_DUPLICATE_SHADOW_UNSUPPORTED", "linked shadow accounts cannot be duplicated")
	}

	duplicate := buildDuplicateAccount(source, actorScope, operationKey)
	if err := s.createDuplicateAccountWithGroups(ctx, duplicate); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, duplicate.ID)
}

func (s *adminServiceImpl) RecoverDuplicateAccount(ctx context.Context, id int64, actorScope, operationKey string) (*Account, error) {
	operationID := duplicateAccountOperationID(id, actorScope, operationKey)
	if operationID == "" {
		return nil, nil
	}
	matches, err := s.accountRepo.FindByExtraField(ctx, duplicateAccountOperationIDExtraKey, operationID)
	if err != nil {
		return nil, err
	}
	for i := range matches {
		account := matches[i]
		if account.GetExtraString(duplicateAccountSourceIDExtraKey) != strconv.FormatInt(id, 10) {
			continue
		}
		return &account, nil
	}
	return nil, nil
}

func buildDuplicateAccount(source *Account, actorScope, operationKey string) *Account {
	duplicate := &Account{
		Name:                    duplicateAccountName(source.Name),
		Notes:                   cloneStringPtr(source.Notes),
		Platform:                source.Platform,
		Type:                    source.Type,
		Credentials:             cloneJSONMap(source.Credentials),
		Extra:                   duplicateAccountExtra(source.Extra),
		ProxyID:                 cloneInt64Ptr(source.ProxyID),
		Concurrency:             source.Concurrency,
		Priority:                source.Priority,
		RateMultiplier:          cloneAccountFloat64Ptr(source.RateMultiplier),
		LoadFactor:              cloneIntPtr(source.LoadFactor),
		Status:                  StatusActive,
		Schedulable:             false,
		LifecycleState:          AccountLifecycleNormal,
		ExpiresAt:               cloneTimePtr(source.ExpiresAt),
		AutoPauseOnExpired:      source.AutoPauseOnExpired,
		AutoRenewEnabled:        source.AutoRenewEnabled,
		AutoRenewPeriod:         source.AutoRenewPeriod,
		GroupIDs:                append([]int64(nil), source.GroupIDs...),
		TempUnschedulableReason: "",
	}
	if duplicate.Extra == nil {
		duplicate.Extra = map[string]any{}
	}
	duplicate.Extra[duplicateAccountSourceIDExtraKey] = strconv.FormatInt(source.ID, 10)
	if operationID := duplicateAccountOperationID(source.ID, actorScope, operationKey); operationID != "" {
		duplicate.Extra[duplicateAccountOperationIDExtraKey] = operationID
	}
	return duplicate
}

func (s *adminServiceImpl) createDuplicateAccountWithGroups(ctx context.Context, account *Account) error {
	if err := s.accountRepo.Create(ctx, account); err != nil {
		return err
	}
	if len(account.GroupIDs) == 0 {
		return nil
	}
	if err := s.accountRepo.BindGroups(ctx, account.ID, account.GroupIDs); err != nil {
		_ = s.accountRepo.Delete(ctx, account.ID)
		return err
	}
	return nil
}

func isDuplicateAccountTypeSupported(accountType string) bool {
	switch strings.TrimSpace(strings.ToLower(accountType)) {
	case AccountTypeAPIKey, AccountTypeUpstream, AccountTypeBedrock:
		return true
	default:
		return false
	}
}

func duplicateAccountName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Account"
	}
	const suffix = " (Copy)"
	limit := accountNameMaxRunes - utf8.RuneCountInString(suffix)
	runes := []rune(base)
	if limit < 1 {
		limit = 1
	}
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes) + suffix
}

func duplicateAccountExtra(extra map[string]any) map[string]any {
	cloned := cloneJSONMap(extra)
	for key := range duplicateAccountRuntimeExtraKeys {
		delete(cloned, key)
	}
	return cloned
}

func duplicateAccountOperationID(sourceID int64, actorScope, operationKey string) string {
	operationKey = strings.TrimSpace(operationKey)
	if sourceID <= 0 || operationKey == "" {
		return ""
	}
	actorScope = strings.TrimSpace(actorScope)
	if actorScope == "" {
		actorScope = "admin:0"
	}
	sum := sha256.Sum256([]byte(actorScope + "\n" + strconv.FormatInt(sourceID, 10) + "\n" + operationKey))
	return hex.EncodeToString(sum[:])
}

func cloneJSONMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	raw, err := json.Marshal(src)
	if err != nil {
		out := make(map[string]any, len(src))
		for key, value := range src {
			out[key] = value
		}
		return out
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		out = make(map[string]any, len(src))
		for key, value := range src {
			out[key] = value
		}
	}
	return out
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneAccountFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
