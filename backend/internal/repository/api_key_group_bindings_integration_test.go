//go:build integration

package repository

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSetAPIKeyGroups_UsesEntTransactionExecutor(t *testing.T) {
	tx := testEntTx(t)
	repo := newAPIKeyRepositoryWithSQL(tx.Client(), integrationDB)
	ctx := dbent.NewTxContext(context.Background(), tx)

	userEnt, err := tx.Client().User.Create().
		SetEmail("tx-apikey-groups@test.com").
		SetPasswordHash("hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err)

	groupEnt, err := tx.Client().Group.Create().
		SetName("tx-group-bindings").
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	key := &service.APIKey{
		UserID: userEnt.ID,
		Key:    "sk-tx-group-bindings",
		Name:   "Tx Key",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, key))

	err = repo.SetAPIKeyGroups(ctx, key.ID, []service.APIKeyGroupBinding{{
		APIKeyID: key.ID,
		GroupID:  groupEnt.ID,
	}})
	require.NoError(t, err)

	bindings, err := repo.GetAPIKeyGroups(ctx, key.ID)
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	require.Equal(t, key.ID, bindings[0].APIKeyID)
	require.Equal(t, groupEnt.ID, bindings[0].GroupID)
}
