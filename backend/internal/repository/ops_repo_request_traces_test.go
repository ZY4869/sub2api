package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newOpsSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db, mock
}

func TestOpsInsertRequestTraceArgsUsesEmptyStringsForRequiredTextColumns(t *testing.T) {
	t.Parallel()

	args := opsInsertRequestTraceArgs(&service.OpsInsertRequestTraceInput{})
	require.Equal(t, sql.NullString{}, args[0])
	require.Equal(t, sql.NullString{}, args[1])
	require.Equal(t, sql.NullString{}, args[2])
	require.Equal(t, "", args[3])
	require.Equal(t, "", args[4])
	require.Equal(t, "", args[5])
	require.Equal(t, "", args[10])
	require.Equal(t, "", args[11])
	require.Equal(t, "", args[12])
	require.Equal(t, "", args[13])
	require.Equal(t, "", args[14])
	require.Equal(t, "", args[15])
	require.Equal(t, "", args[16])
	require.Equal(t, "", args[17])
	require.Equal(t, "", args[18])
	require.Equal(t, "", args[19])
	require.Equal(t, "", args[20])
	require.Equal(t, "", args[28])
	require.Equal(t, "", args[29])
	require.Equal(t, "", args[34])
	require.Equal(t, "", args[35])
	require.Equal(t, "", args[37])
	require.Equal(t, "", args[38])
	require.Equal(t, "", args[39])
}

func TestOpsInsertRequestTraceArgsUsesEmptyArrayForToolKinds(t *testing.T) {
	t.Parallel()

	args := opsInsertRequestTraceArgs(&service.OpsInsertRequestTraceInput{})
	valuer, ok := args[32].(driver.Valuer)
	require.True(t, ok)

	value, err := valuer.Value()
	require.NoError(t, err)
	require.Equal(t, "{}", value)
}

func TestBuildOpsRequestTracesWhere_UsesGeminiMetadataExactMatchFilters(t *testing.T) {
	t.Parallel()

	filter := &service.OpsRequestTraceFilter{
		GeminiSurface: "native",
		BillingRuleID: "rule_text_output",
		ProbeAction:   "test",
	}

	where, args := buildOpsRequestTracesWhere(filter)
	require.Len(t, args, 3)
	require.Contains(t, where, "COALESCE(t.gemini_surface,'') = $")
	require.Contains(t, where, "COALESCE(t.billing_rule_id,'') = $")
	require.Contains(t, where, "COALESCE(t.probe_action,'') = $")
}

func TestBuildOpsRequestTracesWhere_GracefullyHandlesMissingGeminiMetadataColumns(t *testing.T) {
	t.Parallel()

	filter := &service.OpsRequestTraceFilter{
		GeminiSurface: "native",
		BillingRuleID: "rule_text_output",
		ProbeAction:   "test",
	}

	where, args := buildOpsRequestTracesWhereWithSchema(filter, opsRequestTraceSchema{})
	require.Len(t, args, 3)
	require.NotContains(t, where, "t.gemini_surface")
	require.NotContains(t, where, "t.billing_rule_id")
	require.NotContains(t, where, "t.probe_action")
	require.Equal(t, 3, strings.Count(where, "'' = $"))
}

func TestBuildInsertOpsRequestTraceSQLAndArgs_OmitsMissingGeminiMetadataColumns(t *testing.T) {
	t.Parallel()

	query, args := buildInsertOpsRequestTraceSQLAndArgs(&service.OpsInsertRequestTraceInput{}, opsRequestTraceSchema{})
	require.NotContains(t, query, "gemini_surface")
	require.NotContains(t, query, "billing_rule_id")
	require.NotContains(t, query, "probe_action")
	require.Len(t, args, 54)
}

