package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type documentAISettingRepoStub struct {
	values map[string]string
}

func (s *documentAISettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *documentAISettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *documentAISettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *documentAISettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *documentAISettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *documentAISettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *documentAISettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type documentAIAccountRepoStub struct {
	googleBatchAccountRepoStub
	schedulableByGroup map[int64][]Account
	schedulableErr     error
	getByIDErr         error
}

func (s *documentAIAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	return s.googleBatchAccountRepoStub.GetByID(ctx, id)
}

func (s *documentAIAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]Account, error) {
	if s.schedulableErr != nil {
		return nil, s.schedulableErr
	}
	if platform != PlatformBaiduDocumentAI {
		return nil, nil
	}
	items := s.schedulableByGroup[groupID]
	out := make([]Account, len(items))
	copy(out, items)
	return out, nil
}

type memoryDocumentAIJobRepo struct {
	mu   sync.Mutex
	jobs map[string]*DocumentAIJob
}

func (r *memoryDocumentAIJobRepo) Create(_ context.Context, job *DocumentAIJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.jobs == nil {
		r.jobs = map[string]*DocumentAIJob{}
	}
	cloned := cloneDocumentAIJob(job)
	now := time.Now().UTC()
	cloned.ID = int64(len(r.jobs) + 1)
	if cloned.CreatedAt.IsZero() {
		cloned.CreatedAt = now
	}
	cloned.UpdatedAt = now
	r.jobs[cloned.JobID] = cloned
	*job = *cloneDocumentAIJob(cloned)
	return nil
}

func (r *memoryDocumentAIJobRepo) GetByJobIDForUser(_ context.Context, jobID string, userID int64) (*DocumentAIJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[strings.TrimSpace(jobID)]
	if !ok || job.UserID != userID {
		return nil, sql.ErrNoRows
	}
	return cloneDocumentAIJob(job), nil
}

func (r *memoryDocumentAIJobRepo) UpdateAfterSubmit(_ context.Context, jobID string, providerJobID, providerBatchID *string, status string, providerResultJSON *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[strings.TrimSpace(jobID)]
	if job == nil {
		return sql.ErrNoRows
	}
	if providerJobID != nil {
		job.ProviderJobID = stringPtr(*providerJobID)
	}
	if providerBatchID != nil {
		job.ProviderBatchID = stringPtr(*providerBatchID)
	}
	job.Status = status
	if providerResultJSON != nil {
		job.ProviderResultJSON = stringPtr(*providerResultJSON)
	}
	job.ErrorCode = nil
	job.ErrorMessage = nil
	job.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *memoryDocumentAIJobRepo) ListPollable(_ context.Context, limit int) ([]DocumentAIJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]DocumentAIJob, 0, limit)
	for _, job := range r.jobs {
		if job.Status != DocumentAIJobStatusPending && job.Status != DocumentAIJobStatusRunning {
			continue
		}
		out = append(out, *cloneDocumentAIJob(job))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *memoryDocumentAIJobRepo) MarkRunning(_ context.Context, jobID string, providerResultJSON *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[strings.TrimSpace(jobID)]
	if job == nil {
		return sql.ErrNoRows
	}
	job.Status = DocumentAIJobStatusRunning
	if providerResultJSON != nil {
		job.ProviderResultJSON = stringPtr(*providerResultJSON)
	}
	now := time.Now().UTC()
	job.LastPolledAt = &now
	job.UpdatedAt = now
	return nil
}

func (r *memoryDocumentAIJobRepo) MarkSucceeded(_ context.Context, jobID string, providerResultJSON, normalizedResultJSON *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[strings.TrimSpace(jobID)]
	if job == nil {
		return sql.ErrNoRows
	}
	job.Status = DocumentAIJobStatusSucceeded
	if providerResultJSON != nil {
		job.ProviderResultJSON = stringPtr(*providerResultJSON)
	}
	if normalizedResultJSON != nil {
		job.NormalizedResultJSON = stringPtr(*normalizedResultJSON)
	}
	job.ErrorCode = nil
	job.ErrorMessage = nil
	now := time.Now().UTC()
	job.CompletedAt = &now
	job.LastPolledAt = &now
	job.UpdatedAt = now
	return nil
}

