//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func mustSetBillingWallet(t *testing.T, ctx context.Context, userID int64, currency string, balance float64) {
	t.Helper()

	currency = service.NormalizeUsageBillingCurrency(currency)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO billing_wallets (user_id, currency, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, currency) DO UPDATE SET balance = EXCLUDED.balance, updated_at = NOW()
	`, userID, currency, balance)
	require.NoError(t, err)

	if currency == service.ModelPricingCurrencyUSD {
		_, err = integrationDB.ExecContext(ctx, `
			UPDATE users
			SET balance = $2, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, userID, balance)
		require.NoError(t, err)
	}
}
