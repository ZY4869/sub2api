package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *usageRepairRepository) ListClaudeRequestMetadataCandidates(ctx context.Context, since time.Time, afterID int64, limit int) ([]service.ClaudeUsageRepairCandidate, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			ul.id,
			COALESCE(ul.request_id, ''),
			ul.created_at,
			COALESCE(ul.model, ''),
			COALESCE(ul.requested_model, ''),
			COALESCE(ul.upstream_model, ''),
			NULLIF(COALESCE(ul.inbound_endpoint, ''), ''),
			NULLIF(COALESCE(ul.upstream_endpoint, ''), ''),
			ul.thinking_enabled,
			NULLIF(COALESCE(ul.reasoning_effort, ''), ''),
			COALESCE(trace.route_path, ''),
			COALESCE(trace.upstream_path, ''),
			trace.has_thinking,
			COALESCE(trace.inbound_request::text, ''),
			COALESCE(trace.normalized_request::text, '')
		FROM usage_logs ul
		LEFT JOIN LATERAL (
			SELECT
				t.route_path,
				t.upstream_path,
				t.has_thinking,
				t.inbound_request,
				t.normalized_request
			FROM ops_request_traces t
			WHERE t.user_id = ul.user_id
				AND t.api_key_id = ul.api_key_id
				AND COALESCE(t.request_id, '') = COALESCE(ul.request_id, '')
			ORDER BY t.created_at DESC, t.id DESC
			LIMIT 1
		) trace ON TRUE
		WHERE ul.id > $1
			AND ul.created_at >= $2
			AND COALESCE(ul.request_id, '') <> ''
			AND (
				COALESCE(ul.model, '') ILIKE '%claude%'
				OR COALESCE(ul.requested_model, '') ILIKE '%claude%'
				OR COALESCE(ul.upstream_model, '') ILIKE '%claude%'
				OR COALESCE(ul.inbound_endpoint, '') = $3
				OR COALESCE(ul.upstream_endpoint, '') = $3
			)
			AND (
				COALESCE(ul.inbound_endpoint, '') = ''
				OR COALESCE(ul.upstream_endpoint, '') = ''
				OR ul.thinking_enabled IS NULL
				OR COALESCE(ul.reasoning_effort, '') = ''
			)
		ORDER BY ul.id ASC
		LIMIT $4
	`, afterID, since.UTC(), service.EndpointMessages, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	candidates := make([]service.ClaudeUsageRepairCandidate, 0, limit)
	for rows.Next() {
		candidate, err := scanClaudeUsageRepairCandidate(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func scanClaudeUsageRepairCandidate(scanner interface{ Scan(dest ...any) error }) (service.ClaudeUsageRepairCandidate, error) {
	candidate := service.ClaudeUsageRepairCandidate{}
	var (
		inboundEndpoint  sql.NullString
		upstreamEndpoint sql.NullString
		thinkingEnabled  sql.NullBool
		reasoningEffort  sql.NullString
		traceHasThinking sql.NullBool
	)
	if err := scanner.Scan(
		&candidate.UsageID,
		&candidate.RequestID,
		&candidate.CreatedAt,
		&candidate.Model,
		&candidate.RequestedModel,
		&candidate.UpstreamModel,
		&inboundEndpoint,
		&upstreamEndpoint,
		&thinkingEnabled,
		&reasoningEffort,
		&candidate.TraceRoutePath,
		&candidate.TraceUpstreamPath,
		&traceHasThinking,
		&candidate.TraceInboundJSON,
		&candidate.TraceNormalizedJSON,
	); err != nil {
		return service.ClaudeUsageRepairCandidate{}, err
	}
	if inboundEndpoint.Valid {
		candidate.InboundEndpoint = &inboundEndpoint.String
	}
	if upstreamEndpoint.Valid {
		candidate.UpstreamEndpoint = &upstreamEndpoint.String
	}
	if thinkingEnabled.Valid {
		value := thinkingEnabled.Bool
		candidate.ThinkingEnabled = &value
	}
	if reasoningEffort.Valid {
		candidate.ReasoningEffort = &reasoningEffort.String
	}
	if traceHasThinking.Valid {
		value := traceHasThinking.Bool
		candidate.TraceHasThinking = &value
	}
	return candidate, nil
}
