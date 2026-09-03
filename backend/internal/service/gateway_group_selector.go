package service

import (
	"context"
	"path"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrNoAvailableGroup         = infraerrors.ServiceUnavailable("NO_AVAILABLE_GROUP", "no available group for this request")
	ErrAPIKeyGroupQuotaExceeded = infraerrors.TooManyRequests("API_KEY_GROUP_QUOTA_EXCEEDED", "all matching api key groups are quota exhausted")
	ErrInvalidGroupBinding      = infraerrors.BadRequest("INVALID_GROUP_BINDING", "invalid api key group binding")
)

type groupBindingAvailabilityChecker func(ctx context.Context, binding *APIKeyGroupBinding) (bool, error)

type candidateBinding struct {
	binding         APIKeyGroupBinding
	explicit        bool
	priority        int
	requestPlatform string
}

func CloneAPIKeyWithSelectedGroup(apiKey *APIKey, binding *APIKeyGroupBinding) *APIKey {
	if apiKey == nil || binding == nil {
		return apiKey
	}
	cloned := *apiKey
	cloned.GroupBindings = append([]APIKeyGroupBinding(nil), apiKey.GroupBindings...)
	bindingCopy := *binding
	if binding.Group != nil {
		groupCopy := *binding.Group
		bindingCopy.Group = &groupCopy
	}
	cloned.SelectedGroupBinding = &bindingCopy
	cloned.SyncLegacyGroupShadow()
	return &cloned
}

func GroupsFromContext(ctx context.Context) []*Group {
	if ctx == nil {
		return nil
	}
	groups, _ := ctx.Value(ctxkey.Groups).([]*Group)
	return groups
}

func SelectGroupBindingForRequest(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	model string,
	excludedGroupIDs map[int64]struct{},
	checker groupBindingAvailabilityChecker,
) (*APIKeyGroupBinding, error) {
	return selectGroupBindingForRequest(ctx, apiKey, platform, model, excludedGroupIDs, checker)
}

func selectGroupBindingForRequest(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	model string,
	excludedGroupIDs map[int64]struct{},
	checker groupBindingAvailabilityChecker,
) (*APIKeyGroupBinding, error) {
	candidates := candidateGroupBindingsForRequest(ctx, apiKey, platform, model, excludedGroupIDs)
	if len(candidates) == 0 {
		return nil, ErrNoAvailableGroup
	}

	allQuotaExhausted := true
	for _, candidate := range candidates {
		if !candidate.binding.IsQuotaExhausted() {
			allQuotaExhausted = false
		}
		if candidate.binding.IsQuotaExhausted() {
			continue
		}
		if checker == nil {
			selected := candidate.binding
			return &selected, nil
		}
		checkCtx := contextWithCandidateRequestPlatform(ctx, candidate.requestPlatform)
		available, err := checker(checkCtx, &candidate.binding)
		if err != nil {
			return nil, err
		}
		if available {
			selected := candidate.binding
			return &selected, nil
		}
	}

	if allQuotaExhausted {
		return nil, ErrAPIKeyGroupQuotaExceeded
	}
	return nil, ErrNoAvailableGroup
}

func SelectGroupBindingForAllowedPlatforms(
	ctx context.Context,
	apiKey *APIKey,
	allowedPlatforms []string,
	model string,
	excludedGroupIDs map[int64]struct{},
	checker groupBindingAvailabilityChecker,
) (*APIKeyGroupBinding, error) {
	return selectGroupBindingForAllowedPlatforms(ctx, apiKey, allowedPlatforms, model, excludedGroupIDs, checker)
}

// SelectGroupForOpenAIEndpointCapability selects an OpenAI group for a
// protocol endpoint (for example alpha/search).  Endpoint capabilities are
// not models: direct OpenAI bindings therefore must not be filtered by the
// API-key or group model visibility patterns.  Composite bindings remain
// model-routed and use requestedModel to resolve their target group.
func (s *OpenAIGatewayService) SelectGroupForOpenAIEndpointCapability(
	ctx context.Context,
	apiKey *APIKey,
	allowedPlatforms []string,
	requestedModel string,
	capability OpenAIEndpointCapability,
	excludedGroupIDs map[int64]struct{},
) (*APIKeyGroupBinding, error) {
	if s == nil || apiKey == nil || capability == "" {
		return nil, ErrNoAvailableGroup
	}
	requestedModel = strings.TrimSpace(requestedModel)
	// Always overwrite the request model context, including the empty value,
	// so a stale model from another protocol cannot accidentally resolve a
	// composite route for this capability endpoint.
	ctx = context.WithValue(ctx, ctxkey.Model, requestedModel)
	candidates := candidateGroupBindingsForAllowedPlatformsWithOptions(
		ctx, apiKey, allowedPlatforms, requestedModel, excludedGroupIDs, true,
	)
	if len(candidates) == 0 {
		return nil, ErrNoAvailableGroup
	}

	allQuotaExhausted := true
	for _, candidate := range candidates {
		if !candidate.binding.IsQuotaExhausted() {
			allQuotaExhausted = false
		} else {
			continue
		}
		checkCtx := contextWithCandidateRequestPlatform(ctx, candidate.requestPlatform)
		available, err := s.groupBindingHasSchedulableAccountsForCapability(checkCtx, &candidate.binding, capability)
		if err != nil {
			return nil, err
		}
		if available {
			selected := candidate.binding
			return &selected, nil
		}
	}
	if allQuotaExhausted {
		return nil, ErrAPIKeyGroupQuotaExceeded
	}
	return nil, ErrNoAvailableGroup
}

