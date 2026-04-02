package service

import (
	"context"
	"testing"
)

func TestAccountRuntimeFiltersFromContext(t *testing.T) {
	ctx := WithAccountRuntimeFilters(context.TODO(), "in_use_only", []int64{9, 3, 9})
	filters := AccountRuntimeFiltersFromContext(ctx)
	if filters.RuntimeView != AccountRuntimeViewInUseOnly {
		t.Fatalf("RuntimeView = %q, want %q", filters.RuntimeView, AccountRuntimeViewInUseOnly)
	}
	if len(filters.CandidateAccountIDs) != 3 {
		t.Fatalf("CandidateAccountIDs len = %d, want 3", len(filters.CandidateAccountIDs))
	}
	if filters.CandidateAccountIDs[0] != 3 || filters.CandidateAccountIDs[1] != 9 || filters.CandidateAccountIDs[2] != 9 {
		t.Fatalf("CandidateAccountIDs = %v, want sorted ids", filters.CandidateAccountIDs)
	}
}

func TestNormalizeAccountRuntimeViewInput(t *testing.T) {
	if got := NormalizeAccountRuntimeViewInput("in_use_only"); got != AccountRuntimeViewInUseOnly {
		t.Fatalf("NormalizeAccountRuntimeViewInput() = %q, want %q", got, AccountRuntimeViewInUseOnly)
	}
	if got := NormalizeAccountRuntimeViewInput("bad-value"); got != AccountRuntimeViewAll {
		t.Fatalf("NormalizeAccountRuntimeViewInput() = %q, want %q", got, AccountRuntimeViewAll)
	}
}
