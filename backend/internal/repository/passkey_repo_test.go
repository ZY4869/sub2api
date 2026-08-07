package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newPasskeyRepoSQLite(t *testing.T) (service.PasskeyRepository, *dbent.Client) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()))
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return NewPasskeyRepository(client), client
}

func mustCreatePasskeyRepoUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *service.User {
	t.Helper()

	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return userEntityToService(user)
}

func TestPasskeyRepositoryUserHandleIsIdempotent(t *testing.T) {
	repo, client := newPasskeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreatePasskeyRepoUser(t, ctx, client, "passkey-handle@test.com")

	first, err := repo.GetOrCreateUserHandle(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, first, 32)

	second, err := repo.GetOrCreateUserHandle(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, first, second)

	userID, err := repo.GetUserIDByHandle(ctx, first)
	require.NoError(t, err)
	require.Equal(t, user.ID, userID)
}

func TestPasskeyRepositoryListRenameAndDeleteEnforceOwnership(t *testing.T) {
	repo, client := newPasskeyRepoSQLite(t)
	ctx := context.Background()
	owner := mustCreatePasskeyRepoUser(t, ctx, client, "passkey-owner@test.com")
	other := mustCreatePasskeyRepoUser(t, ctx, client, "passkey-other@test.com")

	credential := &service.PasskeyCredential{
		UserID:       owner.ID,
		CredentialID: []byte("credential-owner"),
		Name:         "Owner key",
		Credential:   webauthn.Credential{ID: []byte("credential-owner"), PublicKey: []byte("public-key")},
	}
	require.NoError(t, repo.CreateCredential(ctx, credential))
	require.NoError(t, repo.CreateCredential(ctx, &service.PasskeyCredential{
		UserID:       other.ID,
		CredentialID: []byte("credential-other"),
		Name:         "Other key",
		Credential:   webauthn.Credential{ID: []byte("credential-other"), PublicKey: []byte("public-key")},
	}))

	ownerItems, err := repo.ListByUser(ctx, owner.ID)
	require.NoError(t, err)
	require.Len(t, ownerItems, 1)
	require.Equal(t, "Owner key", ownerItems[0].Name)

	_, err = repo.GetByIDForUser(ctx, credential.ID, other.ID)
	require.ErrorIs(t, err, service.ErrPasskeyNotFound)
	require.ErrorIs(t, repo.UpdateName(ctx, credential.ID, other.ID, "Stolen key"), service.ErrPasskeyNotFound)

	unchanged, err := repo.GetByIDForUser(ctx, credential.ID, owner.ID)
	require.NoError(t, err)
	require.Equal(t, "Owner key", unchanged.Name)

	require.ErrorIs(t, repo.Delete(ctx, credential.ID, other.ID), service.ErrPasskeyNotFound)
	require.NoError(t, repo.Delete(ctx, credential.ID, owner.ID))
	_, err = repo.GetByIDForUser(ctx, credential.ID, owner.ID)
	require.ErrorIs(t, err, service.ErrPasskeyNotFound)
}

func TestPasskeyRepositoryUpdateCredentialStoresLastUsedAt(t *testing.T) {
	repo, client := newPasskeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreatePasskeyRepoUser(t, ctx, client, "passkey-last-used@test.com")
	credentialID := []byte("credential-last-used")
	item := &service.PasskeyCredential{
		UserID:       user.ID,
		CredentialID: credentialID,
		Name:         "Laptop",
		Credential:   webauthn.Credential{ID: credentialID, PublicKey: []byte("old-key")},
	}
	require.NoError(t, repo.CreateCredential(ctx, item))

	lastUsedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	updatedCredential := webauthn.Credential{ID: credentialID, PublicKey: []byte("new-key")}
	require.NoError(t, repo.UpdateCredential(ctx, credentialID, updatedCredential, lastUsedAt))

	got, err := repo.GetByCredentialID(ctx, credentialID)
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
	require.WithinDuration(t, lastUsedAt, *got.LastUsedAt, time.Second)
	require.Equal(t, []byte("new-key"), got.Credential.PublicKey)
}