func selectGroupBindingForAllowedPlatforms(
	ctx context.Context,
	apiKey *APIKey,
	allowedPlatforms []string,
	model string,
	excludedGroupIDs map[int64]struct{},
	checker groupBindingAvailabilityChecker,
) (*APIKeyGroupBinding, error) {
	candidates := candidateGroupBindingsForAllowedPlatforms(ctx, apiKey, allowedPlatforms, model, excludedGroupIDs)
	if len(candidates) == 0 {
		return nil, ErrNoAvailableGroup
	}

	allQuotaExhausted := true
	for _, candidate := range candidates {
		if !candidate.binding.IsQuotaExhausted() {
			allQuotaExhausted = false
		}
		if candidate.binding.IsQuotaExhausted() {
			continue
		}
		if checker == nil {
			selected := candidate.binding
			return &selected, nil
		}
		checkCtx := contextWithCandidateRequestPlatform(ctx, candidate.requestPlatform)
		available, err := checker(checkCtx, &candidate.binding)
		if err != nil {
			return nil, err
		}
		if available {
			selected := candidate.binding
			return &selected, nil
		}
	}

	if allQuotaExhausted {
		return nil, ErrAPIKeyGroupQuotaExceeded
	}
	return nil, ErrNoAvailableGroup
}

func candidateGroupBindingsForRequest(ctx context.Context, apiKey *APIKey, platform string, model string, excludedGroupIDs map[int64]struct{}) []candidateBinding {
	if strings.TrimSpace(platform) == "" {
		return candidateGroupBindingsForAllowedPlatforms(ctx, apiKey, nil, model, excludedGroupIDs)
	}
	return candidateGroupBindingsForAllowedPlatforms(ctx, apiKey, []string{platform}, model, excludedGroupIDs)
}

func candidateGroupBindingsForAllowedPlatforms(ctx context.Context, apiKey *APIKey, allowedPlatforms []string, model string, excludedGroupIDs map[int64]struct{}) []candidateBinding {
	return candidateGroupBindingsForAllowedPlatformsWithOptions(ctx, apiKey, allowedPlatforms, model, excludedGroupIDs, false)
}

func candidateGroupBindingsForAllowedPlatformsWithOptions(ctx context.Context, apiKey *APIKey, allowedPlatforms []string, model string, excludedGroupIDs map[int64]struct{}, ignoreDirectOpenAIModelFilters bool) []candidateBinding {
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(allowedPlatforms))
	for _, platform := range allowedPlatforms {
		normalized := strings.TrimSpace(strings.ToLower(platform))
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	candidates := make([]candidateBinding, 0, len(bindings))
	for _, binding := range bindings {
		if excludedGroupIDs != nil {
			if _, excluded := excludedGroupIDs[binding.GroupID]; excluded {
				continue
			}
		}
		group := binding.Group
		if group == nil || !group.IsActive() {
			continue
		}
		if !apiKey.UserCanAccessGroup(group) {
			recordGroupAccessDenied(ctx, apiKey, group)
			continue
		}
		requestPlatform := ""
		if len(allowed) > 0 {
			matchedPlatform, ok := bindingRequestPlatformForAllowed(group.Platform, allowed)
			if !ok {
				continue
			}
			requestPlatform = matchedPlatform
		}
		if len(allowed) == 0 && len(allowedPlatforms) > 0 {
			continue
		}

		canonicalPlatform := CanonicalizePlatformValue(group.Platform)
		isOpenAIProtocolBinding := ignoreDirectOpenAIModelFilters &&
			(canonicalPlatform == PlatformOpenAI || canonicalPlatform == PlatformComposite ||
				CanonicalizePlatformValue(requestPlatform) == PlatformOpenAI)
		if isOpenAIProtocolBinding {
			// alpha/search is a protocol capability, not a model binding. Composite
			// groups still need the request model solely to resolve their route.
			if canonicalPlatform == PlatformComposite && strings.TrimSpace(model) == "" {
				continue
			}
		} else {
			_, matched := bindingMatchesModel(binding.ModelPatterns, model)
			if !matched {
				continue
			}
			if !groupAllowsVisibleRequestModel(group, model, ctx) {
				continue
			}
		}
		explicit := false
		if !isOpenAIProtocolBinding {
			explicit, _ = bindingMatchesModel(binding.ModelPatterns, model)
		}

		priority := group.Priority
		if priority <= 0 {
			priority = 1
		}
		candidateGroupBinding := binding
		if requestPlatform != "" {
			candidateGroupBinding = groupBindingForRequestPlatform(binding, requestPlatform)
		}
		candidates = append(candidates, candidateBinding{
			binding:         candidateGroupBinding,
			explicit:        explicit,
			priority:        priority,
			requestPlatform: requestPlatform,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].explicit != candidates[j].explicit {
			return candidates[i].explicit
		}
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].binding.GroupID < candidates[j].binding.GroupID
	})
	return candidates
}

