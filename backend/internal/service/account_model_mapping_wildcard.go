package service

import (
	"sort"
	"strings"
)

// matchAntigravityWildcard 通配符匹配（仅支持末尾 *）
// 用于 model_mapping 的通配符匹配
func matchAntigravityWildcard(pattern, str string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(str, prefix)
	}
	return pattern == str
}

// matchWildcard 通用通配符匹配（仅支持末尾 *）
// 复用 Antigravity 的通配符逻辑，供其他平台使用
func matchWildcard(pattern, str string) bool {
	return matchAntigravityWildcard(pattern, str)
}

func matchWildcardMappingResult(mapping map[string]string, requestedModel string) (string, bool) {
	// 收集所有匹配的 pattern，按长度降序排序（最长优先）
	type patternMatch struct {
		pattern string
		target  string
	}
	var matches []patternMatch

	for pattern, target := range mapping {
		if matchWildcard(pattern, requestedModel) {
			matches = append(matches, patternMatch{pattern, target})
		}
	}

	if len(matches) == 0 {
		return requestedModel, false // 无匹配，返回原始模型名
	}

	// 按 pattern 长度降序排序
	sort.Slice(matches, func(i, j int) bool {
		if len(matches[i].pattern) != len(matches[j].pattern) {
			return len(matches[i].pattern) > len(matches[j].pattern)
		}
		return matches[i].pattern < matches[j].pattern
	})

	return matches[0].target, true
}

func matchWildcardMappingResultCandidates(mapping map[string]string, requestedModels ...string) (string, bool) {
	for _, requestedModel := range requestedModels {
		if requestedModel == "" {
			continue
		}
		if mapped, ok := matchWildcardMappingResult(mapping, requestedModel); ok {
			return mapped, true
		}
	}
	if len(requestedModels) == 0 {
		return "", false
	}
	return requestedModels[0], false
}
