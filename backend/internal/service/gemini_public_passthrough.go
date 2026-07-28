package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/tidwall/gjson"
)

type GeminiPublicPassthroughInput struct {
	GoogleBatchForwardInput
	RequestedModel        string
	ResourceKind          string
	UpstreamPath          string
	ForcedPlatform        string
	RequiresAPIKeyAccount bool
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
		return &GeminiPublicPassthroughOutput{
			Response:      &UpstreamHTTPResult{StatusCode: resp.StatusCode, Headers: filteredHeaders, Body: body},
			Account:       account,
			ForwardResult: buildGeminiPassthroughForwardResult(input, requestedModel, filteredHeaders, body, time.Since(startedAt), false),
		}, infraerrors.New(resp.StatusCode, "GEMINI_PASSTHROUGH_UPSTREAM_ERROR", message)
	}

	if err := s.persistGeminiPassthroughBinding(ctx, input, account, binding, resp.StatusCode, body); err != nil {
		return nil, err
	}
	if isOpenAICompatUsageOnlyNonStreamResponse(input, body) {
		return &GeminiPublicPassthroughOutput{
			Response:      &UpstreamHTTPResult{StatusCode: resp.StatusCode, Headers: filteredHeaders, Body: body},
			Account:       account,
			ForwardResult: buildGeminiPassthroughForwardResult(input, requestedModel, filteredHeaders, body, time.Since(startedAt), false),
		}, infraerrors.ServiceUnavailable("GEMINI_OPENAI_COMPAT_USAGE_ONLY_RESPONSE", "upstream returned usage metadata without a completion payload")
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
		if err == nil && geminiPassthroughEligibleAccount(account, input) {
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
		if err == nil && geminiPassthroughEligibleAccount(account, input) {
			return account, binding, nil
		}
	}

	selectionCtx := WithGeminiPublicProtocol(ctx, UpstreamProviderAIStudio)
	if forced := strings.TrimSpace(input.ForcedPlatform); forced != "" {
		selectionCtx = context.WithValue(selectionCtx, ctxkey.ForcePlatform, strings.ToLower(forced))
	}
	if strings.TrimSpace(input.ForcedPlatform) != "" {
		account, err := s.selectGeminiPassthroughAccount(selectionCtx, input, requestedModel)
		if err != nil {
			return nil, nil, err
		}
		if account != nil {
			return account, binding, nil
		}
		return nil, nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_NO_ACCOUNT", "no available Gemini accounts")
	}
	if requestedModel != "" {
		account, err := s.SelectAccountForModelWithExclusions(selectionCtx, input.GroupID, "", requestedModel, nil)
		if err == nil && geminiPassthroughEligibleAccount(account, input) {
			return account, binding, nil
		}
	}

	platform := strings.TrimSpace(input.ForcedPlatform)
	if platform == "" {
		platform = PlatformGemini
	}
	account, err := s.SelectAccountForAIStudioEndpoints(selectionCtx, input.GroupID, platform)
	if err != nil {
		return nil, nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_NO_ACCOUNT", "no available Gemini accounts")
	}
	if !geminiPassthroughEligibleAccount(account, input) {
		return nil, nil, infraerrors.ServiceUnavailable("GEMINI_PASSTHROUGH_NO_ACCOUNT", "no available Gemini accounts")
	}
	return account, binding, nil
}

func (s *GeminiMessagesCompatService) selectGeminiPassthroughAccount(ctx context.Context, input GeminiPublicPassthroughInput, requestedModel string) (*Account, error) {
	platform := strings.TrimSpace(strings.ToLower(input.ForcedPlatform))
	if platform == "" {
		platform = PlatformGemini
	}
	accounts, err := s.listSchedulableAccountsOnce(ctx, input.GroupID, platform, true)
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}
	if len(accounts) == 0 && input.GroupID != nil {
		accounts, err = s.listSchedulableAccountsOnce(ctx, nil, platform, true)
		if err != nil {
			return nil, fmt.Errorf("query accounts failed: %w", err)
		}
	}
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if geminiPassthroughEligibleAccount(&account, input) {
			filtered = append(filtered, account)
		}
	}
	return s.selectBestGeminiAccount(ctx, filtered, requestedModel, nil, platform, false), nil
}

func geminiPassthroughEligibleAccount(account *Account, input GeminiPublicPassthroughInput) bool {
	if account == nil {
		return false
	}
	if input.RequiresAPIKeyAccount && account.Type != AccountTypeAPIKey {
		return false
	}
	if account.Type == AccountTypeAPIKey && strings.TrimSpace(account.GetCredential("api_key")) == "" {
		return false
	}
	forced := strings.TrimSpace(strings.ToLower(input.ForcedPlatform))
	switch forced {
	case "":
		return EffectiveProtocol(account) == PlatformGemini && !account.IsGeminiVertexSource()
	case PlatformAntigravity:
		if strings.TrimSpace(account.GetCredential("base_url")) == "" {
			return false
		}
		return EffectiveProtocol(account) == PlatformAntigravity
	default:
		return EffectiveProtocol(account) == forced
	}
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
	if strings.EqualFold(strings.TrimSpace(input.ForcedPlatform), PlatformAntigravity) {
		upstreamPath = strings.TrimPrefix(upstreamPath, "/antigravity")
		if upstreamPath == "" {
			upstreamPath = "/"
		}
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

func isOpenAICompatUsageOnlyNonStreamResponse(input GeminiPublicPassthroughInput, body []byte) bool {
	if len(body) == 0 || gjson.GetBytes(input.Body, "stream").Bool() {
		return false
	}
	path := strings.ToLower(strings.TrimSpace(firstNonEmptyString(input.UpstreamPath, input.Path)))
	if !strings.Contains(path, "/openai/") {
		return false
	}
	if !gjson.GetBytes(body, "usage").Exists() {
		return false
	}
	for _, payloadPath := range []string{
		"choices",
		"output",
		"data",
		"candidates",
		"response.candidates",
		"response.output",
		"response.choices",
		"response.data",
	} {
		if value := gjson.GetBytes(body, payloadPath); value.Exists() && value.Raw != "[]" && value.Raw != "{}" && value.Raw != "null" {
			return false
		}
	}
	return true
}
