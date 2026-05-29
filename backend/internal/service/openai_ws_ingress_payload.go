package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type openAIWSClientPayload struct {
	payloadRaw         []byte
	rawForHash         []byte
	promptCacheKey     string
	previousResponseID string
	originalModel      string
	payloadBytes       int
}

func (s *OpenAIGatewayService) parseOpenAIWSClientPayload(ctx context.Context, c *gin.Context, account *Account, raw []byte) (openAIWSClientPayload, error) {
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
		next, setErr := applyOpenAIWSClientPayloadMutation(normalized, "type", eventType)
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
	if c != nil {
		if turnMetadata := strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)); turnMetadata != "" {
			next, setErr := applyOpenAIWSClientPayloadMutation(normalized, "client_metadata."+openAIWSTurnMetadataHeader, turnMetadata)
			if setErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", setErr)
			}
			normalized = next
		}
	}
	mappedModel := normalizeOpenAIModelForUpstream(account, account.GetMappedModel(originalModel))
	if mappedModel != originalModel {
		next, setErr := applyOpenAIWSClientPayloadMutation(normalized, "model", mappedModel)
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

func applyOpenAIWSClientPayloadMutation(current []byte, path string, value any) ([]byte, error) {
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
