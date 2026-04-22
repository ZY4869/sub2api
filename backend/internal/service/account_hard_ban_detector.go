package service

import (
	"encoding/json"
	"strings"
)

type HardBanMatch struct {
	ReasonCode    string
	ReasonMessage string
}

type hardBanErrorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
	Error   struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status int `json:"status"`
}

func DetectHardBannedAccount(statusCode int, responseBody []byte) *HardBanMatch {
	bodyText := strings.TrimSpace(string(responseBody))
	if bodyText == "" {
		return nil
	}
	if statusCode >= 500 {
		return nil
	}

	candidate := strings.ToLower(bodyText)
	envelope := hardBanErrorEnvelope{}
	if err := json.Unmarshal(responseBody, &envelope); err == nil {
		code, message := extractHardBanCodeAndMessage(envelope)
		switch code {
		case "account_deactivated":
			return &HardBanMatch{ReasonCode: "account_deactivated", ReasonMessage: firstNonEmptyHardBanString(message, bodyText)}
		case "organization_disabled":
			return &HardBanMatch{ReasonCode: "organization_disabled", ReasonMessage: firstNonEmptyHardBanString(message, bodyText)}
		case "deactivated_workspace":
			if statusCode == 402 {
				return &HardBanMatch{ReasonCode: "workspace_deactivated", ReasonMessage: firstNonEmptyHardBanString(message, bodyText)}
			}
		}
		if looksLikeDeactivatedMessage(strings.ToLower(message)) {
			return &HardBanMatch{ReasonCode: "account_deactivated", ReasonMessage: firstNonEmptyHardBanString(message, bodyText)}
		}
	}

	if statusCode == 402 && looksLikeWorkspaceDeactivatedCode(candidate) {
		return &HardBanMatch{ReasonCode: "workspace_deactivated", ReasonMessage: bodyText}
	}

	switch {
	case strings.Contains(candidate, "account_deactivated"):
		return &HardBanMatch{ReasonCode: "account_deactivated", ReasonMessage: bodyText}
	case strings.Contains(candidate, "organization has been disabled"),
		strings.Contains(candidate, "organization disabled"):
		return &HardBanMatch{ReasonCode: "organization_disabled", ReasonMessage: bodyText}
	case looksLikeDeactivatedMessage(candidate):
		return &HardBanMatch{ReasonCode: "account_deactivated", ReasonMessage: bodyText}
	}

	if (statusCode == 401 || statusCode == 403) &&
		strings.Contains(candidate, "help.openai.com") &&
		(strings.Contains(candidate, "deactivated") || strings.Contains(candidate, "suspended") || strings.Contains(candidate, "banned")) {
		return &HardBanMatch{ReasonCode: "account_hard_banned", ReasonMessage: bodyText}
	}

	return nil
}

func looksLikeDeactivatedMessage(candidate string) bool {
	if candidate == "" {
		return false
	}

	return (strings.Contains(candidate, "has been deactivated") ||
		strings.Contains(candidate, "account has been deactivated") ||
		strings.Contains(candidate, "account is deactivated") ||
		strings.Contains(candidate, "organization has been disabled") ||
		strings.Contains(candidate, "organization disabled") ||
		strings.Contains(candidate, "account suspended") ||
		strings.Contains(candidate, "has been suspended") ||
		strings.Contains(candidate, "account banned") ||
		strings.Contains(candidate, "has been banned")) &&
		(strings.Contains(candidate, "openai") ||
			strings.Contains(candidate, "help.openai.com") ||
			strings.Contains(candidate, "\"status\": 401") ||
			strings.Contains(candidate, "status\":401"))
}

func firstNonEmptyHardBanString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func extractHardBanCodeAndMessage(envelope hardBanErrorEnvelope) (string, string) {
	code := strings.ToLower(strings.TrimSpace(envelope.Error.Code))
	if code == "" {
		code = strings.ToLower(strings.TrimSpace(envelope.Code))
	}

	message := strings.TrimSpace(envelope.Error.Message)
	if message == "" {
		message = strings.TrimSpace(envelope.Error.Detail)
	}
	if message == "" {
		message = strings.TrimSpace(envelope.Message)
	}
	if message == "" {
		message = strings.TrimSpace(envelope.Detail)
	}

	return code, message
}

func looksLikeWorkspaceDeactivatedCode(candidate string) bool {
	if candidate == "" {
		return false
	}

	return candidate == "deactivated_workspace" ||
		strings.Contains(candidate, "\"code\":\"deactivated_workspace\"") ||
		strings.Contains(candidate, "\"code\": \"deactivated_workspace\"")
}
