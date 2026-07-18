package securityaudit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (r *Repository) DeleteEvent(ctx context.Context, id int64) (*DeleteResult, error) {
	if r == nil || r.db == nil || id <= 0 {
		return nil, ErrEventNotFound
	}
	var jobID int64
	err := r.db.QueryRowContext(ctx, `DELETE FROM prompt_audit_events WHERE id=$1 RETURNING job_id`, id).Scan(&jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	return &DeleteResult{Deleted: 1, JobIDs: []int64{jobID}}, nil
}

func (r *Repository) PreviewDelete(ctx context.Context, filter EventFilter) (*DeletePreview, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("prompt audit repository unavailable")
	}
	filter = normalizeEventFilter(filter)
	where, args := buildEventWhere(filter)
	query := `SELECT COUNT(*), COALESCE(MAX(id), 0) FROM prompt_audit_events` + where
	var preview DeletePreview
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&preview.Matched, &preview.SnapshotMaxID); err != nil {
		return nil, err
	}
	preview.FilterHash = filterHash(filter, preview.SnapshotMaxID)
	return &preview, nil
}

func (r *Repository) DeleteEventsByFilter(ctx context.Context, filter EventFilter, snapshotMaxID int64) (*DeleteResult, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("prompt audit repository unavailable")
	}
	filter = normalizeEventFilter(filter)
	where, args := buildEventWhere(filter)
	args = append(args, snapshotMaxID)
	query := `DELETE FROM prompt_audit_events` + where + " AND id <= $" + fmt.Sprint(len(args)) + " RETURNING job_id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanDeleteResult(rows)
}
