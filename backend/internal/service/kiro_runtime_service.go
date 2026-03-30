package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const (
	kiroRuntimeProbeSource   = "kiro_runtime_probe"
	kiroRuntimeRequestSource = "kiro_runtime_request"
)

type KiroRuntimeProbeResult struct {
	Region           string
	Endpoint         KiroEndpointConfig
	ProfileARN       string
	FallbackUsed     bool
	ResolvedUpstream ResolvedUpstreamInfo
}

type KiroRuntimeService struct {
	accountRepo                  AccountRepository
	httpUpstream                 HTTPUpstream
	claudeTokenProvider          *ClaudeTokenProvider
	tlsFingerprintProfileService *TLSFingerprintProfileService
}

func NewKiroRuntimeService(accountRepo AccountRepository, httpUpstream HTTPUpstream, claudeTokenProvider *ClaudeTokenProvider) *KiroRuntimeService {
	return &KiroRuntimeService{
		accountRepo:         accountRepo,
		httpUpstream:        httpUpstream,
		claudeTokenProvider: claudeTokenProvider,
	}
}

func (s *KiroRuntimeService) SetTLSFingerprintProfileService(tlsFingerprintProfileService *TLSFingerprintProfileService) {
	s.tlsFingerprintProfileService = tlsFingerprintProfileService
}

func (s *KiroRuntimeService) Probe(ctx context.Context, account *Account) (*KiroRuntimeProbeResult, error) {
	if account == nil || account.Platform != PlatformKiro {
		return nil, infraerrors.BadRequest("KIRO_RUNTIME_INVALID_ACCOUNT", "kiro runtime probe requires a kiro account")
	}
	if NormalizeKiroAccountCredentials(account) {
		s.persistKiroAccount(ctx, account)
	}

	token, err := s.resolveKiroAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	profileARN := s.ensureKiroProfileARN(ctx, account, token)
	region := ResolveKiroAPIRegion(account)
	payload, err := buildKiroClaudePayload(
		[]byte(`{"messages":[{"role":"user","content":"ping"}],"max_tokens":1}`),
		"claude-haiku-4.5",
		effectiveKiroProfileARN(account, profileARN),
		kiroPrimaryOrigin,
		nil,
	)
	if err != nil {
		return nil, infraerrors.InternalServer("KIRO_RUNTIME_PROBE_PAYLOAD_FAILED", "failed to build kiro runtime probe payload").WithCause(err)
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)

	var lastErr error
	for idx, endpoint := range buildKiroEndpointConfigs(region) {
		fallbackUsed := idx > 0
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(payload))
		if err != nil {
			return nil, infraerrors.InternalServer("KIRO_RUNTIME_PROBE_REQUEST_BUILD_FAILED", "failed to build kiro runtime probe request").WithCause(err)
		}
		s.applyKiroRequestHeaders(req, account, token, endpoint)

		resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
		if err != nil {
			lastErr = err
			s.logKiroRuntimeAttempt("kiro_runtime_probe_request_failed", account, region, endpoint, fallbackUsed, profileARN, err, 0)
			if shouldKiroFallbackEndpoint(idx, 0, err) {
				s.logKiroRuntimeAttempt("kiro_runtime_probe_endpoint_fallback", account, region, endpoint, fallbackUsed, profileARN, err, 0)
				continue
			}
			return nil, infraerrors.ServiceUnavailable("KIRO_RUNTIME_PROBE_FAILED", "failed to verify kiro runtime endpoint").WithCause(err)
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			s.logKiroRuntimeAttempt("kiro_runtime_probe_read_failed", account, region, endpoint, fallbackUsed, profileARN, readErr, resp.StatusCode)
			if shouldKiroFallbackEndpoint(idx, http.StatusBadGateway, readErr) {
				s.logKiroRuntimeAttempt("kiro_runtime_probe_endpoint_fallback", account, region, endpoint, fallbackUsed, profileARN, readErr, http.StatusBadGateway)
				continue
			}
			return nil, infraerrors.ServiceUnavailable("KIRO_RUNTIME_PROBE_FAILED", "failed to read kiro runtime probe response").WithCause(readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			authErr := fmt.Errorf("kiro runtime authentication failed: %s", extractKiroErrorMessage(body))
			s.logKiroRuntimeAttempt("kiro_runtime_probe_auth_failed", account, region, endpoint, fallbackUsed, profileARN, authErr, resp.StatusCode)
			return nil, infraerrors.BadRequest("KIRO_RUNTIME_AUTH_FAILED", "kiro runtime authentication failed").WithCause(authErr)
		}
		if shouldKiroFallbackEndpoint(idx, resp.StatusCode, nil) {
			lastErr = fmt.Errorf("kiro runtime probe endpoint %s returned %d", endpoint.Name, resp.StatusCode)
			s.logKiroRuntimeAttempt("kiro_runtime_probe_endpoint_fallback", account, region, endpoint, fallbackUsed, profileARN, lastErr, resp.StatusCode)
			continue
		}
		if resp.StatusCode >= http.StatusInternalServerError {
			lastErr = fmt.Errorf("kiro runtime probe endpoint %s returned %d", endpoint.Name, resp.StatusCode)
			s.logKiroRuntimeAttempt("kiro_runtime_probe_upstream_error", account, region, endpoint, fallbackUsed, profileARN, lastErr, resp.StatusCode)
			return nil, infraerrors.ServiceUnavailable("KIRO_RUNTIME_PROBE_FAILED", "kiro runtime endpoint verification failed").WithCause(lastErr)
		}

		resolved := ResolveUpstreamInfo(endpoint.URL, PlatformKiro, kiroRuntimeProbeSource)
		resolved.Region = strings.TrimSpace(region)
		resolved.ProbedAt = time.Now().UTC()
		s.persistResolvedKiroUpstream(ctx, account, resolved)
		s.logKiroRuntimeAttempt("kiro_runtime_probe_succeeded", account, region, endpoint, fallbackUsed, profileARN, nil, resp.StatusCode)
		return &KiroRuntimeProbeResult{
			Region:           region,
			Endpoint:         endpoint,
			ProfileARN:       profileARN,
			FallbackUsed:     fallbackUsed,
			ResolvedUpstream: resolved,
		}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("kiro runtime probe failed")
	}
	return nil, infraerrors.ServiceUnavailable("KIRO_RUNTIME_PROBE_FAILED", "failed to verify kiro runtime endpoint").WithCause(lastErr)
}

