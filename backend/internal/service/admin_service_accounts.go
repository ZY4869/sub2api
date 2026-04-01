package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"strings"
	"time"
)

func (s *adminServiceImpl) ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) ([]Account, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	accounts, result, err := s.accountRepo.ListWithFilters(ctx, params, platform, accountType, status, search, groupID, lifecycle, privacyMode)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	for i := range accounts {
		syncOpenAICodexRateLimitFromExtra(ctx, s.accountRepo, &accounts[i], now)
	}
	return accounts, result.Total, nil
}
func (s *adminServiceImpl) GetAccountStatusSummary(ctx context.Context, filters AccountStatusSummaryFilters) (*AccountStatusSummary, error) {
	filters.Platform = strings.TrimSpace(filters.Platform)
	filters.AccountType = strings.TrimSpace(filters.AccountType)
	filters.Search = strings.TrimSpace(filters.Search)
	filters.Lifecycle = NormalizeAccountLifecycleInput(filters.Lifecycle)
	filters.LimitedView = NormalizeAccountLimitedViewInput(filters.LimitedView)
	filters.LimitedReason = NormalizeAccountRateLimitReasonInput(filters.LimitedReason)
	filters.RuntimeView = NormalizeAccountRuntimeViewInput(filters.RuntimeView)
	if filters.Lifecycle == AccountLifecycleAll {
		filters.Lifecycle = AccountLifecycleAll
	}
	return s.accountRepo.GetStatusSummary(ctx, filters)
}
func (s *adminServiceImpl) GetAccount(ctx context.Context, id int64) (*Account, error) {
	return s.accountRepo.GetByID(ctx, id)
}
func (s *adminServiceImpl) GetAccountsByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	if len(ids) == 0 {
		return []*Account{}, nil
	}
	accounts, err := s.accountRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts by IDs: %w", err)
	}
	return accounts, nil
}
func (s *adminServiceImpl) CreateAccount(ctx context.Context, input *CreateAccountInput) (*Account, error) {
	if err := validateProtocolGatewayAccountInput(input.Platform, input.Type, input.Extra); err != nil {
		return nil, err
	}
	if err := validateGrokAccountInput(input.Platform, input.Type, input.Credentials, input.Extra); err != nil {
		return nil, err
	}
	bindingPlatform := RoutingPlatformFromValues(input.Platform, input.Extra)
	groupIDs := input.GroupIDs
	if len(groupIDs) == 0 && !input.SkipDefaultGroupBind {
		defaultGroupName := bindingPlatform + "-default"
		groups, err := s.groupRepo.ListActiveByPlatform(ctx, bindingPlatform)
		if err == nil {
			for _, g := range groups {
				if g.Name == defaultGroupName {
					groupIDs = []int64{g.ID}
					break
				}
			}
		}
	}
	if len(groupIDs) > 0 {
		if err := s.validateGroupIDsExist(ctx, groupIDs); err != nil {
			return nil, err
		}
		if err := s.validateAccountGroupBindings(ctx, groupIDs, input.Platform, input.Extra); err != nil {
			return nil, err
		}
	}
	if len(groupIDs) > 0 && shouldEnforceMixedChannelCheck(bindingPlatform, input.SkipMixedChannelCheck) {
		if err := s.checkMixedChannelRisk(ctx, 0, bindingPlatform, groupIDs); err != nil {
			return nil, err
		}
	}
	if input.Platform == PlatformSora && input.Type == AccountTypeAPIKey {
		baseURL, _ := input.Credentials["base_url"].(string)
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			return nil, errors.New("sora apikey 账号必须设置 base_url")
		}
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			return nil, errors.New("base_url 必须以 http:// 或 https:// 开头")
		}
	}
	accountStatus := strings.TrimSpace(input.Status)
	lifecycleState := NormalizeAccountLifecycleInput(input.LifecycleState)
	if lifecycleState == AccountLifecycleAll {
		lifecycleState = AccountLifecycleNormal
	}
	if accountStatus == "" {
		accountStatus = StatusActive
	}
	schedulable := true
	if lifecycleState == AccountLifecycleBlacklisted {
		accountStatus = StatusDisabled
		schedulable = false
	}
	credentials := input.Credentials
	if strings.EqualFold(strings.TrimSpace(input.Platform), PlatformKiro) {
		credentials = NormalizeKiroCredentialsForStorage(credentials)
	}
	if strings.EqualFold(strings.TrimSpace(input.Platform), PlatformGemini) {
		credentials = NormalizeGeminiCredentialsForStorage(input.Type, credentials)
	}
	if strings.EqualFold(strings.TrimSpace(input.Platform), PlatformGrok) {
		input.Extra = normalizeGrokExtraForStorageByType(input.Type, input.Extra)
		credentials = normalizeGrokCredentialsForStorage(input.Type, credentials, ResolveGrokTier(input.Extra))
	}
	account := &Account{
		Name:                   input.Name,
		Notes:                  normalizeAccountNotes(input.Notes),
		Platform:               input.Platform,
		Type:                   input.Type,
		Credentials:            credentials,
		Extra:                  input.Extra,
		ProxyID:                input.ProxyID,
		Concurrency:            input.Concurrency,
		Priority:               input.Priority,
		Status:                 accountStatus,
		Schedulable:            schedulable,
		LifecycleState:         lifecycleState,
		LifecycleReasonCode:    strings.TrimSpace(input.LifecycleReasonCode),
		LifecycleReasonMessage: strings.TrimSpace(input.LifecycleReasonMessage),
	}
	if input.ExpiresAt != nil && *input.ExpiresAt > 0 {
		expiresAt := time.Unix(*input.ExpiresAt, 0)
		account.ExpiresAt = &expiresAt
	}
	if input.AutoPauseOnExpired != nil {
		account.AutoPauseOnExpired = *input.AutoPauseOnExpired
	} else {
		account.AutoPauseOnExpired = true
	}
	if input.RateMultiplier != nil {
		if *input.RateMultiplier < 0 {
			return nil, errors.New("rate_multiplier must be >= 0")
		}
		account.RateMultiplier = input.RateMultiplier
	}
	if input.LoadFactor != nil && *input.LoadFactor > 0 {
		if *input.LoadFactor > 10000 {
			return nil, errors.New("load_factor must be <= 10000")
		}
		account.LoadFactor = input.LoadFactor
	}
	if lifecycleState == AccountLifecycleBlacklisted {
		now := time.Now()
		purgeAt := now.Add(AccountBlacklistRetention)
		account.BlacklistedAt = &now
		account.BlacklistPurgeAt = &purgeAt
	}
	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, err
	}
	if account.Platform == PlatformSora && s.soraAccountRepo != nil {
		soraUpdates := map[string]any{"access_token": account.GetCredential("access_token"), "refresh_token": account.GetCredential("refresh_token")}
		if err := s.soraAccountRepo.Upsert(ctx, account.ID, soraUpdates); err != nil {
			logger.LegacyPrintf("service.admin", "[AdminService] 创建 sora_accounts 记录失败: account_id=%d err=%v", account.ID, err)
		}
	}
	if len(groupIDs) > 0 {
		if err := s.accountRepo.BindGroups(ctx, account.ID, groupIDs); err != nil {
			return nil, err
		}
	}
	return account, nil
}
func (s *adminServiceImpl) UpdateAccount(ctx context.Context, id int64, input *UpdateAccountInput) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ensureBlacklistedAccountNotRestored(account, input.Status, nil); err != nil {
		return nil, err
	}
	if input.Name != "" {
		account.Name = input.Name
	}
	if input.Type != "" {
		account.Type = input.Type
	}
	if input.Notes != nil {
		account.Notes = normalizeAccountNotes(input.Notes)
	}
	if len(input.Credentials) > 0 {
		credentials := input.Credentials
		if strings.EqualFold(strings.TrimSpace(account.Platform), PlatformKiro) {
			credentials = NormalizeKiroCredentialsForStorage(credentials)
		}
		if strings.EqualFold(strings.TrimSpace(account.Platform), PlatformGemini) {
			credentials = NormalizeGeminiCredentialsForStorage(account.Type, credentials)
		}
		account.Credentials = credentials
	} else if strings.EqualFold(strings.TrimSpace(account.Platform), PlatformKiro) {
		NormalizeKiroAccountCredentials(account)
	}
	if input.Extra != nil {
		for _, key := range []string{"quota_used", "quota_daily_used", "quota_daily_start", "quota_weekly_used", "quota_weekly_start"} {
			if v, ok := account.Extra[key]; ok {
				input.Extra[key] = v
			}
		}
		account.Extra = input.Extra
	}
	sanitizeAntigravityOveragesExtra(account.Platform, account.Extra)
	if err := validateProtocolGatewayAccountInput(account.Platform, account.Type, account.Extra); err != nil {
		return nil, err
	}
	if err := validateGrokAccountInput(account.Platform, account.Type, account.Credentials, account.Extra); err != nil {
		return nil, err
	}
	if input.ProxyID != nil {
		if *input.ProxyID == 0 {
			account.ProxyID = nil
		} else {
			account.ProxyID = input.ProxyID
		}
		account.Proxy = nil
	}
	if input.Concurrency != nil {
		account.Concurrency = *input.Concurrency
	}
	if input.Priority != nil {
		account.Priority = *input.Priority
	}
	if input.RateMultiplier != nil {
		if *input.RateMultiplier < 0 {
			return nil, errors.New("rate_multiplier must be >= 0")
		}
		account.RateMultiplier = input.RateMultiplier
	}
	if input.LoadFactor != nil {
		if *input.LoadFactor <= 0 {
			account.LoadFactor = nil
		} else if *input.LoadFactor > 10000 {
			return nil, errors.New("load_factor must be <= 10000")
		} else {
			account.LoadFactor = input.LoadFactor
		}
	}
	if input.Status != "" {
		account.Status = input.Status
	}
	if input.ExpiresAt != nil {
		if *input.ExpiresAt <= 0 {
			account.ExpiresAt = nil
		} else {
			expiresAt := time.Unix(*input.ExpiresAt, 0)
			account.ExpiresAt = &expiresAt
		}
	}
	if input.AutoPauseOnExpired != nil {
		account.AutoPauseOnExpired = *input.AutoPauseOnExpired
	}
	if account.Platform == PlatformSora && account.Type == AccountTypeAPIKey {
		baseURL, _ := account.Credentials["base_url"].(string)
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			return nil, errors.New("sora apikey 账号必须设置 base_url")
		}
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			return nil, errors.New("base_url 必须以 http:// 或 https:// 开头")
		}
	}
	if strings.EqualFold(strings.TrimSpace(account.Platform), PlatformGrok) {
		account.Extra = normalizeGrokExtraForStorageByType(account.Type, account.Extra)
		account.Credentials = normalizeGrokCredentialsForStorage(account.Type, account.Credentials, ResolveGrokTier(account.Extra))
	}
	if input.GroupIDs != nil {
		if err := s.validateGroupIDsExist(ctx, *input.GroupIDs); err != nil {
			return nil, err
		}
		if err := s.validateAccountGroupBindings(ctx, *input.GroupIDs, account.Platform, account.Extra); err != nil {
			return nil, err
		}
		bindingPlatform := RoutingPlatformFromValues(account.Platform, account.Extra)
		if shouldEnforceMixedChannelCheck(bindingPlatform, input.SkipMixedChannelCheck) {
			if err := s.checkMixedChannelRisk(ctx, account.ID, bindingPlatform, *input.GroupIDs); err != nil {
				return nil, err
			}
		}
	}
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	if input.GroupIDs != nil {
		if err := s.accountRepo.BindGroups(ctx, account.ID, *input.GroupIDs); err != nil {
			return nil, err
		}
	}
	updated, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}
