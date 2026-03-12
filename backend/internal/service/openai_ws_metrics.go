package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	coderws "github.com/coder/websocket"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net"
	"sort"
	"strings"
	"time"
)

func logOpenAIWSModeInfo(format string, args ...any) {
	logger.LegacyPrintf("service.openai_gateway", "[OpenAI WS Mode][openai_ws_mode=true] "+format, args...)
}
func isOpenAIWSModeDebugEnabled() bool {
	return logger.L().Core().Enabled(zap.DebugLevel)
}
func logOpenAIWSModeDebug(format string, args ...any) {
	if !isOpenAIWSModeDebugEnabled() {
		return
	}
	logger.LegacyPrintf("service.openai_gateway", "[debug] [OpenAI WS Mode][openai_ws_mode=true] "+format, args...)
}
func logOpenAIWSBindResponseAccountWarn(groupID, accountID int64, responseID string, err error) {
	if err == nil {
		return
	}
	logger.L().Warn("openai.ws_bind_response_account_failed", zap.Int64("group_id", groupID), zap.Int64("account_id", accountID), zap.String("response_id", truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen)), zap.Error(err))
}
func summarizeOpenAIWSReadCloseError(err error) (status string, reason string) {
	if err == nil {
		return "-", "-"
	}
	statusCode := coderws.CloseStatus(err)
	if statusCode == -1 {
		return "-", "-"
	}
	closeStatus := fmt.Sprintf("%d(%s)", int(statusCode), statusCode.String())
	closeReason := "-"
	var closeErr coderws.CloseError
	if errors.As(err, &closeErr) {
		reasonText := strings.TrimSpace(closeErr.Reason)
		if reasonText != "" {
			closeReason = normalizeOpenAIWSLogValue(reasonText)
		}
	}
	return normalizeOpenAIWSLogValue(closeStatus), closeReason
}
func unwrapOpenAIWSDialBaseError(err error) error {
	if err == nil {
		return nil
	}
	var dialErr *openAIWSDialError
	if errors.As(err, &dialErr) && dialErr != nil && dialErr.Err != nil {
		return dialErr.Err
	}
	return err
}
func openAIWSDialRespHeaderForLog(err error, key string) string {
	var dialErr *openAIWSDialError
	if !errors.As(err, &dialErr) || dialErr == nil || dialErr.ResponseHeaders == nil {
		return "-"
	}
	return truncateOpenAIWSLogValue(dialErr.ResponseHeaders.Get(key), openAIWSHeaderValueMaxLen)
}
func classifyOpenAIWSDialError(err error) string {
	if err == nil {
		return "-"
	}
	baseErr := unwrapOpenAIWSDialBaseError(err)
	if baseErr == nil {
		return "-"
	}
	if errors.Is(baseErr, context.DeadlineExceeded) {
		return "ctx_deadline_exceeded"
	}
	if errors.Is(baseErr, context.Canceled) {
		return "ctx_canceled"
	}
	var netErr net.Error
	if errors.As(baseErr, &netErr) && netErr.Timeout() {
		return "net_timeout"
	}
	if status := coderws.CloseStatus(baseErr); status != -1 {
		return normalizeOpenAIWSLogValue(fmt.Sprintf("ws_close_%d", int(status)))
	}
	message := strings.ToLower(strings.TrimSpace(baseErr.Error()))
	switch {
	case strings.Contains(message, "handshake not finished"):
		return "handshake_not_finished"
	case strings.Contains(message, "bad handshake"):
		return "bad_handshake"
	case strings.Contains(message, "connection refused"):
		return "connection_refused"
	case strings.Contains(message, "no such host"):
		return "dns_not_found"
	case strings.Contains(message, "tls"):
		return "tls_error"
	case strings.Contains(message, "i/o timeout"):
		return "io_timeout"
	case strings.Contains(message, "context deadline exceeded"):
		return "ctx_deadline_exceeded"
	default:
		return "dial_error"
	}
}
func summarizeOpenAIWSDialError(err error) (statusCode int, dialClass string, closeStatus string, closeReason string, respServer string, respVia string, respCFRay string, respRequestID string) {
	dialClass = "-"
	closeStatus = "-"
	closeReason = "-"
	respServer = "-"
	respVia = "-"
	respCFRay = "-"
	respRequestID = "-"
	if err == nil {
		return
	}
	var dialErr *openAIWSDialError
	if errors.As(err, &dialErr) && dialErr != nil {
		statusCode = dialErr.StatusCode
		respServer = openAIWSDialRespHeaderForLog(err, "server")
		respVia = openAIWSDialRespHeaderForLog(err, "via")
		respCFRay = openAIWSDialRespHeaderForLog(err, "cf-ray")
		respRequestID = openAIWSDialRespHeaderForLog(err, "x-request-id")
	}
	dialClass = normalizeOpenAIWSLogValue(classifyOpenAIWSDialError(err))
	closeStatus, closeReason = summarizeOpenAIWSReadCloseError(unwrapOpenAIWSDialBaseError(err))
	return
}
func isOpenAIWSClientDisconnectError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, context.Canceled) {
		return true
	}
	switch coderws.CloseStatus(err) {
	case coderws.StatusNormalClosure, coderws.StatusGoingAway, coderws.StatusNoStatusRcvd, coderws.StatusAbnormalClosure:
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}
	return strings.Contains(message, "failed to read frame header: eof") || strings.Contains(message, "unexpected eof") || strings.Contains(message, "use of closed network connection") || strings.Contains(message, "connection reset by peer") || strings.Contains(message, "broken pipe") || strings.Contains(message, "an established connection was aborted")
}
func classifyOpenAIWSReadFallbackReason(err error) string {
	if err == nil {
		return "read_event"
	}
	switch coderws.CloseStatus(err) {
	case coderws.StatusPolicyViolation:
		return "policy_violation"
	case coderws.StatusMessageTooBig:
		return "message_too_big"
	default:
		return "read_event"
	}
}
func sortedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (s *OpenAIGatewayService) getOpenAIWSConnPool() *openAIWSConnPool {
	if s == nil {
		return nil
	}
	s.openaiWSPoolOnce.Do(func() {
		if s.openaiWSPool == nil {
			s.openaiWSPool = newOpenAIWSConnPool(s.cfg)
		}
	})
	return s.openaiWSPool
}
func (s *OpenAIGatewayService) getOpenAIWSPassthroughDialer() openAIWSClientDialer {
	if s == nil {
		return nil
	}
	s.openaiWSPassthroughDialerOnce.Do(func() {
		if s.openaiWSPassthroughDialer == nil {
			s.openaiWSPassthroughDialer = newDefaultOpenAIWSClientDialer()
		}
	})
	return s.openaiWSPassthroughDialer
}
func (s *OpenAIGatewayService) SnapshotOpenAIWSPoolMetrics() OpenAIWSPoolMetricsSnapshot {
	pool := s.getOpenAIWSConnPool()
	if pool == nil {
		return OpenAIWSPoolMetricsSnapshot{}
	}
	return pool.SnapshotMetrics()
}

