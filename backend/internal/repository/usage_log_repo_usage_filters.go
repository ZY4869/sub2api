package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *usageLogRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	platform := normalizeUsagePlatformFilter(filters.Platform)
	withPlatformJoin := platform != ""
	conditions := make([]string, 0, 9)
	args := make([]any, 0, 8)
	if filters.UserID > 0 {
		conditions = append(conditions, fmt.Sprintf("%suser_id = $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, filters.UserID)
	}
	if filters.APIKeyID > 0 {
		conditions = append(conditions, fmt.Sprintf("%sapi_key_id = $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, filters.APIKeyID)
	}
	if filters.AccountID > 0 {
		conditions = append(conditions, fmt.Sprintf("%saccount_id = $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, filters.AccountID)
	}
	if filters.GroupID > 0 {
		conditions = append(conditions, fmt.Sprintf("%sgroup_id = $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, filters.GroupID)
	}
	if filters.ChannelID > 0 {
		conditions = append(conditions, fmt.Sprintf("%schannel_id = $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, filters.ChannelID)
	}
	conditions, args = appendRawUsageLogModelWhereConditionForColumn(conditions, args, usageLogColumnPrefix(withPlatformJoin)+rawUsageLogModelColumn, filters.Model)
	conditions, args = appendRequestTypeOrStreamWhereConditionWithPrefix(conditions, args, usageLogColumnPrefix(withPlatformJoin), filters.RequestType, filters.Stream)
	if filters.BillingType != nil {
		conditions = append(conditions, fmt.Sprintf("%sbilling_type = $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, int16(*filters.BillingType))
	}
	if filters.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("%screated_at >= $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, *filters.StartTime)
	}
	if filters.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("%screated_at < $%d", usageLogColumnPrefix(withPlatformJoin), len(args)+1))
		args = append(args, *filters.EndTime)
	}
	if platform != "" {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", usagePlatformExpression, len(args)+1))
		args = append(args, platform)
	}
	whereClause := buildWhere(conditions)
	fromClause := "usage_logs"
	if withPlatformJoin {
		fromClause = usageLogPlatformJoinFromClause()
	}
	var (
		logs []service.UsageLog
		page *pagination.PaginationResult
		err  error
	)
	columnPrefix := usageLogColumnPrefix(withPlatformJoin)
	if shouldUseFastUsageLogTotal(filters) {
		logs, page, err = r.listUsageLogsFromWithFastPagination(ctx, fromClause, columnPrefix, whereClause, args, params)
	} else {
		logs, page, err = r.listUsageLogsFromWithPagination(ctx, fromClause, columnPrefix, whereClause, args, params)
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
	return filters.UserID == 0 && filters.APIKeyID == 0 && filters.AccountID == 0 && filters.ChannelID == 0 && normalizeUsagePlatformFilter(filters.Platform) == ""
}

func normalizeUsagePlatformFilter(platform string) string {
	normalized := service.CanonicalizePlatformValue(platform)
	if normalized == "" || service.IsUnsupportedPrimaryPlatform(normalized) {
		return ""
	}
	return normalized
}

func usageLogColumnPrefix(withPlatformJoin bool) string {
	if withPlatformJoin {
		return "ul."
	}
	return ""
}

func usageLogPlatformJoinFromClause() string {
	return "usage_logs ul LEFT JOIN accounts a ON a.id = ul.account_id LEFT JOIN groups g ON g.id = ul.group_id"
}
