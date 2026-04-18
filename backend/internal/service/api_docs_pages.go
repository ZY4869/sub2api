package service

import (
	"regexp"
	"strings"
)

var (
	apiDocsFencePattern   = regexp.MustCompile(`^\s*(` + "```+" + `|~~~+)`)
	apiDocsTitlePattern   = regexp.MustCompile(`^#\s+(.+)$`)
	apiDocsPageIDPattern  = regexp.MustCompile(`^##\s+(.+)$`)
)

var apiDocsPageOrder = []string{
	"common",
	"openai-native",
	"openai",
	"anthropic",
	"gemini",
	"grok",
	"antigravity",
	"vertex-batch",
	"document-ai",
}

const defaultAPIDocsTitleFallback = "API 文档中心"

type apiDocsParsedDocument struct {
	Title string
	Pages map[string]string
}

func isAPIDocsPageID(value string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	for _, pageID := range apiDocsPageOrder {
		if normalized == pageID {
			return true
		}
	}
	return false
}

func normalizeAPIDocsPageID(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if isAPIDocsPageID(normalized) {
		return normalized
	}
	return ""
}

func apiDocsPageSettingKey(pageID string) string {
	return SettingKeyAPIDocsMarkdown + "_page_" + pageID
}

func parseAPIDocsDocument(content string) apiDocsParsedDocument {
	normalized := normalizeAPIDocsContent(content)
	if normalized == "" {
		return apiDocsParsedDocument{
			Title: defaultAPIDocsTitleFallback,
			Pages: map[string]string{},
		}
	}

	lines := strings.Split(normalized, "\n")
	title := defaultAPIDocsTitleFallback
	pages := make(map[string][]string)
	currentPageID := ""
	inFence := false
	fenceMarker := ""

	for _, line := range lines {
		if title == defaultAPIDocsTitleFallback {
			if match := apiDocsTitlePattern.FindStringSubmatch(line); match != nil {
				candidate := strings.TrimSpace(match[1])
				if candidate != "" {
					title = candidate
				}
			}
		}

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
			if match := apiDocsPageIDPattern.FindStringSubmatch(line); match != nil {
				if pageID := normalizeAPIDocsPageID(match[1]); pageID != "" {
					currentPageID = pageID
					if _, ok := pages[currentPageID]; !ok {
						pages[currentPageID] = []string{}
					}
					continue
				}
			}
		}

		if currentPageID != "" {
			pages[currentPageID] = append(pages[currentPageID], line)
		}
	}

	result := make(map[string]string, len(pages))
	for pageID, sectionLines := range pages {
		result[pageID] = strings.Trim(strings.Join(sectionLines, "\n"), "\n")
	}

	return apiDocsParsedDocument{
		Title: title,
		Pages: result,
	}
}

func buildAPIDocsDocument(title string, pages map[string]string) string {
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		normalizedTitle = defaultAPIDocsTitleFallback
	}

	lines := []string{"# " + normalizedTitle, ""}
	for index, pageID := range apiDocsPageOrder {
		lines = append(lines, "## "+pageID)
		body := strings.Trim(strings.ReplaceAll(pages[pageID], "\r\n", "\n"), "\n")
		if body != "" {
			lines = append(lines, strings.Split(body, "\n")...)
		}
		if index < len(apiDocsPageOrder)-1 {
			lines = append(lines, "")
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func buildAPIDocsPageSection(title string, pageID string, pageBody string) string {
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		normalizedTitle = defaultAPIDocsTitleFallback
	}

	lines := []string{"# " + normalizedTitle, "", "## " + pageID}
	body := strings.Trim(strings.ReplaceAll(pageBody, "\r\n", "\n"), "\n")
	if body != "" {
		lines = append(lines, strings.Split(body, "\n")...)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func normalizeAPIDocsPageSectionContent(pageID string, title string, content string) string {
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		normalizedTitle = defaultAPIDocsTitleFallback
	}

	normalized := normalizeAPIDocsContent(content)
	if normalized == "" {
		return ""
	}

	parsed := parseAPIDocsDocument(normalized)
	if strings.TrimSpace(parsed.Title) != "" {
		normalizedTitle = parsed.Title
	}
	if body, ok := parsed.Pages[pageID]; ok {
		return buildAPIDocsPageSection(normalizedTitle, pageID, body)
	}

	lines := strings.Split(strings.TrimRight(normalized, "\n"), "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
		lines = lines[1:]
		if len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
			lines = lines[1:]
		}
	}
	return buildAPIDocsPageSection(normalizedTitle, pageID, strings.Join(lines, "\n"))
}

func parseAPIDocsFence(line string) string {
	match := apiDocsFencePattern.FindStringSubmatch(line)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func matchesAPIDocsFence(line string, fence string) bool {
	return strings.TrimSpace(line) == fence
}
