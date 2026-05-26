package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *paymentRepository) AddWalletBalance(ctx context.Context, userID int64, currency string, amount float64) error {
	currency = service.NormalizePaymentCurrency(currency)
	normalized, err := service.NormalizeAndValidateBillingAmount(amount)
	if err != nil {
		return err
	}
	amountMoney, err := service.NewBillingMoneyFromFloat(normalized)
	if err != nil {
		return err
	}
	if currency == "" || amountMoney.IsZero() {
		return nil
	}
	exec := paymentExec(ctx, r.db)
	tx, ok := exec.(*sql.Tx)
	if !ok {
		var err error
		tx, err = r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		if err := addWalletBalanceTx(ctx, tx, userID, currency, amountMoney); err != nil {
			return err
		}
		return tx.Commit()
	}
	return addWalletBalanceTx(ctx, tx, userID, currency, amountMoney)
}

func (r *paymentRepository) AssignOrExtendSubscription(ctx context.Context, input *service.AssignSubscriptionInput) error {
	if r == nil || input == nil {
		return nil
	}
	exec := paymentExec(ctx, r.db)
	tx, ok := exec.(*sql.Tx)
	if !ok {
		var err error
		tx, err = r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		if err := assignOrExtendSubscriptionTx(ctx, tx, input); err != nil {
			return err
		}
		return tx.Commit()
	}
	return assignOrExtendSubscriptionTx(ctx, tx, input)
}

func assignOrExtendSubscriptionTx(ctx context.Context, exec sqlExecutor, input *service.AssignSubscriptionInput) error {
	var subscriptionType string
	if err := scanSingleRow(ctx, exec, `
		SELECT subscription_type
		FROM groups
		WHERE id = $1 AND deleted_at IS NULL
	`, []any{input.GroupID}, &subscriptionType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrGroupNotSubscriptionType
		}
		return err
	}
	if subscriptionType != service.SubscriptionTypeSubscription {
		return service.ErrGroupNotSubscriptionType
	}

	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > service.MaxValidityDays {
		validityDays = service.MaxValidityDays
	}

	var existing struct {
		ID        int64
		ExpiresAt time.Time
		Status    string
		Notes     sql.NullString
	}
	err := scanSingleRow(ctx, exec, `
		SELECT id, expires_at, status, notes
		FROM user_subscriptions
		WHERE user_id = $1 AND group_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`, []any{input.UserID, input.GroupID}, &existing.ID, &existing.ExpiresAt, &existing.Status, &existing.Notes)
	now := time.Now()
	if err == nil {
		base := now
		if existing.ExpiresAt.After(now) {
			base = existing.ExpiresAt
		}
		expiresAt := base.AddDate(0, 0, validityDays)
		if expiresAt.After(service.MaxExpiresAt) {
			expiresAt = service.MaxExpiresAt
		}
		notes := existing.Notes.String
		if strings.TrimSpace(input.Notes) != "" {
			if notes != "" {
				notes += "\n"
			}
			notes += input.Notes
		}
		_, err = exec.ExecContext(ctx, `
			UPDATE user_subscriptions
			SET expires_at = $2, status = $3, notes = $4, updated_at = NOW()
			WHERE id = $1
		`, existing.ID, expiresAt, service.SubscriptionStatusActive, notes)
		return err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	expiresAt := now.AddDate(0, 0, validityDays)
	if expiresAt.After(service.MaxExpiresAt) {
		expiresAt = service.MaxExpiresAt
	}
	var assignedBy any
	if input.AssignedBy > 0 {
		assignedBy = input.AssignedBy
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO user_subscriptions (
			user_id, group_id, starts_at, expires_at, status, assigned_by, assigned_at, notes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7, NOW(), NOW())
	`, input.UserID, input.GroupID, now, expiresAt, service.SubscriptionStatusActive, assignedBy, input.Notes)
	return err
}

func addWalletBalanceTx(ctx context.Context, exec sqlExecutor, userID int64, currency string, amountMoney service.BillingMoney) error {
	if currency == service.ModelPricingCurrencyUSD {
		result, err := exec.ExecContext(ctx, `
			UPDATE users
			SET balance = balance + $2, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, userID, amountMoney.DBValue())
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return service.ErrUserNotFound
		}
	}
	_, err := exec.ExecContext(ctx, `
		INSERT INTO billing_wallets (user_id, currency, balance)
		SELECT id, $2, $3
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		ON CONFLICT (user_id, currency) DO UPDATE
		SET balance = billing_wallets.balance + EXCLUDED.balance,
			updated_at = NOW()
	`, userID, currency, amountMoney.DBValue())
	return err
}
