package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type googleBatchHTTPUpstreamFunc func(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error)

func (f googleBatchHTTPUpstreamFunc) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return f(req, proxyURL, accountID, accountConcurrency)
}

func (f googleBatchHTTPUpstreamFunc) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *TLSFingerprintProfile) (*http.Response, error) {
	return f(req, proxyURL, accountID, accountConcurrency)
}

type googleBatchAccountRepoStub struct {
	accountsByID map[int64]*Account
}

func (s *googleBatchAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if account, ok := s.accountsByID[id]; ok {
		return account, nil
	}
	return nil, errors.New("account not found")
}

func (s *googleBatchAccountRepoStub) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	result := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := s.accountsByID[id]; ok {
			result = append(result, account)
		}
	}
	return result, nil
}

func (s *googleBatchAccountRepoStub) ExistsByID(_ context.Context, id int64) (bool, error) {
	_, ok := s.accountsByID[id]
	return ok, nil
}

func (s *googleBatchAccountRepoStub) UpdateLastUsed(_ context.Context, _ int64) error {
	return nil
}

func (s *googleBatchAccountRepoStub) Create(_ context.Context, _ *Account) error { return nil }
func (s *googleBatchAccountRepoStub) GetByCRSAccountID(_ context.Context, _ string) (*Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) FindByExtraField(_ context.Context, _ string, _ any) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListCRSAccountIDs(_ context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) Update(_ context.Context, _ *Account) error { return nil }
func (s *googleBatchAccountRepoStub) Delete(_ context.Context, _ int64) error    { return nil }
func (s *googleBatchAccountRepoStub) List(_ context.Context, _ pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *googleBatchAccountRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string, _ int64, _, _ string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *googleBatchAccountRepoStub) GetStatusSummary(_ context.Context, _ AccountStatusSummaryFilters) (*AccountStatusSummary, error) {
	return &AccountStatusSummary{}, nil
}
func (s *googleBatchAccountRepoStub) ListByGroup(_ context.Context, _ int64) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListActive(_ context.Context) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListByPlatform(_ context.Context, _ string) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) BatchUpdateLastUsed(_ context.Context, _ map[int64]time.Time) error {
	return nil
}
func (s *googleBatchAccountRepoStub) SetError(_ context.Context, _ int64, _ string) error { return nil }
func (s *googleBatchAccountRepoStub) ClearError(_ context.Context, _ int64) error         { return nil }
func (s *googleBatchAccountRepoStub) SetSchedulable(_ context.Context, _ int64, _ bool) error {
	return nil
}
func (s *googleBatchAccountRepoStub) AutoPauseExpiredAccounts(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (s *googleBatchAccountRepoStub) BindGroups(_ context.Context, _ int64, _ []int64) error {
	return nil
}
func (s *googleBatchAccountRepoStub) ListSchedulable(_ context.Context) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListSchedulableByGroupID(_ context.Context, _ int64) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListSchedulableByPlatform(_ context.Context, _ string) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, _ string) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListSchedulableByPlatforms(_ context.Context, _ []string) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, _ int64, _ []string) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListSchedulableUngroupedByPlatform(_ context.Context, _ string) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListSchedulableUngroupedByPlatforms(_ context.Context, _ []string) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) SetRateLimited(_ context.Context, _ int64, _ time.Time) error {
	return nil
}
func (s *googleBatchAccountRepoStub) SetModelRateLimit(_ context.Context, _ int64, _ string, _ time.Time) error {
	return nil
}
func (s *googleBatchAccountRepoStub) SetOverloaded(_ context.Context, _ int64, _ time.Time) error {
	return nil
}
func (s *googleBatchAccountRepoStub) SetTempUnschedulable(_ context.Context, _ int64, _ time.Time, _ string) error {
	return nil
}
func (s *googleBatchAccountRepoStub) ClearTempUnschedulable(_ context.Context, _ int64) error {
	return nil
}
func (s *googleBatchAccountRepoStub) ClearRateLimit(_ context.Context, _ int64) error { return nil }
func (s *googleBatchAccountRepoStub) ClearAntigravityQuotaScopes(_ context.Context, _ int64) error {
	return nil
}
func (s *googleBatchAccountRepoStub) ClearModelRateLimits(_ context.Context, _ int64) error {
	return nil
}
func (s *googleBatchAccountRepoStub) UpdateSessionWindow(_ context.Context, _ int64, _, _ *time.Time, _ string) error {
	return nil
}
func (s *googleBatchAccountRepoStub) UpdateExtra(_ context.Context, _ int64, _ map[string]any) error {
	return nil
}
func (s *googleBatchAccountRepoStub) BulkUpdate(_ context.Context, _ []int64, _ AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (s *googleBatchAccountRepoStub) MarkBlacklisted(_ context.Context, _ int64, _, _ string, _, _ time.Time) error {
	return nil
}
func (s *googleBatchAccountRepoStub) RestoreBlacklisted(_ context.Context, _ int64) error {
	return nil
}
func (s *googleBatchAccountRepoStub) ListBlacklistedIDs(_ context.Context) ([]int64, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) ListBlacklistedForPurge(_ context.Context, _ time.Time, _ int) ([]Account, error) {
	return nil, nil
}
func (s *googleBatchAccountRepoStub) IncrementQuotaUsed(_ context.Context, _ int64, _ float64) error {
	return nil
}
func (s *googleBatchAccountRepoStub) ResetQuotaUsed(_ context.Context, _ int64) error { return nil }

type googleBatchArchiveJobRepoStub struct {
	mu               sync.Mutex
	stateByID        map[int64]string
	successfulClaims int
}

func (s *googleBatchArchiveJobRepoStub) Upsert(_ context.Context, job *GoogleBatchArchiveJob) error {
	if job == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stateByID == nil {
		s.stateByID = map[int64]string{}
	}
	if strings.TrimSpace(job.BillingSettlementState) != "" {
		s.stateByID[job.ID] = strings.TrimSpace(job.BillingSettlementState)
	}
	return nil
}

func (s *googleBatchArchiveJobRepoStub) GetByID(_ context.Context, id int64) (*GoogleBatchArchiveJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &GoogleBatchArchiveJob{ID: id, BillingSettlementState: s.stateByID[id]}, nil
}

func (s *googleBatchArchiveJobRepoStub) GetByPublicBatchName(_ context.Context, _ string) (*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoStub) GetByExecutionBatchName(_ context.Context, _ string) (*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoStub) ListDueForPoll(_ context.Context, _ time.Time, _ int) ([]*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoStub) ListDueForPrefetch(_ context.Context, _ time.Time, _ int) ([]*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoStub) ListExpiredForCleanup(_ context.Context, _ time.Time, _ int) ([]*GoogleBatchArchiveJob, error) {
	return nil, nil
}

func (s *googleBatchArchiveJobRepoStub) TryMarkBillingSettled(_ context.Context, id int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stateByID == nil {
		s.stateByID = map[int64]string{}
	}
	if s.stateByID[id] == GoogleBatchArchiveBillingSettled {
		return false, nil
	}
	s.stateByID[id] = GoogleBatchArchiveBillingSettled
	s.successfulClaims++
	return true, nil
}

func (s *googleBatchArchiveJobRepoStub) TryRestoreBillingPending(_ context.Context, id int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stateByID[id] != GoogleBatchArchiveBillingSettled {
		return false, nil
	}
	s.stateByID[id] = GoogleBatchArchiveBillingPending
	return true, nil
}

func (s *googleBatchArchiveJobRepoStub) TouchLastPublicResultAccess(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (s *googleBatchArchiveJobRepoStub) SoftDelete(_ context.Context, _ int64) error {
	return nil
}

func newGoogleBatchAIStudioTestAccount(id int64) *Account {
	return &Account{
		ID:          id,
		Name:        "gemini-batch-test",
		Platform:    PlatformGemini,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: NormalizeGeminiCredentialsForStorage(AccountTypeAPIKey, map[string]any{
			"api_key":            "test-api-key",
			"gemini_api_variant": GeminiAPIKeyVariantAIStudio,
			"base_url":           "https://generativelanguage.googleapis.com",
		}),
	}
}

func TestForwardAggregatedGoogleListDegradesTimedOutAccountAndKeepsStableOrder(t *testing.T) {
	accounts := []*Account{
		newGoogleBatchAIStudioTestAccount(1),
		newGoogleBatchAIStudioTestAccount(2),
		newGoogleBatchAIStudioTestAccount(3),
	}
	svc := &GeminiMessagesCompatService{
		accountRepo: &googleBatchAccountRepoStub{
			accountsByID: map[int64]*Account{
				1: accounts[0],
				2: accounts[1],
				3: accounts[2],
			},
		},
		httpUpstream: googleBatchHTTPUpstreamFunc(func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
			switch accountID {
			case 1:
				<-req.Context().Done()
				return nil, req.Context().Err()
			case 2:
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{
						"files": [
							{"name": "files/zeta", "source": "second"},
							{"name": "files/alpha", "source": "second"}
						]
					}`)),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{
						"files": [
							{"name": "files/alpha", "source": "third"},
							{"name": "files/beta", "source": "third"}
						]
					}`)),
				}, nil
			}
		}),
		cfg: &config.Config{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, _, err := svc.forwardAggregatedGoogleList(ctx, GoogleBatchForwardInput{
		Method: http.MethodGet,
		Path:   "/v1beta/files",
	}, accounts, googleBatchTargetAIStudio, "files")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "files/alpha", gjson.GetBytes(result.Body, "files.0.name").String())
	require.Equal(t, "second", gjson.GetBytes(result.Body, "files.0.source").String())
	require.Equal(t, "files/beta", gjson.GetBytes(result.Body, "files.1.name").String())
	require.Equal(t, "files/zeta", gjson.GetBytes(result.Body, "files.2.name").String())
}

