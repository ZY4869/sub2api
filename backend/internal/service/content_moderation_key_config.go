package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

const (
	ContentModerationAPIKeysModeAppend  = "append"
	ContentModerationAPIKeysModeReplace = "replace"
	contentModerationMaxKeywords        = 500
	contentModerationMaxKeywordRunes    = 120
	contentModerationMaxCyberCategories = 20
)

var knownContentModerationCategories = []string{
	"hate",
	"hate/threatening",
	"harassment",
	"harassment/threatening",
	"self-harm",
	"self-harm/intent",
	"self-harm/instructions",
	"sexual",
	"sexual/minors",
	"violence",
	"violence/graphic",
	"illicit",
	"illicit/violent",
}

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

type ContentModerationCyberCategory struct {
	ID       string   `json:"id"`
	Keywords []string `json:"keywords"`
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
		runes := []rune(keyword)
		if len(runes) > contentModerationMaxKeywordRunes {
			keyword = string(runes[:contentModerationMaxKeywordRunes])
		}
		folded := normalizeContentModerationKeywordComparable(keyword)
		if folded == "" {
			continue
		}
		if _, ok := seen[folded]; ok {
			continue
		}
		seen[folded] = struct{}{}
		out = append(out, keyword)
		if len(out) >= contentModerationMaxKeywords {
			break
		}
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

func DefaultContentModerationCategoryThresholds() map[string]float64 {
	out := make(map[string]float64, len(knownContentModerationCategories))
	for _, category := range knownContentModerationCategories {
		out[category] = 1.0
	}
	return out
}

func DefaultContentModerationCyberCategories() []ContentModerationCyberCategory {
	return []ContentModerationCyberCategory{
		{ID: "credential_theft", Keywords: []string{"credential stuffing", "steal api key", "phishing kit"}},
		{ID: "malware", Keywords: []string{"malware payload", "ransomware", "keylogger"}},
		{ID: "exploit", Keywords: []string{"zero day exploit", "privilege escalation", "remote code execution"}},
		{ID: "exfiltration", Keywords: []string{"data exfiltration", "dump database", "bypass dlp"}},
	}
}

func NormalizeContentModerationCyberCategories(raw string) []ContentModerationCyberCategory {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultContentModerationCyberCategories()
	}
	if raw == "[]" {
		return []ContentModerationCyberCategory{}
	}
	var parsed []ContentModerationCyberCategory
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return DefaultContentModerationCyberCategories()
	}
	return NormalizeContentModerationCyberCategoryList(parsed)
}

func MarshalContentModerationCyberCategories(values []ContentModerationCyberCategory) (string, error) {
	normalized := NormalizeContentModerationCyberCategoryList(values)
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func NormalizeContentModerationCyberCategoryList(values []ContentModerationCyberCategory) []ContentModerationCyberCategory {
	if len(values) == 0 {
		return []ContentModerationCyberCategory{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]ContentModerationCyberCategory, 0, len(values))
	for _, item := range values {
		id := normalizeCyberPolicyCategoryID(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		keywords := NormalizeContentModerationKeywordList(item.Keywords)
		if len(keywords) == 0 {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, ContentModerationCyberCategory{ID: id, Keywords: keywords})
		if len(out) >= contentModerationMaxCyberCategories {
			break
		}
	}
	return out
}

func normalizeCyberPolicyCategoryID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			_, _ = b.WriteRune(r)
		case r == '-' || r == '_' || unicode.IsSpace(r):
			_ = b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func EvaluateContentModerationCyberPolicy(settings *ContentModerationSettings, rawContent string) ContentModerationKeywordDecision {
	content := strings.TrimSpace(rawContent)
	if settings == nil || !settings.Enabled || !settings.CyberPolicyEnabled || content == "" {
		return ContentModerationKeywordDecision{Content: content}
	}
	normalizedContent := normalizeContentModerationKeywordComparable(content)
	for _, category := range NormalizeContentModerationCyberCategoryList(settings.CyberCategories) {
		for _, keyword := range NormalizeContentModerationKeywordList(category.Keywords) {
			if normalizedKeyword := normalizeContentModerationKeywordComparable(keyword); normalizedKeyword != "" && strings.Contains(normalizedContent, normalizedKeyword) {
				reason := "cyber_policy:" + category.ID
				return ContentModerationKeywordDecision{
					Blocked:     true,
					Content:     content,
					ErrorReason: reason,
					Categories:  []string{reason},
				}
			}
		}
	}
	return ContentModerationKeywordDecision{Content: content}
}

func NormalizeContentModerationCategoryThresholds(raw string) map[string]float64 {
	thresholds := DefaultContentModerationCategoryThresholds()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return thresholds
	}
	var parsed map[string]float64
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return thresholds
	}
	for key, value := range parsed {
		category := normalizeContentModerationCategory(key)
		if category == "" || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		thresholds[category] = value
	}
	return thresholds
}

