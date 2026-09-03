package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuthServiceValidateCodexPersonalAccessToken(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"user@example.com","chatgpt_user_id":"user-123","chatgpt_account_id":"acct-123","chatgpt_plan_type":"plus","chatgpt_account_is_fedramp":true}`))
	}))
	defer server.Close()
	previous := openAICodexPATWhoamiURL
	openAICodexPATWhoamiURL = server.URL
	defer func() { openAICodexPATWhoamiURL = previous }()

	svc := NewOpenAIOAuthService(nil, nil)
	defer svc.Stop()
	info, err := svc.ValidateCodexPersonalAccessToken(context.Background(), " at-test-token ", "")

	require.NoError(t, err)
	require.Equal(t, "Bearer at-test-token", authorization)
	require.Equal(t, OpenAIAuthModePersonalAccessToken, info.AuthMode)
	require.Equal(t, "acct-123", info.ChatGPTAccountID)
	require.True(t, info.ChatGPTAccountFedRAMP)
}

func TestOpenAIOAuthServiceBuildAccountCredentialsForPAT(t *testing.T) {
	svc := NewOpenAIOAuthService(nil, nil)
	defer svc.Stop()
	credentials := svc.BuildAccountCredentials(&OpenAITokenInfo{
		AccessToken:           "at-test-token",
		AuthMode:              OpenAIAuthModePersonalAccessToken,
		ChatGPTAccountFedRAMP: true,
	})

	require.Equal(t, OpenAIAuthModePersonalAccessToken, credentials["auth_mode"])
	require.Equal(t, "personal_access_token", credentials["openai_auth_mode"])
	require.Equal(t, "Bearer", credentials["token_type"])
	require.Equal(t, true, credentials["chatgpt_account_is_fedramp"])
	require.NotContains(t, credentials, "refresh_token")
	require.NotContains(t, credentials, "expires_at")
}

func TestAccountIsChatGPTAccountFedRAMP(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"chatgpt_account_is_fedramp": "true"}}
	require.True(t, account.IsChatGPTAccountFedRAMP())
	account.Credentials["chatgpt_account_is_fedramp"] = false
	require.False(t, account.IsChatGPTAccountFedRAMP())
}
