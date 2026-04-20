package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	opsRequestTraceDisabledMessage     = "ops request details is disabled"
	opsRequestTraceListPageSize        = 200
	opsRequestTraceExportMaxRows       = 50000
	opsRequestTraceRawExportMaxWindow  = 7 * 24 * time.Hour
	opsRequestTraceExportMaxWindow     = 30 * 24 * time.Hour
	opsRequestTraceInboundPreviewLimit = 512 * 1024
	opsRequestTraceRawRequestLimit     = 512 * 1024
	opsRequestTraceRawResponseLimit    = 1024 * 1024
	opsRequestTraceSearchTextLimit     = 4096
	opsRequestTracePayloadJSONLimit    = 64 * 1024
)

var ErrOpsRequestTracesDisabled = infraerrors.NotFound("OPS_REQUEST_TRACES_DISABLED", opsRequestTraceDisabledMessage)

type OpsRecordRequestTraceInput struct {
	RequestID          string
	ClientRequestID    string
	UpstreamRequestID  string
	UserID             *int64
	APIKeyID           *int64
	AccountID          *int64
	GroupID            *int64
	Status             string
	StatusCode         int
	UpstreamStatusCode *int
	DurationMs         int64
	TTFTMs             *int64
	InputTokens        int
	OutputTokens       int
	TotalTokens        int
	Trace              GatewayTraceContext
	CreatedAt          time.Time
}

type opsRequestTraceRuntimeConfig struct {
	Enabled            bool
	EncryptionKey      string
	RawAccessUserIDs   map[int64]struct{}
	RetentionDays      int
	SuccessSampleRate  float64
	ForceCaptureSlowMs int64
	RawExportMaxRows   int
}

