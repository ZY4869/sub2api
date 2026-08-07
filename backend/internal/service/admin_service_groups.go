package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	DefaultImageBatchMaxItems            = 50
	DefaultImageBatchMaxDownloadBytes    = int64(100 * 1024 * 1024)
	DefaultImageBatchDownloadConcurrency = 2
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
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil || group == nil {
		return group, err
	}
	group.Platform = CanonicalizePlatformValue(group.Platform)
	if err := EnsureValidPrimaryGroupPlatform(group.Platform); err != nil {
		return nil, err
	}
	if group.Platform == PlatformComposite {
		routes, err := s.groupRepo.ListCompositeRoutes(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		group.CompositeRoutes = routes
	}
	return group, nil
}
func (s *adminServiceImpl) GetGroupByName(ctx context.Context, name string) (*Group, error) {
	group, err := s.groupRepo.GetByName(ctx, name)
	if err != nil || group == nil {
		return group, err
	}
	group.Platform = CanonicalizePlatformValue(group.Platform)
	if err := EnsureValidPrimaryGroupPlatform(group.Platform); err != nil {
		return nil, err
	}
	return group, nil
}
func (s *adminServiceImpl) CreateGroup(ctx context.Context, input *CreateGroupInput) (*Group, error) {
	platform := CanonicalizePlatformValue(input.Platform)
	if platform == "" {
		platform = PlatformAnthropic
	}
	if err := EnsureValidPrimaryGroupPlatform(platform); err != nil {
		return nil, err
	}
	if platform != PlatformComposite && len(input.CompositeRoutes) > 0 {
		return nil, ErrCompositeGroupRequired
	}
	normalizedCompositeRoutes := make([]CompositeModelRoute, 0)
	if platform == PlatformComposite {
		for i := range input.CompositeRoutes {
			input.CompositeRoutes[i].ParentGroupID = 0
		}
		normalizedRoutes, err := s.validateCompositeRouteInputs(ctx, 0, input.CompositeRoutes)
		if err != nil {
			return nil, err
		}
		normalizedCompositeRoutes = normalizedRoutes
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
	webSearchPricePerCall := normalizeWebSearchPrice(input.WebSearchPricePerCall)
	imageProtocolMode := NormalizeOpenAIGroupImageProtocolMode(input.ImageProtocolMode)
	if imageProtocolMode == "" {
		imageProtocolMode = OpenAIGroupImageProtocolModeInherit
	}
	if platform == PlatformAnthropic && input.FallbackGroupID != nil {
		if err := s.validateFallbackGroup(ctx, 0, *input.FallbackGroupID); err != nil {
			return nil, err
		}
	}
	fallbackOnInvalidRequest := input.FallbackGroupIDOnInvalidRequest
	if platform != PlatformAnthropic {
		fallbackOnInvalidRequest = nil
	}
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
	peakMultiplier := 1.0
	if input.PeakRateMultiplier != nil {
		peakMultiplier = *input.PeakRateMultiplier
	}
	peakEnabled, peakStart, peakEnd, peakRateMultiplier, err := NormalizeGroupPeakRateConfig(subscriptionType, input.PeakRateEnabled, input.PeakStart, input.PeakEnd, peakMultiplier)
	if err != nil {
		return nil, err
	}
	profitEnabled, profitMinMargin, profitSafetyBuffer := NormalizeGroupProfitConfig(input.ProfitControlEnabled, input.ProfitMinMargin, input.ProfitSafetyBuffer)
	group := &Group{Name: input.Name, Description: input.Description, Platform: platform, Priority: priority, RateMultiplier: input.RateMultiplier, ProfitControlEnabled: profitEnabled, ProfitMinMargin: profitMinMargin, ProfitSafetyBuffer: profitSafetyBuffer, PeakRateEnabled: peakEnabled, PeakStart: peakStart, PeakEnd: peakEnd, PeakRateMultiplier: peakRateMultiplier, IsExclusive: input.IsExclusive, Status: StatusActive, SubscriptionType: subscriptionType, DailyLimitUSD: dailyLimit, WeeklyLimitUSD: weeklyLimit, MonthlyLimitUSD: monthlyLimit, ImagePrice1K: imagePrice1K, ImagePrice2K: imagePrice2K, ImagePrice4K: imagePrice4K, WebSearchPricePerCall: webSearchPricePerCall, ImageProtocolMode: imageProtocolMode, ClaudeCodeOnly: input.ClaudeCodeOnly, FallbackGroupID: input.FallbackGroupID, FallbackGroupIDOnInvalidRequest: fallbackOnInvalidRequest, ModelRouting: input.ModelRouting, GeminiMixedProtocolEnabled: input.GeminiMixedProtocolEnabled, MCPXMLInject: mcpXMLInject, SupportedModelScopes: input.SupportedModelScopes, AllowMessagesDispatch: input.AllowMessagesDispatch, DefaultMappedModel: input.DefaultMappedModel, AllowLive: input.AllowLive, MaxReasoningEffort: NormalizeOpenAIReasoningEffortSetting(input.MaxReasoningEffort), ReasoningEffortMappings: NormalizeReasoningEffortMappings(input.ReasoningEffortMappings), VisibleModelPatterns: NormalizeGroupVisibleModelPatterns(input.VisibleModelPatterns), ImageBatchEnabled: input.ImageBatchEnabled, ImageBatchAllowedProviders: NormalizeImageBatchAllowList(input.ImageBatchAllowedProviders), ImageBatchAllowedModels: NormalizeImageBatchAllowList(input.ImageBatchAllowedModels), ImageBatchMaxItems: NormalizeImageBatchMaxItems(input.ImageBatchMaxItems), ImageBatchMaxDownloadBytes: NormalizeImageBatchMaxDownloadBytes(input.ImageBatchMaxDownloadBytes), ImageBatchDownloadConcurrency: NormalizeImageBatchDownloadConcurrency(input.ImageBatchDownloadConcurrency)}
	sanitizeGroupPlatformFields(group)
	if err := s.groupRepo.Create(ctx, group); err != nil {
		return nil, err
	}
	if len(accountIDsToCopy) > 0 {
		if err := s.groupRepo.BindAccountsToGroup(ctx, group.ID, accountIDsToCopy); err != nil {
			return nil, fmt.Errorf("failed to bind accounts to new group: %w", err)
		}
		group.AccountCount = int64(len(accountIDsToCopy))
	}
	if platform == PlatformComposite && len(normalizedCompositeRoutes) > 0 {
		routes := append([]CompositeModelRoute(nil), normalizedCompositeRoutes...)
		for i := range routes {
			routes[i].ParentGroupID = group.ID
		}
		updatedRoutes, err := s.groupRepo.ReplaceCompositeRoutes(ctx, group.ID, routes)
		if err != nil {
			return nil, err
		}
		group.CompositeRoutes = updatedRoutes
	}
	return group, nil
}

func (s *adminServiceImpl) DuplicateGroup(ctx context.Context, id int64, input *DuplicateGroupInput) (*Group, error) {
	source, err := s.GetGroup(ctx, id)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrGroupNotFound
	}
	copyAccounts := true
	name := ""
	if input != nil {
		copyAccounts = input.CopyAccounts
		name = strings.TrimSpace(input.Name)
	}
	if name == "" {
		name = s.nextDuplicateGroupName(ctx, source.Name)
	}
	var copyAccountSources []int64
	if copyAccounts {
		copyAccountSources = []int64{source.ID}
	}
	return s.CreateGroup(ctx, &CreateGroupInput{
		Name:                            name,
		Description:                     source.Description,
		Platform:                        source.Platform,
		Priority:                        source.Priority,
		RateMultiplier:                  source.RateMultiplier,
		ProfitControlEnabled:            source.ProfitControlEnabled,
		ProfitMinMargin:                 source.ProfitMinMargin,
		ProfitSafetyBuffer:              source.ProfitSafetyBuffer,
		PeakRateEnabled:                 source.PeakRateEnabled,
		PeakStart:                       source.PeakStart,
		PeakEnd:                         source.PeakEnd,
		PeakRateMultiplier:              &source.PeakRateMultiplier,
		IsExclusive:                     source.IsExclusive,
		SubscriptionType:                source.SubscriptionType,
		DailyLimitUSD:                   cloneFloat64Ptr(source.DailyLimitUSD),
		WeeklyLimitUSD:                  cloneFloat64Ptr(source.WeeklyLimitUSD),
		MonthlyLimitUSD:                 cloneFloat64Ptr(source.MonthlyLimitUSD),
		ImagePrice1K:                    cloneFloat64Ptr(source.ImagePrice1K),
		ImagePrice2K:                    cloneFloat64Ptr(source.ImagePrice2K),
		ImagePrice4K:                    cloneFloat64Ptr(source.ImagePrice4K),
		WebSearchPricePerCall:           cloneFloat64Ptr(source.WebSearchPricePerCall),
		ImageProtocolMode:               source.ImageProtocolMode,
		ClaudeCodeOnly:                  source.ClaudeCodeOnly,
		FallbackGroupID:                 cloneInt64Ptr(source.FallbackGroupID),
		FallbackGroupIDOnInvalidRequest: cloneInt64Ptr(source.FallbackGroupIDOnInvalidRequest),
		ModelRouting:                    cloneGroupModelRouting(source.ModelRouting),
		ModelRoutingEnabled:             source.ModelRoutingEnabled,
		GeminiMixedProtocolEnabled:      source.GeminiMixedProtocolEnabled,
		MCPXMLInject:                    &source.MCPXMLInject,
		SupportedModelScopes:            append([]string(nil), source.SupportedModelScopes...),
		AllowMessagesDispatch:           source.AllowMessagesDispatch,
		DefaultMappedModel:              source.DefaultMappedModel,
		AllowLive:                       source.AllowLive,
		MaxReasoningEffort:              source.MaxReasoningEffort,
		ReasoningEffortMappings:         append([]ReasoningEffortMapping(nil), source.ReasoningEffortMappings...),
		VisibleModelPatterns:            append([]string(nil), source.VisibleModelPatterns...),
		ImageBatchEnabled:               source.ImageBatchEnabled,
		ImageBatchAllowedProviders:      append([]string(nil), source.ImageBatchAllowedProviders...),
		ImageBatchAllowedModels:         append([]string(nil), source.ImageBatchAllowedModels...),
		ImageBatchMaxItems:              source.ImageBatchMaxItems,
		ImageBatchMaxDownloadBytes:      source.ImageBatchMaxDownloadBytes,
		ImageBatchDownloadConcurrency:   source.ImageBatchDownloadConcurrency,
		CopyAccountsFromGroupIDs:        copyAccountSources,
	})
}

