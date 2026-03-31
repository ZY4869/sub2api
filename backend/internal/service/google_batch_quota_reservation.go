package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	GoogleBatchQuotaReservationStatusActive   = "active"
	GoogleBatchQuotaReservationStatusReleased = "released"
)

type GoogleBatchQuotaReservation struct {
	ID             int64
	ProviderFamily string
	AccountID      int64
	ResourceName   string
	ModelFamily    string
	ReservedTokens int64
	Status         string
	MetadataJSON   map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type GoogleBatchQuotaReservationRepository interface {
	Upsert(ctx context.Context, reservation *GoogleBatchQuotaReservation) error
	GetByResourceName(ctx context.Context, resourceName string) (*GoogleBatchQuotaReservation, error)
	ReleaseByResourceName(ctx context.Context, resourceName string, status string) error
	SumActiveReservedTokens(ctx context.Context, providerFamily string, accountID int64, modelFamily string) (int64, error)
}

func (s *GeminiMessagesCompatService) SetGoogleBatchQuotaReservationRepository(repo GoogleBatchQuotaReservationRepository) {
	s.googleBatchQuotaReservationRepo = repo
}

func (s *GeminiMessagesCompatService) googleBatchAccountHasQuotaCapacity(ctx context.Context, account *Account, target googleBatchTarget, selector *vertexBatchSelector) bool {
	if account == nil {
		return false
	}
	if target != googleBatchTargetAIStudio || selector == nil || selector.estimatedTokens <= 0 || s.googleBatchQuotaReservationRepo == nil {
		return true
	}
	limit := aiStudioBatchReservationLimitForAccount(account, selector.modelFamily)
	if limit <= 0 {
		return true
	}
	reserved, err := s.googleBatchQuotaReservationRepo.SumActiveReservedTokens(ctx, providerFamilyForTarget(target), account.ID, selector.modelFamily)
	if err != nil {
		return true
	}
	return reserved+selector.estimatedTokens <= limit
}

func (s *GeminiMessagesCompatService) reserveGoogleBatchQuota(ctx context.Context, input GoogleBatchForwardInput, account *Account, target googleBatchTarget, resourceName string) error {
	if s.googleBatchQuotaReservationRepo == nil || account == nil || strings.TrimSpace(resourceName) == "" {
		return nil
	}
	selector := buildGoogleBatchSelectorFromInput(input)
	if selector.estimatedTokens <= 0 {
		selector.estimatedTokens = estimateGoogleBatchTokensFromPayload(input.Body)
	}
	reservation := &GoogleBatchQuotaReservation{
		ProviderFamily: providerFamilyForTarget(target),
		AccountID:      account.ID,
		ResourceName:   strings.TrimSpace(resourceName),
		ModelFamily:    selector.modelFamily,
		ReservedTokens: selector.estimatedTokens,
		Status:         GoogleBatchQuotaReservationStatusActive,
		MetadataJSON: map[string]any{
			"public_protocol":        publicGoogleBatchProtocol(input.Path),
			"upstream_protocol":      providerFamilyForTarget(target),
			"estimated_batch_tokens": selector.estimatedTokens,
			"model_family":           selector.modelFamily,
			"request_path":           strings.TrimSpace(input.Path),
		},
	}
	return s.googleBatchQuotaReservationRepo.Upsert(ctx, reservation)
}

func (s *GeminiMessagesCompatService) releaseGoogleBatchQuota(ctx context.Context, resourceName string, status string) {
	if s.googleBatchQuotaReservationRepo == nil || strings.TrimSpace(resourceName) == "" {
		return
	}
	if strings.TrimSpace(status) == "" {
		status = GoogleBatchQuotaReservationStatusReleased
	}
	_ = s.googleBatchQuotaReservationRepo.ReleaseByResourceName(ctx, resourceName, status)
}

func aiStudioBatchReservationLimitForAccount(account *Account, modelFamily string) int64 {
	if account == nil {
		return 0
	}
	tierID := canonicalGeminiTierID(account.GeminiTierID())
	if tierID == "" {
		tierID = GeminiTierAIStudioFree
	}
	if tierID == GeminiTierAIStudioTier1 {
		tierID = GeminiTierAIStudioTier2
	}
	catalog := defaultGeminiRateCatalog()
	for _, tier := range catalog.BatchLimits.ByTier {
		if tier.TierID != tierID {
			continue
		}
		for _, entry := range tier.Entries {
			if strings.EqualFold(strings.TrimSpace(entry.ModelFamily), strings.TrimSpace(modelFamily)) {
				return entry.EnqueuedTokens
			}
		}
	}
	return 0
}

func buildGoogleBatchSelectorFromInput(input GoogleBatchForwardInput) *vertexBatchSelector {
	selector := &vertexBatchSelector{
		modelFamily:     normalizeGoogleBatchModelFamily(extractGoogleBatchModelID(input.Path, input.Body)),
		estimatedTokens: estimateGoogleBatchTokensFromPayload(input.Body),
	}
	return selector
}

func extractGoogleBatchModelID(path string, body []byte) string {
	trimmed := strings.TrimSpace(path)
	if strings.Contains(trimmed, ":batchGenerateContent") {
		modelPath := strings.TrimPrefix(trimmed, "/v1beta/models/")
		if idx := strings.Index(modelPath, ":"); idx >= 0 {
			modelPath = modelPath[:idx]
		}
		return strings.TrimPrefix(strings.TrimSpace(modelPath), "models/")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if model, ok := payload["model"].(string); ok {
		return strings.TrimSpace(strings.TrimPrefix(model, "publishers/google/models/"))
	}
	return ""
}

func normalizeGoogleBatchModelFamily(model string) string {
	value := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(value, "2.5-pro"), strings.Contains(value, "pro"):
		return "gemini_pro"
	case strings.Contains(value, "flash-lite"), strings.Contains(value, "lite"):
		return "gemini_flash_lite"
	case strings.Contains(value, "2.0-flash"):
		return "gemini_2_flash"
	case strings.Contains(value, "flash"):
		return "gemini_flash"
	default:
		return "gemini_flash"
	}
}

func estimateGoogleBatchTokensFromPayload(body []byte) int64 {
	if len(body) == 0 {
		return 0
	}
	values := collectIntFieldsByKeys(body, []string{"enqueuedTokens", "enqueued_tokens", "estimatedBatchTokens", "estimated_batch_tokens", "tokenCount", "token_count", "totalTokens", "total_tokens"})
	var maxValue int64
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func collectIntFieldsByKeys(body []byte, keys []string) []int64 {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	lookup := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		lookup[strings.ToLower(strings.TrimSpace(key))] = struct{}{}
	}
	var values []int64
	collectIntFieldsByKeyRecursive(payload, lookup, &values)
	return values
}

func collectIntFieldsByKeyRecursive(value any, keys map[string]struct{}, out *[]int64) {
	switch typed := value.(type) {
	case map[string]any:
		for itemKey, itemValue := range typed {
			if _, ok := keys[strings.ToLower(strings.TrimSpace(itemKey))]; ok {
				switch v := itemValue.(type) {
				case float64:
					*out = append(*out, int64(v))
				case int64:
					*out = append(*out, v)
				case int:
					*out = append(*out, int64(v))
				}
			}
			collectIntFieldsByKeyRecursive(itemValue, keys, out)
		}
	case []any:
		for _, item := range typed {
			collectIntFieldsByKeyRecursive(item, keys, out)
		}
	}
}

func publicGoogleBatchProtocol(path string) string {
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, "/v1/projects/") {
		return UpstreamProviderVertexAI
	}
	return UpstreamProviderAIStudio
}
