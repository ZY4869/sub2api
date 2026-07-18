package securityaudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (r *Repository) CreateEvent(ctx context.Context, job *Job, result *NormalizedResult, fullPrompt string) (*Event, error) {
	if r == nil || r.db == nil || job == nil || result == nil {
		return nil, errors.New("prompt audit event input invalid")
	}
	event := eventFromJob(job, result, fullPrompt)
	if err := r.insertEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (r *Repository) insertEvent(ctx context.Context, event *Event) error {
	categories, _ := json.Marshal(event.Categories)
	matched, _ := json.Marshal(event.MatchedScanners)
	scores, _ := json.Marshal(event.ScannerScores)
	evidence, _ := json.Marshal(event.ScannerEvidence)
	err := r.db.QueryRowContext(ctx, insertEventSQL,
		event.JobID, event.RequestID, int64OrNil(event.UserID), event.Username, event.UserEmail,
		int64OrNil(event.APIKeyID), event.APIKeyName, int64OrNil(event.GroupID), event.GroupName,
		event.Provider, event.Endpoint, event.Protocol, event.Model, event.PromptHash,
		event.RedactedPreview, event.FullPrompt, event.Stage, event.Decision, event.RiskLevel,
		event.Action, categories, matched, scores, evidence, event.ScannerBackend,
		event.ScannerVersion, event.GuardEndpointID, event.PolicyID, event.PolicyVersion,
		event.ConfigVersion, event.ChunkTotal, event.LatencyMS, time.Now().UTC(),
	).Scan(&event.ID, &event.CreatedAt)
	return err
}

func (r *Repository) ListEvents(ctx context.Context, filter EventFilter) (*EventList, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("prompt audit repository unavailable")
	}
	filter = normalizeEventFilter(filter)
	where, args := buildEventWhere(filter)
	total, err := r.countEvents(ctx, where, args)
	if err != nil {
		return nil, err
	}
	items, err := r.queryEvents(ctx, filter, where, args)
	if err != nil {
		return nil, err
	}
	return &EventList{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (r *Repository) countEvents(ctx context.Context, where string, args []any) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM prompt_audit_events"+where, args...).Scan(&total)
	return total, err
}

func (r *Repository) queryEvents(ctx context.Context, filter EventFilter, where string, args []any) ([]*Event, error) {
	selectSQL := eventSelectSQL(filter.IncludeFullText)
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := selectSQL + where + " ORDER BY created_at DESC, id DESC LIMIT $" + fmt.Sprint(len(args)-1) + " OFFSET $" + fmt.Sprint(len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []*Event{}
	for rows.Next() {
		event, err := scanEvent(rows, filter.IncludeFullText)
		if err != nil {
			return nil, err
		}
		items = append(items, event)
	}
	return items, rows.Err()
}

func (r *Repository) GetEvent(ctx context.Context, id int64, includeFullText bool) (*Event, error) {
	if r == nil || r.db == nil || id <= 0 {
		return nil, ErrEventNotFound
	}
	query := eventSelectSQL(includeFullText) + " WHERE id=$1"
	event, err := scanEvent(r.db.QueryRowContext(ctx, query, id), includeFullText)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	return event, err
}

const insertEventSQL = `INSERT INTO prompt_audit_events (
job_id,request_id,user_id,username_snapshot,user_email_snapshot,api_key_id,api_key_name_snapshot,
group_id,group_name,provider,endpoint,protocol,model,prompt_hash,redacted_preview,full_prompt,
stage,decision,risk_level,action,categories,matched_scanners,scanner_scores,scanner_evidence,
scanner_backend,scanner_version,guard_endpoint_id,policy_id,policy_version,config_version,chunk_total,latency_ms,created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33)
RETURNING id, created_at`
