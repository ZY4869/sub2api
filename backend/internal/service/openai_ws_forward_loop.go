package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *OpenAIGatewayService) forwardOpenAIWSV2(ctx context.Context, c *gin.Context, account *Account, reqBody map[string]any, token string, decision OpenAIWSProtocolDecision, isCodexCLI bool, reqStream bool, originalModel string, mappedModel string, startTime time.Time, attempt int, lastFailureReason string) (*OpenAIForwardResult, error) {
	if s == nil || account == nil {
		return nil, wrapOpenAIWSFallback("invalid_state", errors.New("service or account is nil"))
	}
	ctx = WithOpenAICodexRequestModel(ctx, mappedModel)
	if c != nil && c.Request != nil {
		c.Request = c.Request.WithContext(ctx)
	}
	wsURL, err := s.buildOpenAIResponsesWSURL(account)
	if err != nil {
		return nil, wrapOpenAIWSFallback("build_ws_url", err)
	}
	wsHost := "-"
	wsPath := "-"
	if parsed, parseErr := url.Parse(wsURL); parseErr == nil && parsed != nil {
		if h := strings.TrimSpace(parsed.Host); h != "" {
			wsHost = normalizeOpenAIWSLogValue(h)
		}
		if p := strings.TrimSpace(parsed.Path); p != "" {
			wsPath = normalizeOpenAIWSLogValue(p)
		}
	}
	logOpenAIWSModeDebug("dial_target account_id=%d account_type=%s ws_host=%s ws_path=%s", account.ID, account.Type, wsHost, wsPath)
	payload := s.buildOpenAIWSCreatePayload(reqBody, account)
	payloadStrategy, removedKeys := applyOpenAIWSRetryPayloadStrategy(payload, attempt)
	previousResponseID := openAIWSPayloadString(payload, "previous_response_id")
	previousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
	promptCacheKey := openAIWSPayloadString(payload, "prompt_cache_key")
	_, hasTools := payload["tools"]
	debugEnabled := isOpenAIWSModeDebugEnabled()
	payloadBytes := -1
	resolvePayloadBytes := func() int {
		if payloadBytes >= 0 {
			return payloadBytes
		}
		payloadBytes = len(payloadAsJSONBytes(payload))
		return payloadBytes
	}
	streamValue := "-"
	if raw, ok := payload["stream"]; ok {
		streamValue = normalizeOpenAIWSLogValue(strings.TrimSpace(fmt.Sprintf("%v", raw)))
	}
	turnState := ""
	turnMetadata := ""
	if c != nil && c.Request != nil {
		turnState = strings.TrimSpace(c.GetHeader(openAIWSTurnStateHeader))
		turnMetadata = strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader))
	}
	setOpenAIWSTurnMetadata(payload, turnMetadata)
	payloadEventType := openAIWSPayloadString(payload, "type")
	if payloadEventType == "" {
		payloadEventType = "response.create"
	}
	if s.shouldEmitOpenAIWSPayloadSchema(attempt) {
		logOpenAIWSModeInfo("[debug] payload_schema account_id=%d attempt=%d event=%s payload_keys=%s payload_bytes=%d payload_key_sizes=%s input_summary=%s stream=%s payload_strategy=%s removed_keys=%s has_previous_response_id=%v has_prompt_cache_key=%v has_tools=%v", account.ID, attempt, payloadEventType, normalizeOpenAIWSLogValue(strings.Join(sortedKeys(payload), ",")), resolvePayloadBytes(), normalizeOpenAIWSLogValue(summarizeOpenAIWSPayloadKeySizes(payload, openAIWSPayloadKeySizeTopN)), normalizeOpenAIWSLogValue(summarizeOpenAIWSInput(payload["input"])), streamValue, normalizeOpenAIWSLogValue(payloadStrategy), normalizeOpenAIWSLogValue(strings.Join(removedKeys, ",")), previousResponseID != "", promptCacheKey != "", hasTools)
	}
	stateStore := s.getOpenAIWSStateStore()
	groupID := getOpenAIGroupIDFromContext(c)
	sessionHash := s.GenerateSessionHash(c, nil)
	if sessionHash == "" {
		var legacySessionHash string
		sessionHash, legacySessionHash = openAIWSSessionHashesFromID(promptCacheKey)
		attachOpenAILegacySessionHashToGin(c, legacySessionHash)
	}
	if turnState == "" && stateStore != nil && sessionHash != "" {
		if savedTurnState, ok := stateStore.GetSessionTurnState(groupID, sessionHash); ok {
			turnState = savedTurnState
		}
	}
	preferredConnID := ""
	if stateStore != nil && previousResponseID != "" {
		if connID, ok := stateStore.GetResponseConn(previousResponseID); ok {
			preferredConnID = connID
		}
	}
	storeDisabled := s.isOpenAIWSStoreDisabledInRequest(reqBody, account)
	if stateStore != nil && storeDisabled && previousResponseID == "" && sessionHash != "" {
		if connID, ok := stateStore.GetSessionConn(groupID, sessionHash); ok {
			preferredConnID = connID
		}
	}
	storeDisabledConnMode := s.openAIWSStoreDisabledConnMode()
	forceNewConnByPolicy := shouldForceNewConnOnStoreDisabled(storeDisabledConnMode, lastFailureReason)
	forceNewConn := forceNewConnByPolicy && storeDisabled && previousResponseID == "" && sessionHash != "" && preferredConnID == ""
	wsHeaders, sessionResolution := s.buildOpenAIWSHeaders(c, account, token, decision, isCodexCLI, turnState, turnMetadata, promptCacheKey)
	logOpenAIWSModeDebug("acquire_start account_id=%d account_type=%s transport=%s preferred_conn_id=%s has_previous_response_id=%v session_hash=%s has_turn_state=%v turn_state_len=%d has_turn_metadata=%v turn_metadata_len=%d store_disabled=%v store_disabled_conn_mode=%s retry_last_reason=%s force_new_conn=%v header_user_agent=%s header_openai_beta=%s header_originator=%s header_accept_language=%s header_session_id=%s header_conversation_id=%s session_id_source=%s conversation_id_source=%s has_prompt_cache_key=%v has_chatgpt_account_id=%v has_authorization=%v has_session_id=%v has_conversation_id=%v proxy_enabled=%v", account.ID, account.Type, normalizeOpenAIWSLogValue(string(decision.Transport)), truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen), previousResponseID != "", truncateOpenAIWSLogValue(sessionHash, 12), turnState != "", len(turnState), turnMetadata != "", len(turnMetadata), storeDisabled, normalizeOpenAIWSLogValue(storeDisabledConnMode), truncateOpenAIWSLogValue(lastFailureReason, openAIWSLogValueMaxLen), forceNewConn, openAIWSHeaderValueForLog(wsHeaders, "user-agent"), openAIWSHeaderValueForLog(wsHeaders, "openai-beta"), openAIWSHeaderValueForLog(wsHeaders, "originator"), openAIWSHeaderValueForLog(wsHeaders, "accept-language"), openAIWSHeaderValueForLog(wsHeaders, "session_id"), openAIWSHeaderValueForLog(wsHeaders, "conversation_id"), normalizeOpenAIWSLogValue(sessionResolution.SessionSource), normalizeOpenAIWSLogValue(sessionResolution.ConversationSource), promptCacheKey != "", hasOpenAIWSHeader(wsHeaders, "chatgpt-account-id"), hasOpenAIWSHeader(wsHeaders, "authorization"), hasOpenAIWSHeader(wsHeaders, "session_id"), hasOpenAIWSHeader(wsHeaders, "conversation_id"), account.ProxyID != nil && account.Proxy != nil)
	acquireCtx, acquireCancel := context.WithTimeout(ctx, s.openAIWSAcquireTimeout())
	defer acquireCancel()
	lease, err := s.getOpenAIWSConnPool().Acquire(acquireCtx, openAIWSAcquireRequest{Account: account, WSURL: wsURL, Headers: wsHeaders, PreferredConnID: preferredConnID, ForceNewConn: forceNewConn, ProxyURL: func() string {
		if account.ProxyID != nil && account.Proxy != nil {
			return account.Proxy.URL()
		}
		return ""
	}()})
	if err != nil {
		dialStatus, dialClass, dialCloseStatus, dialCloseReason, dialRespServer, dialRespVia, dialRespCFRay, dialRespReqID := summarizeOpenAIWSDialError(err)
		logOpenAIWSModeInfo("acquire_fail account_id=%d account_type=%s transport=%s reason=%s dial_status=%d dial_class=%s dial_close_status=%s dial_close_reason=%s dial_resp_server=%s dial_resp_via=%s dial_resp_cf_ray=%s dial_resp_x_request_id=%s cause=%s preferred_conn_id=%s force_new_conn=%v ws_host=%s ws_path=%s proxy_enabled=%v", account.ID, account.Type, normalizeOpenAIWSLogValue(string(decision.Transport)), normalizeOpenAIWSLogValue(classifyOpenAIWSAcquireError(err)), dialStatus, dialClass, dialCloseStatus, truncateOpenAIWSLogValue(dialCloseReason, openAIWSHeaderValueMaxLen), dialRespServer, dialRespVia, dialRespCFRay, dialRespReqID, truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen), forceNewConn, wsHost, wsPath, account.ProxyID != nil && account.Proxy != nil)
		var dialErr *openAIWSDialError
		if errors.As(err, &dialErr) && dialErr != nil && dialErr.StatusCode == http.StatusTooManyRequests {
			s.persistOpenAIWSRateLimitSignal(ctx, account, dialErr.ResponseHeaders, nil, "rate_limit_exceeded", "rate_limit_error", strings.TrimSpace(err.Error()))
		}
		return nil, wrapOpenAIWSFallback(classifyOpenAIWSAcquireError(err), err)
	}
	// cleanExit 标记正常终端事件退出，此时上游不会再发送帧，连接可安全归还复用。
	// 所有异常路径（读写错误、error 事件等）已在各自分支中提前调用 MarkBroken，
	// 因此 defer 中只需处理正常退出时不 MarkBroken 即可。
	cleanExit := false
	defer func() {
		if !cleanExit {
			lease.MarkBroken()
		}
		lease.Release()
	}()
	connID := strings.TrimSpace(lease.ConnID())
	logOpenAIWSModeDebug("connected account_id=%d account_type=%s transport=%s conn_id=%s conn_reused=%v conn_pick_ms=%d queue_wait_ms=%d has_previous_response_id=%v", account.ID, account.Type, normalizeOpenAIWSLogValue(string(decision.Transport)), connID, lease.Reused(), lease.ConnPickDuration().Milliseconds(), lease.QueueWaitDuration().Milliseconds(), previousResponseID != "")
	if previousResponseID != "" {
		logOpenAIWSModeInfo("continuation_probe account_id=%d account_type=%s conn_id=%s previous_response_id=%s previous_response_id_kind=%s preferred_conn_id=%s conn_reused=%v store_disabled=%v session_hash=%s header_session_id=%s header_conversation_id=%s session_id_source=%s conversation_id_source=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v", account.ID, account.Type, truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(previousResponseIDKind), truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen), lease.Reused(), storeDisabled, truncateOpenAIWSLogValue(sessionHash, 12), openAIWSHeaderValueForLog(wsHeaders, "session_id"), openAIWSHeaderValueForLog(wsHeaders, "conversation_id"), normalizeOpenAIWSLogValue(sessionResolution.SessionSource), normalizeOpenAIWSLogValue(sessionResolution.ConversationSource), turnState != "", len(turnState), promptCacheKey != "")
	}
	if c != nil {
		SetOpsLatencyMs(c, OpsOpenAIWSConnPickMsKey, lease.ConnPickDuration().Milliseconds())
		SetOpsLatencyMs(c, OpsOpenAIWSQueueWaitMsKey, lease.QueueWaitDuration().Milliseconds())
		c.Set(OpsOpenAIWSConnReusedKey, lease.Reused())
		if connID != "" {
			c.Set(OpsOpenAIWSConnIDKey, connID)
		}
	}
	handshakeTurnState := strings.TrimSpace(lease.HandshakeHeader(openAIWSTurnStateHeader))
	logOpenAIWSModeDebug("handshake account_id=%d conn_id=%s has_turn_state=%v turn_state_len=%d", account.ID, connID, handshakeTurnState != "", len(handshakeTurnState))
	if handshakeTurnState != "" {
		if stateStore != nil && sessionHash != "" {
			stateStore.BindSessionTurnState(groupID, sessionHash, handshakeTurnState, s.openAIWSSessionStickyTTL())
		}
		if c != nil {
			c.Header(http.CanonicalHeaderKey(openAIWSTurnStateHeader), handshakeTurnState)
		}
	}
	if err := s.performOpenAIWSGeneratePrewarm(ctx, lease, decision, payload, previousResponseID, reqBody, account, stateStore, groupID); err != nil {
		return nil, err
	}
	if err := lease.WriteJSONWithContextTimeout(ctx, payload, s.openAIWSWriteTimeout()); err != nil {
		lease.MarkBroken()
		logOpenAIWSModeInfo("write_request_fail account_id=%d conn_id=%s cause=%s payload_bytes=%d", account.ID, connID, truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen), resolvePayloadBytes())
		return nil, wrapOpenAIWSFallback("write_request", err)
	}
	if debugEnabled {
		logOpenAIWSModeDebug("write_request_sent account_id=%d conn_id=%s stream=%v payload_bytes=%d previous_response_id=%s", account.ID, connID, reqStream, resolvePayloadBytes(), truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen))
	}
	usage := &OpenAIUsage{}
	var firstTokenMs *int
	responseID := ""
	var finalResponse []byte
	wroteDownstream := false
	needModelReplace := originalModel != mappedModel
	var mappedModelBytes []byte
	if needModelReplace && mappedModel != "" {
		mappedModelBytes = []byte(mappedModel)
	}
	bufferedStreamEvents := make([][]byte, 0, 4)
	eventCount := 0
	tokenEventCount := 0
	terminalEventCount := 0
	bufferedEventCount := 0
	flushedBufferedEventCount := 0
	firstEventType := ""
	lastEventType := ""
	var flusher http.Flusher
	if reqStream {
		if s.responseHeaderFilter != nil {
			responseheaders.WriteFilteredHeaders(c.Writer.Header(), http.Header{}, s.responseHeaderFilter)
		}
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		f, ok := c.Writer.(http.Flusher)
		if !ok {
			lease.MarkBroken()
			return nil, wrapOpenAIWSFallback("streaming_not_supported", errors.New("streaming not supported"))
		}
		flusher = f
	}
	clientDisconnected := false
	flushBatchSize := s.openAIWSEventFlushBatchSize()
	flushInterval := s.openAIWSEventFlushInterval()
	pendingFlushEvents := 0
	lastFlushAt := time.Now()
	flushStreamWriter := func(force bool) {
		if clientDisconnected || flusher == nil || pendingFlushEvents <= 0 {
			return
		}
		if !force && flushBatchSize > 1 && pendingFlushEvents < flushBatchSize {
			if flushInterval <= 0 || time.Since(lastFlushAt) < flushInterval {
				return
			}
		}
		flusher.Flush()
		pendingFlushEvents = 0
		lastFlushAt = time.Now()
	}
	emitStreamMessage := func(message []byte, forceFlush bool) {
		if clientDisconnected {
			return
		}
		frame := make([]byte, 0, len(message)+8)
		frame = append(frame, "data: "...)
		frame = append(frame, message...)
		frame = append(frame, '\n', '\n')
		_, wErr := c.Writer.Write(frame)
		if wErr == nil {
			wroteDownstream = true
			pendingFlushEvents++
			flushStreamWriter(forceFlush)
			return
		}
		clientDisconnected = true
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI WS Mode] client disconnected, continue draining upstream: account=%d", account.ID)
	}
	flushBufferedStreamEvents := func(reason string) {
		if len(bufferedStreamEvents) == 0 {
			return
		}
		flushed := len(bufferedStreamEvents)
		for _, buffered := range bufferedStreamEvents {
			emitStreamMessage(buffered, false)
		}
		bufferedStreamEvents = bufferedStreamEvents[:0]
		flushStreamWriter(true)
		flushedBufferedEventCount += flushed
		if debugEnabled {
			logOpenAIWSModeDebug("buffer_flush account_id=%d conn_id=%s reason=%s flushed=%d total_flushed=%d client_disconnected=%v", account.ID, connID, truncateOpenAIWSLogValue(reason, openAIWSLogValueMaxLen), flushed, flushedBufferedEventCount, clientDisconnected)
		}
	}
	readTimeout := s.openAIWSReadTimeout()
	for {
		message, readErr := lease.ReadMessageWithContextTimeout(ctx, readTimeout)
		if readErr != nil {
			lease.MarkBroken()
			closeStatus, closeReason := summarizeOpenAIWSReadCloseError(readErr)
			logOpenAIWSModeInfo("read_fail account_id=%d conn_id=%s wrote_downstream=%v close_status=%s close_reason=%s cause=%s events=%d token_events=%d terminal_events=%d buffered_pending=%d buffered_flushed=%d first_event=%s last_event=%s", account.ID, connID, wroteDownstream, closeStatus, closeReason, truncateOpenAIWSLogValue(readErr.Error(), openAIWSLogValueMaxLen), eventCount, tokenEventCount, terminalEventCount, len(bufferedStreamEvents), flushedBufferedEventCount, truncateOpenAIWSLogValue(firstEventType, openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(lastEventType, openAIWSLogValueMaxLen))
			if !wroteDownstream {
				return nil, wrapOpenAIWSFallback(classifyOpenAIWSReadFallbackReason(readErr), readErr)
			}
			if clientDisconnected {
				break
			}
			setOpsUpstreamError(c, 0, sanitizeUpstreamErrorMessage(readErr.Error()), "")
			return nil, fmt.Errorf("openai ws read event: %w", readErr)
		}
		eventType, eventResponseID, responseField := parseOpenAIWSEventEnvelope(message)
		if eventType == "" {
			continue
		}
		eventCount++
		if firstEventType == "" {
			firstEventType = eventType
		}
		lastEventType = eventType
		if responseID == "" && eventResponseID != "" {
			responseID = eventResponseID
		}
		isTokenEvent := isOpenAIWSTokenEvent(eventType)
		if isTokenEvent {
			tokenEventCount++
		}
		isTerminalEvent := isOpenAIWSTerminalEvent(eventType)
		if isTerminalEvent {
			terminalEventCount++
		}
		if firstTokenMs == nil && isTokenEvent {
			ms := int(time.Since(startTime).Milliseconds())
			firstTokenMs = &ms
		}
		if debugEnabled && shouldLogOpenAIWSEvent(eventCount, eventType) {
			logOpenAIWSModeDebug("event_received account_id=%d conn_id=%s idx=%d type=%s bytes=%d token=%v terminal=%v buffered_pending=%d", account.ID, connID, eventCount, truncateOpenAIWSLogValue(eventType, openAIWSLogValueMaxLen), len(message), isTokenEvent, isTerminalEvent, len(bufferedStreamEvents))
		}
		if !clientDisconnected {
			if needModelReplace && len(mappedModelBytes) > 0 && openAIWSEventMayContainModel(eventType) && bytes.Contains(message, mappedModelBytes) {
				message = replaceOpenAIWSMessageModel(message, mappedModel, originalModel)
			}
			if openAIWSEventMayContainToolCalls(eventType) && openAIWSMessageLikelyContainsToolCalls(message) {
				if corrected, changed := s.toolCorrector.CorrectToolCallsInSSEBytes(message); changed {
					message = corrected
				}
			}
		}
		if openAIWSEventShouldParseUsage(eventType) {
			parseOpenAIWSResponseUsageFromCompletedEvent(message, usage)
		}
		if eventType == "error" {
			errCodeRaw, errTypeRaw, errMsgRaw := parseOpenAIWSErrorEventFields(message)
			s.persistOpenAIWSRateLimitSignal(ctx, account, lease.HandshakeHeaders(), message, errCodeRaw, errTypeRaw, errMsgRaw)
			errMsg := strings.TrimSpace(errMsgRaw)
			if errMsg == "" {
				errMsg = "Upstream websocket error"
			}
			fallbackReason, canFallback := classifyOpenAIWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			errCode, errType, errMessage := summarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			logOpenAIWSModeInfo("error_event account_id=%d conn_id=%s idx=%d fallback_reason=%s can_fallback=%v err_code=%s err_type=%s err_message=%s", account.ID, connID, eventCount, truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen), canFallback, errCode, errType, errMessage)
			if fallbackReason == "previous_response_not_found" {
				logOpenAIWSModeInfo("previous_response_not_found_diag account_id=%d account_type=%s conn_id=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s event_idx=%d req_stream=%v store_disabled=%v conn_reused=%v session_hash=%s header_session_id=%s header_conversation_id=%s session_id_source=%s conversation_id_source=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v err_code=%s err_type=%s err_message=%s", account.ID, account.Type, connID, truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(previousResponseIDKind), truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen), eventCount, reqStream, storeDisabled, lease.Reused(), truncateOpenAIWSLogValue(sessionHash, 12), openAIWSHeaderValueForLog(wsHeaders, "session_id"), openAIWSHeaderValueForLog(wsHeaders, "conversation_id"), normalizeOpenAIWSLogValue(sessionResolution.SessionSource), normalizeOpenAIWSLogValue(sessionResolution.ConversationSource), turnState != "", len(turnState), promptCacheKey != "", errCode, errType, errMessage)
			}
			lease.MarkBroken()
			if !wroteDownstream && canFallback {
				return nil, wrapOpenAIWSFallback(fallbackReason, errors.New(errMsg))
			}
			statusCode := openAIWSErrorHTTPStatusFromRaw(errCodeRaw, errTypeRaw)
			setOpsUpstreamError(c, statusCode, errMsg, "")
			if reqStream && !clientDisconnected {
				flushBufferedStreamEvents("error_event")
				emitStreamMessage(message, true)
			}
			if !reqStream {
				c.JSON(statusCode, gin.H{"error": gin.H{"type": "upstream_error", "message": errMsg}})
			}
			return nil, fmt.Errorf("openai ws error event: %s", errMsg)
		}
		if reqStream {
			shouldBuffer := firstTokenMs == nil && !isTokenEvent && !isTerminalEvent
			if shouldBuffer {
				buffered := make([]byte, len(message))
				copy(buffered, message)
				bufferedStreamEvents = append(bufferedStreamEvents, buffered)
				bufferedEventCount++
				if debugEnabled && shouldLogOpenAIWSBufferedEvent(bufferedEventCount) {
					logOpenAIWSModeDebug("buffer_enqueue account_id=%d conn_id=%s idx=%d event_idx=%d event_type=%s buffer_size=%d", account.ID, connID, bufferedEventCount, eventCount, truncateOpenAIWSLogValue(eventType, openAIWSLogValueMaxLen), len(bufferedStreamEvents))
				}
			} else {
				flushBufferedStreamEvents(eventType)
				emitStreamMessage(message, isTerminalEvent)
			}
		} else {
			if responseField.Exists() && responseField.Type == gjson.JSON {
				finalResponse = []byte(responseField.Raw)
			}
		}
		if isTerminalEvent {
			cleanExit = true
			break
		}
	}
	if !reqStream {
		if len(finalResponse) == 0 {
			logOpenAIWSModeInfo("missing_final_response account_id=%d conn_id=%s events=%d token_events=%d terminal_events=%d wrote_downstream=%v", account.ID, connID, eventCount, tokenEventCount, terminalEventCount, wroteDownstream)
			if !wroteDownstream {
				return nil, wrapOpenAIWSFallback("missing_final_response", errors.New("no terminal response payload"))
			}
			return nil, errors.New("ws finished without final response")
		}
		if needModelReplace {
			finalResponse = s.replaceModelInResponseBody(finalResponse, mappedModel, originalModel)
		}
		finalResponse = s.correctToolCallsInResponseBody(finalResponse)
		populateOpenAIUsageFromResponseJSON(finalResponse, usage)
		if responseID == "" {
			responseID = strings.TrimSpace(gjson.GetBytes(finalResponse, "id").String())
		}
		c.Data(http.StatusOK, "application/json", finalResponse)
	} else {
		flushStreamWriter(true)
	}
	if responseID != "" && stateStore != nil {
		ttl := s.openAIWSResponseStickyTTL()
		logOpenAIWSBindResponseAccountWarn(groupID, account.ID, responseID, stateStore.BindResponseAccount(ctx, groupID, responseID, account.ID, ttl))
		stateStore.BindResponseConn(responseID, lease.ConnID(), ttl)
	}
	if stateStore != nil && storeDisabled && sessionHash != "" {
		stateStore.BindSessionConn(groupID, sessionHash, lease.ConnID(), s.openAIWSSessionStickyTTL())
	}
	firstTokenMsValue := -1
	if firstTokenMs != nil {
		firstTokenMsValue = *firstTokenMs
	}
	logOpenAIWSModeDebug("completed account_id=%d conn_id=%s response_id=%s stream=%v duration_ms=%d events=%d token_events=%d terminal_events=%d buffered_events=%d buffered_flushed=%d first_event=%s last_event=%s first_token_ms=%d wrote_downstream=%v client_disconnected=%v", account.ID, connID, truncateOpenAIWSLogValue(strings.TrimSpace(responseID), openAIWSIDValueMaxLen), reqStream, time.Since(startTime).Milliseconds(), eventCount, tokenEventCount, terminalEventCount, bufferedEventCount, flushedBufferedEventCount, truncateOpenAIWSLogValue(firstEventType, openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(lastEventType, openAIWSLogValueMaxLen), firstTokenMsValue, wroteDownstream, clientDisconnected)
	return &OpenAIForwardResult{RequestID: responseID, Usage: *usage, Model: originalModel, UpstreamModel: mappedModel, ServiceTier: extractOpenAIServiceTier(reqBody), ReasoningEffort: extractOpenAIReasoningEffort(reqBody, originalModel), Stream: reqStream, OpenAIWSMode: true, ResponseHeaders: lease.HandshakeHeaders(), Duration: time.Since(startTime), FirstTokenMs: firstTokenMs}, nil
}
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
	type openAIWSClientPayload struct {
		payloadRaw         []byte
		rawForHash         []byte
		promptCacheKey     string
		previousResponseID string
		originalModel      string
		payloadBytes       int
	}
	applyPayloadMutation := func(current []byte, path string, value any) ([]byte, error) {
		next, err := sjson.SetBytes(current, path, value)
		if err == nil {
			return next, nil
		}
		payload := make(map[string]any)
		if unmarshalErr := json.Unmarshal(current, &payload); unmarshalErr != nil {
			return nil, err
		}
		switch path {
		case "type", "model":
			payload[path] = value
		case "client_metadata." + openAIWSTurnMetadataHeader:
			setOpenAIWSTurnMetadata(payload, fmt.Sprintf("%v", value))
		default:
			return nil, err
		}
		rebuilt, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return rebuilt, nil
	}
	parseClientPayload := func(raw []byte) (openAIWSClientPayload, error) {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "empty websocket request payload", nil)
		}
		if !gjson.ValidBytes(trimmed) {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", errors.New("invalid json"))
		}
		values := gjson.GetManyBytes(trimmed, "type", "model", "prompt_cache_key", "previous_response_id")
		eventType := strings.TrimSpace(values[0].String())
		normalized := trimmed
		switch eventType {
		case "":
			eventType = "response.create"
			next, setErr := applyPayloadMutation(normalized, "type", eventType)
			if setErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", setErr)
			}
			normalized = next
		case "response.create":
		case "response.append":
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "response.append is not supported in ws v2; use response.create with previous_response_id", nil)
		default:
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, fmt.Sprintf("unsupported websocket request type: %s", eventType), nil)
		}
		originalModel := strings.TrimSpace(values[1].String())
		if originalModel == "" {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "model is required in response.create payload", nil)
		}
		promptCacheKey := strings.TrimSpace(values[2].String())
		previousResponseID := strings.TrimSpace(values[3].String())
		previousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
		if previousResponseID != "" && previousResponseIDKind == OpenAIPreviousResponseIDKindMessageID {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "previous_response_id must be a response.id (resp_*), not a message id", nil)
		}
		if turnMetadata := strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)); turnMetadata != "" {
			next, setErr := applyPayloadMutation(normalized, "client_metadata."+openAIWSTurnMetadataHeader, turnMetadata)
			if setErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", setErr)
			}
			normalized = next
		}
		mappedModel := normalizeOpenAIModelForUpstream(account, account.GetMappedModel(originalModel))
		if mappedModel != originalModel {
			next, setErr := applyPayloadMutation(normalized, "model", mappedModel)
			if setErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", setErr)
			}
			normalized = next
		}

		// Enforce OpenAI Fast/Flex policy for WS ingress (response.create only).
		serviceTier := strings.TrimSpace(gjson.GetBytes(normalized, "service_tier").String())
		if serviceTier != "" {
			decision := s.evaluateOpenAIFastPolicy(ctx, account, serviceTier, mappedModel)
			if decision.matched {
				s.logOpenAIFastPolicyDecision(ctx, account, mappedModel, serviceTier, decision, "ws_ingress")
				switch decision.action {
				case OpenAIFastPolicyActionBlock:
					return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "This request is blocked by policy", nil)
				case OpenAIFastPolicyActionFilter:
					next, deleteErr := deleteTopLevelJSONKeyBytes(normalized, "service_tier")
					if deleteErr != nil {
						return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", deleteErr)
					}
					normalized = next
				}
			}
		}
		return openAIWSClientPayload{payloadRaw: normalized, rawForHash: trimmed, promptCacheKey: promptCacheKey, previousResponseID: previousResponseID, originalModel: originalModel, payloadBytes: len(normalized)}, nil
	}
	firstPayload, err := parseClientPayload(firstClientMessage)
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
	wsHeaders, _ := s.buildOpenAIWSHeaders(c, account, token, wsDecision, isCodexCLI, turnState, strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)), firstPayload.promptCacheKey)
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
	writeClientMessage := func(message []byte) error {
		writeCtx, cancel := context.WithTimeout(ctx, s.openAIWSWriteTimeout())
		defer cancel()
		return clientConn.Write(writeCtx, coderws.MessageText, message)
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
	sendAndRelay := func(turn int, lease *openAIWSConnLease, payload []byte, payloadBytes int, originalModel string) (*OpenAIForwardResult, error) {
		if lease == nil {
			return nil, errors.New("upstream websocket lease is nil")
		}
		turnStart := time.Now()
		wroteDownstream := false
		if err := lease.WriteJSONWithContextTimeout(ctx, json.RawMessage(payload), s.openAIWSWriteTimeout()); err != nil {
			return nil, wrapOpenAIWSIngressTurnError("write_upstream", fmt.Errorf("write upstream websocket request: %w", err), false)
		}
		if debugEnabled {
			logOpenAIWSModeDebug("ingress_ws_turn_request_sent account_id=%d turn=%d conn_id=%s payload_bytes=%d", account.ID, turn, truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen), payloadBytes)
		}
		responseID := ""
		usage := OpenAIUsage{}
		var firstTokenMs *int
		reqStream := openAIWSPayloadBoolFromRaw(payload, "stream", true)
		turnPreviousResponseID := openAIWSPayloadStringFromRaw(payload, "previous_response_id")
		turnPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(turnPreviousResponseID)
		turnPromptCacheKey := openAIWSPayloadStringFromRaw(payload, "prompt_cache_key")
		turnStoreDisabled := s.isOpenAIWSStoreDisabledInRequestRaw(payload, account)
		turnHasFunctionCallOutput := gjson.GetBytes(payload, `input.#(type=="function_call_output")`).Exists()
		eventCount := 0
		tokenEventCount := 0
		terminalEventCount := 0
		firstEventType := ""
		lastEventType := ""
		needModelReplace := false
		clientDisconnected := false
		mappedModel := ""
		var mappedModelBytes []byte
		if originalModel != "" {
			mappedModel = normalizeOpenAIModelForUpstream(account, account.GetMappedModel(originalModel))
			needModelReplace = mappedModel != "" && mappedModel != originalModel
			if needModelReplace {
				mappedModelBytes = []byte(mappedModel)
			}
		}
		for {
			upstreamMessage, readErr := lease.ReadMessageWithContextTimeout(ctx, s.openAIWSReadTimeout())
			if readErr != nil {
				lease.MarkBroken()
				return nil, wrapOpenAIWSIngressTurnError("read_upstream", fmt.Errorf("read upstream websocket event: %w", readErr), wroteDownstream)
			}
			eventType, eventResponseID, _ := parseOpenAIWSEventEnvelope(upstreamMessage)
			if responseID == "" && eventResponseID != "" {
				responseID = eventResponseID
			}
			if eventType != "" {
				eventCount++
				if firstEventType == "" {
					firstEventType = eventType
				}
				lastEventType = eventType
			}
			if eventType == "error" {
				errCodeRaw, errTypeRaw, errMsgRaw := parseOpenAIWSErrorEventFields(upstreamMessage)
				s.persistOpenAIWSRateLimitSignal(ctx, account, lease.HandshakeHeaders(), upstreamMessage, errCodeRaw, errTypeRaw, errMsgRaw)
				fallbackReason, _ := classifyOpenAIWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
				errCode, errType, errMessage := summarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
				recoverablePrevNotFound := fallbackReason == openAIWSIngressStagePreviousResponseNotFound && turnPreviousResponseID != "" && !turnHasFunctionCallOutput && s.openAIWSIngressPreviousResponseRecoveryEnabled() && !wroteDownstream
				if recoverablePrevNotFound {
					logOpenAIWSModeInfo("ingress_ws_prev_response_recoverable account_id=%d turn=%d conn_id=%s idx=%d reason=%s code=%s type=%s message=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s store_disabled=%v has_prompt_cache_key=%v", account.ID, turn, truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen), eventCount, truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen), errCode, errType, errMessage, truncateOpenAIWSLogValue(turnPreviousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(turnPreviousResponseIDKind), truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen), turnStoreDisabled, turnPromptCacheKey != "")
				} else {
					logOpenAIWSModeInfo("ingress_ws_error_event account_id=%d turn=%d conn_id=%s idx=%d fallback_reason=%s err_code=%s err_type=%s err_message=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s store_disabled=%v has_prompt_cache_key=%v", account.ID, turn, truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen), eventCount, truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen), errCode, errType, errMessage, truncateOpenAIWSLogValue(turnPreviousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(turnPreviousResponseIDKind), truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen), turnStoreDisabled, turnPromptCacheKey != "")
				}
				if recoverablePrevNotFound {
					lease.MarkBroken()
					errMsg := strings.TrimSpace(errMsgRaw)
					if errMsg == "" {
						errMsg = "previous response not found"
					}
					return nil, wrapOpenAIWSIngressTurnError(openAIWSIngressStagePreviousResponseNotFound, errors.New(errMsg), false)
				}
			}
			isTokenEvent := isOpenAIWSTokenEvent(eventType)
			if isTokenEvent {
				tokenEventCount++
			}
			isTerminalEvent := isOpenAIWSTerminalEvent(eventType)
			if isTerminalEvent {
				terminalEventCount++
			}
			if firstTokenMs == nil && isTokenEvent {
				ms := int(time.Since(turnStart).Milliseconds())
				firstTokenMs = &ms
			}
			if openAIWSEventShouldParseUsage(eventType) {
				parseOpenAIWSResponseUsageFromCompletedEvent(upstreamMessage, &usage)
			}
			if !clientDisconnected {
				if needModelReplace && len(mappedModelBytes) > 0 && openAIWSEventMayContainModel(eventType) && bytes.Contains(upstreamMessage, mappedModelBytes) {
					upstreamMessage = replaceOpenAIWSMessageModel(upstreamMessage, mappedModel, originalModel)
				}
				if openAIWSEventMayContainToolCalls(eventType) && openAIWSMessageLikelyContainsToolCalls(upstreamMessage) {
					if corrected, changed := s.toolCorrector.CorrectToolCallsInSSEBytes(upstreamMessage); changed {
						upstreamMessage = corrected
					}
				}
				if err := writeClientMessage(upstreamMessage); err != nil {
					if isOpenAIWSClientDisconnectError(err) {
						clientDisconnected = true
						closeStatus, closeReason := summarizeOpenAIWSReadCloseError(err)
						logOpenAIWSModeInfo("ingress_ws_client_disconnected_drain account_id=%d turn=%d conn_id=%s close_status=%s close_reason=%s", account.ID, turn, truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen), closeStatus, truncateOpenAIWSLogValue(closeReason, openAIWSHeaderValueMaxLen))
					} else {
						return nil, wrapOpenAIWSIngressTurnError("write_client", fmt.Errorf("write client websocket event: %w", err), wroteDownstream)
					}
				} else {
					wroteDownstream = true
				}
			}
			if isTerminalEvent {
				if clientDisconnected {
					lease.MarkBroken()
				}
				firstTokenMsValue := -1
				if firstTokenMs != nil {
					firstTokenMsValue = *firstTokenMs
				}
				if debugEnabled {
					logOpenAIWSModeDebug("ingress_ws_turn_completed account_id=%d turn=%d conn_id=%s response_id=%s duration_ms=%d events=%d token_events=%d terminal_events=%d first_event=%s last_event=%s first_token_ms=%d client_disconnected=%v", account.ID, turn, truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen), time.Since(turnStart).Milliseconds(), eventCount, tokenEventCount, terminalEventCount, truncateOpenAIWSLogValue(firstEventType, openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(lastEventType, openAIWSLogValueMaxLen), firstTokenMsValue, clientDisconnected)
				}
				return &OpenAIForwardResult{RequestID: responseID, Usage: usage, Model: originalModel, UpstreamModel: mappedModel, ServiceTier: extractOpenAIServiceTierFromBody(payload), ReasoningEffort: extractOpenAIReasoningEffortFromBody(payload, originalModel), Stream: reqStream, OpenAIWSMode: true, ResponseHeaders: lease.HandshakeHeaders(), Duration: time.Since(turnStart), FirstTokenMs: firstTokenMs}, nil
			}
		}
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
	// lastTurnClean 标记最后一轮 sendAndRelay 是否正常完成（收到终端事件且客户端未断连）。
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
		hasFunctionCallOutput := gjson.GetBytes(currentPayload, `input.#(type=="function_call_output")`).Exists()
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
		result, relayErr := sendAndRelay(turn, sessionLease, currentPayload, currentPayloadBytes, currentOriginalModel)
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
		nextPayload, parseErr := parseClientPayload(nextClientMessage)
		if parseErr != nil {
			return parseErr
		}
		if nextPayload.promptCacheKey != "" {
			updatedHeaders, _ := s.buildOpenAIWSHeaders(c, account, token, wsDecision, isCodexCLI, turnState, strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)), nextPayload.promptCacheKey)
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
