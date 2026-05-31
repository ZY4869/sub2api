package service

import (
	"context"
	"strings"
)

const (
	publicModelCatalogExampleSourceDocs     = "docs_section"
	publicModelCatalogExampleSourceOverride = "override_template"
)

type publicModelCatalogExampleSpec struct {
	Keywords    []string
	OverrideID  string
	PageID      string
	Protocol    string
	EndpointKey string
}

type publicModelCatalogMarkdownSection struct {
	Content string
	Title   string
}

func (s *ModelCatalogService) buildPublicModelCatalogDetailExample(
	ctx context.Context,
	item PublicModelCatalogItem,
) (string, string, string, string, string, string) {
	spec, ok := selectPublicModelCatalogExampleSpec(item, s.publicModelCatalogExampleCapability(ctx, item))
	if !ok {
		return "", "", "", "", "", ""
	}
	if spec.OverrideID != "" {
		return publicModelCatalogExampleSourceOverride, spec.Protocol, spec.PageID, "", spec.OverrideID, PublicModelCatalogExampleValidationDryRunContract
	}
	if s == nil || s.docsService == nil || strings.TrimSpace(spec.PageID) == "" {
		return "", spec.Protocol, spec.PageID, "", "", ""
	}

	document, err := s.docsService.GetPageDocument(ctx, spec.PageID)
	if err != nil || document == nil {
		return "", spec.Protocol, spec.PageID, "", "", ""
	}

	selected := extractPublicModelCatalogExampleMarkdown(document.EffectiveContent, spec.PageID, spec.Keywords)
	if strings.TrimSpace(selected) == "" {
		return "", spec.Protocol, spec.PageID, "", "", ""
	}
	return publicModelCatalogExampleSourceDocs, spec.Protocol, spec.PageID, selected, "", PublicModelCatalogExampleValidationDryRunContract
}

func selectPublicModelCatalogExampleSpec(item PublicModelCatalogItem, capability string) (publicModelCatalogExampleSpec, bool) {
	endpoint, ok := pickPublicModelCatalogExampleEndpoint(item, capability)
	if !ok {
		return publicModelCatalogExampleSpec{}, false
	}
	protocol := endpoint.Protocol
	modelID := strings.ToLower(strings.TrimSpace(item.Model))
	mode := strings.ToLower(strings.TrimSpace(item.Mode))

	switch {
	case capability == "image_generation_tool":
		return publicModelCatalogExampleSpec{
			OverrideID:  "image-generation-tool",
			PageID:      "openai-native",
			Protocol:    firstNonEmptyTrimmed(protocol, PlatformOpenAI),
			EndpointKey: endpoint.Key,
		}, true
	case strings.Contains(modelID, "embedding") || strings.Contains(modelID, "embed"):
		return publicModelCatalogExampleSpec{
			OverrideID:  "embeddings",
			Protocol:    firstNonEmptyTrimmed(protocol, PlatformOpenAI),
			EndpointKey: endpoint.Key,
		}, true
	case strings.Contains(modelID, "tts") || strings.Contains(modelID, "speech"):
		return publicModelCatalogExampleSpec{
			OverrideID:  "tts",
			Protocol:    firstNonEmptyTrimmed(protocol, PlatformOpenAI),
			EndpointKey: endpoint.Key,
		}, true
	case capability == "image_generation" || mode == "image" || strings.Contains(modelID, "imagen") || strings.Contains(modelID, "-image"):
		return publicModelCatalogExampleSpec{
			OverrideID:  "image-generation",
			Protocol:    firstNonEmptyTrimmed(protocol, PlatformOpenAI),
			EndpointKey: endpoint.Key,
		}, true
	case mode == "video" || strings.Contains(modelID, "video"):
		return publicModelCatalogExampleSpec{
			OverrideID:  "video-generation",
			Protocol:    firstNonEmptyTrimmed(protocol, PlatformGrok),
			EndpointKey: endpoint.Key,
		}, true
	case protocol == publicModelCatalogProtocolVertex:
		return publicModelCatalogExampleSpec{
			PageID:      "vertex-batch",
			Protocol:    protocol,
			Keywords:    []string{"/vertex-batch/jobs", "/v1/vertex"},
			EndpointKey: endpoint.Key,
		}, true
	}

	switch protocol {
	case PlatformAnthropic:
		return publicModelCatalogExampleSpec{
			PageID:      "anthropic",
			Protocol:    protocol,
			Keywords:    []string{"/v1/messages"},
			EndpointKey: endpoint.Key,
		}, true
	case PlatformGemini:
		return publicModelCatalogExampleSpec{
			PageID:      "gemini",
			Protocol:    protocol,
			Keywords:    []string{"generatecontent", ":counttokens"},
			EndpointKey: endpoint.Key,
		}, true
	case PlatformGrok:
		return publicModelCatalogExampleSpec{
			PageID:      "grok",
			Protocol:    protocol,
			Keywords:    []string{"/grok/v1", "/v1/responses"},
			EndpointKey: endpoint.Key,
		}, true
	case PlatformAntigravity:
		return publicModelCatalogExampleSpec{
			PageID:      "antigravity",
			Protocol:    protocol,
			Keywords:    []string{"/antigravity"},
			EndpointKey: endpoint.Key,
		}, true
	default:
		return publicModelCatalogExampleSpec{
			PageID:      "common",
			Protocol:    firstNonEmptyTrimmed(protocol, PlatformOpenAI),
			Keywords:    []string{"/v1/responses"},
			EndpointKey: endpoint.Key,
		}, true
	}
}

