package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"log/slog"
	"sort"
	"strings"
	"time"
)

func (s *OpenAIGatewayService) ExtractSessionID(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}
	sessionID := strings.TrimSpace(c.GetHeader("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.GetHeader("conversation_id"))
	}
	if sessionID == "" && len(body) > 0 {
		sessionID = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	return sessionID
}
func (s *OpenAIGatewayService) GenerateSessionHash(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}
	sessionID := strings.TrimSpace(c.GetHeader("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.GetHeader("conversation_id"))
	}
	if sessionID == "" && len(body) > 0 {
		sessionID = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	if sessionID == "" {
		return ""
	}
	currentHash, legacyHash := deriveOpenAISessionHashes(sessionID)
	attachOpenAILegacySessionHashToGin(c, legacyHash)
	return currentHash
}
func (s *OpenAIGatewayService) GenerateSessionHashWithFallback(c *gin.Context, body []byte, fallbackSeed string) string {
	sessionHash := s.GenerateSessionHash(c, body)
	if sessionHash != "" {
		return sessionHash
	}
	seed := strings.TrimSpace(fallbackSeed)
	if seed == "" {
		return ""
	}
	currentHash, legacyHash := deriveOpenAISessionHashes(seed)
	attachOpenAILegacySessionHashToGin(c, legacyHash)
	return currentHash
}

func (s *OpenAIGatewayService) ResolveSessionHashWithSource(c *gin.Context, body []byte, fallbackSeed string) (string, string, int) {
	if seed, source := resolveOpenAIResponsesSessionSeed(c, body); seed != "" {
		currentHash, legacyHash := deriveOpenAISessionHashes(seed)
		attachOpenAILegacySessionHashToGin(c, legacyHash)
		return currentHash, source, len(seed)
	}
	seed := strings.TrimSpace(fallbackSeed)
	if seed == "" {
		return "", "", 0
	}
	currentHash, legacyHash := deriveOpenAISessionHashes(seed)
	attachOpenAILegacySessionHashToGin(c, legacyHash)
	return currentHash, "fallback_seed", len(seed)
}

func resolveOpenAIResponsesSessionSeed(c *gin.Context, body []byte) (string, string) {
	if c == nil {
		return "", ""
	}
	if sessionID := strings.TrimSpace(c.GetHeader("session_id")); sessionID != "" {
		return sessionID, "session_id"
	}
	if conversationID := strings.TrimSpace(c.GetHeader("conversation_id")); conversationID != "" {
		return conversationID, "conversation_id"
	}
	if len(body) == 0 {
		return "", ""
	}
	if promptCacheKey := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); promptCacheKey != "" {
		return promptCacheKey, "prompt_cache_key"
	}
	if previousResponseID := strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String()); previousResponseID != "" {
		if ClassifyOpenAIPreviousResponseIDKind(previousResponseID) == OpenAIPreviousResponseIDKindResponseID {
			return previousResponseID, "previous_response_id"
		}
	}
	if inputSeed := canonicalOpenAIResponsesInputSessionSeed(body); inputSeed != "" {
		return inputSeed, "input"
	}
	return "", ""
}

