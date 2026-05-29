package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

func ensureAntigravityDefaultPassthrough(mapping map[string]string, model string) {
	if mapping == nil || model == "" {
		return
	}
	if _, exists := mapping[model]; exists {
		return
	}
	for pattern := range mapping {
		if matchWildcard(pattern, model) {
			return
		}
	}
	mapping[model] = model
}

func ensureAntigravityDefaultPassthroughs(mapping map[string]string, models []string) {
	for _, model := range models {
		ensureAntigravityDefaultPassthrough(mapping, model)
	}
}

func normalizeRequestedModelForLookup(platform, requestedModel string) string {
	trimmed := NormalizeRequestedModelForClaudeCapability(requestedModel)
	if trimmed == "" {
		return ""
	}
	if platform != PlatformGemini && platform != PlatformAntigravity {
		return trimmed
	}
	if trimmed == "gemini-3.1-pro-preview-customtools" {
		return "gemini-3.1-pro-preview"
	}
	return trimmed
}

func requestedModelLookupCandidates(platform, requestedModel string) []string {
	candidates := []string{strings.TrimSpace(requestedModel)}
	normalized := normalizeRequestedModelForLookup(platform, requestedModel)
	if normalized != "" {
		candidates = append(candidates, normalized)
	}
	if platform == PlatformDeepSeek {
		candidates = append(candidates, modelregistry.AlternateVersionVariants(requestedModel)...)
		if canonical, ok := modelregistry.ResolveToCanonicalID(requestedModel); ok {
			candidates = append(candidates, canonical)
		}
	}
	return uniqueNonEmptyStrings(candidates)
}

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func mappingSupportsRequestedModel(mapping map[string]string, requestedModel string) bool {
	if requestedModel == "" {
		return false
	}
	if _, exists := mapping[requestedModel]; exists {
		return true
	}
	for pattern := range mapping {
		if matchWildcard(pattern, requestedModel) {
			return true
		}
	}
	return false
}

func resolveRequestedModelInMapping(mapping map[string]string, requestedModel string) (mappedModel string, matched bool) {
	if requestedModel == "" {
		return "", false
	}
	if mappedModel, exists := mapping[requestedModel]; exists {
		return mappedModel, true
	}
	return matchWildcardMappingResult(mapping, requestedModel)
}

// IsModelSupported 检查模型是否在 model_mapping 中（支持通配符）
// 如果未配置 mapping，返回 true（允许所有模型）
func (a *Account) IsModelSupported(requestedModel string) bool {
	mapping := a.GetModelMapping()
	if len(mapping) == 0 {
		return true // 无映射 = 允许所有
	}
	// 精确匹配
	if a.Platform == domain.PlatformGrok {
		candidates := grokModelMatchCandidates(requestedModel)
		for _, candidate := range candidates {
			if _, exists := mapping[candidate]; exists {
				return true
			}
		}
		for pattern := range mapping {
			for _, candidate := range candidates {
				if matchWildcard(pattern, candidate) {
					return true
				}
			}
		}
		return false
	}
	for _, candidate := range requestedModelLookupCandidates(a.Platform, requestedModel) {
		if mappingSupportsRequestedModel(mapping, candidate) {
			return true
		}
	}
	return false
}

// GetMappedModel 获取映射后的模型名（支持通配符，最长优先匹配）
// 如果未配置 mapping，返回原始模型名
func (a *Account) GetMappedModel(requestedModel string) string {
	mappedModel, _ := a.ResolveMappedModel(requestedModel)
	return mappedModel
}

// ResolveMappedModel 获取映射后的模型名，并返回是否命中了账号级映射。
// matched=true 表示命中了精确映射或通配符映射，即使映射结果与原模型名相同。
func (a *Account) ResolveMappedModel(requestedModel string) (mappedModel string, matched bool) {
	mapping := a.GetModelMapping()
	if len(mapping) == 0 {
		return requestedModel, false
	}
	// 精确匹配优先
	if a.Platform == domain.PlatformGrok {
		candidates := grokModelMatchCandidates(requestedModel)
		for _, candidate := range candidates {
			if mappedModel, exists := mapping[candidate]; exists {
				return mappedModel, true
			}
		}
		return matchWildcardMappingResultCandidates(mapping, candidates...)
	}
	for _, candidate := range requestedModelLookupCandidates(a.Platform, requestedModel) {
		if mappedModel, matched := resolveRequestedModelInMapping(mapping, candidate); matched {
			return mappedModel, true
		}
	}
	return requestedModel, false
}
