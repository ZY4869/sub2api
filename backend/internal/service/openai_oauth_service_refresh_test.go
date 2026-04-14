package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

type openaiOAuthClientRefreshStub struct {
	refreshCalls int32
	tokenResp    *openai.TokenResponse
	refreshErr   error
}

func (s *openaiOAuthClientRefreshStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientRefreshStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openai.TokenResponse, error) {
	atomic.AddInt32(&s.refreshCalls, 1)
	if s.refreshErr != nil {
		return nil, s.refreshErr
	}
	if s.tokenResp != nil {
		return s.tokenResp, nil
	}
	return nil, errors.New("not implemented")
}

func (s *openaiOAuthClientRefreshStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openai.TokenResponse, error) {
	return s.RefreshToken(ctx, refreshToken, proxyURL)
}

func TestOpenAIOAuthService_RefreshAccountToken_NoRefreshTokenUsesExistingAccessToken(t *testing.T) {
	client := &openaiOAuthClientRefreshStub{}
	svc := NewOpenAIOAuthService(nil, client)

	expiresAt := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)
	account := &Account{
		ID:       77,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "existing-access-token",
			"expires_at":   expiresAt,
			"client_id":    "client-id-1",
		},
	}

	info, err := svc.RefreshAccountToken(context.Background(), account)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, "existing-access-token", info.AccessToken)
	require.Equal(t, "client-id-1", info.ClientID)
	require.Zero(t, atomic.LoadInt32(&client.refreshCalls), "existing access token should be reused without calling refresh")
}

func TestOpenAIOAuthService_RefreshTokenWithClientID_EnrichesPlanTypeAndPrivacyMode(t *testing.T) {
	client := &openaiOAuthClientRefreshStub{
		tokenResp: &openai.TokenResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
		},
	}
	svc := NewOpenAIOAuthService(nil, client)

	var accountCheckCalls int32
	var privacyCalls int32
	svc.SetPrivacyClientFactory(func(_ string) (*req.Client, error) {
		httpClient := req.C()
		httpClient.GetClient().Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && strings.Contains(req.URL.String(), chatGPTAccountsCheckURL):
				atomic.AddInt32(&accountCheckCalls, 1)
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"accounts": {
							"default-org": {
								"account": {
									"plan_type": "chatgptpro",
									"is_default": true
								}
							}
						}
					}`)),
				}, nil
			case req.Method == http.MethodPatch && strings.Contains(req.URL.String(), openAISettingsURL):
				atomic.AddInt32(&privacyCalls, 1)
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}
		})
		return httpClient, nil
	})

	info, err := svc.RefreshTokenWithClientID(context.Background(), "refresh-token", "", "client-id-2")
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, "pro", info.PlanType)
	require.Equal(t, PrivacyModeTrainingOff, info.PrivacyMode)
	require.Equal(t, "client-id-2", info.ClientID)
	require.Equal(t, int32(1), atomic.LoadInt32(&client.refreshCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&accountCheckCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&privacyCalls))
}
