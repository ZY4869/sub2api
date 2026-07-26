package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type openAICompositeRuntimeGroupRepoStub struct {
	groupRepoNoop

	route     *CompositeModelRoute
	groups    map[int64]*Group
	findCalls int
	getCalls  int
}

func (s *openAICompositeRuntimeGroupRepoStub) FindCompositeRoute(_ context.Context, parentGroupID int64, displayModelID string) (*CompositeModelRoute, error) {
	s.findCalls++
	if s.route == nil || s.route.ParentGroupID != parentGroupID || !matchModelPattern(s.route.DisplayModelID, displayModelID) {
		return nil, nil
	}
	routeCopy := *s.route
	return &routeCopy, nil
}

func (s *openAICompositeRuntimeGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	s.getCalls++
	if group := s.groups[id]; group != nil {
		copyGroup := *group
		copyGroup.Hydrated = true
		return &copyGroup, nil
	}
	return nil, ErrGroupNotFound
}

func TestOpenAIGatewayServiceResolveCompositeRouteRuntime_SelectsTargetGroupAndModel(t *testing.T) {
	parentGroupID := int64(10)
	targetGroupID := int64(20)
	repo := &openAICompositeRuntimeGroupRepoStub{
		route: &CompositeModelRoute{
			ID:             1,
			ParentGroupID:  parentGroupID,
			DisplayModelID: "public-gpt",
			TargetGroupID:  targetGroupID,
			TargetModelID:  "gpt-5.4",
			Enabled:        true,
		},
		groups: map[int64]*Group{
			targetGroupID: {ID: targetGroupID, Name: "openai-target", Platform: PlatformOpenAI, Status: StatusActive},
		},
	}
	svc := &OpenAIGatewayService{groupRepo: repo}
	apiKey := &APIKey{
		ID:      101,
		UserID:  201,
		GroupID: &parentGroupID,
		Group:   &Group{ID: parentGroupID, Name: "parent", Platform: PlatformComposite, Status: StatusActive, Hydrated: true},
		GroupBindings: []APIKeyGroupBinding{
			{APIKeyID: 101, GroupID: parentGroupID, Group: &Group{ID: parentGroupID, Platform: PlatformComposite, Status: StatusActive, Hydrated: true}},
			{APIKeyID: 101, GroupID: targetGroupID, Group: &Group{ID: targetGroupID, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}, Quota: 50},
		},
	}

	got, err := svc.ResolveCompositeRouteRuntime(context.Background(), apiKey, nil, "public-gpt")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.True(t, got.Matched)
	require.Equal(t, "public-gpt", got.DisplayModelID)
	require.Equal(t, "gpt-5.4", got.RuntimeModelID)
	require.Equal(t, targetGroupID, got.TargetGroupID)
	require.NotSame(t, apiKey, got.APIKey)
	require.NotNil(t, got.APIKey.GroupID)
	require.Equal(t, targetGroupID, *got.APIKey.GroupID)
	require.NotNil(t, got.APIKey.Group)
	require.Equal(t, targetGroupID, got.APIKey.Group.ID)
	require.NotNil(t, got.APIKey.SelectedGroupBinding)
	require.Equal(t, targetGroupID, got.APIKey.SelectedGroupBinding.GroupID)
	require.Equal(t, 50.0, got.APIKey.SelectedGroupBinding.Quota)
	require.Equal(t, 1, repo.findCalls)
	require.Equal(t, 1, repo.getCalls)
}

func TestOpenAIGatewayServiceResolveCompositeRouteRuntime_RejectsCompositeTargetGroup(t *testing.T) {
	parentGroupID := int64(10)
	targetGroupID := int64(20)
	repo := &openAICompositeRuntimeGroupRepoStub{
		route: &CompositeModelRoute{
			ID:             1,
			ParentGroupID:  parentGroupID,
			DisplayModelID: "public-gpt",
			TargetGroupID:  targetGroupID,
			Enabled:        true,
		},
		groups: map[int64]*Group{
			targetGroupID: {ID: targetGroupID, Name: "bad-target", Platform: PlatformComposite, Status: StatusActive},
		},
	}
	svc := &OpenAIGatewayService{groupRepo: repo}

	_, err := svc.ResolveCompositeRouteRuntime(context.Background(), &APIKey{
		ID:      101,
		GroupID: &parentGroupID,
		Group:   &Group{ID: parentGroupID, Platform: PlatformComposite, Status: StatusActive, Hydrated: true},
	}, nil, "public-gpt")

	require.ErrorIs(t, err, ErrCompositeTargetGroupInvalid)
}

func TestOpenAIGatewayServiceResolveCompositeRouteRuntime_NonCompositeGroupPassesThrough(t *testing.T) {
	groupID := int64(20)
	repo := &openAICompositeRuntimeGroupRepoStub{}
	svc := &OpenAIGatewayService{groupRepo: repo}
	apiKey := &APIKey{
		ID:      101,
		GroupID: &groupID,
		Group:   &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true},
	}

	got, err := svc.ResolveCompositeRouteRuntime(context.Background(), apiKey, nil, "gpt-5.4")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.False(t, got.Matched)
	require.Same(t, apiKey, got.APIKey)
	require.Equal(t, "gpt-5.4", got.RuntimeModelID)
	require.Equal(t, 0, repo.findCalls)
}
