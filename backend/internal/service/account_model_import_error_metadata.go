package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
)

const (
	accountModelImportReasonKindOpenAIScopeInsufficient = "openai_scope_insufficient"
	accountModelImportReasonKindGoogleScopeInsufficient = "google_scope_insufficient"
	accountModelImportReasonKindPermissionDenied        = "permission_denied"
	accountModelImportReasonKindUnauthorized            = "unauthorized"
	accountModelImportReasonKindRateLimited             = "rate_limited"
	accountModelImportReasonKindInvalidResponse         = "upstream_invalid_response"

	accountModelImportHintKeyOpenAIModelRead        = "openai_api_model_read"
	accountModelImportHintKeyGoogleAccessTokenScope = "google_access_token_scope"
	accountModelImportHintKeyPermissionDenied       = "permission_denied"
	accountModelImportHintKeyUnauthorized           = "unauthorized"
	accountModelImportHintKeyRateLimited            = "rate_limited"
	accountModelImportHintKeyInvalidResponse        = "upstream_invalid_response"
)

func accountModelImportProvider(account *Account) string {
	if account == nil {
		return ""
	}
	if account.IsGeminiVertexSource() {
		return "vertex"
	}
	return strings.TrimSpace(strings.ToLower(RoutingPlatformForAccount(account)))
}

func newAccountModelImportUpstreamStatusErrorForAccount(
	account *Account,
	operation string,
	statusCode int,
	headers http.Header,
	body []byte,
) error {
	message := fmt.Sprintf("%s with status %d", strings.TrimSpace(operation), statusCode)
	if truncated := summarizeAccountModelImportErrorBody(body); truncated != "" {
		message = fmt.Sprintf("%s: %s", message, truncated)
	}
	provider := accountModelImportProvider(account)
	metadata := classifyAccountModelImportUpstreamMetadata(provider, accountModelImportAuthMode(account), statusCode, headers, body)
	switch statusCode {
	case http.StatusUnauthorized:
		return infraerrors.Unauthorized("MODEL_IMPORT_UPSTREAM_UNAUTHORIZED", message).WithMetadata(metadata)
	case http.StatusForbidden:
		return infraerrors.Forbidden("MODEL_IMPORT_UPSTREAM_FORBIDDEN", message).WithMetadata(metadata)
	case http.StatusTooManyRequests:
		return infraerrors.TooManyRequests("MODEL_IMPORT_UPSTREAM_RATE_LIMITED", message).WithMetadata(metadata)
	default:
		if statusCode >= http.StatusInternalServerError {
			return infraerrors.ServiceUnavailable("MODEL_IMPORT_UPSTREAM_SERVER_ERROR", message).WithMetadata(metadata)
		}
		return infraerrors.BadRequest("MODEL_IMPORT_UPSTREAM_FAILED", message).WithMetadata(metadata)
	}
}

func newAccountModelImportInvalidResponseError(account *Account, operation string, cause error) error {
	message := strings.TrimSpace(operation)
	if message == "" {
		message = "upstream model listing returned invalid response"
	}
	err := infraerrors.BadRequest("MODEL_IMPORT_INVALID_RESPONSE", message).WithMetadata(map[string]string{
		"provider":        accountModelImportProvider(account),
		"auth_mode":       accountModelImportAuthMode(account),
		"reason_kind":     accountModelImportReasonKindInvalidResponse,
		"hint_key":        accountModelImportHintKeyInvalidResponse,
		"upstream_status": "",
		"raw_summary":     "",
	})
	if cause == nil {
		return err
	}
	summary := infraerrors.Message(cause)
	if strings.TrimSpace(summary) == "" || summary == infraerrors.UnknownMessage {
		summary = strings.TrimSpace(cause.Error())
	}
	err.Metadata["raw_summary"] = summary
	return err.WithCause(cause)
}

func classifyAccountModelImportUpstreamMetadata(
	provider string,
	authMode string,
	statusCode int,
	headers http.Header,
	body []byte,
) map[string]string {
	reasonKind := ""
	hintKey := ""
	lowerBody := strings.ToLower(string(body))
	switch {
	case statusCode == http.StatusTooManyRequests:
		reasonKind = accountModelImportReasonKindRateLimited
		hintKey = accountModelImportHintKeyRateLimited
	case statusCode == http.StatusUnauthorized:
		reasonKind = accountModelImportReasonKindUnauthorized
		hintKey = accountModelImportHintKeyUnauthorized
	case statusCode == http.StatusForbidden && isOpenAIModelReadScopeError(lowerBody):
		reasonKind = accountModelImportReasonKindOpenAIScopeInsufficient
		hintKey = accountModelImportHintKeyOpenAIModelRead
	case statusCode == http.StatusForbidden && isGoogleScopeInsufficient(headers, lowerBody):
		reasonKind = accountModelImportReasonKindGoogleScopeInsufficient
		hintKey = accountModelImportHintKeyGoogleAccessTokenScope
	case statusCode == http.StatusForbidden:
		reasonKind = accountModelImportReasonKindPermissionDenied
		hintKey = accountModelImportHintKeyPermissionDenied
	default:
		reasonKind = accountModelImportReasonKindInvalidResponse
		hintKey = accountModelImportHintKeyInvalidResponse
	}

	return map[string]string{
		"provider":        provider,
		"auth_mode":       authMode,
		"upstream_status": strconv.Itoa(statusCode),
		"reason_kind":     reasonKind,
		"hint_key":        hintKey,
		"raw_summary":     summarizeAccountModelImportErrorBody(body),
	}
}

func summarizeAccountModelImportErrorBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if parsed, err := googleapi.ParseError(string(body)); err == nil && parsed != nil {
		message := strings.TrimSpace(parsed.Error.Message)
		status := strings.TrimSpace(parsed.Error.Status)
		if message != "" && status != "" {
			return fmt.Sprintf("%s (%s)", message, status)
		}
		if message != "" {
			return message
		}
	}
	return truncateImportBody(body)
}

func isOpenAIModelReadScopeError(lowerBody string) bool {
	return strings.Contains(lowerBody, "api.model.read") ||
		strings.Contains(lowerBody, "restricted api key") ||
		strings.Contains(lowerBody, "scoped api key") ||
		strings.Contains(lowerBody, "missing scopes")
}

func isGoogleScopeInsufficient(headers http.Header, lowerBody string) bool {
	if strings.Contains(strings.ToLower(headers.Get("Www-Authenticate")), "insufficient_scope") {
		return true
	}
	return strings.Contains(lowerBody, "insufficient authentication scopes") ||
		strings.Contains(lowerBody, "access_token_scope_insufficient")
}