func groupBindingForRequestPlatform(binding APIKeyGroupBinding, requestPlatform string) APIKeyGroupBinding {
	requestPlatform = strings.TrimSpace(strings.ToLower(requestPlatform))
	if requestPlatform == "" || binding.Group == nil || binding.Group.Platform != PlatformProtocolGateway {
		return binding
	}
	groupCopy := *binding.Group
	groupCopy.Platform = requestPlatform
	binding.Group = &groupCopy
	return binding
}

func recordGroupAccessDenied(ctx context.Context, apiKey *APIKey, group *Group) {
	if apiKey == nil || group == nil {
		return
	}
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	fields := []zap.Field{
		zap.String("request_id", strings.TrimSpace(requestID)),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Int64("group_id", group.ID),
		zap.String("group_name", group.Name),
		zap.Bool("group_exclusive", group.IsExclusive),
	}
	if apiKey.User != nil {
		fields = append(fields, zap.Int64("user_id", apiKey.User.ID))
	}
	logger.FromContext(ctx).Warn("api key group access denied", fields...)
}

func bindingRequestPlatformForAllowed(groupPlatform string, allowed map[string]struct{}) (string, bool) {
	groupPlatform = strings.TrimSpace(strings.ToLower(groupPlatform))
	if groupPlatform == "" || len(allowed) == 0 {
		return "", false
	}
	if _, ok := allowed[groupPlatform]; ok {
		return "", true
	}
	if groupPlatform != PlatformProtocolGateway {
		return "", false
	}
	for _, platform := range []string{PlatformOpenAI, PlatformAnthropic, PlatformGemini} {
		if _, ok := allowed[platform]; ok {
			return platform, true
		}
	}
	return "", false
}

func contextWithCandidateRequestPlatform(ctx context.Context, platform string) context.Context {
	platform = strings.TrimSpace(strings.ToLower(platform))
	if platform == "" || platform == PlatformProtocolGateway {
		return ctx
	}
	if _, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string); hasForcePlatform {
		return ctx
	}
	return context.WithValue(ctx, ctxkey.ForcePlatform, platform)
}

func apiKeyBindingsForSelection(apiKey *APIKey) []APIKeyGroupBinding {
	if apiKey == nil {
		return nil
	}
	if len(apiKey.GroupBindings) > 0 {
		return append([]APIKeyGroupBinding(nil), apiKey.GroupBindings...)
	}
	if apiKey.GroupID != nil && apiKey.Group != nil {
		return []APIKeyGroupBinding{{
			APIKeyID: apiKey.ID,
			GroupID:  *apiKey.GroupID,
			Group:    apiKey.Group,
		}}
	}
	return nil
}

func APIKeyBindingsForSelection(apiKey *APIKey) []APIKeyGroupBinding {
	return apiKeyBindingsForSelection(apiKey)
}

func groupAllowsVisibleRequestModel(group *Group, model string, ctx context.Context) bool {
	if group == nil || !group.HasVisibleModelPatternFilter() {
		return true
	}
	candidates := []string{model}
	candidates = append(candidates, VisibleModelCandidatesFromContext(ctx)...)
	for _, candidate := range visibleModelCandidates("", candidates...) {
		if group.AllowsVisibleModel(candidate) {
			return true
		}
	}
	return false
}

