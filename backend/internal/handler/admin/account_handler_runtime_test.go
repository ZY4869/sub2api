package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerGetRuntimeSummaryUsesSnakeCaseJSON(t *testing.T) {
	adminSvc := newStubAdminService()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/runtime-summary", handler.GetRuntimeSummary)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/runtime-summary", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			InUse int64 `json:"in_use"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(0), resp.Data.InUse)
}

func TestAccountHandlerGetRuntimeSummaryAvailableOnlyReturnsZeroInUse(t *testing.T) {
	adminSvc := newStubAdminService()
	now := time.Now().UTC()
	adminSvc.accounts = []service.Account{
		{
			ID:          1,
			Name:        "available",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
		},
		{
			ID:               2,
			Name:             "in-use",
			Platform:         service.PlatformOpenAI,
			Type:             service.AccountTypeAPIKey,
			Status:           service.StatusActive,
			Schedulable:      true,
			SessionWindowEnd: &now,
		},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/runtime-summary", handler.GetRuntimeSummary)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/runtime-summary?runtime_view=available_only", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			InUse int64 `json:"in_use"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(0), resp.Data.InUse)
}

func TestAccountMatchesRuntimeSummaryFilters_RequiresDispatchableAccount(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(6 * time.Hour).UTC().Truncate(time.Second)

	normal := &service.Account{
		ID:          1,
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
	}
	require.True(t, accountMatchesRuntimeSummaryFilters(normal, service.AccountStatusSummaryFilters{}, now))
	require.True(t, accountMatchesRuntimeSummaryFilters(normal, service.AccountStatusSummaryFilters{LimitedView: service.AccountLimitedViewNormalOnly}, now))
	require.False(t, accountMatchesRuntimeSummaryFilters(normal, service.AccountStatusSummaryFilters{LimitedView: service.AccountLimitedViewLimitedOnly}, now))

	nonProLimited := &service.Account{
		ID:          2,
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_7d_used_percent": 100.0,
			"codex_7d_reset_at":     resetAt.Format(time.RFC3339),
		},
	}
	require.False(t, accountMatchesRuntimeSummaryFilters(nonProLimited, service.AccountStatusSummaryFilters{}, now))
	require.False(t, accountMatchesRuntimeSummaryFilters(nonProLimited, service.AccountStatusSummaryFilters{LimitedView: service.AccountLimitedViewLimitedOnly}, now))
	require.False(t, accountMatchesRuntimeSummaryFilters(nonProLimited, service.AccountStatusSummaryFilters{LimitedView: service.AccountLimitedViewNormalOnly}, now))

	proPartial := &service.Account{
		ID:          3,
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{
			"codex_7d_used_percent": 100.0,
			"codex_7d_reset_at":     resetAt.Format(time.RFC3339),
		},
	}
	require.False(t, accountMatchesRuntimeSummaryFilters(proPartial, service.AccountStatusSummaryFilters{LimitedView: service.AccountLimitedViewLimitedOnly}, now))
	require.True(t, accountMatchesRuntimeSummaryFilters(proPartial, service.AccountStatusSummaryFilters{LimitedView: service.AccountLimitedViewNormalOnly}, now))

	future := now.Add(10 * time.Minute)
	expired := now.Add(-10 * time.Minute)
	cases := []struct {
		name    string
		account *service.Account
	}{
		{
			name: "persisted_rate_limited",
			account: &service.Account{
				ID:               4,
				Platform:         service.PlatformOpenAI,
				Type:             service.AccountTypeOAuth,
				Status:           service.StatusActive,
				Schedulable:      true,
				RateLimitResetAt: &future,
			},
		},
		{
			name: "paused",
			account: &service.Account{
				ID:          5,
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Status:      service.StatusActive,
				Schedulable: false,
			},
		},
		{
			name: "error",
			account: &service.Account{
				ID:          6,
				Platform:    service.PlatformOpenAI,
				Type:        service.AccountTypeOAuth,
				Status:      service.StatusError,
				Schedulable: true,
			},
		},
		{
			name: "temp_unschedulable",
			account: &service.Account{
				ID:                      7,
				Platform:                service.PlatformOpenAI,
				Type:                    service.AccountTypeOAuth,
				Status:                  service.StatusActive,
				Schedulable:             true,
				TempUnschedulableUntil:  &future,
				TempUnschedulableReason: "temporary upstream error",
			},
		},
		{
			name: "overloaded",
			account: &service.Account{
				ID:            8,
				Platform:      service.PlatformOpenAI,
				Type:          service.AccountTypeOAuth,
				Status:        service.StatusActive,
				Schedulable:   true,
				OverloadUntil: &future,
			},
		},
		{
			name: "auto_paused_expired",
			account: &service.Account{
				ID:                 9,
				Platform:           service.PlatformOpenAI,
				Type:               service.AccountTypeOAuth,
				Status:             service.StatusActive,
				Schedulable:        true,
				AutoPauseOnExpired: true,
				ExpiresAt:          &expired,
			},
		},
		{
			name: "blacklisted",
			account: &service.Account{
				ID:             10,
				Platform:       service.PlatformOpenAI,
				Type:           service.AccountTypeOAuth,
				Status:         service.StatusActive,
				Schedulable:    true,
				LifecycleState: service.AccountLifecycleBlacklisted,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			require.False(t, accountMatchesRuntimeSummaryFilters(tt.account, service.AccountStatusSummaryFilters{}, now))
		})
	}
}
