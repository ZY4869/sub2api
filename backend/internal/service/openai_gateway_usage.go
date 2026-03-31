package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func openAIThinkingEnabledFromReasoningEffort(effort *string) *bool {
	if effort == nil {
		return nil
	}
	normalized := strings.TrimSpace(strings.ToLower(*effort))
	switch normalized {
	case "none":
		disabled := false
		return &disabled
	case "low", "medium", "high", "xhigh":
		enabled := true
		return &enabled
	default:
		return nil
	}
}

type OpenAIRecordUsageInput struct {
	Result             *OpenAIForwardResult
	APIKey             *APIKey
	User               *User
	Account            *Account
	Subscription       *UserSubscription
	InboundEndpoint    string
	UpstreamEndpoint   string
	UpstreamURL        string
	UpstreamService    string
	UserAgent          string
	IPAddress          string
	RequestPayloadHash string
	APIKeyService      APIKeyQuotaUpdater
}

func (s *OpenAIGatewayService) RecordUsage(ctx context.Context, input *OpenAIRecordUsageInput) error {
	result := input.Result

	apiKey := input.APIKey
	user := input.User
	account := input.Account
	subscription := input.Subscription

	// input_tokens includes cache_read tokens; cached reads are not billed at input price.
	actualInputTokens := result.Usage.InputTokens - result.Usage.CacheReadInputTokens
	if actualInputTokens < 0 {
		actualInputTokens = 0
	}

	// Calculate cost.
	tokens := UsageTokens{
		InputTokens:         actualInputTokens,
		OutputTokens:        result.Usage.OutputTokens,
		CacheCreationTokens: result.Usage.CacheCreationInputTokens,
		CacheReadTokens:     result.Usage.CacheReadInputTokens,
	}

	// Resolve rate multiplier.
	multiplier := s.cfg.Default.RateMultiplier
	if apiKey.GroupID != nil && apiKey.Group != nil {
		resolver := s.userGroupRateResolver
		if resolver == nil {
			resolver = newUserGroupRateResolver(nil, nil, resolveUserGroupRateCacheTTL(s.cfg), nil, "service.openai_gateway")
		}
		multiplier = resolver.Resolve(ctx, user.ID, *apiKey.GroupID, apiKey.Group.RateMultiplier)
	}

	billingModel := forwardResultBillingModel(result.Model, result.UpstreamModel)
	serviceTier := ""
	if result.ServiceTier != nil {
		serviceTier = strings.TrimSpace(*result.ServiceTier)
	}
	cost, err := s.billingService.CalculateCostWithServiceTier(billingModel, tokens, multiplier, serviceTier)
	if err != nil {
		cost = &CostBreakdown{ActualCost: 0}
	}

	// Determine billing type.
	isSubscriptionBilling := subscription != nil && apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	billingType := BillingTypeBalance
	if isSubscriptionBilling {
		billingType = BillingTypeSubscription
	}

	durationMs := int(result.Duration.Milliseconds())
	accountRateMultiplier := account.BillingRateMultiplier()
	requestID := resolveUsageBillingRequestID(ctx, result.RequestID)

	usageLog := &UsageLog{
		UserID:                user.ID,
		APIKeyID:              apiKey.ID,
		AccountID:             account.ID,
		RequestID:             requestID,
		Model:                 result.Model,
		RequestedModel:        result.Model,
		UpstreamModel:         optionalNonEqualStringPtr(result.UpstreamModel, result.Model),
		ServiceTier:           result.ServiceTier,
		ReasoningEffort:       result.ReasoningEffort,
		ThinkingEnabled:       openAIThinkingEnabledFromReasoningEffort(result.ReasoningEffort),
		InboundEndpoint:       optionalTrimmedStringPtr(input.InboundEndpoint),
		UpstreamEndpoint:      optionalTrimmedStringPtr(input.UpstreamEndpoint),
		UpstreamURL:           optionalTrimmedStringPtr(ResolveUsageLogUpstreamURL(account, input.UpstreamURL)),
		UpstreamService:       optionalTrimmedStringPtr(ResolveUsageLogUpstreamService(account, input.UpstreamService)),
		InputTokens:           actualInputTokens,
		OutputTokens:          result.Usage.OutputTokens,
		CacheCreationTokens:   result.Usage.CacheCreationInputTokens,
		CacheReadTokens:       result.Usage.CacheReadInputTokens,
		InputCost:             cost.InputCost,
		OutputCost:            cost.OutputCost,
		CacheCreationCost:     cost.CacheCreationCost,
		CacheReadCost:         cost.CacheReadCost,
		TotalCost:             cost.TotalCost,
		ActualCost:            cost.ActualCost,
		RateMultiplier:        multiplier,
		AccountRateMultiplier: &accountRateMultiplier,
		BillingType:           billingType,
		Status:                UsageLogStatusSucceeded,
		Stream:                result.Stream,
		OpenAIWSMode:          result.OpenAIWSMode,
		DurationMs:            &durationMs,
		FirstTokenMs:          result.FirstTokenMs,
		CreatedAt:             time.Now(),
	}

	if input.UserAgent != "" {
		usageLog.UserAgent = &input.UserAgent
	}
	if input.IPAddress != "" {
		usageLog.IPAddress = &input.IPAddress
	}
	if apiKey.GroupID != nil {
		usageLog.GroupID = apiKey.GroupID
	}
	if subscription != nil {
		usageLog.SubscriptionID = &subscription.ID
	}
	if simulatedClient := NormalizeUsageLogSimulatedClient(result.SimulatedClient); simulatedClient != nil {
		usageLog.SimulatedClient = simulatedClient
	}

	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")
		logger.LegacyPrintf("service.openai_gateway", "[SIMPLE MODE] Usage recorded (not billed): user=%d, tokens=%d", usageLog.UserID, usageLog.TotalTokens())
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
		return nil
	}

	if _, billingErr := applyUsageBilling(ctx, requestID, usageLog, &postUsageBillingParams{
		Cost:                  cost,
		User:                  user,
		APIKey:                apiKey,
		Account:               account,
		Subscription:          subscription,
		RequestPayloadHash:    resolveUsageBillingPayloadFingerprint(ctx, input.RequestPayloadHash),
		IsSubscriptionBill:    isSubscriptionBilling,
		AccountRateMultiplier: accountRateMultiplier,
		APIKeyService:         input.APIKeyService,
	}, s.billingDeps(), s.usageBillingRepo); billingErr != nil {
		return billingErr
	}

	writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")
	return nil
}
func ParseCodexRateLimitHeaders(headers http.Header) *OpenAICodexUsageSnapshot {
	snapshot := &OpenAICodexUsageSnapshot{}
	hasData := false
	parseFloat := func(key string) *float64 {
		if v := headers.Get(key); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return &f
			}
		}
		return nil
	}
	parseInt := func(key string) *int {
		if v := headers.Get(key); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return &i
			}
		}
		return nil
	}
	if v := parseFloat("x-codex-primary-used-percent"); v != nil {
		snapshot.PrimaryUsedPercent = v
		hasData = true
	}
	if v := parseInt("x-codex-primary-reset-after-seconds"); v != nil {
		snapshot.PrimaryResetAfterSeconds = v
		hasData = true
	}
	if v := parseInt("x-codex-primary-window-minutes"); v != nil {
		snapshot.PrimaryWindowMinutes = v
		hasData = true
	}
	if v := parseFloat("x-codex-secondary-used-percent"); v != nil {
		snapshot.SecondaryUsedPercent = v
		hasData = true
	}
	if v := parseInt("x-codex-secondary-reset-after-seconds"); v != nil {
		snapshot.SecondaryResetAfterSeconds = v
		hasData = true
	}
	if v := parseInt("x-codex-secondary-window-minutes"); v != nil {
		snapshot.SecondaryWindowMinutes = v
		hasData = true
	}
	if v := parseFloat("x-codex-primary-over-secondary-limit-percent"); v != nil {
		snapshot.PrimaryOverSecondaryPercent = v
		hasData = true
	}
	if !hasData {
		return nil
	}
	snapshot.UpdatedAt = time.Now().Format(time.RFC3339)
	return snapshot
}
func codexSnapshotBaseTime(snapshot *OpenAICodexUsageSnapshot, fallback time.Time) time.Time {
	if snapshot == nil {
		return fallback
	}
	if snapshot.UpdatedAt == "" {
		return fallback
	}
	base, err := time.Parse(time.RFC3339, snapshot.UpdatedAt)
	if err != nil {
		return fallback
	}
	return base
}
func codexResetAtRFC3339(base time.Time, resetAfterSeconds *int) *string {
	if resetAfterSeconds == nil {
		return nil
	}
	sec := *resetAfterSeconds
	if sec < 0 {
		sec = 0
	}
	resetAt := base.Add(time.Duration(sec) * time.Second).Format(time.RFC3339)
	return &resetAt
}
func buildCodexUsageExtraUpdates(snapshot *OpenAICodexUsageSnapshot, fallbackNow time.Time) map[string]any {
	if snapshot == nil {
		return nil
	}
	baseTime := codexSnapshotBaseTime(snapshot, fallbackNow)
	updates := make(map[string]any)
	if snapshot.PrimaryUsedPercent != nil {
		updates["codex_primary_used_percent"] = *snapshot.PrimaryUsedPercent
	}
	if snapshot.PrimaryResetAfterSeconds != nil {
		updates["codex_primary_reset_after_seconds"] = *snapshot.PrimaryResetAfterSeconds
	}
	if snapshot.PrimaryWindowMinutes != nil {
		updates["codex_primary_window_minutes"] = *snapshot.PrimaryWindowMinutes
	}
	if snapshot.SecondaryUsedPercent != nil {
		updates["codex_secondary_used_percent"] = *snapshot.SecondaryUsedPercent
	}
	if snapshot.SecondaryResetAfterSeconds != nil {
		updates["codex_secondary_reset_after_seconds"] = *snapshot.SecondaryResetAfterSeconds
	}
	if snapshot.SecondaryWindowMinutes != nil {
		updates["codex_secondary_window_minutes"] = *snapshot.SecondaryWindowMinutes
	}
	if snapshot.PrimaryOverSecondaryPercent != nil {
		updates["codex_primary_over_secondary_percent"] = *snapshot.PrimaryOverSecondaryPercent
	}
	updates["codex_usage_updated_at"] = baseTime.Format(time.RFC3339)
	if normalized := snapshot.Normalize(); normalized != nil {
		if normalized.Used5hPercent != nil {
			updates["codex_5h_used_percent"] = *normalized.Used5hPercent
		}
		if normalized.Reset5hSeconds != nil {
			updates["codex_5h_reset_after_seconds"] = *normalized.Reset5hSeconds
		}
		if normalized.Window5hMinutes != nil {
			updates["codex_5h_window_minutes"] = *normalized.Window5hMinutes
		}
		if normalized.Used7dPercent != nil {
			updates["codex_7d_used_percent"] = *normalized.Used7dPercent
		}
		if normalized.Reset7dSeconds != nil {
			updates["codex_7d_reset_after_seconds"] = *normalized.Reset7dSeconds
		}
		if normalized.Window7dMinutes != nil {
			updates["codex_7d_window_minutes"] = *normalized.Window7dMinutes
		}
		if reset5hAt := codexResetAtRFC3339(baseTime, normalized.Reset5hSeconds); reset5hAt != nil {
			updates["codex_5h_reset_at"] = *reset5hAt
		}
		if reset7dAt := codexResetAtRFC3339(baseTime, normalized.Reset7dSeconds); reset7dAt != nil {
			updates["codex_7d_reset_at"] = *reset7dAt
		}
	}
	return updates
}
func codexUsagePercentExhausted(value *float64) bool {
	return value != nil && *value >= 100-1e-9
}
func codexRateLimitResetAtFromSnapshot(snapshot *OpenAICodexUsageSnapshot, fallbackNow time.Time) *time.Time {
	if snapshot == nil {
		return nil
	}
	normalized := snapshot.Normalize()
	if normalized == nil {
		return nil
	}
	baseTime := codexSnapshotBaseTime(snapshot, fallbackNow)
	if codexUsagePercentExhausted(normalized.Used7dPercent) && normalized.Reset7dSeconds != nil {
		resetAt := baseTime.Add(time.Duration(*normalized.Reset7dSeconds) * time.Second)
		return &resetAt
	}
	if codexUsagePercentExhausted(normalized.Used5hPercent) && normalized.Reset5hSeconds != nil {
		resetAt := baseTime.Add(time.Duration(*normalized.Reset5hSeconds) * time.Second)
		return &resetAt
	}
	return nil
}
func codexRateLimitReasonFromSnapshot(snapshot *OpenAICodexUsageSnapshot) string {
	if snapshot == nil {
		return AccountRateLimitReason429
	}
	normalized := snapshot.Normalize()
	if normalized == nil {
		return AccountRateLimitReason429
	}
	if codexUsagePercentExhausted(normalized.Used7dPercent) {
		return AccountRateLimitReasonUsage7d
	}
	if codexUsagePercentExhausted(normalized.Used5hPercent) {
		return AccountRateLimitReasonUsage5h
	}
	return AccountRateLimitReason429
}
func codexRateLimitResetAtFromExtra(extra map[string]any, now time.Time) *time.Time {
	if len(extra) == 0 {
		return nil
	}
	if progress := buildCodexUsageProgressFromExtra(extra, "7d", now); progress != nil && codexUsagePercentExhausted(&progress.Utilization) && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) {
		resetAt := progress.ResetsAt.UTC()
		return &resetAt
	}
	if progress := buildCodexUsageProgressFromExtra(extra, "5h", now); progress != nil && codexUsagePercentExhausted(&progress.Utilization) && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) {
		resetAt := progress.ResetsAt.UTC()
		return &resetAt
	}
	return nil
}
func applyOpenAICodexRateLimitFromExtra(account *Account, now time.Time) (*time.Time, bool) {
	if account == nil || !account.IsOpenAI() {
		return nil, false
	}
	resetAt := codexRateLimitResetAtFromExtra(account.Extra, now)
	if resetAt == nil {
		return nil, false
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) && !account.RateLimitResetAt.Before(*resetAt) {
		return account.RateLimitResetAt, false
	}
	account.RateLimitResetAt = resetAt
	return resetAt, true
}
func syncOpenAICodexRateLimitFromExtra(ctx context.Context, repo AccountRepository, account *Account, now time.Time) *time.Time {
	resetAt, changed := applyOpenAICodexRateLimitFromExtra(account, now)
	if !changed || resetAt == nil || repo == nil || account == nil || account.ID <= 0 {
		return resetAt
	}
	_ = setAccountRateLimited(ctx, repo, account.ID, *resetAt, AccountRateLimitReason(account, now))
	return resetAt
}
func (s *OpenAIGatewayService) updateCodexUsageSnapshot(ctx context.Context, accountID int64, snapshot *OpenAICodexUsageSnapshot) {
	if snapshot == nil {
		return
	}
	if s == nil || s.accountRepo == nil {
		return
	}
	now := time.Now()
	updates := buildCodexUsageExtraUpdates(snapshot, now)
	resetAt := codexRateLimitResetAtFromSnapshot(snapshot, now)
	if len(updates) == 0 && resetAt == nil {
		return
	}
	shouldPersistUpdates := len(updates) > 0 && s.getCodexSnapshotThrottle().Allow(accountID, now)
	if !shouldPersistUpdates && resetAt == nil {
		return
	}
	updateParentCtx := ctx
	if updateParentCtx == nil {
		updateParentCtx = context.Background()
	}
	go func() {
		updateCtx, cancel := context.WithTimeout(updateParentCtx, 5*time.Second)
		defer cancel()
		if shouldPersistUpdates {
			_ = s.accountRepo.UpdateExtra(updateCtx, accountID, updates)
		}
		if resetAt != nil {
			_ = setAccountRateLimited(updateCtx, s.accountRepo, accountID, *resetAt, codexRateLimitReasonFromSnapshot(snapshot))
		}
	}()
}
func (s *OpenAIGatewayService) UpdateCodexUsageSnapshotFromHeaders(ctx context.Context, accountID int64, headers http.Header) {
	if accountID <= 0 || headers == nil {
		return
	}
	if snapshot := ParseCodexRateLimitHeaders(headers); snapshot != nil {
		s.updateCodexUsageSnapshot(ctx, accountID, snapshot)
	}
}
