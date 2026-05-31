//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestUsageBillingRepositoryApply_DeduplicatesBalanceBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-" + uuid.NewString(),
		Name:   "billing",
		Quota:  1,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:           requestID,
		APIKeyID:            apiKey.ID,
		UserID:              user.ID,
		AccountID:           account.ID,
		AccountType:         service.AccountTypeAPIKey,
		BalanceCost:         1.25,
		APIKeyQuotaCost:     1.25,
		APIKeyRateLimitCost: 1.25,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result1)
	require.True(t, result1.Applied)
	require.True(t, result1.APIKeyQuotaExhausted)

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result2)
	require.False(t, result2.Applied)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 98.75, balance, 0.000001)

	var quotaUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used FROM api_keys WHERE id = $1", apiKey.ID).Scan(&quotaUsed))
	require.InDelta(t, 1.25, quotaUsed, 0.000001)

	var usage5h float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT usage_5h FROM api_keys WHERE id = $1", apiKey.ID).Scan(&usage5h))
	require.InDelta(t, 1.25, usage5h, 0.000001)

	var status string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT status FROM api_keys WHERE id = $1", apiKey.ID).Scan(&status))
	require.Equal(t, service.StatusAPIKeyQuotaExhausted, status)

	var dedupCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&dedupCount))
	require.Equal(t, 1, dedupCount)
}

func TestUsageBillingRepositoryApply_SettlesExistingBillingHold(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-hold-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.01,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-hold-" + uuid.NewString(),
		Name:   "billing-hold",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-hold-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.01,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)

	_, err = billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		AccountID:   account.ID,
		AccountType: service.AccountTypeAPIKey,
		BalanceCost: 0.03,
	})
	require.NoError(t, err)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, -0.02, balance, 0.000001)

	var status string
	var actual float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT status, actual_amount FROM billing_request_holds WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&status, &actual))
	require.Equal(t, string(service.BillingHoldStatusSettled), status)
	require.InDelta(t, 0.03, actual, 0.000001)
}

func TestUsageBillingRepositoryApply_RefundsHoldDeltaWhenActualBelowHold(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-hold-refund-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-hold-refund-" + uuid.NewString(),
		Name:   "billing-hold-refund",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-hold-refund-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.03,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)

	_, err = billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		AccountID:   account.ID,
		AccountType: service.AccountTypeAPIKey,
		BalanceCost: 0.01,
	})
	require.NoError(t, err)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0.04, balance, 0.000001)
}

func TestUsageBillingRepositoryApply_ReleasesHoldForZeroCost(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-hold-zero-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-hold-zero-" + uuid.NewString(),
		Name:   "billing-hold-zero",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-hold-zero-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.01,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)

	_, err = billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		AccountID:   account.ID,
		AccountType: service.AccountTypeAPIKey,
		BalanceCost: 0,
	})
	require.NoError(t, err)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0.05, balance, 0.000001)

	var status string
	var actual float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT status, actual_amount FROM billing_request_holds WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&status, &actual))
	require.Equal(t, string(service.BillingHoldStatusSettled), status)
	require.InDelta(t, 0, actual, 0.000001)
}

func TestBillingHoldRepositoryReleaseRefundsHeldAmount(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("billing-hold-release-refund-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-billing-hold-release-refund-" + uuid.NewString(),
		Name:   "billing-hold-release-refund",
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.01,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)

	released, err := holdRepo.Release(ctx, requestID, apiKey.ID)
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusReleased, released.Status)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0.05, balance, 0.000001)
}

func TestUsageBillingRepositoryApply_BackfillsEmptyHoldFingerprint(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-hold-empty-fp-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-hold-empty-fp-" + uuid.NewString(),
		Name:   "billing-hold-empty-fp",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-hold-empty-fp-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.01,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)
	require.Empty(t, hold.RequestFingerprint)

	_, err = billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:          requestID,
		APIKeyID:           apiKey.ID,
		UserID:             user.ID,
		AccountID:          account.ID,
		AccountType:        service.AccountTypeAPIKey,
		RequestPayloadHash: "payload-hash-late",
		BalanceCost:        0.02,
	})
	require.NoError(t, err)

	var fingerprint string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COALESCE(request_fingerprint, '') FROM billing_request_holds WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&fingerprint))
	require.Equal(t, "payload-hash-late", fingerprint)
}

