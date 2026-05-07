package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const (
	ContentModerationSourceAnthropicMessages   = "anthropic_messages"
	ContentModerationSourceOpenAIResponses     = "openai_responses"
	ContentModerationSourceOpenAIChat          = "openai_chat_completions"
	ContentModerationSourceOpenAIMessages      = "openai_messages"
	ContentModerationSourceGeminiGenerate      = "gemini_generate_content"
	ContentModerationSourceGeminiOpenAICompat  = "gemini_openai_chat_completions"
	contentModerationDefaultTimeoutMs          = 1500
	contentModerationDefaultDedupeWindowSecond = 300
	contentModerationSummaryHashPrefixLen      = 12
)

var ErrContentModerationAuditNotFound = infraerrors.NotFound("CONTENT_MODERATION_AUDIT_NOT_FOUND", "content moderation audit not found")

type ContentModerationAudit struct {
	ID              int64
	RequestID       string
	ClientRequestID string
	UserID          *int64
	APIKeyID        *int64
	Provider        string
	Model           string
	SourceEndpoint  string
	ContentHash     string
	ContentSummary  string
	Hit             bool
	DedupeHit       bool
	ErrorReason     string
	LatencyMs       int
	CreatedAt       time.Time
}

type ContentModerationAuditList struct {
	Items    []*ContentModerationAudit
	Total    int64
	Page     int
	PageSize int
}

type ContentModerationAuditFilter struct {
	Page           int
	PageSize       int
	RequestID      string
	ClientRequestID string
	Provider       string
	Model          string
	SourceEndpoint string
	ContentHash    string
	Hit            *bool
	UserID         *int64
}

func (f *ContentModerationAuditFilter) Normalize() (int, int) {
	if f == nil {
		return 1, 20
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

type ContentModerationSettings struct {
	Enabled             bool
	Provider            string
	BaseURL             string
	APIKey              string
	Model               string
	TimeoutMs           int
	DedupeWindowSeconds int
	FailOpen            bool
}

type ContentModerationRecordInput struct {
	SourceEndpoint string
	Provider       string
	Model          string
	Content        string
	RequestID      string
	ClientRequestID string
	UserID         *int64
	APIKeyID       *int64
}

type ContentModerationService struct {
	repo        ContentModerationAuditRepository
	settingRepo SettingRepository
}

type ContentModerationAuditRepository interface {
	CreateContentModerationAudit(ctx context.Context, audit *ContentModerationAudit) error
	FindRecentContentModerationAuditByHash(ctx context.Context, contentHash string, since time.Time) (*ContentModerationAudit, error)
	ListContentModerationAudits(ctx context.Context, filter *ContentModerationAuditFilter) (*ContentModerationAuditList, error)
	GetContentModerationAuditByID(ctx context.Context, id int64) (*ContentModerationAudit, error)
}

func NewContentModerationService(repo ContentModerationAuditRepository, settingRepo SettingRepository) *ContentModerationService {
	return &ContentModerationService{repo: repo, settingRepo: settingRepo}
}

func (s *ContentModerationService) GetSettings(ctx context.Context) (*ContentModerationSettings, error) {
	if s == nil || s.settingRepo == nil {
		return &ContentModerationSettings{FailOpen: true, TimeoutMs: contentModerationDefaultTimeoutMs, DedupeWindowSeconds: contentModerationDefaultDedupeWindowSecond}, nil
	}
	keys := []string{
		SettingKeyContentModerationEnabled,
		SettingKeyContentModerationProvider,
		SettingKeyContentModerationBaseURL,
		SettingKeyContentModerationAPIKey,
		SettingKeyContentModerationModel,
		SettingKeyContentModerationTimeoutMs,
		SettingKeyContentModerationDedupeWindowSeconds,
		SettingKeyContentModerationFailOpen,
	}
	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, err
	}
	settings := &ContentModerationSettings{
		Enabled:             values[SettingKeyContentModerationEnabled] == "true",
		Provider:            strings.TrimSpace(values[SettingKeyContentModerationProvider]),
		BaseURL:             strings.TrimSpace(values[SettingKeyContentModerationBaseURL]),
		APIKey:              strings.TrimSpace(values[SettingKeyContentModerationAPIKey]),
		Model:               strings.TrimSpace(values[SettingKeyContentModerationModel]),
		TimeoutMs:           parseIntWithDefault(values[SettingKeyContentModerationTimeoutMs], contentModerationDefaultTimeoutMs),
		DedupeWindowSeconds: parseIntWithDefault(values[SettingKeyContentModerationDedupeWindowSeconds], contentModerationDefaultDedupeWindowSecond),
		FailOpen:            values[SettingKeyContentModerationFailOpen] != "false",
	}
	if settings.TimeoutMs <= 0 {
		settings.TimeoutMs = contentModerationDefaultTimeoutMs
	}
	if settings.DedupeWindowSeconds < 0 {
		settings.DedupeWindowSeconds = 0
	}
	if settings.Provider == "" {
		settings.Provider = "openai"
	}
	return settings, nil
}

func (s *ContentModerationService) RecordAudit(ctx context.Context, input *ContentModerationRecordInput) {
	if s == nil || s.repo == nil || input == nil {
		return
	}
	settings, err := s.GetSettings(ctx)
	if err != nil || settings == nil || !settings.Enabled {
		return
	}

	normalizedContent := strings.TrimSpace(input.Content)
	if normalizedContent == "" {
		return
	}

	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		if v, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(v) != "" {
			requestID = strings.TrimSpace(v)
		}
	}
	clientRequestID := strings.TrimSpace(input.ClientRequestID)
	if clientRequestID == "" {
		if v, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(v) != "" {
			clientRequestID = strings.TrimSpace(v)
		}
	}

	hash := sha256.Sum256([]byte(normalizedContent))
	contentHash := hex.EncodeToString(hash[:])
	now := time.Now().UTC()

	result := &ContentModerationAudit{
		RequestID:       requestID,
		ClientRequestID: clientRequestID,
		UserID:          input.UserID,
		APIKeyID:        input.APIKeyID,
		Provider:        strings.TrimSpace(firstNonEmptyString(input.Provider, settings.Provider)),
		Model:           strings.TrimSpace(firstNonEmptyString(input.Model, settings.Model)),
		SourceEndpoint:  strings.TrimSpace(input.SourceEndpoint),
		ContentHash:     contentHash,
		ContentSummary:  summarizeModerationContent(normalizedContent),
		Hit:             false,
		DedupeHit:       false,
		ErrorReason:     "",
		LatencyMs:       0,
		CreatedAt:       now,
	}

	if settings.DedupeWindowSeconds > 0 {
		since := now.Add(-time.Duration(settings.DedupeWindowSeconds) * time.Second)
		if previous, findErr := s.repo.FindRecentContentModerationAuditByHash(ctx, contentHash, since); findErr == nil && previous != nil {
			result.Hit = previous.Hit
			result.DedupeHit = true
			result.ErrorReason = strings.TrimSpace(previous.ErrorReason)
		}
	}

	start := time.Now()
	if !result.DedupeHit {
		evaluated := evaluateContentModeration(ctx, settings, input, normalizedContent)
		result.Hit = evaluated.Hit
		result.ErrorReason = strings.TrimSpace(evaluated.ErrorReason)
	}
	result.LatencyMs = int(time.Since(start).Milliseconds())

	_ = s.repo.CreateContentModerationAudit(ctx, result)
}

