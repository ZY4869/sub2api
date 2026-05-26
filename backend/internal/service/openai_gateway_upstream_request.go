package service

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) buildUpstreamRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token string, isStream bool, promptCacheKey string, isCodexCLI bool) (*http.Request, error) {
	targetURL, err := resolveOpenAIResponsesTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		return nil, err
	}
	targetURL = appendOpenAIResponsesRequestPathSuffix(targetURL, openAIResponsesRequestPathSuffix(c))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("authorization", "Bearer "+token)
	applyOpenRouterAttributionRequestHeaders(account, req.Header)
	if isChatGPTOpenAIOAuthAccount(account) {
		req.Host = "chatgpt.com"
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	for key, values := range c.Request.Header {
		lowerKey := strings.ToLower(key)
		if !openaiAllowedHeaders[lowerKey] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if isChatGPTOpenAIOAuthAccount(account) {
		req.Header.Del("conversation_id")
		req.Header.Del("session_id")
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("originator", resolveOpenAIUpstreamOriginator(c, isCodexCLI))

		apiKeyID := getAPIKeyIDFromContext(c)
		if isOpenAIResponsesCompactPath(c) {
			req.Header.Set("accept", "application/json")
			if req.Header.Get("version") == "" {
				req.Header.Set("version", codexCLIVersion)
			}
			req.Header.Set("session_id", isolateOpenAISessionID(apiKeyID, resolveOpenAICompactSessionID(c)))
		} else {
			req.Header.Set("accept", "text/event-stream")
		}
		if promptCacheKey != "" {
			isolated := isolateOpenAISessionID(apiKeyID, promptCacheKey)
			req.Header.Set("conversation_id", isolated)
			req.Header.Set("session_id", isolated)
		}
	}

	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI && isChatGPTOpenAIOAuthAccount(account) {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}
	s.applyCodexOAuthUserAgentPolicy(ctx, req.Header, account)
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	return req, nil
}

func (s *OpenAIGatewayService) validateUpstreamBaseURL(raw string) (string, error) {
	if s == nil {
		return validateUpstreamBaseURLWithConfig(nil, raw)
	}
	return validateUpstreamBaseURLWithConfig(s.cfg, raw)
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
