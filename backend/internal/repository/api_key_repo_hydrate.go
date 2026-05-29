package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *apiKeyRepository) hydrateAPIKeyUserBalances(ctx context.Context, keys []*service.APIKey) error {
	if r == nil || r.sql == nil || len(keys) == 0 {
		return nil
	}
	users := make(map[int64]*service.User)
	userIDs := make([]int64, 0, len(keys))
	for _, key := range keys {
		if key == nil || key.User == nil {
			continue
		}
		if _, exists := users[key.User.ID]; exists {
			continue
		}
		key.User.Balances = map[string]float64{service.ModelPricingCurrencyUSD: key.User.Balance}
		users[key.User.ID] = key.User
		userIDs = append(userIDs, key.User.ID)
	}
	if len(userIDs) == 0 {
		return nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT user_id, currency, balance
		FROM billing_wallets
		WHERE user_id = ANY($1)
	`, pq.Array(userIDs))
	if err != nil {
		if isUndefinedBillingWalletTable(err) {
			return nil
		}
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			userID   int64
			currency string
			balance  float64
		)
		if err := rows.Scan(&userID, &currency, &balance); err != nil {
			return err
		}
		user := users[userID]
		if user == nil {
			continue
		}
		normalized := service.NormalizeUsageBillingCurrency(currency)
		if normalized == "" {
			continue
		}
		if user.Balances == nil {
			user.Balances = map[string]float64{}
		}
		user.Balances[normalized] = balance
	}
	return rows.Err()
}
