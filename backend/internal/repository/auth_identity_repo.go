package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type authIdentityRepository struct {
	sql *sql.DB
}

func NewAuthIdentityRepository(sqlDB *sql.DB) service.AuthIdentityRepository {
	return &authIdentityRepository{sql: sqlDB}
}

func (r *authIdentityRepository) Create(ctx context.Context, identity *service.AuthIdentity) error {
	if r == nil || r.sql == nil || identity == nil {
		return errors.New("auth identity repository not configured")
	}
	now := time.Now()
	identity.Provider = strings.TrimSpace(identity.Provider)
	identity.ProviderUserID = strings.TrimSpace(identity.ProviderUserID)
	identity.Email = strings.TrimSpace(identity.Email)
	identity.DisplayName = strings.TrimSpace(identity.DisplayName)
	identity.AvatarURL = strings.TrimSpace(identity.AvatarURL)
	query := `
INSERT INTO auth_identities (
  provider,
  provider_user_id,
  user_id,
  email,
  email_verified,
  display_name,
  avatar_url,
  created_at,
  updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING id, created_at, updated_at`
	return r.sql.QueryRowContext(
		ctx,
		query,
		identity.Provider,
		identity.ProviderUserID,
		identity.UserID,
		identity.Email,
		identity.EmailVerified,
		identity.DisplayName,
		identity.AvatarURL,
		now,
		now,
	).Scan(&identity.ID, &identity.CreatedAt, &identity.UpdatedAt)
}

func (r *authIdentityRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*service.AuthIdentity, error) {
	return r.getOne(
		ctx,
		`SELECT id, provider, provider_user_id, user_id, email, email_verified, display_name, avatar_url, created_at, updated_at
		 FROM auth_identities
		 WHERE provider = $1 AND provider_user_id = $2`,
		strings.TrimSpace(provider),
		strings.TrimSpace(providerUserID),
	)
}

func (r *authIdentityRepository) GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*service.AuthIdentity, error) {
	return r.getOne(
		ctx,
		`SELECT id, provider, provider_user_id, user_id, email, email_verified, display_name, avatar_url, created_at, updated_at
		 FROM auth_identities
		 WHERE user_id = $1 AND provider = $2`,
		userID,
		strings.TrimSpace(provider),
	)
}

func (r *authIdentityRepository) ListByUserID(ctx context.Context, userID int64) ([]*service.AuthIdentity, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("auth identity repository not configured")
	}
	rows, err := r.sql.QueryContext(
		ctx,
		`SELECT id, provider, provider_user_id, user_id, email, email_verified, display_name, avatar_url, created_at, updated_at
		 FROM auth_identities
		 WHERE user_id = $1
		 ORDER BY provider ASC, id ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*service.AuthIdentity, 0)
	for rows.Next() {
		item := &service.AuthIdentity{}
		if scanErr := rows.Scan(
			&item.ID,
			&item.Provider,
			&item.ProviderUserID,
			&item.UserID,
			&item.Email,
			&item.EmailVerified,
			&item.DisplayName,
			&item.AvatarURL,
			&item.CreatedAt,
			&item.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *authIdentityRepository) DeleteByUserIDAndProvider(ctx context.Context, userID int64, provider string) error {
	if r == nil || r.sql == nil {
		return errors.New("auth identity repository not configured")
	}
	res, err := r.sql.ExecContext(
		ctx,
		`DELETE FROM auth_identities WHERE user_id = $1 AND provider = $2`,
		userID,
		strings.TrimSpace(provider),
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAuthIdentityNotFound
	}
	return nil
}

func (r *authIdentityRepository) getOne(ctx context.Context, query string, args ...any) (*service.AuthIdentity, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("auth identity repository not configured")
	}
	item := &service.AuthIdentity{}
	err := r.sql.QueryRowContext(ctx, query, args...).Scan(
		&item.ID,
		&item.Provider,
		&item.ProviderUserID,
		&item.UserID,
		&item.Email,
		&item.EmailVerified,
		&item.DisplayName,
		&item.AvatarURL,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAuthIdentityNotFound
		}
		return nil, err
	}
	return item, nil
}
