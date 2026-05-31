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

type contentModerationAuditRepository struct {
	sql *sql.DB
}

func NewContentModerationAuditRepository(sqlDB *sql.DB) service.ContentModerationAuditRepository {
	return &contentModerationAuditRepository{sql: sqlDB}
}

func (r *contentModerationAuditRepository) CreateContentModerationAudit(ctx context.Context, audit *service.ContentModerationAudit) error {
	if r == nil || r.sql == nil || audit == nil {
		return errors.New("content moderation audit repository not configured")
	}
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = time.Now().UTC()
	}
	query := `
INSERT INTO content_moderation_audits (
  request_id,
  client_request_id,
  user_id,
  api_key_id,
  provider,
  model,
  source_endpoint,
  content_hash,
  content_summary,
  categories,
  hit,
  dedupe_hit,
  error_reason,
  latency_ms,
  created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
RETURNING id, created_at`
	categories, err := json.Marshal(normalizeAuditCategoriesForStorage(audit.Categories))
	if err != nil {
		return err
	}
	return r.sql.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(audit.RequestID),
		strings.TrimSpace(audit.ClientRequestID),
		nullInt64OrNil(audit.UserID),
		nullInt64OrNil(audit.APIKeyID),
		strings.TrimSpace(audit.Provider),
		strings.TrimSpace(audit.Model),
		strings.TrimSpace(audit.SourceEndpoint),
		strings.TrimSpace(audit.ContentHash),
		strings.TrimSpace(audit.ContentSummary),
		categories,
		audit.Hit,
		audit.DedupeHit,
		strings.TrimSpace(audit.ErrorReason),
		audit.LatencyMs,
		audit.CreatedAt,
	).Scan(&audit.ID, &audit.CreatedAt)
}

func (r *contentModerationAuditRepository) FindRecentContentModerationAuditByHash(ctx context.Context, contentHash string, since time.Time) (*service.ContentModerationAudit, error) {
	return r.getOne(
		ctx,
		`SELECT id, request_id, client_request_id, user_id, api_key_id, provider, model, source_endpoint, content_hash, content_summary, categories, hit, dedupe_hit, error_reason, latency_ms, created_at
		 FROM content_moderation_audits
		 WHERE content_hash = $1 AND created_at >= $2
		 ORDER BY created_at DESC, id DESC
		 LIMIT 1`,
		strings.TrimSpace(contentHash),
		since,
	)
}

func (r *contentModerationAuditRepository) ListContentModerationAudits(ctx context.Context, filter *service.ContentModerationAuditFilter) (*service.ContentModerationAuditList, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("content moderation audit repository not configured")
	}
	page, pageSize := filter.Normalize()
	where, args := buildContentModerationAuditWhere(filter)

	countSQL := "SELECT COUNT(*) FROM content_moderation_audits a " + where
	var total int64
	if err := r.sql.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	query := `
SELECT
  id,
  request_id,
  client_request_id,
  user_id,
  api_key_id,
  provider,
  model,
  source_endpoint,
  content_hash,
  content_summary,
  categories,
  hit,
  dedupe_hit,
  error_reason,
  latency_ms,
  created_at
FROM content_moderation_audits a
` + where + `
ORDER BY created_at DESC, id DESC
LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, pageSize, offset)
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.ContentModerationAudit, 0)
	for rows.Next() {
		item, scanErr := scanContentModerationAudit(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &service.ContentModerationAuditList{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (r *contentModerationAuditRepository) GetContentModerationAuditByID(ctx context.Context, id int64) (*service.ContentModerationAudit, error) {
	return r.getOne(
		ctx,
		`SELECT id, request_id, client_request_id, user_id, api_key_id, provider, model, source_endpoint, content_hash, content_summary, categories, hit, dedupe_hit, error_reason, latency_ms, created_at
		 FROM content_moderation_audits
		 WHERE id = $1`,
		id,
	)
}

func (r *contentModerationAuditRepository) getOne(ctx context.Context, query string, args ...any) (*service.ContentModerationAudit, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("content moderation audit repository not configured")
	}
	row := r.sql.QueryRowContext(ctx, query, args...)
	item, err := scanContentModerationAudit(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrContentModerationAuditNotFound
		}
		return nil, err
	}
	return item, nil
}

type contentModerationScanner interface {
	Scan(dest ...any) error
}

func scanContentModerationAudit(scanner contentModerationScanner) (*service.ContentModerationAudit, error) {
	var userID sql.NullInt64
	var apiKeyID sql.NullInt64
	var categoriesRaw []byte
	item := &service.ContentModerationAudit{}
	if err := scanner.Scan(
		&item.ID,
		&item.RequestID,
		&item.ClientRequestID,
		&userID,
		&apiKeyID,
		&item.Provider,
		&item.Model,
		&item.SourceEndpoint,
		&item.ContentHash,
		&item.ContentSummary,
		&categoriesRaw,
		&item.Hit,
		&item.DedupeHit,
		&item.ErrorReason,
		&item.LatencyMs,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	if userID.Valid {
		item.UserID = &userID.Int64
	}
	if apiKeyID.Valid {
		item.APIKeyID = &apiKeyID.Int64
	}
	item.Categories = parseAuditCategories(categoriesRaw)
	return item, nil
}

func buildContentModerationAuditWhere(filter *service.ContentModerationAuditFilter) (string, []any) {
	if filter == nil {
		return "", nil
	}
	clauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	add := func(sqlExpr string, value any) {
		args = append(args, value)
		clauses = append(clauses, sqlExpr+" $"+fmt.Sprintf("%d", len(args)))
	}
	addStringLike := func(column string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		add(column+" ILIKE", "%"+value+"%")
	}
	addStringEqual := func(column string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		add(column+" =", value)
	}

	addStringLike("COALESCE(a.request_id,'')", filter.RequestID)
	addStringLike("COALESCE(a.client_request_id,'')", filter.ClientRequestID)
	addStringEqual("COALESCE(a.provider,'')", filter.Provider)
	addStringEqual("COALESCE(a.model,'')", filter.Model)
	addStringEqual("COALESCE(a.source_endpoint,'')", filter.SourceEndpoint)
	addStringEqual("COALESCE(a.content_hash,'')", filter.ContentHash)
	if filter.UserID != nil && *filter.UserID > 0 {
		add("a.user_id =", *filter.UserID)
	}
	if filter.Hit != nil {
		add("a.hit =", *filter.Hit)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func nullInt64OrNil(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func normalizeAuditCategoriesForStorage(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		category := strings.TrimSpace(value)
		if category == "" {
			continue
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		out = append(out, category)
	}
	return out
}

func parseAuditCategories(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return normalizeAuditCategoriesForStorage(values)
}
