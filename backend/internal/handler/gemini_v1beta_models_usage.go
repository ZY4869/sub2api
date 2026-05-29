package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) submitGeminiSuccessUsageRecord(
	c *gin.Context,
	reqLog *zap.Logger,
	authSubject middleware.AuthSubject,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	account *service.Account,
	result *service.ForwardResult,
	body []byte,
	modelName string,
	publicModelName string,
	publicCatalogEntry *service.PublishedPublicCatalogEntry,
	channelState *service.GatewayChannelState,
	fs *FailoverState,
) bool {
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetTrustedClientIP(c)
	requestPayloadHash := service.HashUsageRequestPayload(body)
	inboundEndpoint := GetInboundEndpoint(c)
	rawInboundPath := strings.TrimSpace(c.Request.URL.Path)
	upstreamEndpoint := GetUpstreamEndpointForAccount(c, account)
	usageDecision := service.DecideGeminiSuccessUsagePersistence(inboundEndpoint, rawInboundPath, body)
	if !usageDecision.Persist {
		reqLog.Info("gemini.usage_record_skipped", zap.String("reason", usageDecision.Reason), zap.String("operation_type", usageDecision.OperationType), zap.String("inbound_endpoint", inboundEndpoint))
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, apiKey)
	} else {
		h.submitUsageRecordTask(func(ctx context.Context) {
			ctx = service.AttachPublishedPublicCatalogEntry(ctx, publicCatalogEntry)
			ctx = reattachGatewayChannelState(ctx, channelState)
			if err := h.gatewayService.RecordUsageWithLongContext(ctx, &service.RecordUsageLongContextInput{
				Result:                result,
				APIKey:                apiKey,
				User:                  apiKey.User,
				Account:               account,
				Subscription:          subscription,
				InboundEndpoint:       inboundEndpoint,
				RawInboundPath:        rawInboundPath,
				UpstreamEndpoint:      upstreamEndpoint,
				UserAgent:             userAgent,
				IPAddress:             clientIP,
				RequestBody:           body,
				RequestPayloadHash:    requestPayloadHash,
				LongContextThreshold:  200000,
				LongContextMultiplier: 2.0,
				ForceCacheBilling:     fs.ForceCacheBilling,
				APIKeyService:         h.apiKeyService,
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.gemini_v1beta.models"),
					zap.Int64("user_id", authSubject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Any("group_id", apiKey.GroupID),
					zap.String("model", publicModelName),
					zap.Int64("account_id", account.ID),
				).Error("gemini.record_usage_failed", zap.Error(err))
			}
		})
	}
	reqLog.Debug("gemini.request_completed",
		zap.Int64("account_id", account.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.Int("switch_count", fs.SwitchCount),
	)
	return true
}