func TestUsageBillingRepositoryApply_DebitsWhenBillingHoldAlreadyReleased(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-released-hold-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-released-hold-" + uuid.NewString(),
		Name:   "released-hold",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-released-hold-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.01,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)
	_, err = holdRepo.Release(ctx, requestID, apiKey.ID)
	require.NoError(t, err)

	_, err = billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		AccountID:   account.ID,
		AccountType: service.AccountTypeAPIKey,
		BalanceCost: 0.03,
	})
	require.NoError(t, err)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0.02, balance, 0.000001)

	var status string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT status FROM billing_request_holds WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&status))
	require.Equal(t, string(service.BillingHoldStatusReleased), status)
}

func TestBillingHoldRepositoryReserve_RejectsFinishedHoldReplay(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("billing-hold-replay-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-billing-hold-replay-" + uuid.NewString(),
		Name:   "billing-hold-replay",
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.01,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)
	_, err = holdRepo.Release(ctx, requestID, apiKey.ID)
	require.NoError(t, err)

	_, err = holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.01,
	})
	require.ErrorIs(t, err, service.ErrBillingHoldAlreadyFinished)
}

func TestBillingHoldRepositoryReserve_RejectsSameRequestDifferentFingerprint(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("billing-hold-fingerprint-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-billing-hold-fingerprint-" + uuid.NewString(),
		Name:   "billing-hold-fingerprint",
	})

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID:          requestID,
		RequestFingerprint: "payload-a",
		APIKeyID:           apiKey.ID,
		UserID:             user.ID,
		Currency:           service.ModelPricingCurrencyUSD,
		Amount:             0.01,
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)

	_, err = holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID:          requestID,
		RequestFingerprint: "payload-b",
		APIKeyID:           apiKey.ID,
		UserID:             user.ID,
		Currency:           service.ModelPricingCurrencyUSD,
		Amount:             0.01,
	})
	require.ErrorIs(t, err, service.ErrBillingRequestReplayed)
}

func TestUsageBillingRepositoryApply_RejectsHoldFingerprintMismatch(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-hold-fingerprint-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.05,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-hold-fingerprint-" + uuid.NewString(),
		Name:   "usage-billing-hold-fingerprint",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-hold-fingerprint-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	_, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID:          requestID,
		RequestFingerprint: "payload-a",
		APIKeyID:           apiKey.ID,
		UserID:             user.ID,
		Currency:           service.ModelPricingCurrencyUSD,
		Amount:             0.01,
	})
	require.NoError(t, err)

	_, err = billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:          requestID,
		RequestPayloadHash: "payload-b",
		APIKeyID:           apiKey.ID,
		UserID:             user.ID,
		AccountID:          account.ID,
		AccountType:        service.AccountTypeAPIKey,
		BalanceCost:        0.01,
	})
	require.ErrorIs(t, err, service.ErrBillingRequestReplayed)
}

func TestBillingHoldRepositoryReserve_AllowsOnlyCoveredConcurrentHolds(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("billing-hold-concurrent-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.01,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-billing-hold-concurrent-" + uuid.NewString(),
		Name:   "billing-hold-concurrent",
	})

	var successCount int64
	var insufficientCount int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
				RequestID: fmt.Sprintf("%s-%02d", uuid.NewString(), i),
				APIKeyID:  apiKey.ID,
				UserID:    user.ID,
				Currency:  service.ModelPricingCurrencyUSD,
				Amount:    0.01,
			})
			if err == nil && hold != nil && hold.Status == service.BillingHoldStatusHeld {
				atomic.AddInt64(&successCount, 1)
				return
			}
			if errors.Is(err, service.ErrInsufficientBalance) {
				atomic.AddInt64(&insufficientCount, 1)
				return
			}
			require.NoError(t, err)
		}(i)
	}
	close(start)
	wg.Wait()

	require.Equal(t, int64(1), successCount)
	require.Equal(t, int64(19), insufficientCount)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0, balance, 0.000001)
}

