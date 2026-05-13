package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

type openAIOAuthCreateFromOAuthClientStub struct{}

func (s *openAIOAuthCreateFromOAuthClientStub) ExchangeCode(
	context.Context,
	string,
	string,
	string,
	string,
	string,
) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
	}, nil
}

func (s *openAIOAuthCreateFromOAuthClientStub) RefreshToken(context.Context, string, string) (*openai.TokenResponse, error) {
	return nil, nil
}

func (s *openAIOAuthCreateFromOAuthClientStub) RefreshTokenWithClientID(context.Context, string, string, string) (*openai.TokenResponse, error) {
	return nil, nil
}

type openAIOAuthRoundTripper func(*http.Request) (*http.Response, error)

func (f openAIOAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newOpenAIOAuthPrivacyClientFactory(planType string) service.PrivacyClientFactory {
	return func(_ string) (*req.Client, error) {
		client := req.C()
		client.GetClient().Transport = openAIOAuthRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && strings.Contains(req.URL.String(), "backend-api/accounts/check/"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"accounts": {
							"default-org": {
								"account": {
									"plan_type": "` + planType + `",
									"is_default": true
								}
							}
						}
					}`)),
				}, nil
			case req.Method == http.MethodPatch && strings.Contains(req.URL.String(), "backend-api/settings/account_user_setting"):
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
		return client, nil
	}
}

func TestOpenAIOAuthHandler_CreateAccountFromOAuth_AppliesDefaultModelScope(t *testing.T) {
	tests := []struct {
		name       string
		planType   string
		wantModels []string
	}{
		{
			name:     "free",
			planType: "free",
			wantModels: []string{
				"gpt-5.2",
				"gpt-5.4",
				"gpt-5.4-mini",
				"gpt-5.5",
			},
		},
		{
			name:     "plus",
			planType: "plus",
			wantModels: []string{
				"gpt-image-2",
				"gpt-5.2",
				"gpt-5.4",
				"gpt-5.4-mini",
				"gpt-5.5",
			},
		},
		{
			name:     "pro",
			planType: "chatgptpro20x",
			wantModels: []string{
				"gpt-image-2",
				"gpt-5.2",
				"gpt-5.4",
				"gpt-5.4-mini",
				"gpt-5.5",
				"gpt-5.3-codex-spark",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			adminSvc := newStubAdminService()
			oauthSvc := service.NewOpenAIOAuthService(nil, &openAIOAuthCreateFromOAuthClientStub{})
			oauthSvc.SetPrivacyClientFactory(newOpenAIOAuthPrivacyClientFactory(tt.planType))
			defer oauthSvc.Stop()

			authResult, err := oauthSvc.GenerateAuthURL(context.Background(), nil, "", service.PlatformOpenAI)
			require.NoError(t, err)
			require.NotNil(t, authResult)

			authURL, err := url.Parse(authResult.AuthURL)
			require.NoError(t, err)
			state := authURL.Query().Get("state")
			require.NotEmpty(t, state)

			handler := NewOpenAIOAuthHandler(oauthSvc, adminSvc)
			router := gin.New()
			router.POST("/api/v1/admin/openai/create-from-oauth", handler.CreateAccountFromOAuth)

			body, err := json.Marshal(map[string]any{
				"session_id":  authResult.SessionID,
				"code":        "auth-code",
				"state":       state,
				"name":        "created-account",
				"concurrency": 2,
				"priority":    3,
				"group_ids":   []int64{11, 12},
			})
			require.NoError(t, err)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/openai/create-from-oauth", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			require.Len(t, adminSvc.createdAccounts, 1)
			created := adminSvc.createdAccounts[0]
			require.Equal(t, "created-account", created.Name)
			require.Equal(t, service.PlatformOpenAI, created.Platform)
			require.Equal(t, service.AccountTypeOAuth, created.Type)

			scope, ok := service.ExtractAccountModelScopeV2(created.Extra)
			require.True(t, ok)
			require.NotNil(t, scope)
			require.Equal(t, service.AccountModelPolicyModeWhitelist, scope.PolicyMode)

			gotModels := make([]string, 0, len(scope.Entries))
			for _, entry := range scope.Entries {
				gotModels = append(gotModels, entry.DisplayModelID)
			}
			require.ElementsMatch(t, tt.wantModels, gotModels)
			require.Equal(t, service.PlatformOpenAI, created.Extra["gateway_test_provider"])
			require.Equal(t, service.OpenAIOAuthDefaultTestModelID, created.Extra["gateway_test_model_id"])
		})
	}
}
