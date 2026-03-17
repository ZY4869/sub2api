package repository

import (
	"database/sql"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"strings"
	"time"
)

func scanUsageLog(scanner interface{ Scan(...any) error }) (*service.UsageLog, error) {
	var (
		id                    int64
		userID                int64
		apiKeyID              int64
		accountID             int64
		requestID             sql.NullString
		model                 string
		groupID               sql.NullInt64
		subscriptionID        sql.NullInt64
		inputTokens           int
		outputTokens          int
		cacheCreationTokens   int
		cacheReadTokens       int
		cacheCreation5m       int
		cacheCreation1h       int
		inputCost             float64
		outputCost            float64
		cacheCreationCost     float64
		cacheReadCost         float64
		totalCost             float64
		actualCost            float64
		billingExemptReason   sql.NullString
		rateMultiplier        float64
		accountRateMultiplier sql.NullFloat64
		billingType           int16
		requestTypeRaw        int16
		stream                bool
		openaiWSMode          bool
		durationMs            sql.NullInt64
		firstTokenMs          sql.NullInt64
		userAgent             sql.NullString
		ipAddress             sql.NullString
		imageCount            int
		imageSize             sql.NullString
		mediaType             sql.NullString
		serviceTier           sql.NullString
		reasoningEffort       sql.NullString
		thinkingEnabled       sql.NullBool
		inboundEndpoint       sql.NullString
		upstreamEndpoint      sql.NullString
		cacheTTLOverridden    bool
		createdAt             time.Time
	)
	if err := scanner.Scan(&id, &userID, &apiKeyID, &accountID, &requestID, &model, &groupID, &subscriptionID, &inputTokens, &outputTokens, &cacheCreationTokens, &cacheReadTokens, &cacheCreation5m, &cacheCreation1h, &inputCost, &outputCost, &cacheCreationCost, &cacheReadCost, &totalCost, &actualCost, &billingExemptReason, &rateMultiplier, &accountRateMultiplier, &billingType, &requestTypeRaw, &stream, &openaiWSMode, &durationMs, &firstTokenMs, &userAgent, &ipAddress, &imageCount, &imageSize, &mediaType, &serviceTier, &reasoningEffort, &thinkingEnabled, &inboundEndpoint, &upstreamEndpoint, &cacheTTLOverridden, &createdAt); err != nil {
		return nil, err
	}
	log := &service.UsageLog{ID: id, UserID: userID, APIKeyID: apiKeyID, AccountID: accountID, Model: model, InputTokens: inputTokens, OutputTokens: outputTokens, CacheCreationTokens: cacheCreationTokens, CacheReadTokens: cacheReadTokens, CacheCreation5mTokens: cacheCreation5m, CacheCreation1hTokens: cacheCreation1h, InputCost: inputCost, OutputCost: outputCost, CacheCreationCost: cacheCreationCost, CacheReadCost: cacheReadCost, TotalCost: totalCost, ActualCost: actualCost, RateMultiplier: rateMultiplier, AccountRateMultiplier: nullFloat64Ptr(accountRateMultiplier), BillingType: int8(billingType), RequestType: service.RequestTypeFromInt16(requestTypeRaw), ImageCount: imageCount, CacheTTLOverridden: cacheTTLOverridden, CreatedAt: createdAt}
	log.Stream = stream
	log.OpenAIWSMode = openaiWSMode
	log.RequestType = log.EffectiveRequestType()
	log.Stream, log.OpenAIWSMode = service.ApplyLegacyRequestFields(log.RequestType, stream, openaiWSMode)
	if requestID.Valid {
		log.RequestID = requestID.String
	}
	if groupID.Valid {
		value := groupID.Int64
		log.GroupID = &value
	}
	if subscriptionID.Valid {
		value := subscriptionID.Int64
		log.SubscriptionID = &value
	}
	if durationMs.Valid {
		value := int(durationMs.Int64)
		log.DurationMs = &value
	}
	if firstTokenMs.Valid {
		value := int(firstTokenMs.Int64)
		log.FirstTokenMs = &value
	}
	if userAgent.Valid {
		log.UserAgent = &userAgent.String
	}
	if ipAddress.Valid {
		log.IPAddress = &ipAddress.String
	}
	if imageSize.Valid {
		log.ImageSize = &imageSize.String
	}
	if mediaType.Valid {
		log.MediaType = &mediaType.String
	}
	if serviceTier.Valid {
		log.ServiceTier = &serviceTier.String
	}
	if reasoningEffort.Valid {
		log.ReasoningEffort = &reasoningEffort.String
	}
	if thinkingEnabled.Valid {
		log.ThinkingEnabled = nullBoolPtr(thinkingEnabled)
	}
	if inboundEndpoint.Valid {
		log.InboundEndpoint = &inboundEndpoint.String
	}
	if upstreamEndpoint.Valid {
		log.UpstreamEndpoint = &upstreamEndpoint.String
	}
	if billingExemptReason.Valid {
		log.BillingExemptReason = &billingExemptReason.String
	}
	return log, nil
}
func scanTrendRows(rows *sql.Rows) ([]TrendDataPoint, error) {
	results := make([]TrendDataPoint, 0)
	for rows.Next() {
		var row TrendDataPoint
		if err := rows.Scan(&row.Date, &row.Requests, &row.InputTokens, &row.OutputTokens, &row.CacheCreationTokens, &row.CacheReadTokens, &row.TotalTokens, &row.Cost, &row.ActualCost); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
func scanModelStatsRows(rows *sql.Rows) ([]ModelStat, error) {
	results := make([]ModelStat, 0)
	for rows.Next() {
		var row ModelStat
		if err := rows.Scan(&row.Model, &row.Requests, &row.InputTokens, &row.OutputTokens, &row.CacheCreationTokens, &row.CacheReadTokens, &row.TotalTokens, &row.Cost, &row.ActualCost); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
func buildWhere(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(conditions, " AND ")
}
func appendRequestTypeOrStreamWhereCondition(conditions []string, args []any, requestType *int16, stream *bool) ([]string, []any) {
	if requestType != nil {
		condition, conditionArgs := buildRequestTypeFilterCondition(len(args)+1, *requestType)
		conditions = append(conditions, condition)
		args = append(args, conditionArgs...)
		return conditions, args
	}
	if stream != nil {
		conditions = append(conditions, fmt.Sprintf("stream = $%d", len(args)+1))
		args = append(args, *stream)
	}
	return conditions, args
}
func appendRequestTypeOrStreamQueryFilter(query string, args []any, requestType *int16, stream *bool) (string, []any) {
	if requestType != nil {
		condition, conditionArgs := buildRequestTypeFilterCondition(len(args)+1, *requestType)
		query += " AND " + condition
		args = append(args, conditionArgs...)
		return query, args
	}
	if stream != nil {
		query += fmt.Sprintf(" AND stream = $%d", len(args)+1)
		args = append(args, *stream)
	}
	return query, args
}
func buildRequestTypeFilterCondition(startArgIndex int, requestType int16) (string, []any) {
	normalized := service.RequestTypeFromInt16(requestType)
	requestTypeArg := int16(normalized)
	switch normalized {
	case service.RequestTypeSync:
		return fmt.Sprintf("(request_type = $%d OR (request_type = %d AND stream = FALSE AND openai_ws_mode = FALSE))", startArgIndex, int16(service.RequestTypeUnknown)), []any{requestTypeArg}
	case service.RequestTypeStream:
		return fmt.Sprintf("(request_type = $%d OR (request_type = %d AND stream = TRUE AND openai_ws_mode = FALSE))", startArgIndex, int16(service.RequestTypeUnknown)), []any{requestTypeArg}
	case service.RequestTypeWSV2:
		return fmt.Sprintf("(request_type = $%d OR (request_type = %d AND openai_ws_mode = TRUE))", startArgIndex, int16(service.RequestTypeUnknown)), []any{requestTypeArg}
	default:
		return fmt.Sprintf("request_type = $%d", startArgIndex), []any{requestTypeArg}
	}
}
func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}
func nullInt(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}
func nullFloat64Ptr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	out := v.Float64
	return &out
}
func nullUsageLogBool(v *bool) sql.NullBool {
	if v == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *v, Valid: true}
}
func nullBoolPtr(v sql.NullBool) *bool {
	if !v.Valid {
		return nil
	}
	out := v.Bool
	return &out
}
func nullString(v *string) sql.NullString {
	if v == nil || *v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}
