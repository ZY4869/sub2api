package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAppendAdminLimitedWhereClauses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		filters            adminAccountListFilters
		argIndex           int
		wantClauses        int
		wantArg            any
		wantClauseContains [][]string
	}{
		{
			name: "normal only",
			filters: adminAccountListFilters{
				LimitedView: service.AccountLimitedViewNormalOnly,
			},
			argIndex:    1,
			wantClauses: 1,
			wantClauseContains: [][]string{
				{"NOT (", "rate_limit_reset_at > NOW()", "codex_7d_used_percent", "credentials->>'plan_type'"},
			},
		},
		{
			name: "limited reason filter",
			filters: adminAccountListFilters{
				LimitedView:   service.AccountLimitedViewLimitedOnly,
				LimitedReason: service.AccountRateLimitReasonUsage7d,
			},
			argIndex:    3,
			wantClauses: 2,
			wantArg:     service.AccountRateLimitReasonUsage7d,
			wantClauseContains: [][]string{
				{"rate_limit_reset_at > NOW()", "codex_7d_used_percent", "codex_spark_7d_used_percent"},
				{"CASE", "extra->>'rate_limit_reason'", service.AccountRateLimitReasonUsage7d},
			},
		},
		{
			name: "all 7d limited reason filter",
			filters: adminAccountListFilters{
				LimitedView:   service.AccountLimitedViewLimitedOnly,
				LimitedReason: service.AccountRateLimitReasonUsage7dAll,
			},
			argIndex:    5,
			wantClauses: 2,
			wantArg:     service.AccountRateLimitReasonUsage7dAll,
			wantClauseContains: [][]string{
				{"rate_limit_reset_at > NOW()", "codex_7d_used_percent", "codex_spark_7d_used_percent"},
				{"CASE", service.AccountRateLimitReasonUsage7dAll, "credentials->>'plan_type'", "codex_spark_7d_used_percent"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clauses, args, nextArgIndex := appendAdminLimitedWhereClauses(nil, nil, tt.argIndex, tt.filters, "a")
			if len(clauses) != tt.wantClauses {
				t.Fatalf("len(clauses) = %d, want %d", len(clauses), tt.wantClauses)
			}
			for clauseIndex, fragments := range tt.wantClauseContains {
				if clauseIndex >= len(clauses) {
					t.Fatalf("missing clause %d in %#v", clauseIndex, clauses)
				}
				for _, fragment := range fragments {
					if !strings.Contains(clauses[clauseIndex], fragment) {
						t.Fatalf("clause %d (%s) does not contain fragment %q", clauseIndex, clauses[clauseIndex], fragment)
					}
				}
			}
			if tt.wantArg != nil {
				if len(args) != 1 || args[0] != tt.wantArg {
					t.Fatalf("args = %#v, want [%#v]", args, tt.wantArg)
				}
				if nextArgIndex != tt.argIndex+1 {
					t.Fatalf("nextArgIndex = %d, want %d", nextArgIndex, tt.argIndex+1)
				}
			}
		})
	}
}