func TestUsageBillingRepositoryApply_DebitsCNYWallet(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-cny-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      10,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-cny-" + uuid.NewString(),
		Name:   "billing-cny",
	})
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO billing_wallets (user_id, currency, balance)
		VALUES ($1, 'CNY', 5)
		ON CONFLICT (user_id, currency) DO UPDATE SET balance = EXCLUDED.balance
	`, user.ID)
	require.NoError(t, err)

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:       uuid.NewString(),
		APIKeyID:        apiKey.ID,
		UserID:          user.ID,
		BillingCurrency: service.ModelPricingCurrencyCNY,
		BalanceCost:     3,
		USDToCNYRate:    6.8,
		FXRateDate:      "2026-04-24",
	})
	require.NoError(t, err)
	require.True(t, result.Applied)

	var cnyBalance, usdBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'CNY'", user.ID).Scan(&cnyBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'USD'", user.ID).Scan(&usdBalance))
	require.InDelta(t, 2, cnyBalance, 0.000001)
	require.InDelta(t, 10, usdBalance, 0.000001)

	var debitCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM billing_ledger_entries WHERE user_id = $1 AND currency = 'CNY' AND type = 'usage_debit'", user.ID).Scan(&debitCount))
	require.Equal(t, 1, debitCount)
}

func TestUsageBillingRepositoryApply_AutoFXForCNYDeficit(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-fx-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      10,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-fx-" + uuid.NewString(),
		Name:   "billing-fx",
	})
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO billing_wallets (user_id, currency, balance)
		VALUES ($1, 'CNY', 2)
		ON CONFLICT (user_id, currency) DO UPDATE SET balance = EXCLUDED.balance
	`, user.ID)
	require.NoError(t, err)

	requestID := uuid.NewString()
	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:                 requestID,
		APIKeyID:                  apiKey.ID,
		UserID:                    user.ID,
		Model:                     "deepseek-chat",
		BillingCurrency:           service.ModelPricingCurrencyCNY,
		BalanceCost:               5.4,
		USDToCNYRate:              6.8,
		CurrencyConversionEnabled: true,
		USDToCNYConversionRate:    6.8,
		CNYToUSDRate:              0.6,
		FXRateDate:                "2026-04-24",
	})
	require.NoError(t, err)
	require.True(t, result.Applied)

	var cnyBalance, usdBalance, shadowBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'CNY'", user.ID).Scan(&cnyBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'USD'", user.ID).Scan(&usdBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&shadowBalance))
	require.InDelta(t, 0, cnyBalance, 0.000001)
	require.InDelta(t, 9.5, usdBalance, 0.000001)
	require.InDelta(t, 9.5, shadowBalance, 0.000001)

	rows, err := integrationDB.QueryContext(ctx, `
		SELECT currency, type
		FROM billing_ledger_entries
		WHERE request_id = $1
		ORDER BY id ASC
	`, requestID)
	require.NoError(t, err)
	defer rows.Close()
	var entries []string
	for rows.Next() {
		var currency, entryType string
		require.NoError(t, rows.Scan(&currency, &entryType))
		entries = append(entries, currency+":"+entryType)
	}
	require.NoError(t, rows.Err())
	require.ElementsMatch(t, []string{"USD:fx_out", "CNY:fx_in", "CNY:usage_debit"}, entries)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingResolverByPath["currency_fx"])
}