func canonicalOpenAIResponsesInputSessionSeed(body []byte) string {
	items, exists, err := openAIWSExtractNormalizedInputSequence(body)
	if err != nil || !exists || len(items) == 0 {
		return ""
	}
	normalized := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		raw := strings.TrimSpace(string(item))
		if raw == "" || raw == "null" {
			continue
		}
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			return ""
		}
		encoded, err := json.Marshal(decoded)
		if err != nil {
			return ""
		}
		normalized = append(normalized, json.RawMessage(encoded))
	}
	if len(normalized) == 0 {
		return ""
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	return "responses:input:" + string(encoded)
}
func resolveOpenAIUpstreamOriginator(c *gin.Context, isOfficialClient bool) string {
	if c != nil {
		if originator := strings.TrimSpace(c.GetHeader("originator")); originator != "" {
			return originator
		}
	}
	if isOfficialClient {
		return "codex_cli_rs"
	}
	return "opencode"
}
func (s *OpenAIGatewayService) BindStickySession(ctx context.Context, groupID *int64, sessionHash string, accountID int64) error {
	if sessionHash == "" || accountID <= 0 {
		return nil
	}
	ttl := openaiStickySessionTTL
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.StickySessionTTLSeconds > 0 {
		ttl = time.Duration(s.cfg.Gateway.OpenAIWS.StickySessionTTLSeconds) * time.Second
	}
	return s.setStickySessionAccountID(ctx, groupID, sessionHash, accountID, ttl)
}
func (s *OpenAIGatewayService) SelectAccount(ctx context.Context, groupID *int64, sessionHash string) (*Account, error) {
	return s.SelectAccountForModel(ctx, groupID, sessionHash, "")
}
func (s *OpenAIGatewayService) SelectAccountForModel(ctx context.Context, groupID *int64, sessionHash string, requestedModel string) (*Account, error) {
	return s.SelectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, nil)
}
func (s *OpenAIGatewayService) SelectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*Account, error) {
	if pinned := s.publicCatalogPinnedAccount(ctx, groupID, requestedModel, excludedIDs); pinned != nil {
		return pinned, nil
	}
	return s.selectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, excludedIDs, 0)
}
func (s *OpenAIGatewayService) selectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, stickyAccountID int64) (*Account, error) {
	if account := s.tryStickySessionHit(ctx, groupID, sessionHash, requestedModel, excludedIDs, stickyAccountID); account != nil {
		return account, nil
	}
	accounts, err := s.listSchedulableAccounts(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}
	selected := s.selectBestAccount(ctx, accounts, requestedModel, excludedIDs)
	if selected == nil {
		if s.openAISelectionFailureIsModelUnsupported(ctx, accounts, requestedModel, excludedIDs) {
			return nil, fmt.Errorf("%w: %s", ErrOpenAIModelNotFound, requestedModel)
		}
		if requestedModel != "" {
			return nil, fmt.Errorf("no available OpenAI accounts supporting model: %s", requestedModel)
		}
		return nil, errors.New("no available OpenAI accounts")
	}
	s.logSelectedAccountUsagePressure("standard_selected", groupID, sessionHash, requestedModel, selected)
	if sessionHash != "" {
		_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, selected.ID, openaiStickySessionTTL)
	}
	return selected, nil
}
func (s *OpenAIGatewayService) tryStickySessionHit(ctx context.Context, groupID *int64, sessionHash, requestedModel string, excludedIDs map[int64]struct{}, stickyAccountID int64) *Account {
	if sessionHash == "" {
		return nil
	}
	accountID := stickyAccountID
	if accountID <= 0 {
		var err error
		accountID, err = s.getStickySessionAccountID(ctx, groupID, sessionHash)
		if err != nil || accountID <= 0 {
			return nil
		}
	}
	if _, excluded := excludedIDs[accountID]; excluded {
		return nil
	}
	account, err := s.getSchedulableAccount(ctx, accountID)
	if err != nil {
		return nil
	}
	if shouldClearStickySession(account, requestedModel) {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}
	if !account.IsSchedulable() || !isOpenAITextRuntimeAccount(account) {
		return nil
	}
	if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
		return nil
	}
	if !account.IsSchedulableForModelWithContext(ctx, requestedModel) {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}
	account = s.recheckSelectedOpenAIAccountFromDB(ctx, account, requestedModel)
	if account == nil {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}
	_ = s.refreshStickySessionTTL(ctx, groupID, sessionHash, openaiStickySessionTTL)
	return account
}
func (s *OpenAIGatewayService) selectBestAccount(ctx context.Context, accounts []Account, requestedModel string, excludedIDs map[int64]struct{}) *Account {
	localExcluded := copyAccountIDSet(excludedIDs)
	for {
		var selected *Account
		for i := range accounts {
			acc := &accounts[i]
			if _, excluded := localExcluded[acc.ID]; excluded {
				continue
			}
			fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel)
			if fresh == nil {
				continue
			}
			if selected == nil {
				selected = fresh
				continue
			}
			if s.isBetterAccount(fresh, selected, requestedModel) {
				selected = fresh
			}
		}
		if selected == nil {
			return nil
		}
		rechecked := s.recheckSelectedOpenAIAccountFromDB(ctx, selected, requestedModel)
		if rechecked != nil {
			return rechecked
		}
		if localExcluded == nil {
			localExcluded = make(map[int64]struct{})
		}
		localExcluded[selected.ID] = struct{}{}
	}
}

