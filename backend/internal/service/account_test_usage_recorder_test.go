//go:build unit

package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemUsageUserRepoStub struct {
	user *User
}

func (s *systemUsageUserRepoStub) Create(_ context.Context, user *User) error {
	if user.ID == 0 {
		user.ID = 7001
	}
	clone := *user
	s.user = &clone
	return nil
}

func (s *systemUsageUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	panic("unexpected GetByID call")
}

func (s *systemUsageUserRepoStub) GetByEmail(_ context.Context, email string) (*User, error) {
	if s.user == nil || s.user.Email != email {
		return nil, ErrUserNotFound
	}
	clone := *s.user
	return &clone, nil
}

func (s *systemUsageUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}

func (s *systemUsageUserRepoStub) Update(context.Context, *User) error {
	panic("unexpected Update call")
}

func (s *systemUsageUserRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *systemUsageUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *systemUsageUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *systemUsageUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *systemUsageUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}

func (s *systemUsageUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *systemUsageUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (s *systemUsageUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *systemUsageUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *systemUsageUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *systemUsageUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *systemUsageUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}

func (s *systemUsageUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

type systemUsageAPIKeyRepoStub struct {
	apiKey *APIKey
}

func (s *systemUsageAPIKeyRepoStub) Create(_ context.Context, key *APIKey) error {
	if key.ID == 0 {
		key.ID = 8001
	}
	clone := *key
	s.apiKey = &clone
	return nil
}

func (s *systemUsageAPIKeyRepoStub) GetByID(context.Context, int64) (*APIKey, error) {
	panic("unexpected GetByID call")
}

func (s *systemUsageAPIKeyRepoStub) GetKeyAndOwnerID(context.Context, int64) (string, int64, error) {
	panic("unexpected GetKeyAndOwnerID call")
}

func (s *systemUsageAPIKeyRepoStub) GetByKey(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKey call")
}

func (s *systemUsageAPIKeyRepoStub) GetByKeyForAuth(context.Context, string) (*APIKey, error) {
	panic("unexpected GetByKeyForAuth call")
}

func (s *systemUsageAPIKeyRepoStub) Update(context.Context, *APIKey) error {
	panic("unexpected Update call")
}

func (s *systemUsageAPIKeyRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *systemUsageAPIKeyRepoStub) ListByUserID(_ context.Context, userID int64, _ pagination.PaginationParams, _ APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	if s.apiKey == nil || s.apiKey.UserID != userID {
		return nil, &pagination.PaginationResult{}, nil
	}
	clone := *s.apiKey
	return []APIKey{clone}, &pagination.PaginationResult{}, nil
}

func (s *systemUsageAPIKeyRepoStub) VerifyOwnership(context.Context, int64, []int64) ([]int64, error) {
	panic("unexpected VerifyOwnership call")
}

func (s *systemUsageAPIKeyRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	panic("unexpected CountByUserID call")
}

func (s *systemUsageAPIKeyRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	panic("unexpected ExistsByKey call")
}

func (s *systemUsageAPIKeyRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}

func (s *systemUsageAPIKeyRepoStub) SearchAPIKeys(context.Context, int64, string, int) ([]APIKey, error) {
	panic("unexpected SearchAPIKeys call")
}

func (s *systemUsageAPIKeyRepoStub) ClearGroupIDByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected ClearGroupIDByGroupID call")
}

func (s *systemUsageAPIKeyRepoStub) UpdateGroupIDByUserAndGroup(context.Context, int64, int64, int64) (int64, error) {
	panic("unexpected UpdateGroupIDByUserAndGroup call")
}

func (s *systemUsageAPIKeyRepoStub) CountByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected CountByGroupID call")
}

func (s *systemUsageAPIKeyRepoStub) ListKeysByUserID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByUserID call")
}

func (s *systemUsageAPIKeyRepoStub) ListKeysByGroupID(context.Context, int64) ([]string, error) {
	panic("unexpected ListKeysByGroupID call")
}

func (s *systemUsageAPIKeyRepoStub) GetAPIKeyGroups(context.Context, int64) ([]APIKeyGroupBinding, error) {
	panic("unexpected GetAPIKeyGroups call")
}

