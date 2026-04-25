package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelMonitorRepository struct {
	db *sql.DB
}

func NewChannelMonitorRepository(db *sql.DB) service.ChannelMonitorRepository {
	return &channelMonitorRepository{db: db}
}

func (r *channelMonitorRepository) Create(ctx context.Context, monitor *service.ChannelMonitor) (*service.ChannelMonitor, error) {
	if monitor == nil {
		return nil, errors.New("nil monitor")
	}

	extraHeaders, err := json.Marshal(orEmptyStringMap(monitor.ExtraHeaders))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_EXTRA_HEADERS_INVALID", "invalid extra headers")
	}
	bodyOverride, err := json.Marshal(orEmptyAnyMap(monitor.BodyOverride))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_BODY_OVERRIDE_INVALID", "invalid body override")
	}

	var apiKeyEncrypted sql.NullString
	if monitor.APIKeyEncrypted != nil && *monitor.APIKeyEncrypted != "" {
		apiKeyEncrypted = sql.NullString{String: *monitor.APIKeyEncrypted, Valid: true}
	}

	var templateID sql.NullInt64
	if monitor.TemplateID != nil && *monitor.TemplateID > 0 {
		templateID = sql.NullInt64{Int64: *monitor.TemplateID, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO channel_monitors (
	name,
	provider,
	endpoint,
	api_key_encrypted,
	interval_seconds,
	enabled,
	primary_model_id,
	additional_model_ids,
	template_id,
	extra_headers,
	body_override_mode,
	body_override,
	last_run_at,
	next_run_at,
	created_at,
	updated_at
)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10::jsonb, $11, $12::jsonb, $13, $14, NOW(), NOW())
RETURNING
	id,
	created_at,
	updated_at
`,
		monitor.Name,
		monitor.Provider,
		monitor.Endpoint,
		apiKeyEncrypted.String,
		monitor.IntervalSeconds,
		monitor.Enabled,
		monitor.PrimaryModelID,
		pq.Array(monitor.AdditionalModelIDs),
		nullInt64Ptr(templateID),
		string(extraHeaders),
		monitor.BodyOverrideMode,
		string(bodyOverride),
		monitor.LastRunAt,
		monitor.NextRunAt,
	)
	if err := row.Scan(&monitor.ID, &monitor.CreatedAt, &monitor.UpdatedAt); err != nil {
		return nil, translateChannelMonitorSQLError(err)
	}
	return r.GetByID(ctx, monitor.ID)
}

func (r *channelMonitorRepository) GetByID(ctx context.Context, id int64) (*service.ChannelMonitor, error) {
	row := r.db.QueryRowContext(ctx, monitorSelectSQL()+` WHERE id = $1`, id)
	out, err := scanChannelMonitorRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrChannelMonitorNotFound
		}
		return nil, err
	}
	return out, nil
}

func (r *channelMonitorRepository) ListAll(ctx context.Context) ([]*service.ChannelMonitor, error) {
	rows, err := r.db.QueryContext(ctx, monitorSelectSQL()+` ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorRows(rows)
}