func bindingMatchesModel(patterns []string, model string) (explicit bool, matched bool) {
	if len(patterns) == 0 || strings.TrimSpace(model) == "" {
		return false, true
	}

	candidates := []string{strings.TrimSpace(model)}
	candidates = append(candidates, grokModelMatchCandidates(model)...)
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		for _, candidate := range dedupeStrings(candidates) {
			if ok, err := path.Match(trimmed, candidate); err == nil && ok {
				return true, true
			}
			if matchModelPattern(trimmed, candidate) {
				return true, true
			}
		}
	}
	return true, false
}

func (s *GatewayService) SelectGroupForRequest(ctx context.Context, apiKey *APIKey, platform string, model string, excludedGroupIDs map[int64]struct{}) (*APIKeyGroupBinding, error) {
	if strings.TrimSpace(model) != "" {
		ctx = context.WithValue(ctx, ctxkey.Model, model)
	}
	return selectGroupBindingForRequest(ctx, apiKey, platform, model, excludedGroupIDs, s.groupBindingHasSchedulableAccounts)
}

func (s *GatewayService) SelectGroupForAllowedPlatforms(ctx context.Context, apiKey *APIKey, allowedPlatforms []string, model string, excludedGroupIDs map[int64]struct{}) (*APIKeyGroupBinding, error) {
	if strings.TrimSpace(model) != "" {
		ctx = context.WithValue(ctx, ctxkey.Model, model)
	}
	return selectGroupBindingForAllowedPlatforms(ctx, apiKey, allowedPlatforms, model, excludedGroupIDs, s.groupBindingHasSchedulableAccounts)
}

func (s *GatewayService) groupBindingHasSchedulableAccounts(ctx context.Context, binding *APIKeyGroupBinding) (bool, error) {
	if s == nil || binding == nil || binding.Group == nil {
		return false, nil
	}
	ctx = s.withGroupContext(ctx, binding.Group)
	platform := binding.Group.Platform
	forcePlatform, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string)
	if hasForcePlatform && forcePlatform != "" {
		platform = forcePlatform
	}
	accounts, useMixed, err := s.listSchedulableAccounts(ctx, &binding.GroupID, platform, hasForcePlatform && forcePlatform != "")
	if err != nil {
		return false, err
	}
	requestedModel := strings.TrimSpace(modelFromContext(ctx))
	for i := range accounts {
		account := &accounts[i]
		if !s.isAccountSchedulableForSelection(account) {
			continue
		}
		if !s.isAccountAllowedForPlatformWithContext(ctx, account, platform, useMixed) {
			continue
		}
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
			continue
		}
		if !s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) {
			continue
		}
		if !s.isAccountSchedulableForQuota(account) {
			continue
		}
		if !s.isAccountSchedulableForWindowCost(ctx, account, false) {
			continue
		}
		if !s.isAccountSchedulableForRPM(ctx, account, false) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *OpenAIGatewayService) SelectGroupForRequest(ctx context.Context, apiKey *APIKey, platform string, model string, excludedGroupIDs map[int64]struct{}) (*APIKeyGroupBinding, error) {
	if strings.TrimSpace(model) != "" {
		ctx = context.WithValue(ctx, ctxkey.Model, model)
	}
	return SelectGroupBindingForRequest(ctx, apiKey, platform, model, excludedGroupIDs, s.groupBindingHasSchedulableAccounts)
}

func (s *OpenAIGatewayService) SelectGroupForAllowedPlatforms(ctx context.Context, apiKey *APIKey, allowedPlatforms []string, model string, excludedGroupIDs map[int64]struct{}) (*APIKeyGroupBinding, error) {
	if strings.TrimSpace(model) != "" {
		ctx = context.WithValue(ctx, ctxkey.Model, model)
	}
	return SelectGroupBindingForAllowedPlatforms(ctx, apiKey, allowedPlatforms, model, excludedGroupIDs, s.groupBindingHasSchedulableAccounts)
}

