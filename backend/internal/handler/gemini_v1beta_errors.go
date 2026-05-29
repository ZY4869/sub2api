package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) handleGeminiFailoverExhausted(c *gin.Context, failoverErr *service.UpstreamFailoverError) {
	if failoverErr == nil {
		googleErrorKey(c, http.StatusBadGateway, "gateway.gemini.upstream_failed", "Upstream request failed")
		return
	}

	statusCode := failoverErr.StatusCode
	responseBody := failoverErr.ResponseBody

	if h.errorPassthroughService != nil && len(responseBody) > 0 {
		if rule := h.errorPassthroughService.MatchRule(service.PlatformGemini, statusCode, responseBody); rule != nil {
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

			googleError(c, respCode, msg)
			return
		}
	}

	upstreamMsg := service.ExtractUpstreamErrorMessage(responseBody)
	service.SetOpsUpstreamError(c, statusCode, upstreamMsg, "")

	status, messageKey, message := mapGeminiUpstreamError(statusCode)
	googleErrorKey(c, status, messageKey, message)
}

func mapGeminiUpstreamError(statusCode int) (int, string, string) {
	switch statusCode {
	case 401:
		return http.StatusBadGateway, "gateway.gemini.upstream_auth_failed", "Upstream authentication failed, please contact administrator"
	case 403:
		return http.StatusBadGateway, "gateway.gemini.upstream_forbidden", "Upstream access forbidden, please contact administrator"
	case 429:
		return http.StatusTooManyRequests, "gateway.gemini.upstream_rate_limited", "Upstream rate limit exceeded, please retry later"
	case 529:
		return http.StatusServiceUnavailable, "gateway.gemini.upstream_overloaded", "Upstream service overloaded, please retry later"
	case 500, 502, 503, 504:
		return http.StatusBadGateway, "gateway.gemini.upstream_unavailable", "Upstream service temporarily unavailable"
	default:
		return http.StatusBadGateway, "gateway.gemini.upstream_failed", "Upstream request failed"
	}
}

const googleRPCTypeErrorInfo = "type.googleapis.com/google.rpc.ErrorInfo"

func googleError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
}

func googleErrorKey(c *gin.Context, status int, messageKey string, fallback string, args ...any) {
	googleErrorWithReason(c, status, "", messageKey, fallback, args...)
}

func googleErrorWithReason(c *gin.Context, status int, reason string, messageKey string, fallback string, args ...any) {
	payload := gin.H{
		"code":    status,
		"message": response.LocalizedMessage(c, messageKey, fallback, args...),
		"status":  googleapi.HTTPStatusToGoogleStatus(status),
	}
	if strings.TrimSpace(reason) != "" {
		payload["details"] = []gin.H{
			{
				"@type":  googleRPCTypeErrorInfo,
				"reason": reason,
			},
		}
	}
	c.JSON(status, gin.H{"error": payload})
}

func isGeminiModelOperationsPath(modelPath string) bool {
	trimmed := strings.Trim(strings.TrimSpace(modelPath), "/")
	if trimmed == "" {
		return false
	}
	return strings.HasSuffix(trimmed, "/operations") || strings.Contains(trimmed, "/operations/")
}

func googleErrorFromDecision(c *gin.Context, decision service.ProtocolCapabilityDecision) {
	fallback := "%s is not supported for this platform"
	if decision.Reason == service.GatewayReasonUnsupportedAction {
		fallback = "%s does not support this action on the current route"
	}
	googleErrorWithReason(c, decision.StatusCode, decision.Reason, decision.MessageKey, fallback, decision.RequestFormat)
}

func writeUpstreamResponse(c *gin.Context, res *service.UpstreamHTTPResult) {
	if res == nil {
		googleErrorKey(c, http.StatusBadGateway, "gateway.gemini.upstream_empty", "Empty upstream response")
		return
	}
	for k, vv := range res.Headers {
		if strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") {
			continue
		}
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}
	contentType := res.Headers.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(res.StatusCode, contentType, res.Body)
}
