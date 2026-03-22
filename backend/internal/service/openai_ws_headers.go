package service

import (
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

type OpenAIWSIngressHooks struct {
	BeforeTurn func(turn int) error
	AfterTurn  func(turn int, result *OpenAIForwardResult, turnErr error)
}

func normalizeOpenAIWSLogValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return openAIWSLogValueReplacer.Replace(trimmed)
}
func truncateOpenAIWSLogValue(value string, maxLen int) string {
	normalized := normalizeOpenAIWSLogValue(value)
	if normalized == "-" || maxLen <= 0 {
		return normalized
	}
	if len(normalized) <= maxLen {
		return normalized
	}
	return normalized[:maxLen] + "..."
}
func openAIWSHeaderValueForLog(headers http.Header, key string) string {
	if headers == nil {
		return "-"
	}
	return truncateOpenAIWSLogValue(headers.Get(key), openAIWSHeaderValueMaxLen)
}
func hasOpenAIWSHeader(headers http.Header, key string) bool {
	if headers == nil {
		return false
	}
	return strings.TrimSpace(headers.Get(key)) != ""
}

type openAIWSSessionHeaderResolution struct {
	SessionID          string
	ConversationID     string
	SessionSource      string
	ConversationSource string
}

func resolveOpenAIWSSessionHeaders(c *gin.Context, promptCacheKey string) openAIWSSessionHeaderResolution {
	resolution := openAIWSSessionHeaderResolution{SessionSource: "none", ConversationSource: "none"}
	if c != nil && c.Request != nil {
		if sessionID := strings.TrimSpace(c.Request.Header.Get("session_id")); sessionID != "" {
			resolution.SessionID = sessionID
			resolution.SessionSource = "header_session_id"
		}
		if conversationID := strings.TrimSpace(c.Request.Header.Get("conversation_id")); conversationID != "" {
			resolution.ConversationID = conversationID
			resolution.ConversationSource = "header_conversation_id"
			if resolution.SessionID == "" {
				resolution.SessionID = conversationID
				resolution.SessionSource = "header_conversation_id"
			}
		}
	}
	cacheKey := strings.TrimSpace(promptCacheKey)
	if cacheKey != "" {
		if resolution.SessionID == "" {
			resolution.SessionID = cacheKey
			resolution.SessionSource = "prompt_cache_key"
		}
	}
	return resolution
}
func (s *OpenAIGatewayService) buildOpenAIResponsesWSURL(account *Account) (string, error) {
	if account == nil {
		return "", errors.New("account is nil")
	}
	targetURL, err := resolveOpenAIResponsesTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return "", fmt.Errorf("invalid target url: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", fmt.Errorf("unsupported scheme for ws: %s", parsed.Scheme)
	}
	return parsed.String(), nil
}
func (s *OpenAIGatewayService) buildOpenAIWSHeaders(c *gin.Context, account *Account, token string, decision OpenAIWSProtocolDecision, isCodexCLI bool, turnState string, turnMetadata string, promptCacheKey string) (http.Header, openAIWSSessionHeaderResolution) {
	headers := make(http.Header)
	headers.Set("authorization", "Bearer "+token)
	sessionResolution := resolveOpenAIWSSessionHeaders(c, promptCacheKey)
	apiKeyID := int64(0)
	if isChatGPTOpenAIOAuthAccount(account) {
		apiKeyID = getAPIKeyIDFromContext(c)
	}
	if c != nil && c.Request != nil {
		if v := strings.TrimSpace(c.Request.Header.Get("accept-language")); v != "" {
			headers.Set("accept-language", v)
		}
	}
	if sessionResolution.SessionID != "" {
		if isChatGPTOpenAIOAuthAccount(account) {
			headers.Set("session_id", isolateOpenAISessionID(apiKeyID, sessionResolution.SessionID))
		} else {
			headers.Set("session_id", sessionResolution.SessionID)
		}
	}
	if sessionResolution.ConversationID != "" {
		if isChatGPTOpenAIOAuthAccount(account) {
			headers.Set("conversation_id", isolateOpenAISessionID(apiKeyID, sessionResolution.ConversationID))
		} else {
			headers.Set("conversation_id", sessionResolution.ConversationID)
		}
	}
	if state := strings.TrimSpace(turnState); state != "" {
		headers.Set(openAIWSTurnStateHeader, state)
	}
	if metadata := strings.TrimSpace(turnMetadata); metadata != "" {
		headers.Set(openAIWSTurnMetadataHeader, metadata)
	}
	if isCopilotOAuthAccount(account) {
		applyCopilotDefaultHeaders(headers, account)
	}
	if isChatGPTOpenAIOAuthAccount(account) {
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			headers.Set("chatgpt-account-id", chatgptAccountID)
		}
		headers.Set("originator", resolveOpenAIUpstreamOriginator(c, isCodexCLI))
	}
	if isChatGPTOpenAIOAuthAccount(account) {
		betaValue := openAIWSBetaV2Value
		if decision.Transport == OpenAIUpstreamTransportResponsesWebsocket {
			betaValue = openAIWSBetaV1Value
		}
		headers.Set("OpenAI-Beta", betaValue)
	}
	customUA := ""
	if account != nil {
		customUA = account.GetOpenAIUserAgent()
	}
	if strings.TrimSpace(customUA) != "" {
		headers.Set("user-agent", customUA)
	} else if c != nil {
		if ua := strings.TrimSpace(c.GetHeader("User-Agent")); ua != "" {
			headers.Set("user-agent", ua)
		}
	}
	if s != nil && s.cfg != nil && s.cfg.Gateway.ForceCodexCLI && isChatGPTOpenAIOAuthAccount(account) {
		headers.Set("user-agent", codexCLIUserAgent)
	}
	if isChatGPTOpenAIOAuthAccount(account) && !openai.IsCodexCLIRequest(headers.Get("user-agent")) {
		headers.Set("user-agent", codexCLIUserAgent)
	}
	return headers, sessionResolution
}
func (s *OpenAIGatewayService) buildOpenAIWSCreatePayload(reqBody map[string]any, account *Account) map[string]any {
	payload := make(map[string]any, len(reqBody)+1)
	for k, v := range reqBody {
		payload[k] = v
	}
	delete(payload, "background")
	if _, exists := payload["stream"]; !exists {
		payload["stream"] = true
	}
	payload["type"] = "response.create"
	if isChatGPTOpenAIOAuthAccount(account) && !s.isOpenAIWSStoreRecoveryAllowed(account) {
		payload["store"] = false
	}
	return payload
}
func setOpenAIWSTurnMetadata(payload map[string]any, turnMetadata string) {
	if len(payload) == 0 {
		return
	}
	metadata := strings.TrimSpace(turnMetadata)
	if metadata == "" {
		return
	}
	switch existing := payload["client_metadata"].(type) {
	case map[string]any:
		existing[openAIWSTurnMetadataHeader] = metadata
		payload["client_metadata"] = existing
	case map[string]string:
		next := make(map[string]any, len(existing)+1)
		for k, v := range existing {
			next[k] = v
		}
		next[openAIWSTurnMetadataHeader] = metadata
		payload["client_metadata"] = next
	default:
		payload["client_metadata"] = map[string]any{openAIWSTurnMetadataHeader: metadata}
	}
}
