package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/tidwall/gjson"
)

type GeminiPublicPassthroughInput struct {
	GoogleBatchForwardInput
	RequestedModel string
	ResourceKind   string
	UpstreamPath   string
}

type GeminiPublicPassthroughOutput struct {
	Response      GoogleBatchUpstreamResult
	Account       *Account
	ForwardResult *ForwardResult
}

func (s *GeminiMessagesCompatService) forwardGeminiPassthrough(ctx context.Context, input GeminiPublicPassthroughInput) (*GeminiPublicPassthroughOutput, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_UNAVAILABLE", "gemini passthrough service unavailable")
	}

	requestedModel := strings.TrimSpace(firstNonEmptyString(input.RequestedModel, detectGeminiPassthroughRequestedModel(input.Path, input.Body)))
	account, binding, err := s.resolveGeminiPassthroughAccount(ctx, input, requestedModel)
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	req, proxyURL, _, err := s.buildGeminiPassthroughRequest(ctx, input, account)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_REQUEST_FAILED", "failed to request Gemini upstream").WithCause(err)
	}

	if shouldStreamGeminiPassthrough(resp, input) {
		filteredHeaders := responseheaders.FilterHeaders(resp.Header, s.responseHeaderFilter)
		forwardResult := buildGeminiPassthroughForwardResult(input, requestedModel, filteredHeaders, nil, time.Since(startedAt), true)
		return &GeminiPublicPassthroughOutput{
			Response: &UpstreamHTTPStreamResult{
				StatusCode:    resp.StatusCode,
				Headers:       filteredHeaders,
				Body:          resp.Body,
				ContentLength: resp.ContentLength,
			},
			Account:       account,
			ForwardResult: forwardResult,
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := readUpstreamResponseBodyLimited(resp.Body, googleBatchResponseReadLimit)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			return nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_RESPONSE_TOO_LARGE", "Gemini upstream response too large").WithCause(err)
		}
		return nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_READ_FAILED", "failed to read Gemini upstream response").WithCause(err)
	}
	filteredHeaders := responseheaders.FilterHeaders(resp.Header, s.responseHeaderFilter)

	if resp.StatusCode >= http.StatusBadRequest {
		s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
		message := sanitizeUpstreamErrorMessage(strings.TrimSpace(ExtractUpstreamErrorMessage(body)))
		if message == "" {
			message = "Gemini upstream request failed"
		}
		return nil, infraerrors.New(resp.StatusCode, "GEMINI_PASSTHROUGH_UPSTREAM_ERROR", message)
	}

	if err := s.persistGeminiPassthroughBinding(ctx, input, account, binding, resp.StatusCode, body); err != nil {
		return nil, err
	}

	return &GeminiPublicPassthroughOutput{
		Response:      &UpstreamHTTPResult{StatusCode: resp.StatusCode, Headers: filteredHeaders, Body: body},
		Account:       account,
		ForwardResult: buildGeminiPassthroughForwardResult(input, requestedModel, filteredHeaders, body, time.Since(startedAt), false),
	}, nil
}

func (s *GeminiMessagesCompatService) resolveGeminiPassthroughAccount(ctx context.Context, input GeminiPublicPassthroughInput, requestedModel string) (*Account, *UpstreamResourceBinding, error) {
	if input.AccountID != nil && *input.AccountID > 0 {
		account, err := s.getSchedulableAccount(ctx, *input.AccountID)
		if err == nil && geminiPassthroughEligibleAccount(account) {
			return account, nil, nil
		}
	}

	var binding *UpstreamResourceBinding
	resourceName := extractGeminiPassthroughResourceName(input.ResourceKind, input.Path)
	if resourceName != "" && s.resourceBindingRepo != nil {
		binding, _ = s.resourceBindingRepo.Get(ctx, input.ResourceKind, resourceName)
	}
	if binding != nil {
		account, err := s.getSchedulableAccount(ctx, binding.AccountID)
		if err == nil && geminiPassthroughEligibleAccount(account) {
			return account, binding, nil
		}
	}

	selectionCtx := WithGeminiPublicProtocol(ctx, UpstreamProviderAIStudio)
	if requestedModel != "" {
		account, err := s.SelectAccountForModelWithExclusions(selectionCtx, input.GroupID, "", requestedModel, nil)
		if err == nil && geminiPassthroughEligibleAccount(account) {
			return account, binding, nil
		}
	}

	account, err := s.SelectAccountForAIStudioEndpoints(selectionCtx, input.GroupID)
	if err != nil {
		return nil, nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_NO_ACCOUNT", "no available Gemini accounts")
	}
	if !geminiPassthroughEligibleAccount(account) {
		return nil, nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_NO_ACCOUNT", "no available Gemini accounts")
	}
	return account, binding, nil
}

func geminiPassthroughEligibleAccount(account *Account) bool {
	return account != nil && EffectiveProtocol(account) == PlatformGemini && !account.IsGeminiVertexSource()
}

func (s *GeminiMessagesCompatService) buildGeminiPassthroughRequest(ctx context.Context, input GeminiPublicPassthroughInput, account *Account) (*http.Request, string, string, error) {
	baseURL, err := s.validateUpstreamBaseURL(account.GetGeminiBaseURL(geminicli.AIStudioBaseURL))
	if err != nil {
		return nil, "", "", err
	}
	upstreamPath := strings.TrimSpace(input.UpstreamPath)
	if upstreamPath == "" {
		upstreamPath = strings.TrimSpace(input.Path)
	}
	fullURL := strings.TrimRight(baseURL, "/") + upstreamPath
	if strings.TrimSpace(input.RawQuery) != "" {
		fullURL += "?" + strings.TrimPrefix(strings.TrimSpace(input.RawQuery), "?")
	}
	body, err := input.OpenRequestBody()
	if err != nil {
		return nil, "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(strings.TrimSpace(input.Method)), fullURL, body)
	if err != nil {
		if body != nil {
			_ = body.Close()
		}
		return nil, "", "", err
	}
	if input.ContentLength > 0 || (input.ContentLength == 0 && len(input.Body) == 0) {
		req.ContentLength = input.ContentLength
	}
	copyGoogleForwardHeaders(req.Header, input.Headers)
	if err := s.applyGoogleBatchAuth(ctx, req, account); err != nil {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, "", "", err
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return req, proxyURL, fullURL, nil
}

func shouldStreamGeminiPassthrough(resp *http.Response, input GeminiPublicPassthroughInput) bool {
	if resp == nil {
		return false
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type"))), "text/event-stream") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(input.Method), http.MethodGet) && strings.Contains(strings.ToLower(strings.TrimSpace(input.RawQuery)), "stream=true") {
		return true
	}
	return gjson.GetBytes(input.Body, "stream").Bool()
}
