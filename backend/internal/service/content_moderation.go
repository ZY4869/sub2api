package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ContentModerationSourceAnthropicMessages   = "anthropic_messages"
	ContentModerationSourceOpenAIResponses     = "openai_responses"
	ContentModerationSourceOpenAIChat          = "openai_chat_completions"
	ContentModerationSourceOpenAIMessages      = "openai_messages"
	ContentModerationSourceGeminiGenerate      = "gemini_generate_content"
	ContentModerationSourceGeminiOpenAICompat  = "gemini_openai_chat_completions"
	ContentModerationModelFilterAll            = "all"
	ContentModerationModelFilterInclude        = "include"
	ContentModerationModelFilterExclude        = "exclude"
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
	Page            int
	PageSize        int
	RequestID       string
	ClientRequestID string
	Provider        string
	Model           string
	SourceEndpoint  string
	ContentHash     string
	Hit             *bool
	UserID          *int64
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
	APIKeys             []ContentModerationAPIKey
	Model               string
	TimeoutMs           int
	DedupeWindowSeconds int
	FailOpen            bool
	KeywordBlockEnabled bool
	Keywords            []string
	ModelFilter         ContentModerationModelFilter
	CategoryThresholds  map[string]float64
}

type ContentModerationModelFilter struct {
	Type   string   `json:"type"`
	Models []string `json:"models"`
}

