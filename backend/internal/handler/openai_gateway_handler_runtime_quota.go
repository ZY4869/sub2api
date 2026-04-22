package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

const openAIRuntimeQuotaCooldownMessage = "Requested model is temporarily unavailable because the relevant OpenAI Pro quota is cooling down, please retry later"

func (h *OpenAIGatewayHandler) isOpenAIRuntimeQuotaOnlySelectionFailure(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
) bool {
	if h == nil || h.gatewayService == nil {
		return false
	}
	return h.gatewayService.IsModelUnavailableDueToRuntimeQuota(ctx, groupID, requestedModel, nil)
}

func (h *OpenAIGatewayHandler) handleOpenAIRuntimeQuotaUnavailable(c *gin.Context, streamStarted bool) {
	h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", openAIRuntimeQuotaCooldownMessage, streamStarted)
}

func (h *OpenAIGatewayHandler) handleAnthropicOpenAIRuntimeQuotaUnavailable(c *gin.Context, streamStarted bool) {
	h.anthropicStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", openAIRuntimeQuotaCooldownMessage, streamStarted)
}
