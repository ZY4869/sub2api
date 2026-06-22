package service

import (
	"strings"
	"time"
)

const (
	accountExpiryProbeExtensionDaysKey  = "expiry_probe_extension_days"
	accountExpiryProbeStatusKey         = "expiry_probe_status"
	accountExpiryProbeCheckedAtKey      = "expiry_probe_checked_at"
	accountExpiryProbeNextCheckAtKey    = "expiry_probe_next_check_at"
	accountExpiryProbePriorityUntilKey  = "expiry_probe_priority_until"
	accountExpiryProbeSummaryKey        = "expiry_probe_summary"
	accountAutoRenewStatusKey           = "auto_renew_status"
	accountAutoRenewLastRenewedAtKey    = "auto_renew_last_renewed_at"
	accountAutoRenewPeriodKey           = "auto_renew_last_period"
	accountAutoRenewPreviousExpiresKey  = "auto_renew_previous_expires_at"
	accountAutoRenewNextExpiresKey      = "auto_renew_next_expires_at"
	accountAutoRenewSummaryKey          = "auto_renew_summary"
	accountDaily5HLastLocalDateKey      = "daily_5h_trigger_last_local_date"
	accountDaily5HLastStatusKey         = "daily_5h_trigger_last_status"
	accountDaily5HLastModelIDKey        = "daily_5h_trigger_last_model_id"
	accountDaily5HLastSummaryKey        = "daily_5h_trigger_last_summary"
	AccountExpiryProbeStatusSuccess     = "success"
	AccountExpiryProbeStatusWaiting     = "waiting_window"
	AccountExpiryProbeStatusDisabled    = "disabled"
	AccountExpiryProbeStatusBlacklisted = "blacklisted"
	AccountExpiryProbeStatusFailed      = "failed"
	AccountAutoRenewStatusSuccess       = "success"
	AccountAutoRenewStatusFailed        = "failed"
	AccountDaily5HTriggerStatusSuccess  = "success"
	AccountDaily5HTriggerStatusSkipped  = "skipped"
	AccountDaily5HTriggerStatusFailed   = "failed"
	defaultExpiryProbeExtensionDays     = 1
	defaultAccountDaily5HTriggerHour    = 7
	AccountDaily5HTypeOpenAI            = "chatgpt_oauth"
	AccountDaily5HTypeAnthropic         = "claude_code_oauth_setup_token"
	AccountDaily5HTypeGemini            = "google_oauth"
	AccountDaily5HModelModeAuto         = "auto"
	AccountDaily5HModelModeFixed        = "fixed"
	accountDaily5HPrompt                = "Output exactly: OK"
)

const (
	AccountDaily5HSkipReasonLifecycleExcluded = "lifecycle_excluded"
	AccountDaily5HSkipReasonAccountType       = "account_type_not_selected"
	AccountDaily5HSkipReasonPausedExcluded    = "paused_excluded"
	AccountDaily5HSkipReasonFreeExcluded      = "free_account_excluded"
	AccountDaily5HSkipReasonRateLimited       = "rate_limited"
	AccountDaily5HSkipReasonTempUnsched       = "temp_unschedulable"
	AccountDaily5HSkipReasonOverloaded        = "overloaded"
	AccountDaily5HSkipReasonSessionWindow     = "session_window_active"
	AccountDaily5HSkipReasonFixedModelHidden  = "fixed_model_not_visible"
	AccountDaily5HSkipReasonNoFamilyModel     = "no_family_model_available"
)