func TestResolveAIStudioBatchCreateAccountIDRejectsCrossAccountConflict(t *testing.T) {
	svc := &GeminiMessagesCompatService{
		resourceBindingRepo: &googleBatchBindingRepoStub{
			items: map[string]*UpstreamResourceBinding{
				"files/input-a": {ResourceName: "files/input-a", AccountID: 10},
				"files/input-b": {ResourceName: "files/input-b", AccountID: 11},
			},
		},
	}

	_, err := svc.resolveAIStudioBatchCreateAccountID(context.Background(), GoogleBatchForwardInput{
		Method: http.MethodPost,
		Path:   "/v1beta/models/gemini-2.5-flash:batchGenerateContent",
		Body: []byte(`{
			"batch": {
				"input_config": {
					"requests": [
						{"fileName": "files/input-a"},
						{"fileName": "files/input-b"}
					]
				}
			}
		}`),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "different accounts")
}

func TestStoreGoogleBatchArchiveObjectBytesEnrichesFileBindingMetadata(t *testing.T) {
	tempDir := t.TempDir()
	bindingRepo := &googleBatchBindingRepoStub{
		items: map[string]*UpstreamResourceBinding{
			"files/test-result": {
				ResourceKind: UpstreamResourceKindGeminiFile,
				ResourceName: "files/test-result",
				AccountID:    10,
				MetadataJSON: map[string]any{},
			},
		},
	}
	svc := &GeminiMessagesCompatService{
		resourceBindingRepo:       bindingRepo,
		googleBatchArchiveStorage: NewGoogleBatchArchiveStorage(),
	}
	job := &GoogleBatchArchiveJob{
		ID:              7,
		PublicBatchName: "batches/test",
		RequestedModel:  "gemini-2.5-flash",
		CreatedAt:       time.Now().UTC(),
		MetadataJSON: map[string]any{
			googleBatchBindingMetadataPublicProtocol:       UpstreamProviderAIStudio,
			googleBatchBindingMetadataExecutionProtocol:    UpstreamProviderVertexAI,
			googleBatchBindingMetadataPublicResultFileName: "files/test-result",
			googleBatchBindingMetadataRequestedModel:       "gemini-2.5-flash",
			googleBatchBindingMetadataModelFamily:          "gemini_flash",
			googleBatchBindingMetadataEstimatedTokens:      int64(42),
			googleBatchBindingMetadataSourceProtocol:       UpstreamProviderAIStudio,
		},
	}
	object := &GoogleBatchArchiveObject{
		JobID:              job.ID,
		PublicResourceKind: GoogleBatchArchiveResourceKindFile,
		PublicResourceName: "files/test-result",
		IsResultPayload:    true,
	}

	err := svc.storeGoogleBatchArchiveObjectBytes(context.Background(), &GoogleBatchArchiveSettings{
		Enabled:          true,
		LocalStorageRoot: tempDir,
	}, job, object, googleBatchArchiveResultFilename, "application/x-ndjson", []byte(`{"response":{"usageMetadata":{"promptTokenCount":3}}}`+"\n"))
	require.NoError(t, err)

	binding, err := bindingRepo.Get(context.Background(), UpstreamResourceKindGeminiFile, "files/test-result")
	require.NoError(t, err)
	require.NotNil(t, binding)
	require.Equal(t, "gemini-2.5-flash", binding.MetadataJSON[googleBatchBindingMetadataRequestedModel])
	require.Equal(t, "gemini_flash", binding.MetadataJSON[googleBatchBindingMetadataModelFamily])
	require.NotEmpty(t, binding.MetadataJSON[googleBatchBindingMetadataContentDigest])
	require.NotEmpty(t, binding.MetadataJSON[googleBatchBindingMetadataUploadedAt])
}

func TestEnsureGoogleBatchArchiveResultStreamReturnsLocalArchivePayload(t *testing.T) {
	tempDir := t.TempDir()
	settings := &GoogleBatchArchiveSettings{
		Enabled:          true,
		LocalStorageRoot: tempDir,
	}
	svc := &GeminiMessagesCompatService{
		googleBatchArchiveStorage: NewGoogleBatchArchiveStorage(),
	}
	job := &GoogleBatchArchiveJob{
		ID:              9,
		PublicBatchName: "batches/test",
		CreatedAt:       time.Now().UTC(),
	}
	object := &GoogleBatchArchiveObject{
		JobID:              job.ID,
		PublicResourceKind: GoogleBatchArchiveResourceKindFile,
		PublicResourceName: "files/test-result",
		IsResultPayload:    true,
	}
	payload := []byte(`{"response":{"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2}}}` + "\n")
	err := svc.storeGoogleBatchArchiveObjectBytes(context.Background(), settings, job, object, googleBatchArchiveResultFilename, "application/x-ndjson", payload)
	require.NoError(t, err)

	result, err := svc.openGoogleBatchArchiveObjectStreamResult(settings, object, archiveFilenameForPublicResource(object.PublicResourceName, googleBatchArchiveResultFilename))
	require.NoError(t, err)
	require.NotNil(t, result)
	defer func() { _ = result.Body.Close() }()

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.Equal(t, string(payload), string(body))
	require.Equal(t, int64(len(payload)), result.ContentLength)
	require.Equal(t, "application/x-ndjson", result.Headers.Get("Content-Type"))
}

func TestMaybeSettleGoogleBatchArchiveJobFromObjectClaimsOnce(t *testing.T) {
	tempDir := t.TempDir()
	jobRepo := &googleBatchArchiveJobRepoStub{
		stateByID: map[int64]string{42: GoogleBatchArchiveBillingPending},
	}
	svc := &GeminiMessagesCompatService{
		googleBatchArchiveJobRepo: jobRepo,
		googleBatchArchiveStorage: NewGoogleBatchArchiveStorage(),
	}
	job := &GoogleBatchArchiveJob{
		ID:                     42,
		PublicBatchName:        "batches/test",
		RequestedModel:         "gemini-2.5-flash",
		BillingSettlementState: GoogleBatchArchiveBillingPending,
		CreatedAt:              time.Now().UTC(),
	}
	object := &GoogleBatchArchiveObject{
		JobID:              job.ID,
		PublicResourceKind: GoogleBatchArchiveResourceKindFile,
		PublicResourceName: "files/test-result",
		IsResultPayload:    true,
	}
	settings := &GoogleBatchArchiveSettings{
		Enabled:          true,
		LocalStorageRoot: tempDir,
	}
	err := svc.storeGoogleBatchArchiveObjectBytes(context.Background(), settings, job, object, googleBatchArchiveResultFilename, "application/x-ndjson", []byte(`{"response":{"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":1}}}`+"\n"))
	require.NoError(t, err)

	account := &Account{ID: 77, Platform: PlatformGemini, Type: AccountTypeAPIKey}
	input := GoogleBatchForwardInput{
		APIKey:   &APIKey{ID: 3},
		APIKeyID: 3,
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobCopy := *job
			errCh <- svc.maybeSettleGoogleBatchArchiveJobFromObject(context.Background(), input, account, &jobCopy, settings, object)
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	jobRepo.mu.Lock()
	defer jobRepo.mu.Unlock()
	require.Equal(t, 1, jobRepo.successfulClaims)
	require.Equal(t, GoogleBatchArchiveBillingSettled, jobRepo.stateByID[job.ID])
}

func TestGoogleBatchRuntimeMetricsResponseJSONShapeIsStable(t *testing.T) {
	snapshot := SnapshotGoogleBatchRuntimeMetrics()
	body, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(body, "batch_create_success_rate").Exists())
	require.True(t, gjson.GetBytes(body, "overflow_hit_rate").Exists())
	require.True(t, gjson.GetBytes(body, "list_fanout_avg_ms").Exists())
}