func (s *adminServiceImpl) BulkUpdateAccounts(ctx context.Context, input *BulkUpdateAccountsInput) (*BulkUpdateAccountsResult, error) {
	result := &BulkUpdateAccountsResult{SuccessIDs: make([]int64, 0, len(input.AccountIDs)), FailedIDs: make([]int64, 0, len(input.AccountIDs)), Results: make([]BulkUpdateAccountResult, 0, len(input.AccountIDs))}
	if len(input.AccountIDs) == 0 {
		return result, nil
	}
	if input.GroupIDs != nil {
		if err := s.validateGroupIDsExist(ctx, *input.GroupIDs); err != nil {
			return nil, err
		}
	}
	needMixedChannelCheck := input.GroupIDs != nil
	targetLifecycle := NormalizeAccountLifecycleInput(input.LifecycleState)
	needsArchiveSnapshot := input.GroupIDs != nil && targetLifecycle == AccountLifecycleArchived
	needAccountFetch := needMixedChannelCheck || input.Status != "" || input.Schedulable != nil || needsArchiveSnapshot
	platformByID := map[int64]string{}
	accountsByID := map[int64]*Account{}
	if needAccountFetch {
		accounts, err := s.accountRepo.GetByIDs(ctx, input.AccountIDs)
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			if account != nil {
				accountsByID[account.ID] = account
				platformByID[account.ID] = RoutingPlatformForAccount(account)
			}
		}
		if input.GroupIDs != nil {
			for _, account := range accountsByID {
				if account == nil {
					continue
				}
				if err := s.validateAccountGroupBindings(ctx, *input.GroupIDs, account.Platform, account.Extra); err != nil {
					return nil, err
				}
			}
		}
		if needMixedChannelCheck && input.SkipMixedChannelCheck {
			needMixedChannelCheck = false
			for _, platform := range platformByID {
				if shouldEnforceMixedChannelCheck(platform, true) {
					needMixedChannelCheck = true
					break
				}
			}
		}
	}
	if needsArchiveSnapshot {
		if err := s.captureArchiveRestoreSnapshots(ctx, accountsByID, input.AccountIDs); err != nil {
			return nil, err
		}
	}
	for _, accountID := range input.AccountIDs {
		if err := ensureBlacklistedAccountNotRestored(accountsByID[accountID], input.Status, input.Schedulable); err != nil {
			return nil, err
		}
	}
	if needMixedChannelCheck {
		for _, accountID := range input.AccountIDs {
			platform := platformByID[accountID]
			if platform == "" {
				continue
			}
			if err := s.checkMixedChannelRisk(ctx, accountID, platform, *input.GroupIDs); err != nil {
				return nil, err
			}
		}
	}
	if input.RateMultiplier != nil {
		if *input.RateMultiplier < 0 {
			return nil, errors.New("rate_multiplier must be >= 0")
		}
	}
	repoUpdates := AccountBulkUpdate{Credentials: input.Credentials, Extra: input.Extra}
	if input.Name != "" {
		repoUpdates.Name = &input.Name
	}
	if input.ProxyID != nil {
		repoUpdates.ProxyID = input.ProxyID
	}
	if input.Concurrency != nil {
		repoUpdates.Concurrency = input.Concurrency
	}
	if input.Priority != nil {
		repoUpdates.Priority = input.Priority
	}
	if input.RateMultiplier != nil {
		repoUpdates.RateMultiplier = input.RateMultiplier
	}
	if input.LoadFactor != nil {
		if *input.LoadFactor <= 0 {
			repoUpdates.LoadFactor = nil
		} else if *input.LoadFactor > 10000 {
			return nil, errors.New("load_factor must be <= 10000")
		} else {
			repoUpdates.LoadFactor = input.LoadFactor
		}
	}
	if input.Status != "" {
		repoUpdates.Status = &input.Status
	}
	if input.Schedulable != nil {
		repoUpdates.Schedulable = input.Schedulable
	}
	if input.LifecycleState != "" {
		lifecycleState := normalizeAccountLifecycleWriteInput(input.LifecycleState)
		repoUpdates.LifecycleState = &lifecycleState
		if lifecycleState == AccountLifecycleBlacklisted {
			disabledStatus := StatusDisabled
			repoUpdates.Status = &disabledStatus
			schedulable := false
			repoUpdates.Schedulable = &schedulable
		}
	}
	if trimmed := strings.TrimSpace(input.LifecycleReasonCode); trimmed != "" {
		repoUpdates.LifecycleReasonCode = &trimmed
	}
	if trimmed := strings.TrimSpace(input.LifecycleReasonMessage); trimmed != "" {
		repoUpdates.LifecycleReasonMessage = &trimmed
	}
	if _, err := s.accountRepo.BulkUpdate(ctx, input.AccountIDs, repoUpdates); err != nil {
		return nil, err
	}
	for _, accountID := range input.AccountIDs {
		entry := BulkUpdateAccountResult{AccountID: accountID}
		if input.GroupIDs != nil {
			if err := s.accountRepo.BindGroups(ctx, accountID, *input.GroupIDs); err != nil {
				entry.Success = false
				entry.Error = err.Error()
				result.Failed++
				result.FailedIDs = append(result.FailedIDs, accountID)
				result.Results = append(result.Results, entry)
				continue
			}
		}
		entry.Success = true
		result.Success++
		result.SuccessIDs = append(result.SuccessIDs, accountID)
		result.Results = append(result.Results, entry)
	}
	return result, nil
}

