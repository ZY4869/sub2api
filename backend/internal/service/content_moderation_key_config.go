package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ContentModerationAPIKeysModeAppend  = "append"
	ContentModerationAPIKeysModeReplace = "replace"
)

type ContentModerationAPIKey struct {
	Key       string `json:"key"`
	Hash      string `json:"hash"`
	CreatedAt string `json:"created_at,omitempty"`
}

type ContentModerationAPIKeyUpdate struct {
	Existing     []ContentModerationAPIKey
	NewKeys      []string
	Mode         string
	DeleteHashes []string
	Now          time.Time
}

func NormalizeContentModerationAPIKeys(legacyKey, rawList string) []ContentModerationAPIKey {
	items := parseContentModerationAPIKeys(rawList)
	if strings.TrimSpace(legacyKey) != "" {
		items = append([]ContentModerationAPIKey{{Key: strings.TrimSpace(legacyKey)}}, items...)
	}
	return normalizeContentModerationAPIKeys(items, time.Time{})
}

func BuildContentModerationAPIKeyUpdate(input ContentModerationAPIKeyUpdate) []ContentModerationAPIKey {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = ContentModerationAPIKeysModeAppend
	}
	if mode != ContentModerationAPIKeysModeReplace {
		mode = ContentModerationAPIKeysModeAppend
	}

	base := input.Existing
	if mode == ContentModerationAPIKeysModeReplace {
		base = nil
	}

	deleted := make(map[string]struct{}, len(input.DeleteHashes))
	for _, hash := range input.DeleteHashes {
		hash = strings.TrimSpace(strings.ToLower(hash))
		if hash != "" {
			deleted[hash] = struct{}{}
		}
	}

	next := make([]ContentModerationAPIKey, 0, len(base)+len(input.NewKeys))
	for _, item := range base {
		hash := strings.TrimSpace(strings.ToLower(item.Hash))
		if hash == "" {
			hash = ContentModerationAPIKeyHash(item.Key)
		}
		if _, ok := deleted[hash]; ok {
			continue
		}
		next = append(next, item)
	}
	for _, key := range input.NewKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		next = append(next, ContentModerationAPIKey{
			Key:       key,
			CreatedAt: now.Format(time.RFC3339),
		})
	}
	return normalizeContentModerationAPIKeys(next, now)
}

func MarshalContentModerationAPIKeys(items []ContentModerationAPIKey) (string, error) {
	normalized := normalizeContentModerationAPIKeys(items, time.Time{})
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ContentModerationAPIKeyHash(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func NormalizeContentModerationKeywords(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		values = strings.FieldsFunc(raw, func(r rune) bool {
			switch r {
			case '\n', '\r', ',', '，', ';', '；':
				return true
			default:
				return false
			}
		})
	}
	return NormalizeContentModerationKeywordList(values)
}

func NormalizeContentModerationKeywordList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		keyword := strings.TrimSpace(value)
		if keyword == "" {
			continue
		}
		folded := strings.ToLower(keyword)
		if _, ok := seen[folded]; ok {
			continue
		}
		seen[folded] = struct{}{}
		out = append(out, keyword)
	}
	return out
}

func MarshalContentModerationKeywords(values []string) (string, error) {
	normalized := NormalizeContentModerationKeywordList(values)
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func EvaluateContentModerationKeywordBlock(settings *ContentModerationSettings, rawContent string) ContentModerationKeywordDecision {
	content := strings.TrimSpace(rawContent)
	if settings == nil || !settings.Enabled || !settings.KeywordBlockEnabled || content == "" {
		return ContentModerationKeywordDecision{Content: content}
	}
	keywords := NormalizeContentModerationKeywordList(settings.Keywords)
	if len(keywords) == 0 {
		return ContentModerationKeywordDecision{Content: content}
	}
	lowerContent := strings.ToLower(content)
	for _, keyword := range keywords {
		normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
		if normalizedKeyword == "" || !strings.Contains(lowerContent, normalizedKeyword) {
			continue
		}
		sum := sha256.Sum256([]byte(normalizedKeyword))
		return ContentModerationKeywordDecision{
			Blocked:     true,
			Content:     content,
			ErrorReason: fmt.Sprintf("keyword_blocked:%s", hex.EncodeToString(sum[:])[:12]),
		}
	}
	return ContentModerationKeywordDecision{Content: content}
}

func parseContentModerationAPIKeys(raw string) []ContentModerationAPIKey {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var items []ContentModerationAPIKey
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items
	}
	var stringsOnly []string
	if err := json.Unmarshal([]byte(raw), &stringsOnly); err != nil {
		return nil
	}
	out := make([]ContentModerationAPIKey, 0, len(stringsOnly))
	for _, key := range stringsOnly {
		out = append(out, ContentModerationAPIKey{Key: key})
	}
	return out
}

func normalizeContentModerationAPIKeys(items []ContentModerationAPIKey, now time.Time) []ContentModerationAPIKey {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]ContentModerationAPIKey, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		hash := strings.TrimSpace(strings.ToLower(item.Hash))
		if hash == "" {
			hash = ContentModerationAPIKeyHash(key)
		}
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}
		createdAt := strings.TrimSpace(item.CreatedAt)
		if createdAt == "" {
			createdAt = now.UTC().Format(time.RFC3339)
		}
		out = append(out, ContentModerationAPIKey{
			Key:       key,
			Hash:      hash,
			CreatedAt: createdAt,
		})
	}
	return out
}