func (s *KiroRuntimeService) ExecuteClaude(ctx context.Context, account *Account, input KiroRuntimeExecuteInput) (*KiroRuntimeExecuteResult, error) {
	if account == nil || account.Platform != PlatformKiro {
		return nil, fmt.Errorf("kiro runtime requires a kiro account")
	}
	if NormalizeKiroAccountCredentials(account) {
		s.persistKiroAccount(ctx, account)
	}
	token, err := s.resolveKiroAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	profileARN := s.ensureKiroProfileARN(ctx, account, token)
	region := ResolveKiroAPIRegion(account)
	payload, err := buildKiroClaudePayload(input.Body, input.ModelID, effectiveKiroProfileARN(account, profileARN), kiroPrimaryOrigin, input.RequestHeaders)
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)

	var lastErr error
	for idx, endpoint := range buildKiroEndpointConfigs(region) {
		fallbackUsed := idx > 0
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		s.applyKiroRequestHeaders(req, account, token, endpoint)

		resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
		if err != nil {
			lastErr = err
			s.logKiroRuntimeAttempt("kiro_runtime_request_failed", account, region, endpoint, fallbackUsed, profileARN, err, 0)
			if shouldKiroFallbackEndpoint(idx, 0, err) {
				s.logKiroRuntimeAttempt("kiro_runtime_endpoint_fallback", account, region, endpoint, fallbackUsed, profileARN, err, 0)
				continue
			}
			return nil, err
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			errorResp := s.translateKiroErrorResponse(resp)
			s.logKiroRuntimeAttempt("kiro_runtime_upstream_error", account, region, endpoint, fallbackUsed, profileARN, nil, errorResp.StatusCode)
			if shouldKiroFallbackEndpoint(idx, resp.StatusCode, nil) {
				lastErr = fmt.Errorf("kiro endpoint %s returned %d", endpoint.Name, resp.StatusCode)
				s.logKiroRuntimeAttempt("kiro_runtime_endpoint_fallback", account, region, endpoint, fallbackUsed, profileARN, lastErr, resp.StatusCode)
				continue
			}
			resolved := ResolveUpstreamInfo(endpoint.URL, PlatformKiro, kiroRuntimeRequestSource)
			resolved.Region = strings.TrimSpace(region)
			resolved.ProbedAt = time.Now().UTC()
			s.persistResolvedKiroUpstream(ctx, account, resolved)
			return &KiroRuntimeExecuteResult{
				Response:         errorResp,
				Region:           region,
				Endpoint:         endpoint,
				ProfileARN:       profileARN,
				FallbackUsed:     fallbackUsed,
				ResolvedUpstream: resolved,
			}, nil
		}

		translated, err := s.translateKiroSuccessResponse(resp, input.ModelID, input.Stream)
		if err != nil {
			lastErr = err
			s.logKiroRuntimeAttempt("kiro_runtime_translate_failed", account, region, endpoint, fallbackUsed, profileARN, err, http.StatusBadGateway)
			if shouldKiroFallbackEndpoint(idx, http.StatusBadGateway, err) {
				s.logKiroRuntimeAttempt("kiro_runtime_endpoint_fallback", account, region, endpoint, fallbackUsed, profileARN, err, http.StatusBadGateway)
				continue
			}
			return nil, err
		}
		resolved := ResolveUpstreamInfo(endpoint.URL, PlatformKiro, kiroRuntimeRequestSource)
		resolved.Region = strings.TrimSpace(region)
		resolved.ProbedAt = time.Now().UTC()
		s.persistResolvedKiroUpstream(ctx, account, resolved)
		s.logKiroRuntimeAttempt("kiro_runtime_request_succeeded", account, region, endpoint, fallbackUsed, profileARN, nil, translated.StatusCode)
		return &KiroRuntimeExecuteResult{
			Response:         translated,
			Region:           region,
			Endpoint:         endpoint,
			ProfileARN:       profileARN,
			FallbackUsed:     fallbackUsed,
			ResolvedUpstream: resolved,
		}, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("kiro runtime request failed")
	}
	return nil, lastErr
}

