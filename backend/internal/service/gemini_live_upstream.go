package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

const (
	geminiLiveWSPath            = "/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	geminiLiveWSConstrainedPath = "/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContentConstrained"
	GeminiLiveAuthTokensPath    = "/v1alpha/authTokens"
)

type GeminiLiveUpstream struct {
	URL      string
	Headers  http.Header
	ProxyURL string
}

func (s *GeminiLiveGatewayService) BuildGeminiLiveUpstream(ctx context.Context, account *Account, constrained bool, ephemeralToken string) (*GeminiLiveUpstream, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, infraerrors.ServiceUnavailable("GEMINI_LIVE_UNAVAILABLE", "gemini live service unavailable")
	}
	if account == nil {
		return nil, infraerrors.BadRequest("GEMINI_LIVE_ACCOUNT_REQUIRED", "gemini live account is required")
	}
	baseURL, err := s.validateUpstreamBaseURL(account.GetGeminiBaseURL(geminicli.AIStudioBaseURL))
	if err != nil {
		return nil, err
	}
	wsURL, err := buildGeminiLiveWebSocketURL(baseURL, constrained)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	switch {
	case strings.TrimSpace(ephemeralToken) != "":
		headers.Set("Authorization", "Token "+strings.TrimSpace(ephemeralToken))
	case account.Type == AccountTypeAPIKey:
		parsedURL, parseErr := url.Parse(wsURL)
		if parseErr != nil {
			return nil, infraerrors.ServiceUnavailable("GEMINI_LIVE_URL_INVALID", "invalid Gemini Live websocket URL").WithCause(parseErr)
		}
		query := parsedURL.Query()
		query.Set("key", strings.TrimSpace(account.GetCredential("api_key")))
		parsedURL.RawQuery = query.Encode()
		wsURL = parsedURL.String()
	case account.Type == AccountTypeOAuth:
		if s.tokenProvider == nil {
			return nil, infraerrors.ServiceUnavailable("GEMINI_LIVE_TOKEN_PROVIDER_MISSING", "gemini token provider not configured")
		}
		accessToken, tokenErr := s.tokenProvider.GetAccessToken(ctx, account)
		if tokenErr != nil {
			return nil, tokenErr
		}
		headers.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	default:
		return nil, infraerrors.BadRequest("GEMINI_LIVE_ACCOUNT_TYPE_UNSUPPORTED", "unsupported Gemini Live account type")
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return &GeminiLiveUpstream{
		URL:      wsURL,
		Headers:  headers,
		ProxyURL: proxyURL,
	}, nil
}

func buildGeminiLiveWebSocketURL(baseURL string, constrained bool) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse Gemini base url: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(parsedURL.Scheme)) {
	case "https":
		parsedURL.Scheme = "wss"
	case "http":
		parsedURL.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported Gemini Live URL scheme: %s", parsedURL.Scheme)
	}
	methodPath := geminiLiveWSPath
	if constrained {
		methodPath = geminiLiveWSConstrainedPath
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + methodPath
	return parsedURL.String(), nil
}
