package service

import (
	"context"
	"fmt"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"time"
)

func (s *adminServiceImpl) ListGroups(ctx context.Context, page, pageSize int, platform, status, search string, isExclusive *bool) ([]Group, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	groups, result, err := s.groupRepo.ListWithFilters(ctx, params, platform, status, search, isExclusive)
	if err != nil {
		return nil, 0, err
	}
	return groups, result.Total, nil
}
func (s *adminServiceImpl) GetAllGroups(ctx context.Context) ([]Group, error) {
	return s.groupRepo.ListActive(ctx)
}
func (s *adminServiceImpl) GetAllGroupsByPlatform(ctx context.Context, platform string) ([]Group, error) {
	return s.groupRepo.ListActiveByPlatform(ctx, platform)
}
func (s *adminServiceImpl) GetGroup(ctx context.Context, id int64) (*Group, error) {
	return s.groupRepo.GetByID(ctx, id)
}
func (s *adminServiceImpl) GetGroupByName(ctx context.Context, name string) (*Group, error) {
	return s.groupRepo.GetByName(ctx, name)
}
func (s *adminServiceImpl) CreateGroup(ctx context.Context, input *CreateGroupInput) (*Group, error) {
	platform := CanonicalizePlatformValue(input.Platform)
	if platform == "" {
		platform = PlatformAnthropic
	}
	subscriptionType := input.SubscriptionType
	if subscriptionType == "" {
		subscriptionType = SubscriptionTypeStandard
	}
	dailyLimit := normalizeLimit(input.DailyLimitUSD)
	weeklyLimit := normalizeLimit(input.WeeklyLimitUSD)
	monthlyLimit := normalizeLimit(input.MonthlyLimitUSD)
	imagePrice1K := normalizePrice(input.ImagePrice1K)
	imagePrice2K := normalizePrice(input.ImagePrice2K)
	imagePrice4K := normalizePrice(input.ImagePrice4K)
	if input.FallbackGroupID != nil {
		if err := s.validateFallbackGroup(ctx, 0, *input.FallbackGroupID); err != nil {
			return nil, err
		}
	}
	fallbackOnInvalidRequest := input.FallbackGroupIDOnInvalidRequest
	if fallbackOnInvalidRequest != nil && *fallbackOnInvalidRequest <= 0 {
		fallbackOnInvalidRequest = nil
	}
	if fallbackOnInvalidRequest != nil {
		if err := s.validateFallbackGroupOnInvalidRequest(ctx, 0, platform, subscriptionType, *fallbackOnInvalidRequest); err != nil {
			return nil, err
		}
	}
	mcpXMLInject := true
	if input.MCPXMLInject != nil {
		mcpXMLInject = *input.MCPXMLInject
	}
	priority := input.Priority
	if priority <= 0 {
		priority = 1
	}
	var accountIDsToCopy []int64
	if len(input.CopyAccountsFromGroupIDs) > 0 {
		seen := make(map[int64]struct{})
		uniqueSourceGroupIDs := make([]int64, 0, len(input.CopyAccountsFromGroupIDs))
		for _, srcGroupID := range input.CopyAccountsFromGroupIDs {
			if _, exists := seen[srcGroupID]; !exists {
				seen[srcGroupID] = struct{}{}
				uniqueSourceGroupIDs = append(uniqueSourceGroupIDs, srcGroupID)
			}
		}
		for _, srcGroupID := range uniqueSourceGroupIDs {
			srcGroup, err := s.groupRepo.GetByIDLite(ctx, srcGroupID)
			if err != nil {
				return nil, fmt.Errorf("source group %d not found: %w", srcGroupID, err)
			}
			if srcGroup.Platform != platform {
				return nil, fmt.Errorf("source group %d platform mismatch: expected %s, got %s", srcGroupID, platform, srcGroup.Platform)
			}
		}
		var err error
		accountIDsToCopy, err = s.groupRepo.GetAccountIDsByGroupIDs(ctx, uniqueSourceGroupIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get accounts from source groups: %w", err)
		}
	}
	group := &Group{Name: input.Name, Description: input.Description, Platform: platform, Priority: priority, RateMultiplier: input.RateMultiplier, IsExclusive: input.IsExclusive, Status: StatusActive, SubscriptionType: subscriptionType, DailyLimitUSD: dailyLimit, WeeklyLimitUSD: weeklyLimit, MonthlyLimitUSD: monthlyLimit, ImagePrice1K: imagePrice1K, ImagePrice2K: imagePrice2K, ImagePrice4K: imagePrice4K, ClaudeCodeOnly: input.ClaudeCodeOnly, FallbackGroupID: input.FallbackGroupID, FallbackGroupIDOnInvalidRequest: fallbackOnInvalidRequest, ModelRouting: input.ModelRouting, GeminiMixedProtocolEnabled: input.GeminiMixedProtocolEnabled, MCPXMLInject: mcpXMLInject, SupportedModelScopes: input.SupportedModelScopes, AllowMessagesDispatch: input.AllowMessagesDispatch, DefaultMappedModel: input.DefaultMappedModel}
	if err := s.groupRepo.Create(ctx, group); err != nil {
		return nil, err
	}
	if len(accountIDsToCopy) > 0 {
		if err := s.groupRepo.BindAccountsToGroup(ctx, group.ID, accountIDsToCopy); err != nil {
			return nil, fmt.Errorf("failed to bind accounts to new group: %w", err)
		}
		group.AccountCount = int64(len(accountIDsToCopy))
	}
	return group, nil
}
func normalizeLimit(limit *float64) *float64 {
	if limit == nil || *limit <= 0 {
		return nil
	}
	return limit
}
func normalizePrice(price *float64) *float64 {
	if price == nil || *price < 0 {
		return nil
	}
	return price
}
func (s *adminServiceImpl) validateFallbackGroup(ctx context.Context, currentGroupID, fallbackGroupID int64) error {
	if currentGroupID > 0 && currentGroupID == fallbackGroupID {
		return fmt.Errorf("cannot set self as fallback group")
	}
	visited := map[int64]struct{}{}
	nextID := fallbackGroupID
	for {
		if _, seen := visited[nextID]; seen {
			return fmt.Errorf("fallback group cycle detected")
		}
		visited[nextID] = struct{}{}
		if currentGroupID > 0 && nextID == currentGroupID {
			return fmt.Errorf("fallback group cycle detected")
		}
		fallbackGroup, err := s.groupRepo.GetByIDLite(ctx, nextID)
		if err != nil {
			return fmt.Errorf("fallback group not found: %w", err)
		}
		if nextID == fallbackGroupID && fallbackGroup.ClaudeCodeOnly {
			return fmt.Errorf("fallback group cannot have claude_code_only enabled")
		}
		if fallbackGroup.FallbackGroupID == nil {
			return nil
		}
		nextID = *fallbackGroup.FallbackGroupID
	}
}
func (s *adminServiceImpl) validateFallbackGroupOnInvalidRequest(ctx context.Context, currentGroupID int64, platform, subscriptionType string, fallbackGroupID int64) error {
	if platform != PlatformAnthropic && platform != PlatformAntigravity {
		return fmt.Errorf("invalid request fallback only supported for anthropic or antigravity groups")
	}
	if subscriptionType == SubscriptionTypeSubscription {
		return fmt.Errorf("subscription groups cannot set invalid request fallback")
	}
	if currentGroupID > 0 && currentGroupID == fallbackGroupID {
		return fmt.Errorf("cannot set self as invalid request fallback group")
	}
	fallbackGroup, err := s.groupRepo.GetByIDLite(ctx, fallbackGroupID)
	if err != nil {
		return fmt.Errorf("fallback group not found: %w", err)
	}
	if fallbackGroup.Platform != PlatformAnthropic {
		return fmt.Errorf("fallback group must be anthropic platform")
	}
	if fallbackGroup.SubscriptionType == SubscriptionTypeSubscription {
		return fmt.Errorf("fallback group cannot be subscription type")
	}
	if fallbackGroup.FallbackGroupIDOnInvalidRequest != nil {
		return fmt.Errorf("fallback group cannot have invalid request fallback configured")
	}
	return nil
}
func (s *adminServiceImpl) UpdateGroup(ctx context.Context, id int64, input *UpdateGroupInput) (*Group, error) {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	group.Platform = CanonicalizePlatformValue(group.Platform)
	if input.Name != "" {
		group.Name = input.Name
	}
	if input.Description != "" {
		group.Description = input.Description
	}
	if input.Platform != "" {
		group.Platform = CanonicalizePlatformValue(input.Platform)
	}
	if input.Priority != nil && *input.Priority > 0 {
		group.Priority = *input.Priority
	}
	if input.RateMultiplier != nil {
		group.RateMultiplier = *input.RateMultiplier
	}
	if input.IsExclusive != nil {
		group.IsExclusive = *input.IsExclusive
	}
	if input.Status != "" {
		group.Status = input.Status
	}
	if input.SubscriptionType != "" {
		group.SubscriptionType = input.SubscriptionType
	}
	if input.DailyLimitUSD != nil {
		group.DailyLimitUSD = normalizeLimit(input.DailyLimitUSD)
	}
	if input.WeeklyLimitUSD != nil {
		group.WeeklyLimitUSD = normalizeLimit(input.WeeklyLimitUSD)
	}
	if input.MonthlyLimitUSD != nil {
		group.MonthlyLimitUSD = normalizeLimit(input.MonthlyLimitUSD)
	}
	if input.ImagePrice1K != nil {
		group.ImagePrice1K = normalizePrice(input.ImagePrice1K)
	}
	if input.ImagePrice2K != nil {
		group.ImagePrice2K = normalizePrice(input.ImagePrice2K)
	}
	if input.ImagePrice4K != nil {
		group.ImagePrice4K = normalizePrice(input.ImagePrice4K)
	}
	if input.ClaudeCodeOnly != nil {
		group.ClaudeCodeOnly = *input.ClaudeCodeOnly
	}
	if input.FallbackGroupID != nil {
		if *input.FallbackGroupID > 0 {
			if err := s.validateFallbackGroup(ctx, id, *input.FallbackGroupID); err != nil {
				return nil, err
			}
			group.FallbackGroupID = input.FallbackGroupID
		} else {
			group.FallbackGroupID = nil
		}
	}
	fallbackOnInvalidRequest := group.FallbackGroupIDOnInvalidRequest
	if input.FallbackGroupIDOnInvalidRequest != nil {
		if *input.FallbackGroupIDOnInvalidRequest > 0 {
			fallbackOnInvalidRequest = input.FallbackGroupIDOnInvalidRequest
		} else {
			fallbackOnInvalidRequest = nil
		}
	}
	if fallbackOnInvalidRequest != nil {
		if err := s.validateFallbackGroupOnInvalidRequest(ctx, id, group.Platform, group.SubscriptionType, *fallbackOnInvalidRequest); err != nil {
			return nil, err
		}
	}
	group.FallbackGroupIDOnInvalidRequest = fallbackOnInvalidRequest
	if input.ModelRouting != nil {
		group.ModelRouting = input.ModelRouting
	}
	if input.ModelRoutingEnabled != nil {
		group.ModelRoutingEnabled = *input.ModelRoutingEnabled
	}
	if input.GeminiMixedProtocolEnabled != nil {
		group.GeminiMixedProtocolEnabled = *input.GeminiMixedProtocolEnabled
	}
	if input.MCPXMLInject != nil {
		group.MCPXMLInject = *input.MCPXMLInject
	}
	if input.SupportedModelScopes != nil {
		group.SupportedModelScopes = *input.SupportedModelScopes
	}
	if input.AllowMessagesDispatch != nil {
		group.AllowMessagesDispatch = *input.AllowMessagesDispatch
	}
	if input.DefaultMappedModel != nil {
		group.DefaultMappedModel = *input.DefaultMappedModel
	}
	if err := s.groupRepo.Update(ctx, group); err != nil {
		return nil, err
	}
	if len(input.CopyAccountsFromGroupIDs) > 0 {
		seen := make(map[int64]struct{})
		uniqueSourceGroupIDs := make([]int64, 0, len(input.CopyAccountsFromGroupIDs))
		for _, srcGroupID := range input.CopyAccountsFromGroupIDs {
			if srcGroupID == id {
				return nil, fmt.Errorf("cannot copy accounts from self")
			}
			if _, exists := seen[srcGroupID]; !exists {
				seen[srcGroupID] = struct{}{}
				uniqueSourceGroupIDs = append(uniqueSourceGroupIDs, srcGroupID)
			}
		}
		for _, srcGroupID := range uniqueSourceGroupIDs {
			srcGroup, err := s.groupRepo.GetByIDLite(ctx, srcGroupID)
			if err != nil {
				return nil, fmt.Errorf("source group %d not found: %w", srcGroupID, err)
			}
			if srcGroup.Platform != group.Platform {
				return nil, fmt.Errorf("source group %d platform mismatch: expected %s, got %s", srcGroupID, group.Platform, srcGroup.Platform)
			}
		}
		accountIDsToCopy, err := s.groupRepo.GetAccountIDsByGroupIDs(ctx, uniqueSourceGroupIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get accounts from source groups: %w", err)
		}
		if _, err := s.groupRepo.DeleteAccountGroupsByGroupID(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to clear existing account bindings: %w", err)
		}
		if len(accountIDsToCopy) > 0 {
			if err := s.groupRepo.BindAccountsToGroup(ctx, id, accountIDsToCopy); err != nil {
				return nil, fmt.Errorf("failed to bind accounts to group: %w", err)
			}
		}
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, id)
	}
	return group, nil
}
func (s *adminServiceImpl) DeleteGroup(ctx context.Context, id int64) error {
	var groupKeys []string
	if s.authCacheInvalidator != nil {
		keys, err := s.apiKeyRepo.ListKeysByGroupID(ctx, id)
		if err == nil {
			groupKeys = keys
		}
	}
	affectedUserIDs, err := s.groupRepo.DeleteCascade(ctx, id)
	if err != nil {
		return err
	}
	if len(affectedUserIDs) > 0 && s.billingCacheService != nil {
		groupID := id
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			for _, userID := range affectedUserIDs {
				if err := s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID); err != nil {
					logger.LegacyPrintf("service.admin", "invalidate subscription cache failed: user_id=%d group_id=%d err=%v", userID, groupID, err)
				}
			}
		}()
	}
	if s.authCacheInvalidator != nil {
		for _, key := range groupKeys {
			s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, key)
		}
	}
	return nil
}
func (s *adminServiceImpl) GetGroupAPIKeys(ctx context.Context, groupID int64, page, pageSize int) ([]APIKey, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	keys, result, err := s.apiKeyRepo.ListByGroupID(ctx, groupID, params)
	if err != nil {
		return nil, 0, err
	}
	return keys, result.Total, nil
}
func (s *adminServiceImpl) UpdateGroupSortOrders(ctx context.Context, updates []GroupSortOrderUpdate) error {
	return s.groupRepo.UpdateSortOrders(ctx, updates)
}