func (r *memoryDocumentAIJobRepo) MarkFailed(_ context.Context, jobID string, providerResultJSON *string, errorCode, errorMessage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[strings.TrimSpace(jobID)]
	if job == nil {
		return sql.ErrNoRows
	}
	job.Status = DocumentAIJobStatusFailed
	if providerResultJSON != nil {
		job.ProviderResultJSON = stringPtr(*providerResultJSON)
	}
	job.ErrorCode = stringPtr(errorCode)
	job.ErrorMessage = stringPtr(errorMessage)
	now := time.Now().UTC()
	job.CompletedAt = &now
	job.LastPolledAt = &now
	job.UpdatedAt = now
	return nil
}

func (r *memoryDocumentAIJobRepo) TouchLastPolledAt(_ context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[strings.TrimSpace(jobID)]
	if job == nil {
		return sql.ErrNoRows
	}
	now := time.Now().UTC()
	job.LastPolledAt = &now
	job.UpdatedAt = now
	return nil
}

func cloneDocumentAIJob(job *DocumentAIJob) *DocumentAIJob {
	if job == nil {
		return nil
	}
	cloned := *job
	cloned.ProviderJobID = cloneStringPointer(job.ProviderJobID)
	cloned.ProviderBatchID = cloneStringPointer(job.ProviderBatchID)
	cloned.AccountID = cloneInt64Pointer(job.AccountID)
	cloned.GroupID = cloneInt64Pointer(job.GroupID)
	cloned.FileName = cloneStringPointer(job.FileName)
	cloned.ContentType = cloneStringPointer(job.ContentType)
	cloned.FileSize = cloneInt64Pointer(job.FileSize)
	cloned.FileHash = cloneStringPointer(job.FileHash)
	cloned.ProviderResultJSON = cloneStringPointer(job.ProviderResultJSON)
	cloned.NormalizedResultJSON = cloneStringPointer(job.NormalizedResultJSON)
	cloned.ErrorCode = cloneStringPointer(job.ErrorCode)
	cloned.ErrorMessage = cloneStringPointer(job.ErrorMessage)
	cloned.CompletedAt = cloneTimePointer(job.CompletedAt)
	cloned.LastPolledAt = cloneTimePointer(job.LastPolledAt)
	return &cloned
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func documentAIServiceResponse(status int, body string, headers map[string]string) *http.Response {
	header := http.Header{}
	for key, value := range headers {
		header.Set(key, value)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newDocumentAIServiceForTest(
	repo DocumentAIJobRepository,
	accountRepo AccountRepository,
	httpUpstream HTTPUpstream,
	enabled bool,
) *DocumentAIService {
	value := "false"
	if enabled {
		value = "true"
	}
	settingService := NewSettingService(&documentAISettingRepoStub{
		values: map[string]string{SettingKeyDocumentAIEnabled: value},
	}, &config.Config{})
	return NewDocumentAIService(repo, accountRepo, httpUpstream, nil, nil, settingService)
}

func TestDocumentAIServiceSubmitJobSelectsFirstCapableAccount(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{}
	accountRepo := &documentAIAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{
			accountsByID: map[int64]*Account{
				22: {
					ID:          22,
					Platform:    PlatformBaiduDocumentAI,
					Type:        AccountTypeAPIKey,
					Concurrency: 2,
					Credentials: map[string]any{
						"async_bearer_token": "async-token",
						"async_base_url":     DefaultBaiduDocumentAIAsyncBaseURL(),
					},
				},
			},
		},
		schedulableByGroup: map[int64][]Account{
			12: {
				{
					ID:          21,
					Platform:    PlatformBaiduDocumentAI,
					Type:        AccountTypeAPIKey,
					Concurrency: 1,
					Credentials: map[string]any{
						"direct_token": "direct-only",
					},
				},
				{
					ID:          22,
					Platform:    PlatformBaiduDocumentAI,
					Type:        AccountTypeAPIKey,
					Concurrency: 2,
					Credentials: map[string]any{
						"async_bearer_token": "async-token",
						"async_base_url":     DefaultBaiduDocumentAIAsyncBaseURL(),
					},
				},
			},
		},
	}

	var capturedURL string
	var capturedAuth string
	var capturedAccountID int64
	var capturedBody string
	upstream := googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		payload, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		capturedURL = req.URL.String()
		capturedAuth = req.Header.Get("Authorization")
		capturedAccountID = accountID
		capturedBody = string(payload)
		return documentAIServiceResponse(http.StatusOK, `{"jobId":"provider-job-1","batchId":"batch-1","status":"RUNNING"}`, nil), nil
	})

	svc := newDocumentAIServiceForTest(repo, accountRepo, upstream, true)
	job, err := svc.SubmitJob(context.Background(), DocumentAISubmitJobInput{
		APIKey:      &APIKey{ID: 8, User: &User{ID: 7}},
		GroupID:     12,
		Model:       DocumentAIModelPPStructureV3,
		SourceType:  DocumentAISourceTypeFile,
		FileName:    "sample.pdf",
		ContentType: "application/pdf",
		FileBytes:   []byte("pdf"),
	})
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, DocumentAIJobStatusRunning, job.Status)
	require.NotNil(t, job.AccountID)
	require.Equal(t, int64(22), *job.AccountID)
	require.NotNil(t, job.ProviderJobID)
	require.Equal(t, "provider-job-1", *job.ProviderJobID)
	require.Equal(t, DefaultBaiduDocumentAIAsyncBaseURL()+"/jobs", capturedURL)
	require.Equal(t, "Bearer async-token", capturedAuth)
	require.Equal(t, int64(22), capturedAccountID)
	require.Contains(t, capturedBody, "PP-StructureV3")
	require.Contains(t, capturedBody, `name="file"`)

	stored, err := repo.GetByJobIDForUser(context.Background(), job.JobID, 7)
	require.NoError(t, err)
	require.NotNil(t, stored.FileHash)
	require.Len(t, *stored.FileHash, 64)
}

