package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountListLiteKeepsDaily5HDiagnosticsWithoutCredentials(t *testing.T) {
	account := &service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "test-secret"}, Extra: map[string]any{
			"daily_5h_trigger_last_status": "waiting", "daily_5h_trigger_task_date": "2026-09-08",
			"daily_5h_trigger_attempts": 2, "daily_5h_trigger_next_retry_at": "2026-09-08T01:00:00Z",
			"daily_5h_trigger_last_summary": "Waiting for upstream window", "unrelated_private_state": "private",
		}}
	row := AccountFromServiceListLite(account)
	require.Equal(t, "waiting", row.Extra["daily_5h_trigger_last_status"])
	require.Equal(t, 2, row.Extra["daily_5h_trigger_attempts"])
	require.Equal(t, "2026-09-08T01:00:00Z", row.Extra["daily_5h_trigger_next_retry_at"])
	require.NotContains(t, row.Extra, "unrelated_private_state")
	require.NotContains(t, row.Credentials, "access_token")
}
