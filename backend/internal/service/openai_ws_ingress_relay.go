package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	coderws "github.com/coder/websocket"
)

type openAIWSIngressTurnRelayInput struct {
	ctx           context.Context
	account       *Account
	clientConn    *coderws.Conn
	lease         *openAIWSConnLease
	payload       []byte
	payloadBytes  int
	originalModel string
	turn          int
	debugEnabled  bool
}

func (s *OpenAIGatewayService) relayOpenAIWSIngressTurn(input openAIWSIngressTurnRelayInput) (*OpenAIForwardResult, error) {
	if input.lease == nil {
		return nil, errors.New("upstream websocket lease is nil")
	}
	turnStart := time.Now()
	wroteDownstream := false
	if err := input.lease.WriteJSONWithContextTimeout(input.ctx, json.RawMessage(input.payload), s.openAIWSWriteTimeout()); err != nil {
		return nil, wrapOpenAIWSIngressTurnError("write_upstream", fmt.Errorf("write upstream websocket request: %w", err), false)
	}
	account := input.account
	turn := input.turn
	if input.debugEnabled {
		logOpenAIWSModeDebug("ingress_ws_turn_request_sent account_id=%d turn=%d conn_id=%s payload_bytes=%d", account.ID, turn, truncateOpenAIWSLogValue(input.lease.ConnID(), openAIWSIDValueMaxLen), input.payloadBytes)
	}
	responseID := ""
	usage := OpenAIUsage{}
	var firstTokenMs *int
	reqStream := openAIWSPayloadBoolFromRaw(input.payload, "stream", true)
	turnPreviousResponseID := openAIWSPayloadStringFromRaw(input.payload, "previous_response_id")
	turnPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(turnPreviousResponseID)
	turnPromptCacheKey := openAIWSPayloadStringFromRaw(input.payload, "prompt_cache_key")
	turnStoreDisabled := s.isOpenAIWSStoreDisabledInRequestRaw(input.payload, account)
	turnHasFunctionCallOutput := openAIWSRawPayloadHasToolCallOutput(input.payload)
	eventCount := 0
	tokenEventCount := 0
	terminalEventCount := 0
	firstEventType := ""
	lastEventType := ""
	needModelReplace := false
	clientDisconnected := false
	mappedModel := ""
	var mappedModelBytes []byte
	if input.originalModel != "" {
		mappedModel = normalizeOpenAIModelForUpstream(account, account.GetMappedModel(input.originalModel))
		needModelReplace = mappedModel != "" && mappedModel != input.originalModel
		if needModelReplace {
			mappedModelBytes = []byte(mappedModel)
		}
	}
	for {
		upstreamMessage, readErr := input.lease.ReadMessageWithContextTimeout(input.ctx, s.openAIWSReadTimeout())
		if readErr != nil {
			input.lease.MarkBroken()
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
			s.persistOpenAIWSRateLimitSignal(input.ctx, account, input.lease.HandshakeHeaders(), upstreamMessage, errCodeRaw, errTypeRaw, errMsgRaw)
			fallbackReason, _ := classifyOpenAIWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			errCode, errType, errMessage := summarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			recoverablePrevNotFound := fallbackReason == openAIWSIngressStagePreviousResponseNotFound && turnPreviousResponseID != "" && !turnHasFunctionCallOutput && s.openAIWSIngressPreviousResponseRecoveryEnabled() && !wroteDownstream
			if recoverablePrevNotFound {
				logOpenAIWSModeInfo("ingress_ws_prev_response_recoverable account_id=%d turn=%d conn_id=%s idx=%d reason=%s code=%s type=%s message=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s store_disabled=%v has_prompt_cache_key=%v", account.ID, turn, truncateOpenAIWSLogValue(input.lease.ConnID(), openAIWSIDValueMaxLen), eventCount, truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen), errCode, errType, errMessage, truncateOpenAIWSLogValue(turnPreviousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(turnPreviousResponseIDKind), truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen), turnStoreDisabled, turnPromptCacheKey != "")
			} else {
				logOpenAIWSModeInfo("ingress_ws_error_event account_id=%d turn=%d conn_id=%s idx=%d fallback_reason=%s err_code=%s err_type=%s err_message=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s store_disabled=%v has_prompt_cache_key=%v", account.ID, turn, truncateOpenAIWSLogValue(input.lease.ConnID(), openAIWSIDValueMaxLen), eventCount, truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen), errCode, errType, errMessage, truncateOpenAIWSLogValue(turnPreviousResponseID, openAIWSIDValueMaxLen), normalizeOpenAIWSLogValue(turnPreviousResponseIDKind), truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen), turnStoreDisabled, turnPromptCacheKey != "")
			}
			if recoverablePrevNotFound {
				input.lease.MarkBroken()
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
				upstreamMessage = replaceOpenAIWSMessageModel(upstreamMessage, mappedModel, input.originalModel)
			}
			if openAIWSEventMayContainToolCalls(eventType) && openAIWSMessageLikelyContainsToolCalls(upstreamMessage) {
				if corrected, changed := s.toolCorrector.CorrectToolCallsInSSEBytes(upstreamMessage); changed {
					upstreamMessage = corrected
				}
			}
			if err := s.writeOpenAIWSClientMessage(input.ctx, input.clientConn, upstreamMessage); err != nil {
				if isOpenAIWSClientDisconnectError(err) {
					clientDisconnected = true
					closeStatus, closeReason := summarizeOpenAIWSReadCloseError(err)
					logOpenAIWSModeInfo("ingress_ws_client_disconnected_drain account_id=%d turn=%d conn_id=%s close_status=%s close_reason=%s", account.ID, turn, truncateOpenAIWSLogValue(input.lease.ConnID(), openAIWSIDValueMaxLen), closeStatus, truncateOpenAIWSLogValue(closeReason, openAIWSHeaderValueMaxLen))
				} else {
					return nil, wrapOpenAIWSIngressTurnError("write_client", fmt.Errorf("write client websocket event: %w", err), wroteDownstream)
				}
			} else {
				wroteDownstream = true
			}
		}
		if isTerminalEvent {
			if clientDisconnected {
				input.lease.MarkBroken()
			}
			firstTokenMsValue := -1
			if firstTokenMs != nil {
				firstTokenMsValue = *firstTokenMs
			}
			if input.debugEnabled {
				logOpenAIWSModeDebug("ingress_ws_turn_completed account_id=%d turn=%d conn_id=%s response_id=%s duration_ms=%d events=%d token_events=%d terminal_events=%d first_event=%s last_event=%s first_token_ms=%d client_disconnected=%v", account.ID, turn, truncateOpenAIWSLogValue(input.lease.ConnID(), openAIWSIDValueMaxLen), truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen), time.Since(turnStart).Milliseconds(), eventCount, tokenEventCount, terminalEventCount, truncateOpenAIWSLogValue(firstEventType, openAIWSLogValueMaxLen), truncateOpenAIWSLogValue(lastEventType, openAIWSLogValueMaxLen), firstTokenMsValue, clientDisconnected)
			}
			effortResolution := extractOpenAIReasoningEffortResolutionFromBody(input.payload, input.originalModel)
			return &OpenAIForwardResult{
				RequestID:                responseID,
				Usage:                    usage,
				Model:                    input.originalModel,
				UpstreamModel:            mappedModel,
				ServiceTier:              extractOpenAIServiceTierFromBody(input.payload),
				ReasoningEffort:          effortResolution.Effective,
				ReasoningEffortRaw:       effortResolution.Raw,
				ReasoningEffortEffective: effortResolution.Effective,
				ReasoningEffortSource:    effortResolution.Source,
				Stream:                   reqStream,
				OpenAIWSMode:             true,
				ResponseHeaders:          input.lease.HandshakeHeaders(),
				Duration:                 time.Since(turnStart),
				FirstTokenMs:             firstTokenMs,
			}, nil
		}
	}
}

func (s *OpenAIGatewayService) writeOpenAIWSClientMessage(ctx context.Context, clientConn *coderws.Conn, message []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, s.openAIWSWriteTimeout())
	defer cancel()
	return clientConn.Write(writeCtx, coderws.MessageText, message)
}
