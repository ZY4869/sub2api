package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func buildUsageWhere(filter *service.OpsDashboardFilter, start, end time.Time, startIndex int) (join string, where string, args []any, nextIndex int) {
	platform := ""
	groupID := (*int64)(nil)
	channelID := (*int64)(nil)
	if filter != nil {
		platform = strings.TrimSpace(strings.ToLower(filter.Platform))
		groupID = filter.GroupID
		channelID = filter.ChannelID
	}

	idx := startIndex
	clauses := make([]string, 0, 4)
	args = make([]any, 0, 4)

	args = append(args, start)
	clauses = append(clauses, fmt.Sprintf("ul.created_at >= $%d", idx))
	idx++
	args = append(args, end)
	clauses = append(clauses, fmt.Sprintf("ul.created_at < $%d", idx))
	idx++

	if groupID != nil && *groupID > 0 {
		args = append(args, *groupID)
		clauses = append(clauses, fmt.Sprintf("ul.group_id = $%d", idx))
		idx++
	}
	if channelID != nil && *channelID > 0 {
		args = append(args, *channelID)
		clauses = append(clauses, fmt.Sprintf("ul.channel_id = $%d", idx))
		idx++
	}
	if platform != "" {
		// Prefer group.platform when available; fall back to account.platform so we don't
		// drop rows where group_id is NULL.
		join = "LEFT JOIN groups g ON g.id = ul.group_id LEFT JOIN accounts a ON a.id = ul.account_id"
		args = append(args, platform)
		clauses = append(clauses, fmt.Sprintf("COALESCE(NULLIF(g.platform,''), a.platform) = $%d", idx))
		idx++
	}

	where = "WHERE " + strings.Join(clauses, " AND ")
	return join, where, args, idx
}

func buildErrorWhere(filter *service.OpsDashboardFilter, start, end time.Time, startIndex int) (where string, args []any, nextIndex int) {
	platform := ""
	groupID := (*int64)(nil)
	channelID := (*int64)(nil)
	if filter != nil {
		platform = strings.TrimSpace(strings.ToLower(filter.Platform))
		groupID = filter.GroupID
		channelID = filter.ChannelID
	}

	idx := startIndex
	clauses := make([]string, 0, 5)
	args = make([]any, 0, 5)

	args = append(args, start)
	clauses = append(clauses, fmt.Sprintf("created_at >= $%d", idx))
	idx++
	args = append(args, end)
	clauses = append(clauses, fmt.Sprintf("created_at < $%d", idx))
	idx++

	clauses = append(clauses, "is_count_tokens = FALSE")

	if groupID != nil && *groupID > 0 {
		args = append(args, *groupID)
		clauses = append(clauses, fmt.Sprintf("group_id = $%d", idx))
		idx++
	}
	if channelID != nil && *channelID > 0 {
		args = append(args, *channelID)
		clauses = append(clauses, fmt.Sprintf("channel_id = $%d", idx))
		idx++
	}
	if platform != "" {
		args = append(args, platform)
		clauses = append(clauses, fmt.Sprintf("platform = $%d", idx))
		idx++
	}

	where = "WHERE " + strings.Join(clauses, " AND ")
	return where, args, idx
}
