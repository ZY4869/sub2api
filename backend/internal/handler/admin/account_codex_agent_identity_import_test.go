//go:build unit

package admin

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCodexImportEntryAcceptsAgentIdentityAuthJSON(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	privateKeyBase64 := base64.StdEncoding.EncodeToString(der)

	item, err := normalizeCodexImportEntry(codexImportEntry{
		Index: 1,
		Value: map[string]any{
			"auth_mode": "agentIdentity",
			"agent_identity": map[string]any{
				"agent_runtime_id":           "runtime-import",
				"agent_private_key":          privateKeyBase64,
				"account_id":                 "account-import",
				"chatgpt_user_id":            "user-import",
				"email":                      "agent@example.invalid",
				"plan_type":                  "pro",
				"organization_id":            "org-import",
				"team_id":                    "team-import",
				"chatgpt_account_is_fedramp": false,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, item)
	require.True(t, item.IsAgentIdentity)
	require.Equal(t, service.OpenAIAuthModeAgentIdentity, item.Credentials["auth_mode"])
	require.Equal(t, "runtime-import", item.Credentials["agent_runtime_id"])
	require.Equal(t, privateKeyBase64, item.Credentials["agent_private_key"])
	require.Equal(t, "account-import", item.Credentials["chatgpt_account_id"])
	require.Equal(t, "user-import", item.Credentials["chatgpt_user_id"])
	require.Equal(t, "org-import", item.Credentials["organization_id"])
	require.Equal(t, "team-import", item.Credentials["team_id"])
	require.Contains(t, item.IdentityKeys[0], "org:org-import")
	require.Contains(t, item.IdentityKeys[0], "team_id:team-import")
	require.NotContains(t, item.Credentials, "access_token")
	require.NotContains(t, item.Credentials, "refresh_token")
	require.NotEmpty(t, item.WarningTexts)
}

func TestCodexImportAgentIdentityKeysIncludeTeamAndKeepLegacyFallback(t *testing.T) {
	base := buildCodexImportAgentIdentityKeysWithScope("acct", "user", "a@example.invalid", "runtime", "org-a", "team-a", "")
	otherTeam := buildCodexImportAgentIdentityKeysWithScope("acct", "user", "a@example.invalid", "runtime", "org-a", "team-b", "")
	require.NotEqual(t, base[0], otherTeam[0])
	require.Contains(t, base[0], "agent_runtime:runtime")
	require.Contains(t, base, "account:acct")

	seen := map[string]codexSeenIdentity{}
	markCodexIdentitySeen(seen, base[:1], 1, "user")
	_, ok := firstSeenCodexIdentity(seen, otherTeam[:1], "user")
	require.False(t, ok)

	markCodexIdentitySeen(seen, base, 1, "user")
	duplicateIndex, ok := firstSeenCodexIdentity(seen, buildCodexImportAgentIdentityKeysWithScope("acct", "user", "a@example.invalid", "runtime", "org-a", "team-a", ""), "user")
	require.True(t, ok)
	require.Equal(t, 1, duplicateIndex)
}

func TestCodexAccountIndexScopedIdentityKeepsLegacyCompatibility(t *testing.T) {
	index := buildCodexAccountIndex([]service.Account{{
		ID: 1,
		Credentials: map[string]any{
			"chatgpt_account_id": "acct",
			"chatgpt_user_id":    "user",
			"organization_id":    "org-a",
			"team_id":            "team-a",
		},
	}})

	scoped, matched := index.Find(buildCodexImportAgentIdentityKeysWithScope("acct", "user", "", "runtime", "org-a", "team-a", ""), "user")
	require.NotNil(t, scoped)
	require.Contains(t, matched, "team_id:team-a")

	legacy, matched := index.Find(buildCodexImportIdentityKeysWithScope("acct", "user", "", "", "", "", "", ""), "user")
	require.NotNil(t, legacy)
	require.Equal(t, "account:acct", matched)
}

func TestSanitizeCodexImportCredentialExtrasBlocksAgentIdentitySecrets(t *testing.T) {
	got := sanitizeCodexImportCredentialExtras(map[string]any{
		"agent_private_key": "override",
		"agent_runtime_id":  "runtime",
		"task_id":           "task",
		"note":              "kept",
	})

	require.Equal(t, map[string]any{"note": "kept"}, got)
}
