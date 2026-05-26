package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type billingHoldRepository struct {
	db *sql.DB
}

func NewBillingHoldRepository(db *sql.DB) service.BillingHoldRepository {
	if db == nil {
		return nil
	}
	return &billingHoldRepository{db: db}
}

func (r *billingHoldRepository) Reserve(ctx context.Context, hold *service.BillingHold) (_ *service.BillingHold, err error) {
	if r == nil || r.db == nil {
		return nil, errors.New("billing hold repository db is nil")
	}
	if hold == nil {
		return nil, service.ErrInvalidBillingAmount
	}
	requestID := service.NormalizeBillingHoldRequestID(hold.RequestID)
	requestFingerprint := service.NormalizeBillingHoldRequestFingerprint(hold.RequestFingerprint)
	amount, err := service.NormalizeAndValidatePositiveBillingAmount(hold.Amount)
	if err != nil {
		return nil, err
	}
	amountMoney, err := service.NewPositiveBillingMoneyFromFloat(amount)
	if err != nil {
		return nil, err
	}
	currency := service.NormalizeUsageBillingCurrency(hold.Currency)
	if requestID == "" || hold.APIKeyID <= 0 || hold.UserID <= 0 || currency != service.ModelPricingCurrencyUSD || !amountMoney.IsPositive() {
		return nil, service.ErrInvalidBillingAmount
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var existing service.BillingHold
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT request_id, api_key_id, user_id, currency, hold_amount, status, COALESCE(request_fingerprint, '')
		FROM billing_request_holds
		WHERE request_id = $1 AND api_key_id = $2
		FOR UPDATE
	`, requestID, hold.APIKeyID).Scan(&existing.RequestID, &existing.APIKeyID, &existing.UserID, &existing.Currency, &existing.Amount, &status, &existing.RequestFingerprint)
	if err == nil {
		existing.Status = service.BillingHoldStatus(status)
		existingFingerprint := service.NormalizeBillingHoldRequestFingerprint(existing.RequestFingerprint)
		if requestFingerprint != "" && existingFingerprint != "" && existingFingerprint != requestFingerprint {
			if txErr := tx.Commit(); txErr != nil {
				return nil, txErr
			}
			tx = nil
			return nil, service.ErrBillingRequestReplayed
		}
		if requestFingerprint != "" && existingFingerprint == "" {
			if _, updateErr := tx.ExecContext(ctx, `
				UPDATE billing_request_holds
				SET request_fingerprint = $3,
					updated_at = NOW()
				WHERE request_id = $1 AND api_key_id = $2
			`, requestID, hold.APIKeyID, requestFingerprint); updateErr != nil {
				return nil, updateErr
			}
			existing.RequestFingerprint = requestFingerprint
		}
		if txErr := tx.Commit(); txErr != nil {
			return nil, txErr
		}
		tx = nil
		if existing.Status != service.BillingHoldStatusHeld {
			return nil, service.ErrBillingHoldAlreadyFinished
		}
		return &existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err := ensureUsageBillingWallet(ctx, tx, hold.UserID, currency); err != nil {
		return nil, err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE billing_wallets
		SET balance = balance - $3,
			updated_at = NOW()
		WHERE user_id = $1
		  AND currency = $2
		  AND balance >= $3
	`, hold.UserID, currency, amountMoney.DBValue())
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrInsufficientBalance
	}
	res, err = tx.ExecContext(ctx, `
		UPDATE users
		SET balance = balance - $2,
			updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND balance >= $2
	`, hold.UserID, amountMoney.DBValue())
	if err != nil {
		return nil, err
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrInsufficientBalance
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO billing_request_holds (
			request_id, api_key_id, user_id, currency, hold_amount, status, request_fingerprint, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`, requestID, hold.APIKeyID, hold.UserID, currency, amountMoney.DBValue(), string(service.BillingHoldStatusHeld), requestFingerprint)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return &service.BillingHold{
		RequestID:          requestID,
		APIKeyID:           hold.APIKeyID,
		UserID:             hold.UserID,
		Currency:           currency,
		Amount:             amount,
		Status:             service.BillingHoldStatusHeld,
		RequestFingerprint: requestFingerprint,
	}, nil
}

func (r *billingHoldRepository) Settle(ctx context.Context, requestID string, apiKeyID int64, actualAmount float64) (_ *service.BillingHold, err error) {
	return r.finish(ctx, requestID, apiKeyID, actualAmount, true)
}

func (r *billingHoldRepository) Release(ctx context.Context, requestID string, apiKeyID int64) (_ *service.BillingHold, err error) {
	return r.finish(ctx, requestID, apiKeyID, 0, false)
}

func (r *billingHoldRepository) finish(ctx context.Context, requestID string, apiKeyID int64, actualAmount float64, settle bool) (_ *service.BillingHold, err error) {
	if r == nil || r.db == nil {
		return nil, errors.New("billing hold repository db is nil")
	}
	requestID = service.NormalizeBillingHoldRequestID(requestID)
	actualAmount, err = service.NormalizeAndValidateNonNegativeBillingAmount(actualAmount)
	if err != nil {
		return nil, err
	}
	actualMoney, err := service.NewNonNegativeBillingMoneyFromFloat(actualAmount)
	if err != nil {
		return nil, err
	}
	if requestID == "" || apiKeyID <= 0 {
		return nil, service.ErrInvalidBillingAmount
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var hold service.BillingHold
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT request_id, api_key_id, user_id, currency, hold_amount, status, COALESCE(request_fingerprint, '')
		FROM billing_request_holds
		WHERE request_id = $1 AND api_key_id = $2
		FOR UPDATE
	`, requestID, apiKeyID).Scan(&hold.RequestID, &hold.APIKeyID, &hold.UserID, &hold.Currency, &hold.Amount, &status, &hold.RequestFingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBillingHoldNotFound
	}
	if err != nil {
		return nil, err
	}
	hold.Status = service.BillingHoldStatus(status)
	if hold.Status != service.BillingHoldStatusHeld {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		tx = nil
		return &hold, nil
	}

	holdMoney, err := service.NewNonNegativeBillingMoneyFromFloat(hold.Amount)
	if err != nil {
		return nil, err
	}
	deltaMoney, err := holdMoney.Sub(actualMoney)
	if err != nil {
		return nil, err
	}
	if !settle {
		deltaMoney = holdMoney
	}
	if !deltaMoney.IsZero() {
		if err := adjustUSDWalletBalance(ctx, tx, hold.UserID, deltaMoney); err != nil {
			return nil, err
		}
	}

	newStatus := service.BillingHoldStatusReleased
	if settle {
		newStatus = service.BillingHoldStatusSettled
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE billing_request_holds
		SET actual_amount = $3,
			status = $4,
			settled_at = NOW(),
			updated_at = NOW()
		WHERE request_id = $1 AND api_key_id = $2 AND status = $5
	`, requestID, apiKeyID, actualMoney.DBValue(), string(newStatus), string(service.BillingHoldStatusHeld))
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrBillingHoldNotFound
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	hold.Status = newStatus
	return &hold, nil
}

func adjustUSDWalletBalance(ctx context.Context, tx *sql.Tx, userID int64, deltaMoney service.BillingMoney) error {
	if deltaMoney.IsZero() {
		return nil
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE billing_wallets
		SET balance = balance + $3,
			updated_at = NOW()
		WHERE user_id = $1 AND currency = $2
	`, userID, service.ModelPricingCurrencyUSD, deltaMoney.DBValue())
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	res, err = tx.ExecContext(ctx, `
		UPDATE users
		SET balance = balance + $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, userID, deltaMoney.DBValue())
	if err != nil {
		return err
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}
