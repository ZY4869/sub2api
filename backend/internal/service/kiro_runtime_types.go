package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	kiroDefaultAPIRegion          = "us-east-1"
	kiroPrimaryOrigin             = "AI_EDITOR"
	kiroGenerateAssistantPath     = "/generateAssistantResponse"
	kiroListAvailableProfilesPath = "/ListAvailableProfiles"
	kiroAgentMode                 = "vibe"
	kiroContentType               = "application/json"
	kiroAcceptAll                 = "*/*"
	kiroRuntimeSDKVersion         = "1.0.27"
	kiroRuntimeNodeVersion        = "20"
	kiroRuntimeOSVersion          = "10"
	kiroThinkingStartTag          = "<thinking>"
	kiroThinkingEndTag            = "</thinking>"
	kiroEventStreamMinFrameSize   = 16
	kiroEventStreamMaxMessageSize = 10 << 20
)

type KiroEndpointConfig struct {
	URL       string
	Origin    string
	AmzTarget string
	Name      string
}

type KiroRuntimeExecuteInput struct {
	Body           []byte
	ModelID        string
	Stream         bool
	RequestHeaders http.Header
}

type KiroRuntimeExecuteResult struct {
	Response         *http.Response
	Region           string
	Endpoint         KiroEndpointConfig
	ProfileARN       string
	FallbackUsed     bool
	ResolvedUpstream ResolvedUpstreamInfo
}

type kiroEventStreamMessage struct {
	EventType string
	Payload   []byte
}

type kiroToolUse struct {
	ID    string
	Name  string
	Input map[string]any
}

type kiroToolUseState struct {
	ID    string
	Name  string
	Parts strings.Builder
}

type kiroCollectedResponse struct {
	Content    string
	ToolUses   []kiroToolUse
	Usage      ClaudeUsage
	StopReason string
}

type kiroEmitState struct {
	nextIndex         int
	openTextIndex     int
	openThinkingIndex int
	inThinking        bool
	pendingContent    string
}

func ResolveKiroAPIRegion(account *Account) string {
	if account == nil {
		return kiroDefaultAPIRegion
	}
	if region := strings.TrimSpace(account.GetCredential("api_region")); region != "" {
		return region
	}
	if region := extractKiroRegionFromProfileARN(account.GetCredential("profile_arn")); region != "" {
		return region
	}
	return kiroDefaultAPIRegion
}

func buildKiroEndpointConfigs(region string) []KiroEndpointConfig {
	if strings.TrimSpace(region) == "" {
		region = kiroDefaultAPIRegion
	}
	return []KiroEndpointConfig{
		{
			URL:    fmt.Sprintf("https://q.%s.amazonaws.com%s", region, kiroGenerateAssistantPath),
			Origin: kiroPrimaryOrigin,
			Name:   "AmazonQ",
		},
		{
			URL:       fmt.Sprintf("https://codewhisperer.%s.amazonaws.com%s", region, kiroGenerateAssistantPath),
			Origin:    kiroPrimaryOrigin,
			AmzTarget: "AmazonCodeWhispererStreamingService.GenerateAssistantResponse",
			Name:      "CodeWhisperer",
		},
	}
}

func buildKiroListProfilesURL(region string) string {
	if strings.TrimSpace(region) == "" {
		region = kiroDefaultAPIRegion
	}
	return fmt.Sprintf("https://q.%s.amazonaws.com%s", region, kiroListAvailableProfilesPath)
}

func extractKiroRegionFromProfileARN(profileARN string) string {
	parts := strings.Split(strings.TrimSpace(profileARN), ":")
	if len(parts) < 4 {
		return ""
	}
	return strings.TrimSpace(parts[3])
}

