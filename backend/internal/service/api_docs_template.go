package service

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

const (
	apiDocsIndexPath = "docs/index.md"
	apiDocsPagesGlob = "docs/pages/*.md"
)

//go:embed docs/index.md docs/pages/*.md
var apiDocsTemplateFS embed.FS

var defaultAPIDocsTemplate = mustBuildDefaultAPIDocsTemplate()

func mustBuildDefaultAPIDocsTemplate() string {
	content, err := buildDefaultAPIDocsTemplateFromFS(apiDocsTemplateFS)
	if err != nil {
		panic(err)
	}
	return content
}

func buildDefaultAPIDocsTemplateFromFS(docsFS fs.FS) (string, error) {
	title, err := loadAPIDocsIndexTitle(docsFS)
	if err != nil {
		return "", err
	}

	pageFiles, err := fs.Glob(docsFS, apiDocsPagesGlob)
	if err != nil {
		return "", fmt.Errorf("glob api docs pages: %w", err)
	}

	pages := make(map[string]string, len(apiDocsPageOrder))
	for _, filePath := range pageFiles {
		pageID := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))
		normalizedPageID := normalizeAPIDocsPageID(pageID)
		if normalizedPageID == "" {
			return "", fmt.Errorf("api docs page filename is invalid: %s", filePath)
		}
		if _, exists := pages[normalizedPageID]; exists {
			return "", fmt.Errorf("duplicate api docs page file: %s", normalizedPageID)
		}

		pageBody, err := loadAPIDocsPageBody(docsFS, filePath, normalizedPageID)
		if err != nil {
			return "", err
		}
		pages[normalizedPageID] = pageBody
	}

	for _, pageID := range apiDocsPageOrder {
		if _, ok := pages[pageID]; !ok {
			return "", fmt.Errorf("missing api docs page file for %s", pageID)
		}
	}

	if len(pages) != len(apiDocsPageOrder) {
		return "", fmt.Errorf("api docs page file count mismatch: got %d want %d", len(pages), len(apiDocsPageOrder))
	}

	return buildAPIDocsDocument(title, pages), nil
}

func loadAPIDocsIndexTitle(docsFS fs.FS) (string, error) {
	content, err := fs.ReadFile(docsFS, apiDocsIndexPath)
	if err != nil {
		return "", fmt.Errorf("read api docs index: %w", err)
	}

	normalized := normalizeAPIDocsContent(string(content))
	if normalized == "" {
		return "", fmt.Errorf("api docs index is empty")
	}

	for _, line := range strings.Split(normalized, "\n") {
		match := apiDocsTitlePattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		title := strings.TrimSpace(match[1])
		if title != "" {
			return title, nil
		}
	}

	return "", fmt.Errorf("api docs index missing document title")
}

func loadAPIDocsPageBody(docsFS fs.FS, filePath string, expectedPageID string) (string, error) {
	content, err := fs.ReadFile(docsFS, filePath)
	if err != nil {
		return "", fmt.Errorf("read api docs page %s: %w", expectedPageID, err)
	}

	normalized := normalizeAPIDocsContent(string(content))
	if normalized == "" {
		return "", fmt.Errorf("api docs page %s is empty", expectedPageID)
	}

	parsed := parseAPIDocsDocument(normalized)
	if len(parsed.Pages) != 1 {
		return "", fmt.Errorf("api docs page %s must contain exactly one page section", expectedPageID)
	}

	pageBody, ok := parsed.Pages[expectedPageID]
	if !ok {
		return "", fmt.Errorf("api docs page header mismatch in %s: expected ## %s", filePath, expectedPageID)
	}

	return pageBody, nil
}