func TestUsageBillingRepositoryApply_ConvertedCNYDebitPrefersUSDThenCNY(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-convert-cny-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-convert-cny-" + uuid.NewString(),
		Name:   "billing-convert-cny",
	})
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO billing_wallets (user_id, currency, balance)
		VALUES ($1, 'CNY', 5)
		ON CONFLICT (user_id, currency) DO UPDATE SET balance = EXCLUDED.balance
	`, user.ID)
	require.NoError(t, err)

	requestID := uuid.NewString()
	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:                 requestID,
		APIKeyID:                  apiKey.ID,
		UserID:                    user.ID,
		Model:                     "cny-image-model",
		BillingCurrency:           service.ModelPricingCurrencyCNY,
		BalanceCost:               10,
		CurrencyConversionEnabled: true,
		USDToCNYConversionRate:    7,
		CNYToUSDRate:              0.6,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)

	var cnyBalance, usdBalance, shadowBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'CNY'", user.ID).Scan(&cnyBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'USD'", user.ID).Scan(&usdBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&shadowBalance))
	require.InDelta(t, 2, cnyBalance, 0.000001)
	require.InDelta(t, 0, usdBalance, 0.000001)
	require.InDelta(t, 0, shadowBalance, 0.000001)

	rows, err := integrationDB.QueryContext(ctx, `
		SELECT currency, type
		FROM billing_ledger_entries
		WHERE request_id = $1
		ORDER BY id ASC
	`, requestID)
	require.NoError(t, err)
	defer rows.Close()
	var entries []string
	for rows.Next() {
		var currency, entryType string
		require.NoError(t, rows.Scan(&currency, &entryType))
		entries = append(entries, currency+":"+entryType)
	}
	require.NoError(t, rows.Err())
	require.ElementsMatch(t, []string{"USD:fx_out", "CNY:fx_in", "CNY:usage_debit", "CNY:usage_debit"}, entries)
}

func TestUsageBillingRepositoryApply_ConversionDisabledDoesNotFallbackAcrossCurrencies(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-convert-disabled-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-convert-disabled-" + uuid.NewString(),
		Name:   "billing-convert-disabled",
	})

	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:                 uuid.NewString(),
		APIKeyID:                  apiKey.ID,
		UserID:                    user.ID,
		BillingCurrency:           service.ModelPricingCurrencyCNY,
		BalanceCost:               1,
		CurrencyConversionEnabled: false,
	})
	require.ErrorIs(t, err, service.ErrInsufficientBalance)
}

func TestBillingHoldRepositoryReserveAndSettle_UsesConversionBreakdown(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	holdRepo := NewBillingHoldRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-hold-convert-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.01,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-hold-convert-" + uuid.NewString(),
		Name:   "billing-hold-convert",
	})
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO billing_wallets (user_id, currency, balance)
		VALUES ($1, 'CNY', 0.01666667)
		ON CONFLICT (user_id, currency) DO UPDATE SET balance = EXCLUDED.balance
	`, user.ID)
	require.NoError(t, err)

	requestID := uuid.NewString()
	hold, err := holdRepo.Reserve(ctx, &service.BillingHold{
		RequestID: requestID,
		APIKeyID:  apiKey.ID,
		UserID:    user.ID,
		Currency:  service.ModelPricingCurrencyUSD,
		Amount:    0.02,
		CurrencyConversion: service.BillingCurrencyConversionSettings{
			Enabled:      true,
			CNYToUSDRate: 0.6,
			USDToCNYRate: 7,
		},
	})
	require.NoError(t, err)
	require.Equal(t, service.BillingHoldStatusHeld, hold.Status)
	require.InDelta(t, 0.01666667, hold.ConversionBreakdown[service.ModelPricingCurrencyCNY], 0.000001)
	require.InDelta(t, 0.01, hold.ConversionBreakdown[service.ModelPricingCurrencyUSD], 0.000001)

	_, err = billingRepo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:                 requestID,
		APIKeyID:                  apiKey.ID,
		UserID:                    user.ID,
		Model:                     "usd-model",
		BillingCurrency:           service.ModelPricingCurrencyUSD,
		BalanceCost:               0.01,
		CurrencyConversionEnabled: true,
		CNYToUSDRate:              0.6,
		USDToCNYConversionRate:    7,
	})
	require.NoError(t, err)

	var cnyBalance, usdBalance, shadowBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'CNY'", user.ID).Scan(&cnyBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM billing_wallets WHERE user_id = $1 AND currency = 'USD'", user.ID).Scan(&usdBalance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&shadowBalance))
	require.InDelta(t, 0.00833333, cnyBalance, 0.000001)
	require.InDelta(t, 0.005, usdBalance, 0.000001)
	require.InDelta(t, 0.005, shadowBalance, 0.000001)

	var breakdown string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT conversion_breakdown::text FROM billing_request_holds WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&breakdown))
	require.Contains(t, breakdown, service.ModelPricingCurrencyCNY)
	require.Contains(t, breakdown, service.ModelPricingCurrencyUSD)
}

