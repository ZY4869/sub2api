package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usageSubjectScope struct {
	subjectType service.UsageSubjectType
	column      string
}

type usageSubjectAggregate struct {
	history []usagestats.AccountUsageHistory
	summary service.UsageSubjectSummary
}

func (r *usageLogRepository) GetSubjectUsageInsights(ctx context.Context, query service.UsageSubjectInsightsQuery) (*service.UsageSubjectInsights, error) {
	scope, err := resolveUsageSubjectScope(query.SubjectType)
	if err != nil {
		return nil, err
	}

	subject, err := r.getUsageSubjectReference(ctx, scope, query.SubjectID)
	if err != nil {
		return nil, err
	}
	aggregate, err := r.getUsageSubjectAggregate(ctx, scope, query.SubjectID, query.StartTime, query.EndTime)
	if err != nil {
		return nil, err
	}
	coverage, err := r.getUsageSubjectPreviewCoverage(ctx, scope, query.SubjectID, query.StartTime, query.EndTime)
	if err != nil {
		return nil, err
	}

	var (
		accountID int64
		apiKeyID  int64
		groupID   int64
	)
	switch scope.subjectType {
	case service.UsageSubjectTypeAccount:
		accountID = query.SubjectID
	case service.UsageSubjectTypeAPIKey:
		apiKeyID = query.SubjectID
	case service.UsageSubjectTypeGroup:
		groupID = query.SubjectID
	}

	models, err := r.GetModelStatsWithFilters(ctx, query.StartTime, query.EndTime, 0, apiKeyID, accountID, groupID, 0, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	endpoints, err := r.GetEndpointStatsWithFilters(ctx, query.StartTime, query.EndTime, 0, apiKeyID, accountID, groupID, "", nil, nil, nil)
	if err != nil {
		return nil, err
	}
	upstreamEndpoints, err := r.GetUpstreamEndpointStatsWithFilters(ctx, query.StartTime, query.EndTime, 0, apiKeyID, accountID, groupID, "", nil, nil, nil)
	if err != nil {
		return nil, err
	}

	return &service.UsageSubjectInsights{
		Subject:                *subject,
		Summary:                aggregate.summary,
		History:                aggregate.history,
		Models:                 models,
		Endpoints:              endpoints,
		UpstreamEndpoints:      upstreamEndpoints,
		RequestPreviewCoverage: *coverage,
	}, nil
}

func (r *usageLogRepository) getUsageSubjectReference(ctx context.Context, scope usageSubjectScope, subjectID int64) (*service.UsageSubjectReference, error) {
	switch scope.subjectType {
	case service.UsageSubjectTypeAccount:
		var subject service.UsageSubjectReference
		err := scanSingleRow(
			ctx,
			r.sql,
			"SELECT id, COALESCE(name, '') FROM accounts WHERE id = $1",
			[]any{subjectID},
			&subject.ID,
			&subject.Name,
		)
		if err != nil {
			return nil, infraerrors.NotFound("USAGE_SUBJECT_NOT_FOUND", "subject not found").WithCause(err)
		}
		subject.Type = scope.subjectType
		return &subject, nil
	case service.UsageSubjectTypeGroup:
		var subject service.UsageSubjectReference
		err := scanSingleRow(
			ctx,
			r.sql,
			"SELECT id, COALESCE(name, '') FROM groups WHERE id = $1",
			[]any{subjectID},
			&subject.ID,
			&subject.Name,
		)
		if err != nil {
			return nil, infraerrors.NotFound("USAGE_SUBJECT_NOT_FOUND", "subject not found").WithCause(err)
		}
		subject.Type = scope.subjectType
		groupID := subject.ID
		subject.GroupID = &groupID
		subject.GroupName = subject.Name
		return &subject, nil
	case service.UsageSubjectTypeAPIKey:
		var (
			subject   service.UsageSubjectReference
			userID    sql.NullInt64
			groupID   sql.NullInt64
			userEmail sql.NullString
			groupName sql.NullString
		)
		err := scanSingleRow(
			ctx,
			r.sql,
			`SELECT ak.id, COALESCE(ak.name, ''), ak.user_id, COALESCE(u.email, ''), ak.group_id, COALESCE(g.name, '')
			 FROM api_keys ak
			 LEFT JOIN users u ON u.id = ak.user_id
			 LEFT JOIN groups g ON g.id = ak.group_id
			 WHERE ak.id = $1`,
			[]any{subjectID},
			&subject.ID,
			&subject.Name,
			&userID,
			&userEmail,
			&groupID,
			&groupName,
		)
		if err != nil {
			return nil, infraerrors.NotFound("USAGE_SUBJECT_NOT_FOUND", "subject not found").WithCause(err)
		}
		subject.Type = scope.subjectType
		if userID.Valid {
			value := userID.Int64
			subject.UserID = &value
		}
		if userEmail.Valid {
			subject.UserEmail = userEmail.String
		}
		if groupID.Valid {
			value := groupID.Int64
			subject.GroupID = &value
		}
		if groupName.Valid {
			subject.GroupName = groupName.String
		}
		return &subject, nil
	default:
		return nil, infraerrors.BadRequest("USAGE_SUBJECT_INVALID_TYPE", "invalid subject type")
	}
}

func (r *usageLogRepository) getUsageSubjectAggregate(ctx context.Context, scope usageSubjectScope, subjectID int64, startTime, endTime time.Time) (resp *usageSubjectAggregate, err error) {
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, 'YYYY-MM-DD') AS date,
			COUNT(*) AS requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) AS standard_cost,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) AS account_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) AS user_cost
		FROM usage_logs
		WHERE %s = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY date
		ORDER BY date ASC
	`, scope.column)

	rows, err := r.sql.QueryContext(ctx, query, subjectID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			resp = nil
		}
	}()

	history := make([]usagestats.AccountUsageHistory, 0)
	for rows.Next() {
		var (
			date        string
			requests    int64
			tokens      int64
			standard    float64
			accountCost float64
			userCost    float64
		)
		if err = rows.Scan(&date, &requests, &tokens, &standard, &accountCost, &userCost); err != nil {
			return nil, err
		}
		parsed, _ := time.Parse("2006-01-02", date)
		history = append(history, usagestats.AccountUsageHistory{
			Date:       date,
			Label:      parsed.Format("01/02"),
			Requests:   requests,
			Tokens:     tokens,
			Cost:       standard,
			ActualCost: accountCost,
			UserCost:   userCost,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	var (
		totalAccountCost  float64
		totalUserCost     float64
		totalStandardCost float64
		totalRequests     int64
		totalTokens       int64
		highestCostDay    *usagestats.AccountUsageHistory
		highestRequestDay *usagestats.AccountUsageHistory
	)
	for index := range history {
		item := &history[index]
		totalAccountCost += item.ActualCost
		totalUserCost += item.UserCost
		totalStandardCost += item.Cost
		totalRequests += item.Requests
		totalTokens += item.Tokens
		if highestCostDay == nil || item.ActualCost > highestCostDay.ActualCost {
			highestCostDay = item
		}
		if highestRequestDay == nil || item.Requests > highestRequestDay.Requests {
			highestRequestDay = item
		}
	}

	var avgDurationMs float64
	avgDurationQuery := fmt.Sprintf(
		"SELECT COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) FROM usage_logs WHERE %s = $1 AND created_at >= $2 AND created_at < $3",
		scope.column,
	)
	if err := scanSingleRow(ctx, r.sql, avgDurationQuery, []any{subjectID, startTime, endTime}, &avgDurationMs); err != nil {
		return nil, err
	}

	activeDays := len(history)
	windowDays := int(endTime.Sub(startTime).Hours()/24) + 1
	if windowDays < 1 {
		windowDays = 1
	}

	summary := service.UsageSubjectSummary{
		TotalAccountCost:  totalAccountCost,
		TotalUserCost:     totalUserCost,
		TotalStandardCost: totalStandardCost,
		TotalRequests:     totalRequests,
		TotalTokens:       totalTokens,
		AvgDurationMs:     avgDurationMs,
		ActiveDays:        activeDays,
		WindowDays:        windowDays,
	}
	if activeDays > 0 {
		days := float64(activeDays)
		summary.AvgDailyAccountCost = totalAccountCost / days
		summary.AvgDailyUserCost = totalUserCost / days
		summary.AvgDailyStandardCost = totalStandardCost / days
		summary.AvgDailyRequests = float64(totalRequests) / days
		summary.AvgDailyTokens = float64(totalTokens) / days
	}

	today := timezone.Now().Format("2006-01-02")
	for index := range history {
		if history[index].Date != today {
			continue
		}
		item := history[index]
		summary.Today = &service.UsageSubjectSummaryDay{
			Date:         item.Date,
			AccountCost:  item.ActualCost,
			UserCost:     item.UserCost,
			StandardCost: item.Cost,
			Requests:     item.Requests,
			Tokens:       item.Tokens,
		}
		break
	}
	if highestCostDay != nil {
		summary.HighestCostDay = &service.UsageSubjectSummaryCostDay{
			Date:         highestCostDay.Date,
			Label:        highestCostDay.Label,
			AccountCost:  highestCostDay.ActualCost,
			UserCost:     highestCostDay.UserCost,
			StandardCost: highestCostDay.Cost,
			Requests:     highestCostDay.Requests,
		}
	}
	if highestRequestDay != nil {
		summary.HighestRequestDay = &service.UsageSubjectSummaryRequestDay{
			Date:         highestRequestDay.Date,
			Label:        highestRequestDay.Label,
			Requests:     highestRequestDay.Requests,
			AccountCost:  highestRequestDay.ActualCost,
			UserCost:     highestRequestDay.UserCost,
			StandardCost: highestRequestDay.Cost,
		}
	}

	return &usageSubjectAggregate{
		history: history,
		summary: summary,
	}, nil
}

func (r *usageLogRepository) getUsageSubjectPreviewCoverage(ctx context.Context, scope usageSubjectScope, subjectID int64, startTime, endTime time.Time) (*service.UsageRequestPreviewCoverage, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*)::bigint,
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM ops_request_traces t
					WHERE t.user_id = ul.user_id
					  AND t.api_key_id = ul.api_key_id
					  AND COALESCE(t.request_id, '') = COALESCE(ul.request_id, '')
				)
			)::bigint,
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM ops_request_traces t
					WHERE t.user_id = ul.user_id
					  AND t.api_key_id = ul.api_key_id
					  AND COALESCE(t.request_id, '') = COALESCE(ul.request_id, '')
					  AND COALESCE(t.normalized_request::text, '') <> ''
				)
			)::bigint,
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM ops_request_traces t
					WHERE t.user_id = ul.user_id
					  AND t.api_key_id = ul.api_key_id
					  AND COALESCE(t.request_id, '') = COALESCE(ul.request_id, '')
					  AND COALESCE(t.upstream_request::text, '') <> ''
				)
			)::bigint,
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM ops_request_traces t
					WHERE t.user_id = ul.user_id
					  AND t.api_key_id = ul.api_key_id
					  AND COALESCE(t.request_id, '') = COALESCE(ul.request_id, '')
					  AND COALESCE(t.upstream_response::text, '') <> ''
				)
			)::bigint,
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM ops_request_traces t
					WHERE t.user_id = ul.user_id
					  AND t.api_key_id = ul.api_key_id
					  AND COALESCE(t.request_id, '') = COALESCE(ul.request_id, '')
					  AND COALESCE(t.gateway_response::text, '') <> ''
				)
			)::bigint,
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM ops_request_traces t
					WHERE t.user_id = ul.user_id
					  AND t.api_key_id = ul.api_key_id
					  AND COALESCE(t.request_id, '') = COALESCE(ul.request_id, '')
					  AND COALESCE(t.tool_trace::text, '') <> ''
				)
			)::bigint
		FROM usage_logs ul
		WHERE ul.%s = $1
		  AND ul.created_at >= $2
		  AND ul.created_at < $3
	`, scope.column)

	coverage := &service.UsageRequestPreviewCoverage{}
	if err := scanSingleRow(
		ctx,
		r.sql,
		query,
		[]any{subjectID, startTime, endTime},
		&coverage.TotalRequests,
		&coverage.PreviewAvailableCount,
		&coverage.NormalizedCount,
		&coverage.UpstreamRequestCount,
		&coverage.UpstreamResponseCount,
		&coverage.GatewayResponseCount,
		&coverage.ToolTraceCount,
	); err != nil {
		return nil, err
	}
	if coverage.TotalRequests > 0 {
		coverage.PreviewAvailableRate = float64(coverage.PreviewAvailableCount) / float64(coverage.TotalRequests)
	}
	return coverage, nil
}

func resolveUsageSubjectScope(subjectType service.UsageSubjectType) (usageSubjectScope, error) {
	switch subjectType {
	case service.UsageSubjectTypeAccount:
		return usageSubjectScope{subjectType: subjectType, column: "account_id"}, nil
	case service.UsageSubjectTypeGroup:
		return usageSubjectScope{subjectType: subjectType, column: "group_id"}, nil
	case service.UsageSubjectTypeAPIKey:
		return usageSubjectScope{subjectType: subjectType, column: "api_key_id"}, nil
	default:
		return usageSubjectScope{}, infraerrors.BadRequest("USAGE_SUBJECT_INVALID_TYPE", "subject_type must be account, group, or api_key")
	}
}
