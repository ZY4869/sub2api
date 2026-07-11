package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func imageBatchJobSelectSQL() string {
	return `
		SELECT id, user_id, api_key_id, group_id, provider, display_model_id, target_model_id,
			size, status, item_count, success_count, failed_count, cancelled_count,
			hold_request_id, idempotency_key_hash, request_fingerprint, provider_batch_name,
			provider_result_file_name, friendly_error, error_id, metadata::text, submitted_at,
			started_at, completed_at, cancelled_at, outputs_deleted_at, created_at, updated_at
		FROM image_batch_jobs
	`
}

func scanImageBatchJob(scanner rowScanner) (*service.ImageBatchJob, error) {
	var job service.ImageBatchJob
	var groupID sql.NullInt64
	var metadataRaw string
	var submittedAt, startedAt, completedAt, cancelledAt, outputsDeletedAt sql.NullTime
	var status string
	if err := scanner.Scan(
		&job.ID, &job.UserID, &job.APIKeyID, &groupID, &job.Provider, &job.DisplayModelID,
		&job.TargetModelID, &job.Size, &status, &job.ItemCount, &job.SuccessCount,
		&job.FailedCount, &job.CancelledCount, &job.HoldRequestID, &job.IdempotencyKeyHash,
		&job.RequestFingerprint, &job.ProviderBatchName, &job.ProviderResultFileName,
		&job.FriendlyError, &job.ErrorID, &metadataRaw, &submittedAt, &startedAt,
		&completedAt, &cancelledAt, &outputsDeletedAt, &job.CreatedAt, &job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	job.Status = service.ImageBatchJobStatus(status)
	if groupID.Valid {
		v := groupID.Int64
		job.GroupID = &v
	}
	job.Metadata = decodeJSONMap(metadataRaw)
	job.SubmittedAt = nullTimePtrLocal(submittedAt)
	job.StartedAt = nullTimePtrLocal(startedAt)
	job.CompletedAt = nullTimePtrLocal(completedAt)
	job.CancelledAt = nullTimePtrLocal(cancelledAt)
	job.OutputsDeletedAt = nullTimePtrLocal(outputsDeletedAt)
	return &job, nil
}

func scanImageBatchItem(scanner rowScanner) (*service.ImageBatchItem, error) {
	var item service.ImageBatchItem
	var status string
	var metadataRaw string
	if err := scanner.Scan(
		&item.ID, &item.JobID, &item.CustomID, &item.Prompt, &item.N, &item.Size,
		&status, &item.OutputCount, &item.FriendlyError, &item.ErrorID,
		&metadataRaw, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Status = service.ImageBatchItemStatus(status)
	item.Metadata = decodeJSONMap(metadataRaw)
	return &item, nil
}

func scanImageBatchOutput(scanner rowScanner) (*service.ImageBatchOutput, error) {
	var output service.ImageBatchOutput
	var metadataRaw string
	if err := scanner.Scan(
		&output.ID, &output.JobID, &output.ItemID, &output.CustomID, &output.ContentType,
		&output.StorageBackend, &output.Content, &output.SizeBytes, &output.SHA256,
		&metadataRaw, &output.CreatedAt,
	); err != nil {
		return nil, err
	}
	output.Metadata = decodeJSONMap(metadataRaw)
	return &output, nil
}

func decodeJSONMap(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func nullTimePtrLocal(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
