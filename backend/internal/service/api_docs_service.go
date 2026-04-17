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
	override, hasOverride, err := s.loadOverride(ctx)
	if err != nil {
		return "", err
	}
	if hasOverride {
		return override, nil
	}
	return s.GetDefaultContent(), nil
}

func (s *APIDocsService) GetDocument(ctx context.Context) (*APIDocsDocument, error) {
	defaultContent := s.GetDefaultContent()
	override, hasOverride, err := s.loadOverride(ctx)
	if err != nil {
		return nil, err
	}

	document := &APIDocsDocument{
		DefaultContent: defaultContent,
		HasOverride:    hasOverride,
	}
	if hasOverride {
		document.EffectiveContent = override
	} else {
		document.EffectiveContent = defaultContent
	}
	return document, nil
}

func (s *APIDocsService) SaveOverride(ctx context.Context, content string) error {
	normalized := normalizeAPIDocsContent(content)
	if normalized == "" {
		return ErrAPIDocsEmptyContent
	}
	return s.settingRepo.Set(ctx, SettingKeyAPIDocsMarkdown, normalized)
}

func (s *APIDocsService) ClearOverride(ctx context.Context) error {
	if err := s.settingRepo.Delete(ctx, SettingKeyAPIDocsMarkdown); err != nil && !errors.Is(err, ErrSettingNotFound) {
		return err
	}
	return nil
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

func normalizeAPIDocsContent(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return ""
	}
	return normalized + "\n"
}
