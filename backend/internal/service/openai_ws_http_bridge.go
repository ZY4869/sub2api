package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func (s *OpenAIGatewayService) shouldOpenAIWSHTTPBridgeFirstPayload(firstClientMessage []byte) bool {
	if s == nil || s.cfg == nil {
		return false
	}
	wsCfg := s.cfg.Gateway.OpenAIWS
	if !wsCfg.HTTPBridgeEnabled || wsCfg.HTTPBridgeThresholdBytes <= 0 {
		return false
	}
	return int64(len(firstClientMessage)) >= wsCfg.HTTPBridgeThresholdBytes
}

func (s *OpenAIGatewayService) ShouldUseOpenAIWSHTTPBridgeForIngress(firstClientMessage []byte) bool {
	return s.shouldOpenAIWSHTTPBridgeFirstPayload(firstClientMessage)
}

func (s *OpenAIGatewayService) proxyResponsesWebSocketHTTPBridge(ctx context.Context, c *gin.Context, clientConn *coderws.Conn, account *Account, token string, firstClientMessage []byte, hooks *OpenAIWSIngressHooks, reason string) error {
	if c == nil || c.Request == nil {
		return errors.New("gin request is nil")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unspecified"
	}
	logOpenAIWSModeInfo("ingress_http_bridge_start account_id=%d account_type=%s reason=%s first_payload_bytes=%d", account.ID, account.Type, normalizeOpenAIWSLogValue(reason), len(firstClientMessage))

	currentMessage := firstClientMessage
	for turn := 1; ; turn++ {
		if hooks != nil && hooks.BeforeTurn != nil {
			if err := hooks.BeforeTurn(turn); err != nil {
				return err
			}
		}
		result, turnErr := s.forwardOpenAIWSHTTPBridgeTurn(ctx, c, clientConn, account, currentMessage)
		if hooks != nil && hooks.AfterTurn != nil {
			hooks.AfterTurn(turn, result, turnErr)
		}
		if turnErr != nil {
			return turnErr
		}
		msgType, nextMessage, readErr := clientConn.Read(ctx)
		if readErr != nil {
			if isOpenAIWSClientDisconnectError(readErr) {
				logOpenAIWSModeInfo("ingress_http_bridge_client_closed account_id=%d turn=%d", account.ID, turn)
				return nil
			}
			return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "failed to read websocket request", readErr)
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			return NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "unsupported websocket client message type", nil)
		}
		currentMessage = nextMessage
	}
}

func (s *OpenAIGatewayService) forwardOpenAIWSHTTPBridgeTurn(ctx context.Context, c *gin.Context, clientConn *coderws.Conn, account *Account, payload []byte) (*OpenAIForwardResult, error) {
	bridgeBody, err := openAIWSHTTPBridgeRequestBody(c, account, payload)
	if err != nil {
		return nil, err
	}
	rec := httptest.NewRecorder()
	bridgeCtx, _ := gin.CreateTestContext(rec)
	bridgeCtx.Request = c.Request.Clone(ctx)
	bridgeCtx.Request = bridgeCtx.Request.WithContext(ctx)
	bridgeCtx.Request.Body = nil
	bridgeCtx.Request.ContentLength = int64(len(bridgeBody))
	bridgeCtx.Request.Header = c.Request.Header.Clone()
	for _, key := range []string{"Upgrade", "Connection", "Sec-WebSocket-Key", "Sec-WebSocket-Version", "Sec-WebSocket-Extensions", "Sec-WebSocket-Protocol"} {
		bridgeCtx.Request.Header.Del(key)
	}
	cloneGinKeys(c, bridgeCtx)
	SetOpenAIClientTransport(bridgeCtx, OpenAIClientTransportHTTP)
	result, forwardErr := s.Forward(ctx, bridgeCtx, account, bridgeBody)
	if writeErr := writeOpenAIWSHTTPBridgeRecorder(ctx, clientConn, rec); writeErr != nil {
		return result, writeErr
	}
	return result, forwardErr
}

func openAIWSHTTPBridgeRequestBody(c *gin.Context, account *Account, payload []byte) ([]byte, error) {
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", nil)
	}
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	switch eventType {
	case "", "response.create":
	default:
		return nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "unsupported websocket request type", nil)
	}
	body := payload
	if account != nil && account.IsOpenAIOAuth() && isOpenAIResponsesLiteRequest(c, nil, body) {
		next, changed, liteErr := normalizeOpenAIResponsesLiteToolsPayload(body)
		if liteErr != nil {
			return nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, liteErr.Error(), liteErr)
		}
		if changed {
			body = next
		}
	}
	if gjson.GetBytes(body, "type").Exists() {
		var err error
		body, err = sjson.DeleteBytes(body, "type")
		if err != nil {
			return nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", err)
		}
	}
	var err error
	body, err = sjson.SetBytes(body, "stream", true)
	if err != nil {
		return nil, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", err)
	}
	return body, nil
}

func writeOpenAIWSHTTPBridgeRecorder(ctx context.Context, clientConn *coderws.Conn, rec *httptest.ResponseRecorder) error {
	if rec == nil {
		return errors.New("bridge response recorder is nil")
	}
	body := rec.Body.Bytes()
	if len(body) == 0 {
		return nil
	}
	sentSSEData := false
	for _, line := range strings.Split(string(body), "\n") {
		data, ok := extractOpenAISSEDataLine(line)
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}
		sentSSEData = true
		if err := clientConn.Write(ctx, coderws.MessageText, []byte(data)); err != nil {
			if isOpenAIWSClientDisconnectError(err) {
				return nil
			}
			return NewOpenAIWSClientCloseError(coderws.StatusAbnormalClosure, "failed to write websocket response", err)
		}
	}
	if sentSSEData {
		return nil
	}
	if gjson.ValidBytes(body) {
		if err := clientConn.Write(ctx, coderws.MessageText, body); err != nil {
			if isOpenAIWSClientDisconnectError(err) {
				return nil
			}
			return NewOpenAIWSClientCloseError(coderws.StatusAbnormalClosure, "failed to write websocket response", err)
		}
	}
	return nil
}

func cloneGinKeys(src, dst *gin.Context) {
	if src == nil || dst == nil || len(src.Keys) == 0 {
		return
	}
	dst.Keys = make(map[string]any, len(src.Keys))
	for k, v := range src.Keys {
		dst.Keys[k] = v
	}
}