func validateProtocolGatewayAccountInput(platform string, accountType string, extra map[string]any) error {
	if !IsProtocolGatewayPlatform(platform) {
		return nil
	}
	if accountType != AccountTypeAPIKey {
		return errors.New("protocol_gateway accounts only support apikey type")
	}
	protocol := ResolveAccountGatewayProtocol(platform, extra)
	if protocol == "" {
		return errors.New("protocol_gateway accounts require gateway_protocol")
	}
	acceptedProtocols := ResolveAccountGatewayAcceptedProtocols(platform, extra)
	if len(acceptedProtocols) == 0 {
		return errors.New("protocol_gateway accounts require at least one accepted protocol")
	}
	profiles := ResolveAccountGatewayClientProfiles(platform, extra)
	for _, profile := range profiles {
		if NormalizeGatewayClientProfile(profile) == "" {
			return errors.New("protocol_gateway accounts contain invalid gateway_client_profiles")
		}
	}
	routes := ResolveAccountGatewayClientRoutes(platform, extra)
	if rawRoutes, ok := extra[gatewayExtraClientRoutesKey]; ok {
		rawItems, ok := rawRoutes.([]any)
		if ok && len(rawItems) > 0 && len(routes) != len(rawItems) {
			return errors.New("protocol_gateway accounts contain invalid gateway_client_routes")
		}
	}
	if protocol != GatewayProtocolMixed && len(acceptedProtocols) != 1 {
		return errors.New("single protocol protocol_gateway accounts cannot use multiple accepted protocols")
	}
	return nil
}

