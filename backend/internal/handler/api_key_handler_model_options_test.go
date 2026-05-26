package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyHandler_GetGroupModelOptions_DoesNotExposeSourceIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &apiKeyModelOptionsUserRepo{
		user: &service.User{ID: 7, Role: service.RoleUser, Status: service.StatusActive},
	}
	groupRepo := &apiKeyModelOptionsGroupRepo{
		groups: []service.Group{{
			ID:       11,
			Name:     "OpenAI group",
			Platform: service.PlatformOpenAI,
			Status:   service.StatusActive,
			Hydrated: true,
		}},
	}
	userSubRepo := &apiKeyModelOptionsUserSubRepo{}
	accountRepo := &apiKeyModelOptionsAccountRepo{
		accountsByGroup: map[int64][]service.Account{
			11: {{
				ID:             101,
				Name:           "mapped account",
				Platform:       service.PlatformOpenAI,
				Type:           service.AccountTypeAPIKey,
				Status:         service.StatusActive,
				LifecycleState: service.AccountLifecycleNormal,
				Schedulable:    true,
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"policy_mode": service.AccountModelPolicyModeMapping,
						"entries": []any{
							map[string]any{
								"display_model_id": "friendly-gpt",
								"target_model_id":  "hidden-upstream-model",
								"visibility_mode":  service.AccountModelVisibilityModeAlias,
							},
						},
					},
				},
			}},
		},
	}

	apiKeyService := service.NewAPIKeyService(nil, userRepo, groupRepo, userSubRepo, nil, nil, &config.Config{})
	gatewayService := service.NewGatewayService(
		accountRepo,
		groupRepo,
		nil,
		nil,
		userRepo,
		userSubRepo,
		nil,
		nil,
		&config.Config{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	apiKeyService.SetGatewayService(gatewayService)

	router := gin.New()
	handler := NewAPIKeyHandler(apiKeyService)
	router.GET("/api/v1/groups/model-options", func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7, Concurrency: 1})
		c.Next()
	}, handler.GetGroupModelOptions)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/model-options", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, `"public_id":"friendly-gpt"`)
	require.NotContains(t, body, "source_ids")
	require.NotContains(t, body, "target_model_id")
	require.NotContains(t, body, "hidden-upstream-model")

	var envelope struct {
		Code int `json:"code"`
		Data []struct {
			GroupID int64 `json:"group_id"`
			Models  []struct {
				PublicID string `json:"public_id"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Len(t, envelope.Data, 1)
	require.Equal(t, int64(11), envelope.Data[0].GroupID)
	require.Equal(t, "friendly-gpt", envelope.Data[0].Models[0].PublicID)
}

type apiKeyModelOptionsUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r *apiKeyModelOptionsUserRepo) GetByID(context.Context, int64) (*service.User, error) {
	return r.user, nil
}

type apiKeyModelOptionsGroupRepo struct {
	service.GroupRepository
	groups []service.Group
}

func (r *apiKeyModelOptionsGroupRepo) ListActive(context.Context) ([]service.Group, error) {
	out := make([]service.Group, len(r.groups))
	copy(out, r.groups)
	return out, nil
}

type apiKeyModelOptionsUserSubRepo struct {
	service.UserSubscriptionRepository
}

func (r *apiKeyModelOptionsUserSubRepo) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return nil, nil
}

type apiKeyModelOptionsAccountRepo struct {
	service.AccountRepository
	accountsByGroup map[int64][]service.Account
}

func (r *apiKeyModelOptionsAccountRepo) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, platforms []string) ([]service.Account, error) {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[strings.TrimSpace(strings.ToLower(platform))] = struct{}{}
	}
	out := make([]service.Account, 0, len(r.accountsByGroup[groupID]))
	for _, account := range r.accountsByGroup[groupID] {
		if len(allowed) > 0 {
			if _, ok := allowed[strings.TrimSpace(strings.ToLower(account.Platform))]; !ok {
				continue
			}
		}
		out = append(out, account)
	}
	return out, nil
}
