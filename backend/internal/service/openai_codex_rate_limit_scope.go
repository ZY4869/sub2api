package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

const (
	openAICodexScopeNormal        = "gpt-5.3-codex"
	openAICodexScopeSpark         = "gpt-5.3-codex-spark"
	codexAccountAll7dExhaustedKey = "codex_account_7d_all_exhausted"
	codexSpark5hUsedPercentKey    = "codex_spark_5h_used_percent"
	codexSpark5hResetAfterKey     = "codex_spark_5h_reset_after_seconds"
	codexSpark5hResetAtKey        = "codex_spark_5h_reset_at"
	codexSpark5hWindowMinutesKey  = "codex_spark_5h_window_minutes"
	codexSpark7dUsedPercentKey    = "codex_spark_7d_used_percent"
	codexSpark7dResetAfterKey     = "codex_spark_7d_reset_after_seconds"
	codexSpark7dResetAtKey        = "codex_spark_7d_reset_at"
	codexSpark7dWindowMinutesKey  = "codex_spark_7d_window_minutes"
)

type openAICodexRequestModelContextKey struct{}
type openAICodexSuccessfulSnapshotContextKey struct{}

type openAICodexRateLimitState struct {
	Scope          string
	ScopeResetAt   *time.Time
	ScopeReason    string
	AccountResetAt *time.Time
	All7dExhausted bool
}

func WithOpenAICodexRequestModel(ctx context.Context, model string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return ctx
	}
	return context.WithValue(ctx, openAICodexRequestModelContextKey{}, model)
}

func withOpenAICodexSuccessfulSnapshot(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAICodexSuccessfulSnapshotContextKey{}, true)
}

func openAICodexSuccessfulSnapshotFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	ok, _ := ctx.Value(openAICodexSuccessfulSnapshotContextKey{}).(bool)
	return ok
}

func withOpenAICodexRequestModelFallback(ctx context.Context, models ...string) context.Context {
	if strings.TrimSpace(openAICodexRequestModelFromContext(ctx)) != "" {
		return ctx
	}
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		return WithOpenAICodexRequestModel(ctx, model)
	}
	return ctx
}

func openAICodexRequestModelFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	model, _ := ctx.Value(openAICodexRequestModelContextKey{}).(string)
	return strings.TrimSpace(model)
}

func isOpenAIProPlan(account *Account) bool {
	if account == nil {
		return false
	}
	return normalizeOpenAIPlanType(account.GetCredential("plan_type")) == "pro"
}

func normalizeOpenAIRuntimeQuotaCandidate(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return ""
	}
	normalized = strings.TrimPrefix(normalized, "models/")
	replacer := strings.NewReplacer("_", "-", " ", "-", "/", "-")
	normalized = replacer.Replace(normalized)
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}
	return strings.Trim(normalized, "-")
}

