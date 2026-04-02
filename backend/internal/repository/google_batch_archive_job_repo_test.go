package repository

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGoogleBatchArchiveJobRepositoryTryMarkBillingSettled(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	repo := &googleBatchArchiveJobRepository{sql: db}
	query := regexp.QuoteMeta(`
		UPDATE google_batch_archive_jobs
		SET billing_settlement_state = $2, updated_at = NOW()
		WHERE id = $1
			AND billing_settlement_state = $3
			AND deleted_at IS NULL
	`)

	mock.ExpectExec(query).
		WithArgs(int64(42), "settled", "pending").
		WillReturnResult(sqlmock.NewResult(0, 1))

	applied, err := repo.TryMarkBillingSettled(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, applied)

	mock.ExpectExec(query).
		WithArgs(int64(42), "settled", "pending").
		WillReturnResult(sqlmock.NewResult(0, 0))

	applied, err = repo.TryMarkBillingSettled(context.Background(), 42)
	require.NoError(t, err)
	require.False(t, applied)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGoogleBatchArchiveJobRepositoryTryRestoreBillingPending(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	repo := &googleBatchArchiveJobRepository{sql: db}
	query := regexp.QuoteMeta(`
		UPDATE google_batch_archive_jobs
		SET billing_settlement_state = $2, updated_at = NOW()
		WHERE id = $1
			AND billing_settlement_state = $3
			AND deleted_at IS NULL
	`)

	mock.ExpectExec(query).
		WithArgs(int64(42), "pending", "settled").
		WillReturnResult(sqlmock.NewResult(0, 1))

	applied, err := repo.TryRestoreBillingPending(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, applied)

	mock.ExpectExec(query).
		WithArgs(int64(42), "pending", "settled").
		WillReturnResult(sqlmock.NewResult(0, 0))

	applied, err = repo.TryRestoreBillingPending(context.Background(), 42)
	require.NoError(t, err)
	require.False(t, applied)

	require.NoError(t, mock.ExpectationsWereMet())
}
