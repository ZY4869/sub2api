package service

import (
	"context"
	"strings"
	"time"
)

type openAIRuntimeQuotaStatus struct {
	Scope          string
	ScopeRemaining time.Duration
	AccountResetAt *time.Time
}

func (s openAIRuntimeQuotaStatus) Limited() bool {
	return s.AccountResetAt != nil || s.ScopeRemaining > 0
}

func openAIRuntimeQuotaStatusForCandidates(account *Account, candidates ...string) openAIRuntimeQuotaStatus {
	status := openAIRuntimeQuotaStatus{}
	if account == nil || !account.IsOpenAI() {
		return status
	}

	now := time.Now()
	if resetAt, ok := openAIAccountAll7dRateLimited(account, now); ok && resetAt != nil {
		status.AccountResetAt = resetAt
		return status
	}

	scope, remaining := openAIQuotaScopeRateLimitRemaining(account, candidates...)
	status.Scope = scope
	status.ScopeRemaining = remaining
	return status
}

func openAIRuntimeQuotaStatusForRequestedModel(account *Account, requestedModel string, extras ...string) openAIRuntimeQuotaStatus {
	return openAIRuntimeQuotaStatusForCandidates(account, openAIRuntimeQuotaModelCandidates(account, requestedModel, extras...)...)
}

func shouldHideOpenAIModelForRuntimeQuota(account *Account, candidates ...string) bool {
	return openAIRuntimeQuotaStatusForCandidates(account, candidates...).Limited()
}

func availableTestModelRuntimeQuotaCandidates(account *Account, model AvailableTestModel) []string {
	return openAIRuntimeQuotaModelCandidates(
		account,
		firstNonEmptyTrimmed(model.TargetModelID, model.ID),
		model.CanonicalID,
		model.ID,
	)
}

func apiKeyPublicModelRuntimeQuotaCandidates(account *Account, entry APIKeyPublicModelEntry) []string {
	return openAIRuntimeQuotaModelCandidates(
		account,
		firstNonEmptyTrimmed(entry.SourceID, entry.PublicID),
		entry.SourceID,
		entry.PublicID,
		entry.AliasID,
	)
}

func openAIAdminQuotaCooldownMessage(status openAIRuntimeQuotaStatus) string {
	if status.AccountResetAt != nil {
		return "整号额度冷却中，请等待额度恢复后再测试"
	}
	switch strings.TrimSpace(status.Scope) {
	case openAICodexScopeSpark:
		return "Spark 冷却中，请等待额度恢复后再测试"
	case openAICodexScopeNormal:
		return "普通额度冷却中，请等待额度恢复后再测试"
	default:
		return "额度冷却中，请等待额度恢复后再测试"
	}
}

func (s *OpenAIGatewayService) IsModelUnavailableDueToRuntimeQuota(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	excludedIDs map[int64]struct{},
) bool {
	if s == nil {
		return false
	}
	accounts, err := s.listSchedulableAccounts(ctx, groupID)
	if err != nil || len(accounts) == 0 {
		return false
	}

	supportingAccounts := 0
	quotaBlockedAccounts := 0
	for i := range accounts {
		account := &accounts[i]
		if account == nil || !account.IsOpenAI() {
			continue
		}
		if excludedIDs != nil {
			if _, excluded := excludedIDs[account.ID]; excluded {
				continue
			}
		}
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
			continue
		}
		supportingAccounts++
		if openAIRuntimeQuotaStatusForRequestedModel(account, requestedModel).Limited() {
			quotaBlockedAccounts++
		}
	}
	return supportingAccounts > 0 && supportingAccounts == quotaBlockedAccounts
}
