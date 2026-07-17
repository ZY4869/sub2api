package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type auditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) service.AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) CreateAuditLog(ctx context.Context, log *service.AuditLog) error {
	if r == nil || r.db == nil || log == nil {
		return nil
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	metadata, err := json.Marshal(log.Metadata)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO audit_logs (
  actor_user_id, actor_role, action, target_type, target_id, status,
  request_id, client_ip, user_agent, metadata, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id, created_at`,
		auditNullInt64(log.ActorUserID),
		log.ActorRole,
		log.Action,
		log.TargetType,
		log.TargetID,
		log.Status,
		log.RequestID,
		log.ClientIP,
		log.UserAgent,
		metadata,
		log.CreatedAt,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *auditLogRepository) ListAuditLogs(ctx context.Context, filter *service.AuditLogFilter) (*service.AuditLogList, error) {
	if r == nil || r.db == nil {
		return &service.AuditLogList{Items: []*service.AuditLog{}, Page: 1, PageSize: 20}, nil
	}
	filter = normalizeAuditRepoFilter(filter)
	where, args := buildAuditLogWhere(filter)

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_logs a "+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := `
SELECT id, actor_user_id, actor_role, action, target_type, target_id, status,
       request_id, client_ip, user_agent, metadata, created_at
FROM audit_logs a
` + where + `
ORDER BY created_at DESC, id DESC
LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.AuditLog, 0)
	for rows.Next() {
		item, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &service.AuditLogList{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (r *auditLogRepository) DeleteAuditLogsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	if r == nil || r.db == nil || cutoff.IsZero() {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM audit_logs WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type auditLogScanner interface {
	Scan(dest ...any) error
}

func scanAuditLog(scanner auditLogScanner) (*service.AuditLog, error) {
	var actorID sql.NullInt64
	var metadataRaw []byte
	item := &service.AuditLog{}
	if err := scanner.Scan(
		&item.ID,
		&actorID,
		&item.ActorRole,
		&item.Action,
		&item.TargetType,
		&item.TargetID,
		&item.Status,
		&item.RequestID,
		&item.ClientIP,
		&item.UserAgent,
		&metadataRaw,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAuditLogNotFound
		}
		return nil, err
	}
	if actorID.Valid {
		item.ActorUserID = &actorID.Int64
	}
	item.Metadata = parseAuditMetadata(metadataRaw)
	return item, nil
}

func buildAuditLogWhere(filter *service.AuditLogFilter) (string, []any) {
	clauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	add := func(expr string, value any) {
		args = append(args, value)
		clauses = append(clauses, expr+" $"+fmt.Sprintf("%d", len(args)))
	}
	if filter.ActorUserID != nil && *filter.ActorUserID > 0 {
		add("a.actor_user_id =", *filter.ActorUserID)
	}
	for _, item := range []struct{ column, value string }{
		{"a.action =", filter.Action},
		{"a.target_type =", filter.TargetType},
		{"a.target_id =", filter.TargetID},
		{"a.status =", filter.Status},
		{"a.request_id =", filter.RequestID},
	} {
		if value := strings.TrimSpace(item.value); value != "" {
			add(item.column, value)
		}
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func normalizeAuditRepoFilter(filter *service.AuditLogFilter) *service.AuditLogFilter {
	if filter == nil {
		return &service.AuditLogFilter{Page: 1, PageSize: 20}
	}
	out := *filter
	if out.Page <= 0 {
		out.Page = 1
	}
	if out.PageSize <= 0 {
		out.PageSize = 20
	}
	if out.PageSize > 200 {
		out.PageSize = 200
	}
	return &out
}

func parseAuditMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return map[string]any{}
	}
	return metadata
}

func auditNullInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