func validateGrokAccountInput(platform string, accountType string, credentials map[string]any, extra map[string]any) error {
	if !strings.EqualFold(strings.TrimSpace(platform), PlatformGrok) {
		return nil
	}
	switch strings.TrimSpace(strings.ToLower(accountType)) {
	case AccountTypeAPIKey, AccountTypeSSO:
	default:
		return errors.New("grok accounts only support apikey or sso type")
	}
	if len(credentials) == 0 {
		return errors.New("grok accounts require credentials")
	}
	if strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeAPIKey {
		apiKey, _ := credentials["api_key"].(string)
		if strings.TrimSpace(apiKey) == "" {
			return errors.New("grok apikey accounts require credentials.api_key")
		}
	}
	if strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeSSO {
		ssoToken, _ := credentials["sso_token"].(string)
		if strings.TrimSpace(ssoToken) == "" {
			return errors.New("grok sso accounts require credentials.sso_token")
		}
	}
	if strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeSSO && len(extra) > 0 {
		if rawTier, ok := extra["grok_tier"].(string); ok && strings.TrimSpace(rawTier) != "" && normalizeGrokTier(extra) == "" {
			return errors.New("grok accounts require extra.grok_tier: basic|super|heavy")
		}
		if rawCapabilities, ok := extra["grok_capabilities"]; ok {
			if _, ok := rawCapabilities.(map[string]any); !ok {
				return errors.New("grok accounts require extra.grok_capabilities to be an object")
			}
		}
	}
	return nil
}

