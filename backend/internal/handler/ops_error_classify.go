package handler

import "strings"

// isKnownOpsErrorType returns true if t is a recognized error type used by the
// ops classification pipeline.  Upstream proxies sometimes return garbage values
// (e.g. the Go-serialized literal "<nil>") which would pollute phase/severity
// classification if accepted blindly.
func isKnownOpsErrorType(t string) bool {
	switch t {
	case "invalid_request_error",
		"authentication_error",
		"rate_limit_error",
		"billing_error",
		"subscription_error",
		"upstream_error",
		"overloaded_error",
		"api_error",
		"not_found_error",
		"forbidden_error":
		return true
	}
	return false
}

func normalizeOpsErrorType(errType string, code string) string {
	if errType != "" && isKnownOpsErrorType(errType) {
		return errType
	}
	switch strings.TrimSpace(code) {
	case opsCodeInsufficientBalance:
		return "billing_error"
	case opsCodeUsageLimitExceeded, opsCodeSubscriptionNotFound, opsCodeSubscriptionInvalid:
		return "subscription_error"
	default:
		return "api_error"
	}
}

func classifyOpsPhase(errType, message, code string) string {
	msg := strings.ToLower(message)
	// Standardized phases: request|auth|routing|upstream|network|internal
	// Map billing/concurrency/response => request; scheduling => routing.
	switch strings.TrimSpace(code) {
	case opsCodeInsufficientBalance, opsCodeUsageLimitExceeded, opsCodeSubscriptionNotFound, opsCodeSubscriptionInvalid:
		return "request"
	}

	switch errType {
	case "authentication_error":
		return "auth"
	case "billing_error", "subscription_error":
		return "request"
	case "rate_limit_error":
		if strings.Contains(msg, "concurrency") || strings.Contains(msg, "pending") || strings.Contains(msg, "queue") {
			return "request"
		}
		return "upstream"
	case "invalid_request_error":
		return "request"
	case "upstream_error", "overloaded_error":
		return "upstream"
	case "api_error":
		if strings.Contains(msg, opsErrNoAvailableAccounts) {
			return "routing"
		}
		return "internal"
	default:
		return "internal"
	}
}

func classifyOpsSeverity(errType string, status int) string {
	switch errType {
	case "invalid_request_error", "authentication_error", "billing_error", "subscription_error":
		return "P3"
	}
	if status >= 500 {
		return "P1"
	}
	if status == 429 {
		return "P1"
	}
	if status >= 400 {
		return "P2"
	}
	return "P3"
}

func classifyOpsIsRetryable(errType string, statusCode int) bool {
	switch errType {
	case "authentication_error", "invalid_request_error":
		return false
	case "timeout_error":
		return true
	case "rate_limit_error":
		// May be transient (upstream or queue); retry can help.
		return true
	case "billing_error", "subscription_error":
		return false
	case "upstream_error", "overloaded_error":
		return statusCode >= 500 || statusCode == 429 || statusCode == 529
	default:
		return statusCode >= 500
	}
}

func classifyOpsIsBusinessLimited(errType, phase, code string, status int, message string) bool {
	switch strings.TrimSpace(code) {
	case opsCodeInsufficientBalance, opsCodeUsageLimitExceeded, opsCodeSubscriptionNotFound, opsCodeSubscriptionInvalid, opsCodeUserInactive:
		return true
	}
	if phase == "billing" || phase == "concurrency" {
		// SLA/错误率排除“用户级业务限制”
		return true
	}
	// Avoid treating upstream rate limits as business-limited.
	if errType == "rate_limit_error" && strings.Contains(strings.ToLower(message), "upstream") {
		return false
	}
	_ = status
	return false
}
