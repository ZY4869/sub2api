package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const geminiProtocolRankExcluded = 99

func withGeminiGroupContext(ctx context.Context, group *Group) context.Context {
	if !IsGroupContextValid(group) {
		return ctx
	}
	if existing, ok := ctx.Value(ctxkey.Group).(*Group); !ok || existing == nil || existing.ID != group.ID || !IsGroupContextValid(existing) {
		ctx = context.WithValue(ctx, ctxkey.Group, group)
	}
	return context.WithValue(ctx, ctxkey.GeminiMixedProtocolEnabled, group.GeminiMixedProtocolEnabled)
}

func WithGeminiPublicProtocol(ctx context.Context, protocol string) context.Context {
	normalized := normalizeGeminiPublicProtocol(protocol)
	if normalized == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxkey.GeminiPublicProtocol, normalized)
}

func WithGeminiPublicProtocolStrict(ctx context.Context, protocol string) context.Context {
	ctx = WithGeminiPublicProtocol(ctx, protocol)
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, ctxkey.GeminiPublicProtocolStrict, true)
}

func GeminiPublicProtocolFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(ctxkey.GeminiPublicProtocol).(string); ok {
		return normalizeGeminiPublicProtocol(value)
	}
	return ""
}

func geminiPublicProtocolStrictFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	value, _ := ctx.Value(ctxkey.GeminiPublicProtocolStrict).(bool)
	return value
}

func normalizeGeminiPublicProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case UpstreamProviderVertexAI:
		return UpstreamProviderVertexAI
	case UpstreamProviderAIStudio:
		return UpstreamProviderAIStudio
	default:
		return ""
	}
}

func geminiMixedProtocolEnabledFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if enabled, ok := ctx.Value(ctxkey.GeminiMixedProtocolEnabled).(bool); ok {
		return enabled
	}
	if group, ok := ctx.Value(ctxkey.Group).(*Group); ok && group != nil {
		return group.GeminiMixedProtocolEnabled
	}
	return false
}

func isGeminiAIStudioSourceAccount(account *Account) bool {
	if account == nil || EffectiveProtocol(account) != PlatformGemini {
		return false
	}
	return !account.IsGeminiVertexSource()
}

func geminiPublicProtocolRank(ctx context.Context, account *Account) int {
	if account == nil {
		return geminiProtocolRankExcluded
	}
	if EffectiveProtocol(account) != PlatformGemini {
		return 0
	}
	publicProtocol := GeminiPublicProtocolFromContext(ctx)
	if publicProtocol == "" {
		return 0
	}
	allowMixed := geminiMixedProtocolEnabledFromContext(ctx)
	strict := geminiPublicProtocolStrictFromContext(ctx)
	switch publicProtocol {
	case UpstreamProviderAIStudio:
		if isGeminiAIStudioSourceAccount(account) {
			return 0
		}
		if strict {
			return geminiProtocolRankExcluded
		}
		if allowMixed && account.IsGeminiVertexAI() {
			return 1
		}
		return geminiProtocolRankExcluded
	case UpstreamProviderVertexAI:
		if account.IsGeminiVertexAI() {
			return 0
		}
		if account.IsGeminiVertexExpress() {
			return 1
		}
		if strict {
			return geminiProtocolRankExcluded
		}
		if allowMixed && isGeminiAIStudioSourceAccount(account) {
			return 2
		}
		return geminiProtocolRankExcluded
	default:
		return 0
	}
}

func geminiPublicProtocolAllowsAccount(ctx context.Context, account *Account) bool {
	return geminiPublicProtocolRank(ctx, account) < geminiProtocolRankExcluded
}

func filterGeminiAccountsByPublicProtocol(ctx context.Context, accounts []Account, platform string) []Account {
	if platform != PlatformGemini || GeminiPublicProtocolFromContext(ctx) == "" || len(accounts) == 0 {
		return accounts
	}
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if geminiPublicProtocolAllowsAccount(ctx, &account) {
			filtered = append(filtered, account)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		leftRank := geminiPublicProtocolRank(ctx, &filtered[i])
		rightRank := geminiPublicProtocolRank(ctx, &filtered[j])
		return leftRank < rightRank
	})
	return filtered
}

func filterByMinGeminiPublicProtocolRank(ctx context.Context, accounts []accountWithLoad) []accountWithLoad {
	if len(accounts) == 0 {
		return accounts
	}
	minRank := geminiPublicProtocolRank(ctx, accounts[0].account)
	for _, candidate := range accounts[1:] {
		if rank := geminiPublicProtocolRank(ctx, candidate.account); rank < minRank {
			minRank = rank
		}
	}
	filtered := make([]accountWithLoad, 0, len(accounts))
	for _, candidate := range accounts {
		if geminiPublicProtocolRank(ctx, candidate.account) == minRank {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func stableSortAccountsByGeminiPublicProtocolRank(ctx context.Context, accounts []*Account) {
	if len(accounts) <= 1 || GeminiPublicProtocolFromContext(ctx) == "" {
		return
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		return geminiPublicProtocolRank(ctx, accounts[i]) < geminiPublicProtocolRank(ctx, accounts[j])
	})
}

func isPreferredAccountBySelectionOrderWithContext(ctx context.Context, candidate, current *Account, preferOAuth bool) bool {
	candidateRank := geminiPublicProtocolRank(ctx, candidate)
	currentRank := geminiPublicProtocolRank(ctx, current)
	if candidateRank != currentRank {
		return candidateRank < currentRank
	}
	return isPreferredAccountBySelectionOrder(candidate, current, preferOAuth)
}