func TestUsageBillingRepositoryApply_AccruesAffiliateUsageRebateWithFixedPointCap(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	inviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-affiliate-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	invitee := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-affiliate-invitee-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: invitee.ID,
		Key:    "sk-usage-affiliate-" + uuid.NewString(),
		Name:   "usage-affiliate",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-affiliate-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

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
		INSERT INTO settings (key, value, updated_at)
		VALUES
			($1, 'true', NOW()),
			($2, 'true', NOW()),
			($3, '33.33333333', NOW()),
			($4, '0.03', NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`,
		service.SettingKeyAffiliateEnabled,
		service.SettingKeyAffiliateRebateOnUsageEnabled,
		service.SettingKeyAffiliateRebateRate,
		service.SettingKeyAffiliateRebatePerInviteeCap,
	)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `
		INSERT INTO user_affiliate_ledger (inviter_user_id, invitee_user_id, event_type, amount, created_at)
		VALUES ($1, $2, 'topup_accrue', 0.02, NOW())
	`, inviter.ID, invitee.ID)
	require.NoError(t, err)

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   "usage-affiliate-" + uuid.NewString(),
		APIKeyID:    apiKey.ID,
		UserID:      invitee.ID,
		AccountID:   account.ID,
		AccountType: service.AccountTypeAPIKey,
		BalanceCost: 0.1 + 0.2,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)

	var amount, baseAmount, rebateBalance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT amount, base_amount
		FROM user_affiliate_ledger
		WHERE inviter_user_id = $1 AND invitee_user_id = $2 AND event_type = 'usage_accrue'
	`, inviter.ID, invitee.ID).Scan(&amount, &baseAmount))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT rebate_balance
		FROM user_affiliates
		WHERE user_id = $1
	`, inviter.ID).Scan(&rebateBalance))

	require.InDelta(t, 0.01, amount, 0.00000001)
	require.InDelta(t, 0.3, baseAmount, 0.00000001)
	require.InDelta(t, 0.01, rebateBalance, 0.00000001)
}

func TestUsageBillingRepositoryApply_AutoFXFailsWhenUSDInsufficient(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-fx-low-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      0.1,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-fx-low-" + uuid.NewString(),
		Name:   "billing-fx-low",
	})

	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:       uuid.NewString(),
		APIKeyID:        apiKey.ID,
		UserID:          user.ID,
		BillingCurrency: service.ModelPricingCurrencyCNY,
		BalanceCost:     5,
		USDToCNYRate:    6.8,
	})
	require.ErrorIs(t, err, service.ErrInsufficientBalance)
	require.Equal(t, int64(0), protocolruntime.Snapshot().BillingResolverFallbackByReason["currency_fx_insufficient_balance"])

	_, err = repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:                 uuid.NewString(),
		APIKeyID:                  apiKey.ID,
		UserID:                    user.ID,
		BillingCurrency:           service.ModelPricingCurrencyCNY,
		BalanceCost:               5,
		CurrencyConversionEnabled: true,
		USDToCNYConversionRate:    6.8,
		CNYToUSDRate:              0.6,
	})
	require.ErrorIs(t, err, service.ErrInsufficientBalance)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingResolverFallbackByReason["currency_fx_insufficient_balance"])
}

func TestUsageBillingRepositoryApply_DeduplicatesSubscriptionBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-sub-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             "usage-billing-group-" + uuid.NewString(),
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeSubscription,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:  user.ID,
		GroupID: &group.ID,
		Key:     "sk-usage-billing-sub-" + uuid.NewString(),
		Name:    "billing-sub",
	})
	subscription := mustCreateSubscription(t, client, &service.UserSubscription{
		UserID:  user.ID,
		GroupID: group.ID,
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:        requestID,
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		AccountID:        0,
		SubscriptionID:   &subscription.ID,
		SubscriptionCost: 2.5,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result1.Applied)

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, result2.Applied)

	var dailyUsage float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT daily_usage_usd FROM user_subscriptions WHERE id = $1", subscription.ID).Scan(&dailyUsage))
	require.InDelta(t, 2.5, dailyUsage, 0.000001)
}

func TestUsageBillingRepositoryApply_RequestFingerprintConflict(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-conflict-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-conflict-" + uuid.NewString(),
		Name:   "billing-conflict",
	})

	requestID := uuid.NewString()
	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 1.25,
	})
	require.NoError(t, err)

	_, err = repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 2.50,
	})
	require.ErrorIs(t, err, service.ErrUsageBillingRequestConflict)
}

func TestUsageBillingRepositoryApply_UpdatesAccountQuota(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-account-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-account-" + uuid.NewString(),
		Name:   "billing-account",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-account-quota-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
		Extra: map[string]any{
			"quota_limit": 100.0,
		},
	})

	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		AccountID:        account.ID,
		AccountType:      service.AccountTypeAPIKey,
		AccountQuotaCost: 3.5,
	})
	require.NoError(t, err)

	var quotaUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COALESCE((extra->>'quota_used')::numeric, 0) FROM accounts WHERE id = $1", account.ID).Scan(&quotaUsed))
	require.InDelta(t, 3.5, quotaUsed, 0.000001)
}

func TestDashboardAggregationRepositoryCleanupUsageBillingDedup_BatchDeletesOldRows(t *testing.T) {
	ctx := context.Background()
	repo := newDashboardAggregationRepositoryWithSQL(integrationDB)

	oldRequestID := "dedup-old-" + uuid.NewString()
	newRequestID := "dedup-new-" + uuid.NewString()
	oldCreatedAt := time.Now().UTC().AddDate(0, 0, -400)
	newCreatedAt := time.Now().UTC().Add(-time.Hour)

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint, created_at)
		VALUES ($1, 1, $2, $3), ($4, 1, $5, $6)
	`,
		oldRequestID, strings.Repeat("a", 64), oldCreatedAt,
		newRequestID, strings.Repeat("b", 64), newCreatedAt,
	)
	require.NoError(t, err)

	require.NoError(t, repo.CleanupUsageBillingDedup(ctx, time.Now().UTC().AddDate(0, 0, -365)))

	var oldCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1", oldRequestID).Scan(&oldCount))
	require.Equal(t, 0, oldCount)

	var newCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1", newRequestID).Scan(&newCount))
	require.Equal(t, 1, newCount)

	var archivedCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup_archive WHERE request_id = $1", oldRequestID).Scan(&archivedCount))
	require.Equal(t, 1, archivedCount)
}

func TestUsageBillingRepositoryApply_DeduplicatesAgainstArchivedKey(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	aggRepo := newDashboardAggregationRepositoryWithSQL(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-archive-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-archive-" + uuid.NewString(),
		Name:   "billing-archive",
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 1.25,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result1.Applied)

	_, err = integrationDB.ExecContext(ctx, `
		UPDATE usage_billing_dedup
		SET created_at = $1
		WHERE request_id = $2 AND api_key_id = $3
	`, time.Now().UTC().AddDate(0, 0, -400), requestID, apiKey.ID)
	require.NoError(t, err)
	require.NoError(t, aggRepo.CleanupUsageBillingDedup(ctx, time.Now().UTC().AddDate(0, 0, -365)))

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, result2.Applied)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 98.75, balance, 0.000001)
}
