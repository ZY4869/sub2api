package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usageBillingRepository struct {
	db *sql.DB
}

func NewUsageBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func (r *usageBillingRepository) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (_ *service.UsageBillingApplyResult, err error) {
	if cmd == nil {
		return &service.UsageBillingApplyResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}

	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
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

	applied, err := r.claimUsageBillingKey(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.UsageBillingApplyResult{Applied: false}, nil
	}

	result := &service.UsageBillingApplyResult{Applied: true}
	if err := r.applyUsageBillingEffects(ctx, tx, cmd, result); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) claimUsageBillingKey(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var existingFingerprint string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM usage_billing_dedup
			WHERE request_id = $1 AND api_key_id = $2
		`, cmd.RequestID, cmd.APIKeyID).Scan(&existingFingerprint); err != nil {
			return false, err
		}
		if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(cmd.RequestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var archivedFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, cmd.RequestID, cmd.APIKeyID).Scan(&archivedFingerprint)
	if err == nil {
		if strings.TrimSpace(archivedFingerprint) != strings.TrimSpace(cmd.RequestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func (r *usageBillingRepository) applyUsageBillingEffects(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	currency := service.NormalizeUsageBillingCurrency(cmd.BillingCurrency)
	if cmd.SubscriptionCost > 0 && cmd.SubscriptionID != nil {
		if currency == service.ModelPricingCurrencyUSD {
			if err := incrementUsageBillingSubscription(ctx, tx, *cmd.SubscriptionID, cmd.SubscriptionCost); err != nil {
				return err
			}
		} else if err := incrementUsageBillingSubscriptionCurrency(ctx, tx, *cmd.SubscriptionID, currency, cmd.SubscriptionCost); err != nil {
			return err
		}
	}

	if cmd.BalanceCost > 0 {
		if err := applyUsageBillingWalletDebit(ctx, tx, cmd, currency, cmd.BalanceCost); err != nil {
			return err
		}
	}

	if cmd.APIKeyQuotaCost > 0 {
		if currency == service.ModelPricingCurrencyUSD {
			exhausted, err := incrementUsageBillingAPIKeyQuota(ctx, tx, cmd.APIKeyID, cmd.APIKeyQuotaCost)
			if err != nil {
				return err
			}
			result.APIKeyQuotaExhausted = exhausted
		} else if err := incrementUsageBillingAPIKeyQuotaCurrency(ctx, tx, cmd.APIKeyID, currency, cmd.APIKeyQuotaCost); err != nil {
			return err
		}
	}

	if cmd.APIKeyGroupQuotaCost > 0 && cmd.GroupID != nil && *cmd.GroupID > 0 {
		if currency == service.ModelPricingCurrencyUSD {
			if err := incrementUsageBillingAPIKeyGroupQuota(ctx, tx, cmd.APIKeyID, *cmd.GroupID, cmd.APIKeyGroupQuotaCost); err != nil {
				return err
			}
		} else if err := incrementUsageBillingAPIKeyGroupQuotaCurrency(ctx, tx, cmd.APIKeyID, *cmd.GroupID, currency, cmd.APIKeyGroupQuotaCost); err != nil {
			return err
		}
	}

	if cmd.APIKeyRateLimitCost > 0 {
		if currency == service.ModelPricingCurrencyUSD {
			if err := incrementUsageBillingAPIKeyRateLimit(ctx, tx, cmd.APIKeyID, cmd.APIKeyRateLimitCost); err != nil {
				return err
			}
		} else if err := incrementUsageBillingAPIKeyRateLimitCurrency(ctx, tx, cmd.APIKeyID, currency, cmd.APIKeyRateLimitCost); err != nil {
			return err
		}
	}

	if cmd.AccountQuotaCost > 0 && strings.TrimSpace(cmd.AccountType) != "" {
		if currency == service.ModelPricingCurrencyUSD {
			if err := incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost); err != nil {
				return err
			}
		} else if err := incrementUsageBillingAccountQuotaCurrency(ctx, tx, cmd.AccountID, currency, cmd.AccountQuotaCost); err != nil {
			return err
		}
	}

	applyAffiliateUsageRebateBestEffort(ctx, tx, cmd)

	return nil
}

func incrementUsageBillingSubscription(ctx context.Context, tx *sql.Tx, subscriptionID int64, costUSD float64) error {
	const updateSQL = `
		UPDATE user_subscriptions us
		SET
			daily_usage_usd = us.daily_usage_usd + $1,
			weekly_usage_usd = us.weekly_usage_usd + $1,
			monthly_usage_usd = us.monthly_usage_usd + $1,
			updated_at = NOW()
		FROM groups g
		WHERE us.id = $2
			AND us.deleted_at IS NULL
			AND us.group_id = g.id
			AND g.deleted_at IS NULL
	`
	res, err := tx.ExecContext(ctx, updateSQL, costUSD, subscriptionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return service.ErrSubscriptionNotFound
}

func incrementUsageBillingSubscriptionCurrency(ctx context.Context, tx *sql.Tx, subscriptionID int64, currency string, amount float64) error {
	const updateSQL = `
		UPDATE user_subscriptions us
		SET
			daily_usage_by_currency = COALESCE(us.daily_usage_by_currency, '{}'::jsonb)
				|| jsonb_build_object($2, COALESCE((us.daily_usage_by_currency->>$2)::numeric, 0) + $1),
			weekly_usage_by_currency = COALESCE(us.weekly_usage_by_currency, '{}'::jsonb)
				|| jsonb_build_object($2, COALESCE((us.weekly_usage_by_currency->>$2)::numeric, 0) + $1),
			monthly_usage_by_currency = COALESCE(us.monthly_usage_by_currency, '{}'::jsonb)
				|| jsonb_build_object($2, COALESCE((us.monthly_usage_by_currency->>$2)::numeric, 0) + $1),
			updated_at = NOW()
		FROM groups g
		WHERE us.id = $3
			AND us.deleted_at IS NULL
			AND us.group_id = g.id
			AND g.deleted_at IS NULL
	`
	res, err := tx.ExecContext(ctx, updateSQL, amount, service.NormalizeUsageBillingCurrency(currency), subscriptionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return service.ErrSubscriptionNotFound
}

func applyUsageBillingWalletDebit(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, currency string, amount float64) error {
	currency = service.NormalizeUsageBillingCurrency(currency)
	if currency == service.ModelPricingCurrencyUSD {
		if err := ensureUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD); err != nil {
			return err
		}
		if _, err := lockUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD); err != nil {
			return err
		}
		if err := addUsageBillingWalletBalance(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD, -amount, true); err != nil {
			return err
		}
		if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, service.ModelPricingCurrencyUSD, -amount, "usage_debit", nil); err != nil {
			return err
		}
		logger.LegacyPrintf("repository.usage_billing", "wallet debit applied request_id=%s user_id=%d model=%s currency=%s amount=%.10f", cmd.RequestID, cmd.UserID, cmd.Model, currency, amount)
		return nil
	}
	if currency != service.ModelPricingCurrencyCNY {
		return fmt.Errorf("unsupported billing currency: %s", currency)
	}
	if cmd.USDToCNYRate <= 0 {
		return fmt.Errorf("missing locked USD/CNY rate for CNY billing")
	}
	if err := ensureUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD); err != nil {
		return err
	}
	if err := ensureUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyCNY); err != nil {
		return err
	}
	usdBalance, err := lockUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD)
	if err != nil {
		return err
	}
	cnyBalance, err := lockUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyCNY)
	if err != nil {
		return err
	}
	if deficit := amount - cnyBalance; deficit > 0 {
		usdDebit := deficit / cmd.USDToCNYRate
		if usdBalance+1e-12 < usdDebit {
			logger.LegacyPrintf("repository.usage_billing", "cny auto fx failed insufficient usd request_id=%s user_id=%d model=%s currency=%s amount=%.10f fx_rate=%.8f", cmd.RequestID, cmd.UserID, cmd.Model, currency, amount, cmd.USDToCNYRate)
			return service.ErrInsufficientBalance
		}
		if err := addUsageBillingWalletBalance(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD, -usdDebit, true); err != nil {
			return err
		}
		if err := addUsageBillingWalletBalance(ctx, tx, cmd.UserID, service.ModelPricingCurrencyCNY, deficit, false); err != nil {
			return err
		}
		meta := map[string]any{"target_currency": service.ModelPricingCurrencyCNY, "target_amount": deficit}
		if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, service.ModelPricingCurrencyUSD, -usdDebit, "fx_out", meta); err != nil {
			return err
		}
		meta = map[string]any{"source_currency": service.ModelPricingCurrencyUSD, "source_amount": usdDebit}
		if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, service.ModelPricingCurrencyCNY, deficit, "fx_in", meta); err != nil {
			return err
		}
		logger.LegacyPrintf("repository.usage_billing", "cny auto fx applied request_id=%s user_id=%d model=%s target_amount=%.10f usd_debit=%.10f fx_rate=%.8f", cmd.RequestID, cmd.UserID, cmd.Model, deficit, usdDebit, cmd.USDToCNYRate)
	}
	if err := addUsageBillingWalletBalance(ctx, tx, cmd.UserID, service.ModelPricingCurrencyCNY, -amount, false); err != nil {
		return err
	}
	if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, service.ModelPricingCurrencyCNY, -amount, "usage_debit", nil); err != nil {
		return err
	}
	logger.LegacyPrintf("repository.usage_billing", "wallet debit applied request_id=%s user_id=%d model=%s currency=%s amount=%.10f fx_rate=%.8f", cmd.RequestID, cmd.UserID, cmd.Model, currency, amount, cmd.USDToCNYRate)
	return nil
}

func ensureUsageBillingWallet(ctx context.Context, tx *sql.Tx, userID int64, currency string) error {
	currency = service.NormalizeUsageBillingCurrency(currency)
	if currency == service.ModelPricingCurrencyUSD {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO billing_wallets (user_id, currency, balance)
			SELECT id, $2, balance
			FROM users
			WHERE id = $1 AND deleted_at IS NULL
			ON CONFLICT (user_id, currency) DO NOTHING
		`, userID, currency)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err == nil && affected == 0 {
			var exists bool
			if scanErr := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, userID).Scan(&exists); scanErr != nil {
				return scanErr
			}
			if !exists {
				return service.ErrUserNotFound
			}
		}
		return nil
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO billing_wallets (user_id, currency, balance)
		SELECT id, $2, 0
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		ON CONFLICT (user_id, currency) DO NOTHING
	`, userID, currency)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		var exists bool
		if scanErr := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, userID).Scan(&exists); scanErr != nil {
			return scanErr
		}
		if !exists {
			return service.ErrUserNotFound
		}
	}
	return nil
}

func lockUsageBillingWallet(ctx context.Context, tx *sql.Tx, userID int64, currency string) (float64, error) {
	var balance float64
	err := tx.QueryRowContext(ctx, `
		SELECT balance
		FROM billing_wallets
		WHERE user_id = $1 AND currency = $2
		FOR UPDATE
	`, userID, service.NormalizeUsageBillingCurrency(currency)).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrUserNotFound
	}
	return balance, err
}

func addUsageBillingWalletBalance(ctx context.Context, tx *sql.Tx, userID int64, currency string, delta float64, updateUSDShadow bool) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE billing_wallets
		SET balance = balance + $3,
			updated_at = NOW()
		WHERE user_id = $1 AND currency = $2
	`, userID, service.NormalizeUsageBillingCurrency(currency), delta)
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
	if updateUSDShadow {
		res, err = tx.ExecContext(ctx, `
			UPDATE users
			SET balance = balance + $2,
				updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, userID, delta)
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
	}
	return nil
}

func insertUsageBillingLedgerEntry(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, currency string, amount float64, entryType string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["model"] = strings.TrimSpace(cmd.Model)
	metadata["api_key_id"] = cmd.APIKeyID
	metadata["account_id"] = cmd.AccountID
	if cmd.GroupID != nil {
		metadata["group_id"] = *cmd.GroupID
	}
	if cmd.SubscriptionID != nil {
		metadata["subscription_id"] = *cmd.SubscriptionID
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	fxRate := sql.NullFloat64{}
	if cmd.USDToCNYRate > 0 {
		fxRate = sql.NullFloat64{Float64: cmd.USDToCNYRate, Valid: true}
	}
	fxRateDate := sql.NullString{}
	if trimmed := strings.TrimSpace(cmd.FXRateDate); trimmed != "" {
		fxRateDate = sql.NullString{String: trimmed, Valid: true}
	}
	fxLockedAt := sql.NullTime{}
	if cmd.FXLockedAt != nil && !cmd.FXLockedAt.IsZero() {
		fxLockedAt = sql.NullTime{Time: cmd.FXLockedAt.UTC(), Valid: true}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO billing_ledger_entries (
			user_id, currency, amount, type, request_id, fx_rate, fx_rate_date, fx_locked_at, metadata, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, NOW())
	`, cmd.UserID, service.NormalizeUsageBillingCurrency(currency), amount, strings.TrimSpace(entryType), strings.TrimSpace(cmd.RequestID), fxRate, fxRateDate, fxLockedAt, string(payload))
	return err
}

