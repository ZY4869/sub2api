package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) DeleteRequestTraces(ctx context.Context, filter *service.OpsRequestTraceFilter) (service.OpsRequestTraceDeleteCounts, error) {
	if r == nil || r.db == nil {
		return service.OpsRequestTraceDeleteCounts{}, fmt.Errorf("nil ops repository")
	}
	schema, err := r.getOpsRequestTraceSchema(ctx)
	if err != nil {
		return service.OpsRequestTraceDeleteCounts{}, err
	}

	filterCopy := &service.OpsRequestTraceFilter{}
	if filter != nil {
		*filterCopy = *filter
	}
	filterCopy.Page = 0
	filterCopy.PageSize = 0
	filterCopy.Sort = ""
	filterCopy.Limit = 0
	if !hasRequestTraceDeleteCondition(filterCopy) {
		return service.OpsRequestTraceDeleteCounts{}, fmt.Errorf("cleanup requires at least one filter condition")
	}

	where, args := buildOpsRequestTracesWhereWithSchema(filterCopy, schema)
	batchSize := 5000
	query := `
WITH matched AS (
  SELECT t.id
  FROM ops_request_traces t
` + where + `
  ORDER BY t.id
  LIMIT $` + itoa(len(args)+1) + `
),
deleted_audits AS (
  DELETE FROM ops_request_trace_audits
  WHERE trace_id IN (SELECT id FROM matched)
  RETURNING 1
),
deleted_traces AS (
  DELETE FROM ops_request_traces
  WHERE id IN (SELECT id FROM matched)
  RETURNING 1
)
SELECT
  COALESCE((SELECT COUNT(*) FROM deleted_traces), 0),
  COALESCE((SELECT COUNT(*) FROM deleted_audits), 0)
`

	total := service.OpsRequestTraceDeleteCounts{}
	for {
		var tracesDeleted int64
		var auditsDeleted int64
		if err := r.db.QueryRowContext(ctx, query, append(args, batchSize)...).Scan(&tracesDeleted, &auditsDeleted); err != nil {
			return total, err
		}
		total.DeletedTraces += tracesDeleted
		total.DeletedAudits += auditsDeleted
		if tracesDeleted == 0 {
			break
		}
	}
	return total, nil
}

func (r *opsRepository) DeleteExpiredRequestTraces(ctx context.Context, cutoff time.Time, batchSize int) (service.OpsRequestTraceDeleteCounts, error) {
	if r == nil || r.db == nil {
		return service.OpsRequestTraceDeleteCounts{}, fmt.Errorf("nil ops repository")
	}
	if batchSize <= 0 {
		batchSize = 5000
	}

	query := `
WITH matched AS (
  SELECT id
  FROM ops_request_traces
  WHERE created_at < $1
  ORDER BY id
  LIMIT $2
),
orphan_audits AS (
  SELECT id
  FROM ops_request_trace_audits
  WHERE trace_id IS NULL
    AND created_at < $1
  ORDER BY id
  LIMIT $2
),
deleted_audits AS (
  DELETE FROM ops_request_trace_audits
  WHERE trace_id IN (SELECT id FROM matched)
     OR id IN (SELECT id FROM orphan_audits)
  RETURNING 1
),
deleted_traces AS (
  DELETE FROM ops_request_traces
  WHERE id IN (SELECT id FROM matched)
  RETURNING 1
)
SELECT
  COALESCE((SELECT COUNT(*) FROM deleted_traces), 0),
  COALESCE((SELECT COUNT(*) FROM deleted_audits), 0)
`

	total := service.OpsRequestTraceDeleteCounts{}
	for {
		var tracesDeleted int64
		var auditsDeleted int64
		if err := r.db.QueryRowContext(ctx, query, cutoff.UTC(), batchSize).Scan(&tracesDeleted, &auditsDeleted); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "does not exist") && strings.Contains(strings.ToLower(err.Error()), "relation") {
				return total, nil
			}
			return total, err
		}
		total.DeletedTraces += tracesDeleted
		total.DeletedAudits += auditsDeleted
		if tracesDeleted == 0 && auditsDeleted == 0 {
			break
		}
	}
	return total, nil
}