func TestDocumentAIServiceSubmitJobRejectsPrivateFileURL(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{}
	accountRepo := &documentAIAccountRepoStub{
		schedulableByGroup: map[int64][]Account{
			12: {
				{
					ID:       44,
					Platform: PlatformBaiduDocumentAI,
					Type:     AccountTypeAPIKey,
					Credentials: map[string]any{
						"async_bearer_token": "async-token",
						"async_base_url":     DefaultBaiduDocumentAIAsyncBaseURL(),
					},
				},
			},
		},
	}
	svc := newDocumentAIServiceForTest(repo, accountRepo, googleBatchHTTPUpstreamFunc(func(*http.Request, string, int64, int) (*http.Response, error) {
		t.Fatal("unexpected upstream call")
		return nil, nil
	}), true)

	_, err := svc.SubmitJob(context.Background(), DocumentAISubmitJobInput{
		APIKey:     &APIKey{ID: 8, User: &User{ID: 7}},
		GroupID:    12,
		Model:      DocumentAIModelPPOCRV5Server,
		SourceType: DocumentAISourceTypeFileURL,
		FileURL:    "https://localhost/private.pdf",
	})
	require.Error(t, err)
	require.Equal(t, "document_ai_invalid_request", infraerrors.Reason(err))
}

func TestDocumentAIServiceParseDirectSucceeds(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{}
	accountRepo := &documentAIAccountRepoStub{
		schedulableByGroup: map[int64][]Account{
			9: {
				{
					ID:          33,
					Platform:    PlatformBaiduDocumentAI,
					Type:        AccountTypeAPIKey,
					Concurrency: 1,
					Credentials: map[string]any{
						"direct_token": "direct-token",
						"direct_api_urls": map[string]any{
							DocumentAIModelPPOCRV5Server: "https://direct.example.com/ocr",
						},
					},
				},
			},
		},
	}

	var capturedURL string
	var capturedAuth string
	var payload map[string]any
	upstream := googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		capturedURL = req.URL.String()
		capturedAuth = req.Header.Get("Authorization")
		require.NoError(t, json.Unmarshal(body, &payload))
		return documentAIServiceResponse(http.StatusOK, `{"result":{"text":"recognized text","layoutParsingResults":[{"page":1}],"tables":[{"id":1}]}}`, nil), nil
	})

	svc := newDocumentAIServiceForTest(repo, accountRepo, upstream, true)
	job, err := svc.ParseDirect(context.Background(), DocumentAIParseDirectInput{
		APIKey:      &APIKey{ID: 8, User: &User{ID: 7}},
		GroupID:     9,
		Model:       DocumentAIModelPPOCRV5Server,
		SourceType:  DocumentAISourceTypeFile,
		FileType:    DocumentAIFileTypeImage,
		FileName:    "page.png",
		ContentType: "image/png",
		FileBytes:   []byte("png"),
	})
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, DocumentAIJobStatusSucceeded, job.Status)
	require.Equal(t, "https://direct.example.com/ocr", capturedURL)
	require.Equal(t, "token direct-token", capturedAuth)
	require.Equal(t, float64(1), payload["fileType"])
	require.NotEmpty(t, payload["file"])

	stored, err := repo.GetByJobIDForUser(context.Background(), job.JobID, 7)
	require.NoError(t, err)
	require.NotNil(t, stored.NormalizedResultJSON)

	var envelope DocumentAIResultEnvelope
	require.NoError(t, json.Unmarshal([]byte(*stored.NormalizedResultJSON), &envelope))
	require.Equal(t, "recognized text", envelope.Text)
	require.True(t, envelope.HasLayout)
	require.GreaterOrEqual(t, envelope.TableCount, 1)
}