func (s *OpsService) RecordRequestTrace(ctx context.Context, input *OpsRecordRequestTraceInput) error {
	if input == nil || !s.IsMonitoringEnabled(ctx) || s.opsRepo == nil {
		return nil
	}

	runtimeCfg := s.getOpsRequestTraceRuntimeConfig(ctx)
	if !runtimeCfg.Enabled {
		return nil
	}

	decision := evaluateOpsRequestTraceDecision(runtimeCfg, input)
	if !decision.Capture {
		return nil
	}

	recordedAt := input.CreatedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	insert := &OpsInsertRequestTraceInput{
		RequestID:           strings.TrimSpace(input.RequestID),
		ClientRequestID:     strings.TrimSpace(input.ClientRequestID),
		UpstreamRequestID:   strings.TrimSpace(firstNonEmptyString(input.UpstreamRequestID, input.Trace.Normalize.UpstreamRequestID)),
		GeminiSurface:       strings.TrimSpace(input.Trace.Normalize.GeminiSurface),
		BillingRuleID:       strings.TrimSpace(input.Trace.Normalize.BillingRuleID),
		ProbeAction:         strings.TrimSpace(input.Trace.Normalize.ProbeAction),
		UserID:              input.UserID,
		APIKeyID:            input.APIKeyID,
		AccountID:           input.AccountID,
		GroupID:             input.GroupID,
		Platform:            strings.TrimSpace(input.Trace.Normalize.Platform),
		ProtocolIn:          strings.TrimSpace(input.Trace.Normalize.ProtocolIn),
		ProtocolOut:         strings.TrimSpace(input.Trace.Normalize.ProtocolOut),
		Channel:             strings.TrimSpace(input.Trace.Normalize.Channel),
		RoutePath:           strings.TrimSpace(input.Trace.Normalize.RoutePath),
		UpstreamPath:        strings.TrimSpace(input.Trace.Normalize.UpstreamPath),
		RequestType:         strings.TrimSpace(input.Trace.Normalize.RequestType),
		RequestedModel:      strings.TrimSpace(input.Trace.Normalize.RequestedModel),
		UpstreamModel:       strings.TrimSpace(input.Trace.Normalize.UpstreamModel),
		ActualUpstreamModel: strings.TrimSpace(input.Trace.Normalize.ActualUpstreamModel),
		Status:              normalizeOpsRequestTraceStatus(input.Status, input.StatusCode),
		StatusCode:          input.StatusCode,
		UpstreamStatusCode:  input.UpstreamStatusCode,
		DurationMs:          input.DurationMs,
		TTFTMs:              input.TTFTMs,
		InputTokens:         input.InputTokens,
		OutputTokens:        input.OutputTokens,
		TotalTokens:         input.TotalTokens,
		FinishReason:        strings.TrimSpace(input.Trace.Normalize.FinishReason),
		PromptBlockReason:   strings.TrimSpace(input.Trace.Normalize.PromptBlockReason),
		Stream:              input.Trace.Normalize.Stream,
		HasTools:            input.Trace.Normalize.HasTools,
		ToolKinds:           dedupeNonEmptyStrings(input.Trace.Normalize.ToolKinds),
		HasThinking:         input.Trace.Normalize.HasThinking,
		ThinkingSource:      strings.TrimSpace(input.Trace.Normalize.ThinkingSource),
		ThinkingLevel:       strings.TrimSpace(input.Trace.Normalize.ThinkingLevel),
		ThinkingBudget:      input.Trace.Normalize.ThinkingBudget,
		MediaResolution:     strings.TrimSpace(input.Trace.Normalize.MediaResolution),
		CountTokensSource:   strings.TrimSpace(input.Trace.Normalize.CountTokensSource),
		CaptureReason:       decision.Reason,
		Sampled:             decision.Sampled,
		CreatedAt:           recordedAt,
	}

	normalizationActions := make([]string, 0, 8)
	normalizeJSONBField := func(field string, source string, contentType string, value *string) *string {
		result := normalizeOpsTraceJSONBPayload(value, source, contentType)
		if result.Action != "" {
			normalizationActions = append(normalizationActions, field+":"+result.Action)
		}
		return result.Value
	}

	insert.InboundRequestJSON = normalizeJSONBField(
		"inbound_request",
		"ops_trace_inbound_request_jsonb",
		"application/json",
		input.Trace.InboundRequestJSON,
	)
	if insert.InboundRequestJSON == nil {
		insert.InboundRequestJSON = normalizeJSONBField(
			"inbound_request_fallback",
			"ops_trace_inbound_request_jsonb_fallback",
			"application/json",
			sanitizeTracePayloadForStorage(input.Trace.RawRequest, opsRequestTraceInboundPreviewLimit, "application/json"),
		)
	}
	insert.NormalizedRequestJSON = normalizeJSONBField(
		"normalized_request",
		"ops_trace_normalized_request_jsonb",
		"application/json",
		input.Trace.NormalizedRequestJSON,
	)
	insert.UpstreamRequestJSON = normalizeJSONBField(
		"upstream_request",
		"ops_trace_upstream_request_jsonb",
		"application/json",
		input.Trace.UpstreamRequestJSON,
	)
	insert.UpstreamResponseJSON = normalizeJSONBField(
		"upstream_response",
		"ops_trace_upstream_response_jsonb",
		"application/json",
		input.Trace.UpstreamResponseJSON,
	)
	insert.GatewayResponseJSON = normalizeJSONBField(
		"gateway_response",
		"ops_trace_gateway_response_jsonb",
		"application/json",
		input.Trace.GatewayResponseJSON,
	)
	if insert.GatewayResponseJSON == nil {
		insert.GatewayResponseJSON = normalizeJSONBField(
			"gateway_response_fallback",
			"ops_trace_gateway_response_jsonb_fallback",
			"application/json",
			sanitizeTracePayloadForStorage(input.Trace.RawResponse, opsRequestTracePayloadJSONLimit, ""),
		)
	}
	insert.ToolTraceJSON = normalizeJSONBField(
		"tool_trace",
		"ops_trace_tool_trace_jsonb",
		"application/json",
		input.Trace.ToolTraceJSON,
	)
	insert.RequestHeadersJSON = normalizeJSONBField(
		"request_headers",
		"ops_trace_request_headers_jsonb",
		"application/json",
		input.Trace.RequestHeadersJSON,
	)
	insert.ResponseHeadersJSON = normalizeJSONBField(
		"response_headers",
		"ops_trace_response_headers_jsonb",
		"application/json",
		input.Trace.ResponseHeadersJSON,
	)
	if len(normalizationActions) > 0 {
		logger.FromContext(ctx).Debug(
			"ops request trace jsonb payload normalized",
			zap.String("request_id", insert.RequestID),
			zap.Strings("actions", normalizationActions),
		)
	}

	if decision.RawEnabled {
		if ciphertext, size, truncated, err := buildEncryptedTracePayload(runtimeCfg.EncryptionKey, input.Trace.RawRequest, opsRequestTraceRawRequestLimit); err != nil {
			log.Printf("[Ops] RecordRequestTrace raw request encryption failed: %v", err)
		} else {
			insert.RawRequestCiphertext = ciphertext
			insert.RawRequestBytes = size
			insert.RawRequestTruncated = truncated
		}
		if ciphertext, size, truncated, err := buildEncryptedTracePayload(runtimeCfg.EncryptionKey, input.Trace.RawResponse, opsRequestTraceRawResponseLimit); err != nil {
			log.Printf("[Ops] RecordRequestTrace raw response encryption failed: %v", err)
		} else {
			insert.RawResponseCiphertext = ciphertext
			insert.RawResponseBytes = size
			insert.RawResponseTruncated = truncated
		}
		insert.RawAvailable = len(insert.RawRequestCiphertext) > 0 || len(insert.RawResponseCiphertext) > 0
	}

	insert.SearchText = buildOpsRequestTraceSearchText(insert)
	if _, err := s.opsRepo.InsertRequestTrace(ctx, insert); err != nil {
		log.Printf("[Ops] RecordRequestTrace failed: %v", err)
		return err
	}
	return nil
}

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

