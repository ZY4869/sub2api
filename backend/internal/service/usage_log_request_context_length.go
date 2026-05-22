package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

const usageLogMillionContextTokens = 1_000_000

func ApplyUsageLogRequestContextLength(log *UsageLog) {
	if log == nil {
		return
	}
	log.RequestContextLengthTokens = ResolveUsageLogRequestContextLengthTokens(log)
}

func ResolveUsageLogRequestContextLengthTokens(log *UsageLog) *int {
	if log == nil {
		return nil
	}
	if log.RequestContextLengthTokens != nil && *log.RequestContextLengthTokens > 0 {
		return log.RequestContextLengthTokens
	}
	if usageLogExplicitMillionContextRequested(log) {
		tokens := usageLogMillionContextTokens
		return &tokens
	}
	ids := usageLogRequestContextCandidateIDs(log)
	if len(ids) == 0 {
		return nil
	}
	resolved, ok := modelregistry.ResolveContextWindowTokens(ids...)
	if !ok || resolved <= 0 {
		return nil
	}
	tokens := int(resolved)
	return &tokens
}

func usageLogExplicitMillionContextRequested(log *UsageLog) bool {
	if log == nil {
		return false
	}
	if log.MillionContextRequested != nil && *log.MillionContextRequested {
		return true
	}
	raw := strings.TrimSpace(stringPtrValue(log.RequestedModelRaw))
	if raw == "" {
		return false
	}
	_, requested, _ := StripClaudeMillionContextSuffix(raw)
	return requested
}

func usageLogRequestContextCandidateIDs(log *UsageLog) []string {
	if log == nil {
		return nil
	}
	candidates := []string{
		strings.TrimSpace(stringPtrValue(log.RequestedModelNormalized)),
		strings.TrimSpace(log.RequestedModel),
		strings.TrimSpace(log.Model),
		strings.TrimSpace(stringPtrValue(log.UpstreamModel)),
	}
	ids := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		ids = append(ids, candidate)
	}
	return ids
}
