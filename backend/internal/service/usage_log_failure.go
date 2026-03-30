package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

const (
	failedUsageErrorCodeMaxLen    = 128
	failedUsageErrorMessageMaxLen = 1024
)

type OpenAIRecordFailedUsageInput struct {
	APIKey           *APIKey
	User             *User
	Account          *Account
	Subscription     *UserSubscription
	RequestID        string
	Model            string
	UpstreamModel    string
	InboundEndpoint  string
	UpstreamEndpoint string
	UpstreamURL      string
	UpstreamService  string
	UserAgent        string
	IPAddress        string
	HTTPStatus       int
	ErrorCode        string
	ErrorMessage     string
	SimulatedClient  string
	Stream           bool
	OpenAIWSMode     bool
	Duration         time.Duration
	ReasoningEffort  *string
	ThinkingEnabled  *bool
}

type RecordFailedUsageInput struct {
	APIKey           *APIKey
	User             *User
	Account          *Account
	Subscription     *UserSubscription
	RequestID        string
	Model            string
	UpstreamModel    string
	InboundEndpoint  string
	UpstreamEndpoint string
	UpstreamURL      string
	UpstreamService  string
	UserAgent        string
	IPAddress        string
	HTTPStatus       int
	ErrorCode        string
	ErrorMessage     string
	SimulatedClient  string
	Stream           bool
	OpenAIWSMode     bool
	Duration         time.Duration
	ReasoningEffort  *string
	ThinkingEnabled  *bool
	ImageCount       int
	ImageSize        string
	MediaType        string
}

func resolveFailedUsageUser(user *User, apiKey *APIKey) *User {
	if user != nil {
		return user
	}
	if apiKey != nil && apiKey.User != nil {
		return apiKey.User
	}
	return nil
}

func optionalTruncatedTrimmedStringPtr(value string, maxLen int) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if maxLen > 0 && len(trimmed) > maxLen {
		trimmed = trimmed[:maxLen]
	}
	return &trimmed
}

func sanitizeUsageFailureErrorMessage(message string) *string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return nil
	}
	redacted := logredact.RedactText(trimmed, "api_key", "authorization", "x-api-key")
	redacted = strings.Join(strings.Fields(redacted), " ")
	if redacted == "" {
		return nil
	}
	if len(redacted) > failedUsageErrorMessageMaxLen {
		redacted = redacted[:failedUsageErrorMessageMaxLen-3] + "..."
	}
	return &redacted
}