func ValidateContentModerationCategoryThresholds(input map[string]float64) (map[string]float64, error) {
	out := DefaultContentModerationCategoryThresholds()
	for key, value := range input {
		category := normalizeContentModerationCategory(key)
		if category == "" {
			continue
		}
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
			return nil, fmt.Errorf("content moderation category threshold for %s must be between 0 and 1", category)
		}
		out[category] = value
	}
	return out, nil
}

func MarshalContentModerationCategoryThresholds(values map[string]float64) (string, error) {
	normalized := DefaultContentModerationCategoryThresholds()
	for key, value := range values {
		category := normalizeContentModerationCategory(key)
		if category == "" || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		normalized[category] = value
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func evaluateContentModerationCategoryThresholds(scores map[string]float64, thresholds map[string]float64) (bool, string) {
	if len(scores) == 0 {
		return false, ""
	}
	if len(thresholds) == 0 {
		thresholds = DefaultContentModerationCategoryThresholds()
	}
	for rawCategory, score := range scores {
		category := normalizeContentModerationCategory(rawCategory)
		if category == "" || math.IsNaN(score) || math.IsInf(score, 0) {
			continue
		}
		threshold, ok := thresholds[category]
		if !ok {
			continue
		}
		if score >= threshold {
			protocolruntime.RecordContentModerationThresholdHit(category)
			return true, "moderation_threshold:" + category
		}
	}
	return false, ""
}

func moderationCategoriesForReason(reason string) []string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil
	}
	switch {
	case strings.HasPrefix(reason, "keyword_blocked:"):
		return []string{"keyword_blocked"}
	case strings.HasPrefix(reason, "moderation_threshold:"):
		category := strings.TrimSpace(strings.TrimPrefix(reason, "moderation_threshold:"))
		if category == "" {
			return []string{"moderation_threshold"}
		}
		return []string{"moderation_threshold:" + category}
	case reason == ContentModerationReasonModerationFlagged:
		return []string{ContentModerationReasonModerationFlagged}
	case reason == ContentModerationReasonModerationUnavailable:
		return []string{ContentModerationReasonModerationUnavailable}
	default:
		if strings.HasPrefix(reason, "moderation_") {
			return []string{ContentModerationReasonModerationUnavailable}
		}
		return nil
	}
}

func normalizeModerationCategories(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		category := strings.Join(strings.Fields(strings.TrimSpace(value)), "_")
		if category == "" {
			continue
		}
		if len(category) > 120 {
			category = category[:120]
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		out = append(out, category)
	}
	return out
}

func normalizeContentModerationCategory(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	for _, category := range knownContentModerationCategories {
		if value == category {
			return category
		}
	}
	return ""
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
	normalizedContent := normalizeContentModerationKeywordComparable(content)
	for _, keyword := range keywords {
		normalizedKeyword := normalizeContentModerationKeywordComparable(keyword)
		if normalizedKeyword == "" || !strings.Contains(normalizedContent, normalizedKeyword) {
			continue
		}
		sum := sha256.Sum256([]byte(normalizedKeyword))
		return ContentModerationKeywordDecision{
			Blocked:     true,
			Content:     content,
			ErrorReason: fmt.Sprintf("keyword_blocked:%s", hex.EncodeToString(sum[:])[:12]),
			Categories:  []string{"keyword_blocked"},
		}
	}
	return ContentModerationKeywordDecision{Content: content}
}

func normalizeContentModerationKeywordComparable(value string) string {
	var b strings.Builder
	var lastSpace bool
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		r = normalizeFullWidthASCII(r)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			_, _ = b.WriteRune(r)
			lastSpace = false
			continue
		}
		if unicode.IsSpace(r) || strings.ContainsRune("-_./\\|,，;；:：'\"`~!@#$%^&*()[]{}<>?？、", r) {
			if !lastSpace && b.Len() > 0 {
				_ = b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		_, _ = b.WriteRune(r)
		lastSpace = false
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func normalizeFullWidthASCII(r rune) rune {
	if r >= '！' && r <= '～' {
		return r - 0xFEE0
	}
	if r == '　' {
		return ' '
	}
	return r
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
