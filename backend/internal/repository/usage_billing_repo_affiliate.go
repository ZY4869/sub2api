package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type affiliateUsagePolicy struct {
	enabled     bool
	onUsage     bool
	ratePercent float64

	freezeHours   int
	durationDays  int
	perInviteeCap float64
}

func applyAffiliateUsageRebateBestEffort(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) {
	if cmd == nil || tx == nil {
		return
	}

	baseAmount := cmd.BalanceCost
	if cmd.SubscriptionCost > 0 && cmd.SubscriptionID != nil && *cmd.SubscriptionID > 0 {
		baseAmount = cmd.SubscriptionCost
	}
	if baseAmount <= 0 {
		return
	}

	policy, err := loadAffiliateUsagePolicy(ctx, tx)
	if err != nil {
		logger.LegacyPrintf("repository.usage_billing", "[Billing] affiliate: load policy failed: err=%v", err)
		return
	}
	if !policy.enabled || !policy.onUsage || policy.ratePercent <= 0 {
		return
	}

	// Isolate affiliate accounting from the main billing transaction. Even if affiliate logic fails,
	// the core billing effects should remain unaffected.
	if err := withUsageBillingSavepoint(ctx, tx, "aff_usage_rebate", func() error {
		return accrueAffiliateUsageRebateTx(ctx, tx, cmd, baseAmount, policy)
	}); err != nil {
		logger.LegacyPrintf("repository.usage_billing", "[Billing] affiliate: usage accrue failed: request_id=%s api_key_id=%d user_id=%d err=%v", cmd.RequestID, cmd.APIKeyID, cmd.UserID, err)
	}
}

func withUsageBillingSavepoint(ctx context.Context, tx *sql.Tx, name string, fn func() error) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("savepoint name is empty")
	}
	if _, err := tx.ExecContext(ctx, "SAVEPOINT "+name); err != nil {
		return err
	}
	if err := fn(); err != nil {
		_, rbErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT "+name)
		if rbErr != nil {
			return fmt.Errorf("rollback to savepoint failed: %w (original=%v)", rbErr, err)
		}
		_, _ = tx.ExecContext(ctx, "RELEASE SAVEPOINT "+name)
		return err
	}
	if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT "+name); err != nil {
		return err
	}
	return nil
}

func loadAffiliateUsagePolicy(ctx context.Context, tx *sql.Tx) (affiliateUsagePolicy, error) {
	policy := affiliateUsagePolicy{
		enabled:       false,
		onUsage:       true,
		ratePercent:   20.0,
		freezeHours:   0,
		durationDays:  0,
		perInviteeCap: 0,
	}
	if tx == nil {
		return policy, errors.New("tx is nil")
	}

	// Keep defaults aligned with SettingService defaults / parsing behavior.
	const (
		keyEnabled       = service.SettingKeyAffiliateEnabled
		keyOnUsage       = service.SettingKeyAffiliateRebateOnUsageEnabled
		keyRate          = service.SettingKeyAffiliateRebateRate
		keyFreezeHours   = service.SettingKeyAffiliateRebateFreezeHours
		keyDurationDays  = service.SettingKeyAffiliateRebateDurationDays
		keyPerInviteeCap = service.SettingKeyAffiliateRebatePerInviteeCap
	)

	rows, err := tx.QueryContext(ctx, `
		SELECT key, value
		FROM settings
		WHERE key IN ($1, $2, $3, $4, $5, $6)
	`, keyEnabled, keyOnUsage, keyRate, keyFreezeHours, keyDurationDays, keyPerInviteeCap)
	if err != nil {
		return policy, err
	}
	defer func() { _ = rows.Close() }()

	values := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return policy, err
		}
		values[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if err := rows.Err(); err != nil {
		return policy, err
	}

	if raw, ok := values[keyEnabled]; ok {
		policy.enabled = strings.EqualFold(strings.TrimSpace(raw), "true")
	}
	if raw, ok := values[keyOnUsage]; ok {
		policy.onUsage = !isFalseSettingValue(raw)
	}
	if raw, ok := values[keyRate]; ok && strings.TrimSpace(raw) != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			if v < 0 {
				v = 0
			}
			if v > 100 {
				v = 100
			}
			policy.ratePercent = v
		}
	}
	if raw, ok := values[keyFreezeHours]; ok && strings.TrimSpace(raw) != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			if v < 0 {
				v = 0
			}
			if v > 720 {
				v = 720
			}
			policy.freezeHours = v
		}
	}
	if raw, ok := values[keyDurationDays]; ok && strings.TrimSpace(raw) != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			if v < 0 {
				v = 0
			}
			if v > 3650 {
				v = 3650
			}
			policy.durationDays = v
		}
	}
	if raw, ok := values[keyPerInviteeCap]; ok && strings.TrimSpace(raw) != "" {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			if v < 0 {
				v = 0
			}
			policy.perInviteeCap = v
		}
	}
	return policy, nil
}

func isFalseSettingValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "off", "disabled":
		return true
	default:
		return false
	}
}

func accrueAffiliateUsageRebateTx(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, baseAmount float64, policy affiliateUsagePolicy) error {
	if tx == nil || cmd == nil {
		return nil
	}
	if baseAmount <= 0 {
		return nil
	}
	if cmd.UserID <= 0 || cmd.APIKeyID <= 0 || strings.TrimSpace(cmd.RequestID) == "" {
		return nil
	}

	var inviterUserID sql.NullInt64
	var inviterBoundAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
		SELECT inviter_user_id, inviter_bound_at
		FROM user_affiliates
		WHERE user_id = $1
	`, cmd.UserID).Scan(&inviterUserID, &inviterBoundAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if !inviterUserID.Valid || inviterUserID.Int64 <= 0 || inviterUserID.Int64 == cmd.UserID {
		return nil
	}

	now := time.Now()
	if policy.durationDays > 0 && inviterBoundAt.Valid {
		if now.After(inviterBoundAt.Time.Add(time.Duration(policy.durationDays) * 24 * time.Hour)) {
			return nil
		}
	}

	var customRate sql.NullFloat64
	if err := tx.QueryRowContext(ctx, `
		SELECT custom_rebate_rate_percent
		FROM user_affiliates
		WHERE user_id = $1
		FOR UPDATE
	`, inviterUserID.Int64).Scan(&customRate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	ratePercent := policy.ratePercent
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
		return nil
	}

	accruedAmount := baseAmount * ratePercent / 100
	if accruedAmount <= 0 {
		return nil
	}

	if policy.perInviteeCap > 0 {
		var existing float64
		if err := tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0)
			FROM user_affiliate_ledger
			WHERE inviter_user_id = $1
			  AND invitee_user_id = $2
			  AND event_type IN ('usage_accrue', 'topup_accrue')
		`, inviterUserID.Int64, cmd.UserID).Scan(&existing); err != nil {
			return err
		}
		remaining := policy.perInviteeCap - existing
		if remaining <= 0 {
			return nil
		}
		if accruedAmount > remaining {
			accruedAmount = remaining
		}
		if accruedAmount <= 0 {
			return nil
		}
	}

	var frozenUntil any
	frozenUntil = nil
	creditFrozen := false
	if policy.freezeHours > 0 {
		creditFrozen = true
		frozenUntil = now.Add(time.Duration(policy.freezeHours) * time.Hour)
	}

	var ledgerID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO user_affiliate_ledger (
			inviter_user_id,
			invitee_user_id,
			event_type,
			amount,
			base_amount,
			rate_percent,
			frozen_until,
			request_id,
			api_key_id,
			created_at
		)
		VALUES ($1, $2, 'usage_accrue', $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT DO NOTHING
		RETURNING id
	`, inviterUserID.Int64, cmd.UserID, accruedAmount, baseAmount, ratePercent, frozenUntil, strings.TrimSpace(cmd.RequestID), cmd.APIKeyID).Scan(&ledgerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_ = ledgerID

	if creditFrozen {
		_, err = tx.ExecContext(ctx, `
			UPDATE user_affiliates
			SET rebate_frozen_balance = rebate_frozen_balance + $1,
				lifetime_rebate = lifetime_rebate + $1,
				updated_at = NOW()
			WHERE user_id = $2
		`, accruedAmount, inviterUserID.Int64)
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE user_affiliates
			SET rebate_balance = rebate_balance + $1,
				lifetime_rebate = lifetime_rebate + $1,
				updated_at = NOW()
			WHERE user_id = $2
		`, accruedAmount, inviterUserID.Int64)
	}
	if err != nil {
		return err
	}
	return nil
}
