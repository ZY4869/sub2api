package repository

import (
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var opsSQLPlaceholderPattern = regexp.MustCompile(`\$(\d+)`)

func buildOpsRequestTracesWhere(filter *service.OpsRequestTraceFilter) (string, []any) {
	return buildOpsRequestTracesWhereWithSchema(filter, defaultOpsRequestTraceSchema())
}

func hasRequestTraceDeleteCondition(filter *service.OpsRequestTraceFilter) bool {
	if filter == nil {
		return false
	}
	if filter.StartTime != nil || filter.EndTime != nil {
		return true
	}
	for _, value := range []string{
		filter.Status,
		filter.Platform,
		filter.ProtocolIn,
		filter.ProtocolOut,
		filter.Channel,
		filter.RoutePath,
		filter.RequestType,
		filter.FinishReason,
		filter.CaptureReason,
		filter.RequestedModel,
		filter.UpstreamModel,
		filter.RequestID,
		filter.ClientRequestID,
		filter.UpstreamRequestID,
		filter.GeminiSurface,
		filter.BillingRuleID,
		filter.ProbeAction,
		filter.Query,
	} {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	if filter.UserID != nil || filter.APIKeyID != nil || filter.AccountID != nil || filter.GroupID != nil || filter.StatusCode != nil {
		return true
	}
	return filter.Stream != nil || filter.HasTools != nil || filter.HasThinking != nil || filter.RawAvailable != nil || filter.Sampled != nil
}

func buildOpsRequestTracesWhereWithSchema(filter *service.OpsRequestTraceFilter, schema opsRequestTraceSchema) (string, []any) {
	clauses := make([]string, 0, 20)
	args := make([]any, 0, 20)
	clauses = append(clauses, "1=1")

	if filter == nil {
		return "WHERE " + strings.Join(clauses, " AND "), args
	}

	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		args = append(args, filter.StartTime.UTC())
		clauses = append(clauses, "t.created_at >= $"+itoa(len(args)))
	}
	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		args = append(args, filter.EndTime.UTC())
		clauses = append(clauses, "t.created_at < $"+itoa(len(args)))
	}
	if v := strings.TrimSpace(strings.ToLower(filter.Status)); v != "" {
		switch v {
		case "success":
			clauses = append(clauses, "COALESCE(t.status_code, 0) < 400")
		case "error":
			clauses = append(clauses, "COALESCE(t.status_code, 0) >= 400")
		default:
			args = append(args, v)
			clauses = append(clauses, "LOWER(COALESCE(t.status,'')) = $"+itoa(len(args)))
		}
	}
	addOpsRequestTraceStringFilter := func(column string, value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			args = append(args, trimmed)
			clauses = append(clauses, column+" = $"+itoa(len(args)))
		}
	}

	addOpsRequestTraceStringFilter("COALESCE(t.platform,'')", filter.Platform)
	addOpsRequestTraceStringFilter("COALESCE(t.protocol_in,'')", filter.ProtocolIn)
	addOpsRequestTraceStringFilter("COALESCE(t.protocol_out,'')", filter.ProtocolOut)
	addOpsRequestTraceStringFilter("COALESCE(t.channel,'')", filter.Channel)
	addOpsRequestTraceStringFilter("COALESCE(t.route_path,'')", filter.RoutePath)
	addOpsRequestTraceStringFilter("COALESCE(t.request_type,'')", filter.RequestType)
	addOpsRequestTraceStringFilter("COALESCE(t.finish_reason,'')", filter.FinishReason)
	addOpsRequestTraceStringFilter("COALESCE(t.capture_reason,'')", filter.CaptureReason)
	addOpsRequestTraceStringFilter("COALESCE(t.requested_model,'')", filter.RequestedModel)
	addOpsRequestTraceStringFilter("COALESCE(t.upstream_model,'')", filter.UpstreamModel)
	addOpsRequestTraceStringFilter("COALESCE(t.request_id,'')", filter.RequestID)
	addOpsRequestTraceStringFilter("COALESCE(t.client_request_id,'')", filter.ClientRequestID)
	addOpsRequestTraceStringFilter("COALESCE(t.upstream_request_id,'')", filter.UpstreamRequestID)
	addOpsRequestTraceStringFilter(opsRequestTraceOptionalStringExpr("t.gemini_surface", schema.HasGeminiSurface), filter.GeminiSurface)
	addOpsRequestTraceStringFilter(opsRequestTraceOptionalStringExpr("t.billing_rule_id", schema.HasBillingRuleID), filter.BillingRuleID)
	addOpsRequestTraceStringFilter(opsRequestTraceOptionalStringExpr("t.probe_action", schema.HasProbeAction), filter.ProbeAction)

	if filter.UserID != nil && *filter.UserID > 0 {
		args = append(args, *filter.UserID)
		clauses = append(clauses, "t.user_id = $"+itoa(len(args)))
	}
	if filter.APIKeyID != nil && *filter.APIKeyID > 0 {
		args = append(args, *filter.APIKeyID)
		clauses = append(clauses, "t.api_key_id = $"+itoa(len(args)))
	}
	if filter.AccountID != nil && *filter.AccountID > 0 {
		args = append(args, *filter.AccountID)
		clauses = append(clauses, "t.account_id = $"+itoa(len(args)))
	}
	if filter.GroupID != nil && *filter.GroupID > 0 {
		args = append(args, *filter.GroupID)
		clauses = append(clauses, "t.group_id = $"+itoa(len(args)))
	}
	if filter.StatusCode != nil && *filter.StatusCode > 0 {
		args = append(args, *filter.StatusCode)
		clauses = append(clauses, "t.status_code = $"+itoa(len(args)))
	}
	addBoolClause := func(column string, value *bool) {
		if value == nil {
			return
		}
		args = append(args, *value)
		clauses = append(clauses, column+" = $"+itoa(len(args)))
	}
	addBoolClause("COALESCE(t.stream,false)", filter.Stream)
	addBoolClause("COALESCE(t.has_tools,false)", filter.HasTools)
	addBoolClause("COALESCE(t.has_thinking,false)", filter.HasThinking)
	addBoolClause("COALESCE(t.raw_available,false)", filter.RawAvailable)
	addBoolClause("COALESCE(t.sampled,false)", filter.Sampled)

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		args = append(args, like)
		clauses = append(clauses, "(COALESCE(t.search_text,'') ILIKE $"+itoa(len(args))+" OR COALESCE(t.request_id,'') ILIKE $"+itoa(len(args))+" OR COALESCE(t.client_request_id,'') ILIKE $"+itoa(len(args))+" OR COALESCE(t.upstream_request_id,'') ILIKE $"+itoa(len(args))+")")
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func normalizeJSONText(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "", "null":
		return ""
	default:
		return value
	}
}

func opsRequestTraceBucketSeconds(window time.Duration) int64 {
	switch {
	case window > 7*24*time.Hour:
		return 24 * 60 * 60
	case window > 24*time.Hour:
		return 60 * 60
	case window > 6*time.Hour:
		return 15 * 60
	default:
		return 5 * 60
	}
}

func nullBytes(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

func shiftSQLPlaceholders(query string, offset int) string {
	if offset == 0 || query == "" {
		return query
	}
	return opsSQLPlaceholderPattern.ReplaceAllStringFunc(query, func(placeholder string) string {
		value := strings.TrimPrefix(placeholder, "$")
		number := 0
		for _, ch := range value {
			number = number*10 + int(ch-'0')
		}
		return "$" + itoa(number+offset)
	})
}
