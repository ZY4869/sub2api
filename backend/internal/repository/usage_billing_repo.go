package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
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
	if err := normalizeUsageBillingCommandAmounts(cmd); err != nil {
		return nil, err
	}
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

func normalizeUsageBillingCommandAmounts(cmd *service.UsageBillingCommand) error {
	var err error
	if cmd.BalanceCost, err = service.NormalizeAndValidateNonNegativeBillingAmount(cmd.BalanceCost); err != nil {
		return err
	}
	if cmd.SubscriptionCost, err = service.NormalizeAndValidateNonNegativeBillingAmount(cmd.SubscriptionCost); err != nil {
		return err
	}
	if cmd.APIKeyQuotaCost, err = service.NormalizeAndValidateNonNegativeBillingAmount(cmd.APIKeyQuotaCost); err != nil {
		return err
	}
	if cmd.APIKeyGroupQuotaCost, err = service.NormalizeAndValidateNonNegativeBillingAmount(cmd.APIKeyGroupQuotaCost); err != nil {
		return err
	}
	if cmd.APIKeyRateLimitCost, err = service.NormalizeAndValidateNonNegativeBillingAmount(cmd.APIKeyRateLimitCost); err != nil {
		return err
	}
	if cmd.AccountQuotaCost, err = service.NormalizeAndValidateNonNegativeBillingAmount(cmd.AccountQuotaCost); err != nil {
		return err
	}
	if cmd.UserPlatformCost, err = service.NormalizeAndValidateNonNegativeBillingAmount(cmd.UserPlatformCost); err != nil {
		return err
	}
	return nil
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
	if err := ensureUsageBillingHoldFingerprintCompatible(ctx, tx, cmd); err != nil {
		return false, err
	}
	return true, nil
}