func (s *ModelCatalogService) publicModelCatalogExampleCapability(ctx context.Context, item PublicModelCatalogItem) string {
	if s == nil || s.modelRegistryService == nil {
		return ""
	}
	modelID := NormalizeModelCatalogModelID(item.Model)
	if modelID == "" {
		return ""
	}
	detail, err := s.modelRegistryService.GetDetail(ctx, modelID)
	if err != nil || detail == nil {
		return ""
	}
	switch {
	case containsAnyRegistryValue(detail.Capabilities, "image_generation_tool"):
		return "image_generation_tool"
	case containsAnyRegistryValue(detail.Capabilities, "image_generation"):
		return "image_generation"
	default:
		return ""
	}
}

func pickPublicModelCatalogExampleProtocol(item PublicModelCatalogItem) string {
	if endpoint, ok := pickPublicModelCatalogExampleEndpoint(item, ""); ok {
		return endpoint.Protocol
	}
	for _, protocol := range item.RequestProtocols {
		normalized := strings.TrimSpace(protocol)
		switch normalized {
		case PlatformOpenAI, PlatformAnthropic, PlatformGemini, PlatformGrok, PlatformAntigravity, publicModelCatalogProtocolVertex:
			return normalized
		}
	}
	if normalized := publicModelCatalogProtocolFamily(item.Provider); normalized != "" {
		return normalized
	}
	return ""
}

func pickPublicModelCatalogExampleEndpoint(item PublicModelCatalogItem, capability string) (PublicModelProtocolEndpoint, bool) {
	endpoints := dedupePublicModelProtocolEndpoints(item.ProtocolEndpoints)
	if len(endpoints) == 0 {
		endpoints = normalizePublicModelProtocolEndpoints(nil, item.RequestProtocols, publicModelCatalogMetadataSourceForPublished(""))
	}
	if len(endpoints) == 0 {
		return PublicModelProtocolEndpoint{}, false
	}
	preferred := publicModelCatalogPreferredExampleEndpointKeys(item, capability)
	for _, key := range preferred {
		for _, endpoint := range endpoints {
			if endpoint.Key != key || !publicModelSupportAllowsSummary(endpoint.Support) {
				continue
			}
			if !publicModelCatalogEndpointSupportsCapability(item, endpoint, capability) {
				continue
			}
			return endpoint, true
		}
	}
	for _, endpoint := range endpoints {
		if publicModelSupportAllowsSummary(endpoint.Support) && publicModelCatalogEndpointSupportsCapability(item, endpoint, capability) {
			return endpoint, true
		}
	}
	return PublicModelProtocolEndpoint{}, false
}

func publicModelCatalogEndpointSupportsCapability(item PublicModelCatalogItem, endpoint PublicModelProtocolEndpoint, capability string) bool {
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return true
	}
	matrix := dedupePublicModelCapabilityMatrix(item.CapabilityMatrix)
	hasRelevantCapability := false
	for _, entry := range matrix {
		if entry.Capability != capability {
			continue
		}
		hasRelevantCapability = true
		if endpoint.Key != "" && entry.Endpoint != "" && entry.Endpoint != endpoint.Key {
			continue
		}
		if endpoint.Protocol != "" && entry.Protocol != "" && entry.Protocol != endpoint.Protocol {
			continue
		}
		if publicModelSupportAllowsSummary(entry.Support) {
			return true
		}
	}
	if hasRelevantCapability {
		return false
	}
	return publicModelEndpointMatchesCapability(endpoint, capability)
}