func copyAccountIDSet(src map[int64]struct{}) map[int64]struct{} {
	if len(src) == 0 {
		return nil
	}
	out := make(map[int64]struct{}, len(src))
	for id := range src {
		out[id] = struct{}{}
	}
	return out
}
func (s *OpenAIGatewayService) isBetterAccount(candidate, current *Account, requestedModel string) bool {
	return compareOpenAIAccountsForSelection(candidate, current, requestedModel, time.Now()) < 0
}
func (s *OpenAIGatewayService) SelectAccountWithLoadAwareness(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*AccountSelectionResult, error) {
	return s.SelectAccountWithLoadAwarenessForCapability(ctx, groupID, sessionHash, requestedModel, excludedIDs, "")
}

func (s *OpenAIGatewayService) SelectAccountWithLoadAwarenessForCapability(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, requiredCapability OpenAIEndpointCapability) (*AccountSelectionResult, error) {
	if pinned := s.publicCatalogPinnedAccount(ctx, groupID, requestedModel, excludedIDs); pinned != nil {
		if SupportsOpenAIEndpointCapability(pinned, requiredCapability) {
			return &AccountSelectionResult{Account: pinned}, nil
		}
	}
	cfg := s.schedulingConfig()
	var stickyAccountID int64
	if sessionHash != "" && s.cache != nil {
		if accountID, err := s.getStickySessionAccountID(ctx, groupID, sessionHash); err == nil {
			stickyAccountID = accountID
		}
	}
	if s.concurrencyService == nil || !cfg.LoadBatchEnabled {
		localExcluded := copyAccountIDSet(excludedIDs)
		if localExcluded == nil {
			localExcluded = make(map[int64]struct{})
		}
		var waitResult *AccountSelectionResult
		for {
			account, err := s.selectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, localExcluded, stickyAccountID)
			if err != nil {
				if waitResult != nil {
					return waitResult, nil
				}
				return nil, err
			}
			if !SupportsOpenAIEndpointCapability(account, requiredCapability) {
				localExcluded[account.ID] = struct{}{}
				continue
			}
			result, err := s.tryAcquireAccountSlot(ctx, account.ID, DeepSeekEffectiveAccountConcurrency(account, requestedModel))
			if err == nil && result.Acquired {
				s.logSelectedAccountUsagePressure("local_acquired", groupID, sessionHash, requestedModel, account)
				return &AccountSelectionResult{Account: account, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
			}
			if waitResult == nil {
				phase := "local_wait"
				timeout := cfg.FallbackWaitTimeout
				maxWaiting := cfg.FallbackMaxWaiting
				if stickyAccountID > 0 && stickyAccountID == account.ID && s.concurrencyService != nil {
					waitingCount, _ := s.concurrencyService.GetAccountWaitingCount(ctx, account.ID)
					if waitingCount < cfg.StickySessionMaxWaiting {
						phase = "local_sticky_wait"
						timeout = cfg.StickySessionWaitTimeout
						maxWaiting = cfg.StickySessionMaxWaiting
					}
				}
				s.logSelectedAccountUsagePressure(phase, groupID, sessionHash, requestedModel, account)
				waitResult = &AccountSelectionResult{Account: account, WaitPlan: &AccountWaitPlan{AccountID: account.ID, MaxConcurrency: DeepSeekEffectiveAccountConcurrency(account, requestedModel), Timeout: timeout, MaxWaiting: maxWaiting}}
			}
			localExcluded[account.ID] = struct{}{}
		}
	}
	accounts, err := s.listSchedulableAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, errors.New("no available accounts")
	}
	var stickyWaitResult *AccountSelectionResult
	isExcluded := func(accountID int64) bool {
		if excludedIDs == nil {
			return false
		}
		_, excluded := excludedIDs[accountID]
		return excluded
	}
	if sessionHash != "" {
		accountID := stickyAccountID
		if accountID > 0 && !isExcluded(accountID) {
			account, err := s.getSchedulableAccount(ctx, accountID)
			if err == nil {
				clearSticky := shouldClearStickySession(account, requestedModel)
				if clearSticky {
					_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
				}
				if !clearSticky &&
					account.IsSchedulable() &&
					isOpenAITextRuntimeAccount(account) &&
					(requestedModel == "" || s.isModelSupportedByAccountWithContext(ctx, account, requestedModel)) &&
					account.IsSchedulableForModelWithContext(ctx, requestedModel) &&
					SupportsOpenAIEndpointCapability(account, requiredCapability) {
					account = s.recheckSelectedOpenAIAccountFromDB(ctx, account, requestedModel)
					if account == nil {
						_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
					} else {
						result, err := s.tryAcquireAccountSlot(ctx, accountID, DeepSeekEffectiveAccountConcurrency(account, requestedModel))
						if err == nil && result != nil && result.Acquired {
							_ = s.refreshStickySessionTTL(ctx, groupID, sessionHash, openaiStickySessionTTL)
							s.logSelectedAccountUsagePressure("sticky_acquired", groupID, sessionHash, requestedModel, account)
							return &AccountSelectionResult{Account: account, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
						}
						if err == nil {
							stickyWaitResult = s.buildOpenAIStickyWaitSelection(ctx, groupID, sessionHash, requestedModel, account, cfg)
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
		if !acc.IsSchedulable() {
			continue
		}
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, acc, requestedModel) {
			continue
		}
		if !acc.IsSchedulableForModelWithContext(ctx, requestedModel) {
			continue
		}
		if !SupportsOpenAIEndpointCapability(acc, requiredCapability) {
			continue
		}
		candidates = append(candidates, acc)
	}
	if len(candidates) == 0 {
		if stickyWaitResult != nil {
			return stickyWaitResult, nil
		}
		if s.openAISelectionFailureIsModelUnsupported(ctx, accounts, requestedModel, excludedIDs) {
			return nil, fmt.Errorf("%w: %s", ErrOpenAIModelNotFound, requestedModel)
		}
		return nil, ErrNoAvailableAccounts
	}
	accountLoads := make([]AccountWithConcurrency, 0, len(candidates))
	for _, acc := range candidates {
		maxConcurrency := acc.EffectiveLoadFactor()
		if effective := DeepSeekEffectiveAccountConcurrency(acc, requestedModel); effective > 0 && effective < maxConcurrency {
			maxConcurrency = effective
		}
		accountLoads = append(accountLoads, AccountWithConcurrency{ID: acc.ID, MaxConcurrency: maxConcurrency})
	}
	loadMap, err := s.concurrencyService.GetAccountsLoadBatch(ctx, accountLoads)
	if err != nil {
		remaining := append([]*Account(nil), candidates...)
		now := time.Now()
		for len(remaining) > 0 {
			bestIndex := 0
			for i := 1; i < len(remaining); i++ {
				if compareOpenAIAccountsForSelection(remaining[i], remaining[bestIndex], requestedModel, now) < 0 {
					bestIndex = i
				}
			}
			acc := remaining[bestIndex]
			remaining = append(remaining[:bestIndex], remaining[bestIndex+1:]...)
			fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel)
			if fresh == nil {
				continue
			}
			if !SupportsOpenAIEndpointCapability(fresh, requiredCapability) {
				continue
			}
			fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel)
			if fresh == nil {
				continue
			}
			result, err := s.tryAcquireAccountSlot(ctx, fresh.ID, DeepSeekEffectiveAccountConcurrency(fresh, requestedModel))
			if err == nil && result.Acquired {
				if sessionHash != "" {
					_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, fresh.ID, openaiStickySessionTTL)
				}
				s.logSelectedAccountUsagePressure("legacy_acquired", groupID, sessionHash, requestedModel, fresh)
				return &AccountSelectionResult{Account: fresh, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
			}
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
		if len(available) > 0 {
			now := time.Now()
			sort.SliceStable(available, func(i, j int) bool {
				return compareOpenAIAccountsWithLoad(available[i], available[j], requestedModel, now) < 0
			})
			for _, item := range available {
				fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, item.account, requestedModel)
				if fresh == nil {
					continue
				}
				if !SupportsOpenAIEndpointCapability(fresh, requiredCapability) {
					continue
				}
				fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel)
				if fresh == nil {
					continue
				}
				result, err := s.tryAcquireAccountSlot(ctx, fresh.ID, DeepSeekEffectiveAccountConcurrency(fresh, requestedModel))
				if err == nil && result.Acquired {
					if sessionHash != "" {
						_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, fresh.ID, openaiStickySessionTTL)
					}
					s.logSelectedAccountUsagePressure("load_acquired", groupID, sessionHash, requestedModel, fresh)
					return &AccountSelectionResult{Account: fresh, Acquired: true, ReleaseFunc: result.ReleaseFunc}, nil
				}
			}
		}
	}
	remaining := append([]*Account(nil), candidates...)
	now := time.Now()
	for len(remaining) > 0 {
		bestIndex := 0
		for i := 1; i < len(remaining); i++ {
			if compareOpenAIAccountsForSelection(remaining[i], remaining[bestIndex], requestedModel, now) < 0 {
				bestIndex = i
			}
		}
		acc := remaining[bestIndex]
		remaining = append(remaining[:bestIndex], remaining[bestIndex+1:]...)
		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel)
		if fresh == nil {
			continue
		}
		if !SupportsOpenAIEndpointCapability(fresh, requiredCapability) {
			continue
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel)
		if fresh == nil {
			continue
		}
		if stickyWaitResult != nil {
			return stickyWaitResult, nil
		}
		s.logSelectedAccountUsagePressure("load_wait", groupID, sessionHash, requestedModel, fresh)
		return &AccountSelectionResult{Account: fresh, WaitPlan: &AccountWaitPlan{AccountID: fresh.ID, MaxConcurrency: DeepSeekEffectiveAccountConcurrency(fresh, requestedModel), Timeout: cfg.FallbackWaitTimeout, MaxWaiting: cfg.FallbackMaxWaiting}}, nil
	}
	if stickyWaitResult != nil {
		return stickyWaitResult, nil
	}
	return nil, ErrNoAvailableAccounts
}

