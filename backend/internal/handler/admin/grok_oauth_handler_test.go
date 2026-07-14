package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/grokoauth"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type grokOAuthHandlerClientStub struct{}

func (s *grokOAuthHandlerClientStub) ExchangeCode(context.Context, string, string, string, string, string, string) (*grokoauth.TokenResponse, error) {
	return &grokoauth.TokenResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scope:        "openid profile email",
	}, nil
}

func (s *grokOAuthHandlerClientStub) StartDeviceFlow(context.Context, string, string, string, string) (*grokoauth.DeviceAuthorizationResponse, error) {
	return &grokoauth.DeviceAuthorizationResponse{
		DeviceCode:              "device-secret",
		UserCode:                "ABCD-EFGH",
		VerificationURI:         "https://auth.x.ai/activate",
		VerificationURIComplete: "https://auth.x.ai/activate?user_code=ABCD-EFGH",
		ExpiresIn:               600,
		Interval:                2,
	}, nil
}

func (s *grokOAuthHandlerClientStub) PollDeviceToken(context.Context, string, string, string, string) (*grokoauth.TokenResponse, error) {
	return &grokoauth.TokenResponse{
		AccessToken:  "device-access",
		RefreshToken: "device-refresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}, nil
}

func (s *grokOAuthHandlerClientStub) RefreshToken(context.Context, string, string, string, string, string) (*grokoauth.TokenResponse, error) {
	return &grokoauth.TokenResponse{
		AccessToken:  "refreshed-access",
		RefreshToken: "rotated-refresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}, nil
}

func (s *grokOAuthHandlerClientStub) FetchUserInfo(context.Context, string, string, string) (*grokoauth.UserInfo, error) {
	return &grokoauth.UserInfo{
		Sub:   "user-1",
		Email: "grok@example.com",
		Name:  "Grok User",
	}, nil
}

func TestGrokOAuthHandler_CreateAccountFromOAuth_CreatesGrokOAuthAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	grokSvc := service.NewGrokOAuthService(nil, &grokOAuthHandlerClientStub{}, &config.Config{})
	authURL, err := grokSvc.GenerateAuthURL(context.Background(), &service.GrokGenerateAuthURLInput{})
	require.NoError(t, err)

	handler := NewGrokOAuthHandler(grokSvc, adminSvc)
	router := gin.New()
	router.POST("/admin/grok/create-from-oauth", handler.CreateAccountFromOAuth)

	expiresAt := int64(1798761600)
	body, err := json.Marshal(map[string]any{
		"session_id": authURL.SessionID,
		"code":       "oauth-code",
		"state":      authURL.State,
		"expires_at": expiresAt,
		"name":       "",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/grok/create-from-oauth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, service.PlatformGrok, created.Platform)
	require.Equal(t, service.AccountTypeOAuth, created.Type)
	require.Equal(t, "access-token", created.Credentials["access_token"])
	require.Equal(t, "refresh-token", created.Credentials["refresh_token"])
	require.Equal(t, "https://api.x.ai/v1", created.Credentials["base_url"])
	require.Equal(t, "grok_browser_oauth", created.Extra["source"])
	require.Equal(t, &expiresAt, created.ExpiresAt)
}

func TestGrokOAuthHandler_DeviceFlow_StartAndPoll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	grokSvc := service.NewGrokOAuthService(nil, &grokOAuthHandlerClientStub{}, &config.Config{})
	handler := NewGrokOAuthHandler(grokSvc, newStubAdminService())
	router := gin.New()
	router.POST("/admin/grok/oauth/device/start", handler.StartDeviceFlow)
	router.POST("/admin/grok/oauth/device/poll", handler.PollDeviceToken)

	startReq := httptest.NewRequest(http.MethodPost, "/admin/grok/oauth/device/start", bytes.NewReader([]byte(`{}`)))
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)

	require.Equal(t, http.StatusOK, startRec.Code)
	var startBody struct {
		Data service.GrokDeviceFlowStartResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(startRec.Body.Bytes(), &startBody))
	require.NotEmpty(t, startBody.Data.SessionID)
	require.Equal(t, "ABCD-EFGH", startBody.Data.UserCode)

	pollPayload, err := json.Marshal(map[string]any{"session_id": startBody.Data.SessionID})
	require.NoError(t, err)
	pollReq := httptest.NewRequest(http.MethodPost, "/admin/grok/oauth/device/poll", bytes.NewReader(pollPayload))
	pollReq.Header.Set("Content-Type", "application/json")
	pollRec := httptest.NewRecorder()
	router.ServeHTTP(pollRec, pollReq)

	require.Equal(t, http.StatusOK, pollRec.Code)
	var pollBody struct {
		Data service.GrokDevicePollResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(pollRec.Body.Bytes(), &pollBody))
	require.Equal(t, service.GrokDeviceStatusAuthorized, pollBody.Data.Status)
	require.NotNil(t, pollBody.Data.TokenInfo)
	require.Equal(t, "device-access", pollBody.Data.TokenInfo.AccessToken)
}