func openAIRuntimeQuotaCandidateVariants(model string) []string {
	values := []string{
		model,
		normalizeRegistryID(model),
		NormalizeModelCatalogModelID(model),
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeOpenAIRuntimeQuotaCandidate(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeOpenAICodexQuotaScope(model string) string {
	normalized := normalizeOpenAIRuntimeQuotaCandidate(model)
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "gpt-5.3-codex-spark") {
		return openAICodexScopeSpark
	}
	if strings.HasPrefix(normalized, "gpt-5.3-codex") {
		return openAICodexScopeNormal
	}
	if strings.HasPrefix(normalized, "gpt-5.3") {
		return openAICodexScopeNormal
	}
	return ""
}

func resolveOpenAICodexQuotaScopeWithCandidates(account *Account, candidates ...string) string {
	var (
		hasCandidate bool
		hasNormal    bool
		hasSpark     bool
	)
	for _, candidate := range candidates {
		for _, variant := range openAIRuntimeQuotaCandidateVariants(candidate) {
			hasCandidate = true
			switch normalizeOpenAICodexQuotaScope(variant) {
			case openAICodexScopeSpark:
				hasSpark = true
			case openAICodexScopeNormal:
				hasNormal = true
			}
		}
	}
	if isOpenAIProPlan(account) {
		if hasSpark {
			return openAICodexScopeSpark
		}
		if hasCandidate {
			return openAICodexScopeNormal
		}
		return ""
	}
	if hasSpark || hasNormal {
		return openAICodexScopeNormal
	}
	return ""
}

func openAIRuntimeQuotaModelCandidates(account *Account, requestedModel string, extras ...string) []string {
	result := make([]string, 0, len(extras)+2)
	seen := make(map[string]struct{}, len(extras)+2)
	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	appendCandidate(requestedModel)
	if account != nil {
		appendCandidate(account.GetMappedModel(requestedModel))
	}
	for _, candidate := range extras {
		appendCandidate(candidate)
	}
	return result
}

func resolveOpenAICodexQuotaScope(account *Account, model string) string {
	return resolveOpenAICodexQuotaScopeWithCandidates(account, openAIRuntimeQuotaModelCandidates(account, model)...)
}

func resolveOpenAICodexQuotaScopeFromContext(ctx context.Context, account *Account) (string, bool) {
	if !isOpenAIProPlan(account) {
		return openAICodexScopeNormal, true
	}
	scope := resolveOpenAICodexQuotaScopeWithCandidates(account, openAIRuntimeQuotaModelCandidates(account, openAICodexRequestModelFromContext(ctx))...)
	if scope == "" {
		return "", false
	}
	return scope, true
}

func openAIQuotaScopeRateLimitRemaining(account *Account, candidates ...string) (string, time.Duration) {
	if account == nil {
		return "", 0
	}
	scope := resolveOpenAICodexQuotaScopeWithCandidates(account, candidates...)
	if scope == "" {
		return "", 0
	}
	return scope, account.getRateLimitRemainingForKey(scope)
}

func openAIAccountAll7dRateLimited(account *Account, now time.Time) (*time.Time, bool) {
	if account == nil || account.RateLimitResetAt == nil || !now.Before(*account.RateLimitResetAt) {
		return nil, false
	}
	if AccountRateLimitReason(account, now) != AccountRateLimitReasonUsage7dAll {
		return nil, false
	}
	resetAt := account.RateLimitResetAt.UTC()
	return &resetAt, true
}

func buildCodexUsageExtraUpdatesForScope(scope string, snapshot *OpenAICodexUsageSnapshot, fallbackNow time.Time) map[string]any {
	if scope == "" || scope == openAICodexScopeNormal {
		return buildCodexUsageExtraUpdates(snapshot, fallbackNow)
	}
	if snapshot == nil {
		return nil
	}

	baseTime := codexSnapshotBaseTime(snapshot, fallbackNow)
	updates := map[string]any{
		"codex_usage_updated_at": baseTime.Format(time.RFC3339),
	}
	normalized := snapshot.Normalize()
	if normalized == nil {
		return updates
	}
	if normalized.Used5hPercent != nil {
		updates[codexSpark5hUsedPercentKey] = *normalized.Used5hPercent
	}
	if normalized.Reset5hSeconds != nil {
		updates[codexSpark5hResetAfterKey] = *normalized.Reset5hSeconds
	}
	if normalized.Window5hMinutes != nil {
		updates[codexSpark5hWindowMinutesKey] = *normalized.Window5hMinutes
	}
	if normalized.Used7dPercent != nil {
		updates[codexSpark7dUsedPercentKey] = *normalized.Used7dPercent
	}
	if normalized.Reset7dSeconds != nil {
		updates[codexSpark7dResetAfterKey] = *normalized.Reset7dSeconds
	}
	if normalized.Window7dMinutes != nil {
		updates[codexSpark7dWindowMinutesKey] = *normalized.Window7dMinutes
	}
	if reset5hAt := codexResetAtRFC3339(baseTime, normalized.Reset5hSeconds); reset5hAt != nil {
		updates[codexSpark5hResetAtKey] = *reset5hAt
	}
	if reset7dAt := codexResetAtRFC3339(baseTime, normalized.Reset7dSeconds); reset7dAt != nil {
		updates[codexSpark7dResetAtKey] = *reset7dAt
	}
	return updates
}

func buildScopedCodexUsageProgressFromExtra(extra map[string]any, scope string, window string, now time.Time) *UsageProgress {
	if scope == "" || scope == openAICodexScopeNormal {
		return buildCodexUsageProgressFromExtra(extra, window, now)
	}
	if len(extra) == 0 {
		return nil
	}

	var (
		usedPercentKey string
		resetAfterKey  string
		resetAtKey     string
	)

	switch window {
	case "5h":
		usedPercentKey = codexSpark5hUsedPercentKey
		resetAfterKey = codexSpark5hResetAfterKey
		resetAtKey = codexSpark5hResetAtKey
	case "7d":
		usedPercentKey = codexSpark7dUsedPercentKey
		resetAfterKey = codexSpark7dResetAfterKey
		resetAtKey = codexSpark7dResetAtKey
	default:
		return nil
	}

	usedRaw, ok := extra[usedPercentKey]
	if !ok {
		return nil
	}

	progress := &UsageProgress{Utilization: parseExtraFloat64(usedRaw)}
	if resetAtRaw, ok := extra[resetAtKey]; ok {
		if resetAt, err := parseTime(strings.TrimSpace(parseExtraString(resetAtRaw))); err == nil {
			progress.ResetsAt = &resetAt
			progress.RemainingSeconds = int(time.Until(resetAt).Seconds())
			if progress.RemainingSeconds < 0 {
				progress.RemainingSeconds = 0
			}
		}
	}
	if progress.ResetsAt == nil {
		if resetAfterSeconds := parseExtraInt(extra[resetAfterKey]); resetAfterSeconds > 0 {
			base := now
			if updatedAtRaw, ok := extra["codex_usage_updated_at"]; ok {
				if updatedAt, err := parseTime(strings.TrimSpace(parseExtraString(updatedAtRaw))); err == nil {
					base = updatedAt
				}
			}
			resetAt := base.Add(time.Duration(resetAfterSeconds) * time.Second)
			progress.ResetsAt = &resetAt
			progress.RemainingSeconds = int(time.Until(resetAt).Seconds())
			if progress.RemainingSeconds < 0 {
				progress.RemainingSeconds = 0
			}
		}
	}
	if progress.ResetsAt != nil && !now.Before(*progress.ResetsAt) {
		progress.Utilization = 0
	}
	return progress
}

func codexRateLimitResetAtFromExtraForScope(extra map[string]any, scope string, now time.Time) *time.Time {
	if progress := buildScopedCodexUsageProgressFromExtra(extra, scope, "7d", now); progress != nil && codexUsagePercentExhausted(&progress.Utilization) && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) {
		resetAt := progress.ResetsAt.UTC()
		return &resetAt
	}
	if progress := buildScopedCodexUsageProgressFromExtra(extra, scope, "5h", now); progress != nil && codexUsagePercentExhausted(&progress.Utilization) && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) {
		resetAt := progress.ResetsAt.UTC()
		return &resetAt
	}
	return nil
}