type OpenAIWSPerformanceMetricsSnapshot struct {
	Pool      OpenAIWSPoolMetricsSnapshot      `json:"pool"`
	Retry     OpenAIWSRetryMetricsSnapshot     `json:"retry"`
	Transport OpenAIWSTransportMetricsSnapshot `json:"transport"`
}

func (s *OpenAIGatewayService) SnapshotOpenAIWSPerformanceMetrics() OpenAIWSPerformanceMetricsSnapshot {
	pool := s.getOpenAIWSConnPool()
	snapshot := OpenAIWSPerformanceMetricsSnapshot{Retry: s.SnapshotOpenAIWSRetryMetrics()}
	if pool == nil {
		return snapshot
	}
	snapshot.Pool = pool.SnapshotMetrics()
	snapshot.Transport = pool.SnapshotTransportMetrics()
	return snapshot
}
func (s *OpenAIGatewayService) getOpenAIWSStateStore() OpenAIWSStateStore {
	if s == nil {
		return nil
	}
	s.openaiWSStateStoreOnce.Do(func() {
		if s.openaiWSStateStore == nil {
			s.openaiWSStateStore = NewOpenAIWSStateStore(s.cache)
		}
	})
	return s.openaiWSStateStore
}
func (s *OpenAIGatewayService) openAIWSResponseStickyTTL() time.Duration {
	if s != nil && s.cfg != nil {
		seconds := s.cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds
		if seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return time.Hour
}
func (s *OpenAIGatewayService) openAIWSIngressPreviousResponseRecoveryEnabled() bool {
	if s != nil && s.cfg != nil {
		return s.cfg.Gateway.OpenAIWS.IngressPreviousResponseRecoveryEnabled
	}
	return true
}
func (s *OpenAIGatewayService) openAIWSReadTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.ReadTimeoutSeconds > 0 {
		return time.Duration(s.cfg.Gateway.OpenAIWS.ReadTimeoutSeconds) * time.Second
	}
	return 15 * time.Minute
}
func (s *OpenAIGatewayService) openAIWSPassthroughIdleTimeout() time.Duration {
	if timeout := s.openAIWSReadTimeout(); timeout > 0 {
		return timeout
	}
	return openAIWSPassthroughIdleTimeoutDefault
}
func (s *OpenAIGatewayService) openAIWSWriteTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.WriteTimeoutSeconds > 0 {
		return time.Duration(s.cfg.Gateway.OpenAIWS.WriteTimeoutSeconds) * time.Second
	}
	return 2 * time.Minute
}
func (s *OpenAIGatewayService) openAIWSEventFlushBatchSize() int {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.EventFlushBatchSize > 0 {
		return s.cfg.Gateway.OpenAIWS.EventFlushBatchSize
	}
	return openAIWSEventFlushBatchSizeDefault
}
func (s *OpenAIGatewayService) openAIWSEventFlushInterval() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.EventFlushIntervalMS >= 0 {
		if s.cfg.Gateway.OpenAIWS.EventFlushIntervalMS == 0 {
			return 0
		}
		return time.Duration(s.cfg.Gateway.OpenAIWS.EventFlushIntervalMS) * time.Millisecond
	}
	return openAIWSEventFlushIntervalDefault
}
func (s *OpenAIGatewayService) openAIWSPayloadLogSampleRate() float64 {
	if s != nil && s.cfg != nil {
		rate := s.cfg.Gateway.OpenAIWS.PayloadLogSampleRate
		if rate < 0 {
			return 0
		}
		if rate > 1 {
			return 1
		}
		return rate
	}
	return openAIWSPayloadLogSampleDefault
}
func (s *OpenAIGatewayService) shouldLogOpenAIWSPayloadSchema(attempt int) bool {
	if attempt <= 1 {
		return true
	}
	rate := s.openAIWSPayloadLogSampleRate()
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	return rand.Float64() < rate
}
func (s *OpenAIGatewayService) shouldEmitOpenAIWSPayloadSchema(attempt int) bool {
	if !s.shouldLogOpenAIWSPayloadSchema(attempt) {
		return false
	}
	return logger.L().Core().Enabled(zap.DebugLevel)
}
func (s *OpenAIGatewayService) openAIWSDialTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.DialTimeoutSeconds > 0 {
		return time.Duration(s.cfg.Gateway.OpenAIWS.DialTimeoutSeconds) * time.Second
	}
	return 10 * time.Second
}
func (s *OpenAIGatewayService) openAIWSAcquireTimeout() time.Duration {
	dial := s.openAIWSDialTimeout()
	if dial <= 0 {
		dial = 10 * time.Second
	}
	return dial + 2*time.Second
}
func (s *OpenAIGatewayService) isOpenAIWSGeneratePrewarmEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.PrewarmGenerateEnabled
}
func (s *OpenAIGatewayService) performOpenAIWSGeneratePrewarm(ctx context.Context, lease *openAIWSConnLease, decision OpenAIWSProtocolDecision, payload map[string]any, previousResponseID string, reqBody map[string]any, account *Account, stateStore OpenAIWSStateStore, groupID int64) error {
	if s == nil {
		return nil
	}
	if lease == nil || account == nil {
		logOpenAIWSModeInfo("prewarm_skip reason=invalid_state has_lease=%v has_account=%v", lease != nil, account != nil)
		return nil
	}
	connID := strings.TrimSpace(lease.ConnID())
	if !s.isOpenAIWSGeneratePrewarmEnabled() {
		return nil
	}
	if decision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
		logOpenAIWSModeInfo("prewarm_skip account_id=%d conn_id=%s reason=transport_not_v2 transport=%s", account.ID, connID, normalizeOpenAIWSLogValue(string(decision.Transport)))
		return nil
	}
	if strings.TrimSpace(previousResponseID) != "" {
		logOpenAIWSModeInfo("prewarm_skip account_id=%d conn_id=%s reason=has_previous_response_id previous_response_id=%s", account.ID, connID, truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen))
		return nil
	}
	if lease.IsPrewarmed() {
		logOpenAIWSModeInfo("prewarm_skip account_id=%d conn_id=%s reason=already_prewarmed", account.ID, connID)
		return nil
	}
	if NeedsToolContinuation(reqBody) {
		logOpenAIWSModeInfo("prewarm_skip account_id=%d conn_id=%s reason=tool_continuation", account.ID, connID)
		return nil
	}
	prewarmStart := time.Now()
	logOpenAIWSModeInfo("prewarm_start account_id=%d conn_id=%s", account.ID, connID)
	prewarmPayload := make(map[string]any, len(payload)+1)
	for k, v := range payload {
		prewarmPayload[k] = v
	}
	prewarmPayload["generate"] = false
	prewarmPayloadJSON := payloadAsJSONBytes(prewarmPayload)
	if err := lease.WriteJSONWithContextTimeout(ctx, prewarmPayload, s.openAIWSWriteTimeout()); err != nil {
		lease.MarkBroken()
		logOpenAIWSModeInfo("prewarm_write_fail account_id=%d conn_id=%s cause=%s", account.ID, connID, truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen))
		return wrapOpenAIWSFallback("prewarm_write", err)
	}
	logOpenAIWSModeInfo("prewarm_write_sent account_id=%d conn_id=%s payload_bytes=%d", account.ID, connID, len(prewarmPayloadJSON))
	prewarmResponseID := ""
	prewarmEventCount := 0
	prewarmTerminalCount := 0
	for {
		message, readErr := lease.ReadMessageWithContextTimeout(ctx, s.openAIWSReadTimeout())
		if readErr != nil {
			lease.MarkBroken()
			closeStatus, closeReason := summarizeOpenAIWSReadCloseError(readErr)
			logOpenAIWSModeInfo("prewarm_read_fail account_id=%d conn_id=%s close_status=%s close_reason=%s cause=%s events=%d", account.ID, connID, closeStatus, closeReason, truncateOpenAIWSLogValue(readErr.Error(), openAIWSLogValueMaxLen), prewarmEventCount)
			return wrapOpenAIWSFallback("prewarm_"+classifyOpenAIWSReadFallbackReason(readErr), readErr)
		}
		eventType, eventResponseID, _ := parseOpenAIWSEventEnvelope(message)
		if eventType == "" {
			continue
		}
		prewarmEventCount++
		if prewarmResponseID == "" && eventResponseID != "" {
			prewarmResponseID = eventResponseID
		}
		if prewarmEventCount <= openAIWSPrewarmEventLogHead || eventType == "error" || isOpenAIWSTerminalEvent(eventType) {
			logOpenAIWSModeInfo("prewarm_event account_id=%d conn_id=%s idx=%d type=%s bytes=%d", account.ID, connID, prewarmEventCount, truncateOpenAIWSLogValue(eventType, openAIWSLogValueMaxLen), len(message))
		}
		if eventType == "error" {
			errCodeRaw, errTypeRaw, errMsgRaw := parseOpenAIWSErrorEventFields(message)
			s.persistOpenAIWSRateLimitSignal(ctx, account, lease.HandshakeHeaders(), message, errCodeRaw, errTypeRaw, errMsgRaw)
			errMsg := strings.TrimSpace(errMsgRaw)
			if errMsg == "" {
				errMsg = "OpenAI websocket prewarm error"
			}
			fallbackReason, canFallback := classifyOpenAIWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			errCode, errType, errMessage := summarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			logOpenAIWSModeInfo("prewarm_error_event account_id=%d conn_id=%s idx=%d fallback_reason=%s can_fallback=%v err_code=%s err_type=%s err_message=%s", account.ID, connID, prewarmEventCount, truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen), canFallback, errCode, errType, errMessage)
			lease.MarkBroken()
			if canFallback {
				return wrapOpenAIWSFallback("prewarm_"+fallbackReason, errors.New(errMsg))
			}
			return wrapOpenAIWSFallback("prewarm_error_event", errors.New(errMsg))
		}
		if isOpenAIWSTerminalEvent(eventType) {
			prewarmTerminalCount++
			break
		}
	}
	lease.MarkPrewarmed()
	if prewarmResponseID != "" && stateStore != nil {
		ttl := s.openAIWSResponseStickyTTL()
		logOpenAIWSBindResponseAccountWarn(groupID, account.ID, prewarmResponseID, stateStore.BindResponseAccount(ctx, groupID, prewarmResponseID, account.ID, ttl))
		stateStore.BindResponseConn(prewarmResponseID, lease.ConnID(), ttl)
	}
	logOpenAIWSModeInfo("prewarm_done account_id=%d conn_id=%s response_id=%s events=%d terminal_events=%d duration_ms=%d", account.ID, connID, truncateOpenAIWSLogValue(prewarmResponseID, openAIWSIDValueMaxLen), prewarmEventCount, prewarmTerminalCount, time.Since(prewarmStart).Milliseconds())
	return nil
}
