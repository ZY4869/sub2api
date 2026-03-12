package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
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
	platform, useMixedScheduling, hasForcePlatform, err := s.resolvePlatformAndSchedulingMode(ctx, groupID)
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
func (s *GeminiMessagesCompatService) resolvePlatformAndSchedulingMode(ctx context.Context, groupID *int64) (platform string, useMixedScheduling bool, hasForcePlatform bool, err error) {
	forcePlatform, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string)
	if hasForcePlatform && forcePlatform != "" {
		return forcePlatform, false, true, nil
	}
	if groupID != nil {
		var group *Group
		if ctxGroup, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(ctxGroup) && ctxGroup.ID == *groupID {
			group = ctxGroup
		} else {
			group, err = s.groupRepo.GetByIDLite(ctx, *groupID)
			if err != nil {
				return "", false, false, fmt.Errorf("get group failed: %w", err)
			}
		}
		return group.Platform, group.Platform == PlatformGemini, false, nil
	}
	return PlatformGemini, true, false, nil
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
	if !s.isAccountValidForPlatform(account, platform, useMixedScheduling) {
		return false
	}
	if !s.passesRateLimitPreCheckWithCache(ctx, account, requestedModel, precheckResult) {
		return false
	}
	return true
}
func (s *GeminiMessagesCompatService) isAccountValidForPlatform(account *Account, platform string, useMixedScheduling bool) bool {
	if account.Platform == platform {
		return true
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
		if s.isBetterGeminiAccount(acc, selected) {
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
func (s *GeminiMessagesCompatService) isBetterGeminiAccount(candidate, current *Account) bool {
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
	return s.accountRepo.GetByID(ctx, accountID)
}
func (s *GeminiMessagesCompatService) listSchedulableAccountsOnce(ctx context.Context, groupID *int64, platform string, hasForcePlatform bool) ([]Account, error) {
	if s.schedulerSnapshot != nil {
		accounts, _, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, hasForcePlatform)
		return accounts, err
	}
	useMixedScheduling := platform == PlatformGemini && !hasForcePlatform
	queryPlatforms := []string{platform}
	if useMixedScheduling {
		queryPlatforms = []string{platform, PlatformAntigravity}
	}
	if groupID != nil {
		return s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, queryPlatforms)
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		return s.accountRepo.ListSchedulableByPlatforms(ctx, queryPlatforms)
	}
	return s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, queryPlatforms)
}
func (s *GeminiMessagesCompatService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg != nil && !s.cfg.Security.URLAllowlist.Enabled {
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
