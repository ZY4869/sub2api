package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"strings"
)

func (s *GeminiMessagesCompatService) GetTokenProvider() *GeminiTokenProvider {
	return s.tokenProvider
}
func (s *GeminiMessagesCompatService) SelectAccountForModel(ctx context.Context, groupID *int64, sessionHash string, requestedModel string) (*Account, error) {
	return s.SelectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, nil)
}
func (s *GeminiMessagesCompatService) SelectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*Account, error) {
	ctx, platform, useMixedScheduling, hasForcePlatform, err := s.resolvePlatformAndSchedulingMode(ctx, groupID)
	if err != nil {
		return nil, err
	}
	cacheKey := "gemini:" + sessionHash
	if account := s.tryStickySessionHit(ctx, groupID, sessionHash, cacheKey, requestedModel, excludedIDs, platform, useMixedScheduling); account != nil {
		return account, nil
	}
	accounts, err := s.listSchedulableAccountsOnce(ctx, groupID, platform, hasForcePlatform)
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}
	if len(accounts) == 0 && groupID != nil && hasForcePlatform {
		accounts, err = s.listSchedulableAccountsOnce(ctx, nil, platform, hasForcePlatform)
		if err != nil {
			return nil, fmt.Errorf("query accounts failed: %w", err)
		}
	}
	selected := s.selectBestGeminiAccount(ctx, accounts, requestedModel, excludedIDs, platform, useMixedScheduling)
	if selected == nil {
		if requestedModel != "" {
			return nil, fmt.Errorf("no available Gemini accounts supporting model: %s", requestedModel)
		}
		return nil, errors.New("no available Gemini accounts")
	}
	if sessionHash != "" {
		_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), cacheKey, selected.ID, geminiStickySessionTTL)
	}
	return selected, nil
}
func (s *GeminiMessagesCompatService) resolvePlatformAndSchedulingMode(ctx context.Context, groupID *int64) (context.Context, string, bool, bool, error) {
	forcePlatform, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string)
	if hasForcePlatform && forcePlatform != "" {
		return ctx, forcePlatform, false, true, nil
	}
	if groupID != nil {
		if s.groupRepo == nil {
			return ctx, "", false, false, infraerrors.ServiceUnavailable("GROUP_REPO_UNAVAILABLE", "group repository is unavailable")
		}
		var group *Group
		var err error
		if ctxGroup, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(ctxGroup) && ctxGroup.ID == *groupID {
			group = ctxGroup
		} else {
			group, err = s.groupRepo.GetByIDLite(ctx, *groupID)
			if err != nil {
				return ctx, "", false, false, fmt.Errorf("get group failed: %w", err)
			}
		}
		ctx = withGeminiGroupContext(ctx, group)
		return ctx, group.Platform, group.Platform == PlatformGemini, false, nil
	}
	return ctx, PlatformGemini, true, false, nil
}
func (s *GeminiMessagesCompatService) tryStickySessionHit(ctx context.Context, groupID *int64, sessionHash, cacheKey, requestedModel string, excludedIDs map[int64]struct{}, platform string, useMixedScheduling bool) *Account {
	if sessionHash == "" {
		return nil
	}
	accountID, err := s.cache.GetSessionAccountID(ctx, derefGroupID(groupID), cacheKey)
	if err != nil || accountID <= 0 {
		return nil
	}
	if _, excluded := excludedIDs[accountID]; excluded {
		return nil
	}
	account, err := s.getSchedulableAccount(ctx, accountID)
	if err != nil {
		return nil
	}
	if shouldClearStickySession(account, requestedModel) {
		_ = s.cache.DeleteSessionAccountID(ctx, derefGroupID(groupID), cacheKey)
		return nil
	}
	if !s.isAccountUsableForRequest(ctx, account, requestedModel, platform, useMixedScheduling) {
		return nil
	}
	_ = s.cache.RefreshSessionTTL(ctx, derefGroupID(groupID), cacheKey, geminiStickySessionTTL)
	return account
}
func (s *GeminiMessagesCompatService) isAccountUsableForRequest(ctx context.Context, account *Account, requestedModel, platform string, useMixedScheduling bool) bool {
	return s.isAccountUsableForRequestWithPrecheck(ctx, account, requestedModel, platform, useMixedScheduling, nil)
}
func (s *GeminiMessagesCompatService) isAccountUsableForRequestWithPrecheck(ctx context.Context, account *Account, requestedModel, platform string, useMixedScheduling bool, precheckResult map[int64]bool) bool {
	if !account.IsSchedulableForModelWithContext(ctx, requestedModel) {
		return false
	}
	if requestedModel != "" && !s.isModelSupportedByAccount(account, requestedModel) {
		return false
	}
	if !s.isAccountValidForPlatform(ctx, account, platform, useMixedScheduling) {
		return false
	}
	if !s.passesRateLimitPreCheckWithCache(ctx, account, requestedModel, precheckResult) {
		return false
	}
	return true
}
func (s *GeminiMessagesCompatService) isAccountValidForPlatform(ctx context.Context, account *Account, platform string, useMixedScheduling bool) bool {
	if MatchesGroupPlatform(account, platform) {
		return geminiPublicProtocolAllowsAccount(ctx, account)
	}
	if useMixedScheduling && account.Platform == PlatformAntigravity && account.IsMixedSchedulingEnabled() {
		return true
	}
	return false
}
func (s *GeminiMessagesCompatService) passesRateLimitPreCheckWithCache(ctx context.Context, account *Account, requestedModel string, precheckResult map[int64]bool) bool {
	if s.rateLimitService == nil || requestedModel == "" {
		return true
	}
	if precheckResult != nil {
		if ok, exists := precheckResult[account.ID]; exists {
			return ok
		}
	}
	ok, err := s.rateLimitService.PreCheckUsage(ctx, account, requestedModel)
	if err != nil {
		logger.LegacyPrintf("service.gemini_messages_compat", "[Gemini PreCheck] Account %d precheck error: %v", account.ID, err)
	}
	return ok
}
func (s *GeminiMessagesCompatService) selectBestGeminiAccount(ctx context.Context, accounts []Account, requestedModel string, excludedIDs map[int64]struct{}, platform string, useMixedScheduling bool) *Account {
	var selected *Account
	precheckResult := s.buildPreCheckUsageResultMap(ctx, accounts, requestedModel)
	for i := range accounts {
		acc := &accounts[i]
		if _, excluded := excludedIDs[acc.ID]; excluded {
			continue
		}
		if !s.isAccountUsableForRequestWithPrecheck(ctx, acc, requestedModel, platform, useMixedScheduling, precheckResult) {
			continue
		}
		if selected == nil {
			selected = acc
			continue
		}
		if s.isBetterGeminiAccount(ctx, acc, selected) {
			selected = acc
		}
	}
	return selected
}
func (s *GeminiMessagesCompatService) buildPreCheckUsageResultMap(ctx context.Context, accounts []Account, requestedModel string) map[int64]bool {
	if s.rateLimitService == nil || requestedModel == "" || len(accounts) == 0 {
		return nil
	}
	candidates := make([]*Account, 0, len(accounts))
	for i := range accounts {
		candidates = append(candidates, &accounts[i])
	}
	result, err := s.rateLimitService.PreCheckUsageBatch(ctx, candidates, requestedModel)
	if err != nil {
		logger.LegacyPrintf("service.gemini_messages_compat", "[Gemini PreCheckBatch] failed: %v", err)
	}
	return result
}
func (s *GeminiMessagesCompatService) isBetterGeminiAccount(ctx context.Context, candidate, current *Account) bool {
	if candidateRank, currentRank := geminiPublicProtocolRank(ctx, candidate), geminiPublicProtocolRank(ctx, current); candidateRank != currentRank {
		return candidateRank < currentRank
	}
	if candidate.Priority < current.Priority {
		return true
	}
	if candidate.Priority > current.Priority {
		return false
	}
	switch {
	case candidate.LastUsedAt == nil && current.LastUsedAt != nil:
		return true
	case candidate.LastUsedAt != nil && current.LastUsedAt == nil:
		return false
	case candidate.LastUsedAt == nil && current.LastUsedAt == nil:
		return candidate.Type == AccountTypeOAuth && current.Type != AccountTypeOAuth
	default:
		return candidate.LastUsedAt.Before(*current.LastUsedAt)
	}
}
func (s *GeminiMessagesCompatService) isModelSupportedByAccount(account *Account, requestedModel string) bool {
	if account.Platform == PlatformAntigravity {
		if strings.TrimSpace(requestedModel) == "" {
			return true
		}
		return mapAntigravityModel(account, requestedModel) != ""
	}
	return account.IsModelSupported(requestedModel)
}
func (s *GeminiMessagesCompatService) GetAntigravityGatewayService() *AntigravityGatewayService {
	return s.antigravityGatewayService
}
func (s *GeminiMessagesCompatService) getSchedulableAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s.schedulerSnapshot != nil {
		return s.schedulerSnapshot.GetAccount(ctx, accountID)
	}
	if s.accountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("ACCOUNT_REPO_UNAVAILABLE", "account repository is unavailable")
	}
	return s.accountRepo.GetByID(ctx, accountID)
}
func (s *GeminiMessagesCompatService) listSchedulableAccountsOnce(ctx context.Context, groupID *int64, platform string, hasForcePlatform bool) ([]Account, error) {
	selectionCtx := ctx
	if groupID != nil && platform == PlatformGemini && GeminiPublicProtocolFromContext(ctx) != "" {
		if ctxGroup, ok := ctx.Value(ctxkey.Group).(*Group); !ok || !IsGroupContextValid(ctxGroup) || ctxGroup.ID != *groupID {
			if s.groupRepo == nil {
				return nil, infraerrors.ServiceUnavailable("GROUP_REPO_UNAVAILABLE", "group repository is unavailable")
			}
			if group, err := s.groupRepo.GetByIDLite(ctx, *groupID); err == nil {
				selectionCtx = withGeminiGroupContext(ctx, group)
			}
		}
	}
	if s.schedulerSnapshot != nil {
		accounts, _, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, hasForcePlatform)
		return filterGeminiAccountsByPublicProtocol(selectionCtx, accounts, platform), err
	}
	if s.accountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("ACCOUNT_REPO_UNAVAILABLE", "account repository is unavailable")
	}
	useMixedScheduling := platform == PlatformGemini && !hasForcePlatform
	queryPlatforms := QueryPlatformsForGroupPlatform(platform, useMixedScheduling)
	if groupID != nil {
		accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, queryPlatforms)
		return filterGeminiAccountsByPublicProtocol(selectionCtx, accounts, platform), err
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		accounts, err := s.accountRepo.ListSchedulableByPlatforms(ctx, queryPlatforms)
		return filterGeminiAccountsByPublicProtocol(selectionCtx, accounts, platform), err
	}
	accounts, err := s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, queryPlatforms)
	return filterGeminiAccountsByPublicProtocol(selectionCtx, accounts, platform), err
}
func (s *GeminiMessagesCompatService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg == nil {
		normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{})
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	if !s.cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{AllowedHosts: s.cfg.Security.URLAllowlist.UpstreamHosts, RequireAllowlist: true, AllowPrivate: s.cfg.Security.URLAllowlist.AllowPrivateHosts})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}
func (s *GeminiMessagesCompatService) HasAntigravityAccounts(ctx context.Context, groupID *int64) (bool, error) {
	accounts, err := s.listSchedulableAccountsOnce(ctx, groupID, PlatformAntigravity, false)
	if err != nil {
		return false, err
	}
	return len(accounts) > 0, nil
}
