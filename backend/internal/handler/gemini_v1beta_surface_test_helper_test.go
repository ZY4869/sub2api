//go:build unit

package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type geminiSurfaceUsageLogRepoStub struct {
	service.UsageLogRepository

	bestEffortCalls int
	createCalls     int
	lastLog         *service.UsageLog
}

func (s *geminiSurfaceUsageLogRepoStub) CreateBestEffort(_ context.Context, log *service.UsageLog) error {
	s.bestEffortCalls++
	s.lastLog = log
	return nil
}

func (s *geminiSurfaceUsageLogRepoStub) Create(_ context.Context, log *service.UsageLog) (bool, error) {
	s.createCalls++
	s.lastLog = log
	return true, nil
}

type geminiSurfaceHTTPResponse struct {
	statusCode int
	body       string
	headers    http.Header
	err        error
}

type geminiSurfaceHTTPUpstreamRecorder struct {
	name                   string
	response               geminiSurfaceHTTPResponse
	calls                  int
	lastReq                *http.Request
	lastBody               []byte
	lastProxyURL           string
	lastAccountID          int64
	lastAccountConcurrency int
}

func (r *geminiSurfaceHTTPUpstreamRecorder) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	r.calls++
	r.lastProxyURL = proxyURL
	r.lastAccountID = accountID
	r.lastAccountConcurrency = accountConcurrency

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	r.lastBody = append([]byte(nil), body...)

	clonedReq := req.Clone(req.Context())
	clonedReq.Header = req.Header.Clone()
	clonedReq.Body = io.NopCloser(bytes.NewReader(body))
	r.lastReq = clonedReq

	if r.response.err != nil {
		return nil, r.response.err
	}

	headers := r.response.headers.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}

	respBody := r.response.body
	return &http.Response{
		StatusCode:    r.response.statusCode,
		Header:        headers,
		Body:          io.NopCloser(strings.NewReader(respBody)),
		ContentLength: int64(len(respBody)),
	}, nil
}

func (r *geminiSurfaceHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return r.Do(req, proxyURL, accountID, accountConcurrency)
}

type geminiSurfaceAccountRepoStub struct {
	service.AccountRepository
	accounts                     []service.Account
	accountsByID                 map[int64]*service.Account
	listByGroupAndPlatformCalls  int
	listByGroupAndPlatformsCalls int
	listByPlatformsCalls         int
	getByIDCalls                 int
	setErrorCalls                int
	lastListGroupID              int64
	lastListPlatform             string
	lastListPlatforms            []string
}

func newGeminiSurfaceAccountRepoStub(accounts []service.Account) *geminiSurfaceAccountRepoStub {
	cloned := make([]service.Account, len(accounts))
	copy(cloned, accounts)
	accountsByID := make(map[int64]*service.Account, len(cloned))
	for i := range cloned {
		accountsByID[cloned[i].ID] = &cloned[i]
	}
	return &geminiSurfaceAccountRepoStub{
		accounts:     cloned,
		accountsByID: accountsByID,
	}
}

func (s *geminiSurfaceAccountRepoStub) GetByID(_ context.Context, id int64) (*service.Account, error) {
	s.getByIDCalls++
	if account, ok := s.accountsByID[id]; ok {
		return account, nil
	}
	return nil, service.ErrAccountNotFound
}

func (s *geminiSurfaceAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, groupID int64, platform string) ([]service.Account, error) {
	s.listByGroupAndPlatformCalls++
	s.lastListGroupID = groupID
	s.lastListPlatform = platform
	return s.filterAccounts(&groupID, []string{platform}), nil
}

func (s *geminiSurfaceAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, platforms []string) ([]service.Account, error) {
	s.listByGroupAndPlatformsCalls++
	s.lastListGroupID = groupID
	s.lastListPlatforms = append([]string(nil), platforms...)
	return s.filterAccounts(&groupID, platforms), nil
}

func (s *geminiSurfaceAccountRepoStub) ListSchedulableByPlatforms(_ context.Context, platforms []string) ([]service.Account, error) {
	s.listByPlatformsCalls++
	s.lastListPlatforms = append([]string(nil), platforms...)
	return s.filterAccounts(nil, platforms), nil
}

func (s *geminiSurfaceAccountRepoStub) SetError(_ context.Context, id int64, errorMsg string) error {
	s.setErrorCalls++
	if account, ok := s.accountsByID[id]; ok {
		account.ErrorMessage = errorMsg
	}
	return nil
}

