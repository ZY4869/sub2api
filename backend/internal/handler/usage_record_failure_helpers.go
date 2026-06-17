package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type failedUsageResolution struct {
	RequestID    string
	HTTPStatus   int
	ErrorCode    string
	ErrorMessage string
}

type contentModerationFailedUsageRequest struct {
	Component        string
	APIKey           *service.APIKey
	Subscription     *service.UserSubscription
	Account          *service.Account
	Model            string
	Protocol         string
	Stream           bool
	RequestID        string
	InboundEndpoint  string
	UpstreamEndpoint string
	UserAgent        string
	IPAddress        string
	ErrorCode        string
	ErrorMessage     string
}

func firstHeaderValue(headers http.Header, keys ...string) string {
	for _, key := range keys {
		if headers == nil {
			break
		}
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func optionalContextString(value string, ok bool) *string {
	if !ok {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalContextBool(value bool, ok bool) *bool {
	if !ok {
		return nil
	}
	resolved := value
	return &resolved
}

func resolveFailedUsageResolution(c *gin.Context, failoverErr *service.UpstreamFailoverError, err error) failedUsageResolution {
	resolution := failedUsageResolution{}
	if failoverErr != nil {
		resolution.RequestID = firstHeaderValue(
			failoverErr.ResponseHeaders,
			"x-request-id",
			"X-Request-Id",
			"x-amzn-requestid",
			"X-Amzn-Requestid",
		)
		resolution.HTTPStatus = failoverErr.StatusCode
		resolution.ErrorCode = strings.TrimSpace(service.ExtractUpstreamErrorCode(failoverErr.ResponseBody))
		resolution.ErrorMessage = strings.TrimSpace(service.ExtractUpstreamErrorMessage(failoverErr.ResponseBody))
		if resolution.ErrorMessage == "" && len(failoverErr.ResponseBody) > 0 {
			resolution.ErrorMessage = strings.TrimSpace(string(failoverErr.ResponseBody))
		}
	}
	if resolution.RequestID == "" && c != nil && c.Writer != nil {
		resolution.RequestID = firstHeaderValue(
			c.Writer.Header(),
			"x-request-id",
			"X-Request-Id",
			"x-amzn-requestid",
			"X-Amzn-Requestid",
		)
	}
	if resolution.HTTPStatus == 0 && c != nil && c.Writer != nil && c.Writer.Status() >= http.StatusBadRequest {
		resolution.HTTPStatus = c.Writer.Status()
	}
	if resolution.ErrorMessage == "" && err != nil {
		resolution.ErrorMessage = err.Error()
	}
	return resolution
}

func resolveFailedUsageSimulatedClient(account *service.Account, protocol string, requestedModel string) string {
	if account == nil || strings.TrimSpace(protocol) == "" || strings.TrimSpace(requestedModel) == "" {
		return ""
	}
	if route := service.MatchGatewayClientRoute(account, protocol, account.GetMappedModel(requestedModel)); route != nil {
		return route.ClientProfile
	}
	if route := service.MatchGatewayClientRoute(account, protocol, requestedModel); route != nil {
		return route.ClientProfile
	}
	return ""
}

func (h *OpenAIGatewayHandler) submitFailedUsageRecordTask(
	component string,
	c *gin.Context,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	account *service.Account,
	model string,
	stream bool,
	duration time.Duration,
	failoverErr *service.UpstreamFailoverError,
	err error,
) {
	if h == nil || c == nil || apiKey == nil || account == nil {
		return
	}
	resolution := resolveFailedUsageResolution(c, failoverErr, err)
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetTrustedClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpointForAccount(c, account)
	simulatedClient := resolveFailedUsageSimulatedClient(account, service.PlatformOpenAI, model)
	requestCtx := c.Request.Context()
	requestedModelRaw := optionalContextString(service.ClaudeRequestedModelRawMetadataFromContext(requestCtx))
	requestedModelNormalized := optionalContextString(service.ClaudeRequestedModelNormalizedMetadataFromContext(requestCtx))
	millionContextRequested := optionalContextBool(service.ClaudeMillionContextRequestedMetadataFromContext(requestCtx))
	millionContextEffective := optionalContextBool(service.ClaudeMillionContextEffectiveMetadataFromContext(requestCtx))
	millionContextSource := optionalContextString(service.ClaudeMillionContextSourceMetadataFromContext(requestCtx))
	millionContextBetaToken := optionalContextString(service.ClaudeMillionContextBetaTokenMetadataFromContext(requestCtx))

	h.submitUsageRecordTask(func(ctx context.Context) {
		recordErr := h.gatewayService.RecordFailedUsage(ctx, &service.OpenAIRecordFailedUsageInput{
			APIKey:                   apiKey,
			User:                     apiKey.User,
			Account:                  account,
			Subscription:             subscription,
			RequestID:                resolution.RequestID,
			Model:                    model,
			UpstreamModel:            account.GetMappedModel(model),
			InboundEndpoint:          inboundEndpoint,
			UpstreamEndpoint:         upstreamEndpoint,
			UserAgent:                userAgent,
			IPAddress:                clientIP,
			HTTPStatus:               resolution.HTTPStatus,
			ErrorCode:                resolution.ErrorCode,
			ErrorMessage:             resolution.ErrorMessage,
			SimulatedClient:          simulatedClient,
			Stream:                   stream,
			Duration:                 duration,
			RequestedModelRaw:        requestedModelRaw,
			RequestedModelNormalized: requestedModelNormalized,
			MillionContextRequested:  millionContextRequested,
			MillionContextEffective:  millionContextEffective,
			MillionContextSource:     millionContextSource,
			MillionContextBetaToken:  millionContextBetaToken,
		})
		if recordErr != nil {
			logger.L().With(
				zap.String("component", component),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", model),
				zap.Int64("account_id", account.ID),
			).Error("openai.record_failed_usage_failed", zap.Error(recordErr))
		}
		releaseHeldBillingHold(ctx, h.apiKeyService, apiKey)
	})
}

func (h *OpenAIGatewayHandler) submitContentModerationFailedUsageRecordTask(
	component string,
	c *gin.Context,
	apiKey *service.APIKey,
	model string,
	stream bool,
	decision *service.ContentModerationKeywordDecision,
) {
	if h == nil || c == nil || apiKey == nil || h.gatewayService == nil {
		return
	}
	resolved, err := h.gatewayService.ResolveContentModerationUsageAccount(
		c.Request.Context(),
		apiKey,
		openAITextCompatiblePlatforms,
		model,
	)
	if err != nil || resolved == nil || resolved.Account == nil || resolved.APIKey == nil {
		logger.FromContext(c.Request.Context()).Warn(
			"openai.content_moderation_failed_usage_account_unavailable",
			zap.String("component", component),
			zap.Int64("api_key_id", apiKey.ID),
			zap.Any("group_id", apiKey.GroupID),
			zap.String("model", model),
			zap.Error(err),
		)
		return
	}
	code, message := contentModerationBlockError(decision)
	request := buildContentModerationFailedUsageRequest(
		component,
		c,
		resolved.APIKey,
		resolved.Subscription,
		resolved.Account,
		model,
		service.PlatformOpenAI,
		stream,
		code,
		message,
	)
	h.submitUsageRecordTask(func(ctx context.Context) {
		recordContentModerationOpenAIFailedUsage(ctx, h.gatewayService, request)
		releaseHeldBillingHold(ctx, h.apiKeyService, resolved.APIKey)
	})
}

func (h *GatewayHandler) submitFailedUsageRecordTask(
	component string,
	c *gin.Context,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	account *service.Account,
	model string,
	stream bool,
	duration time.Duration,
	protocol string,
	failoverErr *service.UpstreamFailoverError,
	err error,
) {
	if h == nil || c == nil || apiKey == nil || account == nil {
		return
	}
	resolution := resolveFailedUsageResolution(c, failoverErr, err)
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetTrustedClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpointForAccount(c, account)
	simulatedClient := resolveFailedUsageSimulatedClient(account, protocol, model)
	requestCtx := c.Request.Context()
	requestedModelRaw := optionalContextString(service.ClaudeRequestedModelRawMetadataFromContext(requestCtx))
	requestedModelNormalized := optionalContextString(service.ClaudeRequestedModelNormalizedMetadataFromContext(requestCtx))
	millionContextRequested := optionalContextBool(service.ClaudeMillionContextRequestedMetadataFromContext(requestCtx))
	millionContextEffective := optionalContextBool(service.ClaudeMillionContextEffectiveMetadataFromContext(requestCtx))
	millionContextSource := optionalContextString(service.ClaudeMillionContextSourceMetadataFromContext(requestCtx))
	millionContextBetaToken := optionalContextString(service.ClaudeMillionContextBetaTokenMetadataFromContext(requestCtx))

	h.submitUsageRecordTask(func(ctx context.Context) {
		recordErr := h.gatewayService.RecordFailedUsage(ctx, &service.RecordFailedUsageInput{
			APIKey:                   apiKey,
			User:                     apiKey.User,
			Account:                  account,
			Subscription:             subscription,
			RequestID:                resolution.RequestID,
			Model:                    model,
			UpstreamModel:            account.GetMappedModel(model),
			InboundEndpoint:          inboundEndpoint,
			UpstreamEndpoint:         upstreamEndpoint,
			UserAgent:                userAgent,
			IPAddress:                clientIP,
			HTTPStatus:               resolution.HTTPStatus,
			ErrorCode:                resolution.ErrorCode,
			ErrorMessage:             resolution.ErrorMessage,
			SimulatedClient:          simulatedClient,
			Stream:                   stream,
			Duration:                 duration,
			RequestedModelRaw:        requestedModelRaw,
			RequestedModelNormalized: requestedModelNormalized,
			MillionContextRequested:  millionContextRequested,
			MillionContextEffective:  millionContextEffective,
			MillionContextSource:     millionContextSource,
			MillionContextBetaToken:  millionContextBetaToken,
		})
		if recordErr != nil {
			logger.L().With(
				zap.String("component", component),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", model),
				zap.Int64("account_id", account.ID),
			).Error("gateway.record_failed_usage_failed", zap.Error(recordErr))
		}
		releaseHeldBillingHold(ctx, h.apiKeyService, apiKey)
	})
}

func (h *GatewayHandler) submitContentModerationFailedUsageRecordTask(
	component string,
	c *gin.Context,
	apiKey *service.APIKey,
	model string,
	stream bool,
	protocol string,
	allowedPlatforms []string,
	decision *service.ContentModerationKeywordDecision,
) {
	if h == nil || c == nil || apiKey == nil || h.gatewayService == nil {
		return
	}
	resolved, err := h.gatewayService.ResolveContentModerationUsageAccount(
		c.Request.Context(),
		apiKey,
		allowedPlatforms,
		model,
	)
	if err != nil || resolved == nil || resolved.Account == nil || resolved.APIKey == nil {
		logger.FromContext(c.Request.Context()).Warn(
			"gateway.content_moderation_failed_usage_account_unavailable",
			zap.String("component", component),
			zap.Int64("api_key_id", apiKey.ID),
			zap.Any("group_id", apiKey.GroupID),
			zap.String("model", model),
			zap.String("protocol", protocol),
			zap.Error(err),
		)
		return
	}
	code, message := contentModerationBlockError(decision)
	request := buildContentModerationFailedUsageRequest(
		component,
		c,
		resolved.APIKey,
		resolved.Subscription,
		resolved.Account,
		model,
		protocol,
		stream,
		code,
		message,
	)
	h.submitUsageRecordTask(func(ctx context.Context) {
		recordContentModerationGatewayFailedUsage(ctx, h.gatewayService, request)
		releaseHeldBillingHold(ctx, h.apiKeyService, resolved.APIKey)
	})
}

func buildContentModerationFailedUsageRequest(
	component string,
	c *gin.Context,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	account *service.Account,
	model string,
	protocol string,
	stream bool,
	errorCode string,
	errorMessage string,
) contentModerationFailedUsageRequest {
	return contentModerationFailedUsageRequest{
		Component:        component,
		APIKey:           apiKey,
		Subscription:     subscription,
		Account:          account,
		Model:            strings.TrimSpace(model),
		Protocol:         strings.TrimSpace(protocol),
		Stream:           stream,
		RequestID:        service.ContentModerationRequestIDFromContext(c.Request.Context()),
		InboundEndpoint:  GetInboundEndpoint(c),
		UpstreamEndpoint: GetUpstreamEndpointForAccount(c, account),
		UserAgent:        c.GetHeader("User-Agent"),
		IPAddress:        ip.GetTrustedClientIP(c),
		ErrorCode:        strings.TrimSpace(errorCode),
		ErrorMessage:     strings.TrimSpace(errorMessage),
	}
}

func recordContentModerationOpenAIFailedUsage(
	ctx context.Context,
	gatewayService *service.OpenAIGatewayService,
	request contentModerationFailedUsageRequest,
) {
	if gatewayService == nil || request.APIKey == nil || request.Account == nil {
		return
	}
	recordErr := gatewayService.RecordFailedUsage(ctx, &service.OpenAIRecordFailedUsageInput{
		APIKey:              request.APIKey,
		User:                request.APIKey.User,
		Account:             request.Account,
		Subscription:        request.Subscription,
		RequestID:           request.RequestID,
		Model:               request.Model,
		UpstreamModel:       request.Account.GetMappedModel(request.Model),
		InboundEndpoint:     request.InboundEndpoint,
		UpstreamEndpoint:    request.UpstreamEndpoint,
		UserAgent:           request.UserAgent,
		IPAddress:           request.IPAddress,
		HTTPStatus:          http.StatusForbidden,
		ErrorCode:           request.ErrorCode,
		ErrorMessage:        request.ErrorMessage,
		BillingExemptReason: service.BillingExemptReasonContentModerationBlocked,
		SimulatedClient:     resolveFailedUsageSimulatedClient(request.Account, service.PlatformOpenAI, request.Model),
		Stream:              request.Stream,
	})
	if recordErr != nil {
		logger.L().With(
			zap.String("component", request.Component),
			zap.Int64("api_key_id", request.APIKey.ID),
			zap.Any("group_id", request.APIKey.GroupID),
			zap.String("model", request.Model),
			zap.Int64("account_id", request.Account.ID),
		).Error("openai.content_moderation_record_failed_usage_failed", zap.Error(recordErr))
	}
}

func recordContentModerationGatewayFailedUsage(
	ctx context.Context,
	gatewayService *service.GatewayService,
	request contentModerationFailedUsageRequest,
) {
	if gatewayService == nil || request.APIKey == nil || request.Account == nil {
		return
	}
	recordErr := gatewayService.RecordFailedUsage(ctx, &service.RecordFailedUsageInput{
		APIKey:              request.APIKey,
		User:                request.APIKey.User,
		Account:             request.Account,
		Subscription:        request.Subscription,
		RequestID:           request.RequestID,
		Model:               request.Model,
		UpstreamModel:       request.Account.GetMappedModel(request.Model),
		InboundEndpoint:     request.InboundEndpoint,
		UpstreamEndpoint:    request.UpstreamEndpoint,
		UserAgent:           request.UserAgent,
		IPAddress:           request.IPAddress,
		HTTPStatus:          http.StatusForbidden,
		ErrorCode:           request.ErrorCode,
		ErrorMessage:        request.ErrorMessage,
		BillingExemptReason: service.BillingExemptReasonContentModerationBlocked,
		SimulatedClient:     resolveFailedUsageSimulatedClient(request.Account, request.Protocol, request.Model),
		Stream:              request.Stream,
	})
	if recordErr != nil {
		logger.L().With(
			zap.String("component", request.Component),
			zap.Int64("api_key_id", request.APIKey.ID),
			zap.Any("group_id", request.APIKey.GroupID),
			zap.String("model", request.Model),
			zap.Int64("account_id", request.Account.ID),
		).Error("gateway.content_moderation_record_failed_usage_failed", zap.Error(recordErr))
	}
}

func (h *GrokGatewayHandler) submitFailedUsageRecordTask(
	component string,
	c *gin.Context,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	account *service.Account,
	model string,
	stream bool,
	duration time.Duration,
	failoverErr *service.UpstreamFailoverError,
	err error,
	mediaType string,
	imageCount int,
	imageSize string,
) {
	if h == nil || c == nil || apiKey == nil || account == nil {
		return
	}
	resolution := resolveFailedUsageResolution(c, failoverErr, err)
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetTrustedClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpointForAccount(c, account)

	h.submitUsageRecordTask(func(ctx context.Context) {
		recordErr := h.gatewayService.RecordFailedUsage(ctx, &service.RecordFailedUsageInput{
			APIKey:           apiKey,
			User:             apiKey.User,
			Account:          account,
			Subscription:     subscription,
			RequestID:        resolution.RequestID,
			Model:            model,
			UpstreamModel:    account.GetMappedModel(model),
			InboundEndpoint:  inboundEndpoint,
			UpstreamEndpoint: upstreamEndpoint,
			UserAgent:        userAgent,
			IPAddress:        clientIP,
			HTTPStatus:       resolution.HTTPStatus,
			ErrorCode:        resolution.ErrorCode,
			ErrorMessage:     resolution.ErrorMessage,
			Stream:           stream,
			Duration:         duration,
			MediaType:        mediaType,
			ImageCount:       imageCount,
			ImageSize:        imageSize,
		})
		if recordErr != nil {
			logger.L().With(
				zap.String("component", component),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", model),
				zap.Int64("account_id", account.ID),
			).Error("grok.record_failed_usage_failed", zap.Error(recordErr))
		}
		releaseHeldBillingHold(ctx, h.apiKeyService, apiKey)
	})
}

func releaseHeldBillingHold(ctx context.Context, apiKeyService *service.APIKeyService, apiKey *service.APIKey) {
	if apiKeyService == nil || apiKey == nil || apiKey.BillingHold == nil || apiKey.BillingHold.Status != service.BillingHoldStatusHeld {
		return
	}
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	releaseCtx, cancel := context.WithTimeout(base, service.BillingHoldSettlementMaxAge(nil))
	defer cancel()
	apiKeyService.ReleaseRequestBillingHold(releaseCtx, apiKey)
	if apiKey.BillingHold != nil && apiKey.BillingHold.Status == service.BillingHoldStatusHeld {
		logger.L().With(
			zap.Int64("api_key_id", apiKey.ID),
			zap.String("request_id", apiKey.BillingHold.RequestID),
		).Warn("billing_hold_release_still_held")
	}
}

func releaseHeldBillingHoldBeforeRetry(ctx context.Context, apiKeyService *service.APIKeyService, apiKey *service.APIKey) {
	_ = ctx
	_ = apiKeyService
	_ = apiKey
}
