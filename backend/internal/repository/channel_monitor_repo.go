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

type channelMonitorSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func NewChannelMonitorRepository(db *sql.DB) service.ChannelMonitorRepository {
	return &channelMonitorRepository{db: db}
}

func (r *channelMonitorRepository) Create(ctx context.Context, monitor *service.ChannelMonitor) (*service.ChannelMonitor, error) {
	return r.createWithExecutor(ctx, r.db, monitor)
}

func (r *channelMonitorRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, monitor *service.ChannelMonitor) (*service.ChannelMonitor, error) {
	return r.createWithExecutor(ctx, tx, monitor)
}

func (r *channelMonitorRepository) createWithExecutor(ctx context.Context, exec channelMonitorSQLExecutor, monitor *service.ChannelMonitor) (*service.ChannelMonitor, error) {
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
	modelProtocols, err := json.Marshal(orEmptyStringMap(monitor.ModelSourceProtocols))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_MODEL_SOURCE_PROTOCOLS_INVALID", "invalid model source protocols")
	}

	var apiKeyEncrypted sql.NullString
	if monitor.APIKeyEncrypted != nil && *monitor.APIKeyEncrypted != "" {
		apiKeyEncrypted = sql.NullString{String: *monitor.APIKeyEncrypted, Valid: true}
	}

	var templateID sql.NullInt64
	if monitor.TemplateID != nil && *monitor.TemplateID > 0 {
		templateID = sql.NullInt64{Int64: *monitor.TemplateID, Valid: true}
	}

	row := exec.QueryRowContext(ctx, `
INSERT INTO channel_monitors (
	name,
	provider,
	endpoint,
	probe_mode,
	request_protocol,
	api_key_encrypted,
	interval_seconds,
	jitter_seconds,
	enabled,
	account_ids,
	primary_model_id,
	additional_model_ids,
	model_source_protocols,
	model_probe_strategy,
	test_prompt_template,
	template_id,
	extra_headers,
	body_override_mode,
	body_override,
	openai_api_mode,
	last_run_at,
	next_run_at,
	created_at,
	updated_at
)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10, $11, $12, $13::jsonb, $14, $15, $16, $17::jsonb, $18, $19::jsonb, $20, $21, $22, NOW(), NOW())
RETURNING
	id,
	created_at,
	updated_at
`,
		monitor.Name,
		monitor.Provider,
		monitor.Endpoint,
		monitor.ProbeMode,
		monitor.RequestProtocol,
		apiKeyEncrypted.String,
		monitor.IntervalSeconds,
		monitor.JitterSeconds,
		monitor.Enabled,
		pq.Array(orEmptyInt64Slice(monitor.AccountIDs)),
		monitor.PrimaryModelID,
		pq.Array(monitor.AdditionalModelIDs),
		string(modelProtocols),
		monitor.ModelProbeStrategy,
		monitor.TestPromptTemplate,
		nullInt64Ptr(templateID),
		string(extraHeaders),
		monitor.BodyOverrideMode,
		string(bodyOverride),
		monitor.OpenAIAPIMode,
		monitor.LastRunAt,
		monitor.NextRunAt,
	)
	if err := row.Scan(&monitor.ID, &monitor.CreatedAt, &monitor.UpdatedAt); err != nil {
		return nil, translateChannelMonitorSQLError(err)
	}
	return r.getByIDWithExecutor(ctx, exec, monitor.ID)
}

func (r *channelMonitorRepository) GetByID(ctx context.Context, id int64) (*service.ChannelMonitor, error) {
	return r.getByIDWithExecutor(ctx, r.db, id)
}