func (s *geminiSurfaceAccountRepoStub) filterAccounts(groupID *int64, platforms []string) []service.Account {
	platformSet := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		platformSet[platform] = struct{}{}
	}

	filtered := make([]service.Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		if !account.IsSchedulable() {
			continue
		}
		if _, ok := platformSet[account.Platform]; !ok {
			continue
		}
		if groupID != nil {
			inGroup := false
			for _, binding := range account.AccountGroups {
				if binding.GroupID == *groupID {
					inGroup = true
					break
				}
			}
			if !inGroup {
				continue
			}
		}
		filtered = append(filtered, account)
	}
	return filtered
}

type geminiSurfaceGroupRepoStub struct {
	service.GroupRepository
	group            *service.Group
	getByIDCalls     int
	getByIDLiteCalls int
}

func (s *geminiSurfaceGroupRepoStub) GetByID(_ context.Context, id int64) (*service.Group, error) {
	s.getByIDCalls++
	if s.group != nil && s.group.ID == id {
		return s.group, nil
	}
	return nil, service.ErrGroupNotFound
}

func (s *geminiSurfaceGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*service.Group, error) {
	s.getByIDLiteCalls++
	if s.group != nil && s.group.ID == id {
		return s.group, nil
	}
	return nil, service.ErrGroupNotFound
}

type geminiSurfaceConcurrencyCacheStub struct{}

func (s *geminiSurfaceConcurrencyCacheStub) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) ReleaseAccountSlot(context.Context, int64, string) error {
	return nil
}

func (s *geminiSurfaceConcurrencyCacheStub) GetAccountConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) GetAccountConcurrencyBatch(_ context.Context, accountIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(accountIDs))
	for _, accountID := range accountIDs {
		result[accountID] = 0
	}
	return result, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) IncrementAccountWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) DecrementAccountWaitCount(context.Context, int64) error {
	return nil
}

func (s *geminiSurfaceConcurrencyCacheStub) GetAccountWaitingCount(context.Context, int64) (int, error) {
	return 0, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) AcquireUserSlot(context.Context, int64, int, string) (bool, error) {
	return true, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) ReleaseUserSlot(context.Context, int64, string) error {
	return nil
}

func (s *geminiSurfaceConcurrencyCacheStub) GetUserConcurrency(context.Context, int64) (int, error) {
	return 0, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) IncrementWaitCount(context.Context, int64, int) (bool, error) {
	return true, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) DecrementWaitCount(context.Context, int64) error {
	return nil
}

func (s *geminiSurfaceConcurrencyCacheStub) GetAccountsLoadBatch(context.Context, []service.AccountWithConcurrency) (map[int64]*service.AccountLoadInfo, error) {
	return map[int64]*service.AccountLoadInfo{}, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) GetUsersLoadBatch(context.Context, []service.UserWithConcurrency) (map[int64]*service.UserLoadInfo, error) {
	return map[int64]*service.UserLoadInfo{}, nil
}

func (s *geminiSurfaceConcurrencyCacheStub) CleanupExpiredAccountSlots(context.Context, int64) error {
	return nil
}

func (s *geminiSurfaceConcurrencyCacheStub) CleanupStaleProcessSlots(context.Context, string) error {
	return nil
}

type geminiSurfaceFixture struct {
	t                    *testing.T
	cfg                  *config.Config
	handler              *GatewayHandler
	group                *service.Group
	apiKey               *service.APIKey
	accountRepo          *geminiSurfaceAccountRepoStub
	groupRepo            *geminiSurfaceGroupRepoStub
	nativeRecorder       *geminiSurfaceHTTPUpstreamRecorder
	compatRecorder       *geminiSurfaceHTTPUpstreamRecorder
	liveRecorder         *geminiSurfaceHTTPUpstreamRecorder
	interactionsRecorder *geminiSurfaceHTTPUpstreamRecorder
	billingCache         *service.BillingCacheService
	usageRepo            *geminiSurfaceUsageLogRepoStub
}

