package handler

import (
	"log/slog"
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

type geminiErrorLocalization struct {
	messageKey string
	fallback   string
}

var geminiReasonLocalizations = map[string]geminiErrorLocalization{
	"API_KEY_RATE_5H_EXCEEDED": {
		messageKey: "gateway.gemini.rate_limit_exceeded",
		fallback:   "Gateway rate limit exceeded, please retry later",
	},
	"API_KEY_RATE_1D_EXCEEDED": {
		messageKey: "gateway.gemini.rate_limit_exceeded",
		fallback:   "Gateway rate limit exceeded, please retry later",
	},
	"API_KEY_RATE_7D_EXCEEDED": {
		messageKey: "gateway.gemini.rate_limit_exceeded",
		fallback:   "Gateway rate limit exceeded, please retry later",
	},
	"BILLING_SERVICE_ERROR": {
		messageKey: "gateway.gemini.billing_service_unavailable",
		fallback:   "Billing service temporarily unavailable, please retry later",
	},
	"GOOGLE_ARCHIVE_FILE_NOT_FOUND": {
		messageKey: "gateway.gemini.archive_file_not_found",
		fallback:   "Archive file not found",
	},
	"GOOGLE_BATCH_ARCHIVE_NOT_FOUND": {
		messageKey: "gateway.gemini.archive_batch_not_found",
		fallback:   "Archive batch not found",
	},
	"GOOGLE_BATCH_NO_ACCOUNT": {
		messageKey: "gateway.gemini.batch_no_account",
		fallback:   "No available Google batch accounts",
	},
	"GOOGLE_BATCH_PATH_UNSUPPORTED": {
		messageKey: "gateway.gemini.batch_path_unsupported",
		fallback:   "Unsupported Gemini Batch path",
	},
	"GOOGLE_FILE_DOWNLOAD_NOT_FOUND": {
		messageKey: "gateway.gemini.file_download_not_found",
		fallback:   "File not found",
	},
	"GOOGLE_FILES_PATH_UNSUPPORTED": {
		messageKey: "gateway.gemini.files_path_unsupported",
		fallback:   "Unsupported Gemini Files path",
	},
	"GROUP_EXHAUSTED": {
		messageKey: "gateway.gemini.group_exhausted",
		fallback:   "All accounts in the selected group have been exhausted",
	},
	"INVALID_GROUP_BINDING": {
		messageKey: "gateway.gemini.invalid_group_binding",
		fallback:   "Selected API key group binding is invalid",
	},
	"NO_AVAILABLE_GROUP": {
		messageKey: "gateway.gemini.no_available_group",
		fallback:   "No available group for this request",
	},
	"SUBSCRIPTION_REQUIRED": {
		messageKey: "gateway.gemini.subscription_required",
		fallback:   "Active subscription required for this group",
	},
	"VERTEX_BATCH_PATH_INVALID": {
		messageKey: "gateway.gemini.vertex_batch_path_invalid",
		fallback:   "Vertex batch request path is invalid",
	},
}

func googleErrorBodyTooLarge(c *gin.Context, limit int64) {
	googleErrorKey(c, http.StatusRequestEntityTooLarge, "gateway.gemini.request_body_too_large", "Request body too large, limit is %s", formatBodyLimit(limit))
}

func googleErrorPendingRequests(c *gin.Context) {
	googleErrorKey(c, http.StatusTooManyRequests, "gateway.gemini.pending_requests", "Too many pending requests, please retry later")
}

func googleNoAvailableAccountsError(c *gin.Context, err error) {
	if strings.TrimSpace(infraerrors.Reason(err)) != "" {
		googleErrorFromServiceError(c, err)
		return
	}
	if err != nil {
		slog.Warn("gateway_gemini_no_available_accounts_unkeyed", "error", err.Error())
	}
	googleErrorKey(c, http.StatusServiceUnavailable, "gateway.gemini.no_available_accounts", "No available Gemini accounts")
}

func geminiReasonLocalization(reason string) (geminiErrorLocalization, bool) {
	localization, ok := geminiReasonLocalizations[strings.TrimSpace(reason)]
	return localization, ok
}

func googleErrorFromServiceError(c *gin.Context, err error) {
	if err == nil {
		googleErrorKey(c, http.StatusBadGateway, "gateway.gemini.upstream_failed", "Upstream request failed")
		return
	}

	appErr := infraerrors.FromError(err)
	status := http.StatusInternalServerError
	if appErr != nil && appErr.Code > 0 {
		status = int(appErr.Code)
	}

	reason := strings.TrimSpace(infraerrors.Reason(err))
	if reason == "" {
		slog.Warn("gateway_gemini_service_error_unkeyed", "status", status, "error", err.Error())
		googleErrorKey(c, status, "gateway.gemini.request_failed", "Request failed")
		return
	}

	if localization, ok := geminiReasonLocalization(reason); ok {
		googleErrorWithReason(c, status, reason, localization.messageKey, localization.fallback)
		return
	}

	message := ""
	if appErr != nil {
		message = strings.TrimSpace(appErr.Message)
	}
	slog.Warn("gateway_gemini_service_error_unmapped", "status", status, "reason", reason, "message", message, "error", err.Error())
	googleErrorWithReason(c, status, reason, "gateway.gemini.request_failed", "Request failed")
}