func (r *channelMonitorRepository) getByIDWithExecutor(ctx context.Context, exec channelMonitorSQLExecutor, id int64) (*service.ChannelMonitor, error) {
	row := exec.QueryRowContext(ctx, monitorSelectSQL()+` WHERE id = $1`, id)
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
	next_run_at = $1 + (
		(
			m.interval_seconds
			+ CASE
				WHEN COALESCE(m.jitter_seconds, 0) <= 0 THEN 0
				ELSE (FLOOR(random() * (COALESCE(m.jitter_seconds, 0) * 2 + 1))::int - COALESCE(m.jitter_seconds, 0))
			  END
		) * INTERVAL '1 second'
	),
	updated_at = NOW()
FROM due
WHERE m.id = due.id
RETURNING
	m.id,
	m.name,
	m.provider,
	m.endpoint,
	COALESCE(m.probe_mode, 'direct'),
	COALESCE(m.request_protocol, 'openai'),
	m.api_key_encrypted,
	m.interval_seconds,
	COALESCE(m.jitter_seconds, 0),
	m.enabled,
	COALESCE(m.account_ids, ARRAY[]::bigint[]),
	m.primary_model_id,
	m.additional_model_ids,
	COALESCE(m.model_source_protocols, '{}'::jsonb),
	COALESCE(m.model_probe_strategy, 'all_selected'),
	COALESCE(m.test_prompt_template, ''),
	m.template_id,
	m.extra_headers,
	m.body_override_mode,
	m.body_override,
	COALESCE(m.openai_api_mode, 'chat_completions'),
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
	modelProtocols, err := json.Marshal(orEmptyStringMap(monitor.ModelSourceProtocols))
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_MODEL_SOURCE_PROTOCOLS_INVALID", "invalid model source protocols")
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
	probe_mode = $5,
	request_protocol = $6,
	api_key_encrypted = NULLIF($7, ''),
	interval_seconds = $8,
	jitter_seconds = $9,
	enabled = $10,
	account_ids = $11,
	primary_model_id = $12,
	additional_model_ids = $13,
	model_source_protocols = $14::jsonb,
	model_probe_strategy = $15,
	test_prompt_template = $16,
	template_id = $17,
	extra_headers = $18::jsonb,
	body_override_mode = $19,
	body_override = $20::jsonb,
	openai_api_mode = $21,
	last_run_at = $22,
	next_run_at = $23,
	updated_at = NOW()
WHERE id = $1
`,
		monitor.ID,
		monitor.Name,
		monitor.Provider,
		monitor.Endpoint,
		monitor.ProbeMode,
		monitor.RequestProtocol,
		apiKeyEncrypted.String,
		monitor.IntervalSeconds,
		monitor.JitterSeconds,
		monitor.Enabled,
		pq.Array(orEmptyInt64Slice(monitor.AccountIDs)),
		monitor.PrimaryModelID,
		pq.Array(monitor.AdditionalModelIDs),
		string(modelProtocols),
		monitor.ModelProbeStrategy,
		monitor.TestPromptTemplate,
		nullInt64Ptr(templateID),
		string(extraHeaders),
		monitor.BodyOverrideMode,
		string(bodyOverride),
		monitor.OpenAIAPIMode,
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
	COALESCE(probe_mode, 'direct'),
	COALESCE(request_protocol, 'openai'),
	api_key_encrypted,
	interval_seconds,
	COALESCE(jitter_seconds, 0),
	enabled,
	COALESCE(account_ids, ARRAY[]::bigint[]),
	primary_model_id,
	additional_model_ids,
	COALESCE(model_source_protocols, '{}'::jsonb),
	COALESCE(model_probe_strategy, 'all_selected'),
	COALESCE(test_prompt_template, ''),
	template_id,
	COALESCE(extra_headers, '{}'::jsonb),
	COALESCE(body_override_mode, 'off'),
	COALESCE(body_override, '{}'::jsonb),
	COALESCE(openai_api_mode, 'chat_completions'),
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
		apiKeyEncrypted   sql.NullString
		templateID        sql.NullInt64
		accountIDs        []int64
		additional        []string
		modelProtocolsRaw []byte
		extraHeadersRaw   []byte
		bodyOverrideRaw   []byte
		lastRunAt         sql.NullTime
		nextRunAt         sql.NullTime
	)

	m := &service.ChannelMonitor{}
	if err := row.Scan(
		&m.ID,
		&m.Name,
		&m.Provider,
		&m.Endpoint,
		&m.ProbeMode,
		&m.RequestProtocol,
		&apiKeyEncrypted,
		&m.IntervalSeconds,
		&m.JitterSeconds,
		&m.Enabled,
		pq.Array(&accountIDs),
		&m.PrimaryModelID,
		pq.Array(&additional),
		&modelProtocolsRaw,
		&m.ModelProbeStrategy,
		&m.TestPromptTemplate,
		&templateID,
		&extraHeadersRaw,
		&m.BodyOverrideMode,
		&bodyOverrideRaw,
		&m.OpenAIAPIMode,
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
	m.AccountIDs = accountIDs
	m.AdditionalModelIDs = additional
	if len(modelProtocolsRaw) > 0 {
		var parsed map[string]string
		if err := json.Unmarshal(modelProtocolsRaw, &parsed); err == nil {
			m.ModelSourceProtocols = parsed
		}
	}
	if m.ModelSourceProtocols == nil {
		m.ModelSourceProtocols = map[string]string{}
	}
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
			case "channel_monitors_probe_mode_check":
				return service.ErrChannelMonitorInvalidProbeMode
			case "channel_monitors_request_protocol_check":
				return service.ErrChannelMonitorInvalidProtocol
			case "channel_monitors_model_probe_strategy_check":
				return service.ErrChannelMonitorInvalidStrategy
			case "channel_monitors_body_override_mode_check":
				return service.ErrChannelMonitorInvalidOverrideMode
			case "channel_monitors_openai_api_mode_check":
				return infraerrors.BadRequest("CHANNEL_MONITOR_OPENAI_API_MODE_INVALID", "invalid OpenAI API mode")
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

func orEmptyInt64Slice(v []int64) []int64 {
	if v == nil {
		return []int64{}
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
