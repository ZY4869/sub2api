package service

import (
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrChannelMonitorNotFound      = infraerrors.NotFound("CHANNEL_MONITOR_NOT_FOUND", "channel monitor not found")
	ErrChannelMonitorAlreadyExists = infraerrors.Conflict("CHANNEL_MONITOR_ALREADY_EXISTS", "channel monitor already exists")

	ErrChannelMonitorTemplateNotFound      = infraerrors.NotFound("CHANNEL_MONITOR_TEMPLATE_NOT_FOUND", "channel monitor template not found")
	ErrChannelMonitorTemplateAlreadyExists = infraerrors.Conflict("CHANNEL_MONITOR_TEMPLATE_ALREADY_EXISTS", "channel monitor template already exists")

	ErrChannelMonitorInvalidProvider     = infraerrors.BadRequest("CHANNEL_MONITOR_INVALID_PROVIDER", "invalid provider")
	ErrChannelMonitorInvalidEndpoint     = infraerrors.BadRequest("CHANNEL_MONITOR_INVALID_ENDPOINT", "invalid endpoint")
	ErrChannelMonitorEndpointNotAllowed  = infraerrors.BadRequest("CHANNEL_MONITOR_ENDPOINT_NOT_ALLOWED", "endpoint is not allowed")
	ErrChannelMonitorInvalidInterval     = infraerrors.BadRequest("CHANNEL_MONITOR_INVALID_INTERVAL", "invalid interval")
	ErrChannelMonitorAPIKeyRequired      = infraerrors.BadRequest("CHANNEL_MONITOR_API_KEY_REQUIRED", "api key is required")
	ErrChannelMonitorAPIKeyDecryptFailed = infraerrors.BadRequest("CHANNEL_MONITOR_API_KEY_DECRYPT_FAILED", "api key decrypt failed")
	ErrChannelMonitorInvalidHeaders      = infraerrors.BadRequest("CHANNEL_MONITOR_EXTRA_HEADERS_INVALID", "invalid extra headers")
	ErrChannelMonitorInvalidBodyOverride = infraerrors.BadRequest("CHANNEL_MONITOR_BODY_OVERRIDE_INVALID", "invalid body override")
	ErrChannelMonitorInvalidOverrideMode = infraerrors.BadRequest("CHANNEL_MONITOR_BODY_OVERRIDE_MODE_INVALID", "invalid body override mode")
)
