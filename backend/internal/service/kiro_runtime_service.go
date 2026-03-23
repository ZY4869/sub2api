package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type KiroRuntimeService struct {
	accountRepo         AccountRepository
	httpUpstream        HTTPUpstream
	claudeTokenProvider *ClaudeTokenProvider
}

func NewKiroRuntimeService(accountRepo AccountRepository, httpUpstream HTTPUpstream, claudeTokenProvider *ClaudeTokenProvider) *KiroRuntimeService {
	return &KiroRuntimeService{
		accountRepo:         accountRepo,
		httpUpstream:        httpUpstream,
		claudeTokenProvider: claudeTokenProvider,
	}
}

func (s *KiroRuntimeService) ExecuteClaude(ctx context.Context, account *Account, input KiroRuntimeExecuteInput) (*KiroRuntimeExecuteResult, error) {
	if account == nil || account.Platform != PlatformKiro {
		return nil, fmt.Errorf("kiro runtime requires a kiro account")
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

	var lastErr error
	for idx, endpoint := range buildKiroEndpointConfigs(region) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		s.applyKiroRequestHeaders(req, account, token, endpoint)

		resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
		if err != nil {
			lastErr = err
			if shouldKiroFallbackEndpoint(idx, 0, err) {
				continue
			}
			return nil, err
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			errorResp := s.translateKiroErrorResponse(resp)
			if shouldKiroFallbackEndpoint(idx, resp.StatusCode, nil) {
				lastErr = fmt.Errorf("kiro endpoint %s returned %d", endpoint.Name, resp.StatusCode)
				continue
			}
			return &KiroRuntimeExecuteResult{
				Response:   errorResp,
				Region:     region,
				Endpoint:   endpoint,
				ProfileARN: profileARN,
			}, nil
		}

		translated, err := s.translateKiroSuccessResponse(resp, input.ModelID, input.Stream)
		if err != nil {
			lastErr = err
			if shouldKiroFallbackEndpoint(idx, http.StatusBadGateway, err) {
				continue
			}
			return nil, err
		}
		return &KiroRuntimeExecuteResult{
			Response:   translated,
			Region:     region,
			Endpoint:   endpoint,
			ProfileARN: profileARN,
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
	if s.accountRepo != nil {
		_ = s.accountRepo.Update(ctx, account)
	}
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
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
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
