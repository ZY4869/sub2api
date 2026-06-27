package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIOAuthHandlerClientStub struct{}

func (s *openAIOAuthHandlerClientStub) ExchangeCode(context.Context, string, string, string, string, string) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{
		AccessToken:  "openai-access-token",
		RefreshToken: "openai-refresh-token",
		ExpiresIn:    3600,
	}, nil
}

func (s *openAIOAuthHandlerClientStub) RefreshToken(context.Context, string, string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openAIOAuthHandlerClientStub) RefreshTokenWithClientID(context.Context, string, string, string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

type openAIOAuthQuotaServiceStub struct {
	queryUsageID  int64
	resetCreditID int64
	queryUsage    *service.OpenAIQuotaUsage
	resetResult   *service.OpenAIQuotaResetResult
	queryErr      error
	resetErr      error
}

func (s *openAIOAuthQuotaServiceStub) QueryUsage(_ context.Context, accountID int64) (*service.OpenAIQuotaUsage, error) {
	s.queryUsageID = accountID
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	return s.queryUsage, nil
}

func (s *openAIOAuthQuotaServiceStub) ResetCredit(_ context.Context, accountID int64) (*service.OpenAIQuotaResetResult, error) {
	s.resetCreditID = accountID
	if s.resetErr != nil {
		return nil, s.resetErr
	}
	return s.resetResult, nil
}

func TestOpenAIOAuthHandlerCreateAccountFromOAuthForwardsTier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	openAISvc := service.NewOpenAIOAuthService(nil, &openAIOAuthHandlerClientStub{})
	defer openAISvc.Stop()
	authURL, err := openAISvc.GenerateAuthURL(context.Background(), nil, "", service.PlatformOpenAI)
	require.NoError(t, err)
	parsedAuthURL, err := url.Parse(authURL.AuthURL)
	require.NoError(t, err)
	state := parsedAuthURL.Query().Get("state")
	require.NotEmpty(t, state)

	handler := NewOpenAIOAuthHandler(openAISvc, adminSvc, nil)
	router := gin.New()
	router.POST("/admin/openai/create-from-oauth", handler.CreateAccountFromOAuth)

	body, err := json.Marshal(map[string]any{
		"session_id":   authURL.SessionID,
		"code":         "oauth-code",
		"state":        state,
		"account_tier": service.OpenAIAccountTierFree,
		"concurrency":  0,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/openai/create-from-oauth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, service.PlatformOpenAI, created.Platform)
	require.Equal(t, service.AccountTypeOAuth, created.Type)
	require.Equal(t, service.OpenAIAccountTierFree, created.Extra[service.AccountExtraKeyAccountTier])
	require.Equal(t, service.PlatformOpenAI, created.Extra["gateway_test_provider"])
	require.Equal(t, service.OpenAIOAuthDefaultTestModelID, created.Extra["gateway_test_model_id"])
	require.Equal(t, 0, created.Concurrency)
}

func TestOpenAIOAuthHandlerQueryQuotaUsesDedicatedService(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	fetchedAt := time.Now().UTC().Unix()
	quotaSvc := &openAIOAuthQuotaServiceStub{
		queryUsage: &service.OpenAIQuotaUsage{
			AccountID: "acct_chatgpt",
			RateLimitResetCredits: &service.OpenAIRateLimitResetCredits{
				AvailableCount: 4,
			},
			FetchedAt: fetchedAt,
		},
	}
	handler := NewOpenAIOAuthHandler(nil, newStubAdminService(), quotaSvc)
	router := gin.New()
	router.GET("/api/v1/admin/openai/accounts/:id/quota", handler.QueryQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/openai/accounts/8101/quota", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(8101), quotaSvc.queryUsageID)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AccountID             string `json:"account_id"`
			RateLimitResetCredits struct {
				AvailableCount int `json:"available_count"`
			} `json:"rate_limit_reset_credits"`
			FetchedAt int64 `json:"fetched_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "acct_chatgpt", resp.Data.AccountID)
	require.Equal(t, 4, resp.Data.RateLimitResetCredits.AvailableCount)
	require.Equal(t, fetchedAt, resp.Data.FetchedAt)
}

func TestOpenAIOAuthHandlerResetQuotaUsesDedicatedService(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	quotaSvc := &openAIOAuthQuotaServiceStub{
		resetResult: &service.OpenAIQuotaResetResult{
			Code:         "success",
			WindowsReset: 2,
			Credit: &service.OpenAIQuotaResetCredit{
				ID:     "credit_1",
				Status: "redeemed",
			},
		},
	}
	handler := NewOpenAIOAuthHandler(nil, newStubAdminService(), quotaSvc)
	router := gin.New()
	router.POST("/api/v1/admin/openai/accounts/:id/reset-quota", handler.ResetQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/openai/accounts/8102/reset-quota", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(8102), quotaSvc.resetCreditID)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Code         string `json:"code"`
			WindowsReset int    `json:"windows_reset"`
			Credit       struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"credit"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "success", resp.Data.Code)
	require.Equal(t, 2, resp.Data.WindowsReset)
	require.Equal(t, "credit_1", resp.Data.Credit.ID)
	require.Equal(t, "redeemed", resp.Data.Credit.Status)
}

func TestOpenAIOAuthHandlerQuotaPropagatesUpstreamStatus(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	quotaSvc := &openAIOAuthQuotaServiceStub{
		resetErr: infraerrors.New(http.StatusTooManyRequests, "OPENAI_QUOTA_RESET_UPSTREAM_ERROR", "OpenAI quota reset upstream returned 429"),
	}
	handler := NewOpenAIOAuthHandler(nil, newStubAdminService(), quotaSvc)
	router := gin.New()
	router.POST("/api/v1/admin/openai/accounts/:id/reset-quota", handler.ResetQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/openai/accounts/8103/reset-quota", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, int64(8103), quotaSvc.resetCreditID)
	require.Contains(t, rec.Body.String(), "OPENAI_QUOTA_RESET_UPSTREAM_ERROR")
}
