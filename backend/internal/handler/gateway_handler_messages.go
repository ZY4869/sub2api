package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func forwardFailedLogFields(account *service.Account, wroteFallback bool, err error) []zap.Field {
	fields := []zap.Field{
		zap.Bool("fallback_error_response_written", wroteFallback),
		zap.Error(err),
	}
	if account == nil {
		return fields
	}
	fields = append(fields,
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.String("account_platform", account.Platform),
	)
	if account.Proxy != nil {
		fields = append(fields,
			zap.Int64("proxy_id", account.Proxy.ID),
			zap.String("proxy_name", account.Proxy.Name),
			zap.String("proxy_host", account.Proxy.Host),
			zap.Int("proxy_port", account.Proxy.Port),
		)
	} else if account.ProxyID != nil {
		fields = append(fields, zap.Int64p("proxy_id", account.ProxyID))
	}
	return fields
}

func (h *GatewayHandler) selectionAccountOrFail(
	c *gin.Context,
	selection *service.AccountSelectionResult,
	streamStarted bool,
) (*service.Account, bool) {
	if selection == nil || selection.Account == nil {
		h.handleStreamingAwareError(c, http.StatusBadGateway, "api_error", "No available accounts", streamStarted)
		return nil, false
	}
	return selection.Account, true
}

func (h *GatewayHandler) Messages(c *gin.Context) {
	req, ok := h.beginGatewayMessagesRequest(c)
	if !ok {
		return
	}
	defer h.maybeLogCompatibilityFallbackMetrics(req.reqLog)
	defer h.releaseGatewayMessagesUserWait(c, req)

	if !h.prepareGatewayMessagesRequest(c, req) {
		return
	}
	if req.userReleaseFunc != nil {
		defer req.userReleaseFunc()
	}

	h.runGatewayMessages(c, req)
}
