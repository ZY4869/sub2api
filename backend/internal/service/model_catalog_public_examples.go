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
	Keywords   []string
	OverrideID string
	PageID     string
	Protocol   string
}

type publicModelCatalogMarkdownSection struct {
	Content string
	Title   string
}

func (s *ModelCatalogService) buildPublicModelCatalogDetailExample(
	ctx context.Context,
	item PublicModelCatalogItem,
) (string, string, string, string, string) {
	spec := selectPublicModelCatalogExampleSpec(item, s.publicModelCatalogExampleCapability(ctx, item))
	if spec.OverrideID != "" {
		return publicModelCatalogExampleSourceOverride, spec.Protocol, spec.PageID, "", spec.OverrideID
	}
	if s == nil || s.docsService == nil || strings.TrimSpace(spec.PageID) == "" {
		return "", spec.Protocol, spec.PageID, "", ""
	}

	document, err := s.docsService.GetPageDocument(ctx, spec.PageID)
	if err != nil || document == nil {
		return "", spec.Protocol, spec.PageID, "", ""
	}

	selected := extractPublicModelCatalogExampleMarkdown(document.EffectiveContent, spec.PageID, spec.Keywords)
	if strings.TrimSpace(selected) == "" {
		return "", spec.Protocol, spec.PageID, "", ""
	}
	return publicModelCatalogExampleSourceDocs, spec.Protocol, spec.PageID, selected, ""
}

func selectPublicModelCatalogExampleSpec(item PublicModelCatalogItem, capability string) publicModelCatalogExampleSpec {
	protocol := pickPublicModelCatalogExampleProtocol(item)
	modelID := strings.ToLower(strings.TrimSpace(item.Model))
	mode := strings.ToLower(strings.TrimSpace(item.Mode))

	switch {
	case capability == "image_generation_tool":
		return publicModelCatalogExampleSpec{
			OverrideID: "image-generation-tool",
			PageID:     "openai-native",
			Protocol:   firstNonEmptyTrimmed(protocol, PlatformOpenAI),
		}
	case strings.Contains(modelID, "embedding") || strings.Contains(modelID, "embed"):
		return publicModelCatalogExampleSpec{
			OverrideID: "embeddings",
			Protocol:   firstNonEmptyTrimmed(protocol, PlatformOpenAI),
		}
	case strings.Contains(modelID, "tts") || strings.Contains(modelID, "speech"):
		return publicModelCatalogExampleSpec{
			OverrideID: "tts",
			Protocol:   firstNonEmptyTrimmed(protocol, PlatformOpenAI),
		}
	case capability == "image_generation" || mode == "image" || strings.Contains(modelID, "imagen") || strings.Contains(modelID, "-image"):
		return publicModelCatalogExampleSpec{
			OverrideID: "image-generation",
			Protocol:   firstNonEmptyTrimmed(protocol, PlatformOpenAI),
		}
	case mode == "video" || strings.Contains(modelID, "video"):
		return publicModelCatalogExampleSpec{
			OverrideID: "video-generation",
			Protocol:   firstNonEmptyTrimmed(protocol, PlatformGrok),
		}
	case protocol == publicModelCatalogProtocolVertex:
		return publicModelCatalogExampleSpec{
			PageID:   "vertex-batch",
			Protocol: protocol,
			Keywords: []string{"/vertex-batch/jobs", "/v1/vertex"},
		}
	}

	switch protocol {
	case PlatformAnthropic:
		return publicModelCatalogExampleSpec{
			PageID:   "anthropic",
			Protocol: protocol,
			Keywords: []string{"/v1/messages"},
		}
	case PlatformGemini:
		return publicModelCatalogExampleSpec{
			PageID:   "gemini",
			Protocol: protocol,
			Keywords: []string{"generatecontent", ":counttokens"},
		}
	case PlatformGrok:
		return publicModelCatalogExampleSpec{
			PageID:   "grok",
			Protocol: protocol,
			Keywords: []string{"/grok/v1", "/v1/responses"},
		}
	case PlatformAntigravity:
		return publicModelCatalogExampleSpec{
			PageID:   "antigravity",
			Protocol: protocol,
			Keywords: []string{"/antigravity"},
		}
	default:
		return publicModelCatalogExampleSpec{
			PageID:   "common",
			Protocol: firstNonEmptyTrimmed(protocol, PlatformOpenAI),
			Keywords: []string{"/v1/responses"},
		}
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