func (s *adminServiceImpl) GetAPIKeyGroups(ctx context.Context, keyID int64) ([]APIKeyGroupBinding, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	bindings, err := s.apiKeyRepo.GetAPIKeyGroups(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if len(bindings) > 0 {
		return bindings, nil
	}
	if len(apiKey.GroupBindings) > 0 {
		return append([]APIKeyGroupBinding(nil), apiKey.GroupBindings...), nil
	}
	if apiKey.GroupID != nil && apiKey.Group != nil {
		return []APIKeyGroupBinding{{
			APIKeyID:  apiKey.ID,
			GroupID:   *apiKey.GroupID,
			Group:     apiKey.Group,
			Quota:     0,
			QuotaUsed: 0,
		}}, nil
	}
	return nil, nil
}

func (s *adminServiceImpl) AdminUpdateAPIKeyGroups(ctx context.Context, keyID int64, inputs []AdminAPIKeyGroupUpdateInput, modelDisplayMode *string) (*AdminUpdateAPIKeyGroupsResult, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if modelDisplayMode != nil {
		apiKey.ModelDisplayMode = NormalizeAPIKeyModelDisplayMode(*modelDisplayMode)
	}
	if len(inputs) == 0 {
		if err := s.apiKeyRepo.SetAPIKeyGroups(ctx, keyID, nil); err != nil {
			return nil, fmt.Errorf("clear api key groups: %w", err)
		}
		apiKey.GroupBindings = nil
		apiKey.SelectedGroupBinding = nil
		apiKey.GroupID = nil
		apiKey.Group = nil
		apiKey.SyncLegacyGroupShadow()
		if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
			return nil, fmt.Errorf("update api key metadata: %w", err)
		}
		updatedAPIKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
		if err != nil {
			return nil, err
		}
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, updatedAPIKey.Key)
		}
		return &AdminUpdateAPIKeyGroupsResult{APIKey: updatedAPIKey}, nil
	}

	owner, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	groupInputs := make([]APIKeyGroupUpdateInput, 0, len(inputs))
	for _, input := range inputs {
		groupInputs = append(groupInputs, APIKeyGroupUpdateInput{
			GroupID:       input.GroupID,
			Quota:         input.Quota,
			ModelPatterns: append([]string(nil), input.ModelPatterns...),
		})
	}

	opCtx := ctx
	var tx *dbent.Tx
	if s.entClient != nil {
		tx, err = s.entClient.Tx(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin transaction: %w", err)
		}
		defer func() {
			_ = tx.Rollback()
		}()
		opCtx = dbent.NewTxContext(ctx, tx)
	}

	existingBindings, err := s.apiKeyRepo.GetAPIKeyGroups(opCtx, keyID)
	if err != nil {
		return nil, err
	}
	bindings, grantedGroups, err := buildAPIKeyGroupBindings(
		opCtx,
		apiKeyGroupBindingMutationDeps{
			groupRepo:   s.groupRepo,
			userRepo:    s.userRepo,
			userSubRepo: s.userSubRepo,
		},
		owner,
		apiKey.ID,
		existingBindings,
		groupInputs,
		true,
	)
	if err != nil {
		return nil, err
	}

	if err := s.apiKeyRepo.SetAPIKeyGroups(opCtx, keyID, bindings); err != nil {
		return nil, fmt.Errorf("set api key groups: %w", err)
	}
	apiKey.GroupBindings = append([]APIKeyGroupBinding(nil), bindings...)
	apiKey.SelectedGroupBinding = nil
	apiKey.SyncLegacyGroupShadow()
	if err := s.apiKeyRepo.Update(opCtx, apiKey); err != nil {
		return nil, fmt.Errorf("update api key metadata: %w", err)
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}
	}

	updatedAPIKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, updatedAPIKey.Key)
	}

	result := &AdminUpdateAPIKeyGroupsResult{
		APIKey:        updatedAPIKey,
		GrantedGroups: grantedGroups,
	}
	if len(grantedGroups) > 0 {
		result.AutoGrantedGroupAccess = true
		first := grantedGroups[0]
		result.GrantedGroupID = &first.GroupID
		result.GrantedGroupName = first.GroupName
	}
	return result, nil
}

