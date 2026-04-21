package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAppendAdminLimitedWhereClauses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filters     adminAccountListFilters
		argIndex    int
		wantClauses int
		wantArg     any
		wantSQL     []string
	}{
		{
			name: "normal only",
			filters: adminAccountListFilters{
				LimitedView: service.AccountLimitedViewNormalOnly,
			},
			argIndex:    1,
			wantClauses: 1,
			wantSQL:     []string{"rate_limit_reset_at IS NULL", "rate_limit_reset_at <= NOW()"},
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
			wantSQL: []string{
				"rate_limit_reset_at > NOW()",
				"rate_limit_reason",
				service.AccountRateLimitReasonUsage7d,
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
			wantSQL: []string{
				"rate_limit_reset_at > NOW()",
				"rate_limit_reason",
				service.AccountRateLimitReasonUsage7dAll,
				"codex_account_7d_all_exhausted",
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
			for _, fragment := range tt.wantSQL {
				found := false
				for _, clause := range clauses {
					if strings.Contains(clause, fragment) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("clauses %v do not contain fragment %q", clauses, fragment)
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
