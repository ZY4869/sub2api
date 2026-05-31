package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryPublicModelCatalogTrafficHealthMatchesPublicID(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"}).
		AddRow(int64(77), "public-model-2", "real-model-1", "openai", service.UsageLogStatusSucceeded, sql.NullInt64{Int64: 123, Valid: true}, now)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(rows)

	statuses, err := repo.PublicModelCatalogTrafficHealth(context.Background(), []service.PublicModelCatalogItem{{
		Model:           "public-model-2",
		PublicModelID:   "public-model-2",
		SourceModelID:   "real-model-1",
		SourceAccountID: 42,
		SourceProtocol:  service.PlatformOpenAI,
	}}, now.AddDate(0, 0, -6), now)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	status := statuses["public-model-2"]
	require.Equal(t, service.PublicModelHealthSourceTraffic, status.HealthSource)
	require.Equal(t, service.PublicModelHealthReasonTrafficRecent, status.StatusReason)
	require.Equal(t, service.PublicModelHealthStatusHealthy, status.Status)
	require.NotNil(t, status.SuccessRate7d)
	require.Equal(t, float64(1), *status.SuccessRate7d)
	require.Equal(t, int64(123), *status.LatencyMs)
}

func TestUsageLogRepositoryPublicModelCatalogTrafficHealthSourceFallbackRequiresAccountAndProtocol(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"}).
		AddRow(int64(99), "unknown-public", "real-model-1", "openai", service.UsageLogStatusFailed, sql.NullInt64{}, now).
		AddRow(int64(42), "unknown-public", "real-model-1", "anthropic", service.UsageLogStatusFailed, sql.NullInt64{}, now).
		AddRow(int64(42), "unknown-public", "real-model-1", "openai", service.UsageLogStatusSucceeded, sql.NullInt64{Int64: 240, Valid: true}, now)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(rows)

	statuses, err := repo.PublicModelCatalogTrafficHealth(context.Background(), []service.PublicModelCatalogItem{{
		Model:           "public-model-2",
		PublicModelID:   "public-model-2",
		SourceModelID:   "real-model-1",
		SourceAccountID: 42,
		SourceProtocol:  service.PlatformOpenAI,
	}}, now.AddDate(0, 0, -6), now)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	status := statuses["public-model-2"]
	require.Equal(t, service.PublicModelHealthSourceTraffic, status.HealthSource)
	require.Equal(t, service.PublicModelHealthStatusHealthy, status.Status)
	require.NotNil(t, status.SuccessRate7d)
	require.Equal(t, float64(1), *status.SuccessRate7d)
	require.NotNil(t, status.SuccessRateToday)
	require.Equal(t, float64(1), *status.SuccessRateToday)
}

func TestUsageLogRepositoryPublicModelCatalogTrafficHealthDoesNotTreatLegacyModelAsPublicAlias(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"}).
		AddRow(int64(99), "real-model-1", "real-model-1", "openai", service.UsageLogStatusFailed, sql.NullInt64{}, now).
		AddRow(int64(42), "real-model-1", "real-model-1", "openai", service.UsageLogStatusSucceeded, sql.NullInt64{Int64: 180, Valid: true}, now)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(rows)

	statuses, err := repo.PublicModelCatalogTrafficHealth(context.Background(), []service.PublicModelCatalogItem{{
		Model:           "real-model-1",
		PublicModelID:   "public-model-2",
		SourceModelID:   "real-model-1",
		SourceAccountID: 42,
		SourceProtocol:  service.PlatformOpenAI,
	}}, now.AddDate(0, 0, -6), now)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	status := statuses["public-model-2"]
	require.Equal(t, service.PublicModelHealthSourceTraffic, status.HealthSource)
	require.NotNil(t, status.SuccessRate7d)
	require.Equal(t, float64(1), *status.SuccessRate7d)
	require.Equal(t, []string{"public-model-2"}, status.Aliases)
}

func TestUsageLogRepositoryPublicModelCatalogTrafficHealthTodayFailureMarksError(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	yesterday := now.AddDate(0, 0, -1)
	rows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"}).
		AddRow(int64(42), "public-model-2", "public-model-2", "openai", service.UsageLogStatusSucceeded, sql.NullInt64{Int64: 100, Valid: true}, yesterday).
		AddRow(int64(42), "public-model-2", "public-model-2", "openai", service.UsageLogStatusFailed, sql.NullInt64{}, now)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(rows)

	statuses, err := repo.PublicModelCatalogTrafficHealth(context.Background(), []service.PublicModelCatalogItem{{
		Model:         "public-model-2",
		PublicModelID: "public-model-2",
	}}, now.AddDate(0, 0, -6), now)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	status := statuses["public-model-2"]
	require.Equal(t, service.PublicModelHealthStatusError, status.Status)
	require.NotNil(t, status.SuccessRateToday)
	require.Equal(t, float64(0), *status.SuccessRateToday)
}

func TestUsageLogRepositoryPublicModelCatalogTrafficHealthStaleHistory(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	oldCheckedAt := now.AddDate(0, 0, -10)
	recentRows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"})
	staleRows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"}).
		AddRow(int64(42), "public-model-2", "real-model-1", "openai", service.UsageLogStatusSucceeded, sql.NullInt64{Int64: 999, Valid: true}, oldCheckedAt)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(recentRows)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(staleRows)

	statuses, err := repo.PublicModelCatalogTrafficHealth(context.Background(), []service.PublicModelCatalogItem{{
		Model:           "public-model-2",
		PublicModelID:   "public-model-2",
		SourceModelID:   "real-model-1",
		SourceAccountID: 42,
		SourceProtocol:  service.PlatformOpenAI,
	}}, now.AddDate(0, 0, -6), now)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	status := statuses["public-model-2"]
	require.Equal(t, service.PublicModelHealthSourceNone, status.HealthSource)
	require.Equal(t, service.PublicModelHealthReasonStaleHistory, status.StatusReason)
	require.Equal(t, service.PublicModelHealthStatusPending, status.Status)
	require.Equal(t, oldCheckedAt.Format(time.RFC3339), status.LastCheckedAt)
	require.Nil(t, status.SuccessRateToday)
	require.Nil(t, status.SuccessRate7d)
	require.Nil(t, status.LatencyMs)
	require.Empty(t, status.Daily)
	require.Empty(t, status.Trend)
}

func TestUsageLogRepositoryPublicModelCatalogTrafficHealthNoHistory(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	recentRows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"})
	staleRows := sqlmock.NewRows([]string{"account_id", "requested_model", "upstream_model", "protocol", "status", "duration_ms", "created_at"})
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(recentRows)
	mock.ExpectQuery(regexp.QuoteMeta("FROM usage_logs")).
		WillReturnRows(staleRows)

	statuses, err := repo.PublicModelCatalogTrafficHealth(context.Background(), []service.PublicModelCatalogItem{{
		Model:         "public-model-2",
		PublicModelID: "public-model-2",
	}}, now.AddDate(0, 0, -6), now)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	status := statuses["public-model-2"]
	require.Equal(t, service.PublicModelHealthSourceNone, status.HealthSource)
	require.Equal(t, service.PublicModelHealthReasonNoHistory, status.StatusReason)
	require.Equal(t, service.PublicModelHealthStatusPending, status.Status)
	require.Empty(t, status.LastCheckedAt)
	require.Nil(t, status.SuccessRateToday)
	require.Nil(t, status.SuccessRate7d)
	require.Nil(t, status.LatencyMs)
	require.Empty(t, status.Daily)
	require.Empty(t, status.Trend)
}