func incrementUsageBillingAPIKeyQuota(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) (bool, error) {
	var exhausted bool
	err := tx.QueryRowContext(ctx, `
		UPDATE api_keys
		SET quota_used = quota_used + $1,
			status = CASE
				WHEN quota > 0
					AND status = $3
					AND quota_used < quota
					AND quota_used + $1 >= quota
				THEN $4
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING quota > 0 AND quota_used >= quota AND quota_used - $1 < quota
	`, amount, apiKeyID, service.StatusAPIKeyActive, service.StatusAPIKeyQuotaExhausted).Scan(&exhausted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrAPIKeyNotFound
	}
	if err != nil {
		return false, err
	}
	return exhausted, nil
}

func incrementUsageBillingAPIKeyQuotaCurrency(ctx context.Context, tx *sql.Tx, apiKeyID int64, currency string, amount float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys
		SET quota_used_by_currency = COALESCE(quota_used_by_currency, '{}'::jsonb)
				|| jsonb_build_object($2, COALESCE((quota_used_by_currency->>$2)::numeric, 0) + $1),
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`, amount, service.NormalizeUsageBillingCurrency(currency), apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAPIKeyGroupQuota(ctx context.Context, tx *sql.Tx, apiKeyID, groupID int64, amount float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_key_groups
		SET quota_used = quota_used + $1,
			updated_at = NOW()
		WHERE api_key_id = $2 AND group_id = $3
	`, amount, apiKeyID, groupID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrInvalidGroupBinding
	}
	return nil
}

