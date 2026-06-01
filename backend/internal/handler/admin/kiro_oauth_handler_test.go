package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgkiro "github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type kiroOAuthHandlerClientStub struct{}

func (s *kiroOAuthHandlerClientStub) RegisterAuthCodeClient(context.Context, string, string, string, string) (*service.KiroClientRegistration, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthHandlerClientStub) ExchangeSocialCode(context.Context, string, string, string, string) (*service.KiroTokenInfo, error) {
	return &service.KiroTokenInfo{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Provider:     "github",
		Email:        "kiro@example.com",
	}, nil
}

func (s *kiroOAuthHandlerClientStub) RefreshSocialToken(context.Context, string, string) (*service.KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthHandlerClientStub) ExchangeOIDCCode(context.Context, string, string, string, string, string, string, string) (*service.KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthHandlerClientStub) RefreshOIDCToken(context.Context, string, string, string, string, string, string) (*service.KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func (s *kiroOAuthHandlerClientStub) FetchOIDCUserInfo(context.Context, string, string, string) (*service.KiroTokenInfo, error) {
	return nil, errors.New("not implemented")
}

func TestKiroOAuthHandler_CreateAccountFromOAuth_ForwardsAutoRenewFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	kiroSvc := service.NewKiroOAuthService(nil, &kiroOAuthHandlerClientStub{})
	authURL, err := kiroSvc.GenerateAuthURL(context.Background(), &service.KiroGenerateAuthURLInput{
		Method:      pkgkiro.OAuthMethodGitHub,
		RedirectURI: "http://localhost:19877/oauth/callback",
	})
	require.NoError(t, err)

	handler := NewKiroOAuthHandler(kiroSvc, adminSvc)
	router := gin.New()
	router.POST("/admin/kiro/create-from-oauth", handler.CreateAccountFromOAuth)

	autoRenewEnabled := true
	autoRenewPeriod := service.AccountAutoRenewPeriodQuarter
	expiresAt := int64(1798761600)
	body, err := json.Marshal(map[string]any{
		"session_id":         authURL.SessionID,
		"code":               "oauth-code",
		"state":              authURL.State,
		"expires_at":         expiresAt,
		"auto_renew_enabled": autoRenewEnabled,
		"auto_renew_period":  autoRenewPeriod,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/kiro/create-from-oauth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, service.PlatformKiro, created.Platform)
	require.Equal(t, service.AccountTypeOAuth, created.Type)
	require.Equal(t, &expiresAt, created.ExpiresAt)
	require.NotNil(t, created.AutoRenewEnabled)
	require.True(t, *created.AutoRenewEnabled)
	require.NotNil(t, created.AutoRenewPeriod)
	require.Equal(t, autoRenewPeriod, *created.AutoRenewPeriod)
}

func TestKiroOAuthHandler_CreateAccountFromOAuth_LeavesAutoRenewFieldsNilWhenOmitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	kiroSvc := service.NewKiroOAuthService(nil, &kiroOAuthHandlerClientStub{})
	authURL, err := kiroSvc.GenerateAuthURL(context.Background(), &service.KiroGenerateAuthURLInput{
		Method:      pkgkiro.OAuthMethodGitHub,
		RedirectURI: "http://localhost:19877/oauth/callback",
	})
	require.NoError(t, err)

	handler := NewKiroOAuthHandler(kiroSvc, adminSvc)
	router := gin.New()
	router.POST("/admin/kiro/create-from-oauth", handler.CreateAccountFromOAuth)

	body, err := json.Marshal(map[string]any{
		"session_id": authURL.SessionID,
		"code":       "oauth-code",
		"state":      authURL.State,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/kiro/create-from-oauth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, service.PlatformKiro, created.Platform)
	require.Equal(t, service.AccountTypeOAuth, created.Type)
	require.Nil(t, created.AutoRenewEnabled)
	require.Nil(t, created.AutoRenewPeriod)
}