func (s *OpenAIGatewayService) openAISelectionFailureIsModelUnsupported(ctx context.Context, accounts []Account, requestedModel string, excludedIDs map[int64]struct{}) bool {
	if strings.TrimSpace(requestedModel) == "" || len(accounts) == 0 {
		return false
	}
	eligible := 0
	unsupported := 0
	for i := range accounts {
		account := &accounts[i]
		if account == nil {
			continue
		}
		if excludedIDs != nil {
			if _, excluded := excludedIDs[account.ID]; excluded {
				continue
			}
		}
		if !account.IsSchedulable() || !isOpenAITextRuntimeAccount(account) {
			continue
		}
		eligible++
		if !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
			unsupported++
		}
	}
	return eligible > 0 && eligible == unsupported
}

func (s *OpenAIGatewayService) buildOpenAIStickyWaitSelection(
	ctx context.Context,
	groupID *int64,
	sessionHash string,
	requestedModel string,
	account *Account,
	cfg config.GatewaySchedulingConfig,
) *AccountSelectionResult {
	if account == nil {
		return nil
	}
	if s.concurrencyService != nil && cfg.StickySessionMaxWaiting > 0 {
		waitingCount, err := s.concurrencyService.GetAccountWaitingCount(ctx, account.ID)
		if err == nil && waitingCount >= cfg.StickySessionMaxWaiting {
			return nil
		}
	}
	s.logSelectedAccountUsagePressure("sticky_wait_prepared", groupID, sessionHash, requestedModel, account)
	return &AccountSelectionResult{
		Account: account,
		WaitPlan: &AccountWaitPlan{
			AccountID:      account.ID,
			MaxConcurrency: DeepSeekEffectiveAccountConcurrency(account, requestedModel),
			Timeout:        cfg.StickySessionWaitTimeout,
			MaxWaiting:     cfg.StickySessionMaxWaiting,
		},
	}
}
func (s *OpenAIGatewayService) listSchedulableAccounts(ctx context.Context, groupID *int64) ([]Account, error) {
	platform := OpenAIPlatformFromContext(ctx)
	if s.schedulerSnapshot != nil {
		accounts, _, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, false)
		if err != nil {
			return nil, err
		}
		return filterOpenAITextRuntimeAccounts(accounts, platform), nil
	}
	var accounts []Account
	var err error
	queryPlatforms := QueryPlatformsForGroupPlatform(platform, false)
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		accounts, err = s.accountRepo.ListSchedulableByPlatforms(ctx, queryPlatforms)
	} else if groupID != nil {
		accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, queryPlatforms)
	} else {
		accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, queryPlatforms)
	}
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}
	filtered := make([]Account, 0, len(accounts))
	for _, acc := range accounts {
		resolved := ResolveProtocolGatewayInboundAccount(&acc, platform)
		if !isOpenAITextRuntimeAccount(resolved) {
			continue
		}
		filtered = append(filtered, *resolved)
	}
	return filtered, nil
}

