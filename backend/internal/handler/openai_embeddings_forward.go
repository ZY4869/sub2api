package handler

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var errEmbeddingsAccountSlotNotAcquired = errors.New("embeddings account slot not acquired")

func (h *OpenAIGatewayHandler) forwardEmbeddingsWithAccount(
	c *gin.Context,
	input openAIEmbeddingsForwardInput,
	selection *service.AccountSelectionResult,
) (*service.OpenAIForwardResult, *service.Account, time.Duration, error) {
	account := selection.Account
	setOpsSelectedAccountDetails(c, account)
	setOpsEndpointContext(c, account.GetMappedModel(input.runtimeSelectionModel), service.RequestTypeSync)
	accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, input.currentAPIKey.GroupID, input.sessionHash, selection, false, input.streamStarted, input.req.reqLog)
	if !acquired {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, input.currentAPIKey)
		return nil, account, 0, errEmbeddingsAccountSlotNotAcquired
	}
	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(input.routingStart).Milliseconds())
	forwardStart := time.Now()
	result, err := h.gatewayService.ForwardEmbeddings(c.Request.Context(), c, account, input.req.body)
	forwardDuration := time.Since(forwardStart)
	if accountReleaseFunc != nil {
		accountReleaseFunc()
	}
	recordEmbeddingsResponseLatency(c, forwardDuration)
	return result, account, forwardDuration, err
}

func recordEmbeddingsResponseLatency(c *gin.Context, forwardDuration time.Duration) {
	upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
	responseLatencyMs := forwardDuration.Milliseconds()
	if upstreamLatencyMs > 0 && responseLatencyMs > upstreamLatencyMs {
		responseLatencyMs -= upstreamLatencyMs
	}
	service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
}

func (h *OpenAIGatewayHandler) handleEmbeddingsForwardError(
	c *gin.Context,
	input openAIEmbeddingsForwardInput,
	account *service.Account,
	forwardDuration time.Duration,
	err error,
	failedAccountIDs map[int64]struct{},
	lastFailoverErr **service.UpstreamFailoverError,
	switchCount *int,
) bool {
	if errors.Is(err, errEmbeddingsAccountSlotNotAcquired) {
		return false
	}
	var failoverErr *service.UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		h.gatewayService.RecordOpenAIAccountSwitch()
		failedAccountIDs[account.ID] = struct{}{}
		*lastFailoverErr = failoverErr
		if *switchCount >= h.maxAccountSwitches {
			if excludeSelectedGroup(input.excludedGroupIDs, input.currentAPIKey) {
				releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, input.currentAPIKey)
				return true
			}
			h.submitFailedUsageRecordTask("handler.openai_gateway.embeddings", c, input.currentAPIKey, input.currentSubscription, account, input.req.reqModel, false, forwardDuration, failoverErr, err)
			h.handleFailoverExhausted(c, failoverErr, false)
			return false
		}
		*switchCount = *switchCount + 1
		input.req.reqLog.Warn("openai_embeddings.upstream_failover_switching",
			zap.Int64("account_id", account.ID),
			zap.Int("upstream_status", failoverErr.StatusCode),
			zap.Int("switch_count", *switchCount),
		)
		return true
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
	wroteFallback := h.ensureForwardErrorResponse(c, false)
	h.submitFailedUsageRecordTask("handler.openai_gateway.embeddings", c, input.currentAPIKey, input.currentSubscription, account, input.req.reqModel, false, forwardDuration, nil, err)
	input.req.reqLog.Warn("openai_embeddings.forward_failed",
		zap.Int64("account_id", account.ID),
		zap.Bool("fallback_error_response_written", wroteFallback),
		zap.Error(err),
	)
	return false
}

func (h *OpenAIGatewayHandler) recordEmbeddingsSuccess(
	c *gin.Context,
	input openAIEmbeddingsForwardInput,
	account *service.Account,
	result *service.OpenAIForwardResult,
	switchCount int,
) {
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetTrustedClientIP(c)
	h.submitUsageRecordTask(func(ctx context.Context) {
		ctx = service.AttachPublishedPublicCatalogEntry(ctx, input.req.publicCatalogEntry)
		ctx = reattachGatewayChannelState(ctx, input.channelState)
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result:             result,
			APIKey:             input.currentAPIKey,
			User:               input.currentAPIKey.User,
			Account:            account,
			Subscription:       input.currentSubscription,
			InboundEndpoint:    GetInboundEndpoint(c),
			UpstreamEndpoint:   GetUpstreamEndpointForAccount(c, account),
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: input.req.requestPayloadHash,
			APIKeyService:      h.apiKeyService,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.embeddings"),
				zap.Int64("user_id", input.req.subject.UserID),
				zap.Int64("api_key_id", input.currentAPIKey.ID),
				zap.Any("group_id", input.currentAPIKey.GroupID),
				zap.String("model", input.req.reqModel),
				zap.Int64("account_id", account.ID),
			).Error("openai_embeddings.record_usage_failed", zap.Error(err))
		}
	})
	input.req.reqLog.Debug("openai_embeddings.request_completed", zap.Int64("account_id", account.ID), zap.Int("switch_count", switchCount))
}