func codexRateLimitReasonFromExtraForScope(extra map[string]any, scope string, now time.Time) string {
	if progress := buildScopedCodexUsageProgressFromExtra(extra, scope, "7d", now); progress != nil && codexUsagePercentExhausted(&progress.Utilization) && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) {
		return AccountRateLimitReasonUsage7d
	}
	if progress := buildScopedCodexUsageProgressFromExtra(extra, scope, "5h", now); progress != nil && codexUsagePercentExhausted(&progress.Utilization) && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) {
		return AccountRateLimitReasonUsage5h
	}
	return ""
}

func codexAccountAll7dResetAtFromExtra(account *Account, extra map[string]any, now time.Time) (*time.Time, bool) {
	if account == nil || !isOpenAIProPlan(account) || len(extra) == 0 {
		return nil, false
	}
	normalProgress := buildScopedCodexUsageProgressFromExtra(extra, openAICodexScopeNormal, "7d", now)
	sparkProgress := buildScopedCodexUsageProgressFromExtra(extra, openAICodexScopeSpark, "7d", now)
	if normalProgress == nil || sparkProgress == nil {
		return nil, false
	}
	if !codexUsagePercentExhausted(&normalProgress.Utilization) || !codexUsagePercentExhausted(&sparkProgress.Utilization) {
		return nil, false
	}
	if normalProgress.ResetsAt == nil || sparkProgress.ResetsAt == nil {
		return nil, false
	}
	if !now.Before(*normalProgress.ResetsAt) || !now.Before(*sparkProgress.ResetsAt) {
		return nil, false
	}
	resetAt := normalProgress.ResetsAt.UTC()
	if sparkProgress.ResetsAt.After(resetAt) {
		resetAt = sparkProgress.ResetsAt.UTC()
	}
	return &resetAt, true
}

