package admin

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

type codexSessionImportAudit struct {
	EntryCount            int
	UpdateExisting        bool
	SkipDefaultGroupBind  bool
	ConfirmMixedChannel   bool
	GroupCount            int
	HasProxy              bool
	HasConcurrency        bool
	HasPriority           bool
	HasRateMultiplier     bool
	HasLoadFactor         bool
	HasExpiresAt          bool
	HasAutoPauseOnExpired bool
	HasCredentialExtras   bool
	HasAccountExtra       bool
	ClientRequestID       string
	RequestID             string
	startedAt             time.Time
}

func newCodexSessionImportAudit(c *gin.Context, req CodexSessionImportRequest, entryCount int) codexSessionImportAudit {
	audit := codexSessionImportAudit{
		EntryCount:            entryCount,
		UpdateExisting:        req.UpdateExisting == nil || *req.UpdateExisting,
		SkipDefaultGroupBind:  req.SkipDefaultGroupBind != nil && *req.SkipDefaultGroupBind,
		ConfirmMixedChannel:   req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk,
		GroupCount:            len(req.GroupIDs),
		HasProxy:              req.ProxyID != nil,
		HasConcurrency:        req.Concurrency != nil,
		HasPriority:           req.Priority != nil,
		HasRateMultiplier:     req.RateMultiplier != nil,
		HasLoadFactor:         req.LoadFactor != nil,
		HasExpiresAt:          req.ExpiresAt != nil,
		HasAutoPauseOnExpired: req.AutoPauseOnExpired != nil,
		HasCredentialExtras:   len(req.CredentialExtras) > 0,
		HasAccountExtra:       len(req.Extra) > 0,
	}
	if c != nil && c.Request != nil {
		ctx := c.Request.Context()
		audit.RequestID = contextString(ctx, ctxkey.RequestID)
		audit.ClientRequestID = contextString(ctx, ctxkey.ClientRequestID)
	}
	return audit
}

func (a *codexSessionImportAudit) logStarted(ctx context.Context) {
	a.startedAt = time.Now()
	fields := a.baseFields(ctx)
	slog.Info("admin_codex_session_import_started", fields...)
}

func (a codexSessionImportAudit) logFinished(ctx context.Context, result CodexSessionImportResult, err error) {
	fields := append(a.baseFields(ctx),
		"duration_ms", time.Since(a.startedAt).Milliseconds(),
		"created", result.Created,
		"updated", result.Updated,
		"skipped", result.Skipped,
		"failed", result.Failed,
		"warning_count", len(result.Warnings),
		"error_count", len(result.Errors),
	)
	if err != nil {
		fields = append(fields, "error", err.Error())
		slog.Warn("admin_codex_session_import_failed", fields...)
		return
	}
	status := "success"
	if result.Failed > 0 {
		status = "partial_failed"
	}
	fields = append(fields, "status", status)
	slog.Info("admin_codex_session_import_completed", fields...)
}

func (a codexSessionImportAudit) baseFields(ctx context.Context) []any {
	fields := []any{
		"entry_count", a.EntryCount,
		"update_existing", a.UpdateExisting,
		"skip_default_group_bind", a.SkipDefaultGroupBind,
		"confirm_mixed_channel_risk", a.ConfirmMixedChannel,
		"group_count", a.GroupCount,
		"has_proxy", a.HasProxy,
		"has_concurrency", a.HasConcurrency,
		"has_priority", a.HasPriority,
		"has_rate_multiplier", a.HasRateMultiplier,
		"has_load_factor", a.HasLoadFactor,
		"has_expires_at", a.HasExpiresAt,
		"has_auto_pause_on_expired", a.HasAutoPauseOnExpired,
		"has_credential_extras", a.HasCredentialExtras,
		"has_account_extra", a.HasAccountExtra,
	}
	if requestID := firstNonEmptyString(a.RequestID, contextString(ctx, ctxkey.RequestID)); requestID != "" {
		fields = append(fields, "request_id", requestID)
	}
	if clientRequestID := firstNonEmptyString(a.ClientRequestID, contextString(ctx, ctxkey.ClientRequestID)); clientRequestID != "" {
		fields = append(fields, "client_request_id", clientRequestID)
	}
	return fields
}

func contextString(ctx context.Context, key ctxkey.Key) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(key).(string)
	return strings.TrimSpace(value)
}
