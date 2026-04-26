package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *affiliateRepository) AccrueTopupRebate(ctx context.Context, redeemCodeID int64, inviteeUserID int64, creditedAmount float64, policy service.AffiliateRebatePolicy) (accruedAmount float64, err error) {
	if r == nil || r.db == nil {
		return 0, errors.New("affiliate repository db is nil")
	}
	if !policy.Enabled || !policy.RebateOnTopupEnabled {
		return 0, nil
	}
	if redeemCodeID <= 0 || inviteeUserID <= 0 || creditedAmount <= 0 {
		return 0, nil
	}

	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return 0, errors.New("affiliate repository sql executor is nil")
	}
	txExec, commit, rollback, err := beginAffiliateSQLTx(ctx, exec)
	if err != nil {
		return 0, err
	}
	defer rollback()

	var inviterUserID sql.NullInt64
	var inviterBoundAt sql.NullTime
	if err := txExec.QueryRowContext(ctx, `
		SELECT inviter_user_id, inviter_bound_at
		FROM user_affiliates
		WHERE user_id = $1
	`, inviteeUserID).Scan(&inviterUserID, &inviterBoundAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	if !inviterUserID.Valid || inviterUserID.Int64 <= 0 || inviterUserID.Int64 == inviteeUserID {
		return 0, nil
	}

	now := time.Now()
	if policy.DurationDays > 0 && inviterBoundAt.Valid {
		if now.After(inviterBoundAt.Time.Add(time.Duration(policy.DurationDays) * 24 * time.Hour)) {
			return 0, nil
		}
	}

	var customRate sql.NullFloat64
	if err := txExec.QueryRowContext(ctx, `
		SELECT custom_rebate_rate_percent
		FROM user_affiliates
		WHERE user_id = $1
		FOR UPDATE
	`, inviterUserID.Int64).Scan(&customRate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	ratePercent := policy.DefaultRatePercent
	if customRate.Valid {
		ratePercent = customRate.Float64
	}
	if ratePercent < 0 {
		ratePercent = 0
	}
	if ratePercent > 100 {
		ratePercent = 100
	}
	if ratePercent <= 0 {
		return 0, nil
	}

	accruedAmount = creditedAmount * ratePercent / 100
	if accruedAmount <= 0 {
		return 0, nil
	}

	if policy.PerInviteeCap > 0 {
		var existing float64
		if err := txExec.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0)
			FROM user_affiliate_ledger
			WHERE inviter_user_id = $1
			  AND invitee_user_id = $2
			  AND event_type IN ('usage_accrue', 'topup_accrue')
		`, inviterUserID.Int64, inviteeUserID).Scan(&existing); err != nil {
			return 0, err
		}
		remaining := policy.PerInviteeCap - existing
		if remaining <= 0 {
			return 0, nil
		}
		if accruedAmount > remaining {
			accruedAmount = remaining
		}
		if accruedAmount <= 0 {
			return 0, nil
		}
	}

	var frozenUntil any
	frozenUntil = nil
	creditFrozen := false
	if policy.FreezeHours > 0 {
		creditFrozen = true
		frozenUntil = now.Add(time.Duration(policy.FreezeHours) * time.Hour)
	}

	var ledgerID int64
	err = txExec.QueryRowContext(ctx, `
		INSERT INTO user_affiliate_ledger (
			inviter_user_id,
			invitee_user_id,
			event_type,
			amount,
			base_amount,
			rate_percent,
			frozen_until,
			redeem_code_id,
			created_at
		)
		VALUES ($1, $2, 'topup_accrue', $3, $4, $5, $6, $7, NOW())
		ON CONFLICT DO NOTHING
		RETURNING id
	`, inviterUserID.Int64, inviteeUserID, accruedAmount, creditedAmount, ratePercent, frozenUntil, redeemCodeID).Scan(&ledgerID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	_ = ledgerID

	if creditFrozen {
		_, err = txExec.ExecContext(ctx, `
			UPDATE user_affiliates
			SET rebate_frozen_balance = rebate_frozen_balance + $1,
				lifetime_rebate = lifetime_rebate + $1,
				updated_at = NOW()
			WHERE user_id = $2
		`, accruedAmount, inviterUserID.Int64)
	} else {
		_, err = txExec.ExecContext(ctx, `
			UPDATE user_affiliates
			SET rebate_balance = rebate_balance + $1,
				lifetime_rebate = lifetime_rebate + $1,
				updated_at = NOW()
			WHERE user_id = $2
		`, accruedAmount, inviterUserID.Int64)
	}
	if err != nil {
		return 0, err
	}

	if err := commit(); err != nil {
		return 0, err
	}
	return accruedAmount, nil
}

func (r *affiliateRepository) ThawFrozenIfNeeded(ctx context.Context, inviterUserID int64) (thawedAmount float64, err error) {
	if r == nil || r.db == nil {
		return 0, errors.New("affiliate repository db is nil")
	}
	if inviterUserID <= 0 {
		return 0, nil
	}

	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return 0, errors.New("affiliate repository sql executor is nil")
	}
	txExec, commit, rollback, err := beginAffiliateSQLTx(ctx, exec)
	if err != nil {
		return 0, err
	}
	defer rollback()

	thawedAmount, err = thawFrozenIfNeededTx(ctx, txExec, inviterUserID)
	if err != nil {
		return 0, err
	}

	if err := commit(); err != nil {
		return 0, err
	}
	return thawedAmount, nil
}

func (r *affiliateRepository) TransferToBalance(ctx context.Context, userID int64) (*service.AffiliateTransferResult, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("affiliate repository db is nil")
	}
	if userID <= 0 {
		return &service.AffiliateTransferResult{}, nil
	}

	exec := affiliateSQLExecutorFromContext(ctx, r.db)
	if exec == nil {
		return nil, errors.New("affiliate repository sql executor is nil")
	}
	txExec, commit, rollback, err := beginAffiliateSQLTx(ctx, exec)
	if err != nil {
		return nil, err
	}
	defer rollback()

	_, _ = thawFrozenIfNeededTx(ctx, txExec, userID)

	var rebateBalance float64
	err = txExec.QueryRowContext(ctx, `
		SELECT rebate_balance
		FROM user_affiliates
		WHERE user_id = $1
		FOR UPDATE
	`, userID).Scan(&rebateBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return &service.AffiliateTransferResult{}, nil
	}
	if err != nil {
		return nil, err
	}

	result := &service.AffiliateTransferResult{}
	if rebateBalance <= 0 {
		if err := txExec.QueryRowContext(ctx, `
			SELECT balance
			FROM users
			WHERE id = $1 AND deleted_at IS NULL
		`, userID).Scan(&result.NewBalance); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, service.ErrUserNotFound
			}
			return nil, err
		}
		if err := commit(); err != nil {
			return nil, err
		}
		return result, nil
	}

	if _, err := txExec.ExecContext(ctx, `
		INSERT INTO user_affiliate_ledger (inviter_user_id, event_type, amount, created_at)
		VALUES ($1, 'transfer', $2, NOW())
	`, userID, rebateBalance); err != nil {
		return nil, err
	}

	if _, err := txExec.ExecContext(ctx, `
		UPDATE user_affiliates
		SET rebate_balance = 0,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID); err != nil {
		return nil, err
	}

	if err := txExec.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance
	`, rebateBalance, userID).Scan(&result.NewBalance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrUserNotFound
		}
		return nil, err
	}

	result.TransferredAmount = rebateBalance
	if err := commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func thawFrozenIfNeededTx(ctx context.Context, exec affiliateSQLExecutor, inviterUserID int64) (float64, error) {
	if exec == nil || inviterUserID <= 0 {
		return 0, nil
	}
	var thawedAmount float64
	err := exec.QueryRowContext(ctx, `
		WITH matured AS (
			UPDATE user_affiliate_ledger
			SET frozen_until = NULL
			WHERE inviter_user_id = $1
			  AND frozen_until IS NOT NULL
			  AND frozen_until <= NOW()
			  AND event_type IN ('usage_accrue', 'topup_accrue')
			RETURNING amount
		),
		tot AS (
			SELECT COALESCE(SUM(amount), 0) AS total
			FROM matured
		)
		UPDATE user_affiliates ua
		SET rebate_frozen_balance = GREATEST(ua.rebate_frozen_balance - tot.total, 0),
			rebate_balance = ua.rebate_balance + tot.total,
			updated_at = NOW()
		FROM tot
		WHERE ua.user_id = $1
		RETURNING tot.total
	`, inviterUserID).Scan(&thawedAmount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return thawedAmount, nil
}