func TestOpsRepositoryListRequestTraces_UsesLiteralFallbackMetadataExpressions(t *testing.T) {
	t.Parallel()

	db, mock := newOpsSQLMock(t)
	repo := &opsRepository{db: db}
	repo.requestTraceSchema.loaded = true
	repo.requestTraceSchema.value = opsRequestTraceSchema{}

	startTime := time.Date(2026, 4, 3, 20, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Hour)
	filter := &service.OpsRequestTraceFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM ops_request_traces t WHERE 1=1 AND t.created_at >= $1 AND t.created_at < $2")).
		WithArgs(startTime.UTC(), endTime.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`(?s)SELECT.*''.*,.*''.*,.*''.*FROM ops_request_traces t\s+LEFT JOIN accounts a ON a.id = t\.account_id\s+LEFT JOIN groups g ON g.id = t\.group_id\s+WHERE 1=1 AND t\.created_at >= \$1 AND t\.created_at < \$2\s+ORDER BY t.created_at DESC, t.id DESC\s+LIMIT \$3 OFFSET \$4`).
		WithArgs(startTime.UTC(), endTime.UTC(), 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	result, err := repo.ListRequestTraces(context.Background(), filter)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Items)
	require.EqualValues(t, 0, result.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetRequestTraceSummaryOffsetsTrendPlaceholders(t *testing.T) {
	t.Parallel()

	db, mock := newOpsSQLMock(t)
	repo := &opsRepository{db: db}
	repo.requestTraceSchema.loaded = true
	repo.requestTraceSchema.value = defaultOpsRequestTraceSchema()

	startTime := time.Date(2026, 4, 3, 20, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Hour)
	filter := &service.OpsRequestTraceFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT
  COUNT(*)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) < 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) >= 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.stream, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_tools, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_thinking, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.raw_available, false))::bigint,
  COALESCE(AVG(COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.50) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.95) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.99) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0)
