package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const (
	openAICodexWSProbeDialTimeout     = 10 * time.Second
	openAICodexWSProbeWriteTimeout    = 5 * time.Second
	openAICodexWSProbeReadTimeout     = 8 * time.Second
	openAICodexWSProbeMaxReadMessages = 64
)

type openAICodexWSProbeDiagnostics struct {
	ReadMessages   int
	LastEventType  string
	ReadExitReason string
}

func (s *AccountUsageService) probeOpenAICodexSnapshotForModelWS(
	ctx context.Context,
	account *Account,
	probeModelID string,
	scope string,
) (map[string]any, *time.Time, *OpenAICodexUsageSnapshot, string, *openAICodexWSProbeDiagnostics, error) {
	if account == nil || !isChatGPTOpenAIOAuthAccount(account) {
		return nil, nil, nil, "", nil, nil
	}
	accessToken := account.GetOpenAIAccessToken()
	if strings.TrimSpace(accessToken) == "" {
		return nil, nil, nil, "", nil, errors.New("openai codex ws probe missing access token")
	}
	wsURL, err := chatgptCodexWSURL()
	if err != nil {
		return nil, nil, nil, "", nil, err
	}

	dialer := openAIWSClientDialer(nil)
	if s != nil {
		dialer = s.openAICodexWSProbeDialer
	}
	if dialer == nil {
		dialer = newDefaultOpenAIWSClientDialer()
	}
	headers := buildCodexWSProbeHeaders(account, accessToken)
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	dialCtx, cancelDial := context.WithTimeout(ctx, openAICodexWSProbeDialTimeout)
	defer cancelDial()
	conn, _, handshakeHeaders, err := dialer.Dial(dialCtx, wsURL, headers, proxyURL)
	if err != nil {
		return nil, nil, nil, "", nil, err
	}
	defer func() { _ = conn.Close() }()

	payload := buildCodexWSProbeCreatePayload(probeModelID)
	writeCtx, cancelWrite := context.WithTimeout(ctx, openAICodexWSProbeWriteTimeout)
	defer cancelWrite()
	if err := conn.WriteJSON(writeCtx, payload); err != nil {
		return nil, nil, nil, "", nil, err
	}

	readTimeout := openAICodexWSProbeReadTimeout
	if s != nil && s.openAICodexWSProbeReadTimeoutOverride > 0 {
		readTimeout = s.openAICodexWSProbeReadTimeoutOverride
	}
	readCtx, cancelRead := context.WithTimeout(ctx, readTimeout)
	defer cancelRead()
	diag := &openAICodexWSProbeDiagnostics{}
	for i := 0; i < openAICodexWSProbeMaxReadMessages; i++ {
		msg, err := conn.ReadMessage(readCtx)
		if err != nil {
			diag.ReadExitReason = "read_error"
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				diag.ReadExitReason = "timeout"
			}
			break
		}
		diag.ReadMessages++
		if eventType := strings.TrimSpace(gjson.GetBytes(msg, "type").String()); eventType != "" {
			diag.LastEventType = eventType
			if isOpenAIWSTerminalEvent(eventType) {
				diag.ReadExitReason = "terminal_event"
			}
		}
		if snapshot := parseCodexRateLimitsFromWSMessage(msg, time.Now()); snapshot != nil {
			baseTime := time.Now()
			updates := buildCodexUsageExtraUpdatesForScope(scope, snapshot, baseTime)
			resetAt := codexRateLimitResetAtFromSnapshot(snapshot, baseTime)
			diag.ReadExitReason = "event_found"
			return updates, resetAt, snapshot, "usage_probe_ws_event", diag, nil
		}
		if diag.ReadExitReason == "terminal_event" {
			break
		}
	}
	if diag.ReadExitReason == "" && diag.ReadMessages == openAICodexWSProbeMaxReadMessages {
		diag.ReadExitReason = "read_error"
	}

	if handshakeSnapshot := ParseCodexRateLimitHeaders(handshakeHeaders); handshakeSnapshot != nil {
		baseTime := time.Now()
		updates := buildCodexUsageExtraUpdatesForScope(scope, handshakeSnapshot, baseTime)
		resetAt := codexRateLimitResetAtFromSnapshot(handshakeSnapshot, baseTime)
		if diag.ReadExitReason == "" {
			diag.ReadExitReason = "read_error"
		}
		return updates, resetAt, handshakeSnapshot, "usage_probe_ws_handshake_header", diag, nil
	}

	return nil, nil, nil, "", diag, nil
}

func chatgptCodexWSURL() (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(chatgptCodexURL))
	if err != nil {
		return "", fmt.Errorf("parse chatgpt codex url: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	case "wss", "ws":
		// ok
	default:
		return "", fmt.Errorf("unsupported scheme for codex ws url: %s", parsed.Scheme)
	}
	return parsed.String(), nil
}

func buildCodexWSProbeHeaders(account *Account, accessToken string) http.Header {
	headers := make(http.Header)
	headers.Set("authorization", "Bearer "+strings.TrimSpace(accessToken))
	headers.Set("content-type", "application/json")
	headers.Set("openai-beta", openAIWSBetaV2Value)
	headers.Set("originator", "codex_cli_rs")
	headers.Set("version", codexCLIVersion)
	headers.Set("user-agent", codexCLIUserAgent)
	headers.Set("session_id", uuid.NewString())
	if account != nil {
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			headers.Set("chatgpt-account-id", chatgptAccountID)
		}
	}
	return headers
}

func buildCodexWSProbeCreatePayload(modelID string) map[string]any {
	modelID = strings.TrimSpace(modelID)
	payload := createOpenAITestPayload(modelID, true)
	payload["type"] = "response.create"
	payload["max_output_tokens"] = 1
	payload["generate"] = false
	// Ensure instructions exist; some upstream paths reject empty instructions even for probes.
	if _, ok := payload["instructions"]; !ok {
		payload["instructions"] = openai.DefaultInstructions
	}
	return payload
}