func syncOpenAICodexRateLimitState(ctx context.Context, repo AccountRepository, account *Account, updates map[string]any, now time.Time) *openAICodexRateLimitState {
	if account == nil || !account.IsOpenAI() {
		return nil
	}

	mergedExtra := mergeStringAnyMap(account.Extra, updates)
	if mergedExtra == nil {
		mergedExtra = map[string]any{}
	}

	state := &openAICodexRateLimitState{}
	scope, _ := resolveOpenAICodexQuotaScopeFromContext(ctx, account)
	state.Scope = scope
	if scope != "" {
		state.ScopeResetAt = codexRateLimitResetAtFromExtraForScope(mergedExtra, scope, now)
		state.ScopeReason = codexRateLimitReasonFromExtraForScope(mergedExtra, scope, now)
	}

	state.AccountResetAt, state.All7dExhausted = codexAccountAll7dResetAtFromExtra(account, mergedExtra, now)

	updatesToPersist := cloneStringAnyMap(updates)
	currentAll7d := parseExtraBool(account.Extra[codexAccountAll7dExhaustedKey])
	if currentAll7d != state.All7dExhausted {
		if updatesToPersist == nil {
			updatesToPersist = map[string]any{}
		}
		updatesToPersist[codexAccountAll7dExhaustedKey] = state.All7dExhausted
	}

	modelLimitsChanged := syncScopedOpenAICodexModelRateLimit(ctx, repo, account, openAICodexScopeNormal, mergedExtra, now)
	if isOpenAIProPlan(account) && syncScopedOpenAICodexModelRateLimit(ctx, repo, account, openAICodexScopeSpark, mergedExtra, now) {
		modelLimitsChanged = true
	}
	if modelLimitsChanged {
		if updatesToPersist == nil {
			updatesToPersist = map[string]any{}
		}
		updatesToPersist[modelRateLimitsKey] = cloneLocalOpenAICodexModelRateLimits(account)
	}

	if len(updatesToPersist) > 0 {
		mergeAccountExtra(account, updatesToPersist)
		if repo != nil && account.ID > 0 {
			if err := repo.UpdateExtra(ctx, account.ID, updatesToPersist); err != nil {
				slog.Warn("openai_codex_snapshot_update_failed", "account_id", account.ID, "error", err)
			}
		}
	}

	if state.AccountResetAt != nil {
		currentAccountResetAt := account.RateLimitResetAt
		currentReason := NormalizeAccountRateLimitReasonInput(parseExtraString(account.Extra["rate_limit_reason"]))
		if account.Extra == nil {
			account.Extra = map[string]any{}
		}
		account.RateLimitedAt = &now
		account.RateLimitResetAt = state.AccountResetAt
		account.Extra["rate_limit_reason"] = AccountRateLimitReasonUsage7dAll
		if repo != nil && account.ID > 0 && shouldPersistOpenAICodexAccountRateLimit(currentAccountResetAt, currentReason, *state.AccountResetAt, now) {
			if err := setAccountRateLimited(ctx, repo, account.ID, *state.AccountResetAt, AccountRateLimitReasonUsage7dAll); err != nil {
				slog.Warn("openai_codex_account_all_7d_limit_failed", "account_id", account.ID, "reset_at", *state.AccountResetAt, "error", err)
			} else {
				slog.Info("openai_codex_account_all_7d_limited", "account_id", account.ID, "reset_at", *state.AccountResetAt)
			}
		}
		return state
	}

	if shouldClearOpenAICodexAccountRateLimit(account, now, openAICodexSuccessfulSnapshotFromContext(ctx)) {
		account.RateLimitedAt = nil
		account.RateLimitResetAt = nil
		delete(account.Extra, "rate_limit_reason")
		if repo != nil && account.ID > 0 {
			if err := repo.ClearRateLimit(ctx, account.ID); err != nil {
				slog.Warn("openai_codex_account_limit_clear_failed", "account_id", account.ID, "error", err)
			} else {
				slog.Info("openai_codex_account_limit_cleared", "account_id", account.ID)
			}
		}
	}

	return state
}

