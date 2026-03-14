package service

import (
	"context"
	"errors"
	"fmt"
	coderws "github.com/coder/websocket"
	"net/http"
	"strings"
	"time"
)

type openAIWSFallbackError struct {
	Reason string
	Err    error
}

func (e *openAIWSFallbackError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("openai ws fallback: %s", strings.TrimSpace(e.Reason))
	}
	return fmt.Sprintf("openai ws fallback: %s: %v", strings.TrimSpace(e.Reason), e.Err)
}
func (e *openAIWSFallbackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
func wrapOpenAIWSFallback(reason string, err error) error {
	return &openAIWSFallbackError{Reason: strings.TrimSpace(reason), Err: err}
}

type OpenAIWSClientCloseError struct {
	statusCode coderws.StatusCode
	reason     string
	err        error
}
type openAIWSIngressTurnError struct {
	stage           string
	cause           error
	wroteDownstream bool
}

func (e *openAIWSIngressTurnError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		return strings.TrimSpace(e.stage)
	}
	return e.cause.Error()
}
func (e *openAIWSIngressTurnError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}
func wrapOpenAIWSIngressTurnError(stage string, cause error, wroteDownstream bool) error {
	if cause == nil {
		return nil
	}
	return &openAIWSIngressTurnError{stage: strings.TrimSpace(stage), cause: cause, wroteDownstream: wroteDownstream}
}
func isOpenAIWSIngressTurnRetryable(err error) bool {
	var turnErr *openAIWSIngressTurnError
	if !errors.As(err, &turnErr) || turnErr == nil {
		return false
	}
	if errors.Is(turnErr.cause, context.Canceled) || errors.Is(turnErr.cause, context.DeadlineExceeded) {
		return false
	}
	if turnErr.wroteDownstream {
		return false
	}
	switch turnErr.stage {
	case "write_upstream", "read_upstream":
		return true
	default:
		return false
	}
}
func openAIWSIngressTurnRetryReason(err error) string {
	var turnErr *openAIWSIngressTurnError
	if !errors.As(err, &turnErr) || turnErr == nil {
		return "unknown"
	}
	if turnErr.stage == "" {
		return "unknown"
	}
	return turnErr.stage
}
func isOpenAIWSIngressPreviousResponseNotFound(err error) bool {
	var turnErr *openAIWSIngressTurnError
	if !errors.As(err, &turnErr) || turnErr == nil {
		return false
	}
	if strings.TrimSpace(turnErr.stage) != openAIWSIngressStagePreviousResponseNotFound {
		return false
	}
	return !turnErr.wroteDownstream
}
func NewOpenAIWSClientCloseError(statusCode coderws.StatusCode, reason string, err error) error {
	return &OpenAIWSClientCloseError{statusCode: statusCode, reason: strings.TrimSpace(reason), err: err}
}
func (e *OpenAIWSClientCloseError) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return fmt.Sprintf("openai ws client close: %d %s", int(e.statusCode), strings.TrimSpace(e.reason))
	}
	return fmt.Sprintf("openai ws client close: %d %s: %v", int(e.statusCode), strings.TrimSpace(e.reason), e.err)
}
func (e *OpenAIWSClientCloseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}
func (e *OpenAIWSClientCloseError) StatusCode() coderws.StatusCode {
	if e == nil {
		return coderws.StatusInternalError
	}
	return e.statusCode
}
func (e *OpenAIWSClientCloseError) Reason() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.reason)
}
func classifyOpenAIWSAcquireError(err error) string {
	if err == nil {
		return "acquire_conn"
	}
	var dialErr *openAIWSDialError
	if errors.As(err, &dialErr) {
		switch dialErr.StatusCode {
		case 426:
			return "upgrade_required"
		case 401, 403:
			return "auth_failed"
		case 429:
			return "upstream_rate_limited"
		}
		if dialErr.StatusCode >= 500 {
			return "upstream_5xx"
		}
		return "dial_failed"
	}
	if errors.Is(err, errOpenAIWSConnQueueFull) {
		return "conn_queue_full"
	}
	if errors.Is(err, errOpenAIWSPreferredConnUnavailable) {
		return "preferred_conn_unavailable"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "acquire_timeout"
	}
	return "acquire_conn"
}
func isOpenAIWSRateLimitError(codeRaw, errTypeRaw, msgRaw string) bool {
	code := strings.ToLower(strings.TrimSpace(codeRaw))
	errType := strings.ToLower(strings.TrimSpace(errTypeRaw))
	msg := strings.ToLower(strings.TrimSpace(msgRaw))
	if strings.Contains(errType, "rate_limit") || strings.Contains(errType, "usage_limit") {
		return true
	}
	if strings.Contains(code, "rate_limit") || strings.Contains(code, "usage_limit") || strings.Contains(code, "insufficient_quota") {
		return true
	}
	if strings.Contains(msg, "usage limit") && strings.Contains(msg, "reached") {
		return true
	}
	if strings.Contains(msg, "rate limit") && (strings.Contains(msg, "reached") || strings.Contains(msg, "exceeded")) {
		return true
	}
	return false
}
func (s *OpenAIGatewayService) persistOpenAIWSRateLimitSignal(ctx context.Context, account *Account, headers http.Header, responseBody []byte, codeRaw, errTypeRaw, msgRaw string) {
	if s == nil || s.rateLimitService == nil || account == nil || account.Platform != PlatformOpenAI {
		return
	}
	if !isOpenAIWSRateLimitError(codeRaw, errTypeRaw, msgRaw) {
		return
	}
	s.rateLimitService.HandleUpstreamError(ctx, account, http.StatusTooManyRequests, headers, responseBody)
}
func classifyOpenAIWSErrorEventFromRaw(codeRaw, errTypeRaw, msgRaw string) (string, bool) {
	code := strings.ToLower(strings.TrimSpace(codeRaw))
	errType := strings.ToLower(strings.TrimSpace(errTypeRaw))
	msg := strings.ToLower(strings.TrimSpace(msgRaw))
	switch code {
	case "upgrade_required":
		return "upgrade_required", true
	case "websocket_not_supported", "websocket_unsupported":
		return "ws_unsupported", true
	case "websocket_connection_limit_reached":
		return "ws_connection_limit_reached", true
	case "invalid_encrypted_content":
		return "invalid_encrypted_content", true
	case "previous_response_not_found":
		return "previous_response_not_found", true
	}
	if isOpenAIWSRateLimitError(codeRaw, errTypeRaw, msgRaw) {
		return "upstream_rate_limited", false
	}
	if strings.Contains(msg, "upgrade required") || strings.Contains(msg, "status 426") {
		return "upgrade_required", true
	}
	if strings.Contains(errType, "upgrade") {
		return "upgrade_required", true
	}
	if strings.Contains(msg, "websocket") && strings.Contains(msg, "unsupported") {
		return "ws_unsupported", true
	}
	if strings.Contains(msg, "connection limit") && strings.Contains(msg, "websocket") {
		return "ws_connection_limit_reached", true
	}
	if strings.Contains(msg, "invalid_encrypted_content") || (strings.Contains(msg, "encrypted content") && strings.Contains(msg, "could not be verified")) {
		return "invalid_encrypted_content", true
	}
	if strings.Contains(msg, "previous_response_not_found") || (strings.Contains(msg, "previous response") && strings.Contains(msg, "not found")) {
		return "previous_response_not_found", true
	}
	if strings.Contains(errType, "server_error") || strings.Contains(code, "server_error") {
		return "upstream_error_event", true
	}
	return "event_error", false
}
func classifyOpenAIWSErrorEvent(message []byte) (string, bool) {
	if len(message) == 0 {
		return "event_error", false
	}
	return classifyOpenAIWSErrorEventFromRaw(parseOpenAIWSErrorEventFields(message))
}
func openAIWSErrorHTTPStatusFromRaw(codeRaw, errTypeRaw string) int {
	code := strings.ToLower(strings.TrimSpace(codeRaw))
	errType := strings.ToLower(strings.TrimSpace(errTypeRaw))
	switch {
	case strings.Contains(errType, "invalid_request"), strings.Contains(code, "invalid_request"), strings.Contains(code, "bad_request"), code == "invalid_encrypted_content", code == "previous_response_not_found":
		return http.StatusBadRequest
	case strings.Contains(errType, "authentication"), strings.Contains(code, "invalid_api_key"), strings.Contains(code, "unauthorized"):
		return http.StatusUnauthorized
	case strings.Contains(errType, "permission"), strings.Contains(code, "forbidden"):
		return http.StatusForbidden
	case isOpenAIWSRateLimitError(codeRaw, errTypeRaw, ""):
		return http.StatusTooManyRequests
	default:
		return http.StatusBadGateway
	}
}
func openAIWSErrorHTTPStatus(message []byte) int {
	if len(message) == 0 {
		return http.StatusBadGateway
	}
	codeRaw, errTypeRaw, _ := parseOpenAIWSErrorEventFields(message)
	return openAIWSErrorHTTPStatusFromRaw(codeRaw, errTypeRaw)
}
func (s *OpenAIGatewayService) openAIWSFallbackCooldown() time.Duration {
	if s == nil || s.cfg == nil {
		return 30 * time.Second
	}
	seconds := s.cfg.Gateway.OpenAIWS.FallbackCooldownSeconds
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
func (s *OpenAIGatewayService) isOpenAIWSFallbackCooling(accountID int64) bool {
	if s == nil || accountID <= 0 {
		return false
	}
	cooldown := s.openAIWSFallbackCooldown()
	if cooldown <= 0 {
		return false
	}
	rawUntil, ok := s.openaiWSFallbackUntil.Load(accountID)
	if !ok || rawUntil == nil {
		return false
	}
	until, ok := rawUntil.(time.Time)
	if !ok || until.IsZero() {
		s.openaiWSFallbackUntil.Delete(accountID)
		return false
	}
	if time.Now().Before(until) {
		return true
	}
	s.openaiWSFallbackUntil.Delete(accountID)
	return false
}
func (s *OpenAIGatewayService) markOpenAIWSFallbackCooling(accountID int64, _ string) {
	if s == nil || accountID <= 0 {
		return
	}
	cooldown := s.openAIWSFallbackCooldown()
	if cooldown <= 0 {
		return
	}
	s.openaiWSFallbackUntil.Store(accountID, time.Now().Add(cooldown))
}
func (s *OpenAIGatewayService) clearOpenAIWSFallbackCooling(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiWSFallbackUntil.Delete(accountID)
}
