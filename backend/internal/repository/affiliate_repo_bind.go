package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (r *affiliateRepository) BindInviterByCode(ctx context.Context, inviteeUserID int64, affCode string) (inviterUserID int64, bound bool, err error) {
	if r == nil || r.db == nil {
		return 0, false, errors.New("affiliate repository db is nil")
	}
	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return 0, false, errors.New("affiliate repository sql executor is nil")
	}
	txExec, commit, rollback, err := beginAffiliateSQLTx(ctx, exec)
	if err != nil {
		return 0, false, err
	}
	defer rollback()

	err = txExec.QueryRowContext(ctx, `
		SELECT user_id
		FROM user_affiliates
		WHERE aff_code = $1
		LIMIT 1
	`, affCode).Scan(&inviterUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if inviterUserID == inviteeUserID {
		return inviterUserID, false, nil
	}

	res, err := txExec.ExecContext(ctx, `
		UPDATE user_affiliates
		SET inviter_user_id = $1,
			inviter_bound_at = NOW(),
			updated_at = NOW()
		WHERE user_id = $2
		  AND inviter_user_id IS NULL
	`, inviterUserID, inviteeUserID)
	if err != nil {
		return 0, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, false, err
	}
	if affected == 0 {
		if err := commit(); err != nil {
			return 0, false, err
		}
		return inviterUserID, false, nil
	}

	res, err = txExec.ExecContext(ctx, `
		UPDATE user_affiliates
		SET invitee_count = invitee_count + 1,
			updated_at = NOW()
		WHERE user_id = $1
	`, inviterUserID)
	if err != nil {
		return 0, false, err
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return 0, false, err
	}
	if affected == 0 {
		return 0, false, fmt.Errorf("inviter affiliate row not found")
	}

	if err := commit(); err != nil {
		return 0, false, err
	}
	return inviterUserID, true, nil
}
