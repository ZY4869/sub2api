package handler

import (
	"context"
	"net/http/httptest"
	"testing"

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

func (s multiGroupRuntimeSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected call")
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
