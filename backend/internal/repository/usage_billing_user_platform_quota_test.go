package repository

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageBillingRepositoryApply_UserPlatformQuotaUsageAppliedOnlyOnce(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewUsageBillingRepository(nil, db)
	cmd := &service.UsageBillingCommand{
		RequestID:        "req-platform-quota-1",
		APIKeyID:         20,
		UserID:           10,
		AccountID:        30,
		AccountType:      service.AccountTypeAPIKey,
		Platform:         service.PlatformOpenAI,
		UserPlatformCost: 0.75,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO usage_billing_dedup`).
		WithArgs(cmd.RequestID, cmd.APIKeyID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectQuery(`SELECT request_fingerprint\s+FROM usage_billing_dedup_archive`).
		WithArgs(cmd.RequestID, cmd.APIKeyID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT request_id, api_key_id, user_id, currency, hold_amount, status, COALESCE\(request_fingerprint, ''\), COALESCE\(conversion_breakdown::text, '\{\}'\), COALESCE\(conversion_policy::text, '\{\}'\)`).
		WithArgs(cmd.RequestID, cmd.APIKeyID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE user_platform_quotas`).
		WithArgs(cmd.UserID, service.PlatformOpenAI, 0.75).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.Apply(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Applied)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO usage_billing_dedup`).
		WithArgs(cmd.RequestID, cmd.APIKeyID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT request_fingerprint\s+FROM usage_billing_dedup`).
		WithArgs(cmd.RequestID, cmd.APIKeyID).
		WillReturnRows(sqlmock.NewRows([]string{"request_fingerprint"}).AddRow(cmd.RequestFingerprint))
	mock.ExpectRollback()

	replayed, err := repo.Apply(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, replayed)
	require.False(t, replayed.Applied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageBillingRepositoryApply_UserPlatformQuotaFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewUsageBillingRepository(nil, db)
	cmd := &service.UsageBillingCommand{
		RequestID:        "req-platform-quota-fail",
		APIKeyID:         21,
		UserID:           11,
		AccountID:        31,
		AccountType:      service.AccountTypeAPIKey,
		Platform:         service.PlatformOpenAI,
		UserPlatformCost: 0.25,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO usage_billing_dedup`).
		WithArgs(cmd.RequestID, cmd.APIKeyID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectQuery(`SELECT request_fingerprint\s+FROM usage_billing_dedup_archive`).
		WithArgs(cmd.RequestID, cmd.APIKeyID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT request_id, api_key_id, user_id, currency, hold_amount, status, COALESCE\(request_fingerprint, ''\), COALESCE\(conversion_breakdown::text, '\{\}'\), COALESCE\(conversion_policy::text, '\{\}'\)`).
		WithArgs(cmd.RequestID, cmd.APIKeyID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE user_platform_quotas`).
		WithArgs(cmd.UserID, service.PlatformOpenAI, 0.25).
		WillReturnError(context.DeadlineExceeded)
	mock.ExpectRollback()

	result, err := repo.Apply(context.Background(), cmd)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}
