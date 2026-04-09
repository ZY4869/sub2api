package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type googleBatchSelectionAccountRepoStub struct {
	googleBatchAccountRepoStub
	schedulableByGroup map[int64][]Account
}

func (s *googleBatchSelectionAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, _ []string) ([]Account, error) {
	accounts := s.schedulableByGroup[groupID]
	cloned := make([]Account, len(accounts))
	copy(cloned, accounts)
	return cloned, nil
}

func (s *googleBatchSelectionAccountRepoStub) ListSchedulableByPlatforms(_ context.Context, _ []string) ([]Account, error) {
	return nil, nil
}

func (s *googleBatchSelectionAccountRepoStub) ListSchedulableUngroupedByPlatforms(_ context.Context, _ []string) ([]Account, error) {
	return nil, nil
}

type googleBatchArchiveJobRepoMemoryStub struct {
	jobsByID            map[int64]*GoogleBatchArchiveJob
	jobsByPublicName    map[string]*GoogleBatchArchiveJob
	jobsByExecutionName map[string]*GoogleBatchArchiveJob
}

func (s *googleBatchArchiveJobRepoMemoryStub) Upsert(_ context.Context, job *GoogleBatchArchiveJob) error {
	if job == nil {
		return nil
	}
	if s.jobsByID == nil {
		s.jobsByID = map[int64]*GoogleBatchArchiveJob{}
	}
	if s.jobsByPublicName == nil {
		s.jobsByPublicName = map[string]*GoogleBatchArchiveJob{}
	}
	if s.jobsByExecutionName == nil {
		s.jobsByExecutionName = map[string]*GoogleBatchArchiveJob{}
	}
	cloned := *job
	s.jobsByID[job.ID] = &cloned
	s.jobsByPublicName[strings.ToLower(strings.TrimSpace(job.PublicBatchName))] = &cloned
	s.jobsByExecutionName[strings.ToLower(strings.TrimSpace(job.ExecutionBatchName))] = &cloned
	return nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) GetByID(_ context.Context, id int64) (*GoogleBatchArchiveJob, error) {
	return s.jobsByID[id], nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) GetByPublicBatchName(_ context.Context, publicBatchName string) (*GoogleBatchArchiveJob, error) {
	return s.jobsByPublicName[strings.ToLower(strings.TrimSpace(publicBatchName))], nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) GetByExecutionBatchName(_ context.Context, executionBatchName string) (*GoogleBatchArchiveJob, error) {
	return s.jobsByExecutionName[strings.ToLower(strings.TrimSpace(executionBatchName))], nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) ListDueForPoll(_ context.Context, _ time.Time, _ int) ([]*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) ListDueForPrefetch(_ context.Context, _ time.Time, _ int) ([]*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) ListExpiredForCleanup(_ context.Context, _ time.Time, _ int) ([]*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) TryMarkBillingSettled(_ context.Context, _ int64) (bool, error) {
	return true, nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) TryRestoreBillingPending(_ context.Context, _ int64) (bool, error) {
	return true, nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) TouchLastPublicResultAccess(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (s *googleBatchArchiveJobRepoMemoryStub) SoftDelete(_ context.Context, _ int64) error {
	return nil
}

type googleBatchArchiveObjectRepoStub struct {
	objectsByPublic map[string]*GoogleBatchArchiveObject
	objectsByJobID  map[int64][]*GoogleBatchArchiveObject
}

func (s *googleBatchArchiveObjectRepoStub) Upsert(_ context.Context, object *GoogleBatchArchiveObject) error {
	if object == nil {
		return nil
	}
	if s.objectsByPublic == nil {
		s.objectsByPublic = map[string]*GoogleBatchArchiveObject{}
	}
	if s.objectsByJobID == nil {
		s.objectsByJobID = map[int64][]*GoogleBatchArchiveObject{}
	}
	cloned := *object
	key := strings.ToLower(strings.TrimSpace(object.PublicResourceKind) + ":" + strings.TrimSpace(object.PublicResourceName))
	s.objectsByPublic[key] = &cloned

	items := s.objectsByJobID[object.JobID]
	for index, item := range items {
		if item != nil && strings.EqualFold(item.PublicResourceKind, object.PublicResourceKind) && strings.EqualFold(item.PublicResourceName, object.PublicResourceName) {
			items[index] = &cloned
			s.objectsByJobID[object.JobID] = items
			return nil
		}
	}
	s.objectsByJobID[object.JobID] = append(items, &cloned)
	return nil
}

func (s *googleBatchArchiveObjectRepoStub) GetByPublicResource(_ context.Context, publicResourceKind string, publicResourceName string) (*GoogleBatchArchiveObject, error) {
	key := strings.ToLower(strings.TrimSpace(publicResourceKind) + ":" + strings.TrimSpace(publicResourceName))
	return s.objectsByPublic[key], nil
}

func (s *googleBatchArchiveObjectRepoStub) ListByJobID(_ context.Context, jobID int64) ([]*GoogleBatchArchiveObject, error) {
	return s.objectsByJobID[jobID], nil
}

func (s *googleBatchArchiveObjectRepoStub) SoftDeleteByJobID(_ context.Context, _ int64) error {
	return nil
}

type staticSettingRepoStub struct {
	values map[string]string
}

func (s *staticSettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	if value, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *staticSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *staticSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *staticSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *staticSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *staticSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *staticSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func newGoogleBatchVertexOAuthTestAccount(id int64, projectID string, location string) *Account {
	return &Account{
		ID:          id,
		Name:        "vertex-batch-test",
		Platform:    PlatformGemini,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"oauth_type":        "vertex_ai",
			"vertex_project_id": projectID,
			"vertex_location":   location,
			"access_token":      "vertex-access-token",
		},
	}
}

func TestGeminiBatchFilesForwardSuccessPaths(t *testing.T) {
	groupID := int64(1)
	account := newGoogleBatchAIStudioTestAccount(101)
	bindingRepo := &googleBatchBindingRepoStub{items: map[string]*UpstreamResourceBinding{
		"file-1": {
			ResourceKind: UpstreamResourceKindGeminiFile,
			ResourceName: "file-1",
			AccountID:    account.ID,
		},
	}}
	accountRepo := &googleBatchSelectionAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{
			accountsByID: map[int64]*Account{account.ID: account},
		},
		schedulableByGroup: map[int64][]Account{
			groupID: []Account{*account},
		},
	}
	svc := &GeminiMessagesCompatService{
		accountRepo:         accountRepo,
		resourceBindingRepo: bindingRepo,
		httpUpstream: googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
			require.Equal(t, account.ID, accountID)
			switch {
			case req.Method == http.MethodGet && req.URL.Path == "/v1beta/files":
				return newGoogleBatchJSONResponse(http.StatusOK, `{"files":[{"name":"file-1"}]}`), nil
			case req.Method == http.MethodPost && req.URL.Path == "/upload/v1beta/files":
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), "gemini-2.5-flash")
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"uploaded-file"}`), nil
			case req.Method == http.MethodGet && req.URL.Path == "/v1beta/files/file-1":
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"file-1","state":"ACTIVE"}`), nil
			case req.Method == http.MethodDelete && req.URL.Path == "/v1beta/files/file-1":
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"file-1","deleted":true}`), nil
			case req.Method == http.MethodGet && req.URL.Path == "/download/v1beta/files/file-1:download":
				return &http.Response{
					StatusCode:    http.StatusOK,
					Header:        http.Header{"Content-Type": []string{"application/octet-stream"}},
					Body:          io.NopCloser(strings.NewReader("file-contents")),
					ContentLength: int64(len("file-contents")),
				}, nil
			default:
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				return nil, nil
			}
		}),
		cfg: &config.Config{},
	}

	listResult, listAccount, err := svc.ForwardGoogleFiles(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodGet,
		Path:    "/v1beta/files",
	})
	require.NoError(t, err)
	require.NotNil(t, listAccount)
	require.Equal(t, account.ID, listAccount.ID)
	require.Equal(t, http.StatusOK, listResult.(*UpstreamHTTPResult).StatusCode)

	uploadResult, uploadAccount, err := svc.ForwardGoogleFiles(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodPost,
		Path:    "/upload/v1beta/files",
		Body:    []byte(`{"key":"line-1","request":{"model":"gemini-2.5-flash"}}` + "\n"),
	})
	require.NoError(t, err)
	require.NotNil(t, uploadAccount)
	require.Equal(t, account.ID, uploadAccount.ID)
	require.Equal(t, http.StatusOK, uploadResult.(*UpstreamHTTPResult).StatusCode)
	require.NotNil(t, bindingRepo.items["uploaded-file"])

	getResult, getAccount, err := svc.ForwardGoogleFiles(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodGet,
		Path:    "/v1beta/files/file-1",
	})
	require.NoError(t, err)
	require.Equal(t, account.ID, getAccount.ID)
	require.Equal(t, http.StatusOK, getResult.(*UpstreamHTTPResult).StatusCode)

	deleteResult, deleteAccount, err := svc.ForwardGoogleFiles(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodDelete,
		Path:    "/v1beta/files/file-1",
	})
	require.NoError(t, err)
	require.Equal(t, account.ID, deleteAccount.ID)
	require.Equal(t, http.StatusOK, deleteResult.(*UpstreamHTTPResult).StatusCode)

	downloadResult, downloadAccount, err := svc.ForwardGoogleFileDownload(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodGet,
		Path:    "/download/v1beta/files/file-1:download",
	})
	require.NoError(t, err)
	require.Equal(t, account.ID, downloadAccount.ID)
	streamResult := downloadResult.(*UpstreamHTTPStreamResult)
	defer func() { _ = streamResult.Body.Close() }()
	body, readErr := io.ReadAll(streamResult.Body)
	require.NoError(t, readErr)
	require.Equal(t, "file-contents", string(body))
}

func TestGeminiBatchArchiveAndVertexSuccessPaths(t *testing.T) {
	groupID := int64(2)
	aistudioAccount := newGoogleBatchAIStudioTestAccount(201)
	vertexAccount := newGoogleBatchVertexOAuthTestAccount(202, "vertex-project", "us-central1")
	accountRepo := &googleBatchSelectionAccountRepoStub{
		googleBatchAccountRepoStub: googleBatchAccountRepoStub{
			accountsByID: map[int64]*Account{
				aistudioAccount.ID: aistudioAccount,
				vertexAccount.ID:   vertexAccount,
			},
		},
		schedulableByGroup: map[int64][]Account{
			groupID: []Account{*aistudioAccount, *vertexAccount},
		},
	}
	bindingRepo := &googleBatchBindingRepoStub{items: map[string]*UpstreamResourceBinding{
		buildVertexBatchJobResourceName("vertex-project", "us-central1", "job-1"): {
			ResourceKind:   UpstreamResourceKindVertexBatchJob,
			ResourceName:   buildVertexBatchJobResourceName("vertex-project", "us-central1", "job-1"),
			AccountID:      vertexAccount.ID,
			ProviderFamily: UpstreamProviderVertexAI,
		},
	}}
	jobRepo := &googleBatchArchiveJobRepoMemoryStub{}
	objectRepo := &googleBatchArchiveObjectRepoStub{}
	settings := NormalizeGoogleBatchArchiveSettings(&GoogleBatchArchiveSettings{
		Enabled:          true,
		LocalStorageRoot: t.TempDir(),
	})
	settingsJSON, err := json.Marshal(settings)
	require.NoError(t, err)
	storage := NewGoogleBatchArchiveStorage()
	settingService := NewSettingService(&staticSettingRepoStub{
		values: map[string]string{SettingKeyGoogleBatchArchiveSettings: string(settingsJSON)},
	}, nil)
	svc := &GeminiMessagesCompatService{
		accountRepo:                  accountRepo,
		resourceBindingRepo:          bindingRepo,
		googleBatchArchiveJobRepo:    jobRepo,
		googleBatchArchiveObjectRepo: objectRepo,
		googleBatchArchiveStorage:    storage,
		httpUpstream: googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.Path == "/v1beta/batches/batch-1":
				require.Equal(t, aistudioAccount.ID, accountID)
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"batches/batch-1","state":"SUCCEEDED"}`), nil
			case req.Method == http.MethodGet && req.URL.Path == "/download/v1beta/files/file-1:download":
				require.Equal(t, aistudioAccount.ID, accountID)
				return &http.Response{
					StatusCode:    http.StatusOK,
					Header:        http.Header{"Content-Type": []string{"application/octet-stream"}},
					Body:          io.NopCloser(strings.NewReader("archive-result")),
					ContentLength: int64(len("archive-result")),
				}, nil
			case req.Method == http.MethodGet && req.URL.Path == "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs":
				require.Equal(t, vertexAccount.ID, accountID)
				require.Equal(t, "Bearer vertex-access-token", req.Header.Get("Authorization"))
				return newGoogleBatchJSONResponse(http.StatusOK, `{"batchPredictionJobs":[{"name":"projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1"}]}`), nil
			case req.Method == http.MethodPost && req.URL.Path == "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs":
				require.Equal(t, vertexAccount.ID, accountID)
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1"}`), nil
			case req.Method == http.MethodGet && req.URL.Path == "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1":
				require.Equal(t, vertexAccount.ID, accountID)
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1","state":"JOB_STATE_SUCCEEDED"}`), nil
			case req.Method == http.MethodPost && req.URL.Path == "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1:cancel":
				require.Equal(t, vertexAccount.ID, accountID)
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1","state":"JOB_STATE_CANCELLED"}`), nil
			case req.Method == http.MethodDelete && req.URL.Path == "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1":
				require.Equal(t, vertexAccount.ID, accountID)
				return newGoogleBatchJSONResponse(http.StatusOK, `{"name":"projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1","deleted":true}`), nil
			default:
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				return nil, nil
			}
		}),
		settingService: settingService,
		tokenProvider:  NewGeminiTokenProvider(nil, nil, nil),
		cfg:            &config.Config{},
	}

	now := time.Now().UTC().Add(time.Hour)
	job := &GoogleBatchArchiveJob{
		ID:                      1,
		PublicBatchName:         "batches/batch-1",
		PublicProtocol:          GoogleBatchArchivePublicProtocolAIStudio,
		ExecutionProviderFamily: UpstreamProviderAIStudio,
		ExecutionBatchName:      "batches/batch-1",
		ExecutionAccountID:      aistudioAccount.ID,
		SourceAccountID:         aistudioAccount.ID,
		GroupID:                 &groupID,
		RequestedModel:          "gemini-2.5-flash",
		ArchiveState:            GoogleBatchArchiveLifecycleArchived,
		OfficialExpiresAt:       &now,
		MetadataJSON: map[string]any{
			googleBatchBindingMetadataPublicResultFileName: "files/file-1",
		},
	}
	require.NoError(t, jobRepo.Upsert(context.Background(), job))
	require.NoError(t, objectRepo.Upsert(context.Background(), &GoogleBatchArchiveObject{
		JobID:              job.ID,
		PublicResourceKind: GoogleBatchArchiveResourceKindFile,
		PublicResourceName: "files/file-1",
		IsResultPayload:    true,
	}))

	archiveBatchResult, archiveBatchAccount, err := svc.ForwardGoogleArchiveBatch(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodGet,
		Path:    "/google/batch/archive/v1beta/batches/batch-1",
	})
	require.NoError(t, err)
	require.NotNil(t, archiveBatchAccount)
	require.Equal(t, aistudioAccount.ID, archiveBatchAccount.ID)
	require.Equal(t, http.StatusOK, archiveBatchResult.(*UpstreamHTTPResult).StatusCode)

	archiveFileResult, archiveFileAccount, err := svc.ForwardGoogleArchiveFileDownload(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodGet,
		Path:    "/google/batch/archive/v1beta/files/file-1:download",
	})
	require.NoError(t, err)
	require.NotNil(t, archiveFileAccount)
	require.Equal(t, aistudioAccount.ID, archiveFileAccount.ID)
	archiveStreamResult := archiveFileResult.(*UpstreamHTTPStreamResult)
	defer func() { _ = archiveStreamResult.Body.Close() }()
	archiveBody, readErr := io.ReadAll(archiveStreamResult.Body)
	require.NoError(t, readErr)
	require.Equal(t, "archive-result", string(archiveBody))

	vertexListResult, vertexListAccount, err := svc.ForwardVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodGet,
		Path:    "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs",
	})
	require.NoError(t, err)
	require.NotNil(t, vertexListAccount)
	require.Equal(t, vertexAccount.ID, vertexListAccount.ID)
	require.Equal(t, http.StatusOK, vertexListResult.(*UpstreamHTTPResult).StatusCode)

	vertexCreateResult, vertexCreateAccount, err := svc.ForwardVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
		GroupID: &groupID,
		Method:  http.MethodPost,
		Path:    "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs",
		Body:    []byte(`{"displayName":"job-1"}`),
	})
	require.NoError(t, err)
	require.NotNil(t, vertexCreateAccount)
	require.Equal(t, vertexAccount.ID, vertexCreateAccount.ID)
	require.Equal(t, http.StatusOK, vertexCreateResult.(*UpstreamHTTPResult).StatusCode)

	vertexPaths := map[string]string{
		http.MethodGet:    "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1",
		http.MethodPost:   "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1:cancel",
		http.MethodDelete: "/v1/projects/vertex-project/locations/us-central1/batchPredictionJobs/job-1",
	}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		result, account, forwardErr := svc.ForwardVertexBatchPredictionJobs(context.Background(), GoogleBatchForwardInput{
			GroupID: &groupID,
			Method:  method,
			Path:    vertexPaths[method],
		})
		require.NoError(t, forwardErr)
		require.NotNil(t, account)
		require.Equal(t, vertexAccount.ID, account.ID)
		require.Equal(t, http.StatusOK, result.(*UpstreamHTTPResult).StatusCode)
	}
}

func newGoogleBatchJSONResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func (s *googleBatchSelectionAccountRepoStub) List(_ context.Context, _ pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