func filterOpenAITextRuntimeAccounts(accounts []Account, platform string) []Account {
	filtered := make([]Account, 0, len(accounts))
	for _, acc := range accounts {
		resolved := ResolveProtocolGatewayInboundAccount(&acc, platform)
		if !isOpenAITextRuntimeAccount(resolved) {
			continue
		}
		filtered = append(filtered, *resolved)
	}
	return filtered
}
func (s *OpenAIGatewayService) tryAcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (*AcquireResult, error) {
	if s.concurrencyService == nil {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {
		}}, nil
	}
	return s.concurrencyService.AcquireAccountSlot(ctx, accountID, maxConcurrency)
}
func (s *OpenAIGatewayService) resolveFreshSchedulableOpenAIAccount(ctx context.Context, account *Account, requestedModel string) *Account {
	if account == nil {
		return nil
	}
	fresh := account
	if s.schedulerSnapshot != nil {
		current, err := s.getSchedulableAccount(ctx, account.ID)
		if err != nil || current == nil {
			return nil
		}
		fresh = current
	}
	if !fresh.IsSchedulable() || !isOpenAITextRuntimeAccount(fresh) {
		return nil
	}
	if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, fresh, requestedModel) {
		return nil
	}
	if !fresh.IsSchedulableForModelWithContext(ctx, requestedModel) {
		return nil
	}
	return fresh
}

