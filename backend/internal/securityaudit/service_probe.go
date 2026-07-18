package securityaudit

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *PromptService) Probe(ctx context.Context, req ProbeRequest) ProbeResult {
	started := time.Now()
	promptAuditLog(ctx).Info("prompt_audit_endpoint_probe_started",
		zap.String("endpoint_id", strings.TrimSpace(req.Endpoint.ID)),
		zap.String("model", strings.TrimSpace(req.Endpoint.Model)),
	)
	endpoint, tokenApplied, err := s.probeEndpoint(req.Endpoint)
	if err != nil {
		result := finishProbe(started, ProbeResult{Status: "failed", ErrorCode: "endpoint_invalid", Message: "endpoint invalid"})
		s.storeProbeResult(req.Endpoint.ID, result)
		s.logProbeFinished(ctx, req.Endpoint.ID, endpoint, result)
		return result
	}
	if result, ok := s.probeModels(ctx, endpoint, tokenApplied, started); ok {
		s.storeProbeResult(endpoint.ID, result)
		s.logProbeFinished(ctx, endpoint.ID, endpoint, result)
		return result
	}
	if _, err := s.scanner.Scan(ctx, endpoint, "hello", AllScannerIDs); err == nil {
		result := finishProbe(started, ProbeResult{OK: true, Status: "healthy", Message: "scanner reachable", TokenApplied: tokenApplied})
		s.storeProbeResult(endpoint.ID, result)
		s.logProbeFinished(ctx, endpoint.ID, endpoint, result)
		return result
	}
	result := finishProbe(started, ProbeResult{Status: "failed", ErrorCode: ErrorCodeUnavailable, Message: "endpoint unavailable", Retryable: true, TokenApplied: tokenApplied})
	s.storeProbeResult(endpoint.ID, result)
	s.logProbeFinished(ctx, endpoint.ID, endpoint, result)
	return result
}

func (s *PromptService) logProbeFinished(ctx context.Context, endpointID string, endpoint ActiveEndpoint, result ProbeResult) {
	level := promptAuditLog(ctx)
	fields := []zap.Field{
		zap.String("endpoint_id", strings.TrimSpace(endpointID)),
		zap.String("model", strings.TrimSpace(endpoint.Model)),
		zap.String("status", strings.TrimSpace(result.Status)),
		zap.String("error_code", strings.TrimSpace(result.ErrorCode)),
		zap.Int("http_status", result.HTTPStatus),
		zap.Bool("retryable", result.Retryable),
		zap.Bool("token_applied", result.TokenApplied),
		zap.Int("latency_ms", result.LatencyMS),
	}
	if result.OK {
		level.Info("prompt_audit_endpoint_probe_succeeded", fields...)
		return
	}
	level.Warn("prompt_audit_endpoint_probe_failed", fields...)
}

func (s *PromptService) probeModels(ctx context.Context, endpoint ActiveEndpoint, tokenApplied bool, started time.Time) (ProbeResult, bool) {
	modelsURL, _ := ModelsURL(endpoint.BaseURL)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if endpoint.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+endpoint.Token)
	}
	client, err := NewSecureHTTPClient(endpoint)
	if err != nil {
		return finishProbe(started, ProbeResult{Status: "failed", ErrorCode: "endpoint_invalid", Message: "endpoint invalid"}), true
	}
	resp, err := client.Do(httpReq)
	if err != nil || resp == nil {
		return ProbeResult{}, false
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return finishProbe(started, ProbeResult{OK: true, Status: "healthy", Message: "endpoint reachable", HTTPStatus: resp.StatusCode, TokenApplied: tokenApplied}), true
	}
	return ProbeResult{}, false
}

func (s *PromptService) probeEndpoint(input UpdateEndpoint) (ActiveEndpoint, bool, error) {
	baseURL, err := NormalizeBaseURL(input.BaseURL)
	if err != nil {
		return ActiveEndpoint{}, false, err
	}
	token, tokenApplied := s.probeToken(input, baseURL)
	model := firstNonEmpty(input.Model, DefaultGuardModel)
	return ActiveEndpoint{
		ID: firstNonEmpty(input.ID, "probe"), Name: input.Name, Protocol: "openai_compatible",
		BaseURL: baseURL, Model: model, Token: token, TimeoutMS: input.TimeoutMS,
		InputLimit: input.InputLimit, Enabled: true,
	}, tokenApplied, nil
}

func (s *PromptService) probeToken(input UpdateEndpoint, baseURL string) (string, bool) {
	token := strings.TrimSpace(input.Token)
	if token != "" {
		return token, true
	}
	if cfg, ok := s.config.Active(); ok {
		for _, ep := range cfg.Endpoints {
			if ep.ID == strings.TrimSpace(input.ID) && ep.BaseURL == baseURL {
				return ep.Token, ep.Token != ""
			}
		}
	}
	return "", false
}

func finishProbe(started time.Time, result ProbeResult) ProbeResult {
	result.CheckedAt = time.Now().UTC()
	result.LatencyMS = int(result.CheckedAt.Sub(started).Milliseconds())
	return result
}

func (s *PromptService) storeProbeResult(endpointID string, result ProbeResult) {
	if s == nil || endpointID == "" {
		return
	}
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	if s.probes == nil {
		s.probes = map[string]ProbeResult{}
	}
	s.probes[endpointID] = result
}

func (s *PromptService) probeSnapshot() map[string]ProbeResult {
	if s == nil {
		return map[string]ProbeResult{}
	}
	s.probeMu.RLock()
	defer s.probeMu.RUnlock()
	out := make(map[string]ProbeResult, len(s.probes))
	for id, result := range s.probes {
		out[id] = result
	}
	return out
}
