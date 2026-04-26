package repository

import (
	"context"
	"database/sql"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type affiliateRepository struct {
	db *sql.DB
}

func NewAffiliateRepository(_ *dbent.Client, sqlDB *sql.DB) service.AffiliateRepository {
	return &affiliateRepository{db: sqlDB}
}

func (r *affiliateRepository) GetUserAffiliate(ctx context.Context, userID int64) (*service.UserAffiliate, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("affiliate repository db is nil")
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	return getUserAffiliateWithExec(ctx, exec, userID)
}

func (r *affiliateRepository) EnsureAffiliateRow(ctx context.Context, userID int64, affCode string) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("affiliate repository db is nil")
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return false, errors.New("affiliate repository sql executor is nil")
	}

	var insertedUserID int64
	err := exec.QueryRowContext(ctx, `
		INSERT INTO user_affiliates (user_id, aff_code, custom_aff_code, created_at, updated_at)
		VALUES ($1, $2, false, NOW(), NOW())
		ON CONFLICT (user_id) DO NOTHING
		RETURNING user_id
	`, userID, affCode).Scan(&insertedUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func getUserAffiliateWithExec(ctx context.Context, exec affiliateSQLExecutor, userID int64) (*service.UserAffiliate, error) {
	if exec == nil {
		return nil, errors.New("affiliate sql executor is nil")
	}

	var inviterUserID sql.NullInt64
	var inviterBoundAt sql.NullTime
	var customRate sql.NullFloat64

	row := &service.UserAffiliate{}
	err := exec.QueryRowContext(ctx, `
		SELECT
			user_id,
			aff_code,
			inviter_user_id,
			inviter_bound_at,
			invitee_count,
			rebate_balance,
			rebate_frozen_balance,
			lifetime_rebate,
			custom_rebate_rate_percent,
			custom_aff_code,
			created_at,
			updated_at
		FROM user_affiliates
		WHERE user_id = $1
	`, userID).Scan(
		&row.UserID,
		&row.AffCode,
		&inviterUserID,
		&inviterBoundAt,
		&row.InviteeCount,
		&row.RebateBalance,
		&row.RebateFrozenBalance,
		&row.LifetimeRebate,
		&customRate,
		&row.CustomAffCode,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if inviterUserID.Valid {
		row.InviterUserID = &inviterUserID.Int64
	}
	if inviterBoundAt.Valid {
		row.InviterBoundAt = &inviterBoundAt.Time
	}
	if customRate.Valid {
		row.CustomRebateRatePercent = &customRate.Float64
	}
	return row, nil
}
