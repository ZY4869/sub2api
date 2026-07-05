package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestImportCodexSessionsCreatesAccessOnlyAccount(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = nil
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	token := testCodexJWT(t, time.Now().Add(time.Hour), "acct-create", "user-create", "user@example.com")
	req := CodexSessionImportRequest{
		Content:          token,
		CredentialExtras: map[string]any{"access_token": "bad", "custom_flag": "ok"},
	}
	entries, err := parseCodexSessionImportEntries(req)
	require.NoError(t, err)

	result, err := handler.importCodexSessions(context.Background(), req, entries)

	require.NoError(t, err)
	require.Equal(t, 1, result.Created)
	require.Equal(t, 0, result.Failed)
	require.Len(t, adminSvc.createdAccounts, 1)
	input := adminSvc.createdAccounts[0]
	require.Equal(t, service.PlatformOpenAI, input.Platform)
	require.Equal(t, service.AccountTypeOAuth, input.Type)
	require.Equal(t, 3, input.Concurrency)
	require.Equal(t, 50, input.Priority)
	require.NotNil(t, input.ExpiresAt)
	require.NotNil(t, input.AutoPauseOnExpired)
	require.True(t, *input.AutoPauseOnExpired)
	require.Equal(t, token, input.Credentials["access_token"])
	require.Equal(t, "ok", input.Credentials["custom_flag"])
	require.NotEqual(t, "bad", input.Credentials["access_token"])
	require.Equal(t, "user-create", input.Credentials["chatgpt_user_id"])
}

func TestImportCodexSessionsAccessOnlyPreservesExistingRefresh(t *testing.T) {
	adminSvc := newStubAdminService()
	token := testCodexJWT(t, time.Now().Add(time.Hour), "acct-existing", "user-existing", "existing@example.com")
	adminSvc.accounts = []service.Account{{
		ID:       42,
		Name:     "existing",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       token,
			"refresh_token":      "keep-refresh",
			"client_id":          "keep-client",
			"chatgpt_account_id": "acct-existing",
			"chatgpt_user_id":    "user-existing",
		},
	}}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := CodexSessionImportRequest{Content: token}
	entries, err := parseCodexSessionImportEntries(req)
	require.NoError(t, err)

	result, err := handler.importCodexSessions(context.Background(), req, entries)

	require.NoError(t, err)
	require.Equal(t, 1, result.Updated)
	require.Len(t, adminSvc.updatedAccounts, 1)
	update := adminSvc.updatedAccounts[0]
	require.Nil(t, update.ExpiresAt)
	require.Nil(t, update.AutoPauseOnExpired)
	require.Equal(t, "keep-refresh", update.Credentials["refresh_token"])
	require.Equal(t, "keep-client", update.Credentials["client_id"])
}

func TestImportCodexSessionsRejectsExpiredAccessToken(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = nil
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := CodexSessionImportRequest{
		Content: testCodexJWT(t, time.Now().Add(-time.Hour), "acct-expired", "user-expired", "expired@example.com"),
	}
	entries, err := parseCodexSessionImportEntries(req)
	require.NoError(t, err)

	result, err := handler.importCodexSessions(context.Background(), req, entries)

	require.NoError(t, err)
	require.Equal(t, 1, result.Failed)
	require.Len(t, adminSvc.createdAccounts, 0)
	require.Contains(t, result.Errors[0].Message, "access_token 已过期")
}

func testCodexJWT(t *testing.T, expiresAt time.Time, accountID, userID, email string) string {
	t.Helper()
	header := map[string]any{"alg": "none", "typ": "JWT"}
	payload := map[string]any{
		"sub":   userID,
		"email": email,
		"exp":   expiresAt.Unix(),
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": accountID,
			"chatgpt_user_id":    userID,
			"chatgpt_plan_type":  "pro",
			"organizations": []openai.OrganizationClaim{{
				ID:        "org-1",
				IsDefault: true,
			}},
		},
	}
	return encodeCodexJWTPart(t, header) + "." + encodeCodexJWTPart(t, payload) + ".sig"
}

func encodeCodexJWTPart(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(data)
}
