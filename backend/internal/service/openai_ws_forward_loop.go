package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *OpenAIGatewayService) ProxyResponsesWebSocketFromClient(ctx context.Context, c *gin.Context, clientConn *coderws.Conn, account *Account, token string, firstClientMessage []byte, hooks *OpenAIWSIngressHooks) error {
	if s == nil {
		return errors.New("service is nil")
	}
	if c == nil {
		return errors.New("gin context is nil")
	}
	if clientConn == nil {
		return errors.New("client websocket is nil")
	}
	if account == nil {
		return errors.New("account is nil")
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("token is empty")
	}
	wsDecision := s.getOpenAIWSProtocolResolver().Resolve(account)
	modeRouterV2Enabled := s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.ModeRouterV2Enabled
	ingressMode := OpenAIWSIngressModeCtxPool
	if modeRouterV2Enabled {
		ingressMode = account.ResolveOpenAIResponsesWebSocketV2Mode(s.cfg.Gateway.OpenAIWS.IngressModeDefault)
		if ingressMode == OpenAIWSIngressModeOff {
			return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "websocket mode is disabled for this account", nil)
		}
		switch ingressMode {
		case OpenAIWSIngressModePassthrough:
			if wsDecision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
				return fmt.Errorf("websocket ingress requires ws_v2 transport, got=%s", wsDecision.Transport)
			}
			return s.proxyResponsesWebSocketV2Passthrough(ctx, c, clientConn, account, token, firstClientMessage, hooks, wsDecision)
		case OpenAIWSIngressModeCtxPool, OpenAIWSIngressModeShared, OpenAIWSIngressModeDedicated:
		default:
			return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "websocket mode only supports ctx_pool/passthrough", nil)
		}
	}
	if wsDecision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
		return fmt.Errorf("websocket ingress requires ws_v2 transport, got=%s", wsDecision.Transport)
	}
	dedicatedMode := modeRouterV2Enabled && ingressMode == OpenAIWSIngressModeDedicated
	wsURL, err := s.buildOpenAIResponsesWSURL(account)
	if err != nil {
		return fmt.Errorf("build ws url: %w", err)
	}
	wsHost := "-"
	wsPath := "-"
	if parsedURL, parseErr := url.Parse(wsURL); parseErr == nil && parsedURL != nil {
		wsHost = normalizeOpenAIWSLogValue(parsedURL.Host)
		wsPath = normalizeOpenAIWSLogValue(parsedURL.Path)
	}
	debugEnabled := isOpenAIWSModeDebugEnabled()
	firstPayload, err := s.parseOpenAIWSClientPayload(ctx, c, account, firstClientMessage)
	if err != nil {
		return err
	}
	ctx = WithOpenAICodexRequestModel(ctx, strings.TrimSpace(gjson.GetBytes(firstPayload.payloadRaw, "model").String()))
	if c.Request != nil {
		c.Request = c.Request.WithContext(ctx)
	}
	turnState := strings.TrimSpace(c.GetHeader(openAIWSTurnStateHeader))
	stateStore := s.getOpenAIWSStateStore()
	groupID := getOpenAIGroupIDFromContext(c)
	sessionHash := s.GenerateSessionHash(c, firstPayload.rawForHash)
	if turnState == "" && stateStore != nil && sessionHash != "" {
		if savedTurnState, ok := stateStore.GetSessionTurnState(groupID, sessionHash); ok {
			turnState = savedTurnState
		}
	}
	preferredConnID := ""
	if stateStore != nil && firstPayload.previousResponseID != "" {
		if connID, ok := stateStore.GetResponseConn(firstPayload.previousResponseID); ok {
			preferredConnID = connID
		}
	}
	storeDisabled := s.isOpenAIWSStoreDisabledInRequestRaw(firstPayload.payloadRaw, account)
	storeDisabledConnMode := s.openAIWSStoreDisabledConnMode()
	if stateStore != nil && storeDisabled && firstPayload.previousResponseID == "" && sessionHash != "" {
		if connID, ok := stateStore.GetSessionConn(groupID, sessionHash); ok {
			preferredConnID = connID
		}
	}
	isCodexCLI := openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) || (s.cfg != nil && s.cfg.Gateway.ForceCodexCLI)
	wsHeaders, _ := s.buildOpenAIWSHeaders(ctx, c, account, token, wsDecision, isCodexCLI, turnState, strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)), firstPayload.promptCacheKey)
	baseAcquireReq := openAIWSAcquireRequest{Account: account, WSURL: wsURL, Headers: wsHeaders, ProxyURL: func() string {
		if account.ProxyID != nil && account.Proxy != nil {
			return account.Proxy.URL()
		}
		return ""
	}(), ForceNewConn: false}
	pool := s.getOpenAIWSConnPool()
	if pool == nil {
		return errors.New("openai ws conn pool is nil")
	}
	logOpenAIWSModeInfo("ingress_ws_protocol_confirm account_id=%d account_type=%s transport=%s ws_host=%s ws_path=%s ws_mode=%s store_disabled=%v has_session_hash=%v has_previous_response_id=%v", account.ID, account.Type, normalizeOpenAIWSLogValue(string(wsDecision.Transport)), wsHost, wsPath, normalizeOpenAIWSLogValue(ingressMode), storeDisabled, sessionHash != "", firstPayload.previousResponseID != "")
	if debugEnabled {
		logOpenAIWSModeDebug("ingress_ws_start account_id=%d account_type=%s transport=%s ws_host=%s preferred_conn_id=%s has_session_hash=%v has_previous_response_id=%v store_disabled=%v", account.ID, account.Type, normalizeOpenAIWSLogValue(string(wsDecision.Transport)), wsHost, truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen), sessionHash != "", firstPayload.previousResponseID != "", storeDisabled)
	}
	if firstPayload.previousResponseID != "" {
		firstPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(firstPayload.previousResponseID)
		logOpenAIWSModeInfo("ingress_ws_continuation_probe account_id=%d turn=%d previous_response_id=%s previous_response_id_kind=%s preferred_conn_id=%s session_hash=%s header_session_id=%s header_conversation_id=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v store_disabled=%v", account.ID, 1, truncateOpenAIWSLogValue(firstPayload.previousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(firstPreviousResponseIDKind), truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(sessionHash, 12), openAIWSHeaderValueForLog(baseAcquireReq.Headers, "session_id"), openAIWSHeaderValueForLog(baseAcquireReq.Headers, "conversation_id"), turnState != "", len(turnState), firstPayload.promptCacheKey != "", storeDisabled)
	}
	acquireTimeout := s.openAIWSAcquireTimeout()
	if acquireTimeout <= 0 {
		acquireTimeout = 30 * time.Second
	}
	acquireTurnLease := func(turn int, preferred string, forcePreferredConn bool) (*openAIWSConnLease, error) {
		req := cloneOpenAIWSAcquireRequest(baseAcquireReq)
		req.PreferredConnID = strings.TrimSpace(preferred)
		req.ForcePreferredConn = forcePreferredConn
		req.ForceNewConn = dedicatedMode
		acquireCtx, acquireCancel := context.WithTimeout(ctx, acquireTimeout)
		lease, acquireErr := pool.Acquire(acquireCtx, req)
		acquireCancel()
		if acquireErr != nil {
			dialStatus, dialClass, dialCloseStatus, dialCloseReason, dialRespServer, dialRespVia, dialRespCFRay, dialRespReqID := summarizeOpenAIWSDialError(acquireErr)
			logOpenAIWSModeInfo("ingress_ws_upstream_acquire_fail account_id=%d turn=%d reason=%s dial_status=%d dial_class=%s dial_close_status=%s dial_close_reason=%s dial_resp_server=%s dial_resp_via=%s dial_resp_cf_ray=%s dial_resp_x_request_id=%s cause=%s preferred_conn_id=%s force_preferred_conn=%v ws_host=%s ws_path=%s proxy_enabled=%v", account.ID, turn, normalizeOpenAIWSLogValue(classifyOpenAIWSAcquireError(acquireErr)), dialStatus, dialClass, dialCloseStatus, truncateOpenAIWSLogValue(dialCloseReason, openAIWSHeaderValueMaxLen), dialRespServer, dialRespVia, dialRespCFRay, dialRespReqID, truncateOpenAIWSLogValue(acquireErr.Error(), openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(preferred, openAIWSIDValueMaxLen), forcePreferredConn, wsHost, wsPath, account.ProxyID != nil && account.Proxy != nil)
			var dialErr *openAIWSDialError
			if errors.As(acquireErr, &dialErr) && dialErr != nil && dialErr.StatusCode == http.StatusTooManyRequests {
				s.persistOpenAIWSRateLimitSignal(ctx, account, dialErr.ResponseHeaders, nil, "rate_limit_exceeded", "rate_limit_error", strings.TrimSpace(acquireErr.Error()))
			}
			if errors.Is(acquireErr, errOpenAIWSPreferredConnUnavailable) {
				return nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "upstream continuation connection is unavailable; please restart the conversation", acquireErr)
			}
			if errors.Is(acquireErr, context.DeadlineExceeded) || errors.Is(acquireErr, errOpenAIWSConnQueueFull) {
				return nil, NewOpenAIWSClientCloseError(coderws.StatusTryAgainLater, "upstream websocket is busy, please retry later", acquireErr)
			}
			return nil, acquireErr
		}
		connID := strings.TrimSpace(lease.ConnID())
		if handshakeTurnState := strings.TrimSpace(lease.HandshakeHeader(openAIWSTurnStateHeader)); handshakeTurnState != "" {
			turnState = handshakeTurnState
			if stateStore != nil && sessionHash != "" {
				stateStore.BindSessionTurnState(groupID, sessionHash, handshakeTurnState, s.openAIWSSessionStickyTTL())
			}
			updatedHeaders := cloneHeader(baseAcquireReq.Headers)
			if updatedHeaders == nil {
				updatedHeaders = make(http.Header)
			}
			updatedHeaders.Set(openAIWSTurnStateHeader, handshakeTurnState)
			baseAcquireReq.Headers = updatedHeaders
		}
		logOpenAIWSModeInfo("ingress_ws_upstream_connected account_id=%d turn=%d conn_id=%s conn_reused=%v conn_pick_ms=%d queue_wait_ms=%d preferred_conn_id=%s", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), lease.Reused(), lease.ConnPickDuration().Milliseconds(), lease.QueueWaitDuration().Milliseconds(), truncateOpenAIWSLogValue(preferred, openAIWSIDValueMaxLen))
		return lease, nil
	}
	readClientMessage := func() ([]byte, error) {
		msgType, payload, readErr := clientConn.Read(ctx)
		if readErr != nil {
			return nil, readErr
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			return nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, fmt.Sprintf("unsupported websocket client message type: %s", msgType.String()), nil)
		}
		return payload, nil
	}
	currentPayload := firstPayload.payloadRaw
	currentOriginalModel := firstPayload.originalModel
	currentPayloadBytes := firstPayload.payloadBytes
	isStrictAffinityTurn := func(payload []byte) bool {
		if !storeDisabled {
			return false
		}
		return strings.TrimSpace(openAIWSPayloadStringFromRaw(payload, "previous_response_id")) != ""
	}
	var sessionLease *openAIWSConnLease
	sessionConnID := ""
	pinnedSessionConnID := ""
	unpinSessionConn := func(connID string) {
		connID = strings.TrimSpace(connID)
		if connID == "" || pinnedSessionConnID != connID {
			return
		}
		pool.UnpinConn(account.ID, connID)
		pinnedSessionConnID = ""
	}
	pinSessionConn := func(connID string) {
		if !storeDisabled {
			return
		}
		connID = strings.TrimSpace(connID)
		if connID == "" || pinnedSessionConnID == connID {
			return
		}
		if pinnedSessionConnID != "" {
			pool.UnpinConn(account.ID, pinnedSessionConnID)
			pinnedSessionConnID = ""
		}
		if pool.PinConn(account.ID, connID) {
			pinnedSessionConnID = connID
		}
	}
	// lastTurnClean 标记最后一轮 relayOpenAIWSIngressTurn 是否正常完成（收到终端事件且客户端未断连）。
	// 所有异常路径（读写错误、error 事件、客户端断连）已在各自分支或上层中 MarkBroken，
	// 因此 releaseSessionLease 中只需在非正常结束时 MarkBroken。
	lastTurnClean := false
	releaseSessionLease := func() {
		if sessionLease == nil {
			return
		}
		if !lastTurnClean {
			sessionLease.MarkBroken()
		}
		unpinSessionConn(sessionConnID)
		sessionLease.Release()
		if debugEnabled {
			logOpenAIWSModeDebug("ingress_ws_upstream_released account_id=%d conn_id=%s", account.ID, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen))
		}
	}
	defer releaseSessionLease()
	turn := 1
	turnRetry := 0
	turnPrevRecoveryTried := false
	lastTurnFinishedAt := time.Time{}
	lastTurnResponseID := ""
	lastTurnPayload := []byte(nil)
	var lastTurnStrictState *openAIWSIngressPreviousTurnStrictState
	lastTurnReplayInput := []json.RawMessage(nil)
	lastTurnReplayInputExists := false
	currentTurnReplayInput := []json.RawMessage(nil)
	currentTurnReplayInputExists := false
	skipBeforeTurn := false
	resetSessionLease := func(markBroken bool) {
		if sessionLease == nil {
			return
		}
		lastTurnClean = false
		if markBroken {
			sessionLease.MarkBroken()
		}
		releaseSessionLease()
		sessionLease = nil
		sessionConnID = ""
		preferredConnID = ""
	}
	recoverIngressPrevResponseNotFound := func(relayErr error, turn int, connID string) bool {
		if !isOpenAIWSIngressPreviousResponseNotFound(relayErr) {
			return false
		}
		if turnPrevRecoveryTried || !s.openAIWSIngressPreviousResponseRecoveryEnabled() {
			return false
		}
		if isStrictAffinityTurn(currentPayload) {
			logOpenAIWSModeInfo("ingress_ws_prev_response_recovery_layer2 account_id=%d turn=%d conn_id=%s store_disabled_conn_mode=%s action=drop_previous_response_id_retry", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(storeDisabledConnMode))
		}
		turnPrevRecoveryTried = true
		updatedPayload, removed, dropErr := dropPreviousResponseIDFromRawPayload(currentPayload)
		if dropErr != nil || !removed {
			reason := "not_removed"
			if dropErr != nil {
				reason = "drop_error"
			}
			logOpenAIWSModeInfo("ingress_ws_prev_response_recovery_skip account_id=%d turn=%d conn_id=%s reason=%s", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(reason))
			return false
		}
		updatedWithInput, setInputErr := setOpenAIWSPayloadInputSequence(updatedPayload, currentTurnReplayInput, currentTurnReplayInputExists)
		if setInputErr != nil {
			logOpenAIWSModeInfo("ingress_ws_prev_response_recovery_skip account_id=%d turn=%d conn_id=%s reason=set_full_input_error cause=%s", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(setInputErr.Error(), openAIWSLogValueMaxLen))
			return false
		}
		logOpenAIWSModeInfo("ingress_ws_prev_response_recovery account_id=%d turn=%d conn_id=%s action=drop_previous_response_id retry=1", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen))
		currentPayload = updatedWithInput
		currentPayloadBytes = len(updatedWithInput)
		resetSessionLease(true)
		skipBeforeTurn = true
		return true
	}
	retryIngressTurn := func(relayErr error, turn int, connID string) bool {
		if !isOpenAIWSIngressTurnRetryable(relayErr) || turnRetry >= 1 {
			return false
		}
		if isStrictAffinityTurn(currentPayload) {
			logOpenAIWSModeInfo("ingress_ws_turn_retry_skip account_id=%d turn=%d conn_id=%s reason=strict_affinity", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen))
			return false
		}
		turnRetry++
		logOpenAIWSModeInfo("ingress_ws_turn_retry account_id=%d turn=%d retry=%d reason=%s conn_id=%s", account.ID, turn, turnRetry, truncateOpenAIWSLogValue(openAIWSIngressTurnRetryReason(relayErr), openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen))
		resetSessionLease(true)
		skipBeforeTurn = true
		return true
	}
	for {
		if !skipBeforeTurn && hooks != nil && hooks.BeforeTurn != nil {
			if err := hooks.BeforeTurn(turn); err != nil {
				return err
			}
		}
		skipBeforeTurn = false
		currentPreviousResponseID := openAIWSPayloadStringFromRaw(currentPayload, "previous_response_id")
		expectedPrev := strings.TrimSpace(lastTurnResponseID)
		hasFunctionCallOutput := openAIWSRawPayloadHasToolCallOutput(currentPayload)
		if shouldInferIngressFunctionCallOutputPreviousResponseID(storeDisabled, turn, hasFunctionCallOutput, currentPreviousResponseID, expectedPrev) {
			updatedPayload, setPrevErr := setPreviousResponseIDToRawPayload(currentPayload, expectedPrev)
			if setPrevErr != nil {
				logOpenAIWSModeInfo("ingress_ws_function_call_output_prev_infer_skip account_id=%d turn=%d conn_id=%s reason=set_previous_response_id_error cause=%s expected_previous_response_id=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(setPrevErr.Error(), openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen))
			} else {
				currentPayload = updatedPayload
				currentPayloadBytes = len(updatedPayload)
				currentPreviousResponseID = expectedPrev
				logOpenAIWSModeInfo("ingress_ws_function_call_output_prev_infer account_id=%d turn=%d conn_id=%s action=set_previous_response_id previous_response_id=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen))
			}
		}
		nextReplayInput, nextReplayInputExists, replayInputErr := buildOpenAIWSReplayInputSequence(lastTurnReplayInput, lastTurnReplayInputExists, currentPayload, currentPreviousResponseID != "")
		if replayInputErr != nil {
			logOpenAIWSModeInfo("ingress_ws_replay_input_skip account_id=%d turn=%d conn_id=%s reason=build_error cause=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(replayInputErr.Error(), openAIWSLogValueMaxLen))
			currentTurnReplayInput = nil
			currentTurnReplayInputExists = false
		} else {
			currentTurnReplayInput = nextReplayInput
			currentTurnReplayInputExists = nextReplayInputExists
		}
		if storeDisabled && turn > 1 && currentPreviousResponseID != "" {
			shouldKeepPreviousResponseID := false
			strictReason := ""
			var strictErr error
			if lastTurnStrictState != nil {
				shouldKeepPreviousResponseID, strictReason, strictErr = shouldKeepIngressPreviousResponseIDWithStrictState(lastTurnStrictState, currentPayload, lastTurnResponseID, hasFunctionCallOutput)
			} else {
				shouldKeepPreviousResponseID, strictReason, strictErr = shouldKeepIngressPreviousResponseID(lastTurnPayload, currentPayload, lastTurnResponseID, hasFunctionCallOutput)
			}
			if strictErr != nil {
				logOpenAIWSModeInfo("ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=keep_previous_response_id reason=%s cause=%s previous_response_id=%s expected_previous_response_id=%s has_function_call_output=%v", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(strictReason), truncateOpenAIWSLogValue(strictErr.Error(), openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen), hasFunctionCallOutput)
			} else if !shouldKeepPreviousResponseID {
				updatedPayload, removed, dropErr := dropPreviousResponseIDFromRawPayload(currentPayload)
				if dropErr != nil || !removed {
					dropReason := "not_removed"
					if dropErr != nil {
						dropReason = "drop_error"
					}
					logOpenAIWSModeInfo("ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=keep_previous_response_id reason=%s drop_reason=%s previous_response_id=%s expected_previous_response_id=%s has_function_call_output=%v", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(strictReason), normalizeOpenAIWSLogValue(dropReason), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen), hasFunctionCallOutput)
				} else {
					updatedWithInput, setInputErr := setOpenAIWSPayloadInputSequence(updatedPayload, currentTurnReplayInput, currentTurnReplayInputExists)
					if setInputErr != nil {
						logOpenAIWSModeInfo("ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=keep_previous_response_id reason=%s drop_reason=set_full_input_error previous_response_id=%s expected_previous_response_id=%s cause=%s has_function_call_output=%v", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(strictReason), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(setInputErr.Error(), openAIWSLogValueMaxLen), hasFunctionCallOutput)
					} else {
						currentPayload = updatedWithInput
						currentPayloadBytes = len(updatedWithInput)
						logOpenAIWSModeInfo("ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=drop_previous_response_id_full_create reason=%s previous_response_id=%s expected_previous_response_id=%s has_function_call_output=%v", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(strictReason), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen), hasFunctionCallOutput)
						currentPreviousResponseID = ""
					}
				}
			}
		}
		forcePreferredConn := isStrictAffinityTurn(currentPayload)
		if sessionLease == nil {
			acquiredLease, acquireErr := acquireTurnLease(turn, preferredConnID, forcePreferredConn)
			if acquireErr != nil {
				return fmt.Errorf("acquire upstream websocket: %w", acquireErr)
			}
			sessionLease = acquiredLease
			sessionConnID = strings.TrimSpace(sessionLease.ConnID())
			if storeDisabled {
				pinSessionConn(sessionConnID)
			} else {
				unpinSessionConn(sessionConnID)
			}
		}
		shouldPreflightPing := turn > 1 && sessionLease != nil && turnRetry == 0
		if shouldPreflightPing && openAIWSIngressPreflightPingIdle > 0 && !lastTurnFinishedAt.IsZero() {
			if time.Since(lastTurnFinishedAt) < openAIWSIngressPreflightPingIdle {
				shouldPreflightPing = false
			}
		}
		if shouldPreflightPing {
			if pingErr := sessionLease.PingWithTimeout(openAIWSConnHealthCheckTO); pingErr != nil {
				logOpenAIWSModeInfo("ingress_ws_upstream_preflight_ping_fail account_id=%d turn=%d conn_id=%s cause=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(pingErr.Error(), openAIWSLogValueMaxLen))
				if forcePreferredConn {
					if !turnPrevRecoveryTried && currentPreviousResponseID != "" {
						updatedPayload, removed, dropErr := dropPreviousResponseIDFromRawPayload(currentPayload)
						if dropErr != nil || !removed {
							reason := "not_removed"
							if dropErr != nil {
								reason = "drop_error"
							}
							logOpenAIWSModeInfo("ingress_ws_preflight_ping_recovery_skip account_id=%d turn=%d conn_id=%s reason=%s previous_response_id=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(reason), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen))
						} else {
							updatedWithInput, setInputErr := setOpenAIWSPayloadInputSequence(updatedPayload, currentTurnReplayInput, currentTurnReplayInputExists)
							if setInputErr != nil {
								logOpenAIWSModeInfo("ingress_ws_preflight_ping_recovery_skip account_id=%d turn=%d conn_id=%s reason=set_full_input_error previous_response_id=%s cause=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(setInputErr.Error(), openAIWSLogValueMaxLen))
							} else {
								logOpenAIWSModeInfo("ingress_ws_preflight_ping_recovery account_id=%d turn=%d conn_id=%s action=drop_previous_response_id_retry previous_response_id=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen))
								turnPrevRecoveryTried = true
								currentPayload = updatedWithInput
								currentPayloadBytes = len(updatedWithInput)
								resetSessionLease(true)
								skipBeforeTurn = true
								continue
							}
						}
					}
					resetSessionLease(true)
					return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "upstream continuation connection is unavailable; please restart the conversation", pingErr)
				}
				resetSessionLease(true)
				acquiredLease, acquireErr := acquireTurnLease(turn, preferredConnID, forcePreferredConn)
				if acquireErr != nil {
					return fmt.Errorf("acquire upstream websocket after preflight ping fail: %w", acquireErr)
				}
				sessionLease = acquiredLease
				sessionConnID = strings.TrimSpace(sessionLease.ConnID())
				if storeDisabled {
					pinSessionConn(sessionConnID)
				}
			}
		}
		connID := sessionConnID
		if currentPreviousResponseID != "" {
			chainedFromLast := expectedPrev != "" && currentPreviousResponseID == expectedPrev
			currentPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(currentPreviousResponseID)
			logOpenAIWSModeInfo("ingress_ws_turn_chain account_id=%d turn=%d conn_id=%s previous_response_id=%s previous_response_id_kind=%s last_turn_response_id=%s chained_from_last=%v preferred_conn_id=%s header_session_id=%s header_conversation_id=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v store_disabled=%v", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(currentPreviousResponseIDKind), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen), chainedFromLast, truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen), openAIWSHeaderValueForLog(baseAcquireReq.Headers, "session_id"), openAIWSHeaderValueForLog(baseAcquireReq.Headers, "conversation_id"), turnState != "", len(turnState), openAIWSPayloadStringFromRaw(currentPayload, "prompt_cache_key") != "", storeDisabled)
		}
		result, relayErr := s.relayOpenAIWSIngressTurn(openAIWSIngressTurnRelayInput{
			ctx:           ctx,
			account:       account,
			clientConn:    clientConn,
			lease:         sessionLease,
			payload:       currentPayload,
			payloadBytes:  currentPayloadBytes,
			originalModel: currentOriginalModel,
			turn:          turn,
			debugEnabled:  debugEnabled,
		})
		if relayErr != nil {
			lastTurnClean = false
			if recoverIngressPrevResponseNotFound(relayErr, turn, connID) {
				continue
			}
			if retryIngressTurn(relayErr, turn, connID) {
				continue
			}
			finalErr := relayErr
			if unwrapped := errors.Unwrap(relayErr); unwrapped != nil {
				finalErr = unwrapped
			}
			if hooks != nil && hooks.AfterTurn != nil {
				hooks.AfterTurn(turn, nil, finalErr)
			}
			sessionLease.MarkBroken()
			return finalErr
		}
		turnRetry = 0
		turnPrevRecoveryTried = false
		lastTurnFinishedAt = time.Now()
		lastTurnClean = true
		if hooks != nil && hooks.AfterTurn != nil {
			hooks.AfterTurn(turn, result, nil)
		}
		if result == nil {
			return errors.New("websocket turn result is nil")
		}
		responseID := strings.TrimSpace(result.RequestID)
		lastTurnResponseID = responseID
		lastTurnPayload = cloneOpenAIWSPayloadBytes(currentPayload)
		lastTurnReplayInput = cloneOpenAIWSRawMessages(currentTurnReplayInput)
		lastTurnReplayInputExists = currentTurnReplayInputExists
		nextStrictState, strictStateErr := buildOpenAIWSIngressPreviousTurnStrictState(currentPayload)
		if strictStateErr != nil {
			lastTurnStrictState = nil
			logOpenAIWSModeInfo("ingress_ws_prev_response_strict_state_skip account_id=%d turn=%d conn_id=%s reason=build_error cause=%s", account.ID, turn, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(strictStateErr.Error(), openAIWSLogValueMaxLen))
		} else {
			lastTurnStrictState = nextStrictState
		}
		if responseID != "" && stateStore != nil {
			ttl := s.openAIWSResponseStickyTTL()
			logOpenAIWSBindResponseAccountWarn(groupID, account.ID, responseID, stateStore.BindResponseAccount(ctx, groupID, responseID, account.ID, ttl))
			stateStore.BindResponseConn(responseID, connID, ttl)
		}
		if stateStore != nil && storeDisabled && sessionHash != "" {
			stateStore.BindSessionConn(groupID, sessionHash, connID, s.openAIWSSessionStickyTTL())
		}
		if connID != "" {
			preferredConnID = connID
		}
		nextClientMessage, readErr := readClientMessage()
		if readErr != nil {
			if isOpenAIWSClientDisconnectError(readErr) {
				lastTurnClean = false
				if sessionLease != nil {
					sessionLease.MarkBroken()
				}
				closeStatus, closeReason := summarizeOpenAIWSReadCloseError(readErr)
				logOpenAIWSModeInfo("ingress_ws_client_closed account_id=%d conn_id=%s close_status=%s close_reason=%s", account.ID, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), closeStatus, truncateOpenAIWSLogValue(closeReason, openAIWSHeaderValueMaxLen))
				return nil
			}
			lastTurnClean = false
			if sessionLease != nil {
				sessionLease.MarkBroken()
			}
			return fmt.Errorf("read client websocket request: %w", readErr)
		}
		nextPayload, parseErr := s.parseOpenAIWSClientPayload(ctx, c, account, nextClientMessage)
		if parseErr != nil {
			return parseErr
		}
		if nextPayload.promptCacheKey != "" {
			updatedHeaders, _ := s.buildOpenAIWSHeaders(ctx, c, account, token, wsDecision, isCodexCLI, turnState, strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)), nextPayload.promptCacheKey)
			baseAcquireReq.Headers = updatedHeaders
		}
		if nextPayload.previousResponseID != "" {
			expectedPrev := strings.TrimSpace(lastTurnResponseID)
			chainedFromLast := expectedPrev != "" && nextPayload.previousResponseID == expectedPrev
			nextPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(nextPayload.previousResponseID)
			logOpenAIWSModeInfo("ingress_ws_next_turn_chain account_id=%d turn=%d next_turn=%d conn_id=%s previous_response_id=%s previous_response_id_kind=%s last_turn_response_id=%s chained_from_last=%v has_prompt_cache_key=%v store_disabled=%v", account.ID, turn, turn+1, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(nextPayload.previousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(nextPreviousResponseIDKind), truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen), chainedFromLast, nextPayload.promptCacheKey != "", storeDisabled)
		}
		if stateStore != nil && nextPayload.previousResponseID != "" {
			if stickyConnID, ok := stateStore.GetResponseConn(nextPayload.previousResponseID); ok {
				if sessionConnID != "" && stickyConnID != "" && stickyConnID != sessionConnID {
					logOpenAIWSModeInfo("ingress_ws_keep_session_conn account_id=%d turn=%d conn_id=%s sticky_conn_id=%s previous_response_id=%s", account.ID, turn, truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(stickyConnID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(nextPayload.previousResponseID, openAIWSIDValueMaxLen))
				} else {
					preferredConnID = stickyConnID
				}
			}
		}
		currentPayload = nextPayload.payloadRaw
		currentOriginalModel = nextPayload.originalModel
		currentPayloadBytes = nextPayload.payloadBytes
		storeDisabled = s.isOpenAIWSStoreDisabledInRequestRaw(currentPayload, account)
		if !storeDisabled {
			unpinSessionConn(sessionConnID)
		}
		turn++
	}
}
