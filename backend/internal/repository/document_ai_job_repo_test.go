package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewDocumentAIJobRepository(t *testing.T) {
	db, _ := newSQLMock(t)
	repo := NewDocumentAIJobRepository(db)
	require.NotNil(t, repo)
}

func TestDocumentAIJobRepositoryCreate(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &documentAIJobRepository{sql: db}

	accountID := int64(12)
	groupID := int64(88)
	fileSize := int64(1024)
	fileName := "sample.pdf"
	contentType := "application/pdf"
	fileHash := "abc123"
	now := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	job := &service.DocumentAIJob{
		JobID:       "job-1",
		AccountID:   &accountID,
		UserID:      7,
		APIKeyID:    9,
		GroupID:     &groupID,
		Mode:        service.DocumentAIJobModeAsync,
		Model:       service.DocumentAIModelPPStructureV3,
		SourceType:  service.DocumentAISourceTypeFile,
		FileName:    &fileName,
		ContentType: &contentType,
		FileSize:    &fileSize,
		FileHash:    &fileHash,
		Status:      service.DocumentAIJobStatusPending,
	}

	mock.ExpectQuery("INSERT INTO document_ai_jobs").
		WithArgs(
			job.JobID,
			job.ProviderJobID,
			job.ProviderBatchID,
			job.AccountID,
			job.UserID,
			job.APIKeyID,
			job.GroupID,
			job.Mode,
			job.Model,
			job.SourceType,
			job.FileName,
			job.ContentType,
			job.FileSize,
			job.FileHash,
			job.Status,
			job.ProviderResultJSON,
			job.NormalizedResultJSON,
			job.ErrorCode,
			job.ErrorMessage,
			job.CompletedAt,
			job.LastPolledAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), now, now))

	err := repo.Create(context.Background(), job)
	require.NoError(t, err)
	require.Equal(t, int64(1), job.ID)
	require.Equal(t, now, job.CreatedAt)
	require.Equal(t, now, job.UpdatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentAIJobRepositoryGetByJobIDForUser(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &documentAIJobRepository{sql: db}

	createdAt := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	completedAt := createdAt.Add(2 * time.Minute)
	lastPolledAt := createdAt.Add(30 * time.Second)
	rows := sqlmock.NewRows([]string{
		"id", "job_id", "provider_job_id", "provider_batch_id", "account_id", "user_id", "api_key_id", "group_id",
		"mode", "model", "source_type", "file_name", "content_type", "file_size", "file_hash", "status",
		"provider_result_json", "normalized_result_json", "error_code", "error_message",
		"created_at", "updated_at", "completed_at", "last_polled_at",
	}).AddRow(
		int64(1),
		"job-1",
		"provider-1",
		"batch-1",
		int64(9),
		int64(7),
		int64(8),
		int64(5),
		service.DocumentAIJobModeDirect,
		service.DocumentAIModelPPOCRV5Server,
		service.DocumentAISourceTypeFile,
		"sample.png",
		"image/png",
		int64(128),
		"hash",
		service.DocumentAIJobStatusSucceeded,
		`{"provider":"ok"}`,
		`{"status":"succeeded"}`,
		nil,
		nil,
		createdAt,
		updatedAt,
		completedAt,
		lastPolledAt,
	)

	mock.ExpectQuery("SELECT id, job_id, provider_job_id").
		WithArgs("job-1", int64(7)).
		WillReturnRows(rows)

	job, err := repo.GetByJobIDForUser(context.Background(), " job-1 ", 7)
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, "job-1", job.JobID)
	require.NotNil(t, job.ProviderJobID)
	require.Equal(t, "provider-1", *job.ProviderJobID)
	require.NotNil(t, job.CompletedAt)
	require.Equal(t, completedAt, *job.CompletedAt)
	require.NotNil(t, job.LastPolledAt)
	require.Equal(t, lastPolledAt, *job.LastPolledAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentAIJobRepositoryUpdateAfterSubmit(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &documentAIJobRepository{sql: db}

	providerJobID := "provider-1"
	providerBatchID := "batch-1"
	providerResult := `{"jobId":"provider-1"}`

	mock.ExpectExec("UPDATE document_ai_jobs").
		WithArgs(&providerJobID, &providerBatchID, service.DocumentAIJobStatusRunning, &providerResult, "job-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateAfterSubmit(
		context.Background(),
		" job-1 ",
		&providerJobID,
		&providerBatchID,
		service.DocumentAIJobStatusRunning,
		&providerResult,
	)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentAIJobRepositoryListPollable(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &documentAIJobRepository{sql: db}

	createdAt := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	rows := sqlmock.NewRows([]string{
		"id", "job_id", "provider_job_id", "provider_batch_id", "account_id", "user_id", "api_key_id", "group_id",
		"mode", "model", "source_type", "file_name", "content_type", "file_size", "file_hash", "status",
		"provider_result_json", "normalized_result_json", "error_code", "error_message",
		"created_at", "updated_at", "completed_at", "last_polled_at",
	}).AddRow(
		int64(1),
		"job-pending",
		"provider-1",
		nil,
		int64(9),
		int64(7),
		int64(8),
		int64(5),
		service.DocumentAIJobModeAsync,
		service.DocumentAIModelPPStructureV3,
		service.DocumentAISourceTypeFileURL,
		nil,
		nil,
		nil,
		nil,
		service.DocumentAIJobStatusPending,
		nil,
		nil,
		nil,
		nil,
		createdAt,
		updatedAt,
		nil,
		nil,
	).AddRow(
		int64(2),
		"job-running",
		"provider-2",
		nil,
		int64(10),
		int64(7),
		int64(8),
		int64(5),
		service.DocumentAIJobModeAsync,
		service.DocumentAIModelPaddleOCRVL15,
		service.DocumentAISourceTypeFile,
		"doc.pdf",
		"application/pdf",
		int64(256),
		"hash-2",
		service.DocumentAIJobStatusRunning,
		`{"status":"running"}`,
		nil,
		nil,
		nil,
		createdAt,
		updatedAt,
		nil,
		nil,
	)

	mock.ExpectQuery("SELECT id, job_id, provider_job_id").
		WithArgs(service.DocumentAIJobStatusPending, service.DocumentAIJobStatusRunning, 2).
		WillReturnRows(rows)

	jobs, err := repo.ListPollable(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	require.Equal(t, "job-pending", jobs[0].JobID)
	require.Equal(t, service.DocumentAIJobStatusRunning, jobs[1].Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentAIJobRepositoryMarkSucceededAndFailed(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &documentAIJobRepository{sql: db}

	providerResult := `{"status":"done"}`
	normalizedResult := `{"status":"succeeded","text":"ok"}`

	mock.ExpectExec("UPDATE document_ai_jobs").
		WithArgs(service.DocumentAIJobStatusSucceeded, &providerResult, &normalizedResult, "job-success").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkSucceeded(context.Background(), " job-success ", &providerResult, &normalizedResult)
	require.NoError(t, err)

	mock.ExpectExec("UPDATE document_ai_jobs").
		WithArgs(service.DocumentAIJobStatusFailed, &providerResult, "document_ai_auth_error", "bad token", "job-failed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.MarkFailed(context.Background(), " job-failed ", &providerResult, " document_ai_auth_error ", " bad token ")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentAIJobRepositoryGetByJobIDForUserNotFound(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &documentAIJobRepository{sql: db}

	mock.ExpectQuery("SELECT id, job_id, provider_job_id").
		WithArgs("job-missing", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_id", "provider_job_id", "provider_batch_id", "account_id", "user_id", "api_key_id", "group_id",
			"mode", "model", "source_type", "file_name", "content_type", "file_size", "file_hash", "status",
			"provider_result_json", "normalized_result_json", "error_code", "error_message",
			"created_at", "updated_at", "completed_at", "last_polled_at",
		}))

	job, err := repo.GetByJobIDForUser(context.Background(), "job-missing", 1)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Nil(t, job)
	require.NoError(t, mock.ExpectationsWereMet())
}