type ContentModerationRecordInput struct {
	SourceEndpoint  string
	Provider        string
	Model           string
	Content         string
	RequestID       string
	ClientRequestID string
	UserID          *int64
	APIKeyID        *int64
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
		SettingKeyContentModerationAPIKeys,
		SettingKeyContentModerationModel,
		SettingKeyContentModerationTimeoutMs,
		SettingKeyContentModerationDedupeWindowSeconds,
		SettingKeyContentModerationFailOpen,
		SettingKeyContentModerationKeywordBlockEnabled,
		SettingKeyContentModerationKeywords,
		SettingKeyContentModerationModelFilter,
		SettingKeyContentModerationCategoryThresholds,
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
		APIKeys:             NormalizeContentModerationAPIKeys(values[SettingKeyContentModerationAPIKey], values[SettingKeyContentModerationAPIKeys]),
		Model:               strings.TrimSpace(values[SettingKeyContentModerationModel]),
		TimeoutMs:           parseIntWithDefault(values[SettingKeyContentModerationTimeoutMs], contentModerationDefaultTimeoutMs),
		DedupeWindowSeconds: parseIntWithDefault(values[SettingKeyContentModerationDedupeWindowSeconds], contentModerationDefaultDedupeWindowSecond),
		FailOpen:            values[SettingKeyContentModerationFailOpen] != "false",
		KeywordBlockEnabled: values[SettingKeyContentModerationKeywordBlockEnabled] == "true",
		Keywords:            NormalizeContentModerationKeywords(values[SettingKeyContentModerationKeywords]),
		ModelFilter:         NormalizeContentModerationModelFilter(values[SettingKeyContentModerationModelFilter]),
		CategoryThresholds:  NormalizeContentModerationCategoryThresholds(values[SettingKeyContentModerationCategoryThresholds]),
	}
	if len(settings.CategoryThresholds) == 0 {
		settings.CategoryThresholds = DefaultContentModerationCategoryThresholds()
	}
	if settings.APIKey == "" && len(settings.APIKeys) > 0 {
		settings.APIKey = settings.APIKeys[0].Key
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

type ContentModerationKeywordDecision struct {
	Blocked     bool
	Content     string
	ErrorReason string
}

func (s *ContentModerationService) CheckKeywordBlock(ctx context.Context, input *ContentModerationRecordInput) (*ContentModerationKeywordDecision, error) {
	if s == nil || s.repo == nil || input == nil {
		return &ContentModerationKeywordDecision{}, nil
	}
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	decision := EvaluateContentModerationKeywordBlock(settings, input.Content)
	if decision.Blocked {
		recordInput := *input
		recordInput.Content = decision.Content
		s.recordAuditWithSettings(ctx, settings, &recordInput, decision.ErrorReason)
	}
	return &decision, nil
}

func (s *ContentModerationService) CheckBlock(ctx context.Context, input *ContentModerationRecordInput) (*ContentModerationKeywordDecision, error) {
	if s == nil || input == nil {
		return &ContentModerationKeywordDecision{}, nil
	}
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil || !settings.Enabled || !settings.ModelFilter.Allows(input.Model) {
		return &ContentModerationKeywordDecision{Content: strings.TrimSpace(input.Content)}, nil
	}
	decision := EvaluateContentModerationKeywordBlock(settings, input.Content)
	if decision.Blocked {
		if s.repo != nil {
			recordInput := *input
			recordInput.Content = decision.Content
			s.recordAuditWithSettings(ctx, settings, &recordInput, decision.ErrorReason)
		}
		return &decision, nil
	}
	evaluated := evaluateContentModeration(ctx, settings, input, decision.Content)
	if !evaluated.Hit {
		return &decision, nil
	}
	decision.Blocked = true
	decision.ErrorReason = strings.TrimSpace(evaluated.ErrorReason)
	if decision.ErrorReason == "" {
		decision.ErrorReason = "moderation_flagged"
	}
	if s.repo != nil {
		recordInput := *input
		recordInput.Content = decision.Content
		s.recordAuditWithSettings(ctx, settings, &recordInput, decision.ErrorReason)
	}
	return &decision, nil
}

func (s *ContentModerationService) RecordAudit(ctx context.Context, input *ContentModerationRecordInput) {
	if s == nil || s.repo == nil || input == nil {
		return
	}
	settings, err := s.GetSettings(ctx)
	if err != nil || settings == nil || !settings.Enabled {
		return
	}
	if !settings.ModelFilter.Allows(input.Model) {
		return
	}
	s.recordAuditWithSettings(ctx, settings, input, "")
}

func (s *ContentModerationService) recordAuditWithSettings(
	ctx context.Context,
	settings *ContentModerationSettings,
	input *ContentModerationRecordInput,
	forcedErrorReason string,
) {
	if s == nil || s.repo == nil || settings == nil || input == nil {
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

	keywordResult := EvaluateContentModerationKeywordBlock(settings, normalizedContent)
	if forcedErrorReason == "" && !keywordResult.Blocked && settings.DedupeWindowSeconds > 0 {
		since := now.Add(-time.Duration(settings.DedupeWindowSeconds) * time.Second)
		if previous, findErr := s.repo.FindRecentContentModerationAuditByHash(ctx, contentHash, since); findErr == nil && previous != nil {
			result.Hit = previous.Hit
			result.DedupeHit = true
			result.ErrorReason = strings.TrimSpace(previous.ErrorReason)
		}
	}

	start := time.Now()
	if !result.DedupeHit {
		if forcedErrorReason != "" {
			result.Hit = true
			result.ErrorReason = strings.TrimSpace(forcedErrorReason)
		} else if keywordResult.Blocked {
			result.Hit = true
			result.ErrorReason = keywordResult.ErrorReason
		} else {
			evaluated := evaluateContentModeration(ctx, settings, input, normalizedContent)
			result.Hit = evaluated.Hit
			result.ErrorReason = strings.TrimSpace(evaluated.ErrorReason)
		}
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
	state := &moderationExtractionState{}
	walkModerationValue(payload, state)
	return strings.TrimSpace(redactContentModerationSecrets(strings.Join(state.parts, "\n")))
}

type moderationExtractionState struct {
	parts      []string
	seen       map[string]struct{}
	imageCount int
}

func (s *moderationExtractionState) appendText(value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	key := strings.Join(strings.Fields(trimmed), " ")
	if s.seen == nil {
		s.seen = make(map[string]struct{})
	}
	if _, ok := s.seen[key]; ok {
		return
	}
	s.seen[key] = struct{}{}
	s.parts = append(s.parts, trimmed)
}

func (s *moderationExtractionState) appendImageMarker() {
	if s.imageCount > 0 {
		return
	}
	s.imageCount++
	s.parts = append(s.parts, "[redacted-image]")
}

func walkModerationValue(value any, state *moderationExtractionState) {
	switch v := value.(type) {
	case map[string]any:
		if isModerationImagePart(v) {
			state.appendImageMarker()
			return
		}
		if isModerationTextPart(v) {
			walkModerationValue(v["text"], state)
			return
		}
		for key, item := range v {
			switch key {
			case "text", "content", "input", "instructions", "system", "prompt":
				walkModerationValue(item, state)
			case "messages", "parts", "contents":
				walkModerationValue(item, state)
			default:
				if nested, ok := item.(map[string]any); ok {
					if itemType, _ := nested["type"].(string); itemType == "text" || itemType == "input_text" || itemType == "output_text" {
						walkModerationValue(nested["text"], state)
					}
				}
			}
		}
	case []any:
		for _, item := range v {
			walkModerationValue(item, state)
		}
	case string:
		state.appendText(v)
	}
}

func isModerationTextPart(value map[string]any) bool {
	itemType, _ := value["type"].(string)
	switch strings.ToLower(strings.TrimSpace(itemType)) {
	case "text", "input_text", "output_text":
		return true
	default:
		return false
	}
}

func isModerationImagePart(value map[string]any) bool {
	itemType, _ := value["type"].(string)
	switch strings.ToLower(strings.TrimSpace(itemType)) {
	case "image", "input_image", "image_url":
		return true
	}
	if _, ok := value["image_url"]; ok {
		return true
	}
	if _, ok := value["inline_data"]; ok {
		return true
	}
	if _, ok := value["file_data"]; ok {
		return true
	}
	return false
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

func NormalizeContentModerationModelFilter(raw string) ContentModerationModelFilter {
	filter := ContentModerationModelFilter{Type: ContentModerationModelFilterAll}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return filter
	}
	if err := json.Unmarshal([]byte(raw), &filter); err != nil {
		return ContentModerationModelFilter{Type: ContentModerationModelFilterAll}
	}
	filter.Type = strings.ToLower(strings.TrimSpace(filter.Type))
	switch filter.Type {
	case ContentModerationModelFilterInclude, ContentModerationModelFilterExclude:
	default:
		filter.Type = ContentModerationModelFilterAll
	}
	filter.Models = normalizeContentModerationModelIDs(filter.Models)
	if filter.Type != ContentModerationModelFilterAll && len(filter.Models) == 0 {
		filter.Type = ContentModerationModelFilterAll
	}
	return filter
}

func MarshalContentModerationModelFilter(filter ContentModerationModelFilter) (string, error) {
	normalized := ContentModerationModelFilter{
		Type:   strings.ToLower(strings.TrimSpace(filter.Type)),
		Models: normalizeContentModerationModelIDs(filter.Models),
	}
	switch normalized.Type {
	case ContentModerationModelFilterInclude, ContentModerationModelFilterExclude:
	default:
		normalized.Type = ContentModerationModelFilterAll
		normalized.Models = []string{}
	}
	if normalized.Type != ContentModerationModelFilterAll && len(normalized.Models) == 0 {
		normalized.Type = ContentModerationModelFilterAll
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f ContentModerationModelFilter) Allows(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	switch f.Type {
	case ContentModerationModelFilterInclude:
		return normalized != "" && containsNormalizedModerationModel(f.Models, normalized)
	case ContentModerationModelFilterExclude:
		return normalized == "" || !containsNormalizedModerationModel(f.Models, normalized)
	default:
		return true
	}
}

func normalizeContentModerationModelIDs(models []string) []string {
	if len(models) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, item := range models {
		model := strings.TrimSpace(item)
		key := strings.ToLower(model)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
	}
	return out
}

func containsNormalizedModerationModel(models []string, normalized string) bool {
	for _, item := range models {
		if strings.ToLower(strings.TrimSpace(item)) == normalized {
			return true
		}
	}
	return false
}
