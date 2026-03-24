package repository

import (
	"context"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestSetAPIKeyGroups_FallsBackToLegacyShadowWhenSchemaMissing(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &apiKeyRepository{sql: db}

	mock.ExpectQuery(`SELECT to_regclass\('public\.' \|\| \$1\) IS NOT NULL`).
		WithArgs("api_key_groups").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE api_keys`).
		WithArgs(int64(42), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SetAPIKeyGroups(context.Background(), 42, []service.APIKeyGroupBinding{
		{GroupID: 7},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetAPIKeyGroups_ReturnsActionableErrorWhenLegacySchemaCannotRepresentBindings(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &apiKeyRepository{sql: db}

	mock.ExpectQuery(`SELECT to_regclass\('public\.' \|\| \$1\) IS NOT NULL`).
		WithArgs("api_key_groups").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err := repo.SetAPIKeyGroups(context.Background(), 42, []service.APIKeyGroupBinding{
		{GroupID: 7, Quota: 1},
	})
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, http.StatusServiceUnavailable, int(appErr.Code))
	require.Equal(t, "API_KEY_GROUP_SCHEMA_OUTDATED", appErr.Reason)
	require.Contains(t, appErr.Message, "latest database migration")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadAPIKeyGroupBindingsMap_IgnoresMissingSchemaError(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &apiKeyRepository{}

	mock.ExpectQuery(`SELECT api_key_id, group_id, quota, quota_used, model_patterns, created_at, updated_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(&pq.Error{
			Code:    "42P01",
			Message: `relation "api_key_groups" does not exist`,
		})

	groupMap, err := repo.loadAPIKeyGroupBindingsMap(context.Background(), db, []int64{42})
	require.NoError(t, err)
	require.Empty(t, groupMap)
	require.NoError(t, mock.ExpectationsWereMet())
}