func publicModelCatalogPreferredExampleEndpointKeys(item PublicModelCatalogItem, capability string) []string {
	modelID := strings.ToLower(strings.TrimSpace(item.Model))
	mode := strings.ToLower(strings.TrimSpace(item.Mode))
	switch {
	case capability == "image_generation_tool" || capability == "image_generation" || mode == "image" || strings.Contains(modelID, "imagen") || strings.Contains(modelID, "-image"):
		return []string{"gemini.images.generations", "openai.images.generations", "grok.images.generations"}
	case mode == "video" || strings.Contains(modelID, "video"):
		return []string{"grok.videos.generations"}
	case strings.Contains(modelID, "embedding") || strings.Contains(modelID, "embed"):
		return []string{"openai.embeddings", "gemini.embedContent"}
	default:
		return []string{"openai.responses", "openai.chat.completions", "anthropic.messages", "gemini.generateContent", "grok.responses"}
	}
}

func extractPublicModelCatalogExampleMarkdown(markdown string, pageID string, keywords []string) string {
	parsed := parseAPIDocsDocument(markdown)
	pageBody := strings.TrimSpace(parsed.Pages[pageID])
	if pageBody == "" {
		return ""
	}

	sections := splitPublicModelCatalogMarkdownSections(pageBody)
	selected := selectPublicModelCatalogMarkdownSection(sections, keywords)
	if strings.TrimSpace(selected.Content) == "" {
		return buildAPIDocsPageSection(parsed.Title, pageID, pageBody)
	}
	return buildAPIDocsPageSection(parsed.Title, pageID, selected.Content)
}

func splitPublicModelCatalogMarkdownSections(body string) []publicModelCatalogMarkdownSection {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	sections := make([]publicModelCatalogMarkdownSection, 0, 6)
	currentTitle := ""
	currentLines := make([]string, 0, len(lines))
	inFence := false
	fenceMarker := ""

	pushSection := func() {
		content := strings.Trim(strings.Join(currentLines, "\n"), "\n")
		if content == "" {
			currentLines = currentLines[:0]
			return
		}
		sections = append(sections, publicModelCatalogMarkdownSection{
			Title:   strings.TrimSpace(currentTitle),
			Content: content,
		})
		currentLines = currentLines[:0]
	}

	for _, line := range lines {
		if fence := parseAPIDocsFence(line); fence != "" {
			if !inFence {
				inFence = true
				fenceMarker = fence
			} else if matchesAPIDocsFence(line, fenceMarker) {
				inFence = false
				fenceMarker = ""
			}
		}
		if !inFence {
			if match := strings.TrimSpace(line); strings.HasPrefix(match, "### ") {
				pushSection()
				currentTitle = strings.TrimSpace(strings.TrimPrefix(match, "### "))
			}
		}
		currentLines = append(currentLines, line)
	}
	pushSection()
	return sections
}

func selectPublicModelCatalogMarkdownSection(
	sections []publicModelCatalogMarkdownSection,
	keywords []string,
) publicModelCatalogMarkdownSection {
	bestIndex := -1
	bestScore := -1
	fallbackIndex := -1
	for index, section := range sections {
		content := strings.TrimSpace(section.Content)
		if content == "" || !strings.Contains(content, "```") {
			continue
		}
		if fallbackIndex < 0 {
			fallbackIndex = index
		}
		score := publicModelCatalogMarkdownSectionScore(section, keywords)
		if score > bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	if bestIndex >= 0 && bestScore > 0 {
		return sections[bestIndex]
	}
	if fallbackIndex >= 0 {
		return sections[fallbackIndex]
	}
	return publicModelCatalogMarkdownSection{}
}

func publicModelCatalogMarkdownSectionScore(
	section publicModelCatalogMarkdownSection,
	keywords []string,
) int {
	if len(keywords) == 0 {
		return 1
	}
	haystack := strings.ToLower(section.Title + "\n" + section.Content)
	score := 0
	for _, keyword := range keywords {
		trimmed := strings.ToLower(strings.TrimSpace(keyword))
		if trimmed == "" {
			continue
		}
		if strings.Contains(haystack, trimmed) {
			score++
		}
	}
	return score
}
