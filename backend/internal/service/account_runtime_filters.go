package service

import (
	"context"
	"slices"
	"strings"
)

const (
	AccountRuntimeViewAll       = "all"
	AccountRuntimeViewInUseOnly = "in_use_only"
)

type accountRuntimeFiltersContextKey struct{}

type AccountRuntimeFilters struct {
	RuntimeView         string
	CandidateAccountIDs []int64
}

func NormalizeAccountRuntimeViewInput(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", AccountRuntimeViewAll:
		return AccountRuntimeViewAll
	case AccountRuntimeViewInUseOnly:
		return AccountRuntimeViewInUseOnly
	default:
		return AccountRuntimeViewAll
	}
}

func WithAccountRuntimeFilters(ctx context.Context, runtimeView string, candidateAccountIDs []int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ids := append([]int64(nil), candidateAccountIDs...)
	slices.Sort(ids)
	return context.WithValue(ctx, accountRuntimeFiltersContextKey{}, AccountRuntimeFilters{
		RuntimeView:         NormalizeAccountRuntimeViewInput(runtimeView),
		CandidateAccountIDs: ids,
	})
}

func AccountRuntimeFiltersFromContext(ctx context.Context) AccountRuntimeFilters {
	if ctx == nil {
		return AccountRuntimeFilters{RuntimeView: AccountRuntimeViewAll}
	}
	filters, ok := ctx.Value(accountRuntimeFiltersContextKey{}).(AccountRuntimeFilters)
	if !ok {
		return AccountRuntimeFilters{RuntimeView: AccountRuntimeViewAll}
	}
	filters.RuntimeView = NormalizeAccountRuntimeViewInput(filters.RuntimeView)
	if len(filters.CandidateAccountIDs) == 0 {
		filters.CandidateAccountIDs = nil
		return filters
	}
	ids := append([]int64(nil), filters.CandidateAccountIDs...)
	slices.Sort(ids)
	filters.CandidateAccountIDs = ids
	return filters
}
