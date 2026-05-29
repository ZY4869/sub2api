package service

import (
	"context"
	"encoding/json"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"net/url"
	"strings"
)

func (s *SettingService) GetFrameSrcOrigins(ctx context.Context) ([]string, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var origins []string
	addOrigin := func(rawURL string) {
		if origin := extractOriginFromURL(rawURL); origin != "" {
			if _, ok := seen[origin]; !ok {
				seen[origin] = struct{}{}
				origins = append(origins, origin)
			}
		}
	}
	if settings.PurchaseSubscriptionEnabled {
		addOrigin(settings.PurchaseSubscriptionURL)
	}
	for _, item := range parseCustomMenuItemURLs(settings.CustomMenuItems) {
		addOrigin(item)
	}
	return origins, nil
}

type CustomPageContent struct {
	ID         string
	Slug       string
	Label      string
	Visibility string
	PageMode   string
	Content    string
}

func (s *SettingService) GetCustomPageBySlug(ctx context.Context, slug string) (*CustomPageContent, error) {
	settings, err := s.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}

	slug = normalizeCustomPageSlug(slug)
	if slug == "" {
		return nil, infraerrors.NotFound("CUSTOM_PAGE_NOT_FOUND", "custom page not found")
	}

	type menuItem struct {
		ID            string `json:"id"`
		Label         string `json:"label"`
		Visibility    string `json:"visibility"`
		PageMode      string `json:"page_mode"`
		PageSlug      string `json:"page_slug"`
		PageContent   string `json:"page_content"`
		PagePublished bool   `json:"page_published"`
	}

	var items []menuItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(settings.CustomMenuItems)), &items); err != nil {
		return nil, infraerrors.NotFound("CUSTOM_PAGE_NOT_FOUND", "custom page not found")
	}

	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.PageMode), "markdown") {
			continue
		}
		if !item.PagePublished {
			continue
		}
		if normalizeCustomPageSlug(item.PageSlug) != slug {
			continue
		}
		return &CustomPageContent{
			ID:         strings.TrimSpace(item.ID),
			Slug:       slug,
			Label:      strings.TrimSpace(item.Label),
			Visibility: normalizeMenuVisibility(item.Visibility),
			PageMode:   "markdown",
			Content:    sanitizeCustomPageContent(item.PageContent),
		}, nil
	}

	return nil, infraerrors.NotFound("CUSTOM_PAGE_NOT_FOUND", "custom page not found")
}

func normalizeCustomPageSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			_, _ = b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			_, _ = b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '/':
			if b.Len() == 0 || lastDash {
				continue
			}
			_ = b.WriteByte('-')
			lastDash = true
		default:
			continue
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" || len(out) > 64 {
		return ""
	}
	return out
}

func NormalizeCustomPageSlugForAdmin(value string) string {
	return normalizeCustomPageSlug(value)
}

func normalizeMenuVisibility(value string) string {
	if strings.TrimSpace(strings.ToLower(value)) == "admin" {
		return "admin"
	}
	return "user"
}

func sanitizeCustomPageContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if len(content) > 128*1024 {
		content = content[:128*1024]
	}
	return content
}

func extractOriginFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
