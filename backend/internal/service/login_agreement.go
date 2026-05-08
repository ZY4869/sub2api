package service

import (
	"encoding/json"
	"strings"
)

const LoginAgreementModeCheckbox = "checkbox"

type LoginAgreementDocument struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	PageSlug string `json:"page_slug"`
}

func NormalizeLoginAgreementMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case LoginAgreementModeCheckbox:
		return LoginAgreementModeCheckbox
	default:
		return LoginAgreementModeCheckbox
	}
}

func NormalizeLoginAgreementDocuments(items []LoginAgreementDocument) []LoginAgreementDocument {
	if len(items) == 0 {
		return []LoginAgreementDocument{}
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		doc := LoginAgreementDocument{
			ID:       strings.TrimSpace(item.ID),
			Title:    strings.TrimSpace(item.Title),
			PageSlug: NormalizeCustomPageSlugForAdmin(item.PageSlug),
		}
		if doc.PageSlug == "" {
			continue
		}
		if doc.ID == "" {
			doc.ID = doc.PageSlug
		}
		if doc.Title == "" {
			doc.Title = doc.PageSlug
		}
		if _, ok := seen[doc.PageSlug]; ok {
			continue
		}
		seen[doc.PageSlug] = struct{}{}
		out = append(out, doc)
		if len(out) >= 10 {
			break
		}
	}
	return out
}

func ParseLoginAgreementDocuments(raw string) []LoginAgreementDocument {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []LoginAgreementDocument{}
	}
	var items []LoginAgreementDocument
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []LoginAgreementDocument{}
	}
	return NormalizeLoginAgreementDocuments(items)
}

func MarshalLoginAgreementDocuments(items []LoginAgreementDocument) (string, error) {
	normalized := NormalizeLoginAgreementDocuments(items)
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