func newGeminiSurfaceFixture(t *testing.T) *geminiSurfaceFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	groupID := int64(7001)
	userID := int64(8001)
	accountID := int64(9001)

	group := &service.Group{
		ID:       groupID,
		Name:     "gemini-tests",
		Platform: service.PlatformGemini,
		Status:   service.StatusActive,
		Hydrated: true,
	}
	user := &service.User{
		ID:          userID,
		Role:        service.RoleUser,
		Status:      service.StatusActive,
		Balance:     100,
		Concurrency: 2,
	}
	account := service.Account{
		ID:          accountID,
		Name:        "gemini-apikey",
		Platform:    service.PlatformGemini,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{
			"api_key":                    "gemini-test-key",
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(http.StatusForbidden), float64(http.StatusInternalServerError)},
		},
		AccountGroups: []service.AccountGroup{{AccountID: accountID, GroupID: groupID}},
	}

	accountRepo := newGeminiSurfaceAccountRepoStub([]service.Account{account})
	groupRepo := &geminiSurfaceGroupRepoStub{group: group}
	cfg := &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{
			GeminiLiveEnabled:         true,
			GeminiInteractionsEnabled: true,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
	}

	nativeRecorder := &geminiSurfaceHTTPUpstreamRecorder{name: "native"}
	compatRecorder := &geminiSurfaceHTTPUpstreamRecorder{name: "compat"}
	liveRecorder := &geminiSurfaceHTTPUpstreamRecorder{name: "live"}
	interactionsRecorder := &geminiSurfaceHTTPUpstreamRecorder{name: "interactions"}
	usageRepo := &geminiSurfaceUsageLogRepoStub{}

	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
	concurrencyService := service.NewConcurrencyService(&geminiSurfaceConcurrencyCacheStub{})
	gatewayService := service.NewGatewayService(
		accountRepo,
		groupRepo,
		usageRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		cfg,
		nil,
		concurrencyService,
		nil,
		nil,
		billingCache,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	nativeCompat := service.NewGeminiMessagesCompatService(accountRepo, groupRepo, nil, nil, nil, service.NewRateLimitService(accountRepo, nil, cfg, nil, nil), nativeRecorder, nil, cfg)
	compatCompat := service.NewGeminiMessagesCompatService(accountRepo, groupRepo, nil, nil, nil, nil, compatRecorder, nil, cfg)
	liveCompat := service.NewGeminiMessagesCompatService(accountRepo, groupRepo, nil, nil, nil, nil, liveRecorder, nil, cfg)
	interactionsCompat := service.NewGeminiMessagesCompatService(accountRepo, groupRepo, nil, nil, nil, nil, interactionsRecorder, nil, cfg)

	apiKey := &service.APIKey{
		ID:      6001,
		UserID:  userID,
		Key:     "sk-gemini-test",
		GroupID: &groupID,
		Status:  service.StatusActive,
		User:    user,
		Group:   group,
	}

	handler := &GatewayHandler{
		gatewayService:            gatewayService,
		geminiNativeService:       service.NewGeminiNativeGatewayService(nativeCompat),
		geminiCompatService:       service.NewGeminiCompatGatewayService(compatCompat),
		geminiLiveService:         service.NewGeminiLiveGatewayService(liveCompat),
		geminiInteractionsService: service.NewGeminiInteractionsGatewayService(interactionsCompat),
		billingCacheService:       billingCache,
		concurrencyHelper:         NewConcurrencyHelper(concurrencyService, SSEPingFormatNone, 0),
		maxAccountSwitchesGemini:  1,
		cfg:                       cfg,
	}

	t.Cleanup(func() {
		billingCache.Stop()
	})

	return &geminiSurfaceFixture{
		t:                    t,
		cfg:                  cfg,
		handler:              handler,
		group:                group,
		apiKey:               apiKey,
		accountRepo:          accountRepo,
		groupRepo:            groupRepo,
		nativeRecorder:       nativeRecorder,
		compatRecorder:       compatRecorder,
		liveRecorder:         liveRecorder,
		interactionsRecorder: interactionsRecorder,
		billingCache:         billingCache,
		usageRepo:            usageRepo,
	}
}

func (f *geminiSurfaceFixture) newContext(method, path string, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	f.t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Group, f.group))
	c.Request = req
	c.Params = params
	c.Set(ctxKeyInboundEndpoint, NormalizeInboundEndpoint(path))
	c.Set(string(servermiddleware.ContextKeyAPIKey), f.apiKey)
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{
		UserID:      f.apiKey.UserID,
		Concurrency: f.apiKey.User.Concurrency,
	})
	return c, recorder
}

func (f *geminiSurfaceFixture) requireOnlyRecorderHit(expected *geminiSurfaceHTTPUpstreamRecorder) {
	f.t.Helper()
	recorders := []*geminiSurfaceHTTPUpstreamRecorder{
		f.nativeRecorder,
		f.compatRecorder,
		f.liveRecorder,
		f.interactionsRecorder,
	}
	for _, recorder := range recorders {
		want := 0
		if recorder == expected {
			want = 1
		}
		require.Equalf(f.t, want, recorder.calls, "unexpected call count for %s surface", recorder.name)
	}
}

type geminiSurfaceErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Type   string `json:"@type"`
			Reason string `json:"reason"`
		} `json:"details"`
	} `json:"error"`
}
