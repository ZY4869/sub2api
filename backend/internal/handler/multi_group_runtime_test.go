package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type multiGroupRuntimeSettingRepoStub struct {
	value string
}

func int64PtrForTest(v int64) *int64 {
	return &v
}

func newTestBillingCacheService() *service.BillingCacheService {
	cfg := &config.Config{}
	cfg.RunMode = config.RunModeSimple
	return service.NewBillingCacheService(nil, nil, nil, nil, cfg)
}

func (s multiGroupRuntimeSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected call")
}

func TestResolveSelectedAPIKey_PublicCatalogEntryPinsMatchedGroup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		call func(*gin.Context, *service.SettingService, *service.APIKey, *service.BillingCacheService) (*service.APIKey, *service.UserSubscription, error)
	}{
		{
			name: "gateway",
			call: func(ctx *gin.Context, settingService *service.SettingService, apiKey *service.APIKey, billingCacheService *service.BillingCacheService) (*service.APIKey, *service.UserSubscription, error) {
				return resolveSelectedGatewayAPIKey(
					ctx,
					settingService,
					service.NewGatewayService(nil, nil, nil, nil, nil, nil, nil, nil, &config.Config{}, nil, nil, nil, nil, billingCacheService, nil, nil, nil, nil, nil, nil, nil, settingService),
					billingCacheService,
					apiKey,
					nil,
					"gpt-5.4@team-b",
					gatewayCompatiblePlatforms,
					nil,
				)
			},
		},
		{
			name: "openai",
			call: func(ctx *gin.Context, settingService *service.SettingService, apiKey *service.APIKey, billingCacheService *service.BillingCacheService) (*service.APIKey, *service.UserSubscription, error) {
				return resolveSelectedOpenAIAPIKey(
					ctx,
					settingService,
					service.NewOpenAIGatewayService(nil, nil, nil, nil, nil, nil, nil, &config.Config{}, nil, nil, nil, nil, billingCacheService, nil, nil, nil, settingService),
					billingCacheService,
					apiKey,
					nil,
					"gpt-5.4@team-b",
					openAICompatiblePlatforms,
					nil,
				)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
			req = req.WithContext(service.AttachPublishedPublicCatalogEntry(req.Context(), &service.PublishedPublicCatalogEntry{
				EntryID:         "entry-team-b",
				PublicModelID:   "gpt-5.4@team-b",
				SourceModelID:   "gpt-5.4",
				SourceAccountID: 202,
				BindingGroupID:  20,
			}))
			ctx.Request = req

			apiKey := &service.APIKey{
				ID:     7,
				UserID: 9,
				User:   &service.User{ID: 9},
				GroupBindings: []service.APIKeyGroupBinding{
					{
						GroupID:       10,
						ModelPatterns: []string{"gpt-5.4@team-a"},
						Group:         &service.Group{ID: 10, Platform: service.PlatformOpenAI, Status: service.StatusActive},
					},
					{
						GroupID:       20,
						ModelPatterns: []string{"gpt-5.4@team-b"},
						Group:         &service.Group{ID: 20, Platform: service.PlatformOpenAI, Status: service.StatusActive},
					},
				},
			}
			settingService := service.NewSettingService(multiGroupRuntimeSettingRepoStub{value: "true"}, nil)
			billingCacheService := newTestBillingCacheService()
			t.Cleanup(billingCacheService.Stop)

			selectedAPIKey, subscription, err := tc.call(ctx, settingService, apiKey, billingCacheService)

			require.NoError(t, err)
			require.Nil(t, subscription)
			require.NotNil(t, selectedAPIKey)
			require.NotNil(t, selectedAPIKey.GroupID)
			require.Equal(t, int64(20), *selectedAPIKey.GroupID)
			require.NotNil(t, selectedAPIKey.Group)
			require.Equal(t, int64(20), selectedAPIKey.Group.ID)
		})
	}
}

func TestEnforcePublicCatalogBindingGroupRejectsUnauthorizedExclusiveGroup(t *testing.T) {
	t.Parallel()

	ctx := service.AttachPublishedPublicCatalogEntry(context.Background(), &service.PublishedPublicCatalogEntry{
		EntryID:        "entry-exclusive",
		PublicModelID:  "gpt-5.4@team-a",
		BindingGroupID: 20,
	})
	apiKey := &service.APIKey{
		ID:     7,
		UserID: 9,
		User:   &service.User{ID: 9, AllowedGroups: []int64{99}},
		GroupBindings: []service.APIKeyGroupBinding{{
			GroupID: 20,
			Group: &service.Group{
				ID:          20,
				Name:        "exclusive-team-a",
				Platform:    service.PlatformOpenAI,
				Status:      service.StatusActive,
				IsExclusive: true,
			},
		}},
	}

	selected, handled, err := enforcePublicCatalogBindingGroup(ctx, apiKey, nil)

	require.Nil(t, selected)
	require.True(t, handled)
	require.Error(t, err)
	require.True(t, infraerrors.IsForbidden(err))
	require.Equal(t, "GROUP_ACCESS_DENIED", infraerrors.Reason(err))
}

func (s multiGroupRuntimeSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key == service.SettingKeyMultiGroupRoutingEnabled {
		return s.value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s multiGroupRuntimeSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected call")
}

func (s multiGroupRuntimeSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected call")
}

func (s multiGroupRuntimeSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected call")
}

func (s multiGroupRuntimeSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected call")
}

func (s multiGroupRuntimeSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected call")
}

func TestResolveSelectedAPIKey_ReturnsGroupExhaustedWhenExcludedGroupReused(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		call func(*gin.Context, *service.SettingService, *service.APIKey, map[int64]struct{}) (*service.APIKey, *service.UserSubscription, error)
	}{
		{
			name: "gateway",
			call: func(ctx *gin.Context, settingService *service.SettingService, apiKey *service.APIKey, excludedGroupIDs map[int64]struct{}) (*service.APIKey, *service.UserSubscription, error) {
				return resolveSelectedGatewayAPIKey(
					ctx,
					settingService,
					nil,
					nil,
					apiKey,
					nil,
					"claude-3-7-sonnet",
					gatewayCompatiblePlatforms,
					excludedGroupIDs,
				)
			},
		},
		{
			name: "openai",
			call: func(ctx *gin.Context, settingService *service.SettingService, apiKey *service.APIKey, excludedGroupIDs map[int64]struct{}) (*service.APIKey, *service.UserSubscription, error) {
				return resolveSelectedOpenAIAPIKey(
					ctx,
					settingService,
					nil,
					nil,
					apiKey,
					nil,
					"gpt-4o",
					openAICompatiblePlatforms,
					excludedGroupIDs,
				)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)

			apiKey := &service.APIKey{
				GroupID: int64PtrForTest(2),
				GroupBindings: []service.APIKeyGroupBinding{
					{GroupID: 2},
					{GroupID: 3},
				},
			}
			settingService := service.NewSettingService(multiGroupRuntimeSettingRepoStub{value: "false"}, nil)
			excludedGroupIDs := map[int64]struct{}{2: {}}

			selectedAPIKey, subscription, err := tc.call(ctx, settingService, apiKey, excludedGroupIDs)

			require.Nil(t, selectedAPIKey)
			require.Nil(t, subscription)
			require.Error(t, err)
			require.True(t, infraerrors.IsServiceUnavailable(err))
			require.Equal(t, "GROUP_EXHAUSTED", infraerrors.Reason(err))
			require.Equal(t, "all accounts in the group have been exhausted", infraerrors.Message(err))
		})
	}
}