func ensureUsageBillingHoldFingerprintCompatible(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) error {
	if cmd == nil || strings.TrimSpace(cmd.RequestID) == "" || cmd.APIKeyID <= 0 || strings.TrimSpace(cmd.RequestPayloadHash) == "" {
		return nil
	}
	var holdFingerprint string
	err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(request_fingerprint, '')
		FROM billing_request_holds
		WHERE request_id = $1 AND api_key_id = $2
	`, strings.TrimSpace(cmd.RequestID), cmd.APIKeyID).Scan(&holdFingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	fingerprint := service.NormalizeBillingHoldRequestFingerprint(holdFingerprint)
	payloadHash := strings.TrimSpace(cmd.RequestPayloadHash)
	if fingerprint != "" && fingerprint != payloadHash {
		return service.ErrBillingRequestReplayed
	}
	if fingerprint == "" && payloadHash != "" {
		_, err = tx.ExecContext(ctx, `
			UPDATE billing_request_holds
			SET request_fingerprint = $3,
				updated_at = NOW()
			WHERE request_id = $1 AND api_key_id = $2 AND COALESCE(request_fingerprint, '') = ''
		`, strings.TrimSpace(cmd.RequestID), cmd.APIKeyID, payloadHash)
		if err != nil {
			return err
		}
	}
	return nil
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
	} else {
		if err := releaseZeroCostUsageBillingRequestHold(ctx, tx, cmd); err != nil {
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

	if cmd.UserPlatformCost > 0 && strings.TrimSpace(cmd.Platform) != "" {
		if err := incrementUsageBillingUserPlatformQuota(ctx, tx, cmd.UserID, cmd.Platform, cmd.UserPlatformCost); err != nil {
			return err
		}
	}

	applyAffiliateUsageRebateBestEffort(ctx, tx, cmd)

	return nil
}

func releaseZeroCostUsageBillingRequestHold(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) error {
	actualMoney, err := service.NewNonNegativeBillingMoneyFromFloat(0)
	if err != nil {
		return err
	}
	hold, err := settleUsageBillingRequestHold(ctx, tx, cmd, actualMoney)
	if err != nil || hold == nil {
		return err
	}
	logger.LegacyPrintf("repository.usage_billing", "wallet hold released for zero cost request_id=%s user_id=%d model=%s hold=%.10f", cmd.RequestID, cmd.UserID, cmd.Model, hold.Amount)
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
	var err error
	amount, err = service.NormalizeAndValidatePositiveBillingAmount(amount)
	if err != nil {
		return err
	}
	amountMoney, err := service.NewPositiveBillingMoneyFromFloat(amount)
	if err != nil {
		return err
	}
	if currency == service.ModelPricingCurrencyUSD {
		hold, err := settleUsageBillingRequestHold(ctx, tx, cmd, amountMoney)
		if err != nil {
			return err
		}
		if hold != nil {
			meta := map[string]any{"hold_amount": hold.Amount, "settled_from_hold": true}
			if parts := billingHoldBreakdownMoneyMap(hold.ConversionBreakdown); len(parts) > 0 {
				if err := insertUsageBillingConvertedDebitLedgerEntries(ctx, tx, cmd, hold.Currency, amountMoney, parts, meta); err != nil {
					return err
				}
			} else {
				debitMoney, err := amountMoney.Neg()
				if err != nil {
					return err
				}
				if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, service.ModelPricingCurrencyUSD, debitMoney, "usage_debit", meta); err != nil {
					return err
				}
			}
			logger.LegacyPrintf("repository.usage_billing", "wallet debit settled from hold request_id=%s user_id=%d model=%s currency=%s hold=%.10f amount=%s", cmd.RequestID, cmd.UserID, cmd.Model, currency, hold.Amount, amountMoney.DBValue())
			return nil
		}
		if cmd.CurrencyConversionEnabled {
			return applyUsageBillingConvertedWalletDebit(ctx, tx, cmd, currency, amountMoney)
		}
		if err := ensureUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD); err != nil {
			return err
		}
		balanceMoney, err := lockUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD)
		if err != nil {
			return err
		}
		if balanceMoney.Cmp(amountMoney) < 0 {
			return service.ErrInsufficientBalance
		}
		debitMoney, err := amountMoney.Neg()
		if err != nil {
			return err
		}
		if err := addUsageBillingWalletBalance(ctx, tx, cmd.UserID, service.ModelPricingCurrencyUSD, debitMoney, true); err != nil {
			return err
		}
		if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, service.ModelPricingCurrencyUSD, debitMoney, "usage_debit", nil); err != nil {
			return err
		}
		logger.LegacyPrintf("repository.usage_billing", "wallet debit applied request_id=%s user_id=%d model=%s currency=%s amount=%s", cmd.RequestID, cmd.UserID, cmd.Model, currency, amountMoney.DBValue())
		return nil
	}
	if currency != service.ModelPricingCurrencyCNY {
		return fmt.Errorf("unsupported billing currency: %s", currency)
	}
	if err := releaseUsageBillingRequestHoldBeforeCurrencyDebit(ctx, tx, cmd, currency); err != nil {
		return err
	}
	if cmd.CurrencyConversionEnabled {
		return applyUsageBillingConvertedWalletDebit(ctx, tx, cmd, currency, amountMoney)
	}
	if err := ensureUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyCNY); err != nil {
		return err
	}
	balanceMoney, err := lockUsageBillingWallet(ctx, tx, cmd.UserID, service.ModelPricingCurrencyCNY)
	if err != nil {
		return err
	}
	if balanceMoney.Cmp(amountMoney) < 0 {
		return service.ErrInsufficientBalance
	}
	debitMoney, err := amountMoney.Neg()
	if err != nil {
		return err
	}
	if err := addUsageBillingWalletBalance(ctx, tx, cmd.UserID, service.ModelPricingCurrencyCNY, debitMoney, false); err != nil {
		return err
	}
	if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, service.ModelPricingCurrencyCNY, debitMoney, "usage_debit", nil); err != nil {
		return err
	}
	logger.LegacyPrintf("repository.usage_billing", "wallet debit applied request_id=%s user_id=%d model=%s currency=%s amount=%s", cmd.RequestID, cmd.UserID, cmd.Model, currency, amountMoney.DBValue())
	return nil
}

func releaseUsageBillingRequestHoldBeforeCurrencyDebit(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, targetCurrency string) error {
	if service.NormalizeUsageBillingCurrency(targetCurrency) == service.ModelPricingCurrencyUSD {
		return nil
	}
	zero, err := service.NewNonNegativeBillingMoneyFromFloat(0)
	if err != nil {
		return err
	}
	hold, err := settleUsageBillingRequestHold(ctx, tx, cmd, zero)
	if err != nil || hold == nil {
		return err
	}
	logger.LegacyPrintf("repository.usage_billing", "wallet hold released before non-usd debit request_id=%s user_id=%d model=%s hold_currency=%s target_currency=%s hold=%.10f", cmd.RequestID, cmd.UserID, cmd.Model, hold.Currency, targetCurrency, hold.Amount)
	return nil
}

func applyUsageBillingConvertedWalletDebit(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, targetCurrency string, amountMoney service.BillingMoney) error {
	targetCurrency = service.NormalizeUsageBillingCurrency(targetCurrency)
	parts, err := reserveConvertedWalletDebitParts(ctx, tx, cmd.UserID, targetCurrency, amountMoney, service.BillingCurrencyConversionSettings{
		Enabled:      true,
		CNYToUSDRate: cmd.CNYToUSDRate,
		USDToCNYRate: cmd.USDToCNYConversionRate,
	})
	if err != nil {
		recordUsageBillingCurrencyFXFailure(ctx, cmd, targetCurrency, amountMoney, convertedSourceCurrency(targetCurrency), err)
		return err
	}
	for currency, debitMoney := range parts {
		if debitMoney.IsZero() {
			continue
		}
		negativeDebit, err := debitMoney.Neg()
		if err != nil {
			recordUsageBillingCurrencyFXFailure(ctx, cmd, targetCurrency, amountMoney, currency, err)
			return err
		}
		if err := addUsageBillingWalletBalance(ctx, tx, cmd.UserID, currency, negativeDebit, currency == service.ModelPricingCurrencyUSD); err != nil {
			recordUsageBillingCurrencyFXFailure(ctx, cmd, targetCurrency, amountMoney, currency, err)
			return err
		}
	}
	if err := insertUsageBillingConvertedDebitLedgerEntries(ctx, tx, cmd, targetCurrency, amountMoney, parts, nil); err != nil {
		recordUsageBillingCurrencyFXFailure(ctx, cmd, targetCurrency, amountMoney, convertedPartsSourceCurrency(parts, targetCurrency), err)
		return err
	}
	if sourceCurrency := convertedPartsSourceCurrency(parts, targetCurrency); sourceCurrency != "" {
		protocolruntime.RecordBillingResolver("currency_fx")
		logger.FromContext(ctx).Info(
			"currency fx debit applied",
			zap.String("request_id", cmd.RequestID),
			zap.Int64("user_id", cmd.UserID),
			zap.String("model", cmd.Model),
			zap.String("target_currency", targetCurrency),
			zap.String("source_currency", sourceCurrency),
			zap.String("target_amount", amountMoney.DBValue()),
		)
	}
	logger.LegacyPrintf("repository.usage_billing", "converted wallet debit applied request_id=%s user_id=%d model=%s target_currency=%s amount=%s", cmd.RequestID, cmd.UserID, cmd.Model, targetCurrency, amountMoney.DBValue())
	return nil
}

func recordUsageBillingCurrencyFXFailure(ctx context.Context, cmd *service.UsageBillingCommand, targetCurrency string, amountMoney service.BillingMoney, sourceCurrency string, err error) {
	reason := "currency_fx_failed"
	if errors.Is(err, service.ErrInsufficientBalance) {
		reason = "currency_fx_insufficient_balance"
	}
	protocolruntime.RecordBillingResolverFallback(reason)
	if cmd == nil {
		logger.FromContext(ctx).Warn(
			"currency fx debit failed",
			zap.String("reason", reason),
			zap.String("target_currency", targetCurrency),
			zap.String("source_currency", sourceCurrency),
			zap.String("target_amount", amountMoney.DBValue()),
			zap.Error(err),
		)
		return
	}
	logger.FromContext(ctx).Warn(
		"currency fx debit failed",
		zap.String("reason", reason),
		zap.String("request_id", cmd.RequestID),
		zap.Int64("user_id", cmd.UserID),
		zap.String("model", cmd.Model),
		zap.String("target_currency", targetCurrency),
		zap.String("source_currency", sourceCurrency),
		zap.String("target_amount", amountMoney.DBValue()),
		zap.Error(err),
	)
}

func convertedPartsSourceCurrency(parts map[string]service.BillingMoney, targetCurrency string) string {
	targetCurrency = service.NormalizeUsageBillingCurrency(targetCurrency)
	for currency, amount := range parts {
		currency = service.NormalizeUsageBillingCurrency(currency)
		if currency != targetCurrency && !amount.IsZero() {
			return currency
		}
	}
	return ""
}

func insertUsageBillingConvertedDebitLedgerEntries(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, targetCurrency string, amountMoney service.BillingMoney, parts map[string]service.BillingMoney, extra map[string]any) error {
	targetCurrency = service.NormalizeUsageBillingCurrency(targetCurrency)
	if len(parts) == 0 {
		debitMoney, err := amountMoney.Neg()
		if err != nil {
			return err
		}
		return insertUsageBillingLedgerEntry(ctx, tx, cmd, targetCurrency, debitMoney, "usage_debit", extra)
	}
	for currency, debitMoney := range parts {
		currency = service.NormalizeUsageBillingCurrency(currency)
		if debitMoney.IsZero() {
			continue
		}
		meta := mergeBillingLedgerMetadata(extra, map[string]any{
			"target_currency": targetCurrency,
			"target_amount":   amountMoney.Float64(),
		})
		negativeDebit, err := debitMoney.Neg()
		if err != nil {
			return err
		}
		if currency == targetCurrency {
			if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, currency, negativeDebit, "usage_debit", meta); err != nil {
				return err
			}
			continue
		}
		conversionSettings := service.BillingCurrencyConversionSettings{
			Enabled:      true,
			CNYToUSDRate: cmd.CNYToUSDRate,
			USDToCNYRate: cmd.USDToCNYConversionRate,
		}
		targetCredit, err := convertedTargetCreditForDebitPart(targetCurrency, currency, amountMoney, parts, debitMoney, conversionSettings)
		if err != nil {
			return err
		}
		meta["converted_amount"] = targetCredit.Float64()
		if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, currency, negativeDebit, "fx_out", meta); err != nil {
			return err
		}
		fxInMeta := mergeBillingLedgerMetadata(extra, map[string]any{
			"source_currency": currency,
			"source_amount":   debitMoney.Float64(),
		})
		if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, targetCurrency, targetCredit, "fx_in", fxInMeta); err != nil {
			return err
		}
		negativeTargetCredit, err := targetCredit.Neg()
		if err != nil {
			return err
		}
		usageMeta := mergeBillingLedgerMetadata(extra, map[string]any{"converted_from_currency": currency})
		if err := insertUsageBillingLedgerEntry(ctx, tx, cmd, targetCurrency, negativeTargetCredit, "usage_debit", usageMeta); err != nil {
			return err
		}
	}
	return nil
}

func mergeBillingLedgerMetadata(base, extra map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func reserveConvertedWalletDebitParts(ctx context.Context, tx *sql.Tx, userID int64, targetCurrency string, amountMoney service.BillingMoney, settings service.BillingCurrencyConversionSettings) (map[string]service.BillingMoney, error) {
	targetCurrency = service.NormalizeUsageBillingCurrency(targetCurrency)
	sourceCurrency := convertedSourceCurrency(targetCurrency)
	if sourceCurrency == "" || !settings.Enabled {
		return reserveSingleCurrencyDebitParts(ctx, tx, userID, targetCurrency, amountMoney)
	}
	if err := ensureUsageBillingWallet(ctx, tx, userID, targetCurrency); err != nil {
		return nil, err
	}
	if err := ensureUsageBillingWallet(ctx, tx, userID, sourceCurrency); err != nil {
		return nil, err
	}
	balances, err := lockUsageBillingWallets(ctx, tx, userID, sourceCurrency, targetCurrency)
	if err != nil {
		return nil, err
	}
	sourceBalance := balances[sourceCurrency]
	targetBalance := balances[targetCurrency]
	rate := billingConversionRate(sourceCurrency, targetCurrency, settings)
	if rate <= 0 {
		return nil, service.ErrInvalidBillingAmount
	}
	sourceNeeded, err := amountMoney.DivRate(rate)
	if err != nil {
		return nil, err
	}
	sourceDebit := sourceNeeded
	var remaining service.BillingMoney
	if sourceBalance.Cmp(sourceNeeded) >= 0 {
		remaining, err = service.BillingMoneyFromUnits(0)
		if err != nil {
			return nil, err
		}
	} else {
		sourceDebit = sourceBalance
		targetCovered, err := convertBillingMoney(sourceDebit, sourceCurrency, targetCurrency, settings)
		if err != nil {
			return nil, err
		}
		if targetCovered.Cmp(amountMoney) >= 0 {
			remaining, err = service.BillingMoneyFromUnits(0)
			if err != nil {
				return nil, err
			}
		} else {
			remaining, err = amountMoney.Sub(targetCovered)
			if err != nil {
				return nil, err
			}
		}
	}
	if targetBalance.Cmp(remaining) < 0 {
		return nil, service.ErrInsufficientBalance
	}
	return map[string]service.BillingMoney{
		sourceCurrency: sourceDebit,
		targetCurrency: remaining,
	}, nil
}

func convertedTargetCreditForDebitPart(targetCurrency, currency string, amountMoney service.BillingMoney, parts map[string]service.BillingMoney, debitMoney service.BillingMoney, settings service.BillingCurrencyConversionSettings) (service.BillingMoney, error) {
	if service.NormalizeUsageBillingCurrency(currency) == service.NormalizeUsageBillingCurrency(targetCurrency) {
		return debitMoney, nil
	}
	remainingTarget := parts[service.NormalizeUsageBillingCurrency(targetCurrency)]
	targetCovered, err := amountMoney.Sub(remainingTarget)
	if err != nil {
		return service.BillingMoney{}, err
	}
	if targetCovered.IsPositive() || targetCovered.IsZero() || !settings.Enabled {
		return targetCovered, nil
	}
	return convertBillingMoney(debitMoney, currency, targetCurrency, settings)
}

func reserveSingleCurrencyDebitParts(ctx context.Context, tx *sql.Tx, userID int64, currency string, amountMoney service.BillingMoney) (map[string]service.BillingMoney, error) {
	if err := ensureUsageBillingWallet(ctx, tx, userID, currency); err != nil {
		return nil, err
	}
	balance, err := lockUsageBillingWallet(ctx, tx, userID, currency)
	if err != nil {
		return nil, err
	}
	if balance.Cmp(amountMoney) < 0 {
		return nil, service.ErrInsufficientBalance
	}
	return map[string]service.BillingMoney{service.NormalizeUsageBillingCurrency(currency): amountMoney}, nil
}

func lockUsageBillingWallets(ctx context.Context, tx *sql.Tx, userID int64, currencies ...string) (map[string]service.BillingMoney, error) {
	unique := make([]string, 0, len(currencies))
	seen := map[string]bool{}
	for _, currency := range currencies {
		currency = service.NormalizeUsageBillingCurrency(currency)
		if seen[currency] {
			continue
		}
		seen[currency] = true
		unique = append(unique, currency)
	}
	sort.Strings(unique)
	out := map[string]service.BillingMoney{}
	for _, currency := range unique {
		balance, err := lockUsageBillingWallet(ctx, tx, userID, currency)
		if err != nil {
			return nil, err
		}
		out[currency] = balance
	}
	return out, nil
}

func convertedSourceCurrency(targetCurrency string) string {
	switch service.NormalizeUsageBillingCurrency(targetCurrency) {
	case service.ModelPricingCurrencyUSD:
		return service.ModelPricingCurrencyCNY
	case service.ModelPricingCurrencyCNY:
		return service.ModelPricingCurrencyUSD
	default:
		return ""
	}
}

func convertBillingMoney(amount service.BillingMoney, fromCurrency, toCurrency string, settings service.BillingCurrencyConversionSettings) (service.BillingMoney, error) {
	fromCurrency = service.NormalizeUsageBillingCurrency(fromCurrency)
	toCurrency = service.NormalizeUsageBillingCurrency(toCurrency)
	if fromCurrency == toCurrency {
		return amount, nil
	}
	switch {
	case fromCurrency == service.ModelPricingCurrencyCNY && toCurrency == service.ModelPricingCurrencyUSD:
		return amount.MulRate(settings.CNYToUSDRate)
	case fromCurrency == service.ModelPricingCurrencyUSD && toCurrency == service.ModelPricingCurrencyCNY:
		return amount.MulRate(settings.USDToCNYRate)
	default:
		return service.BillingMoney{}, fmt.Errorf("unsupported billing conversion: %s to %s", fromCurrency, toCurrency)
	}
}

func billingConversionRate(fromCurrency, toCurrency string, settings service.BillingCurrencyConversionSettings) float64 {
	fromCurrency = service.NormalizeUsageBillingCurrency(fromCurrency)
	toCurrency = service.NormalizeUsageBillingCurrency(toCurrency)
	switch {
	case fromCurrency == service.ModelPricingCurrencyCNY && toCurrency == service.ModelPricingCurrencyUSD:
		return settings.CNYToUSDRate
	case fromCurrency == service.ModelPricingCurrencyUSD && toCurrency == service.ModelPricingCurrencyCNY:
		return settings.USDToCNYRate
	default:
		return 0
	}
}

func settleUsageBillingRequestHold(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, actualMoney service.BillingMoney) (*service.BillingHold, error) {
	if cmd == nil || strings.TrimSpace(cmd.RequestID) == "" || cmd.APIKeyID <= 0 {
		return nil, nil
	}
	if actualMoney.IsNegative() {
		return nil, service.ErrInvalidBillingAmount
	}
	var hold service.BillingHold
	var status string
	var breakdownRaw, policyRaw string
	err := tx.QueryRowContext(ctx, `
		SELECT request_id, api_key_id, user_id, currency, hold_amount, status, COALESCE(request_fingerprint, ''), COALESCE(conversion_breakdown::text, '{}'), COALESCE(conversion_policy::text, '{}')
		FROM billing_request_holds
		WHERE request_id = $1 AND api_key_id = $2
		FOR UPDATE
	`, strings.TrimSpace(cmd.RequestID), cmd.APIKeyID).Scan(&hold.RequestID, &hold.APIKeyID, &hold.UserID, &hold.Currency, &hold.Amount, &status, &hold.RequestFingerprint, &breakdownRaw, &policyRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	hold.Status = service.BillingHoldStatus(status)
	breakdown := decodeBillingHoldBreakdown(breakdownRaw)
	hold.ConversionBreakdown = billingHoldBreakdownFloatMap(breakdown)
	hold.CurrencyConversion = decodeBillingHoldConversionPolicy(policyRaw)
	if hold.Status != service.BillingHoldStatusHeld {
		return nil, nil
	}
	if hold.UserID != cmd.UserID {
		return nil, fmt.Errorf("billing hold user mismatch")
	}
	fingerprint := service.NormalizeBillingHoldRequestFingerprint(hold.RequestFingerprint)
	payloadHash := strings.TrimSpace(cmd.RequestPayloadHash)
	if fingerprint != "" && payloadHash != "" && fingerprint != payloadHash {
		return nil, service.ErrBillingRequestReplayed
	}
	if fingerprint == "" && payloadHash != "" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE billing_request_holds
			SET request_fingerprint = $3,
				updated_at = NOW()
			WHERE request_id = $1 AND api_key_id = $2 AND COALESCE(request_fingerprint, '') = ''
		`, strings.TrimSpace(cmd.RequestID), cmd.APIKeyID, payloadHash); err != nil {
			return nil, err
		}
		hold.RequestFingerprint = payloadHash
	}
	holdMoney, err := service.NewNonNegativeBillingMoneyFromFloat(hold.Amount)
	if err != nil {
		return nil, err
	}
	finalBreakdown := breakdown
	if len(breakdown) == 0 {
		deltaMoney, err := holdMoney.Sub(actualMoney)
		if err != nil {
			return nil, err
		}
		if err := adjustUSDWalletBalance(ctx, tx, hold.UserID, deltaMoney); err != nil {
			return nil, err
		}
	} else if holdMoney.Cmp(actualMoney) >= 0 {
		unusedMoney, err := holdMoney.Sub(actualMoney)
		if err != nil {
			return nil, err
		}
		refundParts := prorateBillingHoldBreakdown(breakdown, unusedMoney, holdMoney)
		if err := applyBillingHoldBalanceDeltas(ctx, tx, hold.UserID, refundParts, 1); err != nil {
			return nil, err
		}
		finalBreakdown, err = subtractBillingHoldBreakdown(breakdown, refundParts)
		if err != nil {
			return nil, err
		}
	} else {
		extraMoney, err := actualMoney.Sub(holdMoney)
		if err != nil {
			return nil, err
		}
		extraParts, err := reserveConvertedWalletDebitParts(ctx, tx, hold.UserID, hold.Currency, extraMoney, hold.CurrencyConversion)
		if err != nil {
			return nil, err
		}
		if err := applyBillingHoldBalanceDeltas(ctx, tx, hold.UserID, extraParts, -1); err != nil {
			return nil, err
		}
		finalBreakdown, err = addBillingHoldBreakdown(breakdown, extraParts)
		if err != nil {
			return nil, err
		}
	}
	breakdownJSON, err := encodeBillingHoldBreakdown(finalBreakdown)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE billing_request_holds
		SET actual_amount = $3,
			status = $4,
			settled_at = NOW(),
			updated_at = NOW(),
			conversion_breakdown = $6::jsonb
		WHERE request_id = $1 AND api_key_id = $2 AND status = $5
	`, strings.TrimSpace(cmd.RequestID), cmd.APIKeyID, actualMoney.DBValue(), string(service.BillingHoldStatusSettled), string(service.BillingHoldStatusHeld), breakdownJSON)
	if err != nil {
		return nil, err
	}
	hold.Status = service.BillingHoldStatusSettled
	hold.ConversionBreakdown = billingHoldBreakdownFloatMap(finalBreakdown)
	return &hold, nil
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

func lockUsageBillingWallet(ctx context.Context, tx *sql.Tx, userID int64, currency string) (service.BillingMoney, error) {
	var balance float64
	err := tx.QueryRowContext(ctx, `
		SELECT balance
		FROM billing_wallets
		WHERE user_id = $1 AND currency = $2
		FOR UPDATE
	`, userID, service.NormalizeUsageBillingCurrency(currency)).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return service.BillingMoney{}, service.ErrUserNotFound
	}
	if err != nil {
		return service.BillingMoney{}, err
	}
	return service.NewBillingMoneyFromFloat(balance)
}

func addUsageBillingWalletBalance(ctx context.Context, tx *sql.Tx, userID int64, currency string, deltaMoney service.BillingMoney, updateUSDShadow bool) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE billing_wallets
		SET balance = balance + $3,
			updated_at = NOW()
		WHERE user_id = $1 AND currency = $2
	`, userID, service.NormalizeUsageBillingCurrency(currency), deltaMoney.DBValue())
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
	}
	return nil
}