func TestDocumentAIServiceParseDirectFileBase64GeneratesFileHash(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{}
	accountRepo := &documentAIAccountRepoStub{
		schedulableByGroup: map[int64][]Account{
			9: {
				{
					ID:          33,
					Platform:    PlatformBaiduDocumentAI,
					Type:        AccountTypeAPIKey,
					Concurrency: 1,
					Credentials: map[string]any{
						"direct_token": "direct-token",
						"direct_api_urls": map[string]any{
							DocumentAIModelPPOCRV5Server: "https://direct.example.com/ocr",
						},
					},
				},
			},
		},
	}

	encoded := base64.StdEncoding.EncodeToString([]byte("png"))
	var payload map[string]any
	upstream := googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &payload))
		return documentAIServiceResponse(http.StatusOK, `{"result":{"text":"recognized text"}}`, nil), nil
	})

	svc := newDocumentAIServiceForTest(repo, accountRepo, upstream, true)
	job, err := svc.ParseDirect(context.Background(), DocumentAIParseDirectInput{
		APIKey:     &APIKey{ID: 8, User: &User{ID: 7}},
		GroupID:    9,
		Model:      DocumentAIModelPPOCRV5Server,
		SourceType: DocumentAISourceTypeFileBase64,
		FileType:   DocumentAIFileTypeImage,
		FileBase64: encoded,
	})
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, encoded, payload["file"])

	stored, err := repo.GetByJobIDForUser(context.Background(), job.JobID, 7)
	require.NoError(t, err)
	require.NotNil(t, stored.FileHash)
	require.Len(t, *stored.FileHash, 64)
}

