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
	clientIP := ip.GetClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, service.EffectiveProtocol(account))
	simulatedClient := resolveFailedUsageSimulatedClient(account, service.PlatformOpenAI, model)

	h.submitUsageRecordTask(func(ctx context.Context) {
		if recordErr := h.gatewayService.RecordFailedUsage(ctx, &service.OpenAIRecordFailedUsageInput{
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
			SimulatedClient:  simulatedClient,
			Stream:           stream,
			Duration:         duration,
		}); recordErr != nil {
			logger.L().With(
				zap.String("component", component),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", model),
				zap.Int64("account_id", account.ID),
			).Error("openai.record_failed_usage_failed", zap.Error(recordErr))
		}
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
	clientIP := ip.GetClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, service.EffectiveProtocol(account))
	simulatedClient := resolveFailedUsageSimulatedClient(account, protocol, model)

	h.submitUsageRecordTask(func(ctx context.Context) {
		if recordErr := h.gatewayService.RecordFailedUsage(ctx, &service.RecordFailedUsageInput{
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
			SimulatedClient:  simulatedClient,
			Stream:           stream,
			Duration:         duration,
		}); recordErr != nil {
			logger.L().With(
				zap.String("component", component),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", model),
				zap.Int64("account_id", account.ID),
			).Error("gateway.record_failed_usage_failed", zap.Error(recordErr))
		}
	})
}

func (h *SoraGatewayHandler) submitFailedUsageRecordTask(
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
	clientIP := ip.GetClientIP(c)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, service.EffectiveProtocol(account))

	h.submitUsageRecordTask(func(ctx context.Context) {
		if recordErr := h.gatewayService.RecordFailedUsage(ctx, &service.RecordFailedUsageInput{
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
		}); recordErr != nil {
			logger.L().With(
				zap.String("component", component),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", model),
				zap.Int64("account_id", account.ID),
			).Error("sora.record_failed_usage_failed", zap.Error(recordErr))
		}
	})
}
