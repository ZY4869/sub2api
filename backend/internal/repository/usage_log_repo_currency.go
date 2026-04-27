package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func queryUsageCostByCurrency(ctx context.Context, q sqlQueryer, whereClause string, args []any) (map[string]float64, map[string]float64, error) {
	query := `
		SELECT
			COALESCE(jsonb_object_agg(currency, total_amount), '{}'::jsonb) AS cost_by_currency,
			COALESCE(jsonb_object_agg(currency, actual_amount), '{}'::jsonb) AS actual_cost_by_currency
		FROM (
			SELECT
				UPPER(COALESCE(NULLIF(TRIM(billing_currency), ''), 'USD')) AS currency,
				COALESCE(SUM(total_cost), 0) AS total_amount,
				COALESCE(SUM(actual_cost), 0) AS actual_amount
			FROM usage_logs
			` + whereClause + `
			GROUP BY 1
		) by_currency
	`
	var totalRaw, actualRaw []byte
	if err := scanSingleRow(ctx, q, query, args, &totalRaw, &actualRaw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	total := service.CloneBillingCurrencyMap(parseBillingCurrencyJSONMap(totalRaw))
	if total == nil {
		total = map[string]float64{}
	}
	actual := service.CloneBillingCurrencyMap(parseBillingCurrencyJSONMap(actualRaw))
	if actual == nil {
		actual = map[string]float64{}
	}
	return total, actual, nil
}