FROM ops_request_traces t
WHERE 1=1 AND t.created_at >= $1 AND t.created_at < $2`)).
		WithArgs(startTime.UTC(), endTime.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{
			"request_count",
			"success_count",
			"error_count",
			"stream_count",
			"tool_count",
			"thinking_count",
			"raw_available_count",
			"avg_duration_ms",
			"p50_duration_ms",
			"p95_duration_ms",
			"p99_duration_ms",
		}).AddRow(int64(1), int64(1), int64(0), int64(0), int64(0), int64(0), int64(0), float64(12), int64(12), int64(12), int64(12)))

	mock.ExpectQuery(`to_timestamp\(floor\(extract\(epoch from t\.created_at\) / \$1\) \* \$1\).*WHERE 1=1 AND t\.created_at >= \$2 AND t\.created_at < \$3`).
		WithArgs(int64(300), startTime.UTC(), endTime.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket_start",
			"request_count",
			"error_count",
			"p50_duration_ms",
			"p95_duration_ms",
			"p99_duration_ms",
		}))

	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), 8).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`COUNT\(\*\) FILTER \(WHERE COALESCE\(t\.stream, false\)\)::bigint`).
		WithArgs(startTime.UTC(), endTime.UTC()).
		WillReturnRows(sqlmock.NewRows([]string{
			"stream_count",
			"tool_count",
			"thinking_count",
			"raw_count",
			"estimated_count",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), int64(0)))

	summary, err := repo.GetRequestTraceSummary(context.Background(), filter)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetRequestTraceSummaryOffsetsTrendPlaceholdersWithAdditionalFilters(t *testing.T) {
	t.Parallel()

	db, mock := newOpsSQLMock(t)
	repo := &opsRepository{db: db}
	repo.requestTraceSchema.loaded = true
	repo.requestTraceSchema.value = defaultOpsRequestTraceSchema()

	startTime := time.Date(2026, 4, 3, 20, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Hour)
	accountID := int64(654)
	platform := "openai"
	filter := &service.OpsRequestTraceFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
		AccountID: &accountID,
		Platform:  platform,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT
  COUNT(*)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) < 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) >= 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.stream, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_tools, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_thinking, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.raw_available, false))::bigint,
  COALESCE(AVG(COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.50) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.95) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.99) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0)
FROM ops_request_traces t
WHERE 1=1 AND t.created_at >= $1 AND t.created_at < $2 AND COALESCE(t.platform,'') = $3 AND t.account_id = $4`)).
		WithArgs(startTime.UTC(), endTime.UTC(), platform, accountID).
		WillReturnRows(sqlmock.NewRows([]string{
			"request_count",
			"success_count",
			"error_count",
			"stream_count",
			"tool_count",
			"thinking_count",
			"raw_available_count",
			"avg_duration_ms",
			"p50_duration_ms",
			"p95_duration_ms",
			"p99_duration_ms",
		}).AddRow(int64(1), int64(1), int64(0), int64(0), int64(0), int64(0), int64(0), float64(12), int64(12), int64(12), int64(12)))

	mock.ExpectQuery(`to_timestamp\(floor\(extract\(epoch from t\.created_at\) / \$1\) \* \$1\).*WHERE 1=1 AND t\.created_at >= \$2 AND t\.created_at < \$3 AND COALESCE\(t\.platform,''\) = \$4 AND t\.account_id = \$5`).
		WithArgs(int64(300), startTime.UTC(), endTime.UTC(), platform, accountID).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket_start",
			"request_count",
			"error_count",
			"p50_duration_ms",
			"p95_duration_ms",
			"p99_duration_ms",
		}))

	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), platform, accountID, 8).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), platform, accountID, 10).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), platform, accountID, 10).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`SELECT key, label, count`).
		WithArgs(startTime.UTC(), endTime.UTC(), platform, accountID, 10).
		WillReturnRows(sqlmock.NewRows([]string{"key", "label", "count"}))
	mock.ExpectQuery(`COUNT\(\*\) FILTER \(WHERE COALESCE\(t\.stream, false\)\)::bigint`).
		WithArgs(startTime.UTC(), endTime.UTC(), platform, accountID).
		WillReturnRows(sqlmock.NewRows([]string{
			"stream_count",
			"tool_count",
			"thinking_count",
			"raw_count",
			"estimated_count",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), int64(0)))

	summary, err := repo.GetRequestTraceSummary(context.Background(), filter)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetUsageRequestPreviewReturnsLatestPreview(t *testing.T) {
	t.Parallel()

	db, mock := newOpsSQLMock(t)
	repo := &opsRepository{db: db}
	capturedAt := time.Date(2026, 4, 17, 10, 15, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT\s+COALESCE\(t\.request_id, ''\),\s+t\.created_at,.*FROM ops_request_traces t\s+WHERE t\.user_id = \$1\s+AND t\.api_key_id = \$2\s+AND COALESCE\(t\.request_id, ''\) = \$3\s+ORDER BY t\.created_at DESC, t\.id DESC\s+LIMIT 1`).
		WithArgs(int64(42), int64(8), "req-preview").
		WillReturnRows(sqlmock.NewRows([]string{
			"request_id",
			"created_at",
			"inbound_request",
			"normalized_request",
			"upstream_request",
			"upstream_response",
			"gateway_response",
			"tool_trace",
		}).AddRow(
			"req-preview",
			capturedAt,
			`{"messages":[{"role":"user"}]}`,
			"",
			`{"target":"upstream"}`,
			"",
			`{"status":"ok"}`,
			"",
		))

	preview, err := repo.GetUsageRequestPreview(context.Background(), 42, 8, "req-preview")
	require.NoError(t, err)
	require.True(t, preview.Available)
	require.NotNil(t, preview.CapturedAt)
	require.Equal(t, capturedAt, *preview.CapturedAt)
	require.Equal(t, "", preview.NormalizedRequestJSON)
	require.Equal(t, "", preview.UpstreamResponseJSON)
	require.Equal(t, "", preview.ToolTraceJSON)
	require.NoError(t, mock.ExpectationsWereMet())
}
