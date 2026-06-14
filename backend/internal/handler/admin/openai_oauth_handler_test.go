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

	handler := NewOpenAIOAuthHandler(openAISvc, adminSvc)
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
