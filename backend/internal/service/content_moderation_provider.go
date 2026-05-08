package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type contentModerationProviderResult struct {
	Hit         bool
	ErrorReason string
	StatusCode  int
	KeyHash     string
}

type contentModerationOpenAIRequest struct {
	Model string `json:"model,omitempty"`
	Input string `json:"input"`
}

type contentModerationOpenAIResponse struct {
	Results []struct {
		Flagged bool `json:"flagged"`
	} `json:"results"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func evaluateContentModeration(
	ctx context.Context,
	settings *ContentModerationSettings,
	input *ContentModerationRecordInput,
	content string,
) contentModerationProviderResult {
	if settings == nil {
		return contentModerationProviderResult{ErrorReason: "moderation_settings_missing"}
	}

	provider := NormalizeOAuthProvider(settings.Provider)
	if provider == "" && !strings.EqualFold(strings.TrimSpace(settings.Provider), "openai") {
		return contentModerationProviderResult{ErrorReason: "moderation_provider_unsupported"}
	}

	if strings.TrimSpace(settings.Model) == "" {
		return contentModerationProviderResult{ErrorReason: "moderation_not_configured"}
	}
	key, ok := SelectContentModerationAPIKey(settings.APIKeys, time.Now().UTC())
	if !ok && strings.TrimSpace(settings.APIKey) != "" {
		key = ContentModerationAPIKey{
			Key:  strings.TrimSpace(settings.APIKey),
			Hash: ContentModerationAPIKeyHash(settings.APIKey),
		}
		_, frozen := getContentModerationFreeze(key.Hash, time.Now().UTC())
		ok = !frozen
	}
	if !ok || strings.TrimSpace(key.Key) == "" {
		return contentModerationProviderResult{ErrorReason: "moderation_not_configured"}
	}

	if strings.EqualFold(strings.TrimSpace(settings.Provider), "openai") || provider == "" {
		result := callOpenAIContentModeration(ctx, settings, input, content, key)
		if result.ErrorReason == "" {
			ClearContentModerationKeyFreeze(result.KeyHash)
		}
		return result
	}

	return contentModerationProviderResult{ErrorReason: "moderation_provider_unsupported"}
}

func callOpenAIContentModeration(
	ctx context.Context,
	settings *ContentModerationSettings,
	input *ContentModerationRecordInput,
	content string,
	apiKey ContentModerationAPIKey,
) contentModerationProviderResult {
	keyHash := strings.TrimSpace(apiKey.Hash)
	if keyHash == "" {
		keyHash = ContentModerationAPIKeyHash(apiKey.Key)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(settings.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	} else if !strings.HasSuffix(strings.ToLower(baseURL), "/v1") {
		baseURL += "/v1"
	}
	endpoint := baseURL + "/moderations"

	payloadBytes, err := json.Marshal(contentModerationOpenAIRequest{
		Model: strings.TrimSpace(settings.Model),
		Input: content,
	})
	if err != nil {
		return contentModerationProviderResult{ErrorReason: "moderation_request_encode_failed", KeyHash: keyHash}
	}

	timeoutMs := settings.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = contentModerationDefaultTimeoutMs
	}
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return contentModerationProviderResult{ErrorReason: "moderation_request_build_failed", KeyHash: keyHash}
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey.Key))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(input.RequestID) != "" {
		req.Header.Set("X-Request-ID", strings.TrimSpace(input.RequestID))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			result := contentModerationProviderResult{ErrorReason: "moderation_timeout", KeyHash: keyHash}
			RegisterContentModerationKeyFailure(keyHash, result.ErrorReason, 0, err, time.Now().UTC())
			return result
		}
		result := contentModerationProviderResult{ErrorReason: "moderation_upstream_failed", KeyHash: keyHash}
		RegisterContentModerationKeyFailure(keyHash, result.ErrorReason, 0, err, time.Now().UTC())
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	var decoded contentModerationOpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		result := contentModerationProviderResult{
			ErrorReason: "moderation_response_decode_failed",
			StatusCode:  resp.StatusCode,
			KeyHash:     keyHash,
		}
		RegisterContentModerationKeyFailure(keyHash, result.ErrorReason, resp.StatusCode, err, time.Now().UTC())
		return result
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		reason := fmt.Sprintf("moderation_http_%d", resp.StatusCode)
		if decoded.Error != nil {
			if msg := strings.TrimSpace(decoded.Error.Message); msg != "" {
				reason = truncateModerationErrorReason(msg)
			}
		}
		result := contentModerationProviderResult{ErrorReason: reason, StatusCode: resp.StatusCode, KeyHash: keyHash}
		RegisterContentModerationKeyFailure(keyHash, reason, resp.StatusCode, nil, time.Now().UTC())
		return result
	}

	for _, result := range decoded.Results {
		if result.Flagged {
			return contentModerationProviderResult{Hit: true, StatusCode: resp.StatusCode, KeyHash: keyHash}
		}
	}
	return contentModerationProviderResult{StatusCode: resp.StatusCode, KeyHash: keyHash}
}

func truncateModerationErrorReason(value string) string {
	value = redactContentModerationSecrets(value)
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) <= 120 {
		return value
	}
	return value[:120]
}
