//go:build integration

package repository

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBillingMoneyPaths_PaymentTopupAndWalletSync(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	userRepo := newUserRepositoryWithSQL(client, integrationDB)
	paymentRepo := NewPaymentRepository(integrationDB)

	user := &service.User{
		Email:        fmt.Sprintf("billing-money-topup-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.1 + 0.2,
	}
	require.NoError(t, userRepo.Create(ctx, user))

	require.NoError(t, paymentRepo.AddWalletBalance(ctx, user.ID, service.ModelPricingCurrencyUSD, 0.000000014))

	var userBalance, walletBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&userBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = $2", user.ID, service.ModelPricingCurrencyUSD).Scan(&walletBalance))

	require.InDelta(t, 0.30000001, userBalance, 0.000000001)
	require.InDelta(t, 0.30000001, walletBalance, 0.000000001)
}

func TestBillingMoneyPaths_UserDebitSyncsUSDWallet(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	userRepo := newUserRepositoryWithSQL(client, integrationDB)

	user := &service.User{
		Email:        fmt.Sprintf("billing-money-debit-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	}
	require.NoError(t, userRepo.Create(ctx, user))

	require.NoError(t, userRepo.DeductBalance(ctx, user.ID, 0.1+0.2))

	var userBalance, walletBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&userBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = $2", user.ID, service.ModelPricingCurrencyUSD).Scan(&walletBalance))

	require.InDelta(t, 0.7, userBalance, 0.000000001)
	require.InDelta(t, 0.7, walletBalance, 0.000000001)
}

func TestBillingMoneyPaths_RejectUnsafeRepositoryAmounts(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	userRepo := newUserRepositoryWithSQL(client, integrationDB)
	paymentRepo := NewPaymentRepository(integrationDB)
	affiliateRepo := NewAffiliateRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("billing-money-unsafe-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})

	require.ErrorIs(t, userRepo.UpdateBalance(ctx, user.ID, math.NaN()), service.ErrInvalidBillingAmount)
	require.ErrorIs(t, userRepo.DeductBalance(ctx, user.ID, math.Inf(1)), service.ErrInvalidBillingAmount)
	require.ErrorIs(t, paymentRepo.AddWalletBalance(ctx, user.ID, service.ModelPricingCurrencyUSD, math.Copysign(0, -1)), service.ErrInvalidBillingAmount)
	_, err := affiliateRepo.AccrueTopupRebate(ctx, 1, user.ID, math.Inf(-1), service.AffiliateRebatePolicy{Enabled: true, RebateOnTopupEnabled: true, DefaultRatePercent: 10})
	require.ErrorIs(t, err, service.ErrInvalidBillingAmount)
}

func TestBillingMoneyPaths_TopupRebateCapAndTransferUseFixedPoint(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	userRepo := newUserRepositoryWithSQL(client, integrationDB)
	affiliateRepo := NewAffiliateRepository(client, integrationDB)

	inviter := &service.User{
		Email:        fmt.Sprintf("billing-money-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      2,
	}
	require.NoError(t, userRepo.Create(ctx, inviter))
	invitee := &service.User{
		Email:        fmt.Sprintf("billing-money-invitee-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	}
	require.NoError(t, userRepo.Create(ctx, invitee))

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO user_affiliates (user_id, aff_code)
		VALUES ($1, $2)
	`, inviter.ID, "aff-"+uuid.NewString())
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `
		INSERT INTO user_affiliates (user_id, aff_code, inviter_user_id, inviter_bound_at)
		VALUES ($1, $2, $3, NOW())
	`, invitee.ID, "aff-"+uuid.NewString(), inviter.ID)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `
		INSERT INTO user_affiliate_ledger (inviter_user_id, invitee_user_id, event_type, amount, created_at)
		VALUES ($1, $2, 'usage_accrue', 0.02, NOW())
	`, inviter.ID, invitee.ID)
	require.NoError(t, err)

	accrued, err := affiliateRepo.AccrueTopupRebate(ctx, 12345, invitee.ID, 0.1+0.2, service.AffiliateRebatePolicy{
		Enabled:              true,
		RebateOnTopupEnabled: true,
		DefaultRatePercent:   33.33333333,
		PerInviteeCap:        0.03,
	})
	require.NoError(t, err)
	require.InDelta(t, 0.01, accrued, 0.000000001)

	result, err := affiliateRepo.TransferToBalance(ctx, inviter.ID)
	require.NoError(t, err)
	require.InDelta(t, 0.01, result.TransferredAmount, 0.000000001)
	require.InDelta(t, 2.01, result.NewBalance, 0.000000001)

	var rebateBalance, userBalance, walletBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT rebate_balance FROM user_affiliates WHERE user_id = $1", inviter.ID).Scan(&rebateBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", inviter.ID).Scan(&userBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = $2", inviter.ID, service.ModelPricingCurrencyUSD).Scan(&walletBalance))
	require.InDelta(t, 0, rebateBalance, 0.000000001)
	require.InDelta(t, 2.01, userBalance, 0.000000001)
	require.InDelta(t, 2.01, walletBalance, 0.000000001)
}