func (s *systemUsageAPIKeyRepoStub) SetAPIKeyGroups(context.Context, int64, []APIKeyGroupBinding) error {
	panic("unexpected SetAPIKeyGroups call")
}

func (s *systemUsageAPIKeyRepoStub) IncrementAPIKeyGroupQuotaUsed(context.Context, int64, int64, float64) error {
	panic("unexpected IncrementAPIKeyGroupQuotaUsed call")
}

func (s *systemUsageAPIKeyRepoStub) IncrementQuotaUsed(context.Context, int64, float64) (float64, error) {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *systemUsageAPIKeyRepoStub) UpdateLastUsed(context.Context, int64, time.Time) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *systemUsageAPIKeyRepoStub) TryReserveImageCount(context.Context, int64, int) (bool, error) {
	panic("unexpected TryReserveImageCount call")
}

func (s *systemUsageAPIKeyRepoStub) RollbackImageCount(context.Context, int64, int) error {
	panic("unexpected RollbackImageCount call")
}

func (s *systemUsageAPIKeyRepoStub) IncrementRateLimitUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementRateLimitUsage call")
}

func (s *systemUsageAPIKeyRepoStub) ResetRateLimitWindows(context.Context, int64) error {
	panic("unexpected ResetRateLimitWindows call")
}

func (s *systemUsageAPIKeyRepoStub) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	panic("unexpected GetRateLimitData call")
}

type systemUsageLogRepoStub struct {
	UsageLogRepository
	logs    []*UsageLog
	byKey   map[string]*UsageLog
	creates int
}

func (s *systemUsageLogRepoStub) Create(_ context.Context, log *UsageLog) (bool, error) {
	s.creates++
	if s.byKey == nil {
		s.byKey = make(map[string]*UsageLog)
	}
	dedupeKey := fmt.Sprintf("%s:%d", log.RequestID, log.APIKeyID)
	if existing, ok := s.byKey[dedupeKey]; ok {
		log.ID = existing.ID
		log.CreatedAt = existing.CreatedAt
		return false, nil
	}
	clone := *log
	if clone.ID == 0 {
		clone.ID = int64(len(s.logs) + 1)
	}
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = time.Now()
	}
	s.logs = append(s.logs, &clone)
	s.byKey[dedupeKey] = &clone
	return true, nil
}

func TestBuildSystemUsageRecordInputFromTest_UsesContextMetadataAndFallbackHTTPStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	ctx := EnsureRequestMetadata(context.Background())
	ctx = WithThinkingEnabled(ctx, true, false)
	SetClaudeRequestedModelRawMetadata(ctx, "claude-sonnet-4.5[1m]")
	SetClaudeRequestedModelNormalizedMetadata(ctx, "claude-sonnet-4.5")
	SetClaudeMillionContextRequestedMetadata(ctx, true)
	SetClaudeMillionContextEffectiveMetadata(ctx, false)
	SetClaudeMillionContextSourceMetadata(ctx, "model_suffix_[1m]")
	SetClaudeMillionContextBetaTokenMetadata(ctx, "beta-1m")
	c.Request = httptest.NewRequest("POST", "/admin/test", nil).WithContext(ctx)
	c.Set(accountTestOpsProbeActionBaseContextKey, UsageOperationTypeBatchTest)
	c.Set(accountTestUpstreamStatusContextKey, 529)

	startedAt := time.Now().Add(-2 * time.Second)
	input, ok := buildSystemUsageRecordInputFromTest(
		c,
		101,
		"claude-sonnet-4.5[1m]",
		accountTestRuntimeMeta{
			RuntimePlatform: PlatformAnthropic,
			InboundEndpoint: EndpointMessages,
			TargetProvider:  PlatformAnthropic,
			TargetModelID:   "claude-sonnet-4.5-upstream",
			ResolvedModelID: "claude-sonnet-4.5",
			SimulatedClient: GatewayClientProfileCodex,
		},
		AccountTestModeHealthCheck,
		startedAt,
		nil,
	)

	require.True(t, ok)
	require.Equal(t, UsageOperationTypeBatchTest, input.OperationType)
	require.Equal(t, int64(101), input.AccountID)
	require.Equal(t, "claude-sonnet-4.5[1m]", input.RequestedModelID)
	require.Equal(t, "claude-sonnet-4.5[1m]", input.RequestedModelRaw)
	require.Equal(t, "claude-sonnet-4.5", input.RequestedModelNorm)
	require.Equal(t, "claude-sonnet-4.5", input.ResolvedModelID)
	require.Equal(t, "claude-sonnet-4.5-upstream", input.UpstreamModel)
	require.Equal(t, EndpointMessages, input.InboundEndpoint)
	require.Equal(t, EndpointMessages, input.UpstreamEndpoint)
	require.Equal(t, PlatformAnthropic, input.UpstreamService)
	require.Equal(t, GatewayClientProfileCodex, input.SimulatedClient)
	require.Equal(t, RequestTypeSync, input.RequestType)
	require.Equal(t, UsageLogStatusSucceeded, input.Status)
	require.NotNil(t, input.MillionRequested)
	require.True(t, *input.MillionRequested)
	require.NotNil(t, input.MillionEffective)
	require.False(t, *input.MillionEffective)
	require.Equal(t, "model_suffix_[1m]", input.MillionSource)
	require.Equal(t, "beta-1m", input.MillionBetaToken)
	require.NotNil(t, input.ThinkingEnabled)
	require.True(t, *input.ThinkingEnabled)
	require.NotNil(t, input.HTTPStatus)
	require.Equal(t, 529, *input.HTTPStatus)
	require.GreaterOrEqual(t, input.DurationMs, int64(1500))
}