func normalizeGrokCredentialsForStorage(accountType string, credentials map[string]any, tier string) map[string]any {
	if len(credentials) == 0 {
		credentials = map[string]any{}
	}
	normalized := make(map[string]any, len(credentials)+2)
	for key, value := range credentials {
		normalized[key] = value
	}
	tier = NormalizeGrokTierValue(tier)
	if tier == "" {
		tier = GrokTierBasic
	}
	switch strings.TrimSpace(strings.ToLower(accountType)) {
	case AccountTypeAPIKey:
		apiKey, _ := normalized["api_key"].(string)
		normalized["api_key"] = NormalizeGrokCredentialValue(AccountTypeAPIKey, apiKey)
		baseURL, _ := normalized["base_url"].(string)
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			normalized["base_url"] = "https://api.x.ai"
		} else {
			normalized["base_url"] = strings.TrimRight(baseURL, "/")
		}
		if rawMapping, ok := normalized["model_mapping"].(map[string]any); ok {
			if nextMapping := normalizeGrokModelMappingForStorage(AccountTypeAPIKey, rawMapping, tier); len(nextMapping) > 0 {
				normalized["model_mapping"] = nextMapping
			} else {
				delete(normalized, "model_mapping")
			}
		}
	case AccountTypeSSO:
		ssoToken, _ := normalized["sso_token"].(string)
		normalized["sso_token"] = NormalizeGrokCredentialValue(AccountTypeSSO, ssoToken)
		rawMapping, _ := normalized["model_mapping"].(map[string]any)
		normalized["model_mapping"] = normalizeGrokModelMappingForStorage(AccountTypeSSO, rawMapping, tier)
	}
	return normalized
}

