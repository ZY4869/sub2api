package service

import (
	"regexp"
	"strings"
)

func NormalizeInboundEndpoint(path string) string {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return ""
	}
	bestScore := -1
	bestEndpoint := ""
	for _, entry := range publicEndpointRegistry {
		for _, candidate := range publicEndpointNormalizationCandidates(normalizedPath, entry) {
			for _, route := range entry.Routes {
				if !matchPublicEndpointRoute(candidate, route.Pattern) {
					continue
				}
				score := publicEndpointRouteSpecificity(route.Pattern)
				if score > bestScore {
					bestScore = score
					bestEndpoint = entry.CanonicalEndpoint
				}
			}
		}
	}
	if bestEndpoint != "" {
		return bestEndpoint
	}
	return normalizedPath
}

func matchPublicEndpointRoute(path string, pattern string) bool {
	path = normalizePublicRouteValue(path)
	pattern = normalizePublicRouteValue(pattern)
	if path == pattern {
		return true
	}

	expression := buildPublicRouteExpression(pattern)
	matched, _ := regexp.MatchString(expression, path)
	return matched
}

func buildPublicRouteExpression(pattern string) string {
	segments := splitPublicRouteSegments(pattern)
	if len(segments) == 0 {
		return `^/?$`
	}

	var expression strings.Builder
	writeBuilderString(&expression, "^")
	for _, segment := range segments {
		writeBuilderString(&expression, "/")
		switch {
		case strings.HasPrefix(segment, "*"):
			writeBuilderString(&expression, ".+")
			writeBuilderString(&expression, "$")
			return expression.String()
		case strings.HasPrefix(segment, ":"):
			writeBuilderString(&expression, `[^/]+`)
		default:
			segmentExpression := buildPublicRouteSegmentExpression(segment)
			segmentExpression = strings.TrimPrefix(segmentExpression, "^")
			segmentExpression = strings.TrimSuffix(segmentExpression, "$")
			writeBuilderString(&expression, segmentExpression)
		}
	}
	writeBuilderString(&expression, "$")
	return expression.String()
}

func normalizePublicRouteValue(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return ""
	}
	if len(normalized) > 1 {
		normalized = strings.TrimRight(normalized, "/")
	}
	return normalized
}

func publicEndpointNormalizationCandidates(path string, entry PublicEndpointRegistryEntry) []string {
	candidates := []string{path}
	for _, prefix := range entry.NormalizePrefixes {
		trimmedPrefix := strings.TrimSpace(prefix)
		if trimmedPrefix == "" {
			continue
		}
		if trimmed, ok := strings.CutPrefix(path, trimmedPrefix); ok && strings.HasPrefix(trimmed, "/") {
			candidates = append(candidates, trimmed)
		}
	}
	return uniqueTrimmedStringsPreserveCase(candidates)
}

func uniqueTrimmedStringsPreserveCase(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func splitPublicRouteSegments(value string) []string {
	normalized := strings.Trim(normalizePublicRouteValue(value), "/")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "/")
}

func publicEndpointRouteSpecificity(pattern string) int {
	score := 0
	for _, segment := range splitPublicRouteSegments(pattern) {
		literalCount, paramCount, wildcard := publicEndpointRouteSegmentWeights(segment)
		score += 2
		score += literalCount * 10
		score += paramCount * 5
		if wildcard {
			score++
		}
	}
	return score
}

func publicEndpointRouteSegmentWeights(segment string) (literalCount int, paramCount int, wildcard bool) {
	for index := 0; index < len(segment); {
		switch segment[index] {
		case '*':
			if index == 0 {
				return literalCount, paramCount, true
			}
		case '{':
			if end := strings.IndexByte(segment[index:], '}'); end >= 0 {
				paramCount++
				index += end + 1
				continue
			}
		case ':':
			if index > 0 && segment[index-1] == '}' {
				literalCount++
				index++
				continue
			}
			if index+1 < len(segment) && isRouteParamStart(segment[index+1]) {
				paramCount++
				end := index + 2
				for end < len(segment) && isRouteParamContinue(segment[end]) {
					end++
				}
				index = end
				continue
			}
		}
		literalCount++
		index++
	}
	return literalCount, paramCount, false
}

func buildPublicRouteSegmentExpression(pattern string) string {
	var expression strings.Builder
	writeBuilderString(&expression, "^")
	for index := 0; index < len(pattern); {
		switch pattern[index] {
		case '{':
			if end := strings.IndexByte(pattern[index:], '}'); end >= 0 {
				writeBuilderString(&expression, `[^/]+`)
				index += end + 1
				continue
			}
		case ':':
			if index > 0 && pattern[index-1] == '}' {
				break
			}
			if index+1 < len(pattern) && isRouteParamStart(pattern[index+1]) {
				end := index + 2
				for end < len(pattern) && isRouteParamContinue(pattern[end]) {
					end++
				}
				writeBuilderString(&expression, `[^/]+`)
				index = end
				continue
			}
		}
		writeBuilderString(&expression, regexp.QuoteMeta(string(pattern[index])))
		index++
	}
	writeBuilderString(&expression, "$")
	return expression.String()
}

func writeBuilderString(builder *strings.Builder, value string) {
	_, _ = builder.WriteString(value)
}

func isRouteParamStart(ch byte) bool {
	return ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isRouteParamContinue(ch byte) bool {
	return isRouteParamStart(ch) || (ch >= '0' && ch <= '9')
}