func TestBuildSystemUsageRecordInputFromTest_CollectorStatusAndFailureOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	ctx := EnsureRequestMetadata(context.Background())
	c.Request = httptest.NewRequest("POST", "/admin/test", nil).WithContext(ctx)
	c.Set(accountTestOpsProbeActionBaseContextKey, UsageOperationTypeAccountTest)
	c.Set(accountTestOpsCollectorContextKey, &accountTestOpsCollector{
		StartedAt:      time.Now().Add(-3 * time.Second),
		UpstreamStatus: intPtrForSystemUsageTest(503),
		ErrorMessage:   "upstream temporarily unavailable",
	})

	input, ok := buildSystemUsageRecordInputFromTest(
		c,
		202,
		"deepseek-v4-pro",
		accountTestRuntimeMeta{
			RuntimePlatform: PlatformOpenAI,
			InboundEndpoint: "/v1/chat/completions",
			ResolvedModelID: "deepseek-v4-pro",
		},
		AccountTestModeRealForward,
		time.Now().Add(-4*time.Second),
		assertErrForSystemUsageTest("probe failed"),
	)

	require.True(t, ok)
	require.Equal(t, UsageOperationTypeAccountTest, input.OperationType)
	require.Equal(t, UsageLogStatusFailed, input.Status)
	require.Equal(t, RequestTypeStream, input.RequestType)
	require.NotNil(t, input.HTTPStatus)
	require.Equal(t, 503, *input.HTTPStatus)
	require.Equal(t, "upstream temporarily unavailable", input.ErrorMessage)
}