func (s *KiroRuntimeService) resolveKiroAccessToken(ctx context.Context, account *Account) (string, error) {
	if account == nil {
		return "", fmt.Errorf("account is nil")
	}
	if account.Type == AccountTypeOAuth && s.claudeTokenProvider != nil {
		return s.claudeTokenProvider.GetAccessToken(ctx, account)
	}
	if token := strings.TrimSpace(account.GetCredential("access_token")); token != "" {
		return token, nil
	}
	return "", fmt.Errorf("kiro access_token is missing")
}

func (s *KiroRuntimeService) ensureKiroProfileARN(ctx context.Context, account *Account, accessToken string) string {
	if account == nil {
		return ""
	}
	if profileARN := strings.TrimSpace(account.GetCredential("profile_arn")); profileARN != "" {
		return profileARN
	}
	profileARN := s.tryListAvailableProfiles(ctx, account, accessToken)
	if profileARN == "" {
		return ""
	}
	if account.Credentials == nil {
		account.Credentials = map[string]any{}
	}
	account.Credentials["profile_arn"] = profileARN
	s.persistKiroAccount(ctx, account)
	return profileARN
}

func (s *KiroRuntimeService) tryListAvailableProfiles(ctx context.Context, account *Account, accessToken string) string {
	if s == nil || s.httpUpstream == nil || account == nil || strings.TrimSpace(accessToken) == "" {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildKiroListProfilesURL(ResolveKiroAPIRegion(account)), bytes.NewReader([]byte("{}")))
	if err != nil {
		return ""
	}
	s.applyKiroRequestHeaders(req, account, accessToken, KiroEndpointConfig{
		URL:    req.URL.String(),
		Origin: kiroPrimaryOrigin,
		Name:   "ListAvailableProfiles",
	})

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil || resp == nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		Profiles []struct {
			ARN string `json:"arn"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return ""
	}
	if len(parsed.Profiles) == 0 {
		return ""
	}
	return strings.TrimSpace(parsed.Profiles[0].ARN)
}

func (s *KiroRuntimeService) applyKiroRequestHeaders(req *http.Request, account *Account, token string, endpoint KiroEndpointConfig) {
	req.Header.Set("Content-Type", kiroContentType)
	req.Header.Set("Accept", kiroAcceptAll)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("User-Agent", buildKiroUserAgent(account))
	req.Header.Set("X-Amz-User-Agent", buildKiroAmzUserAgent(account))
	req.Header.Set("x-amzn-kiro-agent-mode", kiroAgentMode)
	req.Header.Set("x-amzn-codewhisperer-optout", "true")
	req.Header.Set("Amz-Sdk-Request", "attempt=1; max=3")
	req.Header.Set("Amz-Sdk-Invocation-Id", uuid.NewString())
	if strings.TrimSpace(endpoint.AmzTarget) != "" {
		req.Header.Set("X-Amz-Target", endpoint.AmzTarget)
	}
}

func (s *KiroRuntimeService) translateKiroSuccessResponse(resp *http.Response, modelID string, stream bool) (*http.Response, error) {
	if stream {
		reader, writer := io.Pipe()
		headers := resp.Header.Clone()
		headers.Set("Content-Type", "text/event-stream")
		go func() {
			defer func() { _ = resp.Body.Close() }()
			defer func() { _ = writer.Close() }()
			streamKiroToClaude(resp.Body, writer, modelID)
		}()
		return &http.Response{
			StatusCode: resp.StatusCode,
			Header:     headers,
			Body:       reader,
			Request:    resp.Request,
		}, nil
	}

	defer func() { _ = resp.Body.Close() }()
	collected, err := collectKiroResponse(resp.Body)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Header:     resp.Header.Clone(),
			Body:       io.NopCloser(bytes.NewReader(normalizeKiroErrorResponse(http.StatusBadGateway, []byte(err.Error())))),
			Request:    resp.Request,
		}, nil
	}
	headers := resp.Header.Clone()
	headers.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: resp.StatusCode,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(buildClaudeResponseFromKiro(collected, modelID))),
		Request:    resp.Request,
	}, nil
}

func (s *KiroRuntimeService) translateKiroErrorResponse(resp *http.Response) *http.Response {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
	headers := resp.Header.Clone()
	headers.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: resp.StatusCode,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(normalizeKiroErrorResponse(resp.StatusCode, body))),
		Request:    resp.Request,
	}
}

func (s *KiroRuntimeService) persistKiroAccount(ctx context.Context, account *Account) {
	if s == nil || s.accountRepo == nil || account == nil {
		return
	}
	if err := s.accountRepo.Update(ctx, account); err != nil {
		slog.Warn("kiro_runtime_account_persist_failed",
			"account_id", account.ID,
			"error", err,
		)
	}
}

func (s *KiroRuntimeService) persistResolvedKiroUpstream(ctx context.Context, account *Account, info ResolvedUpstreamInfo) {
	if account == nil {
		return
	}
	merged := MergeUpstreamExtra(account.Extra, info)
	if len(merged) == 0 {
		return
	}
	account.Extra = merged
	if s == nil || s.accountRepo == nil || account.ID <= 0 {
		return
	}
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{
		"upstream_url":          merged["upstream_url"],
		"upstream_host":         merged["upstream_host"],
		"upstream_service":      merged["upstream_service"],
		"upstream_probe_source": merged["upstream_probe_source"],
		"upstream_probed_at":    merged["upstream_probed_at"],
		"upstream_region":       merged["upstream_region"],
	}); err != nil {
		slog.Warn("kiro_runtime_upstream_metadata_persist_failed",
			"account_id", account.ID,
			"error", err,
		)
	}
}

func (s *KiroRuntimeService) logKiroRuntimeAttempt(
	event string,
	account *Account,
	region string,
	endpoint KiroEndpointConfig,
	fallbackUsed bool,
	profileARN string,
	err error,
	statusCode int,
) {
	attrs := []any{
		"account_id", accountIDOrZero(account),
		"resolved_region", strings.TrimSpace(region),
		"endpoint", strings.TrimSpace(endpoint.URL),
		"endpoint_name", strings.TrimSpace(endpoint.Name),
		"fallback", fallbackUsed,
		"profile_arn_present", strings.TrimSpace(profileARN) != "",
	}
	if statusCode > 0 {
		attrs = append(attrs, "status", statusCode)
	}
	if err != nil {
		attrs = append(attrs, "error", err)
		slog.Warn(event, attrs...)
		return
	}
	slog.Info(event, attrs...)
}
