package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

func parseOpenAICodexRateLimitsResult(raw json.RawMessage, now time.Time) (*OpenAICodexAppServerRateLimitsSnapshot, error) {
	if len(raw) == 0 {
		return nil, infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_INVALID_RESPONSE", "Codex app-server 返回空响应")
	}
	var payload struct {
		RateLimits            json.RawMessage `json:"rateLimits"`
		RateLimitsByLimitID   json.RawMessage `json:"rateLimitsByLimitId"`
		RateLimitResetCredits *struct {
			AvailableCount any `json:"availableCount"`
		} `json:"rateLimitResetCredits"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_INVALID_RESPONSE", err)
	}

	var count *int
	if payload.RateLimitResetCredits != nil {
		parsed, ok := parseOpenAIResetCreditsAvailableCount(payload.RateLimitResetCredits.AvailableCount)
		if ok {
			count = &parsed
		}
	}
	updatedAt := now.UTC()
	status := openAIResetCreditsStatusUnknownOrUnsupported
	updates := map[string]any{
		openAIRateLimitsAppServerUpdatedAtExtraKey:  updatedAt.Format(time.RFC3339),
		openAIResetCreditsStatusExtraKey:            status,
		openAIResetCreditsAvailableCountExtraKey:    nil,
		openAIResetCreditsUpdatedAtExtraKey:         nil,
		openAIResetCreditsUnsupportedReasonExtraKey: nil,
	}
	if count != nil {
		status = openAIResetCreditsStatusAvailable
		updates[openAIResetCreditsStatusExtraKey] = status
		updates[openAIResetCreditsAvailableCountExtraKey] = *count
		updates[openAIResetCreditsUpdatedAtExtraKey] = updatedAt.Format(time.RFC3339)
	}
	return &OpenAICodexAppServerRateLimitsSnapshot{
		AvailableCount:      count,
		UpdatedAt:           updatedAt,
		Status:              status,
		RateLimits:          cloneJSONRawMessage(payload.RateLimits),
		RateLimitsByLimitID: cloneJSONRawMessage(payload.RateLimitsByLimitID),
		ExtraUpdates:        updates,
	}, nil
}

func cloneJSONRawMessage(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

func parseOpenAIResetCreditsAvailableCount(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0, false
		}
		return int(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return v, true
	case json.Number:
		i, err := v.Int64()
		if err != nil || i < 0 {
			return 0, false
		}
		return int(i), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		i, err := json.Number(trimmed).Int64()
		if err != nil || i < 0 {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func parseOpenAICodexResetCreditConsumeStatus(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_INVALID_RESPONSE", "Codex app-server 返回空响应")
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_INVALID_RESPONSE", err)
	}
	status := strings.TrimSpace(payload.Status)
	switch status {
	case openAIResetCreditConsumeStatusReset,
		openAIResetCreditConsumeStatusAlreadyRedeemed,
		openAIResetCreditConsumeStatusNothingToReset,
		openAIResetCreditConsumeStatusNoCredit:
		return status, nil
	default:
		return "", infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_INVALID_RESPONSE", "Codex app-server 返回未知重置结果")
	}
}

func openAICodexJSONRPCApplicationError(method string, rpcErr *openAICodexJSONRPCError) error {
	if rpcErr == nil {
		return infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_RPC_ERROR", "Codex app-server 调用失败")
	}
	if method == "account/rateLimitResetCredit/consume" && isOpenAICodexMethodNotFoundError(rpcErr) {
		return openAICodexResetCreditsUnsupportedError()
	}
	metadata := map[string]string{
		"rpc_code": fmt.Sprint(rpcErr.Code),
	}
	return infraerrors.New(http.StatusBadGateway, "OPENAI_CODEX_APP_SERVER_RPC_ERROR", "Codex app-server 调用失败").WithMetadata(metadata)
}

func isOpenAICodexMethodNotFoundError(rpcErr *openAICodexJSONRPCError) bool {
	if rpcErr == nil {
		return false
	}
	if rpcErr.Code == -32601 {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(rpcErr.Message))
	return strings.Contains(message, "method not found") || strings.Contains(message, "unknown method")
}

func openAICodexResetCreditsUnsupportedError() error {
	return infraerrors.New(
		http.StatusNotImplemented,
		"OPENAI_CODEX_RESET_CREDITS_UNSUPPORTED",
		"当前 Codex app-server 不支持 OpenAI 官方真实重置",
	)
}

func openAICodexAppServerTimeoutError(err error) error {
	if err == nil {
		err = context.DeadlineExceeded
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return infraerrors.GatewayTimeout("OPENAI_CODEX_APP_SERVER_TIMEOUT", "Codex app-server 调用超时")
	}
	return sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_UNAVAILABLE", err)
}

func sanitizeOpenAICodexAppServerError(reason string, err error) error {
	if err == nil {
		return infraerrors.ServiceUnavailable(reason, "Codex app-server 不可用")
	}
	msg := strings.TrimSpace(logredact.RedactText(err.Error()))
	if msg == "" {
		msg = "Codex app-server 不可用"
	}
	return infraerrors.ServiceUnavailable(reason, fmt.Sprintf("Codex app-server 不可用: %s", msg))
}