func buildKiroFingerprintSuffix(account *Account) string {
	parts := []string{
		fmt.Sprintf("%d", accountIDOrZero(account)),
		strings.TrimSpace(accountCredentialOrEmpty(account, "client_id")),
		strings.TrimSpace(accountCredentialOrEmpty(account, "refresh_token")),
		strings.TrimSpace(accountCredentialOrEmpty(account, "profile_arn")),
		strings.TrimSpace(accountCredentialOrEmpty(account, "access_token")),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(sum[:])[:12]
}

func buildKiroUserAgent(account *Account) string {
	return fmt.Sprintf(
		"aws-sdk-js/%s ua/2.1 os/windows#%s lang/js md/nodejs#%s api/codewhispererruntime#%s m/N,E KiroIDE-sub2api-%s",
		kiroRuntimeSDKVersion,
		kiroRuntimeOSVersion,
		kiroRuntimeNodeVersion,
		kiroRuntimeSDKVersion,
		buildKiroFingerprintSuffix(account),
	)
}

func buildKiroAmzUserAgent(account *Account) string {
	return fmt.Sprintf("aws-sdk-js/%s KiroIDE-sub2api-%s", kiroRuntimeSDKVersion, buildKiroFingerprintSuffix(account))
}

func effectiveKiroProfileARN(account *Account, profileARN string) string {
	if account != nil {
		switch normalizeStoredKiroAuthMethod(account.GetCredential("auth_method")) {
		case "builder_id", "idc":
			return ""
		}
	}
	return strings.TrimSpace(profileARN)
}

func normalizeKiroStopReason(stopReason string, hasToolUse bool) string {
	normalized := strings.ToLower(strings.TrimSpace(stopReason))
	switch normalized {
	case "":
		if hasToolUse {
			return "tool_use"
		}
		return "end_turn"
	case "tool_use", "tooluse":
		return "tool_use"
	case "stop_sequence", "stopsequence":
		return "stop_sequence"
	case "max_tokens", "maxtokens":
		return "max_tokens"
	case "end_turn", "endturn":
		return "end_turn"
	case "soft_limit_reached":
		if hasToolUse {
			return "tool_use"
		}
		return "end_turn"
	default:
		return normalized
	}
}

func normalizeKiroErrorResponse(statusCode int, body []byte) []byte {
	message := extractKiroErrorMessage(body)
	if message == "" {
		switch statusCode {
		case http.StatusUnauthorized:
			message = "Kiro authentication failed"
		case http.StatusForbidden:
			message = "Kiro access forbidden"
		case http.StatusTooManyRequests:
			message = "Kiro rate limit exceeded"
		default:
			message = "Kiro upstream request failed"
		}
	}

	payload := map[string]any{
		"error": map[string]any{
			"type":    "kiro_upstream_error",
			"message": message,
		},
	}
	if code := extractKiroErrorCode(body); code != "" {
		payload["error"].(map[string]any)["code"] = code
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{"error":{"type":"kiro_upstream_error","message":"Kiro upstream request failed"}}`)
	}
	return data
}

func extractKiroErrorMessage(body []byte) string {
	if msg := strings.TrimSpace(gjson.GetBytes(body, "error.message").String()); msg != "" {
		return msg
	}
	if msg := strings.TrimSpace(gjson.GetBytes(body, "message").String()); msg != "" {
		return msg
	}
	return strings.TrimSpace(string(body))
}

func extractKiroErrorCode(body []byte) string {
	for _, path := range []string{"error.code", "error.type", "_type"} {
		if code := strings.TrimSpace(gjson.GetBytes(body, path).String()); code != "" {
			return code
		}
	}
	return ""
}

func shouldKiroFallbackEndpoint(index, statusCode int, err error) bool {
	if err != nil {
		return index == 0
	}
	if index != 0 {
		return false
	}
	switch statusCode {
	case http.StatusUnauthorized, http.StatusTooManyRequests:
		return false
	case http.StatusForbidden, http.StatusNotFound, http.StatusMethodNotAllowed,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func accountCredentialOrEmpty(account *Account, key string) string {
	if account == nil {
		return ""
	}
	return account.GetCredential(key)
}

func accountIDOrZero(account *Account) int64 {
	if account == nil {
		return 0
	}
	return account.ID
}
