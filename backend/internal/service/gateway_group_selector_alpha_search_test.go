package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type alphaSearchGroupAccountRepo struct {
	stubOpenAIAccountRepo
	byGroup map[int64][]Account
}

func (r *alphaSearchGroupAccountRepo) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, _ []string) ([]Account, error) {
	return append([]Account(nil), r.byGroup[groupID]...), nil
}

type alphaSearchCompositeGroupRepo struct {
	groupRepoNoop
	route  *CompositeModelRoute
	groups map[int64]*Group
}

func (r *alphaSearchCompositeGroupRepo) FindCompositeRoute(_ context.Context, parentGroupID int64, displayModelID string) (*CompositeModelRoute, error) {
	if r.route == nil || r.route.ParentGroupID != parentGroupID || !matchModelPattern(r.route.DisplayModelID, displayModelID) {
		return nil, nil
	}
	copyRoute := *r.route
	return &copyRoute, nil
}

func (r *alphaSearchCompositeGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if group := r.groups[id]; group != nil {
		copyGroup := *group
		copyGroup.Hydrated = true
		return &copyGroup, nil
	}
	return nil, ErrGroupNotFound
}

func alphaSearchTestGroup(id int64, platform string, patterns []string) *Group {
	return &Group{ID: id, Platform: platform, Status: StatusActive, Hydrated: true, VisibleModelPatterns: patterns}
}

func alphaSearchTestAPIKey(bindings ...APIKeyGroupBinding) *APIKey {
	return &APIKey{ID: 1, UserID: 1, Status: StatusActive, GroupBindings: bindings}
}

func TestOpenAIGatewayServiceSelectGroupForAlphaSearchIgnoresModelFilters(t *testing.T) {
	groupID := int64(6101)
	group := alphaSearchTestGroup(groupID, PlatformOpenAI, []string{"gpt-*"})
	account := openAICapabilityTestAccount(61011, nil, 1)
	repo := &alphaSearchGroupAccountRepo{byGroup: map[int64][]Account{groupID: {account}}}
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{RunMode: config.RunModeStandard}}
	apiKey := alphaSearchTestAPIKey(APIKeyGroupBinding{APIKeyID: 1, GroupID: groupID, Group: group, ModelPatterns: []string{"gpt-*"}})

	selected, err := svc.SelectGroupForOpenAIEndpointCapability(context.Background(), apiKey, []string{PlatformOpenAI, PlatformComposite}, "not-in-pattern", OpenAIEndpointCapabilityAlphaSearch, nil)

	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, groupID, selected.GroupID)
}

func TestOpenAIGatewayServiceSelectGroupForAlphaSearchSkipsUnavailableGroup(t *testing.T) {
	firstID, secondID := int64(6201), int64(6202)
	first := alphaSearchTestGroup(firstID, PlatformOpenAI, nil)
	second := alphaSearchTestGroup(secondID, PlatformOpenAI, nil)
	disabled := openAICapabilityTestAccount(62011, nil, 1)
	disabled.Extra = map[string]any{"openai_alpha_search_enabled": false}
	supported := openAICapabilityTestAccount(62021, nil, 2)
	repo := &alphaSearchGroupAccountRepo{byGroup: map[int64][]Account{firstID: {disabled}, secondID: {supported}}}
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{RunMode: config.RunModeStandard}}
	apiKey := alphaSearchTestAPIKey(
		APIKeyGroupBinding{APIKeyID: 1, GroupID: firstID, Group: first},
		APIKeyGroupBinding{APIKeyID: 1, GroupID: secondID, Group: second},
	)

	selected, err := svc.SelectGroupForOpenAIEndpointCapability(context.Background(), apiKey, []string{PlatformOpenAI}, "", OpenAIEndpointCapabilityAlphaSearch, nil)

	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, secondID, selected.GroupID)
}

func TestOpenAIGatewayServiceSelectGroupForAlphaSearchCompositeUsesRequestModel(t *testing.T) {
	parentID, targetID := int64(6301), int64(6302)
	parent := alphaSearchTestGroup(parentID, PlatformComposite, nil)
	target := alphaSearchTestGroup(targetID, PlatformOpenAI, nil)
	account := openAICapabilityTestAccount(63021, nil, 1)
	repo := &alphaSearchGroupAccountRepo{byGroup: map[int64][]Account{targetID: {account}}}
	groupRepo := &alphaSearchCompositeGroupRepo{
		route:  &CompositeModelRoute{ParentGroupID: parentID, DisplayModelID: "search-public", TargetGroupID: targetID, Enabled: true},
		groups: map[int64]*Group{targetID: target},
	}
	svc := &OpenAIGatewayService{accountRepo: repo, groupRepo: groupRepo, cfg: &config.Config{RunMode: config.RunModeStandard}}
	apiKey := alphaSearchTestAPIKey(APIKeyGroupBinding{APIKeyID: 1, GroupID: parentID, Group: parent})

	selected, err := svc.SelectGroupForOpenAIEndpointCapability(context.Background(), apiKey, []string{PlatformOpenAI, PlatformComposite}, "search-public", OpenAIEndpointCapabilityAlphaSearch, nil)

	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, parentID, selected.GroupID)
}

func TestOpenAIGatewayServiceSelectGroupForAlphaSearchCompositeWithoutModelSkips(t *testing.T) {
	parentID := int64(6401)
	parent := alphaSearchTestGroup(parentID, PlatformComposite, nil)
	svc := &OpenAIGatewayService{accountRepo: &alphaSearchGroupAccountRepo{}, cfg: &config.Config{RunMode: config.RunModeStandard}}
	apiKey := alphaSearchTestAPIKey(APIKeyGroupBinding{APIKeyID: 1, GroupID: parentID, Group: parent})

	selected, err := svc.SelectGroupForOpenAIEndpointCapability(context.Background(), apiKey, []string{PlatformOpenAI, PlatformComposite}, "", OpenAIEndpointCapabilityAlphaSearch, nil)

	require.ErrorIs(t, err, ErrNoAvailableGroup)
	require.Nil(t, selected)
}
