package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoreBillingRepositoriesUseBillingMoneyDBValues(t *testing.T) {
	files := []string{
		"billing_hold_repo.go",
		"usage_billing_repo.go",
		"usage_billing_repo_affiliate.go",
		"affiliate_repo_rebate.go",
		"payment_repo_fulfillment.go",
		"user_repo.go",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(".", file))
			require.NoError(t, err)
			source := string(body)
			require.NotContains(t, source, "baseAmount * ratePercent / 100")
			require.NotContains(t, source, "accruedAmount := baseAmount")
			require.NotContains(t, source, ", accruedAmount,")
			require.NotContains(t, source, "`, accruedAmount")
		})
	}
}

func TestCoreBillingSQLBalanceMutationsUseNormalizedMoneyParameters(t *testing.T) {
	files := []string{
		"billing_hold_repo.go",
		"usage_billing_repo.go",
		"usage_billing_repo_affiliate.go",
		"affiliate_repo_rebate.go",
		"payment_repo_fulfillment.go",
		"user_repo.go",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(".", file))
			require.NoError(t, err)
			source := string(body)
			if !strings.Contains(source, "balance") {
				return
			}
			require.Contains(t, source, "DBValue()")
		})
	}
}

func TestCoreBillingInternalHelpersAcceptBillingMoney(t *testing.T) {
	cases := []struct {
		file      string
		signature string
	}{
		{"billing_hold_repo.go", "func adjustUSDWalletBalance(ctx context.Context, tx *sql.Tx, userID int64, deltaMoney service.BillingMoney) error"},
		{"usage_billing_repo.go", "func settleUsageBillingRequestHold(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, actualMoney service.BillingMoney) (*service.BillingHold, error)"},
		{"usage_billing_repo.go", "func addUsageBillingWalletBalance(ctx context.Context, tx *sql.Tx, userID int64, currency string, deltaMoney service.BillingMoney, updateUSDShadow bool) error"},
		{"usage_billing_repo.go", "func insertUsageBillingLedgerEntry(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, currency string, amountMoney service.BillingMoney, entryType string, metadata map[string]any) error"},
		{"payment_repo_fulfillment.go", "func addWalletBalanceTx(ctx context.Context, exec sqlExecutor, userID int64, currency string, amountMoney service.BillingMoney) error"},
		{"user_repo.go", "func setUSDBillingWalletBalance(ctx context.Context, exec sqlExecutor, userID int64, balanceMoney service.BillingMoney) error"},
		{"user_repo.go", "func addUSDBillingWalletBalance(ctx context.Context, exec sqlExecutor, userID int64, deltaMoney service.BillingMoney) error"},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(".", tc.file))
			require.NoError(t, err)
			source := string(body)
			require.Contains(t, source, tc.signature)
			require.NotContains(t, source, strings.Replace(tc.signature, " service.BillingMoney", " float64", 1))
		})
	}
}
