package repository

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"os"
	"strings"
	"time"
)

func (r *usageLogRepository) GetUserStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
	`
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{userID, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheTokens
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE user_id = $1 AND created_at >= $2 AND created_at < $3", []any{userID, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetAPIKeyStatsAggregated(ctx context.Context, apiKeyID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE api_key_id = $1 AND created_at >= $2 AND created_at < $3
	`
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{apiKeyID, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheTokens
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE api_key_id = $1 AND created_at >= $2 AND created_at < $3", []any{apiKeyID, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetAccountStatsAggregated(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE account_id = $1 AND created_at >= $2 AND created_at < $3
	`
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{accountID, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheTokens
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE account_id = $1 AND created_at >= $2 AND created_at < $3", []any{accountID, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetModelStatsAggregated(ctx context.Context, modelName string, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE model = $1 AND created_at >= $2 AND created_at < $3
	`
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{modelName, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheTokens
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE model = $1 AND created_at >= $2 AND created_at < $3", []any{modelName, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetDailyStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) (result []map[string]any, err error) {
	tzName := resolveUsageStatsTimezone()
	query := `
		SELECT
			-- 使用应用时区分组，避免数据库会话时区导致日边界偏移。
			TO_CHAR(created_at AT TIME ZONE $4, 'YYYY-MM-DD') as date,
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY 1
		ORDER BY 1
	`
	rows, err := r.sql.QueryContext(ctx, query, userID, startTime, endTime, tzName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()
	result = make([]map[string]any, 0)
	for rows.Next() {
		var (
			date              string
			totalRequests     int64
			totalInputTokens  int64
			totalOutputTokens int64
			totalCacheTokens  int64
			totalCost         float64
			totalActualCost   float64
			avgDurationMs     float64
		)
		if err = rows.Scan(&date, &totalRequests, &totalInputTokens, &totalOutputTokens, &totalCacheTokens, &totalCost, &totalActualCost, &avgDurationMs); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{"date": date, "total_requests": totalRequests, "total_input_tokens": totalInputTokens, "total_output_tokens": totalOutputTokens, "total_cache_tokens": totalCacheTokens, "total_tokens": totalInputTokens + totalOutputTokens + totalCacheTokens, "total_cost": totalCost, "total_actual_cost": totalActualCost, "average_duration_ms": avgDurationMs})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
func resolveUsageStatsTimezone() string {
	tzName := timezone.Name()
	if tzName != "" && tzName != "Local" {
		return tzName
	}
	if envTZ := strings.TrimSpace(os.Getenv("TZ")); envTZ != "" {
		return envTZ
	}
	return "UTC"
}
func (r *usageLogRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.sql.ExecContext(ctx, "DELETE FROM usage_logs WHERE id = $1", id)
	return err
}
func (r *usageLogRepository) GetAccountTodayStats(ctx context.Context, accountID int64) (*usagestats.AccountStats, error) {
	today := timezone.Today()
	query := `
		SELECT
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as cost,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as user_cost
		FROM usage_logs
		WHERE account_id = $1 AND created_at >= $2
	`
	stats := &usagestats.AccountStats{}
	if err := scanSingleRow(ctx, r.sql, query, []any{accountID, today}, &stats.Requests, &stats.Tokens, &stats.Cost, &stats.StandardCost, &stats.UserCost); err != nil {
		return nil, err
	}
	return stats, nil
}
func (r *usageLogRepository) GetAccountWindowStats(ctx context.Context, accountID int64, startTime time.Time) (*usagestats.AccountStats, error) {
	query := `
		SELECT
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as cost,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as user_cost
		FROM usage_logs
		WHERE account_id = $1 AND created_at >= $2
	`
	stats := &usagestats.AccountStats{}
	if err := scanSingleRow(ctx, r.sql, query, []any{accountID, startTime}, &stats.Requests, &stats.Tokens, &stats.Cost, &stats.StandardCost, &stats.UserCost); err != nil {
		return nil, err
	}
	return stats, nil
}
func (r *usageLogRepository) GetAccountWindowStatsBatch(ctx context.Context, accountIDs []int64, startTime time.Time) (map[int64]*usagestats.AccountStats, error) {
	result := make(map[int64]*usagestats.AccountStats, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT
			account_id,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as cost,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as user_cost
		FROM usage_logs
		WHERE account_id = ANY($1) AND created_at >= $2
		GROUP BY account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(accountIDs), startTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var accountID int64
		stats := &usagestats.AccountStats{}
		if err := rows.Scan(&accountID, &stats.Requests, &stats.Tokens, &stats.Cost, &stats.StandardCost, &stats.UserCost); err != nil {
			return nil, err
		}
		result[accountID] = stats
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, accountID := range accountIDs {
		if _, ok := result[accountID]; !ok {
			result[accountID] = &usagestats.AccountStats{}
		}
	}
	return result, nil
}
func (r *usageLogRepository) GetGeminiUsageTotalsBatch(ctx context.Context, accountIDs []int64, startTime, endTime time.Time) (map[int64]service.GeminiUsageTotals, error) {
	result := make(map[int64]service.GeminiUsageTotals, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT
			account_id,
			COALESCE(SUM(CASE WHEN LOWER(COALESCE(model, '')) LIKE '%flash%' OR LOWER(COALESCE(model, '')) LIKE '%lite%' THEN 1 ELSE 0 END), 0) AS flash_requests,
			COALESCE(SUM(CASE WHEN LOWER(COALESCE(model, '')) LIKE '%flash%' OR LOWER(COALESCE(model, '')) LIKE '%lite%' THEN 0 ELSE 1 END), 0) AS pro_requests,
			COALESCE(SUM(CASE WHEN LOWER(COALESCE(model, '')) LIKE '%flash%' OR LOWER(COALESCE(model, '')) LIKE '%lite%' THEN (input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) ELSE 0 END), 0) AS flash_tokens,
			COALESCE(SUM(CASE WHEN LOWER(COALESCE(model, '')) LIKE '%flash%' OR LOWER(COALESCE(model, '')) LIKE '%lite%' THEN 0 ELSE (input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) END), 0) AS pro_tokens,
			COALESCE(SUM(CASE WHEN LOWER(COALESCE(model, '')) LIKE '%flash%' OR LOWER(COALESCE(model, '')) LIKE '%lite%' THEN actual_cost_usd_equivalent ELSE 0 END), 0) AS flash_cost,
			COALESCE(SUM(CASE WHEN LOWER(COALESCE(model, '')) LIKE '%flash%' OR LOWER(COALESCE(model, '')) LIKE '%lite%' THEN 0 ELSE actual_cost_usd_equivalent END), 0) AS pro_cost
		FROM usage_logs
		WHERE account_id = ANY($1) AND created_at >= $2 AND created_at < $3
		GROUP BY account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(accountIDs), startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var accountID int64
		var totals service.GeminiUsageTotals
		if err := rows.Scan(&accountID, &totals.FlashRequests, &totals.ProRequests, &totals.FlashTokens, &totals.ProTokens, &totals.FlashCost, &totals.ProCost); err != nil {
			return nil, err
		}
		result[accountID] = totals
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, accountID := range accountIDs {
		if _, ok := result[accountID]; !ok {
			result[accountID] = service.GeminiUsageTotals{}
		}
	}
	return result, nil
}

type TrendDataPoint = usagestats.TrendDataPoint
type ModelStat = usagestats.ModelStat
type UserUsageTrendPoint = usagestats.UserUsageTrendPoint
type APIKeyUsageTrendPoint = usagestats.APIKeyUsageTrendPoint

func (r *usageLogRepository) GetAPIKeyUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) (results []APIKeyUsageTrendPoint, err error) {
	dateFormat := safeDateFormat(granularity)
	query := fmt.Sprintf(`
		WITH top_keys AS (
			SELECT api_key_id
			FROM usage_logs
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY api_key_id
			ORDER BY SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) DESC
			LIMIT $3
		)
		SELECT
			TO_CHAR(u.created_at, '%s') as date,
			u.api_key_id,
			COALESCE(k.name, '') as key_name,
			COUNT(*) as requests,
			COALESCE(SUM(u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens), 0) as tokens
		FROM usage_logs u
		LEFT JOIN api_keys k ON u.api_key_id = k.id
		WHERE u.api_key_id IN (SELECT api_key_id FROM top_keys)
		  AND u.created_at >= $4 AND u.created_at < $5
		GROUP BY date, u.api_key_id, k.name
		ORDER BY date ASC, tokens DESC
	`, dateFormat)
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results = make([]APIKeyUsageTrendPoint, 0)
	for rows.Next() {
		var row APIKeyUsageTrendPoint
		if err = rows.Scan(&row.Date, &row.APIKeyID, &row.KeyName, &row.Requests, &row.Tokens); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
func (r *usageLogRepository) GetUserUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) (results []UserUsageTrendPoint, err error) {
	dateFormat := safeDateFormat(granularity)
	query := fmt.Sprintf(`
		WITH top_users AS (
			SELECT user_id
			FROM usage_logs
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY user_id
			ORDER BY SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) DESC
			LIMIT $3
		)
		SELECT
			TO_CHAR(u.created_at, '%s') as date,
			u.user_id,
			COALESCE(us.email, '') as email,
			COUNT(*) as requests,
			COALESCE(SUM(u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens), 0) as tokens,
			COALESCE(SUM(u.total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(u.actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs u
		LEFT JOIN users us ON u.user_id = us.id
		WHERE u.user_id IN (SELECT user_id FROM top_users)
		  AND u.created_at >= $4 AND u.created_at < $5
		GROUP BY date, u.user_id, us.email
		ORDER BY date ASC, tokens DESC
	`, dateFormat)
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results = make([]UserUsageTrendPoint, 0)
	for rows.Next() {
		var row UserUsageTrendPoint
		if err = rows.Scan(&row.Date, &row.UserID, &row.Email, &row.Requests, &row.Tokens, &row.Cost, &row.ActualCost); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

type UserDashboardStats = usagestats.UserDashboardStats

func (r *usageLogRepository) GetUserDashboardStats(ctx context.Context, userID int64) (*UserDashboardStats, error) {
	stats := &UserDashboardStats{}
	today := timezone.Today()
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND deleted_at IS NULL", []any{userID}, &stats.TotalAPIKeys); err != nil {
		return nil, err
	}
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND status = $2 AND deleted_at IS NULL", []any{userID, service.StatusActive}, &stats.ActiveAPIKeys); err != nil {
		return nil, err
	}
	totalStatsQuery := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE user_id = $1
	`
	if err := scanSingleRow(ctx, r.sql, totalStatsQuery, []any{userID}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE user_id = $1", []any{userID})
	if err != nil {
		return nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	todayStatsQuery := `
		SELECT
			COUNT(*) as today_requests,
			COALESCE(SUM(input_tokens), 0) as today_input_tokens,
			COALESCE(SUM(output_tokens), 0) as today_output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as today_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as today_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as today_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as today_actual_cost
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2
	`
	if err := scanSingleRow(ctx, r.sql, todayStatsQuery, []any{userID, today}, &stats.TodayRequests, &stats.TodayInputTokens, &stats.TodayOutputTokens, &stats.TodayCacheCreationTokens, &stats.TodayCacheReadTokens, &stats.TodayCost, &stats.TodayActualCost); err != nil {
		return nil, err
	}
	stats.TodayTokens = stats.TodayInputTokens + stats.TodayOutputTokens + stats.TodayCacheCreationTokens + stats.TodayCacheReadTokens
	todayCostByCurrency, todayActualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE user_id = $1 AND created_at >= $2", []any{userID, today})
	if err != nil {
		return nil, err
	}
	stats.TodayCostByCurrency = todayCostByCurrency
	stats.TodayActualCostByCurrency = todayActualCostByCurrency
	rpm, tpm, err := r.getPerformanceStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.Rpm = rpm
	stats.Tpm = tpm
	return stats, nil
}
func (r *usageLogRepository) getPerformanceStatsByAPIKey(ctx context.Context, apiKeyID int64) (rpm, tpm int64, err error) {
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	query := `
		SELECT
			COUNT(*) as request_count,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as token_count
		FROM usage_logs
		WHERE created_at >= $1 AND api_key_id = $2`
	args := []any{fiveMinutesAgo, apiKeyID}
	var requestCount int64
	var tokenCount int64
	if err := scanSingleRow(ctx, r.sql, query, args, &requestCount, &tokenCount); err != nil {
		return 0, 0, err
	}
	return requestCount / 5, tokenCount / 5, nil
}
func (r *usageLogRepository) GetAPIKeyDashboardStats(ctx context.Context, apiKeyID int64) (*UserDashboardStats, error) {
	stats := &UserDashboardStats{}
	today := timezone.Today()
	stats.TotalAPIKeys = 1
	stats.ActiveAPIKeys = 1
	totalStatsQuery := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE api_key_id = $1
	`
	if err := scanSingleRow(ctx, r.sql, totalStatsQuery, []any{apiKeyID}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE api_key_id = $1", []any{apiKeyID})
	if err != nil {
		return nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	todayStatsQuery := `
		SELECT
			COUNT(*) as today_requests,
			COALESCE(SUM(input_tokens), 0) as today_input_tokens,
			COALESCE(SUM(output_tokens), 0) as today_output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as today_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as today_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as today_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as today_actual_cost
		FROM usage_logs
		WHERE api_key_id = $1 AND created_at >= $2
	`
	if err := scanSingleRow(ctx, r.sql, todayStatsQuery, []any{apiKeyID, today}, &stats.TodayRequests, &stats.TodayInputTokens, &stats.TodayOutputTokens, &stats.TodayCacheCreationTokens, &stats.TodayCacheReadTokens, &stats.TodayCost, &stats.TodayActualCost); err != nil {
		return nil, err
	}
	stats.TodayTokens = stats.TodayInputTokens + stats.TodayOutputTokens + stats.TodayCacheCreationTokens + stats.TodayCacheReadTokens
	todayCostByCurrency, todayActualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE api_key_id = $1 AND created_at >= $2", []any{apiKeyID, today})
	if err != nil {
		return nil, err
	}
	stats.TodayCostByCurrency = todayCostByCurrency
	stats.TodayActualCostByCurrency = todayActualCostByCurrency
	rpm, tpm, err := r.getPerformanceStatsByAPIKey(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}
	stats.Rpm = rpm
	stats.Tpm = tpm
	return stats, nil
}
func (r *usageLogRepository) GetUserUsageTrendByUserID(ctx context.Context, userID int64, startTime, endTime time.Time, granularity string) (results []TrendDataPoint, err error) {
	dateFormat := safeDateFormat(granularity)
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, '%s') as date,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY date
		ORDER BY date ASC
	`, dateFormat)
	rows, err := r.sql.QueryContext(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results, err = scanTrendRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}
func (r *usageLogRepository) GetUserModelStats(ctx context.Context, userID int64, startTime, endTime time.Time) (results []ModelStat, err error) {
	query := `
		SELECT
			model,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY model
		ORDER BY total_tokens DESC
	`
	rows, err := r.sql.QueryContext(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results, err = scanModelStatsRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}

type UsageLogFilters = usagestats.UsageLogFilters

func (r *usageLogRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	conditions := make([]string, 0, 8)
	args := make([]any, 0, 8)
	if filters.UserID > 0 {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)+1))
		args = append(args, filters.UserID)
	}
	if filters.APIKeyID > 0 {
		conditions = append(conditions, fmt.Sprintf("api_key_id = $%d", len(args)+1))
		args = append(args, filters.APIKeyID)
	}
	if filters.AccountID > 0 {
		conditions = append(conditions, fmt.Sprintf("account_id = $%d", len(args)+1))
		args = append(args, filters.AccountID)
	}
	if filters.GroupID > 0 {
		conditions = append(conditions, fmt.Sprintf("group_id = $%d", len(args)+1))
		args = append(args, filters.GroupID)
	}
	if filters.ChannelID > 0 {
		conditions = append(conditions, fmt.Sprintf("channel_id = $%d", len(args)+1))
		args = append(args, filters.ChannelID)
	}
	conditions, args = appendRawUsageLogModelWhereCondition(conditions, args, filters.Model)
	conditions, args = appendRequestTypeOrStreamWhereCondition(conditions, args, filters.RequestType, filters.Stream)
	if filters.BillingType != nil {
		conditions = append(conditions, fmt.Sprintf("billing_type = $%d", len(args)+1))
		args = append(args, int16(*filters.BillingType))
	}
	if filters.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, *filters.StartTime)
	}
	if filters.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, *filters.EndTime)
	}
	whereClause := buildWhere(conditions)
	var (
		logs []service.UsageLog
		page *pagination.PaginationResult
		err  error
	)
	if shouldUseFastUsageLogTotal(filters) {
		logs, page, err = r.listUsageLogsWithFastPagination(ctx, whereClause, args, params)
	} else {
		logs, page, err = r.listUsageLogsWithPagination(ctx, whereClause, args, params)
	}
	if err != nil {
		return nil, nil, err
	}
	if err := r.hydrateUsageLogAssociations(ctx, logs); err != nil {
		return nil, nil, err
	}
	return logs, page, nil
}
func shouldUseFastUsageLogTotal(filters UsageLogFilters) bool {
	if filters.ExactTotal {
		return false
	}
	return filters.UserID == 0 && filters.APIKeyID == 0 && filters.AccountID == 0 && filters.ChannelID == 0
}

type UsageStats = usagestats.UsageStats
type BatchUserUsageStats = usagestats.BatchUserUsageStats

func normalizePositiveInt64IDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
func (r *usageLogRepository) GetBatchUserUsageStats(ctx context.Context, userIDs []int64, startTime, endTime time.Time) (map[int64]*BatchUserUsageStats, error) {
	result := make(map[int64]*BatchUserUsageStats)
	normalizedUserIDs := normalizePositiveInt64IDs(userIDs)
	if len(normalizedUserIDs) == 0 {
		return result, nil
	}
	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}
	for _, id := range normalizedUserIDs {
		result[id] = &BatchUserUsageStats{UserID: id}
	}
	query := `
		SELECT
			user_id,
			COALESCE(SUM(actual_cost_usd_equivalent) FILTER (WHERE created_at >= $2 AND created_at < $3), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent) FILTER (WHERE created_at >= $4), 0) as today_cost
		FROM usage_logs
		WHERE user_id = ANY($1)
		  AND created_at >= LEAST($2, $4)
		GROUP BY user_id
	`
	today := timezone.Today()
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(normalizedUserIDs), startTime, endTime, today)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var userID int64
		var total float64
		var todayTotal float64
		if err := rows.Scan(&userID, &total, &todayTotal); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if stats, ok := result[userID]; ok {
			stats.TotalActualCost = total
			stats.TodayActualCost = todayTotal
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type BatchAPIKeyUsageStats = usagestats.BatchAPIKeyUsageStats

func (r *usageLogRepository) GetBatchAPIKeyUsageStats(ctx context.Context, apiKeyIDs []int64, startTime, endTime time.Time) (map[int64]*BatchAPIKeyUsageStats, error) {
	result := make(map[int64]*BatchAPIKeyUsageStats)
	normalizedAPIKeyIDs := normalizePositiveInt64IDs(apiKeyIDs)
	if len(normalizedAPIKeyIDs) == 0 {
		return result, nil
	}
	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}
	for _, id := range normalizedAPIKeyIDs {
		result[id] = &BatchAPIKeyUsageStats{APIKeyID: id}
	}
	query := `
		SELECT
			api_key_id,
			COALESCE(SUM(actual_cost_usd_equivalent) FILTER (WHERE created_at >= $2 AND created_at < $3), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent) FILTER (WHERE created_at >= $4), 0) as today_cost
		FROM usage_logs
		WHERE api_key_id = ANY($1)
		  AND created_at >= LEAST($2, $4)
		GROUP BY api_key_id
	`
	today := timezone.Today()
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(normalizedAPIKeyIDs), startTime, endTime, today)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var apiKeyID int64
		var total float64
		var todayTotal float64
		if err := rows.Scan(&apiKeyID, &total, &todayTotal); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if stats, ok := result[apiKeyID]; ok {
			stats.TotalActualCost = total
			stats.TodayActualCost = todayTotal
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
func (r *usageLogRepository) GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID, channelID int64, model string, requestType *int16, stream *bool, billingType *int8) (results []TrendDataPoint, err error) {
	if shouldUsePreaggregatedTrend(granularity, userID, apiKeyID, accountID, groupID, channelID, model, requestType, stream, billingType) {
		aggregated, aggregatedErr := r.getUsageTrendFromAggregates(ctx, startTime, endTime, granularity)
		if aggregatedErr == nil && len(aggregated) > 0 {
			return aggregated, nil
		}
	}
	dateFormat := safeDateFormat(granularity)
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, '%s') as date,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
	`, dateFormat)
	args := []any{startTime, endTime}
	if userID > 0 {
		query += fmt.Sprintf(" AND user_id = $%d", len(args)+1)
		args = append(args, userID)
	}
	if apiKeyID > 0 {
		query += fmt.Sprintf(" AND api_key_id = $%d", len(args)+1)
		args = append(args, apiKeyID)
	}
	if accountID > 0 {
		query += fmt.Sprintf(" AND account_id = $%d", len(args)+1)
		args = append(args, accountID)
	}
	if groupID > 0 {
		query += fmt.Sprintf(" AND group_id = $%d", len(args)+1)
		args = append(args, groupID)
	}
	if channelID > 0 {
		query += fmt.Sprintf(" AND channel_id = $%d", len(args)+1)
		args = append(args, channelID)
	}
	query, args = appendRawUsageLogModelQueryFilter(query, args, model)
	query, args = appendRequestTypeOrStreamQueryFilter(query, args, requestType, stream)
	if billingType != nil {
		query += fmt.Sprintf(" AND billing_type = $%d", len(args)+1)
		args = append(args, int16(*billingType))
	}
	query += " GROUP BY date ORDER BY date ASC"
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results, err = scanTrendRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}
func shouldUsePreaggregatedTrend(granularity string, userID, apiKeyID, accountID, groupID, channelID int64, model string, requestType *int16, stream *bool, billingType *int8) bool {
	if granularity != "day" && granularity != "hour" {
		return false
	}
	return userID == 0 && apiKeyID == 0 && accountID == 0 && groupID == 0 && channelID == 0 && model == "" && requestType == nil && stream == nil && billingType == nil
}
func (r *usageLogRepository) getUsageTrendFromAggregates(ctx context.Context, startTime, endTime time.Time, granularity string) (results []TrendDataPoint, err error) {
	dateFormat := safeDateFormat(granularity)
	query := ""
	args := []any{startTime, endTime}
	switch granularity {
	case "hour":
		query = fmt.Sprintf(`
			SELECT
				TO_CHAR(bucket_start, '%s') as date,
				total_requests as requests,
				input_tokens,
				output_tokens,
				cache_creation_tokens,
				cache_read_tokens,
				(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) as total_tokens,
				total_cost as cost,
				actual_cost
			FROM usage_dashboard_hourly
			WHERE bucket_start >= $1 AND bucket_start < $2
			ORDER BY bucket_start ASC
		`, dateFormat)
	case "day":
		query = fmt.Sprintf(`
			SELECT
				TO_CHAR(bucket_date::timestamp, '%s') as date,
				total_requests as requests,
				input_tokens,
				output_tokens,
				cache_creation_tokens,
				cache_read_tokens,
				(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) as total_tokens,
				total_cost as cost,
				actual_cost
			FROM usage_dashboard_daily
			WHERE bucket_date >= $1::date AND bucket_date < $2::date
			ORDER BY bucket_date ASC
		`, dateFormat)
	default:
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results, err = scanTrendRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}
func (r *usageLogRepository) GetModelStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) (results []ModelStat, err error) {
	return r.getModelStatsWithFiltersBySource(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, channelID, requestType, stream, billingType, usagestats.ModelSourceRequested)
}

func (r *usageLogRepository) GetModelStatsWithFiltersBySource(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8, source string) (results []ModelStat, err error) {
	return r.getModelStatsWithFiltersBySource(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, channelID, requestType, stream, billingType, source)
}

func (r *usageLogRepository) getModelStatsWithFiltersBySource(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8, source string) (results []ModelStat, err error) {
	actualCostExpr := "COALESCE(SUM(actual_cost_usd_equivalent), 0) as actual_cost"
	if accountID > 0 && userID == 0 && apiKeyID == 0 {
		actualCostExpr = "COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as actual_cost"
	}
	modelExpr := resolveModelDimensionExpression(source)
	query := fmt.Sprintf(`
		SELECT
			%s as model,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as cost,
			%s
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
	`, modelExpr, actualCostExpr)
	args := []any{startTime, endTime}
	if userID > 0 {
		query += fmt.Sprintf(" AND user_id = $%d", len(args)+1)
		args = append(args, userID)
	}
	if apiKeyID > 0 {
		query += fmt.Sprintf(" AND api_key_id = $%d", len(args)+1)
		args = append(args, apiKeyID)
	}
	if accountID > 0 {
		query += fmt.Sprintf(" AND account_id = $%d", len(args)+1)
		args = append(args, accountID)
	}
	if groupID > 0 {
		query += fmt.Sprintf(" AND group_id = $%d", len(args)+1)
		args = append(args, groupID)
	}
	if channelID > 0 {
		query += fmt.Sprintf(" AND channel_id = $%d", len(args)+1)
		args = append(args, channelID)
	}
	query, args = appendRequestTypeOrStreamQueryFilter(query, args, requestType, stream)
	if billingType != nil {
		query += fmt.Sprintf(" AND billing_type = $%d", len(args)+1)
		args = append(args, int16(*billingType))
	}
	query += fmt.Sprintf(" GROUP BY %s ORDER BY total_tokens DESC", modelExpr)
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results, err = scanModelStatsRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func resolveModelDimensionExpression(modelType string) string {
	requestedExpr := "COALESCE(NULLIF(TRIM(requested_model), ''), model)"
	switch usagestats.NormalizeModelSource(modelType) {
	case usagestats.ModelSourceUpstream:
		return fmt.Sprintf("COALESCE(NULLIF(TRIM(upstream_model), ''), %s)", requestedExpr)
	case usagestats.ModelSourceMapping:
		return fmt.Sprintf("(%s || ' -> ' || COALESCE(NULLIF(TRIM(upstream_model), ''), %s))", requestedExpr, requestedExpr)
	default:
		return requestedExpr
	}
}
func (r *usageLogRepository) GetGroupStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) (results []usagestats.GroupStat, err error) {
	query := `
		SELECT
			COALESCE(ul.group_id, 0) as group_id,
			COALESCE(g.name, '') as group_name,
			COUNT(*) as requests,
			COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(ul.total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(ul.actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		WHERE ul.created_at >= $1 AND ul.created_at < $2
	`
	args := []any{startTime, endTime}
	if userID > 0 {
		query += fmt.Sprintf(" AND ul.user_id = $%d", len(args)+1)
		args = append(args, userID)
	}
	if apiKeyID > 0 {
		query += fmt.Sprintf(" AND ul.api_key_id = $%d", len(args)+1)
		args = append(args, apiKeyID)
	}
	if accountID > 0 {
		query += fmt.Sprintf(" AND ul.account_id = $%d", len(args)+1)
		args = append(args, accountID)
	}
	if groupID > 0 {
		query += fmt.Sprintf(" AND ul.group_id = $%d", len(args)+1)
		args = append(args, groupID)
	}
	if channelID > 0 {
		query += fmt.Sprintf(" AND ul.channel_id = $%d", len(args)+1)
		args = append(args, channelID)
	}
	query, args = appendRequestTypeOrStreamQueryFilter(query, args, requestType, stream)
	if billingType != nil {
		query += fmt.Sprintf(" AND ul.billing_type = $%d", len(args)+1)
		args = append(args, int16(*billingType))
	}
	query += " GROUP BY ul.group_id, g.name ORDER BY total_tokens DESC"
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results = make([]usagestats.GroupStat, 0)
	for rows.Next() {
		var row usagestats.GroupStat
		if err := rows.Scan(&row.GroupID, &row.GroupName, &row.Requests, &row.TotalTokens, &row.Cost, &row.ActualCost); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
func (r *usageLogRepository) GetGlobalStats(ctx context.Context, startTime, endTime time.Time) (*UsageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE created_at >= $1 AND created_at <= $2
	`
	stats := &UsageStats{}
	if err := scanSingleRow(ctx, r.sql, query, []any{startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheTokens
	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE created_at >= $1 AND created_at <= $2", []any{startTime, endTime})
	if err != nil {
		return nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return stats, nil
}
func (r *usageLogRepository) GetStatsWithFilters(ctx context.Context, filters UsageLogFilters) (*UsageStats, error) {
	conditions := make([]string, 0, 9)
	args := make([]any, 0, 9)
	if filters.UserID > 0 {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)+1))
		args = append(args, filters.UserID)
	}
	if filters.APIKeyID > 0 {
		conditions = append(conditions, fmt.Sprintf("api_key_id = $%d", len(args)+1))
		args = append(args, filters.APIKeyID)
	}
	if filters.AccountID > 0 {
		conditions = append(conditions, fmt.Sprintf("account_id = $%d", len(args)+1))
		args = append(args, filters.AccountID)
	}
	if filters.GroupID > 0 {
		conditions = append(conditions, fmt.Sprintf("group_id = $%d", len(args)+1))
		args = append(args, filters.GroupID)
	}
	if filters.ChannelID > 0 {
		conditions = append(conditions, fmt.Sprintf("channel_id = $%d", len(args)+1))
		args = append(args, filters.ChannelID)
	}
	conditions, args = appendRawUsageLogModelWhereCondition(conditions, args, filters.Model)
	conditions, args = appendRequestTypeOrStreamWhereCondition(conditions, args, filters.RequestType, filters.Stream)
	if filters.BillingType != nil {
		conditions = append(conditions, fmt.Sprintf("billing_type = $%d", len(args)+1))
		args = append(args, int16(*filters.BillingType))
	}
	if filters.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, *filters.StartTime)
	}
	if filters.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, *filters.EndTime)
	}
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as total_account_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		%s
	`, buildWhere(conditions))
	stats := &UsageStats{}
	var totalAccountCost float64
	if err := scanSingleRow(ctx, r.sql, query, args, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &totalAccountCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	if filters.AccountID > 0 {
		stats.TotalAccountCost = &totalAccountCost
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheTokens
	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, buildWhere(conditions), args)
	if err != nil {
		return nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return stats, nil
}

type AccountUsageHistory = usagestats.AccountUsageHistory
type AccountUsageSummary = usagestats.AccountUsageSummary
