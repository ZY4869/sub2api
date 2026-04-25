package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelMonitorTemplateRepository struct {
	db *sql.DB
}

func NewChannelMonitorTemplateRepository(db *sql.DB) service.ChannelMonitorTemplateRepository {
	return &channelMonitorTemplateRepository{db: db}
}

func (r *channelMonitorTemplateRepository) Create(ctx context.Context, tpl *service.ChannelMonitorRequestTemplate) (*service.ChannelMonitorRequestTemplate, error) {
	if tpl == nil {
		return nil, errors.New("nil template")
	}
	headers, err := json.Marshal(orEmptyStringMap(tpl.ExtraHeaders))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_EXTRA_HEADERS_INVALID", "invalid extra headers")
	}
	body, err := json.Marshal(orEmptyAnyMap(tpl.BodyOverride))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_BODY_OVERRIDE_INVALID", "invalid body override")
	}

	row := r.db.QueryRowContext(ctx, `
INSERT INTO channel_monitor_request_templates (
	name,
	provider,
	description,
	extra_headers,
	body_override_mode,
	body_override,
	created_at,
	updated_at
)
VALUES ($1, $2, NULLIF($3, ''), $4::jsonb, $5, $6::jsonb, NOW(), NOW())
RETURNING id, created_at, updated_at
`, tpl.Name, tpl.Provider, optionalStringValue(tpl.Description), string(headers), tpl.BodyOverrideMode, string(body))
	if err := row.Scan(&tpl.ID, &tpl.CreatedAt, &tpl.UpdatedAt); err != nil {
		return nil, translateChannelMonitorTemplateSQLError(err)
	}
	return r.GetByID(ctx, tpl.ID)
}

func (r *channelMonitorTemplateRepository) GetByID(ctx context.Context, id int64) (*service.ChannelMonitorRequestTemplate, error) {
	row := r.db.QueryRowContext(ctx, templateSelectSQL()+` WHERE id = $1`, id)
	out, err := scanChannelMonitorTemplateRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrChannelMonitorTemplateNotFound
		}
		return nil, err
	}
	return out, nil
}

func (r *channelMonitorTemplateRepository) ListAll(ctx context.Context) ([]*service.ChannelMonitorRequestTemplate, error) {
	rows, err := r.db.QueryContext(ctx, templateSelectSQL()+` ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorTemplateRows(rows)
}

func (r *channelMonitorTemplateRepository) Update(ctx context.Context, tpl *service.ChannelMonitorRequestTemplate) (*service.ChannelMonitorRequestTemplate, error) {
	if tpl == nil {
		return nil, errors.New("nil template")
	}
	headers, err := json.Marshal(orEmptyStringMap(tpl.ExtraHeaders))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_EXTRA_HEADERS_INVALID", "invalid extra headers")
	}
	body, err := json.Marshal(orEmptyAnyMap(tpl.BodyOverride))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_BODY_OVERRIDE_INVALID", "invalid body override")
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE channel_monitor_request_templates
SET
	name = $2,
	provider = $3,
	description = NULLIF($4, ''),
	extra_headers = $5::jsonb,
	body_override_mode = $6,
	body_override = $7::jsonb,
	updated_at = NOW()
WHERE id = $1
`, tpl.ID, tpl.Name, tpl.Provider, optionalStringValue(tpl.Description), string(headers), tpl.BodyOverrideMode, string(body))
	if err != nil {
		return nil, translateChannelMonitorTemplateSQLError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrChannelMonitorTemplateNotFound
	}
	return r.GetByID(ctx, tpl.ID)
}

func (r *channelMonitorTemplateRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM channel_monitor_request_templates WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrChannelMonitorTemplateNotFound
	}
	return nil
}

func (r *channelMonitorTemplateRepository) ListAssociatedMonitors(ctx context.Context, templateID int64) ([]*service.ChannelMonitor, error) {
	rows, err := r.db.QueryContext(ctx, monitorSelectSQL()+` WHERE template_id = $1 ORDER BY created_at DESC, id DESC`, templateID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanChannelMonitorRows(rows)
}

func templateSelectSQL() string {
	return `
SELECT
	id,
	name,
	provider,
	description,
	COALESCE(extra_headers, '{}'::jsonb),
	COALESCE(body_override_mode, 'off'),
	COALESCE(body_override, '{}'::jsonb),
	created_at,
	updated_at
FROM channel_monitor_request_templates
`
}

type templateRowScanner interface {
	Scan(dest ...any) error
}

func scanChannelMonitorTemplateRow(row templateRowScanner) (*service.ChannelMonitorRequestTemplate, error) {
	var (
		desc            sql.NullString
		extraHeadersRaw []byte
		bodyOverrideRaw []byte
	)
	t := &service.ChannelMonitorRequestTemplate{}
	if err := row.Scan(
		&t.ID,
		&t.Name,
		&t.Provider,
		&desc,
		&extraHeadersRaw,
		&t.BodyOverrideMode,
		&bodyOverrideRaw,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if desc.Valid {
		v := desc.String
		t.Description = &v
	}
	if len(extraHeadersRaw) > 0 {
		var parsed map[string]string
		if err := json.Unmarshal(extraHeadersRaw, &parsed); err == nil {
			t.ExtraHeaders = parsed
		}
	}
	if t.ExtraHeaders == nil {
		t.ExtraHeaders = map[string]string{}
	}
	if len(bodyOverrideRaw) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(bodyOverrideRaw, &parsed); err == nil {
			t.BodyOverride = parsed
		}
	}
	if t.BodyOverride == nil {
		t.BodyOverride = map[string]any{}
	}
	return t, nil
}

func scanChannelMonitorTemplateRows(rows *sql.Rows) ([]*service.ChannelMonitorRequestTemplate, error) {
	var out []*service.ChannelMonitorRequestTemplate
	for rows.Next() {
		t, err := scanChannelMonitorTemplateRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func translateChannelMonitorTemplateSQLError(err error) error {
	if err == nil {
		return nil
	}
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Code {
		case "23505":
			return service.ErrChannelMonitorTemplateAlreadyExists
		case "23514":
			if pqErr.Constraint == "channel_monitor_request_templates_provider_check" {
				return service.ErrChannelMonitorInvalidProvider
			}
			if pqErr.Constraint == "channel_monitor_request_templates_override_mode_check" {
				return service.ErrChannelMonitorInvalidOverrideMode
			}
		}
	}
	return fmt.Errorf("channel monitor template sql error: %w", err)
}

func optionalStringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