type AccountExpiryProbeSummary struct {
	Status      string `json:"status,omitempty"`
	CheckedAt   string `json:"checked_at,omitempty"`
	NextCheckAt string `json:"next_check_at,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

type AccountDaily5HTriggerModelSettings struct {
	Mode         string `json:"mode"`
	FixedModelID string `json:"fixed_model_id,omitempty"`
}

type AccountDaily5HTriggerSettings struct {
	Enabled                   bool                               `json:"enabled"`
	SelectedAccountTypes      []string                           `json:"selected_account_types"`
	IncludePausedAccounts     bool                               `json:"include_paused_accounts"`
	IgnoreFreeAccounts        bool                               `json:"ignore_free_accounts"`
	SkipCNHolidaysAndWeekends bool                               `json:"skip_cn_holidays_and_weekends"`
	OpenAIModel               AccountDaily5HTriggerModelSettings `json:"openai_model_mode"`
	AnthropicModel            AccountDaily5HTriggerModelSettings `json:"anthropic_model_mode"`
	GeminiModel               AccountDaily5HTriggerModelSettings `json:"gemini_model_mode"`
}

type AccountDaily5HTriggerModelOption struct {
	ModelID       string `json:"model_id"`
	DisplayName   string `json:"display_name"`
	Provider      string `json:"provider,omitempty"`
	ProviderLabel string `json:"provider_label,omitempty"`
	AccountCount  int    `json:"account_count"`
}

type AccountDaily5HTriggerAccountTypeSummary struct {
	AccountType string                             `json:"account_type"`
	Count       int                                `json:"count"`
	Models      []AccountDaily5HTriggerModelOption `json:"models"`
}

type AccountDaily5HTriggerSettingsView struct {
	Settings   *AccountDaily5HTriggerSettings            `json:"settings"`
	Candidates []AccountDaily5HTriggerAccountTypeSummary `json:"candidates"`
}

func DefaultAccountDaily5HTriggerSettings() *AccountDaily5HTriggerSettings {
	return &AccountDaily5HTriggerSettings{
		Enabled:               false,
		SelectedAccountTypes:  []string{AccountDaily5HTypeOpenAI},
		IncludePausedAccounts: false,
		OpenAIModel: AccountDaily5HTriggerModelSettings{
			Mode: AccountDaily5HModelModeAuto,
		},
		AnthropicModel: AccountDaily5HTriggerModelSettings{
			Mode: AccountDaily5HModelModeAuto,
		},
		GeminiModel: AccountDaily5HTriggerModelSettings{
			Mode: AccountDaily5HModelModeAuto,
		},
	}
}

func NormalizeAccountDaily5HTriggerSettings(settings *AccountDaily5HTriggerSettings) *AccountDaily5HTriggerSettings {
	if settings == nil {
		return DefaultAccountDaily5HTriggerSettings()
	}
	normalized := *settings
	normalized.SelectedAccountTypes = normalizeAccountDaily5HSelectedTypes(settings.SelectedAccountTypes)
	normalized.OpenAIModel = normalizeAccountDaily5HModelSettings(settings.OpenAIModel)
	normalized.AnthropicModel = normalizeAccountDaily5HModelSettings(settings.AnthropicModel)
	normalized.GeminiModel = normalizeAccountDaily5HModelSettings(settings.GeminiModel)
	return &normalized
}

func normalizeAccountDaily5HSelectedTypes(types []string) []string {
	allowed := map[string]struct{}{
		AccountDaily5HTypeOpenAI:    {},
		AccountDaily5HTypeAnthropic: {},
		AccountDaily5HTypeGemini:    {},
	}
	if len(types) == 0 {
		return []string{AccountDaily5HTypeOpenAI}
	}
	seen := make(map[string]struct{}, len(types))
	out := make([]string, 0, len(types))
	for _, item := range types {
		normalized := strings.TrimSpace(strings.ToLower(item))
		if _, ok := allowed[normalized]; !ok {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return []string{AccountDaily5HTypeOpenAI}
	}
	return out
}

func normalizeAccountDaily5HModelSettings(settings AccountDaily5HTriggerModelSettings) AccountDaily5HTriggerModelSettings {
	mode := strings.TrimSpace(strings.ToLower(settings.Mode))
	if mode != AccountDaily5HModelModeFixed {
		mode = AccountDaily5HModelModeAuto
	}
	return AccountDaily5HTriggerModelSettings{
		Mode:         mode,
		FixedModelID: strings.TrimSpace(settings.FixedModelID),
	}
}

func AccountExpiryProbeExtensionDaysFromExtra(extra map[string]any) int {
	if value := parseExtraInt(extra[accountExpiryProbeExtensionDaysKey]); value > 0 {
		return value
	}
	return defaultExpiryProbeExtensionDays
}

func parseAccountExpiryProbeTime(extra map[string]any, key string) *time.Time {
	value := strings.TrimSpace(stringValueFromAny(extra[key]))
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}
	return nil
}

func BuildAccountExpiryProbeExtra(
	checkedAt time.Time,
	status string,
	summary string,
	nextCheckAt *time.Time,
	priorityUntil *time.Time,
) map[string]any {
	out := map[string]any{
		accountExpiryProbeCheckedAtKey: checkedAt.UTC().Format(time.RFC3339),
		accountExpiryProbeStatusKey:    strings.TrimSpace(status),
		accountExpiryProbeSummaryKey:   strings.TrimSpace(summary),
	}
	if nextCheckAt != nil && !nextCheckAt.IsZero() {
		out[accountExpiryProbeNextCheckAtKey] = nextCheckAt.UTC().Format(time.RFC3339)
	} else {
		out[accountExpiryProbeNextCheckAtKey] = nil
	}
	if priorityUntil != nil && !priorityUntil.IsZero() {
		out[accountExpiryProbePriorityUntilKey] = priorityUntil.UTC().Format(time.RFC3339)
	} else {
		out[accountExpiryProbePriorityUntilKey] = nil
	}
	return out
}

func BuildAccountAutoRenewExtra(
	renewedAt time.Time,
	status string,
	period string,
	previousExpiresAt time.Time,
	nextExpiresAt *time.Time,
	summary string,
) map[string]any {
	out := map[string]any{
		accountAutoRenewStatusKey:          strings.TrimSpace(status),
		accountAutoRenewLastRenewedAtKey:   renewedAt.UTC().Format(time.RFC3339),
		accountAutoRenewPeriodKey:          strings.TrimSpace(period),
		accountAutoRenewPreviousExpiresKey: previousExpiresAt.UTC().Format(time.RFC3339),
		accountAutoRenewSummaryKey:         strings.TrimSpace(summary),
	}
	if nextExpiresAt != nil && !nextExpiresAt.IsZero() {
		out[accountAutoRenewNextExpiresKey] = nextExpiresAt.UTC().Format(time.RFC3339)
	} else {
		out[accountAutoRenewNextExpiresKey] = nil
	}
	return out
}

func BuildAccountDaily5HTriggerExtra(localDate string, status string, modelID string, summary string) map[string]any {
	return map[string]any{
		accountDaily5HLastLocalDateKey: strings.TrimSpace(localDate),
		accountDaily5HLastStatusKey:    strings.TrimSpace(status),
		accountDaily5HLastModelIDKey:   strings.TrimSpace(modelID),
		accountDaily5HLastSummaryKey:   strings.TrimSpace(summary),
	}
}

func AccountDaily5HLastLocalDate(extra map[string]any) string {
	return strings.TrimSpace(stringValueFromAny(extra[accountDaily5HLastLocalDateKey]))
}

func AccountDaily5HLastSummary(extra map[string]any) string {
	return strings.TrimSpace(stringValueFromAny(extra[accountDaily5HLastSummaryKey]))
}

func AccountExpiryProbePriorityUntil(account *Account) *time.Time {
	if account == nil {
		return nil
	}
	return parseAccountExpiryProbeTime(account.Extra, accountExpiryProbePriorityUntilKey)
}

func AccountHasActiveExpiryProbePriority(account *Account, now time.Time) bool {
	if account == nil {
		return false
	}
	priorityUntil := AccountExpiryProbePriorityUntil(account)
	if priorityUntil == nil {
		return false
	}
	return priorityUntil.After(now.UTC())
}

func IsManagedRuntimeAccount(account *Account) bool {
	if account == nil {
		return false
	}
	return NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleNormal
}

func accountDaily5HAccountType(account *Account) string {
	if account == nil {
		return ""
	}
	switch {
	case account.IsOpenAIOAuth():
		return AccountDaily5HTypeOpenAI
	case account.IsAnthropicOAuthOrSetupToken():
		return AccountDaily5HTypeAnthropic
	case account.IsGemini() && account.Type == AccountTypeOAuth:
		return AccountDaily5HTypeGemini
	default:
		return ""
	}
}
