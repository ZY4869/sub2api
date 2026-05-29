package service

import (
	"context"
	"encoding/csv"
	"io"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

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
	if err := csvWriter.Write(buildOpsRequestTraceCSVHeaders(includeRaw)); err != nil {
		return 0, infraerrors.InternalServer("OPS_REQUEST_TRACE_EXPORT_FAILED", "Failed to write export header").WithCause(err)
	}

	rawTraceCache := make(map[int64]*OpsRequestTraceRawDetail, 16)
	writeRows := func(items []*OpsRequestTraceListItem) (int, error) {
		written := 0
		for _, item := range items {
			if item == nil {
				continue
			}
			row := buildOpsRequestTraceCSVRow(item)
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

func normalizeRequestTraceExportWindow(filter *OpsRequestTraceFilter) (time.Duration, error) {
	_, _, startTime, endTime := filter.Normalize()
	if startTime.After(endTime) {
		return 0, infraerrors.BadRequest("OPS_REQUEST_TRACE_EXPORT_WINDOW_INVALID", "invalid export time range")
	}
	return endTime.Sub(startTime), nil
}
