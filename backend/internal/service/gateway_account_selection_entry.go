package service

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"log/slog"
	"sort"
	"time"
)

func (s *GatewayService) SelectAccount(ctx context.Context, groupID *int64, sessionHash string) (*Account, error) {
	return s.SelectAccountForModel(ctx, groupID, sessionHash, "")
}
func (s *GatewayService) SelectAccountForModel(ctx context.Context, groupID *int64, sessionHash string, requestedModel string) (*Account, error) {
	return s.SelectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, nil)
}
func (s *GatewayService) SelectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*Account, error) {
	var platform string
	forcePlatform, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string)
	if hasForcePlatform && forcePlatform != "" {
		platform = forcePlatform
	} else if groupID != nil {
		group, resolvedGroupID, err := s.resolveGatewayGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}
		groupID = resolvedGroupID
		ctx = s.withGroupContext(ctx, group)
		platform = group.Platform
	} else {
		platform = PlatformAnthropic
	}
	if (platform == PlatformAnthropic || platform == PlatformGemini) && !hasForcePlatform {
		return s.selectAccountWithMixedScheduling(ctx, groupID, sessionHash, requestedModel, excludedIDs, platform)
	}
	return s.selectAccountForModelWithPlatform(ctx, groupID, sessionHash, requestedModel, excludedIDs, platform)
}
func (s *GatewayService) SelectAccountWithLoadAwareness(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, metadataUserID string) (*AccountSelectionResult, error) {
	excludedIDsList := make([]int64, 0, len(excludedIDs))
	for id := range excludedIDs {
		excludedIDsList = append(excludedIDsList, id)
	}
	slog.Debug("account_scheduling_starting", "group_id", derefGroupID(groupID), "model", requestedModel, "session", shortSessionHash(sessionHash), "excluded_ids", excludedIDsList)
	cfg := s.schedulingConfig()
	group, groupID, err := s.checkClaudeCodeRestriction(ctx, groupID)
	if err != nil {
		return nil, err
	}
	ctx = s.withGroupContext(ctx, group)
	var stickyAccountID int64
	if prefetch := prefetchedStickyAccountIDFromContext(ctx, groupID); prefetch > 0 {
		stickyAccountID = prefetch
	} else if sessionHash != "" && s.cache != nil {
		if accountID, err := s.cache.GetSessionAccountID(ctx, derefGroupID(groupID), sessionHash); err == nil {
			stickyAccountID = accountID
		}
	}
	if s.debugModelRoutingEnabled() && requestedModel != "" {
		groupPlatform := ""
		if group != nil {
			groupPlatform = group.Platform
		}
		logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] select entry: group_id=%v group_platform=%s model=%s session=%s sticky_account=%d load_batch=%v concurrency=%v", derefGroupID(groupID), groupPlatform, requestedModel, shortSessionHash(sessionHash), stickyAccountID, cfg.LoadBatchEnabled, s.concurrencyService != nil)
	}
	if s.concurrencyService == nil || !cfg.LoadBatchEnabled {
		localExcluded := make(map[int64]struct{})
		for k, v := range excludedIDs {
			localExcluded[k] = v
		}
		for {
			account, err := s.SelectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, localExcluded)
			if err != nil {
				return nil, err
			}
			result, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
			if err == nil && result.Acquired {
				if !s.checkAndRegisterSession(ctx, account, sessionHash) {
					result.ReleaseFunc()
					localExcluded[account.ID] = struct{}{}
					continue
				}
				s.logSelectedAccountUsagePressure("local_acquired", groupID, sessionHash, requestedModel, account)
				return &AccountSelectionResult{Account: account, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
			}
			if !s.checkAndRegisterSession(ctx, account, sessionHash) {
				localExcluded[account.ID] = struct{}{}
				continue
			}
			if stickyAccountID > 0 && stickyAccountID == account.ID && s.concurrencyService != nil {
				waitingCount, _ := s.concurrencyService.GetAccountWaitingCount(ctx, account.ID)
				if waitingCount < cfg.StickySessionMaxWaiting {
					s.logSelectedAccountUsagePressure("local_sticky_wait", groupID, sessionHash, requestedModel, account)
					return &AccountSelectionResult{Account: account, WaitPlan: &AccountWaitPlan{AccountID: account.ID, MaxConcurrency: account.Concurrency, Timeout: cfg.StickySessionWaitTimeout, MaxWaiting: cfg.StickySessionMaxWaiting}}, nil
				}
			}
			s.logSelectedAccountUsagePressure("local_wait", groupID, sessionHash, requestedModel, account)
			return &AccountSelectionResult{Account: account, WaitPlan: &AccountWaitPlan{AccountID: account.ID, MaxConcurrency: account.Concurrency, Timeout: cfg.FallbackWaitTimeout, MaxWaiting: cfg.FallbackMaxWaiting}}, nil
		}
	}
	platform, hasForcePlatform, err := s.resolvePlatform(ctx, groupID, group)
	if err != nil {
		return nil, err
	}
	preferOAuth := platform == PlatformGemini
	if s.debugModelRoutingEnabled() && platform == PlatformAnthropic && requestedModel != "" {
		logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] load-aware enabled: group_id=%v model=%s session=%s platform=%s", derefGroupID(groupID), requestedModel, shortSessionHash(sessionHash), platform)
	}
	accounts, useMixed, err := s.listSchedulableAccounts(ctx, groupID, platform, hasForcePlatform)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, errors.New("no available accounts")
	}
	observeVertexSelection := func(selected *Account, phase string) {
		s.observeGeminiVertexRouting(ctx, accounts, groupID, requestedModel, platform, useMixed, excludedIDs, selected, phase)
	}
	ctx = s.withWindowCostPrefetch(ctx, accounts)
	ctx = s.withRPMPrefetch(ctx, accounts)
	isExcluded := func(accountID int64) bool {
		if excludedIDs == nil {
			return false
		}
		_, excluded := excludedIDs[accountID]
		return excluded
	}
	accountByID := make(map[int64]*Account, len(accounts))
	for i := range accounts {
		accountByID[accounts[i].ID] = &accounts[i]
	}
	var routingAccountIDs []int64
	if group != nil && requestedModel != "" && group.Platform == PlatformAnthropic {
		routingAccountIDs = group.GetRoutingAccountIDs(requestedModel)
		if s.debugModelRoutingEnabled() {
			logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] context group routing: group_id=%d model=%s enabled=%v rules=%d matched_ids=%v session=%s sticky_account=%d", group.ID, requestedModel, group.ModelRoutingEnabled, len(group.ModelRouting), routingAccountIDs, shortSessionHash(sessionHash), stickyAccountID)
			if len(routingAccountIDs) == 0 && group.ModelRoutingEnabled && len(group.ModelRouting) > 0 {
				keys := make([]string, 0, len(group.ModelRouting))
				for k := range group.ModelRouting {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				const maxKeys = 20
				if len(keys) > maxKeys {
					keys = keys[:maxKeys]
				}
				logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] context group routing miss: group_id=%d model=%s patterns(sample)=%v", group.ID, requestedModel, keys)
			}
		}
	}
	if len(routingAccountIDs) > 0 && s.concurrencyService != nil {
		var routingCandidates []*Account
		var filteredExcluded, filteredMissing, filteredUnsched, filteredPlatform, filteredModelScope, filteredModelMapping, filteredWindowCost int
		var modelScopeSkippedIDs []int64
		for _, routingAccountID := range routingAccountIDs {
			if isExcluded(routingAccountID) {
				filteredExcluded++
				continue
			}
			account, ok := accountByID[routingAccountID]
			if !ok || !s.isAccountSchedulableForSelection(account) {
				if !ok {
					filteredMissing++
				} else {
					filteredUnsched++
				}
				continue
			}
			if !s.isAccountAllowedForPlatformWithContext(ctx, account, platform, useMixed) {
				filteredPlatform++
				continue
			}
			if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
				filteredModelMapping++
				continue
			}
			if !s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) {
				filteredModelScope++
				modelScopeSkippedIDs = append(modelScopeSkippedIDs, account.ID)
				continue
			}
			if !s.isAccountSchedulableForQuota(account) {
				continue
			}
			if !s.isAccountSchedulableForWindowCost(ctx, account, false) {
				filteredWindowCost++
				continue
			}
			if !s.isAccountSchedulableForRPM(ctx, account, false) {
				continue
			}
			routingCandidates = append(routingCandidates, account)
		}
		if s.debugModelRoutingEnabled() {
			logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] routed candidates: group_id=%v model=%s routed=%d candidates=%d filtered(excluded=%d missing=%d unsched=%d platform=%d model_scope=%d model_mapping=%d window_cost=%d)", derefGroupID(groupID), requestedModel, len(routingAccountIDs), len(routingCandidates), filteredExcluded, filteredMissing, filteredUnsched, filteredPlatform, filteredModelScope, filteredModelMapping, filteredWindowCost)
			if len(modelScopeSkippedIDs) > 0 {
				logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] model_rate_limited accounts skipped: group_id=%v model=%s account_ids=%v", derefGroupID(groupID), requestedModel, modelScopeSkippedIDs)
			}
		}
		if len(routingCandidates) > 0 {
			if sessionHash != "" && stickyAccountID > 0 {
				if containsInt64(routingAccountIDs, stickyAccountID) && !isExcluded(stickyAccountID) {
					if stickyAccount, ok := accountByID[stickyAccountID]; ok {
						if s.isAccountSchedulableForSelection(stickyAccount) && s.isAccountAllowedForPlatformWithContext(ctx, stickyAccount, platform, useMixed) && (requestedModel == "" || s.isModelSupportedByAccountWithContext(ctx, stickyAccount, requestedModel)) && s.isAccountSchedulableForModelSelection(ctx, stickyAccount, requestedModel) && s.isAccountSchedulableForQuota(stickyAccount) && s.isAccountSchedulableForWindowCost(ctx, stickyAccount, true) && s.isAccountSchedulableForRPM(ctx, stickyAccount, true) {
							result, err := s.tryAcquireAccountSlot(ctx, stickyAccountID, stickyAccount.Concurrency)
							if err == nil && result.Acquired {
								if !s.checkAndRegisterSession(ctx, stickyAccount, sessionHash) {
									result.ReleaseFunc()
								} else {
									if s.debugModelRoutingEnabled() {
										logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] routed sticky hit: group_id=%v model=%s session=%s account=%d", derefGroupID(groupID), requestedModel, shortSessionHash(sessionHash), stickyAccountID)
									}
									s.logSelectedAccountUsagePressure("routed_sticky_acquired", groupID, sessionHash, requestedModel, stickyAccount)
									observeVertexSelection(stickyAccount, "routed_sticky_acquired")
									return &AccountSelectionResult{Account: stickyAccount, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
								}
							}
							waitingCount, _ := s.concurrencyService.GetAccountWaitingCount(ctx, stickyAccountID)
							if waitingCount < cfg.StickySessionMaxWaiting {
								if !s.checkAndRegisterSession(ctx, stickyAccount, sessionHash) {
								} else {
									s.logSelectedAccountUsagePressure("routed_sticky_wait", groupID, sessionHash, requestedModel, stickyAccount)
									observeVertexSelection(stickyAccount, "routed_sticky_wait")
									return &AccountSelectionResult{Account: stickyAccount, WaitPlan: &AccountWaitPlan{AccountID: stickyAccountID, MaxConcurrency: stickyAccount.Concurrency, Timeout: cfg.StickySessionWaitTimeout, MaxWaiting: cfg.StickySessionMaxWaiting}}, nil
								}
							}
						}
					} else {
						_ = s.cache.DeleteSessionAccountID(ctx, derefGroupID(groupID), sessionHash)
					}
				}
			}
			routingLoads := make([]AccountWithConcurrency, 0, len(routingCandidates))
			for _, acc := range routingCandidates {
				routingLoads = append(routingLoads, AccountWithConcurrency{ID: acc.ID, MaxConcurrency: acc.EffectiveLoadFactor()})
			}
			routingLoadMap, _ := s.concurrencyService.GetAccountsLoadBatch(ctx, routingLoads)
			var routingAvailable []accountWithLoad
			for _, acc := range routingCandidates {
				loadInfo := routingLoadMap[acc.ID]
				if loadInfo == nil {
					loadInfo = &AccountLoadInfo{AccountID: acc.ID}
				}
				if loadInfo.LoadRate < 100 {
					routingAvailable = append(routingAvailable, accountWithLoad{account: acc, loadInfo: loadInfo})
				}
			}
			if len(routingAvailable) > 0 {
				routingAvailable = filterByMinGeminiPublicProtocolRank(ctx, routingAvailable)
				now := time.Now()
				sort.SliceStable(routingAvailable, func(i, j int) bool {
					a, b := routingAvailable[i], routingAvailable[j]
					if aRank, bRank := geminiPublicProtocolRank(ctx, a.account), geminiPublicProtocolRank(ctx, b.account); aRank != bRank {
						return aRank < bRank
					}
					return compareAccountsWithLoad(a, b, preferOAuth, now) < 0
				})
				shuffleWithinSortGroups(routingAvailable)
				for _, item := range routingAvailable {
					result, err := s.tryAcquireAccountSlot(ctx, item.account.ID, item.account.Concurrency)
					if err == nil && result.Acquired {
						if !s.checkAndRegisterSession(ctx, item.account, sessionHash) {
							result.ReleaseFunc()
							continue
						}
						if sessionHash != "" && s.cache != nil {
							_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionHash, item.account.ID, stickySessionTTL)
						}
						if s.debugModelRoutingEnabled() {
							logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] routed select: group_id=%v model=%s session=%s account=%d", derefGroupID(groupID), requestedModel, shortSessionHash(sessionHash), item.account.ID)
						}
						s.logSelectedAccountUsagePressure("routed_acquired", groupID, sessionHash, requestedModel, item.account)
						observeVertexSelection(item.account, "routed_acquired")
						return &AccountSelectionResult{Account: item.account, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
					}
				}
				for _, item := range routingAvailable {
					if !s.checkAndRegisterSession(ctx, item.account, sessionHash) {
						continue
					}
					if s.debugModelRoutingEnabled() {
						logger.LegacyPrintf("service.gateway", "[ModelRoutingDebug] routed wait: group_id=%v model=%s session=%s account=%d", derefGroupID(groupID), requestedModel, shortSessionHash(sessionHash), item.account.ID)
					}
					s.logSelectedAccountUsagePressure("routed_wait", groupID, sessionHash, requestedModel, item.account)
					observeVertexSelection(item.account, "routed_wait")
					return &AccountSelectionResult{Account: item.account, WaitPlan: &AccountWaitPlan{AccountID: item.account.ID, MaxConcurrency: item.account.Concurrency, Timeout: cfg.StickySessionWaitTimeout, MaxWaiting: cfg.StickySessionMaxWaiting}}, nil
				}
			}
			logger.LegacyPrintf("service.gateway", "[ModelRouting] All routed accounts unavailable for model=%s, falling back to normal selection", requestedModel)
		}
	}
	if len(routingAccountIDs) == 0 && sessionHash != "" && stickyAccountID > 0 && !isExcluded(stickyAccountID) {
		accountID := stickyAccountID
		if accountID > 0 && !isExcluded(accountID) {
			account, ok := accountByID[accountID]
			if ok {
				clearSticky := shouldClearStickySession(account, requestedModel)
				if clearSticky {
					_ = s.cache.DeleteSessionAccountID(ctx, derefGroupID(groupID), sessionHash)
				}
				if !clearSticky && s.isAccountInGroup(account, groupID) && s.isAccountAllowedForPlatformWithContext(ctx, account, platform, useMixed) && (requestedModel == "" || s.isModelSupportedByAccountWithContext(ctx, account, requestedModel)) && s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) && s.isAccountSchedulableForQuota(account) && s.isAccountSchedulableForWindowCost(ctx, account, true) && s.isAccountSchedulableForRPM(ctx, account, true) {
					result, err := s.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
					if err == nil && result.Acquired {
						if !s.checkAndRegisterSession(ctx, account, sessionHash) {
							result.ReleaseFunc()
						} else {
							s.logSelectedAccountUsagePressure("sticky_acquired", groupID, sessionHash, requestedModel, account)
							observeVertexSelection(account, "sticky_acquired")
							return &AccountSelectionResult{Account: account, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
						}
					}
					waitingCount, _ := s.concurrencyService.GetAccountWaitingCount(ctx, accountID)
					if waitingCount < cfg.StickySessionMaxWaiting {
						if !s.checkAndRegisterSession(ctx, account, sessionHash) {
						} else {
							s.logSelectedAccountUsagePressure("sticky_wait", groupID, sessionHash, requestedModel, account)
							observeVertexSelection(account, "sticky_wait")
							return &AccountSelectionResult{Account: account, WaitPlan: &AccountWaitPlan{AccountID: accountID, MaxConcurrency: account.Concurrency, Timeout: cfg.StickySessionWaitTimeout, MaxWaiting: cfg.StickySessionMaxWaiting}}, nil
						}
					}
				}
			}
		}
	}
	candidates := make([]*Account, 0, len(accounts))
	for i := range accounts {
		acc := &accounts[i]
		if isExcluded(acc.ID) {
			continue
		}
		if !s.isAccountSchedulableForSelection(acc) {
			continue
		}
		if !s.isAccountAllowedForPlatformWithContext(ctx, acc, platform, useMixed) {
			continue
		}
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, acc, requestedModel) {
			continue
		}
		if !s.isAccountSchedulableForModelSelection(ctx, acc, requestedModel) {
			continue
		}
		if !s.isAccountSchedulableForQuota(acc) {
			continue
		}
		if !s.isAccountSchedulableForWindowCost(ctx, acc, false) {
			continue
		}
		if !s.isAccountSchedulableForRPM(ctx, acc, false) {
			continue
		}
		candidates = append(candidates, acc)
	}
	if len(candidates) == 0 {
		observeVertexSelection(nil, "load_exhausted")
		return nil, errors.New("no available accounts")
	}
	accountLoads := make([]AccountWithConcurrency, 0, len(candidates))
	for _, acc := range candidates {
		accountLoads = append(accountLoads, AccountWithConcurrency{ID: acc.ID, MaxConcurrency: acc.EffectiveLoadFactor()})
	}
	loadMap, err := s.concurrencyService.GetAccountsLoadBatch(ctx, accountLoads)
	if err != nil {
		if result, ok := s.tryAcquireByLegacyOrder(ctx, candidates, groupID, sessionHash, preferOAuth); ok {
			phase := "legacy_wait"
			if result.Acquired {
				phase = "legacy_acquired"
			}
			s.logSelectedAccountUsagePressure(phase, groupID, sessionHash, requestedModel, result.Account)
			observeVertexSelection(result.Account, phase)
			return result, nil
		}
	} else {
		var available []accountWithLoad
		for _, acc := range candidates {
			loadInfo := loadMap[acc.ID]
			if loadInfo == nil {
				loadInfo = &AccountLoadInfo{AccountID: acc.ID}
			}
			if loadInfo.LoadRate < 100 {
				available = append(available, accountWithLoad{account: acc, loadInfo: loadInfo})
			}
		}
		for len(available) > 0 {
			now := time.Now()
			candidates := filterByMinGeminiPublicProtocolRank(ctx, available)
			candidates = filterByMinPriority(candidates)
			candidates = filterByMinGeminiRegionalPenalty(candidates, preferOAuth)
			candidates = filterByBestAccountUsagePressure(candidates, now)
			candidates = filterByMinLoadRate(candidates)
			selected := selectByLRU(candidates, preferOAuth)
			if selected == nil {
				break
			}
			result, err := s.tryAcquireAccountSlot(ctx, selected.account.ID, selected.account.Concurrency)
			if err == nil && result.Acquired {
				if !s.checkAndRegisterSession(ctx, selected.account, sessionHash) {
					result.ReleaseFunc()
				} else {
					if sessionHash != "" && s.cache != nil {
						_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionHash, selected.account.ID, stickySessionTTL)
					}
					s.logSelectedAccountUsagePressure("load_acquired", groupID, sessionHash, requestedModel, selected.account)
					observeVertexSelection(selected.account, "load_acquired")
					return &AccountSelectionResult{Account: selected.account, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
				}
			}
			selectedID := selected.account.ID
			newAvailable := make([]accountWithLoad, 0, len(available)-1)
			for _, acc := range available {
				if acc.account.ID != selectedID {
					newAvailable = append(newAvailable, acc)
				}
			}
			available = newAvailable
		}
	}
	s.sortCandidatesForFallback(candidates, preferOAuth, cfg.FallbackSelectionMode)
	stableSortAccountsByGeminiPublicProtocolRank(ctx, candidates)
	for _, acc := range candidates {
		if !s.checkAndRegisterSession(ctx, acc, sessionHash) {
			continue
		}
		s.logSelectedAccountUsagePressure("load_wait", groupID, sessionHash, requestedModel, acc)
		observeVertexSelection(acc, "load_wait")
		return &AccountSelectionResult{Account: acc, WaitPlan: &AccountWaitPlan{AccountID: acc.ID, MaxConcurrency: acc.Concurrency, Timeout: cfg.FallbackWaitTimeout, MaxWaiting: cfg.FallbackMaxWaiting}}, nil
	}
	observeVertexSelection(nil, "load_exhausted")
	return nil, errors.New("no available accounts")
}
func (s *GatewayService) tryAcquireByLegacyOrder(ctx context.Context, candidates []*Account, groupID *int64, sessionHash string, preferOAuth bool) (*AccountSelectionResult, bool) {
	ordered := append([]*Account(nil), candidates...)
	sortAccountsByPriorityAndLastUsed(ordered, preferOAuth)
	stableSortAccountsByGeminiPublicProtocolRank(ctx, ordered)
	for _, acc := range ordered {
		result, err := s.tryAcquireAccountSlot(ctx, acc.ID, acc.Concurrency)
		if err == nil && result.Acquired {
			if !s.checkAndRegisterSession(ctx, acc, sessionHash) {
				result.ReleaseFunc()
				continue
			}
			if sessionHash != "" && s.cache != nil {
				_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionHash, acc.ID, stickySessionTTL)
			}
			return &AccountSelectionResult{Account: acc, Acquired: true, ReleaseFunc: result.ReleaseFunc}, true
		}
	}
	return nil, false
}
func (s *GatewayService) schedulingConfig() config.GatewaySchedulingConfig {
	if s.cfg != nil {
		return s.cfg.Gateway.Scheduling
	}
	return config.GatewaySchedulingConfig{StickySessionMaxWaiting: 3, StickySessionWaitTimeout: 45 * time.Second, FallbackWaitTimeout: 30 * time.Second, FallbackMaxWaiting: 100, LoadBatchEnabled: true, SlotCleanupInterval: 30 * time.Second}
}

func (s *GatewayService) logSelectedAccountUsagePressure(
	phase string,
	groupID *int64,
	sessionHash string,
	requestedModel string,
	account *Account,
) {
	if account == nil {
		return
	}
	now := time.Now()
	window, utilization, resetAt := accountUsagePressureLogValues(buildAccountUsagePressure(account, now))
	slog.Debug(
		"gateway_account_selection_pressure",
		"phase", phase,
		"group_id", derefGroupID(groupID),
		"model", requestedModel,
		"session", shortSessionHash(sessionHash),
		"account_id", account.ID,
		"account_type", account.Type,
		"priority", account.Priority,
		"pressure_window", window,
		"pressure_utilization", utilization,
		"pressure_reset_at", resetAt,
	)
	if s.debugModelRoutingEnabled() && requestedModel != "" {
		logger.LegacyPrintf(
			"service.gateway",
			"[ModelRoutingDebug] account selection: phase=%s group_id=%v model=%s session=%s account=%d pressure_window=%s pressure_utilization=%.2f pressure_reset_at=%s",
			phase,
			derefGroupID(groupID),
			requestedModel,
			shortSessionHash(sessionHash),
			account.ID,
			window,
			utilization,
			resetAt,
		)
	}
}
