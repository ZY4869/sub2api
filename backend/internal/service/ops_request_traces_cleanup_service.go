package service

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
	"strings"
	"time"
)

func (s *OpsService) CleanupRequestTraces(ctx context.Context, mode OpsRequestTraceCleanupMode, filter *OpsRequestTraceFilter, operatorID int64) (*OpsRequestTraceCleanupResult, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}
	if operatorID <= 0 {
		return nil, infraerrors.BadRequest("OPS_REQUEST_TRACE_CLEANUP_INVALID_OPERATOR", "invalid operator")
	}

	log := logger.FromContext(ctx)
	switch mode {
	case OpsRequestTraceCleanupModeFilter:
		normalized, explicit, err := normalizeRequestTraceCleanupFilter(filter)
		if err != nil {
			return nil, err
		}
		if !explicit {
			return nil, infraerrors.BadRequest("OPS_REQUEST_TRACE_CLEANUP_FILTER_REQUIRED", "cleanup requires at least one filter condition")
		}

		log.Info("ops request trace cleanup started", zap.String("mode", string(mode)), zap.Int64("operator_id", operatorID))
		counts, err := s.opsRepo.DeleteRequestTraces(ctx, normalized)
		if err != nil {
			log.Error("ops request trace cleanup failed", zap.String("mode", string(mode)), zap.Int64("operator_id", operatorID), zap.Error(err))
			return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_CLEANUP_FAILED", "Failed to cleanup request details").WithCause(err)
		}

		result := &OpsRequestTraceCleanupResult{
			Mode:                        mode,
			OpsRequestTraceDeleteCounts: counts,
		}
		_ = s.insertRequestTraceAudit(ctx, nil, operatorID, OpsRequestTraceAuditActionCleanupFilter, map[string]any{
			"deleted_traces": counts.DeletedTraces,
			"deleted_audits": counts.DeletedAudits,
			"filter":         buildRequestTraceCleanupAuditFilterMeta(normalized),
		})
		log.Info("ops request trace cleanup finished", zap.String("mode", string(mode)), zap.Int64("operator_id", operatorID), zap.Int64("deleted_traces", counts.DeletedTraces), zap.Int64("deleted_audits", counts.DeletedAudits))
		return result, nil

	case OpsRequestTraceCleanupModeExpired:
		retentionDays := s.requestDetailRetentionDays(ctx)
		if retentionDays <= 0 {
			return nil, infraerrors.BadRequest("OPS_REQUEST_TRACE_CLEANUP_INVALID_RETENTION", "invalid request detail retention days")
		}

		cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
		log.Info("ops request trace cleanup started", zap.String("mode", string(mode)), zap.Int64("operator_id", operatorID), zap.Time("cutoff", cutoff))
		counts, err := s.opsRepo.DeleteExpiredRequestTraces(ctx, cutoff, 5000)
		if err != nil {
			log.Error("ops request trace cleanup failed", zap.String("mode", string(mode)), zap.Int64("operator_id", operatorID), zap.Error(err))
			return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_CLEANUP_FAILED", "Failed to cleanup request details").WithCause(err)
		}

		result := &OpsRequestTraceCleanupResult{
			Mode:                        mode,
			OpsRequestTraceDeleteCounts: counts,
			Cutoff:                      &cutoff,
		}
		_ = s.insertRequestTraceAudit(ctx, nil, operatorID, OpsRequestTraceAuditActionCleanupExpired, map[string]any{
			"cutoff":         cutoff.Format(time.RFC3339Nano),
			"retention_days": retentionDays,
			"deleted_traces": counts.DeletedTraces,
			"deleted_audits": counts.DeletedAudits,
		})
		log.Info("ops request trace cleanup finished", zap.String("mode", string(mode)), zap.Int64("operator_id", operatorID), zap.Time("cutoff", cutoff), zap.Int64("deleted_traces", counts.DeletedTraces), zap.Int64("deleted_audits", counts.DeletedAudits))
		return result, nil
	default:
		return nil, infraerrors.BadRequest("OPS_REQUEST_TRACE_CLEANUP_INVALID_MODE", "invalid cleanup mode")
	}
}

