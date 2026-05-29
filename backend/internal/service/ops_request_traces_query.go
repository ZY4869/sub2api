package service

import (
	"context"
	"database/sql"
	"errors"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"strings"
)

func (s *OpsService) ListRequestTraces(ctx context.Context, operatorID int64, filter *OpsRequestTraceFilter) (*OpsRequestTraceList, error) {
	if err := s.requireRequestTraceEnabled(ctx); err != nil {
		return nil, err
	}
	if s.opsRepo == nil {
		return &OpsRequestTraceList{Items: []*OpsRequestTraceListItem{}, Total: 0, Page: 1, PageSize: 50}, nil
	}

	result, err := s.opsRepo.ListRequestTraces(ctx, filter)
	if err != nil {
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_LIST_FAILED", "Failed to list request details").WithCause(err)
	}

	rawAllowed := s.canAccessRequestTraceRaw(ctx, operatorID)
	for _, item := range result.Items {
		if item == nil {
			continue
		}
		item.RawAccessAllowed = rawAllowed && item.RawAvailable
	}
	return result, nil
}

func (s *OpsService) GetRequestTraceSummary(ctx context.Context, operatorID int64, filter *OpsRequestTraceFilter) (*OpsRequestTraceSummary, error) {
	if err := s.requireRequestTraceEnabled(ctx); err != nil {
		return nil, err
	}
	if s.opsRepo == nil {
		return &OpsRequestTraceSummary{RawAccessAllowed: s.canAccessRequestTraceRaw(ctx, operatorID)}, nil
	}

	summary, err := s.opsRepo.GetRequestTraceSummary(ctx, filter)
	if err != nil {
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_SUMMARY_FAILED", "Failed to load request detail summary").WithCause(err)
	}
	summary.RawAccessAllowed = s.canAccessRequestTraceRaw(ctx, operatorID)
	return summary, nil
}

func (s *OpsService) GetRequestTraceByID(ctx context.Context, operatorID, id int64) (*OpsRequestTraceDetail, error) {
	if err := s.requireRequestTraceEnabled(ctx); err != nil {
		return nil, err
	}
	if s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("OPS_REQUEST_TRACE_INVALID_ID", "invalid request detail id")
	}

	detail, err := s.opsRepo.GetRequestTraceByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("OPS_REQUEST_TRACE_NOT_FOUND", "request detail not found")
		}
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_LOAD_FAILED", "Failed to load request detail").WithCause(err)
	}
	audits, err := s.opsRepo.ListRequestTraceAudits(ctx, id)
	if err != nil {
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_AUDITS_FAILED", "Failed to load request trace audits").WithCause(err)
	}

	detail.Audits = audits
	detail.RawAccessAllowed = s.canAccessRequestTraceRaw(ctx, operatorID) && detail.RawAvailable
	return detail, nil
}

func (s *OpsService) GetRequestTraceRawByID(ctx context.Context, operatorID, id int64) (*OpsRequestTraceRawDetail, error) {
	if err := s.requireRequestTraceEnabled(ctx); err != nil {
		return nil, err
	}
	if !s.canAccessRequestTraceRaw(ctx, operatorID) {
		return nil, infraerrors.Forbidden("OPS_REQUEST_TRACE_RAW_FORBIDDEN", "raw request detail access is not allowed")
	}
	if s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("OPS_REQUEST_TRACE_INVALID_ID", "invalid request detail id")
	}

	runtimeCfg := s.getOpsRequestTraceRuntimeConfig(ctx)
	if strings.TrimSpace(runtimeCfg.EncryptionKey) == "" {
		return nil, infraerrors.BadRequest("OPS_REQUEST_TRACE_RAW_DISABLED", "raw request detail capture is not configured")
	}

	raw, err := s.opsRepo.GetRequestTraceRawByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("OPS_REQUEST_TRACE_NOT_FOUND", "request detail not found")
		}
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_RAW_LOAD_FAILED", "Failed to load raw request detail").WithCause(err)
	}

	if strings.TrimSpace(raw.RawRequest) == "" && strings.TrimSpace(raw.RawResponse) == "" {
		return nil, infraerrors.NotFound("OPS_REQUEST_TRACE_RAW_NOT_FOUND", "raw request detail not available")
	}

	if rawRequest, err := decryptOpsRequestTracePayload(runtimeCfg.EncryptionKey, []byte(raw.RawRequest)); err != nil {
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_RAW_DECRYPT_FAILED", "Failed to decrypt raw request detail").WithCause(err)
	} else {
		raw.RawRequest = string(rawRequest)
	}
	if rawResponse, err := decryptOpsRequestTracePayload(runtimeCfg.EncryptionKey, []byte(raw.RawResponse)); err != nil {
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_RAW_DECRYPT_FAILED", "Failed to decrypt raw response detail").WithCause(err)
	} else {
		raw.RawResponse = string(rawResponse)
	}

	traceID := id
	_ = s.insertRequestTraceAudit(ctx, &traceID, operatorID, OpsRequestTraceAuditActionViewRaw, map[string]any{
		"request_id": raw.RequestID,
	})
	return raw, nil
}

func (s *OpsService) GetUsageRequestPreviewForUsage(ctx context.Context, usage *UsageLog) (*UsageRequestPreview, error) {
	if usage == nil {
		return normalizeUsageRequestPreview(nil, ""), nil
	}

	fallback := newUnavailableUsageRequestPreview(usage.RequestID)
	if s == nil || s.opsRepo == nil || !s.IsMonitoringEnabled(ctx) || !s.getOpsRequestTraceRuntimeConfig(ctx).Enabled {
		return fallback, nil
	}
	if usage.UserID <= 0 || usage.APIKeyID <= 0 || strings.TrimSpace(usage.RequestID) == "" {
		return fallback, nil
	}

	preview, err := s.opsRepo.GetUsageRequestPreview(ctx, usage.UserID, usage.APIKeyID, usage.RequestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fallback, nil
		}
		return nil, infraerrors.InternalServer("OPS_REQUEST_TRACE_PREVIEW_FAILED", "Failed to load request detail preview").WithCause(err)
	}
	return normalizeUsageRequestPreview(preview, usage.RequestID), nil
}