func TestDocumentAIServiceSubmitJobPersistsProviderRawJSONOnFailure(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{}
	accountRepo := &documentAIAccountRepoStub{
		schedulableByGroup: map[int64][]Account{
			12: {
				{
					ID:       22,
					Platform: PlatformBaiduDocumentAI,
					Type:     AccountTypeAPIKey,
					Credentials: map[string]any{
						"async_bearer_token": "async-token",
						"async_base_url":     DefaultBaiduDocumentAIAsyncBaseURL(),
					},
				},
			},
		},
	}
	const providerBody = `{"errorCode":"401","errorMsg":"bad token","traceId":"req-submit-1"}`
	upstream := googleBatchHTTPUpstreamFunc(func(*http.Request, string, int64, int) (*http.Response, error) {
		return documentAIServiceResponse(http.StatusUnauthorized, providerBody, nil), nil
	})

	svc := newDocumentAIServiceForTest(repo, accountRepo, upstream, true)
	_, err := svc.SubmitJob(context.Background(), DocumentAISubmitJobInput{
		APIKey:     &APIKey{ID: 8, User: &User{ID: 7}},
		GroupID:    12,
		Model:      DocumentAIModelPPOCRV5Server,
		SourceType: DocumentAISourceTypeFile,
		FileBytes:  []byte("png"),
	})
	require.Error(t, err)
	require.Equal(t, "document_ai_auth_error", infraerrors.Reason(err))

	var stored *DocumentAIJob
	for _, job := range repo.jobs {
		stored = job
		break
	}
	require.NotNil(t, stored)
	require.Equal(t, DocumentAIJobStatusFailed, stored.Status)
	require.NotNil(t, stored.ProviderResultJSON)
	require.JSONEq(t, providerBody, *stored.ProviderResultJSON)
	require.NotNil(t, stored.ErrorCode)
	require.Equal(t, "document_ai_auth_error", *stored.ErrorCode)
	require.NotNil(t, stored.ErrorMessage)
	require.Equal(t, "bad token", *stored.ErrorMessage)
}

func TestDocumentAIServiceParseDirectPersistsProviderRawJSONOnFailure(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{}
	accountRepo := &documentAIAccountRepoStub{
		schedulableByGroup: map[int64][]Account{
			9: {
				{
					ID:       33,
					Platform: PlatformBaiduDocumentAI,
					Type:     AccountTypeAPIKey,
					Credentials: map[string]any{
						"direct_token": "direct-token",
						"direct_api_urls": map[string]any{
							DocumentAIModelPPOCRV5Server: "https://direct.example.com/ocr",
						},
					},
				},
			},
		},
	}
	const providerBody = `{"errorCode":"1002","errorMsg":"invalid payload","requestId":"req-direct-1"}`
	upstream := googleBatchHTTPUpstreamFunc(func(*http.Request, string, int64, int) (*http.Response, error) {
		return documentAIServiceResponse(http.StatusBadRequest, providerBody, nil), nil
	})

	svc := newDocumentAIServiceForTest(repo, accountRepo, upstream, true)
	_, err := svc.ParseDirect(context.Background(), DocumentAIParseDirectInput{
		APIKey:      &APIKey{ID: 8, User: &User{ID: 7}},
		GroupID:     9,
		Model:       DocumentAIModelPPOCRV5Server,
		SourceType:  DocumentAISourceTypeFile,
		FileType:    DocumentAIFileTypeImage,
		ContentType: "image/png",
		FileBytes:   []byte("png"),
	})
	require.Error(t, err)
	require.Equal(t, "document_ai_invalid_request", infraerrors.Reason(err))

	var stored *DocumentAIJob
	for _, job := range repo.jobs {
		stored = job
		break
	}
	require.NotNil(t, stored)
	require.Equal(t, DocumentAIJobStatusFailed, stored.Status)
	require.NotNil(t, stored.ProviderResultJSON)
	require.JSONEq(t, providerBody, *stored.ProviderResultJSON)
	require.NotNil(t, stored.ErrorCode)
	require.Equal(t, "document_ai_invalid_request", *stored.ErrorCode)
	require.NotNil(t, stored.ErrorMessage)
	require.Equal(t, "invalid payload", *stored.ErrorMessage)
}

