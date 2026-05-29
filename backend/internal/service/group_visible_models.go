package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

func NormalizeGroupVisibleModelPatterns(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (g *Group) HasVisibleModelPatternFilter() bool {
	return g != nil && len(NormalizeGroupVisibleModelPatterns(g.VisibleModelPatterns)) > 0
}

func (g *Group) AllowsVisibleModel(modelID string, aliases ...string) bool {
	if g == nil {
		return true
	}
	patterns := NormalizeGroupVisibleModelPatterns(g.VisibleModelPatterns)
	if len(patterns) == 0 {
		return true
	}
	for _, candidate := range visibleModelCandidates(modelID, aliases...) {
		if _, matched := bindingMatchesModel(patterns, candidate); matched {
			return true
		}
	}
	return false
}

func filterAPIKeyPublicEntriesByGroupVisibleModels(entries []APIKeyPublicModelEntry, group *Group) []APIKeyPublicModelEntry {
	if len(entries) == 0 || group == nil || !group.HasVisibleModelPatternFilter() {
		return entries
	}
	filtered := make([]APIKeyPublicModelEntry, 0, len(entries))
	for _, entry := range entries {
		if group.AllowsVisibleModel(entry.PublicID, entry.AliasID, entry.SourceID, entry.DisplayName) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func filterPublicProjectionEntriesByGroupVisibleModels(entries []PublicModelProjectionEntry, group *Group) []PublicModelProjectionEntry {
	if len(entries) == 0 || group == nil || !group.HasVisibleModelPatternFilter() {
		return entries
	}
	filtered := make([]PublicModelProjectionEntry, 0, len(entries))
	for _, entry := range entries {
		aliases := make([]string, 0, len(entry.AliasIDs)+len(entry.SourceIDs)+1)
		aliases = append(aliases, entry.AliasIDs...)
		aliases = append(aliases, entry.SourceIDs...)
		aliases = append(aliases, entry.DisplayName)
		if group.AllowsVisibleModel(entry.PublicID, aliases...) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func visibleModelCandidates(modelID string, aliases ...string) []string {
	candidates := make([]string, 0, len(aliases)+1)
	candidates = append(candidates, modelID)
	candidates = append(candidates, aliases...)
	return dedupeStrings(candidates)
}

func WithVisibleModelCandidates(ctx context.Context, values ...string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	merged := visibleModelCandidates("", values...)
	if len(merged) == 0 {
		return ctx
	}
	existing := VisibleModelCandidatesFromContext(ctx)
	merged = visibleModelCandidates("", append(existing, merged...)...)
	return context.WithValue(ctx, ctxkey.ModelCandidates, merged)
}

func VisibleModelCandidatesFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	values, _ := ctx.Value(ctxkey.ModelCandidates).([]string)
	return append([]string(nil), values...)
}
