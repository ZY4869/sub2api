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
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
	Status int `json:"status"`
}

func DetectHardBannedAccount(statusCode int, responseBody []byte) *HardBanMatch {
	bodyText := strings.TrimSpace(string(responseBody))
	if bodyText == "" {
		return nil
	}

	candidate := strings.ToLower(bodyText)
	envelope := hardBanErrorEnvelope{}
	if err := json.Unmarshal(responseBody, &envelope); err == nil {
		code := strings.ToLower(strings.TrimSpace(envelope.Error.Code))
		message := strings.TrimSpace(envelope.Error.Message)
		switch code {
		case "account_deactivated":
			return &HardBanMatch{ReasonCode: "account_deactivated", ReasonMessage: firstNonEmptyHardBanString(message, bodyText)}
		case "organization_disabled":
			return &HardBanMatch{ReasonCode: "organization_disabled", ReasonMessage: firstNonEmptyHardBanString(message, bodyText)}
		}
		if looksLikeDeactivatedMessage(strings.ToLower(message)) {
			return &HardBanMatch{ReasonCode: "account_deactivated", ReasonMessage: firstNonEmptyHardBanString(message, bodyText)}
		}
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

	if (statusCode == 401 || statusCode == 403 || statusCode == 503) &&
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
