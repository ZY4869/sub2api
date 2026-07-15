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
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1", created.Credentials["base_url"])
	require.Equal(t, "grok_browser_oauth", created.Extra["source"])
	scope, ok := service.ExtractAccountModelScopeV2(created.Extra)
	require.True(t, ok)
	require.Equal(t, service.AccountModelPolicyModeWhitelist, scope.PolicyMode)
	require.Len(t, scope.Entries, len(service.GrokBuildTextModelIDs()))
	entriesByModel := make(map[string]service.AccountModelScopeEntry, len(scope.Entries))
	for _, entry := range scope.Entries {
		entriesByModel[entry.DisplayModelID] = entry
	}
	for _, modelID := range service.GrokBuildTextModelIDs() {
		entry, exists := entriesByModel[modelID]
		require.True(t, exists)
		require.Equal(t, modelID, entry.TargetModelID)
		require.Equal(t, service.PlatformGrok, entry.Provider)
		require.Equal(t, service.AccountModelVisibilityModeDefault, entry.VisibilityMode)
	}
	require.Equal(t, &expiresAt, created.ExpiresAt)
}

func TestGrokOAuthHandler_ReauthorizeAccountFromOAuth_AddsDefaultScopeWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:          901,
		Name:        "grok",
		Platform:    service.PlatformGrok,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"access_token": "old-access",
		},
		Extra: map[string]any{
			"source": "grok_browser_oauth",
		},
	}}
	grokSvc := service.NewGrokOAuthService(nil, &grokOAuthHandlerClientStub{}, &config.Config{})
	authURL, err := grokSvc.GenerateAuthURL(context.Background(), &service.GrokGenerateAuthURLInput{})
	require.NoError(t, err)

	handler := NewGrokOAuthHandler(grokSvc, adminSvc)
	router := gin.New()
	router.POST("/admin/grok/:id/reauthorize", handler.ReauthorizeAccountFromOAuth)

	body, err := json.Marshal(map[string]any{
		"session_id": authURL.SessionID,
		"code":       "oauth-code",
		"state":      authURL.State,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/grok/901/reauthorize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.updatedAccounts, 1)
	scope, ok := service.ExtractAccountModelScopeV2(adminSvc.updatedAccounts[0].Extra)
	require.True(t, ok)
	require.Len(t, scope.Entries, len(service.GrokBuildTextModelIDs()))
	require.Equal(t, "access-token", adminSvc.updatedAccounts[0].Credentials["access_token"])
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1", adminSvc.updatedAccounts[0].Credentials["base_url"])
	require.Equal(t, "grok_browser_oauth", adminSvc.updatedAccounts[0].Extra["source"])
}

func TestGrokOAuthHandler_ReauthorizeAccountFromOAuth_PreservesExistingScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	existingScope := (&service.AccountModelScopeV2{
		PolicyMode: service.AccountModelPolicyModeWhitelist,
		Entries: []service.AccountModelScopeEntry{{
			DisplayModelID: service.GrokModelBuild01,
			TargetModelID:  service.GrokModelBuild01,
			Provider:       service.PlatformGrok,
			VisibilityMode: service.AccountModelVisibilityModeDefault,
		}},
	}).ToMap()
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:          902,
		Name:        "grok",
		Platform:    service.PlatformGrok,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"access_token": "old-access",
		},
		Extra: map[string]any{
			"source":         "grok_browser_oauth",
			"model_scope_v2": existingScope,
		},
	}}
	grokSvc := service.NewGrokOAuthService(nil, &grokOAuthHandlerClientStub{}, &config.Config{})
	authURL, err := grokSvc.GenerateAuthURL(context.Background(), &service.GrokGenerateAuthURLInput{})
	require.NoError(t, err)

	handler := NewGrokOAuthHandler(grokSvc, adminSvc)
	router := gin.New()
	router.POST("/admin/grok/:id/reauthorize", handler.ReauthorizeAccountFromOAuth)

	body, err := json.Marshal(map[string]any{
		"session_id": authURL.SessionID,
		"code":       "oauth-code",
		"state":      authURL.State,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/grok/902/reauthorize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.updatedAccounts, 1)
	scope, ok := service.ExtractAccountModelScopeV2(adminSvc.updatedAccounts[0].Extra)
	require.True(t, ok)
	require.Len(t, scope.Entries, 1)
	require.Equal(t, service.GrokModelBuild01, scope.Entries[0].DisplayModelID)
	require.Equal(t, service.GrokModelBuild01, scope.Entries[0].TargetModelID)
	require.Equal(t, "access-token", adminSvc.updatedAccounts[0].Credentials["access_token"])
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1", adminSvc.updatedAccounts[0].Credentials["base_url"])
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
