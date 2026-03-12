package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

func (s *AntigravityGatewayService) getLogConfig() (logBody bool, maxBytes int) {
	maxBytes = 2048
	if s.settingService == nil || s.settingService.cfg == nil {
		return false, maxBytes
	}
	cfg := s.settingService.cfg.Gateway
	if cfg.LogUpstreamErrorBodyMaxBytes > 0 {
		maxBytes = cfg.LogUpstreamErrorBodyMaxBytes
	}
	return cfg.LogUpstreamErrorBody, maxBytes
}
func (s *AntigravityGatewayService) getUpstreamErrorDetail(body []byte) string {
	logBody, maxBytes := s.getLogConfig()
	if !logBody {
		return ""
	}
	return truncateString(string(body), maxBytes)
}
func (s *AntigravityGatewayService) checkErrorPolicy(ctx context.Context, account *Account, statusCode int, body []byte) ErrorPolicyResult {
	if s.rateLimitService == nil {
		return ErrorPolicyNone
	}
	return s.rateLimitService.CheckErrorPolicy(ctx, account, statusCode, body)
}
func (s *AntigravityGatewayService) applyErrorPolicy(p antigravityRetryLoopParams, statusCode int, headers http.Header, respBody []byte) (handled bool, outStatus int, retErr error) {
	switch s.checkErrorPolicy(p.ctx, p.account, statusCode, respBody) {
	case ErrorPolicySkipped:
		return true, http.StatusInternalServerError, nil
	case ErrorPolicyMatched:
		_ = p.handleError(p.ctx, p.prefix, p.account, statusCode, headers, respBody, p.requestedModel, p.groupID, p.sessionHash, p.isStickySession)
		return true, statusCode, nil
	case ErrorPolicyTempUnscheduled:
		slog.Info("temp_unschedulable_matched", "prefix", p.prefix, "status_code", statusCode, "account_id", p.account.ID)
		return true, statusCode, &AntigravityAccountSwitchError{OriginalAccountID: p.account.ID, IsStickySession: p.isStickySession}
	}
	return false, statusCode, nil
}
func isSignatureRelatedError(respBody []byte) bool {
	msg := strings.ToLower(strings.TrimSpace(extractAntigravityErrorMessage(respBody)))
	if msg == "" {
		msg = strings.ToLower(string(respBody))
	}
	if strings.Contains(msg, "thought_signature") || strings.Contains(msg, "signature") {
		return true
	}
	if strings.Contains(msg, "expected") && (strings.Contains(msg, "thinking") || strings.Contains(msg, "redacted_thinking")) {
		return true
	}
	return false
}
func isPromptTooLongError(respBody []byte) bool {
	msg := strings.ToLower(strings.TrimSpace(extractAntigravityErrorMessage(respBody)))
	if msg == "" {
		msg = strings.ToLower(string(respBody))
	}
	return strings.Contains(msg, "prompt is too long") || strings.Contains(msg, "request is too long") || strings.Contains(msg, "context length exceeded") || strings.Contains(msg, "max_tokens")
}
func isPassthroughErrorMessage(msg string) bool {
	lower := strings.ToLower(msg)
	for _, pattern := range antigravityPassthroughErrorMessages {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
func getPassthroughOrDefault(upstreamMsg, defaultMsg string) string {
	if isPassthroughErrorMessage(upstreamMsg) {
		return upstreamMsg
	}
	return defaultMsg
}
func extractAntigravityErrorMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if errObj, ok := payload["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok && strings.TrimSpace(msg) != "" {
			return msg
		}
	}
	if msg, ok := payload["message"].(string); ok && strings.TrimSpace(msg) != "" {
		return msg
	}
	return ""
}
