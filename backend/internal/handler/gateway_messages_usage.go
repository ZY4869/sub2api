package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) submitGatewayMessagesFailedUsage(
	c *gin.Context,
	req *gatewayMessagesRequest,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	account *service.Account,
	failoverErr *service.UpstreamFailoverError,
	err error,
) {
	h.submitFailedUsageRecordTask(
		"handler.gateway.messages",
		c,
		apiKey,
		subscription,
		account,
		req.reqModel,
		req.reqStream,
		0,
		service.EffectiveProtocol(account),
		failoverErr,
		err,
	)
}

func (h *GatewayHandler) submitGatewayMessagesSuccessUsage(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	account *service.Account,
	result *service.ForwardResult,
	forceCacheBilling bool,
) {
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetTrustedClientIP(c)
	h.submitUsageRecordTask(func(ctx context.Context) {
		ctx = service.AttachPublishedPublicCatalogEntry(ctx, req.publicCatalogEntry)
		ctx = reattachGatewayChannelState(ctx, route.channelState)
		if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
			Result:             result,
			APIKey:             route.apiKey,
			User:               route.apiKey.User,
			Account:            account,
			Subscription:       route.subscription,
			ThinkingEnabled:    service.ParseExplicitThinkingEnabledValue(req.body),
			InboundEndpoint:    GetInboundEndpoint(c),
			UpstreamEndpoint:   GetUpstreamEndpointForAccount(c, account),
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: req.requestPayloadHash,
			ForceCacheBilling:  forceCacheBilling,
			APIKeyService:      h.apiKeyService,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.gateway.messages"),
				zap.Int64("user_id", req.subject.UserID),
				zap.Int64("api_key_id", route.apiKey.ID),
				zap.Any("group_id", route.apiKey.GroupID),
				zap.String("model", req.reqModel),
				zap.Int64("account_id", account.ID),
			).Error("gateway.record_usage_failed", zap.Error(err))
		}
	})
}