func insertUsageBillingLedgerEntry(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, currency string, amountMoney service.BillingMoney, entryType string, metadata map[string]any) error {
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
	`, cmd.UserID, service.NormalizeUsageBillingCurrency(currency), amountMoney.DBValue(), strings.TrimSpace(entryType), strings.TrimSpace(cmd.RequestID), fxRate, fxRateDate, fxLockedAt, string(payload))
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

func incrementUsageBillingUserPlatformQuota(ctx context.Context, tx *sql.Tx, userID int64, platform string, amount float64) error {
	platform = service.NormalizeUserPlatformQuotaPlatform(platform)
	if platform == "" || amount <= 0 {
		return nil
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE user_platform_quotas
		SET
			daily_usage_usd = CASE
				WHEN daily_limit_usd IS NULL THEN daily_usage_usd
				WHEN daily_window_start IS NULL OR daily_window_start + INTERVAL '24 hours' <= NOW() THEN $3
				ELSE daily_usage_usd + $3
			END,
			weekly_usage_usd = CASE
				WHEN weekly_limit_usd IS NULL THEN weekly_usage_usd
				WHEN weekly_window_start IS NULL OR weekly_window_start + INTERVAL '7 days' <= NOW() THEN $3
				ELSE weekly_usage_usd + $3
			END,
			monthly_usage_usd = CASE
				WHEN monthly_limit_usd IS NULL THEN monthly_usage_usd
				WHEN monthly_window_start IS NULL OR monthly_window_start + INTERVAL '30 days' <= NOW() THEN $3
				ELSE monthly_usage_usd + $3
			END,
			daily_window_start = CASE
				WHEN daily_limit_usd IS NULL THEN daily_window_start
				WHEN daily_window_start IS NULL OR daily_window_start + INTERVAL '24 hours' <= NOW() THEN NOW()
				ELSE daily_window_start
			END,
			weekly_window_start = CASE
				WHEN weekly_limit_usd IS NULL THEN weekly_window_start
				WHEN weekly_window_start IS NULL OR weekly_window_start + INTERVAL '7 days' <= NOW() THEN NOW()
				ELSE weekly_window_start
			END,
			monthly_window_start = CASE
				WHEN monthly_limit_usd IS NULL THEN monthly_window_start
				WHEN monthly_window_start IS NULL OR monthly_window_start + INTERVAL '30 days' <= NOW() THEN NOW()
				ELSE monthly_window_start
			END,
			updated_at = NOW()
		WHERE user_id = $1 AND platform = $2
	`, userID, platform, amount)
	if err != nil {
		return err
	}
	if affected, err := res.RowsAffected(); err == nil && affected > 0 {
		logger.LegacyPrintf("repository.usage_billing", "platform quota usage applied request_user=%d platform=%s amount=%.10f", userID, platform, amount)
	}
	return nil
}
