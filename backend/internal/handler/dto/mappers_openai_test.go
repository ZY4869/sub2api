package dto

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func buildOpenAITestIDToken(t *testing.T, payload map[string]any) string {
	t.Helper()

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "none",
		"typ": "JWT",
	})
	require.NoError(t, err)

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	body := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + body + "."
}

func TestAccountFromServiceShallow_EnrichesOpenAIIdentityFromIDToken(t *testing.T) {
	idToken := buildOpenAITestIDToken(t, map[string]any{
		"email": "demo@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acc_1234567890",
			"chatgpt_user_id":    "user_123",
			"chatgpt_plan_type":  "chatgptpro",
			"organizations": []map[string]any{
				{"id": "org_1", "is_default": true},
			},
		},
	})

	originalCredentials := map[string]any{
		"id_token": idToken,
	}
	account := &service.Account{
		ID:          1,
		Name:        "openai-oauth",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Credentials: originalCredentials,
	}

	dtoAccount := AccountFromServiceShallow(account)
	require.NotNil(t, dtoAccount)
	require.Equal(t, "demo@example.com", dtoAccount.Credentials["email"])
	require.Equal(t, "acc_1234567890", dtoAccount.Credentials["chatgpt_account_id"])
	require.Equal(t, "user_123", dtoAccount.Credentials["chatgpt_user_id"])
	require.Equal(t, "org_1", dtoAccount.Credentials["organization_id"])
	require.Equal(t, "pro", dtoAccount.Credentials["plan_type"])
	require.NotContains(t, originalCredentials, "email")
	require.NotContains(t, originalCredentials, "plan_type")
}

func TestAccountFromServiceShallow_NormalizesPlanTypeWithoutMutatingSource(t *testing.T) {
	account := &service.Account{
		ID:       2,
		Name:     "openai-oauth",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "chatgptpro",
		},
	}

	dtoAccount := AccountFromServiceShallow(account)
	require.NotNil(t, dtoAccount)
	require.Equal(t, "pro", dtoAccount.Credentials["plan_type"])
	require.Equal(t, "chatgptpro", account.Credentials["plan_type"])
}
