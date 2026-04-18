package service

import (
	"context"
	"errors"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrAPIDocsEmptyContent = infraerrors.BadRequest(
	"API_DOCS_EMPTY",
	"api docs markdown cannot be empty",
)

var ErrAPIDocsInvalidPage = infraerrors.BadRequest(
	"API_DOCS_PAGE_INVALID",
	"api docs page is invalid",
)

type APIDocsDocument struct {
	EffectiveContent string
	DefaultContent   string
	HasOverride      bool
}

type APIDocsService struct {
	settingRepo SettingRepository
}

func NewAPIDocsService(settingRepo SettingRepository) *APIDocsService {
	return &APIDocsService{settingRepo: settingRepo}
}

func (s *APIDocsService) GetDefaultContent() string {
	return defaultAPIDocsTemplate
}

func (s *APIDocsService) GetEffectiveContent(ctx context.Context) (string, error) {
	document, err := s.GetDocument(ctx)
	if err != nil {
		return "", err
	}
	return document.EffectiveContent, nil
}

func (s *APIDocsService) GetDocument(ctx context.Context) (*APIDocsDocument, error) {
	defaultContent := s.GetDefaultContent()
	defaultDoc := parseAPIDocsDocument(defaultContent)
	legacyOverride, hasLegacyOverride, err := s.loadOverride(ctx)
	if err != nil {
		return nil, err
	}

	pageOverrides, err := s.loadPageOverrides(ctx)
	if err != nil {
		return nil, err
	}

	if len(pageOverrides) == 0 {
		if hasLegacyOverride {
			return &APIDocsDocument{
				EffectiveContent: legacyOverride,
				DefaultContent:   defaultContent,
				HasOverride:      true,
			}, nil
		}
		return &APIDocsDocument{
			EffectiveContent: defaultContent,
			DefaultContent:   defaultContent,
			HasOverride:      false,
		}, nil
	}

	legacyDoc := parseAPIDocsDocument(legacyOverride)
	title := firstNonEmptyString(defaultDoc.Title, defaultAPIDocsTitleFallback)
	if hasLegacyOverride && strings.TrimSpace(legacyDoc.Title) != "" {
		title = legacyDoc.Title
	}

	pages := make(map[string]string, len(apiDocsPageOrder))
	hasDiff := false
	for _, pageID := range apiDocsPageOrder {
		pageBody := defaultDoc.Pages[pageID]
		if hasLegacyOverride {
			if legacyBody, ok := legacyDoc.Pages[pageID]; ok {
				pageBody = legacyBody
			}
		}
		if overrideSection, ok := pageOverrides[pageID]; ok {
			overrideDoc := parseAPIDocsDocument(overrideSection)
			if overrideBody, exists := overrideDoc.Pages[pageID]; exists {
				pageBody = overrideBody
			}
			if overrideTitle := strings.TrimSpace(overrideDoc.Title); overrideTitle != "" {
				title = overrideTitle
			}
		}
		pages[pageID] = pageBody
		if pageBody != defaultDoc.Pages[pageID] {
			hasDiff = true
		}
	}
	if !hasDiff && title == defaultDoc.Title {
		return &APIDocsDocument{
			EffectiveContent: defaultContent,
			DefaultContent:   defaultContent,
			HasOverride:      false,
		}, nil
	}

	return &APIDocsDocument{
		EffectiveContent: buildAPIDocsDocument(title, pages),
		DefaultContent:   defaultContent,
		HasOverride:      hasDiff || title != defaultDoc.Title,
	}, nil
}

func (s *APIDocsService) GetPageDocument(ctx context.Context, pageID string) (*APIDocsDocument, error) {
	normalizedPageID := normalizeAPIDocsPageID(pageID)
	if normalizedPageID == "" {
		return nil, ErrAPIDocsInvalidPage
	}

	defaultDoc := parseAPIDocsDocument(s.GetDefaultContent())
	defaultSection := buildAPIDocsPageSection(defaultDoc.Title, normalizedPageID, defaultDoc.Pages[normalizedPageID])

	pageOverrides, err := s.loadPageOverrides(ctx)
	if err != nil {
		return nil, err
	}

	if overrideSection, ok := pageOverrides[normalizedPageID]; ok {
		effectiveSection := normalizeAPIDocsPageSectionContent(normalizedPageID, defaultDoc.Title, overrideSection)
		if effectiveSection == "" {
			effectiveSection = defaultSection
		}
		return &APIDocsDocument{
			EffectiveContent: sectionOrDefault(effectiveSection, defaultSection),
			DefaultContent:   defaultSection,
			HasOverride:      effectiveSection != defaultSection,
		}, nil
	}

	legacyOverride, hasLegacyOverride, err := s.loadOverride(ctx)
	if err != nil {
		return nil, err
	}
	if hasLegacyOverride {
		legacyDoc := parseAPIDocsDocument(legacyOverride)
		if legacyBody, ok := legacyDoc.Pages[normalizedPageID]; ok {
			effectiveSection := buildAPIDocsPageSection(legacyDoc.Title, normalizedPageID, legacyBody)
			return &APIDocsDocument{
				EffectiveContent: effectiveSection,
				DefaultContent:   defaultSection,
				HasOverride:      effectiveSection != defaultSection,
			}, nil
		}
	}

	return &APIDocsDocument{
		EffectiveContent: defaultSection,
		DefaultContent:   defaultSection,
		HasOverride:      false,
	}, nil
}

func (s *APIDocsService) SaveOverride(ctx context.Context, content string) error {
	normalized := normalizeAPIDocsContent(content)
	if normalized == "" {
		return ErrAPIDocsEmptyContent
	}
	return s.settingRepo.Set(ctx, SettingKeyAPIDocsMarkdown, normalized)
}

func (s *APIDocsService) SavePageOverride(ctx context.Context, pageID string, content string) error {
	normalizedPageID := normalizeAPIDocsPageID(pageID)
	if normalizedPageID == "" {
		return ErrAPIDocsInvalidPage
	}

	defaultDoc := parseAPIDocsDocument(s.GetDefaultContent())
	normalized := normalizeAPIDocsPageSectionContent(normalizedPageID, defaultDoc.Title, content)
	if normalized == "" {
		return ErrAPIDocsEmptyContent
	}
	return s.settingRepo.Set(ctx, apiDocsPageSettingKey(normalizedPageID), normalized)
}

func (s *APIDocsService) ClearOverride(ctx context.Context) error {
	keys := append([]string{SettingKeyAPIDocsMarkdown}, apiDocsPageKeys()...)
	for _, key := range keys {
		if err := s.settingRepo.Delete(ctx, key); err != nil && !errors.Is(err, ErrSettingNotFound) {
			return err
		}
	}
	return nil
}

func (s *APIDocsService) ClearPageOverride(ctx context.Context, pageID string) error {
	normalizedPageID := normalizeAPIDocsPageID(pageID)
	if normalizedPageID == "" {
		return ErrAPIDocsInvalidPage
	}

	defaultDoc := parseAPIDocsDocument(s.GetDefaultContent())
	defaultSection := buildAPIDocsPageSection(defaultDoc.Title, normalizedPageID, defaultDoc.Pages[normalizedPageID])
	return s.settingRepo.Set(ctx, apiDocsPageSettingKey(normalizedPageID), defaultSection)
}

func (s *APIDocsService) loadOverride(ctx context.Context) (string, bool, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAPIDocsMarkdown)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return "", false, nil
		}
		return "", false, err
	}

	normalized := normalizeAPIDocsContent(value)
	if normalized == "" {
		return "", false, nil
	}
	return normalized, true, nil
}

func (s *APIDocsService) loadPageOverrides(ctx context.Context) (map[string]string, error) {
	overrides := make(map[string]string, len(apiDocsPageOrder))
	defaultDoc := parseAPIDocsDocument(s.GetDefaultContent())
	for _, pageID := range apiDocsPageOrder {
		value, err := s.settingRepo.GetValue(ctx, apiDocsPageSettingKey(pageID))
		if err != nil {
			if errors.Is(err, ErrSettingNotFound) {
				continue
			}
			return nil, err
		}
		normalized := normalizeAPIDocsPageSectionContent(pageID, defaultDoc.Title, value)
		if normalized == "" {
			continue
		}
		overrides[pageID] = normalized
	}
	return overrides, nil
}

func apiDocsPageKeys() []string {
	keys := make([]string, 0, len(apiDocsPageOrder))
	for _, pageID := range apiDocsPageOrder {
		keys = append(keys, apiDocsPageSettingKey(pageID))
	}
	return keys
}

func normalizeAPIDocsContent(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return ""
	}
	return normalized + "\n"
}

func sectionOrDefault(content string, fallback string) string {
	if strings.TrimSpace(content) == "" {
		return fallback
	}
	return content
}