func (s *adminServiceImpl) AdminUpdateAPIKeyGroupID(ctx context.Context, keyID int64, groupID *int64, modelDisplayMode *string) (*AdminUpdateAPIKeyGroupIDResult, error) {
	if groupID == nil {
		apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
		if err != nil {
			return nil, err
		}
		if modelDisplayMode != nil {
			apiKey.ModelDisplayMode = NormalizeAPIKeyModelDisplayMode(*modelDisplayMode)
			if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
				return nil, err
			}
			apiKey, err = s.apiKeyRepo.GetByID(ctx, keyID)
			if err != nil {
				return nil, err
			}
			if s.authCacheInvalidator != nil {
				s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, apiKey.Key)
			}
		}
		return &AdminUpdateAPIKeyGroupIDResult{APIKey: apiKey}, nil
	}
	if *groupID < 0 {
		return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be non-negative")
	}
	inputs := make([]AdminAPIKeyGroupUpdateInput, 0, 1)
	if *groupID > 0 {
		inputs = append(inputs, AdminAPIKeyGroupUpdateInput{GroupID: *groupID})
	}
	result, err := s.AdminUpdateAPIKeyGroups(ctx, keyID, inputs, modelDisplayMode)
	if err != nil {
		return nil, err
	}
	return &AdminUpdateAPIKeyGroupIDResult{
		APIKey:                 result.APIKey,
		AutoGrantedGroupAccess: result.AutoGrantedGroupAccess,
		GrantedGroupID:         result.GrantedGroupID,
		GrantedGroupName:       result.GrantedGroupName,
	}, nil
}