func TestDocumentAIServicePollJobMarksSucceeded(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{
		jobs: map[string]*DocumentAIJob{
			"job-poll": {
				ID:         1,
				JobID:      "job-poll",
				UserID:     7,
				APIKeyID:   8,
				AccountID:  int64Ptr(55),
				GroupID:    int64Ptr(12),
				Mode:       DocumentAIJobModeAsync,
				Model:      DocumentAIModelPPStructureV3,
				SourceType: DocumentAISourceTypeFileURL,
				Status:     DocumentAIJobStatusPending,
				ProviderJobID: func() *string {
					value := "provider-job-1"
					return &value
				}(),
				CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
				UpdatedAt: time.Now().UTC().Add(-10 * time.Minute),
			},
		},
	}
	account := &Account{
		ID:       55,
		Platform: PlatformBaiduDocumentAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"async_bearer_token": "async-token",
			"async_base_url":     DefaultBaiduDocumentAIAsyncBaseURL(),
		},
	}
	accountRepo := &documentAIAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{
			accountsByID: map[int64]*Account{55: account},
		},
	}

	call := 0
	upstream := googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		call++
		switch call {
		case 1:
			require.Contains(t, req.URL.String(), "/jobs/provider-job-1")
			return documentAIServiceResponse(http.StatusOK, `{"status":"SUCCEEDED","resultUrl":{"markdownUrl":"https://download.example.com/result.md","jsonUrl":"https://download.example.com/result.json"}}`, nil), nil
		case 2:
			require.Equal(t, "https://download.example.com/result.md", req.URL.String())
			return documentAIServiceResponse(http.StatusOK, "markdown result", map[string]string{"Content-Type": "text/plain"}), nil
		case 3:
			require.Equal(t, "https://download.example.com/result.json", req.URL.String())
			return documentAIServiceResponse(http.StatusOK, `{"result":{"layoutParsingResults":[{"bbox":[1,2,3,4]}],"tables":[{"id":1}]}}`, nil), nil
		default:
			t.Fatalf("unexpected upstream call %d", call)
			return nil, nil
		}
	})

	svc := newDocumentAIServiceForTest(repo, accountRepo, upstream, true)
	job := cloneDocumentAIJob(repo.jobs["job-poll"])
	svc.pollJob(context.Background(), job)

	stored := repo.jobs["job-poll"]
	require.Equal(t, DocumentAIJobStatusSucceeded, stored.Status)
	require.NotNil(t, stored.CompletedAt)
	require.NotNil(t, stored.NormalizedResultJSON)

	var envelope DocumentAIResultEnvelope
	require.NoError(t, json.Unmarshal([]byte(*stored.NormalizedResultJSON), &envelope))
	require.Equal(t, "markdown result", envelope.Text)
	require.True(t, envelope.HasLayout)
	require.GreaterOrEqual(t, envelope.PageCount, 1)
	require.GreaterOrEqual(t, envelope.TableCount, 1)
}

func TestDocumentAIServicePollJobMarksFailedOnAuthError(t *testing.T) {
	repo := &memoryDocumentAIJobRepo{
		jobs: map[string]*DocumentAIJob{
			"job-auth": {
				ID:         1,
				JobID:      "job-auth",
				UserID:     7,
				APIKeyID:   8,
				AccountID:  int64Ptr(56),
				GroupID:    int64Ptr(12),
				Mode:       DocumentAIJobModeAsync,
				Model:      DocumentAIModelPPOCRV5Server,
				SourceType: DocumentAISourceTypeFile,
				Status:     DocumentAIJobStatusRunning,
				ProviderJobID: func() *string {
					value := "provider-job-auth"
					return &value
				}(),
				CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
				UpdatedAt: time.Now().UTC().Add(-10 * time.Minute),
			},
		},
	}
	account := &Account{
		ID:       56,
		Platform: PlatformBaiduDocumentAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"async_bearer_token": "async-token",
			"async_base_url":     DefaultBaiduDocumentAIAsyncBaseURL(),
		},
	}
	accountRepo := &documentAIAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{
			accountsByID: map[int64]*Account{56: account},
		},
	}
	upstream := googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		require.Contains(t, req.URL.String(), "/jobs/provider-job-auth")
		return documentAIServiceResponse(http.StatusUnauthorized, `{"errorCode":"401","errorMsg":"bad token"}`, nil), nil
	})

	svc := newDocumentAIServiceForTest(repo, accountRepo, upstream, true)
	job := cloneDocumentAIJob(repo.jobs["job-auth"])
	svc.pollJob(context.Background(), job)

	stored := repo.jobs["job-auth"]
	require.Equal(t, DocumentAIJobStatusFailed, stored.Status)
	require.NotNil(t, stored.ErrorCode)
	require.Equal(t, "document_ai_auth_error", *stored.ErrorCode)
	require.NotNil(t, stored.ErrorMessage)
	require.Equal(t, "bad token", *stored.ErrorMessage)
}