func isOpenAITextRuntimeAccount(account *Account) bool {
	return account != nil && account.IsOpenAITextCompatible()
}
func (s *OpenAIGatewayService) getSchedulableAccount(ctx context.Context, accountID int64) (*Account, error) {
	var (
		account *Account
		err     error
	)
	if s.schedulerSnapshot != nil {
		account, err = s.schedulerSnapshot.GetAccount(ctx, accountID)
	} else {
		account, err = s.accountRepo.GetByID(ctx, accountID)
	}
	if err != nil || account == nil {
		return account, err
	}
	syncOpenAICodexRateLimitFromExtra(ctx, s.accountRepo, account, time.Now())
	return account, nil
}
func (s *OpenAIGatewayService) schedulingConfig() config.GatewaySchedulingConfig {
	if s.cfg != nil {
		return s.cfg.Gateway.Scheduling
	}
	return config.GatewaySchedulingConfig{StickySessionMaxWaiting: 3, StickySessionWaitTimeout: 45 * time.Second, FallbackWaitTimeout: 30 * time.Second, FallbackMaxWaiting: 100, LoadBatchEnabled: true, SlotCleanupInterval: 30 * time.Second}
}

func (s *OpenAIGatewayService) logSelectedAccountUsagePressure(
	phase string,
	groupID *int64,
	sessionHash string,
	requestedModel string,
	account *Account,
) {
	if account == nil {
		return
	}
	pressure := buildOpenAIAccountUsagePressure(account, requestedModel, time.Now())
	window, utilization, resetAt := accountUsagePressureLogValues(pressure)
	pressureScope := resolveOpenAIAccountUsagePressureScope(account, requestedModel)
	if pressure != nil && strings.TrimSpace(pressure.scope) != "" {
		pressureScope = pressure.scope
	}
	slog.Debug(
		"openai_account_selection_pressure",
		"phase", phase,
		"group_id", derefGroupID(groupID),
		"model", requestedModel,
		"session", shortSessionHash(sessionHash),
		"account_id", account.ID,
		"account_type", account.Type,
		"priority", account.Priority,
		"selection_concurrency", resolveOpenAIAccountSelectionConcurrency(account),
		"plan_type", openAIAccountPlanType(account),
		"plan_rank", resolveOpenAIAccountPlanRankForLog(account),
		"pressure_scope", pressureScope,
		"pressure_window", window,
		"pressure_utilization", utilization,
		"pressure_reset_at", resetAt,
	)
}
func (s *OpenAIGatewayService) GetAccessToken(ctx context.Context, account *Account) (string, string, error) {
	switch account.Type {
	case AccountTypeOAuth:
		if s.openAITokenProvider != nil {
			accessToken, err := s.openAITokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return "", "", err
			}
			return accessToken, "oauth", nil
		}
		accessToken := account.GetOpenAIAccessToken()
		if accessToken == "" {
			return "", "", errors.New("access_token not found in credentials")
		}
		return accessToken, "oauth", nil
	case AccountTypeAPIKey:
		apiKey := account.GetOpenAIApiKey()
		if account.Platform == PlatformDeepSeek {
			apiKey = strings.TrimSpace(account.GetCredential("api_key"))
		}
		if account.Platform == PlatformOpenRouter {
			apiKey = account.GetOpenRouterAPIKey()
		}
		if apiKey == "" {
			return "", "", errors.New("api_key not found in credentials")
		}
		return apiKey, "apikey", nil
	default:
		return "", "", fmt.Errorf("unsupported account type: %s", account.Type)
	}
}
