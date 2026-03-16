package service

import (
	"context"
	"strings"
)

func (s *GatewayService) isModelSupportedByAccountWithContext(ctx context.Context, account *Account, requestedModel string) bool {
	if account.Platform == PlatformAntigravity {
		if strings.TrimSpace(requestedModel) == "" {
			return true
		}
		mapped := mapAntigravityModel(account, requestedModel)
		if mapped == "" {
			return false
		}
		if enabled, ok := ThinkingEnabledFromContext(ctx); ok {
			finalModel := applyThinkingModelSuffix(mapped, enabled)
			if finalModel == mapped {
				return true
			}
			return account.IsModelSupported(finalModel)
		}
		return true
	}
	return s.isModelSupportedByAccount(account, requestedModel)
}
func (s *GatewayService) isModelSupportedByAccount(account *Account, requestedModel string) bool {
	if account != nil && account.Type == AccountTypeBedrock {
		if strings.TrimSpace(requestedModel) == "" {
			return true
		}
		_, ok := ResolveBedrockModelID(account, requestedModel)
		return ok
	}
	if account.Platform == PlatformAntigravity {
		if strings.TrimSpace(requestedModel) == "" {
			return true
		}
		return mapAntigravityModel(account, requestedModel) != ""
	}
	if account.Platform == PlatformSora {
		return s.isSoraModelSupportedByAccount(account, requestedModel)
	}
	canonicalModel := s.resolveCanonicalRequestModel(context.Background(), requestedModel)
	if canonicalModel == "" {
		return account.IsModelSupported(requestedModel)
	}
	if account.IsModelSupported(canonicalModel) {
		return true
	}
	return account.IsModelSupported(s.resolveUpstreamModelID(context.Background(), account, canonicalModel))
}
func (s *GatewayService) isSoraModelSupportedByAccount(account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	if strings.TrimSpace(requestedModel) == "" {
		return true
	}
	mapping := account.GetModelMapping()
	if len(mapping) == 0 || account.IsModelSupported(requestedModel) {
		return true
	}
	aliases := buildSoraModelAliases(requestedModel)
	if len(aliases) == 0 {
		return false
	}
	hasSoraSelector := false
	for pattern := range mapping {
		if !isSoraModelSelector(pattern) {
			continue
		}
		hasSoraSelector = true
		if matchPatternAnyAlias(pattern, aliases) {
			return true
		}
	}
	return !hasSoraSelector
}
func matchPatternAnyAlias(pattern string, aliases []string) bool {
	normalizedPattern := strings.ToLower(strings.TrimSpace(pattern))
	if normalizedPattern == "" {
		return false
	}
	for _, alias := range aliases {
		if matchWildcard(normalizedPattern, alias) {
			return true
		}
	}
	return false
}
func isSoraModelSelector(pattern string) bool {
	p := strings.ToLower(strings.TrimSpace(pattern))
	if p == "" {
		return false
	}
	switch {
	case strings.HasPrefix(p, "sora"), strings.HasPrefix(p, "gpt-image"), strings.HasPrefix(p, "prompt-enhance"), strings.HasPrefix(p, "sy_"):
		return true
	}
	return p == "video" || p == "image"
}
func buildSoraModelAliases(requestedModel string) []string {
	modelID := strings.ToLower(strings.TrimSpace(requestedModel))
	if modelID == "" {
		return nil
	}
	aliases := make([]string, 0, 8)
	addAlias := func(value string) {
		v := strings.ToLower(strings.TrimSpace(value))
		if v == "" {
			return
		}
		for _, existing := range aliases {
			if existing == v {
				return
			}
		}
		aliases = append(aliases, v)
	}
	addAlias(modelID)
	cfg, ok := GetSoraModelConfig(modelID)
	if ok {
		addAlias(cfg.Model)
		switch cfg.Type {
		case "video":
			addAlias("video")
			addAlias("sora")
			addAlias(soraVideoFamilyAlias(modelID))
		case "image":
			addAlias("image")
			addAlias("gpt-image")
		case "prompt_enhance":
			addAlias("prompt-enhance")
		}
		return aliases
	}
	switch {
	case strings.HasPrefix(modelID, "sora"):
		addAlias("video")
		addAlias("sora")
		addAlias(soraVideoFamilyAlias(modelID))
	case strings.HasPrefix(modelID, "gpt-image"):
		addAlias("image")
		addAlias("gpt-image")
	case strings.HasPrefix(modelID, "prompt-enhance"):
		addAlias("prompt-enhance")
	default:
		return nil
	}
	return aliases
}
func soraVideoFamilyAlias(modelID string) string {
	switch {
	case strings.HasPrefix(modelID, "sora2pro-hd"):
		return "sora2pro-hd"
	case strings.HasPrefix(modelID, "sora2pro"):
		return "sora2pro"
	case strings.HasPrefix(modelID, "sora2"):
		return "sora2"
	default:
		return ""
	}
}