func (s *OpenAIGatewayService) groupBindingHasSchedulableAccounts(ctx context.Context, binding *APIKeyGroupBinding) (bool, error) {
	if s == nil || binding == nil || binding.Group == nil {
		return false, nil
	}
	group := binding.Group
	requestedModel := strings.TrimSpace(modelFromContext(ctx))
	if CanonicalizePlatformValue(group.Platform) == PlatformComposite {
		if requestedModel == "" || s.groupRepo == nil {
			return false, nil
		}
		route, err := s.groupRepo.FindCompositeRoute(ctx, group.ID, requestedModel)
		if err != nil || route == nil || route.TargetGroupID <= 0 {
			return false, err
		}
		targetGroup := route.TargetGroup
		if targetGroup == nil || targetGroup.ID <= 0 || !targetGroup.Hydrated {
			targetGroup, err = s.groupRepo.GetByIDLite(ctx, route.TargetGroupID)
			if err != nil {
				return false, err
			}
		}
		if targetGroup == nil || CanonicalizePlatformValue(targetGroup.Platform) == PlatformComposite {
			return false, nil
		}
		targetModel := strings.TrimSpace(route.TargetModelID)
		if targetModel == "" {
			targetModel = requestedModel
		}
		ctx = context.WithValue(ctx, ctxkey.Model, targetModel)
		ctx = WithOpenAIPlatform(ctx, targetGroup.Platform)
		return s.groupBindingHasSchedulableAccounts(ctx, &APIKeyGroupBinding{
			APIKeyID: binding.APIKeyID,
			GroupID:  targetGroup.ID,
			Group:    targetGroup,
		})
	}
	accounts, err := s.listSchedulableAccounts(ctx, &binding.GroupID)
	if err != nil {
		return false, err
	}
	for i := range accounts {
		account := &accounts[i]
		if !account.IsSchedulable() || !isOpenAITextRuntimeAccount(account) {
			continue
		}
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
			continue
		}
		return true, nil
	}
	return false, nil
}

// groupBindingHasSchedulableAccountsForCapability checks group availability
// using an endpoint capability instead of a requested model. This is used by
// protocol endpoints such as alpha/search, where the request model is only
// meaningful for resolving composite routes and for upstream translation.
func (s *OpenAIGatewayService) groupBindingHasSchedulableAccountsForCapability(ctx context.Context, binding *APIKeyGroupBinding, capability OpenAIEndpointCapability) (bool, error) {
	if s == nil || binding == nil || binding.Group == nil || capability == "" {
		return false, nil
	}
	group := binding.Group
	forcePlatform, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string)
	effectivePlatform := CanonicalizePlatformValue(group.Platform)
	if hasForcePlatform && strings.TrimSpace(forcePlatform) != "" {
		effectivePlatform = CanonicalizePlatformValue(forcePlatform)
	}
	if CanonicalizePlatformValue(group.Platform) == PlatformComposite {
		requestedModel := strings.TrimSpace(modelFromContext(ctx))
		if requestedModel == "" || s.groupRepo == nil {
			return false, nil
		}
		route, err := s.groupRepo.FindCompositeRoute(ctx, group.ID, requestedModel)
		if err != nil {
			return false, err
		}
		if route == nil || route.TargetGroupID <= 0 {
			return false, nil
		}
		targetGroup := route.TargetGroup
		if targetGroup == nil || targetGroup.ID <= 0 || !targetGroup.Hydrated {
			targetGroup, err = s.groupRepo.GetByIDLite(ctx, route.TargetGroupID)
			if err != nil {
				return false, err
			}
		}
		if targetGroup == nil || CanonicalizePlatformValue(targetGroup.Platform) != PlatformOpenAI {
			return false, nil
		}
		ctx = WithOpenAIPlatform(ctx, targetGroup.Platform)
		return s.openAIGroupHasSchedulableCapability(ctx, targetGroup.ID, capability)
	}
	if effectivePlatform != PlatformOpenAI {
		return false, nil
	}
	ctx = WithOpenAIPlatform(ctx, effectivePlatform)
	return s.openAIGroupHasSchedulableCapability(ctx, group.ID, capability)
}

func (s *OpenAIGatewayService) openAIGroupHasSchedulableCapability(ctx context.Context, groupID int64, capability OpenAIEndpointCapability) (bool, error) {
	if s == nil || groupID <= 0 {
		return false, nil
	}
	accounts, err := s.listSchedulableAccounts(ctx, &groupID)
	if err != nil {
		return false, err
	}
	for i := range accounts {
		account := &accounts[i]
		if !account.IsSchedulable() || !isOpenAITextRuntimeAccount(account) {
			continue
		}
		if CanParticipateInAccountQuota(account) && account.IsQuotaExceeded() {
			continue
		}
		if !account.IsSchedulableForModelWithContext(ctx, "") {
			continue
		}
		if !SupportsOpenAIEndpointCapability(account, capability) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func modelFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	model, _ := ctx.Value(ctxkey.Model).(string)
	return model
}

func (s *GatewayService) GetActiveSubscriptionForGroup(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, userID, groupID)
}

func (s *OpenAIGatewayService) GetActiveSubscriptionForGroup(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	if s == nil || s.userSubRepo == nil {
		return nil, ErrSubscriptionNotFound
	}
	return s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, userID, groupID)
}
