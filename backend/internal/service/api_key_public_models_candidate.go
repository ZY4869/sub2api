package service

import (
	"strings"
)

type apiKeyPublicProjectionCandidate struct {
	MatchID     string
	AliasID     string
	SourceID    string
	DisplayName string
	Platform    string
	ExposeAlias bool
}

func apiKeyPublicProjectionPlatform(bindingPlatform string, requestedPlatform string) string {
	bindingPlatform = strings.TrimSpace(strings.ToLower(bindingPlatform))
	requestedPlatform = strings.TrimSpace(strings.ToLower(requestedPlatform))
	if bindingPlatform == "" {
		return ""
	}
	if requestedPlatform == "" || strings.EqualFold(bindingPlatform, requestedPlatform) {
		return bindingPlatform
	}
	if bindingPlatform != PlatformProtocolGateway {
		return ""
	}
	switch requestedPlatform {
	case PlatformOpenAI, PlatformAnthropic, PlatformGemini:
		return requestedPlatform
	default:
		return ""
	}
}

func buildAPIKeyPublicProjectionCandidate(mode, alias, source, platform string) (apiKeyPublicProjectionCandidate, bool) {
	alias = strings.TrimSpace(alias)
	source = strings.TrimSpace(source)
	if alias == "" && source == "" {
		return apiKeyPublicProjectionCandidate{}, false
	}
	if alias == "" {
		alias = source
	}
	if source == "" {
		source = alias
	}
	explicitAlias := shouldExposePublicAlias(platform, alias, source)

	if explicitAlias {
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
			ExposeAlias: true,
		}, true
	}

	switch NormalizeAPIKeyModelDisplayMode(mode) {
	case APIKeyModelDisplayModeSourceOnly:
		return apiKeyPublicProjectionCandidate{
			MatchID:     source,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: source,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	case APIKeyModelDisplayModeAliasAndSource:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias + " | " + source,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	default:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	}
}

func apiKeyPublicProjectionPublicID(platform string, candidate apiKeyPublicProjectionCandidate, sourceID string) string {
	if candidate.ExposeAlias {
		if aliasID := normalizeRegistryID(candidate.AliasID); aliasID != "" {
			return aliasID
		}
		if matchID := normalizeRegistryID(candidate.MatchID); matchID != "" {
			return matchID
		}
	}
	if !strings.EqualFold(platform, PlatformGrok) {
		return sourceID
	}
	if aliasID := normalizeRegistryID(candidate.AliasID); aliasID != "" && aliasID != sourceID {
		return aliasID
	}
	if matchID := normalizeRegistryID(candidate.MatchID); matchID != "" && matchID != sourceID {
		return matchID
	}
	if publicID := grokPublicModelForDetectedSource(sourceID); publicID != "" {
		return publicID
	}
	return sourceID
}

func shouldExposePublicAlias(platform, alias, source string) bool {
	alias = strings.TrimSpace(alias)
	source = strings.TrimSpace(source)
	if alias == "" || source == "" || alias == source {
		return false
	}
	if strings.Contains(alias, "*") {
		return false
	}
	if strings.EqualFold(platform, PlatformGemini) && alias == DefaultVertexPublicModelAlias(source) {
		return false
	}
	return true
}

func bindingMatchesProjectionCandidate(
	modelPatterns []string,
	publicID string,
	candidate apiKeyPublicProjectionCandidate,
) bool {
	for _, modelID := range []string{
		publicID,
		candidate.MatchID,
		candidate.AliasID,
		candidate.SourceID,
	} {
		if _, matched := bindingMatchesModel(modelPatterns, modelID); matched {
			return true
		}
	}
	return false
}

func countAliasOnlyPublicEntries(entries []APIKeyPublicModelEntry) int {
	count := 0
	for _, entry := range entries {
		if strings.TrimSpace(entry.PublicID) == "" {
			continue
		}
		if strings.TrimSpace(entry.PublicID) != strings.TrimSpace(entry.SourceID) {
			count++
		}
	}
	return count
}
