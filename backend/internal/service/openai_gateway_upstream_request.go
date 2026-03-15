package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) buildUpstreamRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token string, isStream bool, promptCacheKey string, isCodexCLI bool) (*http.Request, error) {
	var targetURL string
	switch account.Type {
	case AccountTypeOAuth:
		targetURL = chatgptCodexURL
	case AccountTypeAPIKey:
		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			targetURL = openaiPlatformAPIURL
		} else {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return nil, err
			}
			targetURL = buildOpenAIResponsesURL(validatedURL)
		}
	default:
		targetURL = openaiPlatformAPIURL
	}
	targetURL = appendOpenAIResponsesRequestPathSuffix(targetURL, openAIResponsesRequestPathSuffix(c))
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+token)
	if account.Type == AccountTypeOAuth {
		req.Host = "chatgpt.com"
		chatgptAccountID := account.GetChatGPTAccountID()
		if chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}
	for key, values := range c.Request.Header {
		lowerKey := strings.ToLower(key)
		if openaiAllowedHeaders[lowerKey] {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}
	if account.Type == AccountTypeOAuth {
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("originator", resolveOpenAIUpstreamOriginator(c, isCodexCLI))
		if isOpenAIResponsesCompactPath(c) {
			req.Header.Set("accept", "application/json")
			if req.Header.Get("version") == "" {
				req.Header.Set("version", codexCLIVersion)
			}
			if req.Header.Get("session_id") == "" {
				req.Header.Set("session_id", resolveOpenAICompactSessionID(c))
			}
		} else {
			req.Header.Set("accept", "text/event-stream")
		}
		if promptCacheKey != "" {
			req.Header.Set("conversation_id", promptCacheKey)
			req.Header.Set("session_id", promptCacheKey)
		}
	}
	customUA := account.GetOpenAIUserAgent()
	if customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	return req, nil
}

func (s *OpenAIGatewayService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg != nil && !s.cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{AllowedHosts: s.cfg.Security.URLAllowlist.UpstreamHosts, RequireAllowlist: true, AllowPrivate: s.cfg.Security.URLAllowlist.AllowPrivateHosts})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

func buildOpenAIResponsesURL(base string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(normalized, "/responses") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/responses"
	}
	return normalized + "/v1/responses"
}

func openAIResponsesRequestPathSuffix(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	normalizedPath := strings.TrimRight(strings.TrimSpace(c.Request.URL.Path), "/")
	if normalizedPath == "" {
		return ""
	}
	idx := strings.LastIndex(normalizedPath, "/responses")
	if idx < 0 {
		return ""
	}
	suffix := normalizedPath[idx+len("/responses"):]
	if suffix == "" || suffix == "/" {
		return ""
	}
	if !strings.HasPrefix(suffix, "/") {
		return ""
	}
	return suffix
}

func appendOpenAIResponsesRequestPathSuffix(baseURL, suffix string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedSuffix := strings.TrimSpace(suffix)
	if trimmedBase == "" || trimmedSuffix == "" {
		return trimmedBase
	}
	return trimmedBase + trimmedSuffix
}