func normalizeGrokExtraForStorageByType(accountType string, extra map[string]any) map[string]any {
	if len(extra) == 0 {
		if strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeSSO {
			extra = map[string]any{}
		} else {
			return nil
		}
	}
	normalized := make(map[string]any, len(extra)+1)
	for key, value := range extra {
		normalized[key] = value
	}
	if strings.TrimSpace(strings.ToLower(accountType)) != AccountTypeSSO {
		delete(normalized, "grok_tier")
		delete(normalized, "grok_capabilities")
		if len(normalized) == 0 {
			return nil
		}
		return normalized
	}
	tier := normalizeGrokTier(extra)
	if tier == "" {
		tier = GrokTierBasic
	}
	normalized["grok_tier"] = tier
	normalized["grok_capabilities"] = ResolveGrokCapabilities(normalized).ToMap()
	return normalized
}

func normalizeGrokTier(extra map[string]any) string {
	if len(extra) == 0 {
		return ""
	}
	value, _ := extra["grok_tier"].(string)
	return NormalizeGrokTierValue(value)
}

func defaultGrokCapabilitiesForTier(tier string) map[string]any {
	return DefaultGrokCapabilitiesForTier(tier).ToMap()
}

func (s *adminServiceImpl) DeleteAccount(ctx context.Context, id int64) error {
	if err := s.accountRepo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}
func (s *adminServiceImpl) RefreshAccountCredentials(ctx context.Context, id int64) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return account, nil
}
func (s *adminServiceImpl) ClearAccountError(ctx context.Context, id int64) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleBlacklisted {
		return nil, ensureBlacklistedAccountNotRestored(account, StatusActive, nil)
	}
	if err := s.accountRepo.ClearError(ctx, id); err != nil {
		return nil, err
	}
	if err := s.accountRepo.ClearRateLimit(ctx, id); err != nil {
		return nil, err
	}
	if err := s.accountRepo.ClearAntigravityQuotaScopes(ctx, id); err != nil {
		return nil, err
	}
	if err := s.accountRepo.ClearModelRateLimits(ctx, id); err != nil {
		return nil, err
	}
	if err := s.accountRepo.ClearTempUnschedulable(ctx, id); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, id)
}
func (s *adminServiceImpl) SetAccountError(ctx context.Context, id int64, errorMsg string) error {
	return s.accountRepo.SetError(ctx, id, errorMsg)
}
func (s *adminServiceImpl) SetAccountSchedulable(ctx context.Context, id int64, schedulable bool) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ensureBlacklistedAccountNotRestored(account, "", &schedulable); err != nil {
		return nil, err
	}
	if err := s.accountRepo.SetSchedulable(ctx, id, schedulable); err != nil {
		return nil, err
	}
	updated, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}
