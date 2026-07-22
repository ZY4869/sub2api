package service

import (
	"context"
	"io"
	"time"
)

type ImageBatchRepository interface {
	CreateJob(ctx context.Context, job *ImageBatchJob, items []ImageBatchItem) error
	GetGroupSettings(ctx context.Context, groupID int64) (*ImageBatchGroupSettings, error)
	GetJobForUser(ctx context.Context, userID int64, jobID string) (*ImageBatchJob, error)
	GetJobForAPIKey(ctx context.Context, apiKeyID int64, jobID string) (*ImageBatchJob, error)
	ListJobsForUser(ctx context.Context, userID int64, limit int, before time.Time) ([]ImageBatchJob, error)
	ListItemsForUser(ctx context.Context, userID int64, jobID string, limit int) ([]ImageBatchItem, error)
	GetItemForUser(ctx context.Context, userID int64, jobID string, customID string) (*ImageBatchItem, error)
	GetOutputForUser(ctx context.Context, userID int64, jobID string, customID string) (*ImageBatchOutput, error)
	ListOutputsForUser(ctx context.Context, userID int64, jobID string) ([]ImageBatchOutput, error)
	SoftDeleteJob(ctx context.Context, userID int64, jobID string) error
	MarkOutputsDeleted(ctx context.Context, userID int64, jobID string) error
	ListOutputsForUserIncludingDeleted(ctx context.Context, userID int64, jobID string) ([]ImageBatchOutput, error)
	ListExpiredOutputs(ctx context.Context, before time.Time, limit int) ([]ImageBatchOutput, error)
	CleanupExpiredOutputs(ctx context.Context, before time.Time, limit int) (int64, error)
	TryCancelJob(ctx context.Context, userID int64, jobID string, reason string) (*ImageBatchJob, error)

	ClaimNextRunnableJob(ctx context.Context, before time.Time) (*ImageBatchJob, []ImageBatchItem, error)
	MarkJobSubmitted(ctx context.Context, jobID string, providerBatchName string) error
	MarkJobRunning(ctx context.Context, jobID string, providerResultFileName string) error
	MarkJobIndexing(ctx context.Context, jobID string, providerResultFileName string) error
	MarkJobSettling(ctx context.Context, jobID string, message string, errorID string) error
	IncrementJobSettlementRetry(ctx context.Context, jobID string) (int, error)
	FailJob(ctx context.Context, jobID string, message string, errorID string) error
	CompleteJob(ctx context.Context, jobID string, providerResultFileName string) error
	UpsertOutput(ctx context.Context, output *ImageBatchOutput) error
	MarkItemResult(ctx context.Context, jobID string, customID string, status ImageBatchItemStatus, outputCount int, message string, errorID string) error
	RefreshJobCounts(ctx context.Context, jobID string) error
	AddEvent(ctx context.Context, jobID string, itemID *int64, eventType string, from string, to string, message string, requestID string, payload map[string]any) error
}

type ImageBatchGoogleForwarder interface {
	ForwardGoogleBatches(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error)
	ForwardSimplifiedVertexBatchPredictionJobs(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error)
	ForwardGoogleFileDownload(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error)
	ForwardGoogleArchiveBatch(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error)
}

type ImageBatchStreamResult interface {
	Status() int
	Header() map[string][]string
	ReadAll() ([]byte, error)
	Open() (io.ReadCloser, error)
}
