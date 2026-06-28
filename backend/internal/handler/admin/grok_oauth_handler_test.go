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
