package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/gin-gonic/gin"
)

const (
	systemUsageUserEmail    = "system-usage@sub2api.local"
	systemUsageUsername     = "system_usage"
	systemUsageAPIKeyName   = "system-usage"
	systemUsageRequestIDTag = "system-test"
)

type systemUsageSubject struct {
	user   *User
	apiKey *APIKey
}

type systemUsageRecordInput struct {
	OperationType       string
	AccountID           int64
	RequestedModelID    string
	ResolvedModelID     string
	RequestedModel      string
	UpstreamModel       string
	RequestedModelRaw   string
	RequestedModelNorm  string
	MillionRequested    *bool
	MillionEffective    *bool
	MillionSource       string
	MillionBetaToken    string
	ThinkingEnabled     *bool
	ReasoningEffort     *string
	ReasoningEffortRaw  *string
	ReasoningEffortEff  *string
	InboundEndpoint     string
	UpstreamEndpoint    string
	UpstreamURL         string
	UpstreamService     string
	SimulatedClient     string
	RequestType         RequestType
	Status              string
	DurationMs          int64
	HTTPStatus          *int
	ErrorCode           string
	ErrorMessage        string
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	CacheCreation5m     int
	CacheCreation1h     int
	TotalCost           float64
	TotalCostUSDEq      float64
	CreatedAt           time.Time
	RequestID           string
}

func normalizeSystemUsageOperationType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case UsageOperationTypeAccountTest:
		return UsageOperationTypeAccountTest
	case UsageOperationTypeBatchTest:
		return UsageOperationTypeBatchTest
	case UsageOperationTypeScheduledTest:
		return UsageOperationTypeScheduledTest
	case UsageOperationTypeAutoRecoveryTest:
		return UsageOperationTypeAutoRecoveryTest
	default:
		return ""
	}
}

func (s *AccountTestService) shouldRecordSystemUsage(operationType string) bool {
	if s == nil {
		return false
	}
	if normalizeSystemUsageOperationType(operationType) == "" {
		return false
	}
	return s.userRepo != nil && s.apiKeyRepo != nil && s.usageLogRepo != nil
}

func (s *AccountTestService) ensureSystemUsageSubject(ctx context.Context) (*systemUsageSubject, error) {
	if s == nil || s.userRepo == nil || s.apiKeyRepo == nil {
		return nil, nil
	}

	user, err := s.userRepo.GetByEmail(ctx, systemUsageUserEmail)
	if err != nil {
		if err != ErrUserNotFound {
			return nil, err
		}
		user = &User{
			Email:            systemUsageUserEmail,
			Username:         systemUsageUsername,
			Role:             RoleUser,
			Status:           StatusDisabled,
			AdminFreeBilling: true,
			Concurrency:      1,
		}
		if setErr := user.SetPassword(randomSystemUsageSecret()); setErr != nil {
			return nil, setErr
		}
		if createErr := s.userRepo.Create(ctx, user); createErr != nil {
			if !isSystemUsageConflict(createErr) {
				return nil, createErr
			}
			user, err = s.userRepo.GetByEmail(ctx, systemUsageUserEmail)
			if err != nil {
				return nil, err
			}
		}
	}

	apiKeys, _, err := s.apiKeyRepo.ListByUserID(ctx, user.ID, defaultSystemUsagePageParams(), APIKeyListFilters{})
	if err != nil {
		return nil, err
	}
	for index := range apiKeys {
		if strings.EqualFold(strings.TrimSpace(apiKeys[index].Name), systemUsageAPIKeyName) {
			return &systemUsageSubject{user: user, apiKey: &apiKeys[index]}, nil
		}
	}

	apiKey := &APIKey{
		UserID: user.ID,
		Key:    generateSystemUsageAPIKey(),
		Name:   systemUsageAPIKeyName,
		Status: StatusAPIKeyDisabled,
	}
	if createErr := s.apiKeyRepo.Create(ctx, apiKey); createErr != nil {
		if !isSystemUsageConflict(createErr) {
			return nil, createErr
		}
		apiKeys, _, err = s.apiKeyRepo.ListByUserID(ctx, user.ID, defaultSystemUsagePageParams(), APIKeyListFilters{})
		if err != nil {
			return nil, err
		}
		for index := range apiKeys {
			if strings.EqualFold(strings.TrimSpace(apiKeys[index].Name), systemUsageAPIKeyName) {
				return &systemUsageSubject{user: user, apiKey: &apiKeys[index]}, nil
			}
		}
		return nil, fmt.Errorf("system usage api key conflict but could not be reloaded")
	}

	return &systemUsageSubject{user: user, apiKey: apiKey}, nil
}