func (s *OpsService) ExportRequestTracesCSV(ctx context.Context, writer io.Writer, operatorID int64, filter *OpsRequestTraceFilter, includeRaw bool) (int, error) {
	if writer == nil {
		return 0, infraerrors.BadRequest("OPS_REQUEST_TRACE_EXPORT_INVALID_WRITER", "invalid export writer")
	}
	if err := s.requireRequestTraceEnabled(ctx); err != nil {
		return 0, err
	}
	if s.opsRepo == nil {
		return 0, infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}

	runtimeCfg := s.getOpsRequestTraceRuntimeConfig(ctx)
	if includeRaw && !s.canAccessRequestTraceRaw(ctx, operatorID) {
		return 0, infraerrors.Forbidden("OPS_REQUEST_TRACE_RAW_FORBIDDEN", "raw request detail export is not allowed")
	}
	window, err := normalizeRequestTraceExportWindow(filter)
	if err != nil {
		return 0, err
	}
	if includeRaw && window > opsRequestTraceRawExportMaxWindow {
		return 0, infraerrors.BadRequest("OPS_REQUEST_TRACE_EXPORT_WINDOW_TOO_LARGE", "raw export supports up to 7 days only")
	}
	if !includeRaw && window > opsRequestTraceExportMaxWindow {
		return 0, infraerrors.BadRequest("OPS_REQUEST_TRACE_EXPORT_WINDOW_TOO_LARGE", "export supports up to 30 days only")
	}

	pageSize := opsRequestTraceListPageSize
	if includeRaw {
		pageSize = 100
	}
	filterCopy := &OpsRequestTraceFilter{}
	if filter != nil {
		*filterCopy = *filter
	}
	filterCopy.Page = 1
	filterCopy.PageSize = pageSize

	firstPage, err := s.opsRepo.ListRequestTraces(ctx, filterCopy)
	if err != nil {
		return 0, infraerrors.InternalServer("OPS_REQUEST_TRACE_EXPORT_FAILED", "Failed to export request details").WithCause(err)
	}

	rowLimit := opsRequestTraceExportMaxRows
	if includeRaw {
		rowLimit = runtimeCfg.RawExportMaxRows
	}
	if firstPage.Total > int64(rowLimit) {
		return 0, infraerrors.BadRequest("OPS_REQUEST_TRACE_EXPORT_TOO_LARGE", "Too many rows to export, please narrow the filter range")
	}

	csvWriter := csv.NewWriter(writer)
	headers := []string{
		"id", "created_at", "request_id", "client_request_id", "upstream_request_id",
		"platform", "protocol_in", "protocol_out", "channel", "route_path", "request_type",
		"user_id", "api_key_id", "account_id", "group_id",
		"requested_model", "upstream_model", "actual_upstream_model",
		"status", "status_code", "upstream_status_code", "duration_ms", "ttft_ms",
		"input_tokens", "output_tokens", "total_tokens",
		"finish_reason", "prompt_block_reason", "stream", "has_tools", "tool_kinds",
		"has_thinking", "thinking_source", "thinking_level", "thinking_budget",
		"media_resolution", "count_tokens_source", "capture_reason", "sampled", "raw_available",
	}
	if includeRaw {
		headers = append(headers, "raw_request", "raw_response")
	}
	if err := csvWriter.Write(headers); err != nil {
		return 0, infraerrors.InternalServer("OPS_REQUEST_TRACE_EXPORT_FAILED", "Failed to write export header").WithCause(err)
	}

	rawTraceCache := make(map[int64]*OpsRequestTraceRawDetail, 16)
	writeRows := func(items []*OpsRequestTraceListItem) (int, error) {
		written := 0
		for _, item := range items {
			if item == nil {
				continue
			}
			row := []string{
				strconv.FormatInt(item.ID, 10),
				item.CreatedAt.UTC().Format(time.RFC3339),
				item.RequestID,
				item.ClientRequestID,
				item.UpstreamRequestID,
				item.Platform,
				item.ProtocolIn,
				item.ProtocolOut,
				item.Channel,
				item.RoutePath,
				item.RequestType,
				formatInt64Pointer(item.UserID),
				formatInt64Pointer(item.APIKeyID),
				formatInt64Pointer(item.AccountID),
				formatInt64Pointer(item.GroupID),
				item.RequestedModel,
				item.UpstreamModel,
				item.ActualUpstreamModel,
				item.Status,
				strconv.Itoa(item.StatusCode),
				formatIntPointer(item.UpstreamStatusCode),
				strconv.FormatInt(item.DurationMs, 10),
				formatInt64Pointer(item.TTFTMs),
				strconv.Itoa(item.InputTokens),
				strconv.Itoa(item.OutputTokens),
				strconv.Itoa(item.TotalTokens),
				item.FinishReason,
				item.PromptBlockReason,
				strconv.FormatBool(item.Stream),
				strconv.FormatBool(item.HasTools),
				strings.Join(item.ToolKinds, "|"),
				strconv.FormatBool(item.HasThinking),
				item.ThinkingSource,
				item.ThinkingLevel,
				formatIntPointer(item.ThinkingBudget),
				item.MediaResolution,
				item.CountTokensSource,
				item.CaptureReason,
				strconv.FormatBool(item.Sampled),
				strconv.FormatBool(item.RawAvailable),
			}
			if includeRaw {
				raw := rawTraceCache[item.ID]
				if raw == nil {
					raw, err = s.GetRequestTraceRawByID(ctx, operatorID, item.ID)
					if err != nil {
						return written, err
					}
					rawTraceCache[item.ID] = raw
				}
				row = append(row, raw.RawRequest, raw.RawResponse)
			}
			if err := csvWriter.Write(row); err != nil {
				return written, infraerrors.InternalServer("OPS_REQUEST_TRACE_EXPORT_FAILED", "Failed to write export row").WithCause(err)
			}
			written++
		}
		return written, nil
	}

	totalWritten, err := writeRows(firstPage.Items)
	if err != nil {
		return totalWritten, err
	}

	for totalWritten < int(firstPage.Total) {
		filterCopy.Page++
		page, pageErr := s.opsRepo.ListRequestTraces(ctx, filterCopy)
		if pageErr != nil {
			return totalWritten, infraerrors.InternalServer("OPS_REQUEST_TRACE_EXPORT_FAILED", "Failed to export request details").WithCause(pageErr)
		}
		if len(page.Items) == 0 {
			break
		}
		n, writeErr := writeRows(page.Items)
		totalWritten += n
		if writeErr != nil {
			return totalWritten, writeErr
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return totalWritten, infraerrors.InternalServer("OPS_REQUEST_TRACE_EXPORT_FAILED", "Failed to flush export").WithCause(err)
	}

	_ = s.insertRequestTraceAudit(ctx, nil, operatorID, OpsRequestTraceAuditActionExportCSV, map[string]any{
		"include_raw": includeRaw,
		"row_count":   totalWritten,
	})
	return totalWritten, nil
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

func (s *OpsService) requireRequestTraceEnabled(ctx context.Context) error {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return err
	}
	if !s.getOpsRequestTraceRuntimeConfig(ctx).Enabled {
		return ErrOpsRequestTracesDisabled
	}
	return nil
}

func (s *OpsService) canAccessRequestTraceRaw(ctx context.Context, operatorID int64) bool {
	if operatorID <= 0 {
		return false
	}
	runtimeCfg := s.getOpsRequestTraceRuntimeConfig(ctx)
	if strings.TrimSpace(runtimeCfg.EncryptionKey) == "" {
		return false
	}
	if hasOpsRequestTraceAdminRawAccess(ctx) {
		return true
	}
	_, ok := runtimeCfg.RawAccessUserIDs[operatorID]
	return ok
}

func (s *OpsService) getOpsRequestTraceRuntimeConfig(ctx context.Context) opsRequestTraceRuntimeConfig {
	cfg := opsRequestTraceRuntimeConfig{
		Enabled:            true,
		EncryptionKey:      "",
		RawAccessUserIDs:   map[int64]struct{}{},
		RetentionDays:      30,
		SuccessSampleRate:  0.1,
		ForceCaptureSlowMs: 3000,
		RawExportMaxRows:   10000,
	}

	if s != nil && s.cfg != nil {
		cfg.Enabled = s.cfg.Ops.RequestDetails.Enabled
		cfg.EncryptionKey = strings.TrimSpace(s.cfg.Ops.RequestDetails.EncryptionKey)
		if s.cfg.Ops.RequestDetails.RetentionDays > 0 {
			cfg.RetentionDays = s.cfg.Ops.RequestDetails.RetentionDays
		}
		if s.cfg.Ops.RequestDetails.SuccessSampleRate >= 0 {
			cfg.SuccessSampleRate = s.cfg.Ops.RequestDetails.SuccessSampleRate
		}
		if s.cfg.Ops.RequestDetails.ForceCaptureSlowMs > 0 {
			cfg.ForceCaptureSlowMs = int64(s.cfg.Ops.RequestDetails.ForceCaptureSlowMs)
		}
		if s.cfg.Ops.RequestDetails.RawExportMaxRows > 0 {
			cfg.RawExportMaxRows = s.cfg.Ops.RequestDetails.RawExportMaxRows
		}
		for _, userID := range s.cfg.Ops.RequestDetails.RawAccessUserIDs {
			if userID > 0 {
				cfg.RawAccessUserIDs[userID] = struct{}{}
			}
		}
	}

	if s != nil {
		if advanced, err := s.GetOpsAdvancedSettings(ctx); err == nil && advanced != nil {
			cfg.Enabled = cfg.Enabled && advanced.RequestDetailsEnabled
			if advanced.RequestDetailRetentionDays > 0 {
				cfg.RetentionDays = advanced.RequestDetailRetentionDays
			}
			if advanced.SuccessSampleRate >= 0 {
				cfg.SuccessSampleRate = advanced.SuccessSampleRate
			}
			if advanced.ForceCaptureSlowMs > 0 {
				cfg.ForceCaptureSlowMs = int64(advanced.ForceCaptureSlowMs)
			}
			if advanced.RawExportMaxRows > 0 {
				cfg.RawExportMaxRows = advanced.RawExportMaxRows
			}
		}
	}

	if cfg.SuccessSampleRate < 0 {
		cfg.SuccessSampleRate = 0
	}
	if cfg.SuccessSampleRate > 1 {
		cfg.SuccessSampleRate = 1
	}
	if cfg.ForceCaptureSlowMs <= 0 {
		cfg.ForceCaptureSlowMs = 3000
	}
	if cfg.RawExportMaxRows <= 0 {
		cfg.RawExportMaxRows = 10000
	}
	return cfg
}

func evaluateOpsRequestTraceDecision(runtimeCfg opsRequestTraceRuntimeConfig, input *OpsRecordRequestTraceInput) RequestCaptureDecision {
	decision := RequestCaptureDecision{
		Capture:    false,
		Reason:     "",
		Sampled:    false,
		RawEnabled: strings.TrimSpace(runtimeCfg.EncryptionKey) != "",
	}
	if input == nil || !runtimeCfg.Enabled {
		return decision
	}

	normalize := input.Trace.Normalize
	switch {
	case input.StatusCode >= 400:
		decision.Capture = true
		decision.Reason = "error"
	case input.DurationMs >= runtimeCfg.ForceCaptureSlowMs:
		decision.Capture = true
		decision.Reason = "slow"
	case normalize.Stream:
		decision.Capture = true
		decision.Reason = "stream"
	case normalize.HasTools:
		decision.Capture = true
		decision.Reason = "tools"
	case normalize.HasThinking:
		decision.Capture = true
		decision.Reason = "thinking"
	case isGoogleProtocolTrace(normalize):
		decision.Capture = true
		decision.Reason = "google_gateway"
	case normalize.ProtocolIn != "" && normalize.ProtocolOut != "" && normalize.ProtocolIn != normalize.ProtocolOut:
		decision.Capture = true
		decision.Reason = "protocol_transform"
	case shouldSampleRequestTrace(runtimeCfg.SuccessSampleRate, input):
		decision.Capture = true
		decision.Sampled = true
		decision.Reason = "success_sampled"
	}
	return decision
}

func normalizeOpsRequestTraceStatus(status string, statusCode int) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "" {
		return status
	}
	if statusCode >= 400 {
		return "error"
	}
	return "success"
}

