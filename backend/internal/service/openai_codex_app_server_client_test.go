package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseOpenAICodexRateLimitsResult_ReadsResetCreditsAvailableCount(t *testing.T) {
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	snapshot, err := parseOpenAICodexRateLimitsResult(json.RawMessage(`{
		"rateLimits": {"limitId": "codex"},
		"rateLimitsByLimitId": {"codex": {"usedPercent": 10}},
		"rateLimitResetCredits": {"availableCount": 2}
	}`), now)
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.AvailableCount)
	require.Equal(t, 2, *snapshot.AvailableCount)
	require.Equal(t, openAIResetCreditsStatusAvailable, snapshot.Status)
	require.Equal(t, openAIResetCreditsStatusAvailable, snapshot.ExtraUpdates[openAIResetCreditsStatusExtraKey])
	require.Equal(t, 2, snapshot.ExtraUpdates[openAIResetCreditsAvailableCountExtraKey])
	require.Equal(t, now.Format(time.RFC3339), snapshot.ExtraUpdates[openAIResetCreditsUpdatedAtExtraKey])
	require.NotNil(t, snapshot.RateLimits)
	require.NotNil(t, snapshot.RateLimitsByLimitID)
}

func TestParseOpenAICodexRateLimitsResult_MissingCreditsRemainUnknown(t *testing.T) {
	snapshot, err := parseOpenAICodexRateLimitsResult(json.RawMessage(`{
		"rateLimits": {"limitId": "codex"}
	}`), time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Nil(t, snapshot.AvailableCount)
	require.Equal(t, openAIResetCreditsStatusUnknownOrUnsupported, snapshot.Status)
	require.Equal(t, openAIResetCreditsStatusUnknownOrUnsupported, snapshot.ExtraUpdates[openAIResetCreditsStatusExtraKey])
	require.Contains(t, snapshot.ExtraUpdates, openAIResetCreditsAvailableCountExtraKey)
	require.Nil(t, snapshot.ExtraUpdates[openAIResetCreditsAvailableCountExtraKey])
	require.Contains(t, snapshot.ExtraUpdates, openAIResetCreditsUpdatedAtExtraKey)
	require.Nil(t, snapshot.ExtraUpdates[openAIResetCreditsUpdatedAtExtraKey])
	require.Contains(t, snapshot.ExtraUpdates, openAIRateLimitsAppServerUpdatedAtExtraKey)
}

func TestParseOpenAICodexResetCreditConsumeStatus(t *testing.T) {
	for _, status := range []string{
		openAIResetCreditConsumeStatusReset,
		openAIResetCreditConsumeStatusAlreadyRedeemed,
		openAIResetCreditConsumeStatusNothingToReset,
		openAIResetCreditConsumeStatusNoCredit,
	} {
		got, err := parseOpenAICodexResetCreditConsumeStatus(json.RawMessage(`{"status":"` + status + `"}`))
		require.NoError(t, err)
		require.Equal(t, status, got)
	}
}

func TestOpenAICodexJSONRPCErrorRedactsTokens(t *testing.T) {
	err := openAICodexJSONRPCApplicationError("account/rateLimits/read", &openAICodexJSONRPCError{
		Code:    -32000,
		Message: `failed access_token="secret-token" refresh_token=refresh-secret`,
	})
	msg := err.Error()
	require.NotContains(t, msg, "secret-token")
	require.NotContains(t, msg, "refresh-secret")
	require.Contains(t, msg, "OPENAI_CODEX_APP_SERVER_RPC_ERROR")
	require.Contains(t, msg, "Codex app-server 调用失败")
	require.Contains(t, msg, "rpc_code:-32000")
}

func TestOpenAICodexJSONRPCErrorConsumeMethodNotFoundIsUnsupported(t *testing.T) {
	err := openAICodexJSONRPCApplicationError("account/rateLimitResetCredit/consume", &openAICodexJSONRPCError{
		Code:    -32601,
		Message: "method not found",
	})

	require.Error(t, err)
	require.True(t, isOpenAIResetCreditsUnsupportedError(err))
}

func TestSchemaDirContainsResetCreditConsume(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schema.json"), []byte(`{
		"method": { "const": "account/rateLimitResetCredit/consume" }
	}`), 0o600))

	supported, known := schemaDirContainsResetCreditConsume(dir)

	require.True(t, known)
	require.True(t, supported)
}

func TestSchemaDirMissingResetCreditConsumeIsKnownUnsupported(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schema.json"), []byte(`{
		"method": { "const": "account/rateLimits/read" }
	}`), 0o600))

	supported, known := schemaDirContainsResetCreditConsume(dir)

	require.True(t, known)
	require.False(t, supported)
}
