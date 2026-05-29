package repository

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func buildOpsErrorLogsWhere(filter *service.OpsErrorLogFilter) (string, []any) {
	clauses := make([]string, 0, 12)
	args := make([]any, 0, 12)
	clauses = append(clauses, "1=1")

	phaseFilter := ""
	if filter != nil {
		phaseFilter = strings.TrimSpace(strings.ToLower(filter.Phase))
	}
	// ops_error_logs stores client-visible error requests (status>=400),
	// but we also persist "recovered" upstream errors (status<400) for upstream health visibility.
	// If Resolved is not specified, do not filter by resolved state (backward-compatible).
	resolvedFilter := (*bool)(nil)
	if filter != nil {
		resolvedFilter = filter.Resolved
	}
	// Keep list endpoints scoped to client errors unless explicitly filtering upstream phase.
	if phaseFilter != "upstream" {
		clauses = append(clauses, "COALESCE(e.status_code, 0) >= 400")
	}

	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		args = append(args, filter.StartTime.UTC())
		clauses = append(clauses, "e.created_at >= $"+itoa(len(args)))
	}
	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		args = append(args, filter.EndTime.UTC())
		// Keep time-window semantics consistent with other ops queries: [start, end)
		clauses = append(clauses, "e.created_at < $"+itoa(len(args)))
	}
	if p := strings.TrimSpace(filter.Platform); p != "" {
		args = append(args, p)
		clauses = append(clauses, "e.platform = $"+itoa(len(args)))
	}
	if filter.GroupID != nil && *filter.GroupID > 0 {
		args = append(args, *filter.GroupID)
		clauses = append(clauses, "e.group_id = $"+itoa(len(args)))
	}
	if filter.AccountID != nil && *filter.AccountID > 0 {
		args = append(args, *filter.AccountID)
		clauses = append(clauses, "e.account_id = $"+itoa(len(args)))
	}
	if phase := phaseFilter; phase != "" {
		args = append(args, phase)
		clauses = append(clauses, "e.error_phase = $"+itoa(len(args)))
	}
	if filter != nil {
		if owner := strings.TrimSpace(strings.ToLower(filter.Owner)); owner != "" {
			args = append(args, owner)
			clauses = append(clauses, "LOWER(COALESCE(e.error_owner,'')) = $"+itoa(len(args)))
		}
		if source := strings.TrimSpace(strings.ToLower(filter.Source)); source != "" {
			args = append(args, source)
			clauses = append(clauses, "LOWER(COALESCE(e.error_source,'')) = $"+itoa(len(args)))
		}
	}
	if resolvedFilter != nil {
		args = append(args, *resolvedFilter)
		clauses = append(clauses, "COALESCE(e.resolved,false) = $"+itoa(len(args)))
	}

	// View filter: errors vs excluded vs all.
	// Excluded = business-limited errors (quota/concurrency/billing).
	// Upstream 429/529 are included in errors view to match SLA calculation.
	view := ""
	if filter != nil {
		view = strings.ToLower(strings.TrimSpace(filter.View))
	}
	switch view {
	case "", "errors":
		clauses = append(clauses, "COALESCE(e.is_business_limited,false) = false")
	case "excluded":
		clauses = append(clauses, "COALESCE(e.is_business_limited,false) = true")
	case "all":
		// no-op
	default:
		// treat unknown as default 'errors'
		clauses = append(clauses, "COALESCE(e.is_business_limited,false) = false")
	}
	if len(filter.StatusCodes) > 0 {
		args = append(args, pq.Array(filter.StatusCodes))
		clauses = append(clauses, "COALESCE(e.upstream_status_code, e.status_code, 0) = ANY($"+itoa(len(args))+")")
	} else if filter.StatusCodesOther {
		// "Other" means: status codes not in the common list.
		known := []int{400, 401, 403, 404, 409, 422, 429, 500, 502, 503, 504, 529}
		args = append(args, pq.Array(known))
		clauses = append(clauses, "NOT (COALESCE(e.upstream_status_code, e.status_code, 0) = ANY($"+itoa(len(args))+"))")
	}
	// Exact correlation keys (preferred for request↔upstream linkage).
	if rid := strings.TrimSpace(filter.RequestID); rid != "" {
		args = append(args, rid)
		clauses = append(clauses, "COALESCE(e.request_id,'') = $"+itoa(len(args)))
	}
	if crid := strings.TrimSpace(filter.ClientRequestID); crid != "" {
		args = append(args, crid)
		clauses = append(clauses, "COALESCE(e.client_request_id,'') = $"+itoa(len(args)))
	}
	if surface := strings.TrimSpace(filter.GeminiSurface); surface != "" {
		args = append(args, surface)
		clauses = append(clauses, "COALESCE(e.gemini_surface,'') = $"+itoa(len(args)))
	}
	if billingRuleID := strings.TrimSpace(filter.BillingRuleID); billingRuleID != "" {
		args = append(args, billingRuleID)
		clauses = append(clauses, "COALESCE(e.billing_rule_id,'') = $"+itoa(len(args)))
	}
	if probeAction := strings.TrimSpace(filter.ProbeAction); probeAction != "" {
		args = append(args, probeAction)
		clauses = append(clauses, "COALESCE(e.probe_action,'') = $"+itoa(len(args)))
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		args = append(args, like)
		n := itoa(len(args))
		clauses = append(clauses, "(e.request_id ILIKE $"+n+" OR e.client_request_id ILIKE $"+n+" OR e.error_message ILIKE $"+n+")")
	}

	if userQuery := strings.TrimSpace(filter.UserQuery); userQuery != "" {
		like := "%" + userQuery + "%"
		args = append(args, like)
		n := itoa(len(args))
		clauses = append(clauses, "EXISTS (SELECT 1 FROM users u WHERE u.id = e.user_id AND u.email ILIKE $"+n+")")
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}