func defaultSystemUsagePageParams() pagination.PaginationParams {
	return pagination.PaginationParams{Page: 1, PageSize: 100}
}

func randomSystemUsageSecret() string {
	return "system-usage-" + systemUsageRandomHex(24)
}

func generateSystemUsageAPIKey() string {
	return "sk-system-usage-" + systemUsageRandomHex(16)
}

func systemUsageRandomHex(size int) string {
	if size <= 0 {
		size = 8
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func isSystemUsageConflict(err error) bool {
	return err == ErrEmailExists || err == ErrAPIKeyExists
}

func (s *AccountTestService) recordSystemUsage(ctx context.Context, input ScheduledTestExecutionInput, result *BackgroundAccountTestResult) error {
	operationType := normalizeSystemUsageOperationType(input.OperationType)
	if !s.shouldRecordSystemUsage(operationType) || result == nil || input.AccountID <= 0 {
		return nil
	}

	recordInput := systemUsageRecordInput{
		OperationType:    operationType,
		AccountID:        input.AccountID,
		RequestedModelID: strings.TrimSpace(input.ModelID),
		ResolvedModelID:  strings.TrimSpace(result.ResolvedModelID),
		RequestedModel:   strings.TrimSpace(input.ModelID),
		UpstreamModel:    strings.TrimSpace(result.ResolvedModelID),
		RequestType:      requestTypeForSystemTestMode(input.TestMode),
		Status:           systemUsageStatusFromResult(result.Status),
		DurationMs:       result.LatencyMs,
		ErrorMessage:     strings.TrimSpace(result.ErrorMessage),
		CreatedAt:        firstNonZeroTime(result.FinishedAt, result.StartedAt, time.Now()),
	}
	if recordInput.ErrorMessage != "" {
		statusCode := http.StatusInternalServerError
		recordInput.HTTPStatus = &statusCode
	}
	return s.recordSystemUsageEntry(ctx, recordInput)
}

func (s *AccountTestService) recordSystemUsageForTestConnection(
	c *gin.Context,
	accountID int64,
	requestedModelID string,
	runtimeMeta accountTestRuntimeMeta,
	testMode AccountTestMode,
	startedAt time.Time,
	testErr error,
) {
	if c == nil || c.Request == nil {
		return
	}
	recordInput, ok := buildSystemUsageRecordInputFromTest(c, accountID, requestedModelID, runtimeMeta, testMode, startedAt, testErr)
	if ok {
		if err := s.recordSystemUsageEntry(c.Request.Context(), recordInput); err != nil {
			// best-effort only; do not affect admin test result
			return
		}
		return
	}

	operationType := operationTypeFromSystemTestContext(c)
	if operationType == "" {
		return
	}

	result := &BackgroundAccountTestResult{
		Status:          "success",
		ErrorMessage:    "",
		LatencyMs:       time.Since(startedAt).Milliseconds(),
		StartedAt:       startedAt,
		FinishedAt:      time.Now(),
		ResolvedModelID: strings.TrimSpace(runtimeMeta.ResolvedModelID),
	}
	if testErr != nil {
		result.Status = "failed"
		result.ErrorMessage = strings.TrimSpace(testErr.Error())
	}

	if err := s.recordSystemUsage(c.Request.Context(), ScheduledTestExecutionInput{
		AccountID:     accountID,
		ModelID:       strings.TrimSpace(requestedModelID),
		TestMode:      string(testMode),
		OperationType: operationType,
	}, result); err != nil {
		// best-effort only; do not affect admin test result
		return
	}
}

func (s *AccountTestService) recordSystemUsageEntry(ctx context.Context, input systemUsageRecordInput) error {
	operationType := normalizeSystemUsageOperationType(input.OperationType)
	if !s.shouldRecordSystemUsage(operationType) || input.AccountID <= 0 {
		return nil
	}

	subject, err := s.ensureSystemUsageSubject(ctx)
	if err != nil {
		return err
	}
	if subject == nil || subject.user == nil || subject.apiKey == nil {
		return nil
	}

	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		requestID = fmt.Sprintf("%s:%s:%d:%d", systemUsageRequestIDTag, operationType, input.AccountID, input.CreatedAt.UTC().UnixNano())
	}
	model := firstNonEmptyString(
		strings.TrimSpace(input.ResolvedModelID),
		strings.TrimSpace(input.RequestedModelID),
		strings.TrimSpace(input.RequestedModel),
		"-",
	)
	requestedModel := firstNonEmptyString(
		strings.TrimSpace(input.RequestedModel),
		strings.TrimSpace(input.RequestedModelID),
		model,
	)
	upstreamModel := strings.TrimSpace(input.UpstreamModel)
	legacyReasoningEffort, reasoningEffortRaw, reasoningEffortEffective := NormalizeGatewayEffortForUsage(
		input.ReasoningEffort,
		input.ReasoningEffortRaw,
		input.ReasoningEffortEff,
	)

	log := &UsageLog{
		UserID:                   subject.user.ID,
		APIKeyID:                 subject.apiKey.ID,
		AccountID:                input.AccountID,
		RequestID:                requestID,
		Model:                    model,
		RequestedModel:           requestedModel,
		UpstreamModel:            optionalNonEqualStringPtr(upstreamModel, model),
		ReasoningEffort:          legacyReasoningEffort,
		ReasoningEffortRaw:       reasoningEffortRaw,
		ReasoningEffortEffective: reasoningEffortEffective,
		RequestedModelRaw:        optionalTrimmedStringPtr(input.RequestedModelRaw),
		RequestedModelNormalized: optionalTrimmedStringPtr(input.RequestedModelNorm),
		MillionContextRequested:  input.MillionRequested,
		MillionContextEffective:  input.MillionEffective,
		MillionContextSource:     optionalTrimmedStringPtr(input.MillionSource),
		MillionContextBetaToken:  optionalTrimmedStringPtr(input.MillionBetaToken),
		ThinkingEnabled:          input.ThinkingEnabled,
		InboundEndpoint:          optionalTrimmedStringPtr(input.InboundEndpoint),
		UpstreamEndpoint:         optionalTrimmedStringPtr(input.UpstreamEndpoint),
		UpstreamURL:              optionalTrimmedStringPtr(input.UpstreamURL),
		UpstreamService:          optionalTrimmedStringPtr(input.UpstreamService),
		InputTokens:              input.InputTokens,
		OutputTokens:             input.OutputTokens,
		CacheCreationTokens:      input.CacheCreationTokens,
		CacheReadTokens:          input.CacheReadTokens,
		CacheCreation5mTokens:    input.CacheCreation5m,
		CacheCreation1hTokens:    input.CacheCreation1h,
		TotalCost:                input.TotalCost,
		ActualCost:               0,
		BillingCurrency:          NormalizeUsageBillingCurrency(""),
		TotalCostUSDEquivalent:   input.TotalCostUSDEq,
		BillingExemptReason:      BillingExemptReasonPtr(BillingExemptReasonAdminFree),
		BillingType:              BillingTypeBalance,
		RequestType:              input.RequestType.Normalize(),
		Status:                   NormalizeUsageLogStatus(input.Status),
		DurationMs:               systemUsageIntPtrFromInt64(input.DurationMs),
		HTTPStatus:               input.HTTPStatus,
		ErrorCode:                systemUsageStringPtr(input.ErrorCode),
		ErrorMessage:             systemUsageStringPtr(input.ErrorMessage),
		SimulatedClient:          NormalizeUsageLogSimulatedClient(input.SimulatedClient),
		OperationType:            systemUsageStringPtr(operationType),
		CreatedAt:                firstNonZeroTime(input.CreatedAt, time.Now()),
	}
	if log.DurationMs == nil {
		zero := 0
		log.DurationMs = &zero
	}

	_, err = s.usageLogRepo.Create(ctx, log)
	return err
}

func buildSystemUsageRecordInputFromTest(
	c *gin.Context,
	accountID int64,
	requestedModelID string,
	runtimeMeta accountTestRuntimeMeta,
	testMode AccountTestMode,
	startedAt time.Time,
	testErr error,
) (systemUsageRecordInput, bool) {
	operationType := operationTypeFromSystemTestContext(c)
	if accountID <= 0 || operationType == "" {
		return systemUsageRecordInput{}, false
	}
	ctx := c.Request.Context()
	collector := readSystemUsageOpsCollector(c)
	durationMs := time.Since(startedAt).Milliseconds()
	recordInput := systemUsageRecordInput{
		OperationType:    operationType,
		AccountID:        accountID,
		RequestedModelID: strings.TrimSpace(requestedModelID),
		ResolvedModelID:  strings.TrimSpace(runtimeMeta.ResolvedModelID),
		RequestedModel:   firstNonEmptyString(strings.TrimSpace(requestedModelID), strings.TrimSpace(runtimeMeta.ResolvedModelID)),
		UpstreamModel:    firstNonEmptyString(strings.TrimSpace(runtimeMeta.TargetModelID), strings.TrimSpace(runtimeMeta.ResolvedModelID)),
		InboundEndpoint:  strings.TrimSpace(runtimeMeta.InboundEndpoint),
		UpstreamEndpoint: strings.TrimSpace(runtimeMeta.InboundEndpoint),
		UpstreamService:  firstNonEmptyString(strings.TrimSpace(runtimeMeta.TargetProvider), strings.TrimSpace(runtimeMeta.RuntimePlatform)),
		SimulatedClient:  strings.TrimSpace(runtimeMeta.SimulatedClient),
		RequestType:      requestTypeForSystemTestMode(string(testMode)),
		Status:           UsageLogStatusSucceeded,
		DurationMs:       durationMs,
		CreatedAt:        time.Now(),
	}
	if raw, ok := ClaudeRequestedModelRawMetadataFromContext(ctx); ok {
		recordInput.RequestedModelRaw = raw
	}
	if normalized, ok := ClaudeRequestedModelNormalizedMetadataFromContext(ctx); ok {
		recordInput.RequestedModelNorm = normalized
		if recordInput.RequestedModel == "" {
			recordInput.RequestedModel = normalized
		}
	}
	if requested, ok := ClaudeMillionContextRequestedMetadataFromContext(ctx); ok {
		recordInput.MillionRequested = systemUsageBoolPtr(requested)
	}
	if effective, ok := ClaudeMillionContextEffectiveMetadataFromContext(ctx); ok {
		recordInput.MillionEffective = systemUsageBoolPtr(effective)
	}
	if source, ok := ClaudeMillionContextSourceMetadataFromContext(ctx); ok {
		recordInput.MillionSource = source
	}
	if betaToken, ok := ClaudeMillionContextBetaTokenMetadataFromContext(ctx); ok {
		recordInput.MillionBetaToken = betaToken
	}
	if enabled, ok := ThinkingEnabledFromContext(ctx); ok {
		recordInput.ThinkingEnabled = systemUsageBoolPtr(enabled)
	}
	if collector != nil {
		recordInput.CreatedAt = firstNonZeroTime(collector.StartedAt, time.Now())
		if collector.UpstreamStatus != nil {
			statusCode := *collector.UpstreamStatus
			recordInput.HTTPStatus = &statusCode
		}
		if strings.TrimSpace(collector.ErrorMessage) != "" {
			recordInput.ErrorMessage = strings.TrimSpace(collector.ErrorMessage)
		}
	}
	if recordInput.HTTPStatus == nil {
		if statusCode, ok := readSystemUsageUpstreamStatus(c); ok {
			recordInput.HTTPStatus = &statusCode
		}
	}
	if testErr != nil {
		recordInput.Status = UsageLogStatusFailed
		if recordInput.ErrorMessage == "" {
			recordInput.ErrorMessage = strings.TrimSpace(testErr.Error())
		}
	}
	if recordInput.HTTPStatus == nil && testErr != nil {
		statusCode := http.StatusInternalServerError
		recordInput.HTTPStatus = &statusCode
	}
	return recordInput, true
}

func operationTypeFromSystemTestContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(accountTestOpsProbeActionBaseContextKey)
	if !ok {
		return ""
	}
	return normalizeSystemUsageOperationType(strings.TrimSpace(fmt.Sprint(value)))
}

func readSystemUsageOpsCollector(c *gin.Context) *accountTestOpsCollector {
	if c == nil {
		return nil
	}
	value, ok := c.Get(accountTestOpsCollectorContextKey)
	if !ok {
		return nil
	}
	collector, _ := value.(*accountTestOpsCollector)
	return collector
}

func readSystemUsageUpstreamStatus(c *gin.Context) (int, bool) {
	if c == nil {
		return 0, false
	}
	value, ok := c.Get(accountTestUpstreamStatusContextKey)
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func requestTypeForSystemTestMode(testMode string) RequestType {
	if normalizeAccountTestMode(testMode) == AccountTestModeRealForward {
		return RequestTypeStream
	}
	return RequestTypeSync
}

func systemUsageStatusFromResult(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "success") {
		return UsageLogStatusSucceeded
	}
	return UsageLogStatusFailed
}

func systemUsageBoolPtr(value bool) *bool {
	v := value
	return &v
}

func systemUsageIntPtrFromInt64(value int64) *int {
	intValue := int(value)
	return &intValue
}

func systemUsageStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Now()
}