func normalizeRequestTraceCleanupFilter(filter *OpsRequestTraceFilter) (*OpsRequestTraceFilter, bool, error) {
	filterCopy := &OpsRequestTraceFilter{}
	if filter != nil {
		*filterCopy = *filter
	}
	filterCopy.Page = 0
	filterCopy.PageSize = 0
	filterCopy.Sort = ""
	filterCopy.Limit = 0

	if filterCopy.StartTime != nil && filterCopy.EndTime != nil && filterCopy.StartTime.After(*filterCopy.EndTime) {
		return nil, false, infraerrors.BadRequest("OPS_REQUEST_TRACE_CLEANUP_INVALID_RANGE", "invalid time range")
	}
	return filterCopy, hasRequestTraceCleanupConditions(filterCopy), nil
}

func hasRequestTraceCleanupConditions(filter *OpsRequestTraceFilter) bool {
	if filter == nil {
		return false
	}
	if filter.StartTime != nil || filter.EndTime != nil {
		return true
	}
	for _, value := range []string{
		filter.Status,
		filter.Platform,
		filter.ProtocolIn,
		filter.ProtocolOut,
		filter.Channel,
		filter.RoutePath,
		filter.RequestType,
		filter.FinishReason,
		filter.CaptureReason,
		filter.RequestedModel,
		filter.UpstreamModel,
		filter.RequestID,
		filter.ClientRequestID,
		filter.UpstreamRequestID,
		filter.GeminiSurface,
		filter.BillingRuleID,
		filter.ProbeAction,
		filter.Query,
	} {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	if filter.UserID != nil || filter.APIKeyID != nil || filter.AccountID != nil || filter.GroupID != nil || filter.StatusCode != nil {
		return true
	}
	return filter.Stream != nil || filter.HasTools != nil || filter.HasThinking != nil || filter.RawAvailable != nil || filter.Sampled != nil
}

func buildRequestTraceCleanupAuditFilterMeta(filter *OpsRequestTraceFilter) map[string]any {
	if filter == nil {
		return map[string]any{}
	}
	meta := map[string]any{}
	addString := func(key string, value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			meta[key] = trimmed
		}
	}
	addNumber := func(key string, value *int64) {
		if value != nil {
			meta[key] = *value
		}
	}
	addInt := func(key string, value *int) {
		if value != nil {
			meta[key] = *value
		}
	}
	addBool := func(key string, value *bool) {
		if value != nil {
			meta[key] = *value
		}
	}

	addString("status", filter.Status)
	addString("platform", filter.Platform)
	addString("protocol_in", filter.ProtocolIn)
	addString("protocol_out", filter.ProtocolOut)
	addString("channel", filter.Channel)
	addString("route_path", filter.RoutePath)
	addString("request_type", filter.RequestType)
	addString("finish_reason", filter.FinishReason)
	addString("capture_reason", filter.CaptureReason)
	addString("requested_model", filter.RequestedModel)
	addString("upstream_model", filter.UpstreamModel)
	addString("request_id", filter.RequestID)
	addString("client_request_id", filter.ClientRequestID)
	addString("upstream_request_id", filter.UpstreamRequestID)
	addString("gemini_surface", filter.GeminiSurface)
	addString("billing_rule_id", filter.BillingRuleID)
	addString("probe_action", filter.ProbeAction)
	addString("q", filter.Query)
	addNumber("user_id", filter.UserID)
	addNumber("api_key_id", filter.APIKeyID)
	addNumber("account_id", filter.AccountID)
	addNumber("group_id", filter.GroupID)
	addInt("status_code", filter.StatusCode)
	addBool("stream", filter.Stream)
	addBool("has_tools", filter.HasTools)
	addBool("has_thinking", filter.HasThinking)
	addBool("raw_available", filter.RawAvailable)
	addBool("sampled", filter.Sampled)
	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		meta["start_time"] = filter.StartTime.UTC().Format(time.RFC3339Nano)
	}
	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		meta["end_time"] = filter.EndTime.UTC().Format(time.RFC3339Nano)
	}
	return meta
}
