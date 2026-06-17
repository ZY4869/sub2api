package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type queryCapturingExec struct {
	t           *testing.T
	updateQuery string
	outboxCalls int
}

func (e *queryCapturingExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.Contains(query, "scheduler_outbox") {
		e.outboxCalls++
		return queryCaptureResult(1), nil
	}
	e.updateQuery = query
	return queryCaptureResult(1), nil
}

func (e *queryCapturingExec) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	e.t.Fatalf("unexpected query")
	return nil, nil
}

type queryCaptureResult int64

func (r queryCaptureResult) LastInsertId() (int64, error) { return 0, nil }
func (r queryCaptureResult) RowsAffected() (int64, error) { return int64(r), nil }

func newQueryCapturingExec(t *testing.T) *queryCapturingExec {
	t.Helper()
	return &queryCapturingExec{t: t}
}

func TestResetQuotaUsedClearsCodexAndRateLimitStateInSQL(t *testing.T) {
	exec := newQueryCapturingExec(t)
	repo := &accountRepository{sql: exec}

	require.NoError(t, repo.ResetQuotaUsed(context.Background(), 42))
	require.Equal(t, 1, exec.outboxCalls)
	sqlText := compactSQLForTest(exec.updateQuery)

	for _, want := range []string{
		"quota_monthly_used",
		"quota_used_by_currency",
		"quota_daily_used_by_currency",
		"quota_weekly_used_by_currency",
		"quota_monthly_used_by_currency",
		"quota_monthly_start",
		"codex_usage_updated_at",
		"codex_5h_used_percent",
		"codex_7d_reset_at",
		"codex_primary_used_percent",
		"codex_secondary_reset_after_seconds",
		"codex_primary_over_secondary_percent",
		"codex_spark_usage_updated_at",
		"codex_spark_5h_used_percent",
		"codex_spark_7d_reset_at",
		"codex_account_7d_all_exhausted",
		"model_rate_limits",
		"rate_limit_reason",
		"rate_limited_at = NULL",
		"rate_limit_reset_at = NULL",
	} {
		require.Contains(t, sqlText, want)
	}
}

func compactSQLForTest(query string) string {
	fields := strings.Fields(query)
	return strings.Join(fields, " ")
}