func (s *ContentModerationService) ListAudits(ctx context.Context, filter *ContentModerationAuditFilter) (*ContentModerationAuditList, error) {
	if s == nil || s.repo == nil {
		return &ContentModerationAuditList{Items: []*ContentModerationAudit{}, Total: 0, Page: 1, PageSize: 20}, nil
	}
	return s.repo.ListContentModerationAudits(ctx, filter)
}

func (s *ContentModerationService) GetAuditByID(ctx context.Context, id int64) (*ContentModerationAudit, error) {
	if s == nil || s.repo == nil {
		return nil, ErrContentModerationAuditNotFound
	}
	return s.repo.GetContentModerationAuditByID(ctx, id)
}

func ExtractModerationTextFromJSONBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	var parts []string
	walkModerationValue(payload, &parts)
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func walkModerationValue(value any, parts *[]string) {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			switch key {
			case "text", "content", "input", "instructions", "system", "prompt":
				walkModerationValue(item, parts)
			case "messages", "parts", "contents":
				walkModerationValue(item, parts)
			default:
				if nested, ok := item.(map[string]any); ok {
					if itemType, _ := nested["type"].(string); itemType == "text" || itemType == "input_text" || itemType == "output_text" {
						walkModerationValue(nested["text"], parts)
					}
				}
			}
		}
	case []any:
		for _, item := range v {
			walkModerationValue(item, parts)
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			*parts = append(*parts, trimmed)
		}
	}
}

func summarizeModerationContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	normalized := strings.Join(strings.Fields(content), " ")
	if normalized == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(normalized))
	hashPrefix := hex.EncodeToString(hash[:])
	if len(hashPrefix) > contentModerationSummaryHashPrefixLen {
		hashPrefix = hashPrefix[:contentModerationSummaryHashPrefixLen]
	}
	wordCount := len(strings.Fields(normalized))
	charCount := len([]rune(normalized))
	return fmt.Sprintf("redacted text (%d chars, %d words) #%s", charCount, wordCount, hashPrefix)
}

func parseIntWithDefault(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil {
		return fallback
	}
	return value
}