func isGoogleProtocolTrace(normalize ProtocolNormalizeResult) bool {
	for _, value := range []string{normalize.Platform, normalize.ProtocolIn, normalize.ProtocolOut, normalize.Channel} {
		value = strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(value, "gemini") || strings.Contains(value, "vertex") || strings.Contains(value, "google") {
			return true
		}
	}
	return false
}

func shouldSampleRequestTrace(rate float64, input *OpsRecordRequestTraceInput) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	key := strings.TrimSpace(firstNonEmptyString(input.RequestID, input.ClientRequestID, input.UpstreamRequestID))
	if key == "" {
		key = strconv.FormatInt(input.CreatedAt.UnixNano(), 10)
	}
	sum := sha256.Sum256([]byte(key))
	value := binary.BigEndian.Uint64(sum[:8])
	threshold := uint64(rate * float64(^uint64(0)))
	return value <= threshold
}

func buildEncryptedTracePayload(key string, payload []byte, maxBytes int) ([]byte, *int, bool, error) {
	if len(payload) == 0 {
		return nil, nil, false, nil
	}
	trimmed, size, truncated := trimTraceRawPayload(payload, maxBytes)
	ciphertext, err := encryptOpsRequestTracePayload(key, trimmed)
	return ciphertext, size, truncated, err
}

