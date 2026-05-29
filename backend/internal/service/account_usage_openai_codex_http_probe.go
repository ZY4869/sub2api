package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httppool "github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/google/uuid"
)

func (s *AccountUsageService) probeOpenAICodexSnapshotForModelHTTP(
	ctx context.Context,
	account *Account,
	accessToken string,
	probeModelID string,
	scope string,
) (map[string]any, *time.Time, error) {
	payload := createOpenAITestPayload(probeModelID, "", true)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal openai probe payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatgptCodexURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("create openai probe request: %w", err)
	}
	req.Host = "chatgpt.com"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	// Match official Codex client family headers so upstream applies the correct
	// per-scope quota (especially for Pro Spark).
	req.Header.Set("Originator", "codex_cli_rs")
	req.Header.Set("Version", codexCLIVersion)
	req.Header.Set("User-Agent", codexCLIUserAgent)
	req.Header.Set("Session_id", uuid.NewString())
	if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
		req.Header.Set("chatgpt-account-id", chatgptAccountID)
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	client, err := httppool.GetClient(httppool.Options{
		ProxyURL:              proxyURL,
		Timeout:               15 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("build openai probe client: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("openai codex probe request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	updates, resetAt, _, err := extractOpenAICodexProbeSnapshotForScope(resp, scope)
	if err != nil {
		return nil, nil, err
	}
	if len(updates) == 0 && resetAt == nil {
		return nil, nil, nil
	}
	logOpenAICodexHTTPSnapshot(account, probeModelID, scope, resp, updates)
	state := s.persistOpenAICodexProbeSnapshot(ctx, account, updates)
	if state != nil {
		if state.AccountResetAt != nil {
			return updates, state.AccountResetAt, nil
		}
		if state.ScopeResetAt != nil {
			return updates, state.ScopeResetAt, nil
		}
	}
	return updates, resetAt, nil
}

func logOpenAICodexHTTPSnapshot(account *Account, probeModelID, scope string, resp *http.Response, updates map[string]any) {
	util5h, util7d := extractCodexUsagePercentsFromUpdates(scope, updates)
	primaryUsed, secondaryUsed, primaryWindow, secondaryWindow, primaryResetAfter, secondaryResetAfter := codexProbeHeadersLogFields(resp.Header)
	slog.Info(
		"openai_codex_snapshot_scope_resolved",
		"account_id", account.ID,
		"requested_model", probeModelID,
		"upstream_model", probeModelID,
		"resolved_scope", scope,
		"snapshot_source", "usage_probe_http_header",
		"probe_transport", "http_sse",
		"x_cx_primary_used_percent", primaryUsed,
		"x_cx_secondary_used_percent", secondaryUsed,
		"x_cx_primary_window_minutes", primaryWindow,
		"x_cx_secondary_window_minutes", secondaryWindow,
		"x_cx_primary_reset_after_seconds", primaryResetAfter,
		"x_cx_secondary_reset_after_seconds", secondaryResetAfter,
		"utilization_5h_percent", util5h,
		"utilization_7d_percent", util7d,
	)
}
