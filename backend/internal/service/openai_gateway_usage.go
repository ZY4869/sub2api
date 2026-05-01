package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"log/slog"
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
	usageProvider := ResolveUsageLogUpstreamService(account, input.UpstreamService)
	normalizedUsage := normalizeOpenAIUsageForDisplayAndBilling(usageProvider, result.Usage)

	// input_tokens includes cache_read tokens in compatible payloads; DeepSeek also includes miss tokens.
	actualInputTokens := normalizedUsage.DisplayTokens.InputTokens
	tokens := normalizedUsage.BillingTokens

	// Resolve rate multiplier.
	multiplier := s.cfg.Default.RateMultiplier
	if apiKey.GroupID != nil && apiKey.Group != nil {
		resolver := s.userGroupRateResolver
		if resolver == nil {
			resolver = newUserGroupRateResolver(nil, nil, resolveUserGroupRateCacheTTL(s.cfg), nil, "service.openai_gateway")
		}
		multiplier = resolver.Resolve(ctx, user.ID, *apiKey.GroupID, apiKey.Group.RateMultiplier)
	}

	channelResolution := resolveGatewayChannelBilling(ctx, s.channelService, result.Model, result.UpstreamModel, GatewayChannelUsage{
		TotalTokens:       tokens.InputTokens + tokens.OutputTokens + tokens.CacheCreationTokens + tokens.CacheReadTokens,
		ImageOutputTokens: result.Usage.OutputTokens,
	})
	billingModel := result.BillingModel
	if billingModel == "" {
		billingModel = forwardResultBillingModel(result.Model, result.UpstreamModel)
	}
	serviceTier := ""
	if result.ServiceTier != nil {
		serviceTier = strings.TrimSpace(*result.ServiceTier)
	}
	runtimeResult, err := s.billingService.ResolveRuntime(ctx, BillingRuntimeInput{
		Model:           billingModel,
		Provider:        usageProvider,
		Layer:           BillingLayerSale,
		InboundEndpoint: input.InboundEndpoint,
		Tokens:          tokens,
		ImageCount:      result.ImageCount,
		ImageSize:       result.ImageSize,
		MediaType:       result.MediaType,
		ServiceTier:     serviceTier,
		RateMultiplier:  multiplier,
	})
	if err != nil {
		runtimeResult = &BillingRuntimeResult{Cost: &CostBreakdown{ActualCost: 0}}
	}
	if runtimeResult == nil || runtimeResult.Cost == nil {
		runtimeResult = &BillingRuntimeResult{Cost: &CostBreakdown{}}
	}
	applyBillingRuntimeResultMetadataToContext(ctx, runtimeResult)
	cost := runtimeResult.Cost
	var channelPricing *GatewayChannelResolvedPricing
	if channelResolution != nil {
		channelPricing = channelResolution.Pricing
	}
	cost, imageOutputTokens, imageOutputCost := applyChannelPricingOverride(cost, channelPricing, tokens, multiplier, result.ImageCount)

	// Determine billing type.
	isSubscriptionBilling := subscription != nil && apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	billingType := BillingTypeBalance
	if isSubscriptionBilling {
		billingType = BillingTypeSubscription
	}

	durationMs := int(result.Duration.Milliseconds())
	accountRateMultiplier := account.BillingRateMultiplier()
	requestID := resolveUsageBillingRequestID(ctx, result.RequestID)
	billingCurrency := normalizeBillingCurrency(cost.Currency)

	usageLog := &UsageLog{
		UserID:                  user.ID,
		APIKeyID:                apiKey.ID,
		AccountID:               account.ID,
		RequestID:               requestID,
		Model:                   result.Model,
		RequestedModel:          result.Model,
		UpstreamModel:           optionalNonEqualStringPtr(result.UpstreamModel, result.Model),
		ServiceTier:             result.ServiceTier,
		ReasoningEffort:         result.ReasoningEffort,
		ThinkingEnabled:         openAIThinkingEnabledFromReasoningEffort(result.ReasoningEffort),
		InboundEndpoint:         optionalTrimmedStringPtr(input.InboundEndpoint),
		UpstreamEndpoint:        optionalTrimmedStringPtr(input.UpstreamEndpoint),
		UpstreamURL:             optionalTrimmedStringPtr(ResolveUsageLogUpstreamURL(account, input.UpstreamURL)),
		UpstreamService:         optionalTrimmedStringPtr(usageProvider),
		InputTokens:             actualInputTokens,
		OutputTokens:            result.Usage.OutputTokens,
		CacheCreationTokens:     normalizedUsage.DisplayTokens.CacheCreationTokens,
		CacheReadTokens:         normalizedUsage.DisplayTokens.CacheReadTokens,
		InputCost:               cost.InputCost,
		OutputCost:              cost.OutputCost,
		CacheCreationCost:       cost.CacheCreationCost,
		CacheReadCost:           cost.CacheReadCost,
		TotalCost:               cost.TotalCost,
		ActualCost:              cost.ActualCost,
		BillingCurrency:         billingCurrency,
		TotalCostUSDEquivalent:  cost.TotalCostUSDEquivalent,
		ActualCostUSDEquivalent: cost.ActualCostUSDEquivalent,
		USDToCNYRate:            cost.USDToCNYRate,
		FXRateDate:              optionalTrimmedStringPtr(cost.FXRateDate),
		FXLockedAt:              cloneBillingTime(cost.FXLockedAt),
		CostByCurrency:          cloneBillingStringMapFloat64(cost.CostByCurrency),
		ActualCostByCurrency:    cloneBillingStringMapFloat64(cost.ActualCostByCurrency),
		RateMultiplier:          multiplier,
		AccountRateMultiplier:   &accountRateMultiplier,
		BillingType:             billingType,
		Status:                  UsageLogStatusSucceeded,
		Stream:                  result.Stream,
		OpenAIWSMode:            result.OpenAIWSMode,
		DurationMs:              &durationMs,
		FirstTokenMs:            result.FirstTokenMs,
		ImageCount:              result.ImageCount,
		CreatedAt:               time.Now(),
	}
	if result.ImageSize != "" {
		imageSize := result.ImageSize
		usageLog.ImageSize = &imageSize
	}
	if mediaType := strings.TrimSpace(result.MediaType); mediaType != "" {
		usageLog.MediaType = &mediaType
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
	applyGatewayChannelUsageLogMetadata(usageLog, channelResolution, imageOutputTokens, imageOutputCost)

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
func syncOpenAICodexRateLimitFromExtra(ctx context.Context, repo AccountRepository, account *Account, now time.Time) *openAICodexRateLimitState {
	return syncOpenAICodexRateLimitState(ctx, repo, account, nil, now)
}
func (s *OpenAIGatewayService) updateCodexUsageSnapshot(ctx context.Context, accountID int64, snapshot *OpenAICodexUsageSnapshot, finalModels ...string) {
	if snapshot == nil {
		return
	}
	if s == nil || s.accountRepo == nil {
		return
	}
	now := time.Now()
	updateParentCtx := ctx
	if updateParentCtx == nil {
		updateParentCtx = context.Background()
	}
	go func() {
		updateCtx, cancel := context.WithTimeout(updateParentCtx, 5*time.Second)
		defer cancel()
		account, err := s.accountRepo.GetByID(updateCtx, accountID)
		if err != nil || account == nil || !account.IsOpenAI() {
			return
		}
		scope, ok := resolveOpenAICodexSnapshotScopeFromContext(updateCtx, account, finalModels...)
		if !ok {
			slog.Warn("openai_codex_snapshot_scope_missing", "account_id", accountID)
			return
		}
		updateCtx = withOpenAICodexResolvedQuotaScope(updateCtx, scope)
		slog.Info(
			"openai_codex_snapshot_scope_resolved",
			"account_id", accountID,
			"requested_model", openAICodexRequestModelFromContext(updateCtx),
			"upstream_model", firstNonEmptyString(finalModels...),
			"resolved_scope", scope,
			"snapshot_source", "success_header",
		)
		updates := buildCodexUsageExtraUpdatesForScope(scope, snapshot, now)
		if len(updates) == 0 {
			return
		}
		shouldPersistUpdates := s.getCodexSnapshotThrottle().Allow(accountID, now) || codexRateLimitResetAtFromSnapshot(snapshot, now) != nil
		if !shouldPersistUpdates {
			return
		}
		syncOpenAICodexRateLimitState(updateCtx, s.accountRepo, account, updates, now)
	}()
}
func (s *OpenAIGatewayService) UpdateCodexUsageSnapshotFromHeaders(ctx context.Context, accountID int64, headers http.Header, fallbackModels ...string) {
	if accountID <= 0 || headers == nil {
		return
	}
	if snapshot := ParseCodexRateLimitHeaders(headers); snapshot != nil {
		ctx = withOpenAICodexRequestModelFallback(ctx, fallbackModels...)
		ctx = withOpenAICodexSuccessfulSnapshot(ctx)
		s.updateCodexUsageSnapshot(ctx, accountID, snapshot, fallbackModels...)
	}
}
