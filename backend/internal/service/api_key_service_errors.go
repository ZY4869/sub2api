package service

import (
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrAPIKeyNotFound               = infraerrors.NotFound("API_KEY_NOT_FOUND", "api key not found")
	ErrGroupNotAllowed              = infraerrors.Forbidden("GROUP_NOT_ALLOWED", "user is not allowed to bind this group")
	ErrAPIKeyExists                 = infraerrors.Conflict("API_KEY_EXISTS", "api key already exists")
	ErrAPIKeyTooShort               = infraerrors.BadRequest("API_KEY_TOO_SHORT", "api key must be at least 16 characters")
	ErrAPIKeyInvalidChars           = infraerrors.BadRequest("API_KEY_INVALID_CHARS", "api key can only contain letters, numbers, underscores, and hyphens")
	ErrAPIKeyRateLimited            = infraerrors.TooManyRequests("API_KEY_RATE_LIMITED", "too many failed attempts, please try again later")
	ErrInvalidIPPattern             = infraerrors.BadRequest("INVALID_IP_PATTERN", "invalid IP or CIDR pattern")
	ErrAPIKeyModelSelectionRequired = infraerrors.BadRequest(
		"API_KEY_MODEL_SELECTION_REQUIRED",
		"api key must select at least one visible model for each group",
	)
	ErrAPIKeyModelNotVisible = infraerrors.BadRequest(
		"API_KEY_MODEL_NOT_VISIBLE",
		"selected model is not visible in the target group",
	)
	ErrAPIKeyModelPatternForbidden = infraerrors.BadRequest(
		"API_KEY_MODEL_PATTERN_FORBIDDEN",
		"model patterns are not allowed for user api keys",
	)
	// ErrAPIKeyExpired        = infraerrors.Forbidden("API_KEY_EXPIRED", "api key has expired")
	ErrAPIKeyExpired = infraerrors.Forbidden("API_KEY_EXPIRED", "api key 已过期")
	// ErrAPIKeyQuotaExhausted = infraerrors.TooManyRequests("API_KEY_QUOTA_EXHAUSTED", "api key quota exhausted")
	ErrAPIKeyQuotaExhausted = infraerrors.TooManyRequests("API_KEY_QUOTA_EXHAUSTED", "api key 额度已用完")

	ErrAPIKeyRateLimit5hExceeded = infraerrors.TooManyRequests("API_KEY_RATE_5H_EXCEEDED", "api key 5小时限额已用完")
	ErrAPIKeyRateLimit1dExceeded = infraerrors.TooManyRequests("API_KEY_RATE_1D_EXCEEDED", "api key 日限额已用完")
	ErrAPIKeyRateLimit7dExceeded = infraerrors.TooManyRequests("API_KEY_RATE_7D_EXCEEDED", "api key 7天限额已用完")
)

const (
	apiKeyMaxErrorsPerHour = 20
	apiKeyLastUsedMinTouch = 30 * time.Second
	// DB 写失败后的短退避，避免请求路径持续同步重试造成写风暴与高延迟。
	apiKeyLastUsedFailBackoff = 5 * time.Second
)