func optionalIntPtr(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func optionalDurationMsPtr(duration time.Duration) *int {
	if duration <= 0 {
		return nil
	}
	ms := int(duration.Milliseconds())
	if ms < 0 {
		ms = 0
	}
	return &ms
}

func optionalMediaTypePtr(value string) *string {
	return optionalTrimmedStringPtr(value)
}

func optionalImageSizePtr(value string) *string {
	return optionalTrimmedStringPtr(value)
}

func buildFailedUsageLogBase(
	ctx context.Context,
	apiKey *APIKey,
	user *User,
	account *Account,
	subscription *UserSubscription,
	multiplier float64,
	input *RecordFailedUsageInput,
) *UsageLog {
	if apiKey == nil || user == nil || account == nil || input == nil {
		return nil
	}

	billingType := BillingTypeBalance
	if subscription != nil && apiKey.Group != nil && apiKey.Group.IsSubscriptionType() {
		billingType = BillingTypeSubscription
	}

	accountRateMultiplier := account.BillingRateMultiplier()
	log := &UsageLog{
		UserID:                user.ID,
		APIKeyID:              apiKey.ID,
		AccountID:             account.ID,
		RequestID:             resolveUsageBillingRequestID(ctx, input.RequestID),
		Model:                 strings.TrimSpace(input.Model),
		RequestedModel:        strings.TrimSpace(input.Model),
		UpstreamModel:         optionalNonEqualStringPtr(input.UpstreamModel, input.Model),
		ReasoningEffort:       input.ReasoningEffort,
		ThinkingEnabled:       input.ThinkingEnabled,
		InboundEndpoint:       optionalTrimmedStringPtr(input.InboundEndpoint),
		UpstreamEndpoint:      optionalTrimmedStringPtr(input.UpstreamEndpoint),
		UpstreamURL:           optionalTrimmedStringPtr(ResolveUsageLogUpstreamURL(account, input.UpstreamURL)),
		UpstreamService:       optionalTrimmedStringPtr(ResolveUsageLogUpstreamService(account, input.UpstreamService)),
		RateMultiplier:        multiplier,
		AccountRateMultiplier: &accountRateMultiplier,
		BillingType:           billingType,
		RequestType:           RequestTypeFromLegacy(input.Stream, input.OpenAIWSMode),
		Status:                UsageLogStatusFailed,
		Stream:                input.Stream,
		OpenAIWSMode:          input.OpenAIWSMode,
		DurationMs:            optionalDurationMsPtr(input.Duration),
		UserAgent:             optionalTrimmedStringPtr(input.UserAgent),
		IPAddress:             optionalTrimmedStringPtr(input.IPAddress),
		HTTPStatus:            optionalIntPtr(input.HTTPStatus),
		ErrorCode:             optionalTruncatedTrimmedStringPtr(input.ErrorCode, failedUsageErrorCodeMaxLen),
		ErrorMessage:          sanitizeUsageFailureErrorMessage(input.ErrorMessage),
		SimulatedClient:       NormalizeUsageLogSimulatedClient(input.SimulatedClient),
		ImageCount:            input.ImageCount,
		ImageSize:             optionalImageSizePtr(input.ImageSize),
		MediaType:             optionalMediaTypePtr(input.MediaType),
		CreatedAt:             time.Now(),
	}
	if apiKey.GroupID != nil {
		log.GroupID = apiKey.GroupID
	}
	if subscription != nil {
		log.SubscriptionID = &subscription.ID
	}
	log.SyncRequestTypeAndLegacyFields()
	return log
}

func (s *OpenAIGatewayService) RecordFailedUsage(ctx context.Context, input *OpenAIRecordFailedUsageInput) error {
	if s == nil || input == nil || input.APIKey == nil || input.Account == nil {
		return nil
	}
	user := resolveFailedUsageUser(input.User, input.APIKey)
	if user == nil {
		return nil
	}

	multiplier := 1.0
	if s.cfg != nil {
		multiplier = s.cfg.Default.RateMultiplier
	}
	if input.APIKey.GroupID != nil && input.APIKey.Group != nil {
		resolver := s.userGroupRateResolver
		if resolver == nil {
			resolver = newUserGroupRateResolver(nil, nil, resolveUserGroupRateCacheTTL(s.cfg), nil, "service.openai_gateway")
		}
		multiplier = resolver.Resolve(ctx, user.ID, *input.APIKey.GroupID, input.APIKey.Group.RateMultiplier)
	}

	usageLog := buildFailedUsageLogBase(ctx, input.APIKey, user, input.Account, input.Subscription, multiplier, &RecordFailedUsageInput{
		RequestID:        input.RequestID,
		Model:            input.Model,
		UpstreamModel:    input.UpstreamModel,
		InboundEndpoint:  input.InboundEndpoint,
		UpstreamEndpoint: input.UpstreamEndpoint,
		UpstreamURL:      input.UpstreamURL,
		UpstreamService:  input.UpstreamService,
		UserAgent:        input.UserAgent,
		IPAddress:        input.IPAddress,
		HTTPStatus:       input.HTTPStatus,
		ErrorCode:        input.ErrorCode,
		ErrorMessage:     input.ErrorMessage,
		SimulatedClient:  input.SimulatedClient,
		Stream:           input.Stream,
		OpenAIWSMode:     input.OpenAIWSMode,
		Duration:         input.Duration,
		ReasoningEffort:  input.ReasoningEffort,
		ThinkingEnabled:  input.ThinkingEnabled,
	})
	writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")
	if s.deferredService != nil && input.Account != nil {
		s.deferredService.ScheduleLastUsedUpdate(input.Account.ID)
	}
	return nil
}

func (s *GatewayService) RecordFailedUsage(ctx context.Context, input *RecordFailedUsageInput) error {
	if s == nil || input == nil || input.APIKey == nil || input.Account == nil {
		return nil
	}
	user := resolveFailedUsageUser(input.User, input.APIKey)
	if user == nil {
		return nil
	}

	multiplier := 1.0
	if s.cfg != nil {
		multiplier = s.cfg.Default.RateMultiplier
	}
	if input.APIKey.GroupID != nil && input.APIKey.Group != nil {
		multiplier = s.getUserGroupRateMultiplier(ctx, user.ID, *input.APIKey.GroupID, input.APIKey.Group.RateMultiplier)
	}
	if input.ThinkingEnabled == nil {
		input.ThinkingEnabled = usageLogThinkingEnabledFromContext(ctx)
	}

	usageLog := buildFailedUsageLogBase(ctx, input.APIKey, user, input.Account, input.Subscription, multiplier, input)
	writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.gateway")
	if s.deferredService != nil && input.Account != nil {
		s.deferredService.ScheduleLastUsedUpdate(input.Account.ID)
	}
	return nil
}