func incrementUsageBillingAPIKeyGroupQuotaCurrency(ctx context.Context, tx *sql.Tx, apiKeyID, groupID int64, currency string, amount float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_key_groups
		SET quota_used_by_currency = COALESCE(quota_used_by_currency, '{}'::jsonb)
				|| jsonb_build_object($3, COALESCE((quota_used_by_currency->>$3)::numeric, 0) + $1),
			updated_at = NOW()
		WHERE api_key_id = $2 AND group_id = $4
	`, amount, apiKeyID, service.NormalizeUsageBillingCurrency(currency), groupID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrInvalidGroupBinding
	}
	return nil
}

func incrementUsageBillingAPIKeyRateLimit(ctx context.Context, tx *sql.Tx, apiKeyID int64, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, cost, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAPIKeyRateLimitCurrency(ctx context.Context, tx *sql.Tx, apiKeyID int64, currency string, cost float64) error {
	normalizedCurrency := service.NormalizeUsageBillingCurrency(currency)
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h_by_currency = CASE
				WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW()
				THEN jsonb_build_object($2, $1)
				ELSE COALESCE(usage_5h_by_currency, '{}'::jsonb)
					|| jsonb_build_object($2, COALESCE((usage_5h_by_currency->>$2)::numeric, 0) + $1)
			END,
			usage_1d_by_currency = CASE
				WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW()
				THEN jsonb_build_object($2, $1)
				ELSE COALESCE(usage_1d_by_currency, '{}'::jsonb)
					|| jsonb_build_object($2, COALESCE((usage_1d_by_currency->>$2)::numeric, 0) + $1)
			END,
			usage_7d_by_currency = CASE
				WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW()
				THEN jsonb_build_object($2, $1)
				ELSE COALESCE(usage_7d_by_currency, '{}'::jsonb)
					|| jsonb_build_object($2, COALESCE((usage_7d_by_currency->>$2)::numeric, 0) + $1)
			END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`, cost, normalizedCurrency, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAccountQuota(ctx context.Context, tx *sql.Tx, accountID int64, amount float64) error {
	rows, err := tx.QueryContext(ctx,
		`UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| jsonb_build_object('quota_used', COALESCE((extra->>'quota_used')::numeric, 0) + $1)
			|| CASE WHEN COALESCE((extra->>'quota_daily_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_daily_used',
					CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN $1
					ELSE COALESCE((extra->>'quota_daily_used')::numeric, 0) + $1 END,
					'quota_daily_start',
					CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
				)
			ELSE '{}'::jsonb END
			|| CASE WHEN COALESCE((extra->>'quota_weekly_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_weekly_used',
					CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN $1
					ELSE COALESCE((extra->>'quota_weekly_used')::numeric, 0) + $1 END,
					'quota_weekly_start',
					CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
				)
			ELSE '{}'::jsonb END
		), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->>'quota_used')::numeric, 0),
			COALESCE((extra->>'quota_limit')::numeric, 0)`,
		amount, accountID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var newUsed, limit float64
	if rows.Next() {
		if err := rows.Scan(&newUsed, &limit); err != nil {
			return err
		}
	} else {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrAccountNotFound
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if limit > 0 && newUsed >= limit && (newUsed-amount) < limit {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", accountID, err)
			return err
		}
	}
	return nil
}

func incrementUsageBillingAccountQuotaCurrency(ctx context.Context, tx *sql.Tx, accountID int64, currency string, amount float64) error {
	currency = service.NormalizeUsageBillingCurrency(currency)
	rows, err := tx.QueryContext(ctx,
		`UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| jsonb_build_object(
				'quota_used_by_currency',
				COALESCE(extra->'quota_used_by_currency', '{}'::jsonb)
					|| jsonb_build_object($2, COALESCE((extra->'quota_used_by_currency'->>$2)::numeric, 0) + $1)
			)
			|| jsonb_build_object(
				'quota_daily_used_by_currency',
				CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN jsonb_build_object($2, $1)
					ELSE COALESCE(extra->'quota_daily_used_by_currency', '{}'::jsonb)
						|| jsonb_build_object($2, COALESCE((extra->'quota_daily_used_by_currency'->>$2)::numeric, 0) + $1)
				END,
				'quota_daily_start',
				CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
			)
			|| jsonb_build_object(
				'quota_weekly_used_by_currency',
				CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN jsonb_build_object($2, $1)
					ELSE COALESCE(extra->'quota_weekly_used_by_currency', '{}'::jsonb)
						|| jsonb_build_object($2, COALESCE((extra->'quota_weekly_used_by_currency'->>$2)::numeric, 0) + $1)
				END,
				'quota_weekly_start',
				CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
			)
		), updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->'quota_used_by_currency'->>$2)::numeric, 0),
			COALESCE((extra->'quota_limit_by_currency'->>$2)::numeric, 0)`,
		amount, currency, accountID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var newUsed, limit float64
	if rows.Next() {
		if err := rows.Scan(&newUsed, &limit); err != nil {
			return err
		}
	} else {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrAccountNotFound
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if limit > 0 && newUsed >= limit && (newUsed-amount) < limit {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue currency quota exceeded failed: account=%d currency=%s err=%v", accountID, currency, err)
			return err
		}
	}
	return nil
}
