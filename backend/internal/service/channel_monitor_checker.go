package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const channelMonitorMaxResponseBytes = 1 << 20 // 1MB

type channelMonitorHTTPChecker struct {
	cfg    *config.Config
	client *http.Client
}

func newChannelMonitorHTTPChecker(cfg *config.Config) *channelMonitorHTTPChecker {
	return &channelMonitorHTTPChecker{
		cfg: cfg,
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func (c *channelMonitorHTTPChecker) Check(ctx context.Context, monitor *ChannelMonitor, modelID string, apiKey string) *channelMonitorCheckResult {
	startedAt := time.Now()
	result := &channelMonitorCheckResult{
		Status:    ChannelMonitorStatusFailure,
		StartedAt: startedAt,
	}
	if ctx == nil {
		ctx = context.Background()
	}

	endpoint := strings.TrimSpace(monitor.Endpoint)
	if endpoint == "" {
		result.ErrorMessage = "endpoint is required"
		result.FinishedAt = time.Now()
		return result
	}

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Hostname() == "" {
		result.ErrorMessage = "invalid endpoint url"
		result.FinishedAt = time.Now()
		return result
	}

	if c.cfg != nil && !c.cfg.Security.URLAllowlist.AllowPrivateHosts {
		if err := urlvalidator.ValidateResolvedIP(parsedEndpoint.Hostname()); err != nil {
			result.ErrorMessage = "resolved ip is not allowed"
			result.FinishedAt = time.Now()
			return result
		}
	}

	challenge := randomChallenge()
	path, payload, requireChallenge, err := buildChannelMonitorRequest(monitor, modelID, challenge)
	if err != nil {
		result.ErrorMessage = err.Error()
		result.FinishedAt = time.Now()
		return result
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		result.ErrorMessage = "invalid request body"
		result.FinishedAt = time.Now()
		return result
	}

	fullURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		result.ErrorMessage = "failed to build request"
		result.FinishedAt = time.Now()
		return result
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Sub2API-ChannelMonitor/1.0")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	for k, v := range monitor.ExtraHeaders {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	if monitor.Provider == ChannelMonitorProviderAnthropic || monitor.Provider == ChannelMonitorProviderAntigravity {
		if req.Header.Get("anthropic-version") == "" {
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		result.ErrorMessage = "request failed"
		result.FinishedAt = time.Now()
		result.LatencyMs = int64(result.FinishedAt.Sub(startedAt) / time.Millisecond)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	httpStatus := resp.StatusCode
	result.HTTPStatus = &httpStatus

	raw, readErr := readLimited(resp.Body, channelMonitorMaxResponseBytes)
	if readErr != nil {
		result.ErrorMessage = "response too large"
		result.FinishedAt = time.Now()
		result.LatencyMs = int64(result.FinishedAt.Sub(startedAt) / time.Millisecond)
		return result
	}

	text := extractChannelMonitorResponseText(monitor.Provider, modelID, raw)
	if httpStatus < 200 || httpStatus >= 300 {
		if text == "" {
			text = extractChannelMonitorErrorMessage(raw)
		}
		result.ErrorMessage = "http_error"
		result.ResponseText = truncateText(text, 512)
		result.FinishedAt = time.Now()
		result.LatencyMs = int64(result.FinishedAt.Sub(startedAt) / time.Millisecond)
		return result
	}

	if strings.TrimSpace(text) == "" {
		result.ErrorMessage = "empty_response"
		result.FinishedAt = time.Now()
		result.LatencyMs = int64(result.FinishedAt.Sub(startedAt) / time.Millisecond)
		return result
	}

	if requireChallenge && !strings.Contains(text, challenge) {
		logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitor] challenge mismatch: monitor_id=%d provider=%s model=%s", monitor.ID, monitor.Provider, modelID)
		result.ErrorMessage = "challenge_mismatch"
		result.ResponseText = truncateText(text, 512)
		result.FinishedAt = time.Now()
		result.LatencyMs = int64(result.FinishedAt.Sub(startedAt) / time.Millisecond)
		return result
	}

	result.ResponseText = truncateText(text, 512)
	result.FinishedAt = time.Now()
	result.LatencyMs = int64(result.FinishedAt.Sub(startedAt) / time.Millisecond)
	result.Status = ChannelMonitorStatusSuccess
	if result.LatencyMs > channelMonitorDegradedThreshold {
		result.Status = ChannelMonitorStatusDegraded
	}
	return result
}
