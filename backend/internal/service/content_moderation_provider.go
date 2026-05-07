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

	if strings.TrimSpace(settings.APIKey) == "" || strings.TrimSpace(settings.Model) == "" {
		return contentModerationProviderResult{ErrorReason: "moderation_not_configured"}
	}

	if strings.EqualFold(strings.TrimSpace(settings.Provider), "openai") || provider == "" {
		return callOpenAIContentModeration(ctx, settings, input, content)
	}

	return contentModerationProviderResult{ErrorReason: "moderation_provider_unsupported"}
}

func callOpenAIContentModeration(
	ctx context.Context,
	settings *ContentModerationSettings,
	input *ContentModerationRecordInput,
	content string,
) contentModerationProviderResult {
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
		return contentModerationProviderResult{ErrorReason: "moderation_request_encode_failed"}
	}

	timeoutMs := settings.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = contentModerationDefaultTimeoutMs
	}
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return contentModerationProviderResult{ErrorReason: "moderation_request_build_failed"}
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(settings.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(input.RequestID) != "" {
		req.Header.Set("X-Request-ID", strings.TrimSpace(input.RequestID))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			return contentModerationProviderResult{ErrorReason: "moderation_timeout"}
		}
		return contentModerationProviderResult{ErrorReason: "moderation_upstream_failed"}
	}
	defer func() { _ = resp.Body.Close() }()

	var decoded contentModerationOpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return contentModerationProviderResult{ErrorReason: "moderation_response_decode_failed"}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if decoded.Error != nil {
			if msg := strings.TrimSpace(decoded.Error.Message); msg != "" {
				return contentModerationProviderResult{ErrorReason: truncateModerationErrorReason(msg)}
			}
		}
		return contentModerationProviderResult{ErrorReason: fmt.Sprintf("moderation_http_%d", resp.StatusCode)}
	}

	for _, result := range decoded.Results {
		if result.Flagged {
			return contentModerationProviderResult{Hit: true}
		}
	}
	return contentModerationProviderResult{}
}

func truncateModerationErrorReason(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) <= 120 {
		return value
	}
	return value[:120]
}
