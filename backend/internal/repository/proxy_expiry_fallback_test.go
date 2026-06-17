package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newProxyRepoSQLite(t *testing.T) (*proxyRepository, *dbent.Client, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:proxy_expiry_fallback?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return newProxyRepositoryWithSQL(client, db), client, db
}

func TestProxyRepositoryListExpiredProxiesOnlyReturnsActiveUndeletedExpired(t *testing.T) {
	repo, client, _ := newProxyRepoSQLite(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Minute)
	futureAt := now.Add(time.Minute)

	expired, err := client.Proxy.Create().
		SetName("expired").
		SetProtocol("http").
		SetHost("127.0.0.1").
		SetPort(8080).
		SetStatus(service.StatusActive).
		SetExpiresAt(expiredAt).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Proxy.Create().
		SetName("future").
		SetProtocol("http").
		SetHost("127.0.0.2").
		SetPort(8081).
		SetStatus(service.StatusActive).
		SetExpiresAt(futureAt).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Proxy.Create().
		SetName("inactive-expired").
		SetProtocol("http").
		SetHost("127.0.0.3").
		SetPort(8082).
		SetStatus(service.StatusDisabled).
		SetExpiresAt(expiredAt).
		Save(ctx)
	require.NoError(t, err)

	deleted, err := client.Proxy.Create().
		SetName("deleted-expired").
		SetProtocol("http").
		SetHost("127.0.0.4").
		SetPort(8083).
		SetStatus(service.StatusActive).
		SetExpiresAt(expiredAt).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.Proxy.DeleteOneID(deleted.ID).Exec(ctx))

	proxies, err := repo.ListExpiredProxies(ctx, now)
	require.NoError(t, err)
	require.Len(t, proxies, 1)
	require.Equal(t, expired.ID, proxies[0].ID)
	require.Equal(t, "expired", proxies[0].Name)
}

func TestAccountRepositorySwitchExpiredProxyAccountsRecordsOriginalProxy(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountRepository{sql: db}
	switchedAt := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	expired := service.Proxy{ID: 10, Name: "expired-proxy"}
	fallback := service.Proxy{ID: 20, Name: "fallback-proxy"}

	mock.ExpectQuery("UPDATE accounts").
		WithArgs(expired.ID, fallback.ID, expired.Name).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(101)).AddRow(int64(102)))
	mock.ExpectExec("INSERT INTO scheduler_outbox").
		WithArgs(service.SchedulerOutboxEventAccountBulkChanged, nil, nil, sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ids, err := repo.SwitchExpiredProxyAccounts(context.Background(), expired, fallback, switchedAt)
	require.NoError(t, err)
	require.Equal(t, []int64{101, 102}, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepositorySwitchExpiredProxyAccountsSkipsInvalidProxyIDs(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountRepository{sql: db}

	ids, err := repo.SwitchExpiredProxyAccounts(context.Background(), service.Proxy{ID: 10}, service.Proxy{ID: 10}, time.Now())
	require.NoError(t, err)
	require.Empty(t, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepositoryRestoreAccountOriginalProxySuccess(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountRepository{sql: db}
	accountID := int64(101)

	mock.ExpectQuery("WITH current_account").
		WithArgs(accountID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"original_id",
			"original_name",
			"fallback_id",
			"fallback_name",
		}).AddRow(accountID, int64(10), "original-proxy", int64(20), "fallback-proxy"))
	mock.ExpectExec("INSERT INTO scheduler_outbox").
		WithArgs(service.SchedulerOutboxEventAccountChanged, sqlmock.AnyArg(), nil, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := repo.RestoreAccountOriginalProxy(context.Background(), accountID)
	require.NoError(t, err)
	require.Equal(t, accountID, result.AccountID)
	require.Equal(t, int64(10), result.RestoredProxyID)
	require.Equal(t, "original-proxy", result.RestoredProxyName)
	require.NotNil(t, result.PreviousFallbackID)
	require.Equal(t, int64(20), *result.PreviousFallbackID)
	require.Equal(t, "fallback-proxy", result.PreviousFallbackName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepositoryRestoreAccountOriginalProxyUnavailable(t *testing.T) {
	tests := []string{
		"account has no original proxy record",
		"original proxy is missing or deleted",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			repo := &accountRepository{sql: db}
			accountID := int64(101)

			mock.ExpectQuery("WITH current_account").
				WithArgs(accountID).
				WillReturnRows(sqlmock.NewRows([]string{
					"id",
					"original_id",
					"original_name",
					"fallback_id",
					"fallback_name",
				}))
			mock.ExpectQuery("SELECT 1").
				WithArgs(accountID).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))

			result, err := repo.RestoreAccountOriginalProxy(context.Background(), accountID)
			require.Nil(t, result)
			require.Error(t, err)
			require.True(t, errors.Is(err, service.ErrProxyOriginalNotFound))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAccountRepositoryRestoreAccountOriginalProxyAccountNotFound(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountRepository{sql: db}
	accountID := int64(404)

	mock.ExpectQuery("WITH current_account").
		WithArgs(accountID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"original_id",
			"original_name",
			"fallback_id",
			"fallback_name",
		}))
	mock.ExpectQuery("SELECT 1").
		WithArgs(accountID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}))

	result, err := repo.RestoreAccountOriginalProxy(context.Background(), accountID)
	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, errors.Is(err, service.ErrAccountNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}
