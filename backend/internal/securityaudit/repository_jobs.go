package securityaudit

import (
	"context"
	"errors"
	"time"
)

func (r *Repository) CreateJob(ctx context.Context, job *Job) error {
	if r == nil || r.db == nil || job == nil {
		return errors.New("prompt audit repository unavailable")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.Status == "" {
		job.Status = JobStatusQueued
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 3
	}
	return r.insertJob(ctx, job)
}

func (r *Repository) insertJob(ctx context.Context, job *Job) error {
	query := `INSERT INTO prompt_audit_jobs (
request_id,user_id,username_snapshot,user_email_snapshot,api_key_id,api_key_name_snapshot,
group_id,group_name,provider,endpoint,protocol,model,prompt_hash,redacted_preview,
prompt_length,message_count,stage,execution_mode,config_version,status,attempts,max_attempts,
next_attempt_at,created_at,updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		job.RequestID, int64OrNil(job.UserID), job.Username, job.UserEmail,
		int64OrNil(job.APIKeyID), job.APIKeyName, int64OrNil(job.GroupID),
		job.GroupName, job.Provider, job.Endpoint, job.Protocol, job.Model,
		job.PromptHash, job.RedactedPreview, job.PromptLength, job.MessageCount,
		job.Stage, job.ExecutionMode, job.ConfigVersion, job.Status, job.Attempts,
		job.MaxAttempts, job.NextAttemptAt, job.CreatedAt, job.CreatedAt,
	).Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *Repository) ClaimJobs(ctx context.Context, limit int) ([]*Job, error) {
	if r == nil || r.db == nil || limit <= 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, claimJobsSQL, JobStatusProcessing, JobStatusQueued, JobStatusRetry, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var jobs []*Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) MarkJobDone(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE prompt_audit_jobs SET status=$1, processed_at=NOW(), updated_at=NOW(), last_error_code='', last_error_message='' WHERE id=$2`, JobStatusDone, id)
	return err
}

func (r *Repository) MarkJobFailed(ctx context.Context, job *Job, code string, message string, retry bool) error {
	if r == nil || r.db == nil || job == nil {
		return errors.New("prompt audit repository unavailable")
	}
	status := JobStatusFailed
	next := time.Now().UTC()
	if retry && job.Attempts < job.MaxAttempts {
		status = JobStatusRetry
		next = next.Add(time.Duration(job.Attempts+1) * time.Second)
	}
	_, err := r.db.ExecContext(ctx, markJobFailedSQL, status, next, code, truncate(message, 512), job.ID)
	return err
}

func (r *Repository) QueueStats(ctx context.Context) (QueueStats, error) {
	stats := QueueStats{}
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM prompt_audit_jobs GROUP BY status`)
	if err != nil {
		return stats, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := scanQueueStat(rows, &stats); err != nil {
			return stats, err
		}
	}
	return stats, rows.Err()
}

const claimJobsSQL = `UPDATE prompt_audit_jobs SET status=$1, attempts=attempts+1,
processing_started_at=NOW(), updated_at=NOW()
WHERE id IN (
  SELECT id FROM prompt_audit_jobs
  WHERE status IN ($2,$3) AND next_attempt_at <= NOW() AND attempts < max_attempts
  ORDER BY next_attempt_at ASC, id ASC LIMIT $4 FOR UPDATE SKIP LOCKED
)
RETURNING id,request_id,user_id,username_snapshot,user_email_snapshot,api_key_id,api_key_name_snapshot,
group_id,group_name,provider,endpoint,protocol,model,prompt_hash,redacted_preview,prompt_length,
message_count,stage,execution_mode,config_version,status,attempts,max_attempts,next_attempt_at,
processing_started_at,processed_at,last_error_code,last_error_message,created_at,updated_at`

const markJobFailedSQL = `UPDATE prompt_audit_jobs SET status=$1,next_attempt_at=$2,processed_at=NOW(),updated_at=NOW(),last_error_code=$3,last_error_message=$4 WHERE id=$5`
