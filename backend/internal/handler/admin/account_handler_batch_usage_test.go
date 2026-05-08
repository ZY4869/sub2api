//go:build unit

package admin

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type batchUsageAccountRepo struct {
	service.AccountRepository
	account *service.Account
}

func (r *batchUsageAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if r.account != nil && r.account.ID == id {
		clone := *r.account
		return &clone, nil
	}
	return nil, service.ErrAccountNotFound
}

type batchUsageUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r *batchUsageUserRepo) Create(_ context.Context, user *service.User) error {
	if user.ID == 0 {
		user.ID = 9101
	}
	clone := *user
	r.user = &clone
	return nil
}

func (r *batchUsageUserRepo) GetByEmail(_ context.Context, email string) (*service.User, error) {
	if r.user == nil || r.user.Email != email {
		return nil, service.ErrUserNotFound
	}
	clone := *r.user
	return &clone, nil
}

type batchUsageAPIKeyRepo struct {
	service.APIKeyRepository
	apiKey *service.APIKey
}

func (r *batchUsageAPIKeyRepo) Create(_ context.Context, key *service.APIKey) error {
	if key.ID == 0 {
		key.ID = 9201
	}
	clone := *key
	r.apiKey = &clone
	return nil
}

func (r *batchUsageAPIKeyRepo) ListByUserID(_ context.Context, userID int64, _ pagination.PaginationParams, _ service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	if r.apiKey == nil || r.apiKey.UserID != userID {
		return nil, &pagination.PaginationResult{}, nil
	}
	clone := *r.apiKey
	return []service.APIKey{clone}, &pagination.PaginationResult{}, nil
}

type batchUsageLogRepo struct {
	service.UsageLogRepository
	logs []*service.UsageLog
}

func (r *batchUsageLogRepo) Create(_ context.Context, log *service.UsageLog) (bool, error) {
	clone := *log
	if clone.ID == 0 {
		clone.ID = int64(len(r.logs) + 1)
	}
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = time.Now()
	}
	r.logs = append(r.logs, &clone)
	return true, nil
}

type batchUsageHTTPUpstream struct {
	response *http.Response
}

func (u *batchUsageHTTPUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return u.response, nil
}

func (u *batchUsageHTTPUpstream) DoWithTLS(_ *http.Request, _ string, _ int64, _ int, _ *service.TLSFingerprintProfile) (*http.Response, error) {
	return u.response, nil
}

func TestAccountHandlerBatchTest_RealChainRecordsUsageLogWithBatchOperationType(t *testing.T) {
	account := service.Account{
		ID:          131,
		Name:        "openai-131",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "test-token",
			"base_url": "https://api.openai.com",
		},
	}
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{account}

	accountRepo := &batchUsageAccountRepo{account: &account}
	userRepo := &batchUsageUserRepo{}
	apiKeyRepo := &batchUsageAPIKeyRepo{}
	usageRepo := &batchUsageLogRepo{}
	upstream := &batchUsageHTTPUpstream{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n")),
		},
	}

	accountTestSvc := service.NewAccountTestService(accountRepo, nil, nil, nil, upstream, &config.Config{})
	accountTestSvc.SetUsageLogDependencies(userRepo, apiKeyRepo, usageRepo)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-test", handler.BatchTest)

	body := []byte(`{"account_ids":[131],"model_input_mode":"manual","manual_model_id":"gpt-5.4","source_protocol":"openai","test_mode":"real_forward"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, usageRepo.logs, 1)
	log := usageRepo.logs[0]
	require.NotNil(t, log.OperationType)
	require.Equal(t, service.UsageOperationTypeBatchTest, *log.OperationType)
	require.Equal(t, int64(131), log.AccountID)
	require.Equal(t, userRepo.user.ID, log.UserID)
	require.Equal(t, apiKeyRepo.apiKey.ID, log.APIKeyID)
	require.Equal(t, "gpt-5.4", log.RequestedModel)
	require.Equal(t, "gpt-5.4", log.Model)
	require.Equal(t, service.RequestTypeStream, log.RequestType)
	require.Equal(t, service.UsageLogStatusSucceeded, log.Status)
	require.Zero(t, log.ActualCost)
	require.NotNil(t, log.BillingExemptReason)
	require.Equal(t, service.BillingExemptReasonAdminFree, *log.BillingExemptReason)
	require.Contains(t, rec.Body.String(), `"status":"success"`)
}