func TestRecordSystemUsageEntry_CreatesSystemSubjectAndPersistsUsageLog(t *testing.T) {
	userRepo := &systemUsageUserRepoStub{}
	apiKeyRepo := &systemUsageAPIKeyRepoStub{}
	usageRepo := &systemUsageLogRepoStub{}
	svc := &AccountTestService{
		userRepo:     userRepo,
		apiKeyRepo:   apiKeyRepo,
		usageLogRepo: usageRepo,
	}

	requested := true
	effective := false
	thinking := true
	reasoning := "high"
	reasoningRaw := "max"
	createdAt := time.Date(2026, 5, 8, 9, 30, 0, 0, time.UTC)

	err := svc.recordSystemUsageEntry(context.Background(), systemUsageRecordInput{
		OperationType:      UsageOperationTypeScheduledTest,
		AccountID:          303,
		RequestedModelID:   "claude-sonnet-4.5[1m]",
		ResolvedModelID:    "claude-sonnet-4.5",
		RequestedModel:     "claude-sonnet-4.5",
		UpstreamModel:      "claude-sonnet-4.5-upstream",
		RequestedModelRaw:  "claude-sonnet-4.5[1m]",
		RequestedModelNorm: "claude-sonnet-4.5",
		MillionRequested:   &requested,
		MillionEffective:   &effective,
		MillionSource:      "model_suffix_[1m]",
		MillionBetaToken:   "beta-1m",
		ThinkingEnabled:    &thinking,
		ReasoningEffort:    &reasoning,
		ReasoningEffortRaw: &reasoningRaw,
		InboundEndpoint:    EndpointMessages,
		UpstreamEndpoint:   EndpointMessages,
		UpstreamURL:        "https://api.anthropic.com/v1/messages",
		UpstreamService:    PlatformAnthropic,
		SimulatedClient:    GatewayClientProfileCodex,
		RequestType:        RequestTypeStream,
		Status:             UsageLogStatusFailed,
		DurationMs:         2345,
		HTTPStatus:         intPtrForSystemUsageTest(429),
		ErrorCode:          "upstream_error",
		ErrorMessage:       "quota exceeded",
		InputTokens:        12,
		OutputTokens:       34,
		TotalCost:          1.25,
		TotalCostUSDEq:     1.25,
		CreatedAt:          createdAt,
		RequestID:          "system-test:scheduled",
	})

	require.NoError(t, err)
	require.NotNil(t, userRepo.user)
	require.Equal(t, systemUsageUserEmail, userRepo.user.Email)
	require.Equal(t, systemUsageUsername, userRepo.user.Username)
	require.Equal(t, RoleUser, userRepo.user.Role)
	require.Equal(t, StatusDisabled, userRepo.user.Status)
	require.True(t, userRepo.user.AdminFreeBilling)
	require.NotNil(t, apiKeyRepo.apiKey)
	require.Equal(t, systemUsageAPIKeyName, apiKeyRepo.apiKey.Name)
	require.Equal(t, StatusAPIKeyDisabled, apiKeyRepo.apiKey.Status)
	require.Len(t, usageRepo.logs, 1)

	log := usageRepo.logs[0]
	require.Equal(t, userRepo.user.ID, log.UserID)
	require.Equal(t, apiKeyRepo.apiKey.ID, log.APIKeyID)
	require.Equal(t, int64(303), log.AccountID)
	require.Equal(t, "system-test:scheduled", log.RequestID)
	require.Equal(t, "claude-sonnet-4.5", log.Model)
	require.Equal(t, "claude-sonnet-4.5", log.RequestedModel)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, "claude-sonnet-4.5-upstream", *log.UpstreamModel)
	require.NotNil(t, log.OperationType)
	require.Equal(t, UsageOperationTypeScheduledTest, *log.OperationType)
	require.Equal(t, BillingTypeBalance, log.BillingType)
	require.NotNil(t, log.BillingExemptReason)
	require.Equal(t, BillingExemptReasonAdminFree, *log.BillingExemptReason)
	require.Equal(t, 1.25, log.TotalCost)
	require.Equal(t, 1.25, log.TotalCostUSDEquivalent)
	require.Zero(t, log.ActualCost)
	require.Equal(t, RequestTypeStream, log.RequestType)
	require.Equal(t, UsageLogStatusFailed, log.Status)
	require.NotNil(t, log.HTTPStatus)
	require.Equal(t, 429, *log.HTTPStatus)
	require.NotNil(t, log.ErrorCode)
	require.Equal(t, "upstream_error", *log.ErrorCode)
	require.NotNil(t, log.ErrorMessage)
	require.Equal(t, "quota exceeded", *log.ErrorMessage)
	require.NotNil(t, log.MillionContextRequested)
	require.True(t, *log.MillionContextRequested)
	require.NotNil(t, log.MillionContextEffective)
	require.False(t, *log.MillionContextEffective)
	require.NotNil(t, log.MillionContextSource)
	require.Equal(t, "model_suffix_[1m]", *log.MillionContextSource)
	require.NotNil(t, log.MillionContextBetaToken)
	require.Equal(t, "beta-1m", *log.MillionContextBetaToken)
	require.NotNil(t, log.ThinkingEnabled)
	require.True(t, *log.ThinkingEnabled)
	require.NotNil(t, log.ReasoningEffort)
	require.Equal(t, "high", *log.ReasoningEffort)
	require.NotNil(t, log.ReasoningEffortRaw)
	require.Equal(t, "max", *log.ReasoningEffortRaw)
	require.NotNil(t, log.ReasoningEffortEffective)
	require.Equal(t, "high", *log.ReasoningEffortEffective)
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, EndpointMessages, *log.InboundEndpoint)
	require.NotNil(t, log.UpstreamURL)
	require.Equal(t, "https://api.anthropic.com/v1/messages", *log.UpstreamURL)
	require.NotNil(t, log.UpstreamService)
	require.Equal(t, PlatformAnthropic, *log.UpstreamService)
	require.NotNil(t, log.SimulatedClient)
	require.Equal(t, GatewayClientProfileCodex, *log.SimulatedClient)
	require.Equal(t, createdAt, log.CreatedAt)
}

