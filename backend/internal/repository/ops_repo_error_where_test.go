package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildOpsErrorLogsWhere_QueryUsesQualifiedColumns(t *testing.T) {
	filter := &service.OpsErrorLogFilter{
		Query: "ACCESS_DENIED",
	}

	where, args := buildOpsErrorLogsWhere(filter)
	if where == "" {
		t.Fatalf("where should not be empty")
	}
	if len(args) != 1 {
		t.Fatalf("args len = %d, want 1", len(args))
	}
	if !strings.Contains(where, "e.request_id ILIKE $") {
		t.Fatalf("where should include qualified request_id condition: %s", where)
	}
	if !strings.Contains(where, "e.client_request_id ILIKE $") {
		t.Fatalf("where should include qualified client_request_id condition: %s", where)
	}
	if !strings.Contains(where, "e.error_message ILIKE $") {
		t.Fatalf("where should include qualified error_message condition: %s", where)
	}
}

func TestBuildOpsErrorLogsWhere_UserQueryUsesExistsSubquery(t *testing.T) {
	filter := &service.OpsErrorLogFilter{
		UserQuery: "admin@",
	}

	where, args := buildOpsErrorLogsWhere(filter)
	if where == "" {
		t.Fatalf("where should not be empty")
	}
	if len(args) != 1 {
		t.Fatalf("args len = %d, want 1", len(args))
	}
	if !strings.Contains(where, "EXISTS (SELECT 1 FROM users u WHERE u.id = e.user_id AND u.email ILIKE $") {
		t.Fatalf("where should include EXISTS user email condition: %s", where)
	}
}

func TestBuildOpsErrorLogsWhere_UsesGeminiMetadataExactMatchFilters(t *testing.T) {
	filter := &service.OpsErrorLogFilter{
		GeminiSurface: "openai_compat",
		BillingRuleID: "rule_text_input",
		ProbeAction:   "recover",
	}

	where, args := buildOpsErrorLogsWhere(filter)
	if len(args) != 3 {
		t.Fatalf("args len = %d, want 3", len(args))
	}
	if !strings.Contains(where, "COALESCE(e.gemini_surface,'') = $") {
		t.Fatalf("where should include gemini_surface exact match: %s", where)
	}
	if !strings.Contains(where, "COALESCE(e.billing_rule_id,'') = $") {
		t.Fatalf("where should include billing_rule_id exact match: %s", where)
	}
	if !strings.Contains(where, "COALESCE(e.probe_action,'') = $") {
		t.Fatalf("where should include probe_action exact match: %s", where)
	}
}

func TestBuildOpsErrorLogsWhere_UpstreamPhaseRequiresRecoveredOptIn(t *testing.T) {
	where, _ := buildOpsErrorLogsWhere(&service.OpsErrorLogFilter{Phase: "upstream"})
	if !strings.Contains(where, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("upstream phase without opt-in should keep status guard: %s", where)
	}
	if !strings.Contains(where, "e.error_phase = $") {
		t.Fatalf("upstream phase should still filter by phase: %s", where)
	}

	recoveredWhere, _ := buildOpsErrorLogsWhere(&service.OpsErrorLogFilter{
		Phase:                    "upstream",
		IncludeRecoveredUpstream: true,
	})
	if strings.Contains(recoveredWhere, "COALESCE(e.status_code, 0) >= 400") {
		t.Fatalf("recovered upstream opt-in should skip status guard: %s", recoveredWhere)
	}
}

func TestBuildOpsErrorLogsWhere_UsesCategoryAnyFilters(t *testing.T) {
	where, args := buildOpsErrorLogsWhere(&service.OpsErrorLogFilter{
		ErrorPhasesAny: []string{"upstream", "network"},
		ErrorTypesAny:  []string{"rate_limit_error"},
	})
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2", len(args))
	}
	if !strings.Contains(where, "e.error_phase = ANY($") {
		t.Fatalf("where should include phase ANY filter: %s", where)
	}
	if !strings.Contains(where, "e.error_type = ANY($") {
		t.Fatalf("where should include type ANY filter: %s", where)
	}
}

func TestOpsErrorLogsOrderByWhitelistsSortColumns(t *testing.T) {
	if got := opsErrorLogsOrderBy(&service.OpsErrorLogFilter{SortBy: "status_code", SortOrder: "asc"}); got != "COALESCE(e.upstream_status_code, e.status_code, 0) asc, e.id asc" {
		t.Fatalf("status order = %q", got)
	}
	if got := opsErrorLogsOrderBy(&service.OpsErrorLogFilter{SortBy: "model", SortOrder: "desc"}); !strings.Contains(got, "COALESCE(NULLIF(TRIM(e.requested_model), ''), e.model, '') desc") {
		t.Fatalf("model order = %q", got)
	}
	if got := opsErrorLogsOrderBy(&service.OpsErrorLogFilter{SortBy: "unsafe", SortOrder: "asc"}); got != "e.created_at asc, e.id asc" {
		t.Fatalf("unsafe sort should fall back to created_at: %q", got)
	}
}
