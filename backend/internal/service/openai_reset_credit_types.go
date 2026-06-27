package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	openAIResetCreditsSourceWham                 = "chatgpt_wham"
	openAIResetCreditsStatusAvailable            = "available"
	openAIResetCreditsStatusUnknownOrUnsupported = "unknown_or_unsupported"
	openAIResetCreditsStatusUnsupported          = "unsupported"
	openAIResetCreditsAvailableCountExtraKey     = "openai_rate_limit_reset_credits_available_count"
	openAIResetCreditsUpdatedAtExtraKey          = "openai_rate_limit_reset_credits_updated_at"
	openAIQuotaUsageUpdatedAtExtraKey            = "openai_quota_usage_updated_at"
	openAIResetCreditsStatusExtraKey             = "openai_rate_limit_reset_credits_status"
	openAIResetCreditsUnsupportedReasonExtraKey  = "openai_rate_limit_reset_credits_unsupported_reason"
)

type OpenAIResetCreditsSnapshot struct {
	AvailableCount    *int
	UpdatedAt         time.Time
	Source            string
	Status            string
	UnsupportedReason string
}

type OpenAIResetCreditReader interface {
	ReadResetCredits(ctx context.Context, account *Account) (*OpenAIResetCreditsSnapshot, error)
}

func parseOpenAIResetCreditsAvailableCount(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0, false
		}
		return int(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return v, true
	case int64:
		if v < 0 {
			return 0, false
		}
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < 0 {
			return 0, false
		}
		return int(i), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		i, err := json.Number(trimmed).Int64()
		if err != nil || i < 0 {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}