func syncScopedOpenAICodexModelRateLimit(ctx context.Context, repo AccountRepository, account *Account, scope string, extra map[string]any, now time.Time) bool {
	resetAt := codexRateLimitResetAtFromExtraForScope(extra, scope, now)
	if resetAt == nil {
		return clearLocalOpenAICodexModelRateLimit(account, scope)
	}
	reason := codexRateLimitReasonFromExtraForScope(extra, scope, now)
	current := account.modelRateLimitResetAt(scope)
	applyLocalOpenAICodexModelRateLimit(account, scope, *resetAt, now)
	if repo == nil || account == nil || account.ID <= 0 || !shouldPersistOpenAICodexModelRateLimit(current, *resetAt, now) {
		return false
	}
	if err := repo.SetModelRateLimit(ctx, account.ID, scope, *resetAt); err != nil {
		slog.Warn("openai_codex_model_limit_failed", "account_id", account.ID, "scope", scope, "reason", reason, "reset_at", *resetAt, "error", err)
		return false
	}
	slog.Info("openai_codex_model_limited", "account_id", account.ID, "scope", scope, "reason", reason, "reset_at", *resetAt)
	return true
}

func applyLocalOpenAICodexModelRateLimit(account *Account, scope string, resetAt time.Time, now time.Time) {
	if account == nil || scope == "" {
		return
	}
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	current := account.modelRateLimitResetAt(scope)
	if current != nil && now.Before(*current) && !current.Before(resetAt) {
		return
	}
	limits, _ := account.Extra[modelRateLimitsKey].(map[string]any)
	if limits == nil {
		limits = map[string]any{}
		account.Extra[modelRateLimitsKey] = limits
	}
	limits[scope] = map[string]any{
		"rate_limited_at":     now.UTC().Format(time.RFC3339),
		"rate_limit_reset_at": resetAt.UTC().Format(time.RFC3339),
	}
}

func clearLocalOpenAICodexModelRateLimit(account *Account, scope string) bool {
	if account == nil || account.Extra == nil || scope == "" {
		return false
	}
	limits, ok := account.Extra[modelRateLimitsKey].(map[string]any)
	if !ok || limits == nil {
		return false
	}
	if _, exists := limits[scope]; !exists {
		return false
	}
	delete(limits, scope)
	account.Extra[modelRateLimitsKey] = limits
	return true
}

func cloneLocalOpenAICodexModelRateLimits(account *Account) map[string]any {
	if account == nil || account.Extra == nil {
		return map[string]any{}
	}
	limits, ok := account.Extra[modelRateLimitsKey].(map[string]any)
	if !ok || limits == nil {
		return map[string]any{}
	}
	return cloneStringAnyMap(limits)
}

func shouldPersistOpenAICodexModelRateLimit(current *time.Time, resetAt time.Time, now time.Time) bool {
	if current == nil || !now.Before(*current) {
		return true
	}
	return current.Before(resetAt)
}

func shouldPersistOpenAICodexAccountRateLimit(current *time.Time, currentReason string, resetAt time.Time, now time.Time) bool {
	if current != nil && now.Before(*current) && !current.Before(resetAt) && currentReason == AccountRateLimitReasonUsage7dAll {
		return false
	}
	return true
}

func shouldClearOpenAICodexAccountRateLimit(account *Account, now time.Time, allowUsage7dAllRecovery bool) bool {
	if account == nil || account.RateLimitResetAt == nil || !now.Before(*account.RateLimitResetAt) {
		return false
	}
	reason := NormalizeAccountRateLimitReasonInput(parseExtraString(account.Extra["rate_limit_reason"]))
	switch reason {
	case AccountRateLimitReasonUsage7dAll:
		if !allowUsage7dAllRecovery {
			return false
		}
		resetAt, ok := codexAccountAll7dResetAtFromExtra(account, account.Extra, now)
		return !ok || resetAt == nil || !now.Before(*resetAt)
	case AccountRateLimitReasonUsage5h, AccountRateLimitReasonUsage7d:
		return true
	default:
		return false
	}
}
