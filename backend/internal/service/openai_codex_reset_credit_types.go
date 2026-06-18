package service

import (
	"context"
	"time"
)

const (
	openAIResetCreditsSourceCodexAppServer        = "codex_app_server"
	openAIResetCreditsStatusAvailable             = "available"
	openAIResetCreditsStatusUnknownOrUnsupported  = "unknown_or_unsupported"
	openAIResetCreditsStatusUnsupported           = "unsupported"
	openAIResetCreditsAvailableCountExtraKey      = "openai_rate_limit_reset_credits_available_count"
	openAIResetCreditsUpdatedAtExtraKey           = "openai_rate_limit_reset_credits_updated_at"
	openAIRateLimitsAppServerUpdatedAtExtraKey    = "openai_rate_limits_app_server_updated_at"
	openAIResetCreditsStatusExtraKey              = "openai_rate_limit_reset_credits_status"
	openAIResetCreditsUnsupportedReasonExtraKey   = "openai_rate_limit_reset_credits_unsupported_reason"
	openAIResetCreditLastConsumeStatusExtraKey    = "openai_rate_limit_reset_credit_last_consume_status"
	openAIResetCreditLastConsumeUpdatedAtExtraKey = "openai_rate_limit_reset_credit_last_consume_updated_at"
	openAIResetCreditConsumeStatusReset           = "reset"
	openAIResetCreditConsumeStatusAlreadyRedeemed = "alreadyRedeemed"
	openAIResetCreditConsumeStatusNothingToReset  = "nothingToReset"
	openAIResetCreditConsumeStatusNoCredit        = "noCredit"
)

type OpenAICodexAppServerAuthTokens struct {
	AccessToken      string
	ChatGPTAccountID string
	ChatGPTPlanType  string
}

type OpenAICodexAppServerRateLimitsSnapshot struct {
	AvailableCount      *int
	UpdatedAt           time.Time
	Status              string
	UnsupportedReason   string
	RateLimits          []byte
	RateLimitsByLimitID []byte
	ExtraUpdates        map[string]any
}

type OpenAICodexAppServerConsumeResult struct {
	Status   string
	Snapshot *OpenAICodexAppServerRateLimitsSnapshot
}

type OpenAICodexResetCreditsSnapshot struct {
	AvailableCount    *int
	UpdatedAt         time.Time
	Source            string
	Status            string
	UnsupportedReason string
}

type OpenAICodexResetCreditConsumeResult struct {
	Status   string
	Snapshot *OpenAICodexResetCreditsSnapshot
}

type OpenAICodexAppServerRateLimitClient interface {
	ReadRateLimits(ctx context.Context, auth OpenAICodexAppServerAuthTokens) (*OpenAICodexAppServerRateLimitsSnapshot, error)
	ConsumeResetCredit(ctx context.Context, auth OpenAICodexAppServerAuthTokens, idempotencyKey string) (*OpenAICodexAppServerConsumeResult, error)
}

type OpenAICodexResetCreditReader interface {
	ReadResetCredits(ctx context.Context, account *Account) (*OpenAICodexResetCreditsSnapshot, error)
}

type OpenAICodexResetCreditConsumer interface {
	ConsumeResetCredit(ctx context.Context, account *Account, idempotencyKey string) (*OpenAICodexResetCreditConsumeResult, error)
}
