package handler

import (
	"context"
	"errors"
	"fmt"
	pkgerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *GatewayHandler) handleConcurrencyError(c *gin.Context, err error, slotType string, streamStarted bool) {
	h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", fmt.Sprintf("Concurrency limit exceeded for %s, please retry later", slotType), streamStarted)
}
func (h *GatewayHandler) handleFailoverExhausted(c *gin.Context, failoverErr *service.UpstreamFailoverError, platform string, streamStarted bool) {
	statusCode := failoverErr.StatusCode
	responseBody := failoverErr.ResponseBody
	if h.errorPassthroughService != nil && len(responseBody) > 0 {
		if rule := h.errorPassthroughService.MatchRule(platform, statusCode, responseBody); rule != nil {
			respCode := statusCode
			if !rule.PassthroughCode && rule.ResponseCode != nil {
				respCode = *rule.ResponseCode
			}
			msg := service.ExtractUpstreamErrorMessage(responseBody)
			if !rule.PassthroughBody && rule.CustomMessage != nil {
				msg = *rule.CustomMessage
			}
			if rule.SkipMonitoring {
				c.Set(service.OpsSkipPassthroughKey, true)
			}
			h.handleStreamingAwareError(c, respCode, "upstream_error", msg, streamStarted)
			return
		}
	}
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	h.handleStreamingAwareError(c, status, errType, errMsg, streamStarted)
}
func (h *GatewayHandler) handleFailoverExhaustedSimple(c *gin.Context, statusCode int, streamStarted bool) {
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	h.handleStreamingAwareError(c, status, errType, errMsg, streamStarted)
}
func (h *GatewayHandler) mapUpstreamError(statusCode int) (int, string, string) {
	switch statusCode {
	case 401:
		return http.StatusBadGateway, "upstream_error", "Upstream authentication failed, please contact administrator"
	case 403:
		return http.StatusBadGateway, "upstream_error", "Upstream access forbidden, please contact administrator"
	case 429:
		return http.StatusTooManyRequests, "rate_limit_error", "Upstream rate limit exceeded, please retry later"
	case 529:
		return http.StatusServiceUnavailable, "overloaded_error", "Upstream service overloaded, please retry later"
	case 500, 502, 503, 504:
		return http.StatusBadGateway, "upstream_error", "Upstream service temporarily unavailable"
	default:
		return http.StatusBadGateway, "upstream_error", "Upstream request failed"
	}
}
func (h *GatewayHandler) handleStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	if streamStarted {
		flusher, ok := c.Writer.(http.Flusher)
		if ok {
			errorEvent := `data: {"type":"error","error":{"type":` + strconv.Quote(errType) + `,"message":` + strconv.Quote(message) + `}}` + "\n\n"
			if _, err := fmt.Fprint(c.Writer, errorEvent); err != nil {
				_ = c.Error(err)
			}
			flusher.Flush()
		}
		return
	}
	h.errorResponse(c, status, errType, message)
}
func (h *GatewayHandler) ensureForwardErrorResponse(c *gin.Context, streamStarted bool) bool {
	if c == nil || c.Writer == nil || c.Writer.Written() {
		return false
	}
	h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", streamStarted)
	return true
}
func (h *GatewayHandler) checkClaudeCodeVersion(c *gin.Context) bool {
	ctx := c.Request.Context()
	if !service.IsClaudeCodeClient(ctx) {
		return true
	}
	if strings.HasSuffix(c.Request.URL.Path, "/count_tokens") {
		return true
	}
	minVersion := h.settingService.GetMinClaudeCodeVersion(ctx)
	if minVersion == "" {
		return true
	}
	clientVersion := service.GetClaudeCodeVersion(ctx)
	if clientVersion == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Unable to determine Claude Code version. Please update Claude Code: npm update -g @anthropic-ai/claude-code")
		return false
	}
	if service.CompareVersions(clientVersion, minVersion) < 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("Your Claude Code version (%s) is below the minimum required version (%s). Please update: npm update -g @anthropic-ai/claude-code", clientVersion, minVersion))
		return false
	}
	return true
}
func (h *GatewayHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": errType, "message": message}})
}
func billingErrorDetails(err error) (status int, code, message string) {
	if errors.Is(err, service.ErrBillingServiceUnavailable) {
		msg := pkgerrors.Message(err)
		if msg == "" {
			msg = "Billing service temporarily unavailable. Please retry later."
		}
		return http.StatusServiceUnavailable, "billing_service_error", msg
	}
	if errors.Is(err, service.ErrAPIKeyRateLimit5hExceeded) {
		msg := pkgerrors.Message(err)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg
	}
	if errors.Is(err, service.ErrAPIKeyRateLimit1dExceeded) {
		msg := pkgerrors.Message(err)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg
	}
	if errors.Is(err, service.ErrAPIKeyRateLimit7dExceeded) {
		msg := pkgerrors.Message(err)
		return http.StatusTooManyRequests, "rate_limit_exceeded", msg
	}
	msg := pkgerrors.Message(err)
	if msg == "" {
		logger.L().With(zap.String("component", "handler.gateway.billing"), zap.Error(err)).Warn("gateway.billing_error_missing_message")
		msg = "Billing error"
	}
	return http.StatusForbidden, "billing_error", msg
}
func (h *GatewayHandler) metadataBridgeEnabled() bool {
	if h == nil || h.cfg == nil {
		return true
	}
	return h.cfg.Gateway.OpenAIWS.MetadataBridgeEnabled
}
func (h *GatewayHandler) maybeLogCompatibilityFallbackMetrics(reqLog *zap.Logger) {
	if reqLog == nil {
		return
	}
	if gatewayCompatibilityMetricsLogCounter.Add(1)%gatewayCompatibilityMetricsLogInterval != 0 {
		return
	}
	metrics := service.SnapshotOpenAICompatibilityFallbackMetrics()
	reqLog.Info("gateway.compatibility_fallback_metrics", zap.Int64("session_hash_legacy_read_fallback_total", metrics.SessionHashLegacyReadFallbackTotal), zap.Int64("session_hash_legacy_read_fallback_hit", metrics.SessionHashLegacyReadFallbackHit), zap.Int64("session_hash_legacy_dual_write_total", metrics.SessionHashLegacyDualWriteTotal), zap.Float64("session_hash_legacy_read_hit_rate", metrics.SessionHashLegacyReadHitRate), zap.Int64("metadata_legacy_fallback_total", metrics.MetadataLegacyFallbackTotal))
}
func (h *GatewayHandler) submitUsageRecordTask(task service.UsageRecordTask) {
	if task == nil {
		return
	}
	if h.usageRecordWorkerPool != nil {
		h.usageRecordWorkerPool.Submit(task)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().With(zap.String("component", "handler.gateway.messages"), zap.Any("panic", recovered)).Error("gateway.usage_record_task_panic_recovered")
		}
	}()
	task(ctx)
}