func TestRecordSystemUsage_UsesBackgroundResultAndDefaultsMissingValues(t *testing.T) {
	userRepo := &systemUsageUserRepoStub{}
	apiKeyRepo := &systemUsageAPIKeyRepoStub{}
	usageRepo := &systemUsageLogRepoStub{}
	svc := &AccountTestService{
		userRepo:     userRepo,
		apiKeyRepo:   apiKeyRepo,
		usageLogRepo: usageRepo,
	}

	err := svc.recordSystemUsage(context.Background(), ScheduledTestExecutionInput{
		AccountID:     404,
		ModelID:       "gpt-5.4",
		TestMode:      string(AccountTestModeHealthCheck),
		OperationType: UsageOperationTypeBatchTest,
	}, &BackgroundAccountTestResult{
		Status:          "failed",
		ErrorMessage:    "upstream unavailable",
		LatencyMs:       987,
		StartedAt:       time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC),
		FinishedAt:      time.Date(2026, 5, 8, 10, 0, 1, 0, time.UTC),
		ResolvedModelID: "gpt-5.4-mini",
	})

	require.NoError(t, err)
	require.Len(t, usageRepo.logs, 1)

	log := usageRepo.logs[0]
	require.NotNil(t, log.OperationType)
	require.Equal(t, UsageOperationTypeBatchTest, *log.OperationType)
	require.Equal(t, int64(404), log.AccountID)
	require.Equal(t, "gpt-5.4-mini", log.Model)
	require.Equal(t, "gpt-5.4", log.RequestedModel)
	require.Equal(t, RequestTypeSync, log.RequestType)
	require.Equal(t, UsageLogStatusFailed, log.Status)
	require.NotNil(t, log.HTTPStatus)
	require.Equal(t, http.StatusInternalServerError, *log.HTTPStatus)
	require.NotNil(t, log.ErrorMessage)
	require.Equal(t, "upstream unavailable", *log.ErrorMessage)
	require.Equal(t, 0, log.InputTokens)
	require.Equal(t, 0, log.OutputTokens)
	require.Equal(t, 0.0, log.TotalCost)
	require.Equal(t, 0.0, log.TotalCostUSDEquivalent)
	require.Equal(t, 0.0, log.ActualCost)
	require.Nil(t, log.ReasoningEffort)
	require.Nil(t, log.ReasoningEffortRaw)
	require.Nil(t, log.ReasoningEffortEffective)
	require.Nil(t, log.MillionContextRequested)
	require.Nil(t, log.MillionContextEffective)
	require.Nil(t, log.ThinkingEnabled)
}

func TestNormalizeSystemUsageOperationType_AcceptsAllSystemVariants(t *testing.T) {
	require.Equal(t, UsageOperationTypeAccountTest, normalizeSystemUsageOperationType("ACCOUNT_TEST"))
	require.Equal(t, UsageOperationTypeBatchTest, normalizeSystemUsageOperationType(" batch_test "))
	require.Equal(t, UsageOperationTypeScheduledTest, normalizeSystemUsageOperationType("scheduled_test"))
	require.Equal(t, UsageOperationTypeAutoRecoveryTest, normalizeSystemUsageOperationType("auto_recovery_test"))
	require.Empty(t, normalizeSystemUsageOperationType("generate_content"))
}

func intPtrForSystemUsageTest(value int) *int {
	v := value
	return &v
}

func assertErrForSystemUsageTest(message string) error {
	return &systemUsageTestError{message: message}
}

type systemUsageTestError struct {
	message string
}

func (e *systemUsageTestError) Error() string {
	return e.message
}