func (r *channelMonitorRepository) ListEnabled(ctx context.Context) ([]*service.ChannelMonitor, error) {
	rows, err := r.db.QueryContext(ctx, monitorSelectSQL()+` WHERE enabled = true ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorRows(rows)
}

func (r *channelMonitorRepository) ClaimDue(ctx context.Context, now time.Time, limit int) ([]*service.ChannelMonitor, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
WITH due AS (
	SELECT id
	FROM channel_monitors
	WHERE enabled = true
	  AND (next_run_at IS NULL OR next_run_at <= $1)
	ORDER BY next_run_at ASC NULLS FIRST, id ASC
	FOR UPDATE SKIP LOCKED
	LIMIT $2
)
UPDATE channel_monitors m
SET
	last_run_at = $1,
	next_run_at = $1 + (m.interval_seconds * INTERVAL '1 second'),
	updated_at = NOW()
FROM due
WHERE m.id = due.id
RETURNING
	m.id,
	m.name,
	m.provider,
	m.endpoint,
	m.api_key_encrypted,
	m.interval_seconds,
	m.enabled,
	m.primary_model_id,
	m.additional_model_ids,
	m.template_id,
	m.extra_headers,
	m.body_override_mode,
	m.body_override,
	m.last_run_at,
	m.next_run_at,
	m.created_at,
	m.updated_at
`, now, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorRows(rows)
}

func (r *channelMonitorRepository) Update(ctx context.Context, monitor *service.ChannelMonitor) (*service.ChannelMonitor, error) {
	if monitor == nil {
		return nil, errors.New("nil monitor")
	}

	extraHeaders, err := json.Marshal(orEmptyStringMap(monitor.ExtraHeaders))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_EXTRA_HEADERS_INVALID", "invalid extra headers")
	}
	bodyOverride, err := json.Marshal(orEmptyAnyMap(monitor.BodyOverride))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_BODY_OVERRIDE_INVALID", "invalid body override")
	}

	var apiKeyEncrypted sql.NullString
	if monitor.APIKeyEncrypted != nil && *monitor.APIKeyEncrypted != "" {
		apiKeyEncrypted = sql.NullString{String: *monitor.APIKeyEncrypted, Valid: true}
	}

	var templateID sql.NullInt64
	if monitor.TemplateID != nil && *monitor.TemplateID > 0 {
		templateID = sql.NullInt64{Int64: *monitor.TemplateID, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE channel_monitors
SET
	name = $2,
	provider = $3,
	endpoint = $4,
	api_key_encrypted = NULLIF($5, ''),
	interval_seconds = $6,
	enabled = $7,
	primary_model_id = $8,
	additional_model_ids = $9,
	template_id = $10,
	extra_headers = $11::jsonb,
	body_override_mode = $12,
	body_override = $13::jsonb,
	last_run_at = $14,
	next_run_at = $15,
	updated_at = NOW()
WHERE id = $1
`,
		monitor.ID,
		monitor.Name,
		monitor.Provider,
		monitor.Endpoint,
		apiKeyEncrypted.String,
		monitor.IntervalSeconds,
		monitor.Enabled,
		monitor.PrimaryModelID,
		pq.Array(monitor.AdditionalModelIDs),
		nullInt64Ptr(templateID),
		string(extraHeaders),
		monitor.BodyOverrideMode,
		string(bodyOverride),
		monitor.LastRunAt,
		monitor.NextRunAt,
	)
	if err != nil {
		return nil, translateChannelMonitorSQLError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrChannelMonitorNotFound
	}
	return r.GetByID(ctx, monitor.ID)
}

func (r *channelMonitorRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM channel_monitors WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrChannelMonitorNotFound
	}
	return nil
}

func monitorSelectSQL() string {
	return `
SELECT
	id,
	name,
	provider,
	endpoint,
	api_key_encrypted,
	interval_seconds,
	enabled,
	primary_model_id,
	additional_model_ids,
	template_id,
	COALESCE(extra_headers, '{}'::jsonb),
	COALESCE(body_override_mode, 'off'),
	COALESCE(body_override, '{}'::jsonb),
	last_run_at,
	next_run_at,
	created_at,
	updated_at
FROM channel_monitors
`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanChannelMonitorRow(row rowScanner) (*service.ChannelMonitor, error) {
	var (
		apiKeyEncrypted sql.NullString
		templateID      sql.NullInt64
		additional      []string
		extraHeadersRaw []byte
		bodyOverrideRaw []byte
		lastRunAt       sql.NullTime
		nextRunAt       sql.NullTime
	)

	m := &service.ChannelMonitor{}
	if err := row.Scan(
		&m.ID,
		&m.Name,
		&m.Provider,
		&m.Endpoint,
		&apiKeyEncrypted,
		&m.IntervalSeconds,
		&m.Enabled,
		&m.PrimaryModelID,
		pq.Array(&additional),
		&templateID,
		&extraHeadersRaw,
		&m.BodyOverrideMode,
		&bodyOverrideRaw,
		&lastRunAt,
		&nextRunAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if apiKeyEncrypted.Valid && apiKeyEncrypted.String != "" {
		v := apiKeyEncrypted.String
		m.APIKeyEncrypted = &v
	}
	if templateID.Valid && templateID.Int64 > 0 {
		v := templateID.Int64
		m.TemplateID = &v
	}
	m.AdditionalModelIDs = additional
	if len(extraHeadersRaw) > 0 {
		var parsed map[string]string
		if err := json.Unmarshal(extraHeadersRaw, &parsed); err == nil {
			m.ExtraHeaders = parsed
		}
	}
	if m.ExtraHeaders == nil {
		m.ExtraHeaders = map[string]string{}
	}
	if len(bodyOverrideRaw) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(bodyOverrideRaw, &parsed); err == nil {
			m.BodyOverride = parsed
		}
	}
	if m.BodyOverride == nil {
		m.BodyOverride = map[string]any{}
	}
	if lastRunAt.Valid {
		v := lastRunAt.Time
		m.LastRunAt = &v
	}
	if nextRunAt.Valid {
		v := nextRunAt.Time
		m.NextRunAt = &v
	}
	return m, nil
}

func scanChannelMonitorRows(rows *sql.Rows) ([]*service.ChannelMonitor, error) {
	var out []*service.ChannelMonitor
	for rows.Next() {
		m, err := scanChannelMonitorRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func translateChannelMonitorSQLError(err error) error {
	if err == nil {
		return nil
	}
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Code {
		case "23505":
			// unique_violation
			return service.ErrChannelMonitorAlreadyExists
		case "23514":
			// check_violation
			switch pqErr.Constraint {
			case "channel_monitors_interval_seconds_check":
				return service.ErrChannelMonitorInvalidInterval
			case "channel_monitors_provider_check":
				return service.ErrChannelMonitorInvalidProvider
			case "channel_monitors_body_override_mode_check":
				return service.ErrChannelMonitorInvalidOverrideMode
			}
			return infraerrors.BadRequest("CHANNEL_MONITOR_INVALID_CONSTRAINT", "invalid channel monitor configuration")
		case "23503":
			// foreign_key_violation
			if pqErr.Constraint == "channel_monitors_template_id_fkey" {
				return service.ErrChannelMonitorTemplateNotFound
			}
		}
	}
	return fmt.Errorf("channel monitor sql error: %w", err)
}

func orEmptyStringMap(v map[string]string) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	return v
}

func orEmptyAnyMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func nullInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}
