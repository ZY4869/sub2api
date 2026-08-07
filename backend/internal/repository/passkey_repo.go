package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/passkeycredential"
	"github.com/Wei-Shaw/sub2api/ent/passkeyuserhandle"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/go-webauthn/webauthn/webauthn"
)

type passkeyRepository struct {
	client *dbent.Client
}

func NewPasskeyRepository(client *dbent.Client) service.PasskeyRepository {
	return &passkeyRepository{client: client}
}

func (r *passkeyRepository) GetOrCreateUserHandle(ctx context.Context, userID int64) ([]byte, error) {
	handle, err := r.getUserHandle(ctx, userID)
	if err == nil {
		return handle, nil
	}
	if !errors.Is(err, service.ErrPasskeyNotFound) {
		return nil, err
	}
	handle, err = service.NewPasskeyUserHandle()
	if err != nil {
		return nil, err
	}
	_, err = r.client.PasskeyUserHandle.Create().
		SetUserID(userID).
		SetUserHandle(handle).
		Save(ctx)
	if err == nil {
		return handle, nil
	}
	if dbent.IsConstraintError(err) {
		return r.getUserHandle(ctx, userID)
	}
	return nil, err
}

func (r *passkeyRepository) GetUserIDByHandle(ctx context.Context, handle []byte) (int64, error) {
	row, err := r.client.PasskeyUserHandle.Query().
		Where(passkeyuserhandle.UserHandleEQ(handle)).
		Only(ctx)
	if err != nil {
		return 0, translatePersistenceError(err, service.ErrPasskeyNotFound, nil)
	}
	return row.UserID, nil
}

func (r *passkeyRepository) ListByUser(ctx context.Context, userID int64) ([]service.PasskeyCredential, error) {
	rows, err := r.client.PasskeyCredential.Query().
		Where(passkeycredential.UserIDEQ(userID)).
		Order(dbent.Desc(passkeycredential.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.PasskeyCredential, 0, len(rows))
	for _, row := range rows {
		item, err := passkeyCredentialEntityToService(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (r *passkeyRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*service.PasskeyCredential, error) {
	row, err := r.client.PasskeyCredential.Query().
		Where(passkeycredential.IDEQ(id), passkeycredential.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrPasskeyNotFound, nil)
	}
	return passkeyCredentialEntityToService(row)
}

func (r *passkeyRepository) GetByCredentialID(ctx context.Context, credentialID []byte) (*service.PasskeyCredential, error) {
	row, err := r.client.PasskeyCredential.Query().
		Where(passkeycredential.CredentialIDEQ(credentialID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrPasskeyNotFound, nil)
	}
	return passkeyCredentialEntityToService(row)
}

func (r *passkeyRepository) CreateCredential(ctx context.Context, credential *service.PasskeyCredential) error {
	if credential == nil {
		return nil
	}
	data, err := service.MarshalPasskeyCredentialData(credential.Credential)
	if err != nil {
		return fmt.Errorf("marshal passkey credential: %w", err)
	}
	row, err := r.client.PasskeyCredential.Create().
		SetUserID(credential.UserID).
		SetCredentialID(credential.CredentialID).
		SetName(credential.Name).
		SetCredentialData(data).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrPasskeyExists)
	}
	credential.ID = row.ID
	credential.CreatedAt = row.CreatedAt
	credential.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *passkeyRepository) UpdateCredential(ctx context.Context, credentialID []byte, credential webauthn.Credential, lastUsedAt time.Time) error {
	data, err := service.MarshalPasskeyCredentialData(credential)
	if err != nil {
		return fmt.Errorf("marshal passkey credential: %w", err)
	}
	n, err := r.client.PasskeyCredential.Update().
		Where(passkeycredential.CredentialIDEQ(credentialID)).
		SetCredentialData(data).
		SetLastUsedAt(lastUsedAt).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrPasskeyNotFound
	}
	return nil
}

func (r *passkeyRepository) UpdateName(ctx context.Context, id, userID int64, name string) error {
	n, err := r.client.PasskeyCredential.Update().
		Where(passkeycredential.IDEQ(id), passkeycredential.UserIDEQ(userID)).
		SetName(name).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrPasskeyNotFound
	}
	return nil
}

func (r *passkeyRepository) Delete(ctx context.Context, id, userID int64) error {
	n, err := r.client.PasskeyCredential.Delete().
		Where(passkeycredential.IDEQ(id), passkeycredential.UserIDEQ(userID)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrPasskeyNotFound
	}
	return nil
}

func (r *passkeyRepository) getUserHandle(ctx context.Context, userID int64) ([]byte, error) {
	row, err := r.client.PasskeyUserHandle.Query().
		Where(passkeyuserhandle.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrPasskeyNotFound, nil)
	}
	return row.UserHandle, nil
}

func passkeyCredentialEntityToService(row *dbent.PasskeyCredential) (*service.PasskeyCredential, error) {
	if row == nil {
		return nil, service.ErrPasskeyNotFound
	}
	credential, err := service.UnmarshalPasskeyCredentialData(row.CredentialData)
	if err != nil {
		return nil, fmt.Errorf("unmarshal passkey credential: %w", err)
	}
	return &service.PasskeyCredential{
		ID:           row.ID,
		UserID:       row.UserID,
		CredentialID: row.CredentialID,
		Name:         row.Name,
		Credential:   credential,
		LastUsedAt:   row.LastUsedAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}