func (s *adminServiceImpl) nextDuplicateGroupName(ctx context.Context, name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Group"
	}
	base += " 副本"
	for i := 0; i < 20; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s %d", base, i+1)
		}
		if _, err := s.groupRepo.GetByName(ctx, candidate); err != nil {
			return candidate
		}
	}
	return fmt.Sprintf("%s %s", base, time.Now().Format("20060102150405"))
}

func cloneGroupModelRouting(value map[string][]int64) map[string][]int64 {
	if len(value) == 0 {
		return nil
	}
	out := make(map[string][]int64, len(value))
	for key, ids := range value {
		out[key] = append([]int64(nil), ids...)
	}
	return out
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

func normalizeWebSearchPrice(price *float64) *float64 {
	if price == nil {
		return nil
	}
	return price
}

func NormalizeGroupProfitConfig(enabled bool, minMargin, safetyBuffer float64) (bool, float64, float64) {
	if !enabled {
		return false, 0, 0
	}
	return true, normalizeProfitRatio(minMargin), normalizeProfitRatio(safetyBuffer)
}

func normalizeProfitRatio(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func sanitizeGroupPlatformFields(group *Group) {
	if group == nil {
		return
	}
	group.Platform = CanonicalizePlatformValue(group.Platform)

	if group.Platform != PlatformAnthropic {
		group.ClaudeCodeOnly = false
		group.FallbackGroupID = nil
		group.FallbackGroupIDOnInvalidRequest = nil
		group.ModelRouting = nil
		group.ModelRoutingEnabled = false
	}

	if group.Platform != PlatformOpenAI {
		group.AllowMessagesDispatch = false
		group.DefaultMappedModel = ""
		group.ImageProtocolMode = OpenAIGroupImageProtocolModeInherit
	} else if NormalizeOpenAIGroupImageProtocolMode(group.ImageProtocolMode) == "" {
		group.ImageProtocolMode = OpenAIGroupImageProtocolModeInherit
	}

	if group.Platform != PlatformAntigravity {
		group.MCPXMLInject = true
		group.SupportedModelScopes = []string{}
	}

	if group.Platform != PlatformGemini {
		group.GeminiMixedProtocolEnabled = false
		group.ImageBatchEnabled = false
		group.ImageBatchAllowedProviders = nil
		group.ImageBatchAllowedModels = nil
		group.ImageBatchMaxItems = DefaultImageBatchMaxItems
		group.ImageBatchMaxDownloadBytes = DefaultImageBatchMaxDownloadBytes
		group.ImageBatchDownloadConcurrency = DefaultImageBatchDownloadConcurrency
	}

	if group.Platform != PlatformOpenAI && group.Platform != PlatformGrok {
		group.WebSearchPricePerCall = nil
	}

	if group.Platform != PlatformOpenAI && group.Platform != PlatformComposite {
		group.AllowLive = false
		group.MaxReasoningEffort = ""
		group.ReasoningEffortMappings = nil
	}

	if group.Platform != PlatformAntigravity && group.Platform != PlatformGemini && group.Platform != PlatformGrok {
		group.ImagePrice1K = nil
		group.ImagePrice2K = nil
		group.ImagePrice4K = nil
	}
}

func NormalizeImageBatchAllowList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func NormalizeImageBatchMaxItems(value int) int {
	if value <= 0 {
		return DefaultImageBatchMaxItems
	}
	if value > 1000 {
		return 1000
	}
	return value
}

func NormalizeImageBatchMaxDownloadBytes(value int64) int64 {
	if value <= 0 {
		return DefaultImageBatchMaxDownloadBytes
	}
	max := int64(1024 * 1024 * 1024)
	if value > max {
		return max
	}
	return value
}

func NormalizeImageBatchDownloadConcurrency(value int) int {
	if value <= 0 {
		return DefaultImageBatchDownloadConcurrency
	}
	if value > 16 {
		return 16
	}
	return value
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
	if platform != PlatformAnthropic {
		return fmt.Errorf("invalid request fallback only supported for anthropic groups")
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

func (s *adminServiceImpl) validateCompositeRouteInputs(ctx context.Context, parentGroupID int64, routes []CompositeModelRoute) ([]CompositeModelRoute, error) {
	normalized, err := NormalizeCompositeModelRoutes(parentGroupID, routes)
	if err != nil {
		return nil, err
	}
	for _, route := range normalized {
		target, err := s.groupRepo.GetByIDLite(ctx, route.TargetGroupID)
		if err != nil {
			return nil, fmt.Errorf("target group %d not found: %w", route.TargetGroupID, err)
		}
		if target == nil || CanonicalizePlatformValue(target.Platform) == PlatformComposite {
			return nil, ErrCompositeTargetGroupInvalid
		}
	}
	return normalized, nil
}

func (s *adminServiceImpl) UpdateGroup(ctx context.Context, id int64, input *UpdateGroupInput) (*Group, error) {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	group.Platform = CanonicalizePlatformValue(group.Platform)
	if err := EnsureValidPrimaryGroupPlatform(group.Platform); err != nil {
		return nil, err
	}
	if input.Name != "" {
		group.Name = input.Name
	}
	if input.DescriptionSet || input.Description != "" {
		group.Description = input.Description
	}
	if input.Platform != "" {
		group.Platform = CanonicalizePlatformValue(input.Platform)
		if err := EnsureValidPrimaryGroupPlatform(group.Platform); err != nil {
			return nil, err
		}
	}
	if input.Priority != nil && *input.Priority > 0 {
		group.Priority = *input.Priority
	}
	if input.RateMultiplier != nil {
		group.RateMultiplier = *input.RateMultiplier
	}
	if input.ProfitControlEnabled != nil {
		group.ProfitControlEnabled = *input.ProfitControlEnabled
	}
	if input.ProfitMinMargin != nil {
		group.ProfitMinMargin = *input.ProfitMinMargin
	}
	if input.ProfitSafetyBuffer != nil {
		group.ProfitSafetyBuffer = *input.ProfitSafetyBuffer
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
	if input.WebSearchPricePerCallSet {
		group.WebSearchPricePerCall = normalizeWebSearchPrice(input.WebSearchPricePerCall)
	}
	if input.ImageProtocolMode != "" {
		if normalized := NormalizeOpenAIGroupImageProtocolMode(input.ImageProtocolMode); normalized != "" {
			group.ImageProtocolMode = normalized
		}
	}
	if input.ClaudeCodeOnly != nil {
		group.ClaudeCodeOnly = *input.ClaudeCodeOnly
	}
	if group.Platform != PlatformAnthropic {
		group.FallbackGroupID = nil
	} else if input.FallbackGroupID != nil {
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
	if group.Platform != PlatformAnthropic {
		fallbackOnInvalidRequest = nil
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
	if input.AllowLive != nil {
		group.AllowLive = *input.AllowLive
	}
	if input.MaxReasoningEffort != nil {
		group.MaxReasoningEffort = NormalizeOpenAIReasoningEffortSetting(*input.MaxReasoningEffort)
	}
	if input.ReasoningEffortMappings != nil {
		group.ReasoningEffortMappings = NormalizeReasoningEffortMappings(*input.ReasoningEffortMappings)
	}
	if input.VisibleModelPatterns != nil {
		group.VisibleModelPatterns = NormalizeGroupVisibleModelPatterns(*input.VisibleModelPatterns)
	}
	if input.ImageBatchEnabled != nil {
		group.ImageBatchEnabled = *input.ImageBatchEnabled
	}
	if input.ImageBatchAllowedProviders != nil {
		group.ImageBatchAllowedProviders = NormalizeImageBatchAllowList(*input.ImageBatchAllowedProviders)
	}
	if input.ImageBatchAllowedModels != nil {
		group.ImageBatchAllowedModels = NormalizeImageBatchAllowList(*input.ImageBatchAllowedModels)
	}
	if input.ImageBatchMaxItems != nil {
		group.ImageBatchMaxItems = NormalizeImageBatchMaxItems(*input.ImageBatchMaxItems)
	}
	if input.ImageBatchMaxDownloadBytes != nil {
		group.ImageBatchMaxDownloadBytes = NormalizeImageBatchMaxDownloadBytes(*input.ImageBatchMaxDownloadBytes)
	}
	if input.ImageBatchDownloadConcurrency != nil {
		group.ImageBatchDownloadConcurrency = NormalizeImageBatchDownloadConcurrency(*input.ImageBatchDownloadConcurrency)
	}
	var normalizedCompositeRoutes []CompositeModelRoute
	compositeRoutesSet := input.CompositeRoutes != nil
	if compositeRoutesSet {
		if group.Platform != PlatformComposite && len(*input.CompositeRoutes) > 0 {
			return nil, ErrCompositeGroupRequired
		}
		if group.Platform == PlatformComposite {
			routes := append([]CompositeModelRoute(nil), (*input.CompositeRoutes)...)
			for i := range routes {
				routes[i].ParentGroupID = id
			}
			normalizedRoutes, err := s.validateCompositeRouteInputs(ctx, id, routes)
			if err != nil {
				return nil, err
			}
			normalizedCompositeRoutes = normalizedRoutes
		}
	}
	peakEnabled := group.PeakRateEnabled
	if input.PeakRateEnabled != nil {
		peakEnabled = *input.PeakRateEnabled
	}
	peakStart := group.PeakStart
	if input.PeakStart != nil {
		peakStart = *input.PeakStart
	}
	peakEnd := group.PeakEnd
	if input.PeakEnd != nil {
		peakEnd = *input.PeakEnd
	}
	peakMultiplier := group.PeakRateMultiplier
	if input.PeakRateMultiplier != nil {
		peakMultiplier = *input.PeakRateMultiplier
	}
	group.PeakRateEnabled, group.PeakStart, group.PeakEnd, group.PeakRateMultiplier, err = NormalizeGroupPeakRateConfig(group.SubscriptionType, peakEnabled, peakStart, peakEnd, peakMultiplier)
	if err != nil {
		return nil, err
	}
	group.ProfitControlEnabled, group.ProfitMinMargin, group.ProfitSafetyBuffer = NormalizeGroupProfitConfig(group.ProfitControlEnabled, group.ProfitMinMargin, group.ProfitSafetyBuffer)
	sanitizeGroupPlatformFields(group)
	if err := s.groupRepo.Update(ctx, group); err != nil {
		return nil, err
	}
	if compositeRoutesSet && group.Platform == PlatformComposite {
		updatedRoutes, err := s.groupRepo.ReplaceCompositeRoutes(ctx, id, normalizedCompositeRoutes)
		if err != nil {
			return nil, err
		}
		group.CompositeRoutes = updatedRoutes
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
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	group.Platform = CanonicalizePlatformValue(group.Platform)
	if err := EnsureValidPrimaryGroupPlatform(group.Platform); err != nil {
		return err
	}
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

func (s *adminServiceImpl) ListCompositeRoutes(ctx context.Context, groupID int64) ([]CompositeModelRoute, error) {
	group, err := s.groupRepo.GetByIDLite(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil || CanonicalizePlatformValue(group.Platform) != PlatformComposite {
		return nil, ErrCompositeGroupRequired
	}
	return s.groupRepo.ListCompositeRoutes(ctx, groupID)
}

func (s *adminServiceImpl) ReplaceCompositeRoutes(ctx context.Context, groupID int64, routes []CompositeModelRoute) ([]CompositeModelRoute, error) {
	group, err := s.groupRepo.GetByIDLite(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil || CanonicalizePlatformValue(group.Platform) != PlatformComposite {
		return nil, ErrCompositeGroupRequired
	}
	replaced, err := s.groupRepo.ReplaceCompositeRoutes(ctx, groupID, routes)
	if err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
	}
	return replaced, nil
}

func (s *adminServiceImpl) PreviewCompositeRoute(ctx context.Context, input CompositeRoutePreviewInput) (*CompositeRoutePreviewResult, error) {
	group, err := s.groupRepo.GetByIDLite(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil || CanonicalizePlatformValue(group.Platform) != PlatformComposite {
		return nil, ErrCompositeGroupRequired
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		return &CompositeRoutePreviewResult{Matched: false}, nil
	}
	route, err := s.groupRepo.FindCompositeRoute(ctx, input.GroupID, model)
	if err != nil {
		return nil, err
	}
	if route == nil {
		return &CompositeRoutePreviewResult{Matched: false, DisplayModelID: model}, nil
	}
	targetModel := strings.TrimSpace(route.TargetModelID)
	if targetModel == "" {
		targetModel = model
	}
	if route.TargetGroup == nil {
		route.TargetGroup, _ = s.groupRepo.GetByIDLite(ctx, route.TargetGroupID)
	}
	return &CompositeRoutePreviewResult{
		Matched:        true,
		DisplayModelID: model,
		TargetModelID:  targetModel,
		TargetGroupID:  route.TargetGroupID,
		TargetGroup:    route.TargetGroup,
		Route:          route,
	}, nil
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