func trimTraceRawPayload(payload []byte, maxBytes int) ([]byte, *int, bool) {
	if len(payload) == 0 {
		return nil, nil, false
	}
	size := len(payload)
	sizePtr := &size
	if maxBytes > 0 && len(payload) > maxBytes {
		copied := append([]byte(nil), payload[:maxBytes]...)
		return copied, sizePtr, true
	}
	return append([]byte(nil), payload...), sizePtr, false
}

func sanitizeTracePayloadForStorage(payload []byte, maxBytes int, contentType string) *string {
	if len(payload) == 0 {
		return nil
	}
	return BuildOpsTracePayloadEnvelopeJSONFromBytes(
		payload,
		maxBytes,
		OpsTracePayloadStateCaptured,
		"legacy_capture_fallback",
		contentType,
	)
}

func buildOpsRequestTraceSearchText(input *OpsInsertRequestTraceInput) string {
	if input == nil {
		return ""
	}
	parts := []string{
		input.RequestID,
		input.ClientRequestID,
		input.UpstreamRequestID,
		input.Platform,
		input.ProtocolIn,
		input.ProtocolOut,
		input.Channel,
		input.RoutePath,
		input.RequestType,
		input.RequestedModel,
		input.UpstreamModel,
		input.ActualUpstreamModel,
		input.Status,
		input.FinishReason,
		input.PromptBlockReason,
		input.CaptureReason,
		input.ThinkingSource,
		input.ThinkingLevel,
		input.MediaResolution,
		input.CountTokensSource,
		strings.Join(input.ToolKinds, " "),
	}
	value := strings.Join(dedupeNonEmptyStrings(parts), " ")
	if len(value) > opsRequestTraceSearchTextLimit {
		value = truncateString(value, opsRequestTraceSearchTextLimit)
	}
	return value
}

func dedupeNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func normalizeRequestTraceExportWindow(filter *OpsRequestTraceFilter) (time.Duration, error) {
	_, _, startTime, endTime := filter.Normalize()
	if startTime.After(endTime) {
		return 0, infraerrors.BadRequest("OPS_REQUEST_TRACE_EXPORT_WINDOW_INVALID", "invalid export time range")
	}
	return endTime.Sub(startTime), nil
}

func (s *OpsService) insertRequestTraceAudit(ctx context.Context, traceID *int64, operatorID int64, action OpsRequestTraceAuditAction, meta map[string]any) error {
	if s == nil || s.opsRepo == nil || operatorID <= 0 {
		return nil
	}
	var metaJSON *string
	if len(meta) > 0 {
		if raw, err := json.Marshal(meta); err == nil {
			value := string(raw)
			metaJSON = &value
		}
	}
	return s.opsRepo.InsertRequestTraceAudit(ctx, &OpsInsertRequestTraceAuditInput{
		TraceID:    traceID,
		OperatorID: operatorID,
		Action:     action,
		MetaJSON:   metaJSON,
		CreatedAt:  time.Now().UTC(),
	})
}

func formatIntPointer(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func formatInt64Pointer(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}
