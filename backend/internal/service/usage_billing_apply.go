package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const postUsageBillingTimeout = 15 * time.Second

type apiKeyAuthCacheInvalidator interface {
	InvalidateAuthCacheByKey(ctx context.Context, key string)
}

type usageLogBestEffortWriter interface {
	CreateBestEffort(ctx context.Context, log *UsageLog) error
}

func detachedBillingContext(ctx context.Context) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	return context.WithTimeout(base, postUsageBillingTimeout)
}

func resolveUsageBillingRequestID(ctx context.Context, upstreamRequestID string) string {
	if ctx != nil {
		if clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
			return "client:" + strings.TrimSpace(clientRequestID)
		}
		if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
			return "local:" + strings.TrimSpace(requestID)
		}
	}
	if requestID := strings.TrimSpace(upstreamRequestID); requestID != "" {
		return requestID
	}
	return "generated:" + generateRequestID()
}

func resolveUsageBillingPayloadFingerprint(ctx context.Context, requestPayloadHash string) string {
	if payloadHash := strings.TrimSpace(requestPayloadHash); payloadHash != "" {
		return payloadHash
	}
	if ctx != nil {
		if clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
			return "client:" + strings.TrimSpace(clientRequestID)
		}
		if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
			return "local:" + strings.TrimSpace(requestID)
		}
	}
	return ""
}

func optionalTrimmedStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func writeUsageLogBestEffort(ctx context.Context, repo UsageLogRepository, usageLog *UsageLog, logKey string) {
	if repo == nil || usageLog == nil {
		return
	}
	usageCtx, cancel := detachedBillingContext(ctx)
	defer cancel()

	if writer, ok := repo.(usageLogBestEffortWriter); ok {
		if err := writer.CreateBestEffort(usageCtx, usageLog); err != nil {
			logger.LegacyPrintf(logKey, "Create usage log failed: %v", err)
			if IsUsageLogCreateDropped(err) {
				return
			}
			if _, syncErr := repo.Create(usageCtx, usageLog); syncErr != nil {
				logger.LegacyPrintf(logKey, "Create usage log sync fallback failed: %v", syncErr)
			}
		}
		return
	}

	if _, err := repo.Create(usageCtx, usageLog); err != nil {
		logger.LegacyPrintf(logKey, "Create usage log failed: %v", err)
	}
}

func buildUsageBillingCommand(requestID string, usageLog *UsageLog, p *postUsageBillingParams) *UsageBillingCommand {
	if p == nil || p.Cost == nil || p.APIKey == nil || p.User == nil || p.Account == nil {
		return nil
	}

	cmd := &UsageBillingCommand{
		RequestID:          strings.TrimSpace(requestID),
		APIKeyID:           p.APIKey.ID,
		UserID:             p.User.ID,
		AccountID:          p.Account.ID,
		AccountType:        p.Account.Type,
		RequestPayloadHash: strings.TrimSpace(p.RequestPayloadHash),
	}
	if usageLog != nil {
		cmd.GroupID = usageLog.GroupID
		cmd.Model = usageLog.Model
		cmd.BillingType = usageLog.BillingType
		cmd.InputTokens = usageLog.InputTokens
		cmd.OutputTokens = usageLog.OutputTokens
		cmd.CacheCreationTokens = usageLog.CacheCreationTokens
		cmd.CacheReadTokens = usageLog.CacheReadTokens
		cmd.ImageCount = usageLog.ImageCount
		if usageLog.MediaType != nil {
			cmd.MediaType = *usageLog.MediaType
		}
		if usageLog.ServiceTier != nil {
			cmd.ServiceTier = *usageLog.ServiceTier
		}
		if usageLog.ReasoningEffort != nil {
			cmd.ReasoningEffort = *usageLog.ReasoningEffort
		}
		if usageLog.SubscriptionID != nil {
			cmd.SubscriptionID = usageLog.SubscriptionID
		}
	}

	if !p.SkipUserBilling {
		if p.IsSubscriptionBill && p.Subscription != nil && p.Cost.TotalCost > 0 {
			cmd.SubscriptionID = &p.Subscription.ID
			cmd.SubscriptionCost = p.Cost.TotalCost
		} else if p.Cost.ActualCost > 0 {
			cmd.BalanceCost = p.Cost.ActualCost
		}
	}

	if !p.SkipUserBilling && p.Cost.ActualCost > 0 && p.APIKey.Quota > 0 && p.APIKeyService != nil {
		cmd.APIKeyQuotaCost = p.Cost.ActualCost
	}
	if !p.SkipUserBilling && p.Cost.ActualCost > 0 && p.APIKeyService != nil && p.APIKey != nil && p.APIKey.GroupID != nil {
		cmd.GroupID = p.APIKey.GroupID
		cmd.APIKeyGroupQuotaCost = p.Cost.ActualCost
	}
	if !p.SkipUserBilling && p.Cost.ActualCost > 0 && p.APIKey.HasRateLimits() && p.APIKeyService != nil {
		cmd.APIKeyRateLimitCost = p.Cost.ActualCost
	}
	if p.Cost.TotalCost > 0 && CanParticipateInAccountQuota(p.Account) && p.Account.HasAnyQuotaLimit() {
		cmd.AccountQuotaCost = p.Cost.TotalCost * p.AccountRateMultiplier
	}

	cmd.Normalize()
	return cmd
}

func applyUsageBilling(ctx context.Context, requestID string, usageLog *UsageLog, p *postUsageBillingParams, deps *billingDeps, repo UsageBillingRepository) (bool, error) {
	if p == nil || deps == nil {
		return false, nil
	}

	cmd := buildUsageBillingCommand(requestID, usageLog, p)
	if cmd == nil || cmd.RequestID == "" || repo == nil {
		postUsageBilling(ctx, p, deps)
		return true, nil
	}

	billingCtx, cancel := detachedBillingContext(ctx)
	defer cancel()

	result, err := repo.Apply(billingCtx, cmd)
	if err != nil {
		return false, err
	}

	if result == nil || !result.Applied {
		deps.deferredService.ScheduleLastUsedUpdate(p.Account.ID)
		return false, nil
	}

	if result.APIKeyQuotaExhausted || cmd.APIKeyGroupQuotaCost > 0 {
		if invalidator, ok := p.APIKeyService.(apiKeyAuthCacheInvalidator); ok && p.APIKey != nil && p.APIKey.Key != "" {
			invalidator.InvalidateAuthCacheByKey(billingCtx, p.APIKey.Key)
		}
	}

	finalizePostUsageBilling(p, deps)
	return true, nil
}

func finalizePostUsageBilling(p *postUsageBillingParams, deps *billingDeps) {
	if p == nil || p.Cost == nil || deps == nil || deps.deferredService == nil {
		return
	}

	if !p.SkipUserBilling {
		if p.IsSubscriptionBill {
			if p.Cost.TotalCost > 0 && p.User != nil && p.APIKey != nil && p.APIKey.GroupID != nil {
				deps.billingCacheService.QueueUpdateSubscriptionUsage(p.User.ID, *p.APIKey.GroupID, p.Cost.TotalCost)
			}
		} else if p.Cost.ActualCost > 0 && p.User != nil {
			deps.billingCacheService.QueueDeductBalance(p.User.ID, p.Cost.ActualCost)
		}

		if p.Cost.ActualCost > 0 && p.APIKey != nil && p.APIKey.HasRateLimits() {
			deps.billingCacheService.QueueUpdateAPIKeyRateLimitUsage(p.APIKey.ID, p.Cost.ActualCost)
		}
	}

	deps.deferredService.ScheduleLastUsedUpdate(p.Account.ID)
}
