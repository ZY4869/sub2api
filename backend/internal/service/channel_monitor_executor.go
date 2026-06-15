package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type channelMonitorExecutor struct {
	encryptor   SecretEncryptor
	cfg         *config.Config
	checker     *channelMonitorHTTPChecker
	historyRepo ChannelMonitorHistoryRepository
	accountRepo AccountRepository
	testRunner  channelMonitorAccountTestRunner
}

type channelMonitorAccountTestRunner interface {
	RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
}

func newChannelMonitorExecutor(
	encryptor SecretEncryptor,
	cfg *config.Config,
	checker *channelMonitorHTTPChecker,
	historyRepo ChannelMonitorHistoryRepository,
) *channelMonitorExecutor {
	return &channelMonitorExecutor{
		encryptor:   encryptor,
		cfg:         cfg,
		checker:     checker,
		historyRepo: historyRepo,
	}
}

func (e *channelMonitorExecutor) Execute(ctx context.Context, monitor *ChannelMonitor) ([]*ChannelMonitorHistory, error) {
	if monitor == nil {
		return nil, errors.New("nil monitor")
	}
	if monitor.ID <= 0 {
		return nil, errors.New("invalid monitor id")
	}
	if e.historyRepo == nil || e.checker == nil {
		return nil, errors.New("executor dependencies missing")
	}
	if monitor.ProbeMode == ChannelMonitorProbeModeAccountPool {
		return e.executeAccountPool(ctx, monitor)
	}
	return e.executeDirect(ctx, monitor)
}

func (e *channelMonitorExecutor) executeDirect(ctx context.Context, monitor *ChannelMonitor) ([]*ChannelMonitorHistory, error) {
	if monitor.APIKeyEncrypted == nil || strings.TrimSpace(*monitor.APIKeyEncrypted) == "" {
		return nil, ErrChannelMonitorAPIKeyRequired
	}
	if e.encryptor == nil {
		return nil, ErrChannelMonitorAPIKeyDecryptFailed
	}
	apiKey, err := e.encryptor.Decrypt(*monitor.APIKeyEncrypted)
	if err != nil || strings.TrimSpace(apiKey) == "" {
		return nil, ErrChannelMonitorAPIKeyDecryptFailed
	}

	models := dedupeNonEmptyStrings(append([]string{monitor.PrimaryModelID}, monitor.AdditionalModelIDs...))
	if len(models) == 0 {
		return nil, errors.New("no models to check")
	}

	var created []*ChannelMonitorHistory
	for _, modelID := range models {
		result := e.checker.Check(ctx, monitor, modelID, apiKey)

		h := &ChannelMonitorHistory{
			MonitorID:    monitor.ID,
			ProbeMode:    ChannelMonitorProbeModeDirect,
			ModelID:      modelID,
			Status:       result.Status,
			ResponseText: result.ResponseText,
			ErrorMessage: result.ErrorMessage,
			HTTPStatus:   result.HTTPStatus,
			LatencyMs:    result.LatencyMs,
			StartedAt:    result.StartedAt,
			FinishedAt:   result.FinishedAt,
		}
		if _, err := e.historyRepo.Create(ctx, h); err != nil {
			logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitor] write history failed: monitor_id=%d model=%s err=%v", monitor.ID, modelID, err)
			continue
		}
		created = append(created, h)
	}

	return created, nil
}

func (e *channelMonitorExecutor) executeAccountPool(ctx context.Context, monitor *ChannelMonitor) ([]*ChannelMonitorHistory, error) {
	if e.testRunner == nil {
		return nil, errors.New("account test runner not configured")
	}
	models := channelMonitorModelsForStrategy(monitor)
	if len(models) == 0 {
		return nil, errors.New("no models to check")
	}
	accountNames := e.accountNameSnapshots(ctx, monitor.AccountIDs)
	var created []*ChannelMonitorHistory
	for _, accountID := range monitor.AccountIDs {
		if accountID <= 0 {
			continue
		}
		for _, modelID := range models {
			challenge := randomChallenge()
			prompt := buildChannelMonitorPrompt(monitor.TestPromptTemplate, challenge)
			result, err := e.testRunner.RunTestBackgroundDetailed(ctx, ScheduledTestExecutionInput{
				AccountID:      accountID,
				ModelID:        modelID,
				SourceProtocol: channelMonitorSourceProtocolForModel(monitor, modelID),
				Prompt:         prompt,
				TestMode:       string(AccountTestModeHealthCheck),
				OperationType:  UsageOperationTypeAccountTest,
			})
			if err != nil {
				now := time.Now()
				result = &BackgroundAccountTestResult{
					Status:       "failed",
					ErrorMessage: err.Error(),
					StartedAt:    now,
					FinishedAt:   now,
				}
			}
			h := channelMonitorHistoryFromAccountTest(monitor.ID, accountID, accountNames[accountID], modelID, challenge, result)
			if _, err := e.historyRepo.Create(ctx, h); err != nil {
				logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitor] write account history failed: monitor_id=%d account_id=%d model=%s err=%v", monitor.ID, accountID, modelID, err)
				continue
			}
			created = append(created, h)
		}
	}
	return created, nil
}

func channelMonitorModelsForStrategy(monitor *ChannelMonitor) []string {
	if monitor == nil {
		return nil
	}
	if monitor.ModelProbeStrategy == ChannelMonitorModelProbeStrategyPrimaryOnly {
		return dedupeNonEmptyStrings([]string{monitor.PrimaryModelID})
	}
	return dedupeNonEmptyStrings(append([]string{monitor.PrimaryModelID}, monitor.AdditionalModelIDs...))
}

func channelMonitorSourceProtocolForModel(monitor *ChannelMonitor, modelID string) string {
	if monitor == nil {
		return ""
	}
	if protocol := strings.TrimSpace(monitor.ModelSourceProtocols[strings.TrimSpace(modelID)]); protocol != "" {
		return protocol
	}
	return strings.TrimSpace(monitor.RequestProtocol)
}

func (e *channelMonitorExecutor) accountNameSnapshots(ctx context.Context, accountIDs []int64) map[int64]string {
	out := map[int64]string{}
	if e.accountRepo == nil || len(accountIDs) == 0 {
		return out
	}
	accounts, err := e.accountRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitor] load account snapshots failed: err=%v", err)
		return out
	}
	for _, account := range accounts {
		if account == nil || account.ID <= 0 {
			continue
		}
		out[account.ID] = strings.TrimSpace(account.Name)
	}
	return out
}

func channelMonitorHistoryFromAccountTest(monitorID int64, accountID int64, accountName string, modelID string, challenge string, result *BackgroundAccountTestResult) *ChannelMonitorHistory {
	now := time.Now()
	h := &ChannelMonitorHistory{
		MonitorID:           monitorID,
		AccountID:           &accountID,
		AccountNameSnapshot: accountName,
		ProbeMode:           ChannelMonitorProbeModeAccountPool,
		ModelID:             modelID,
		Status:              ChannelMonitorStatusFailure,
		StartedAt:           now,
		FinishedAt:          now,
	}
	if result == nil {
		h.ErrorMessage = "empty_test_result"
		return h
	}
	h.ResponseText = truncateText(result.ResponseText, 512)
	h.ErrorMessage = truncateText(result.ErrorMessage, 512)
	h.LatencyMs = result.LatencyMs
	h.StartedAt = result.StartedAt
	h.FinishedAt = result.FinishedAt
	if h.StartedAt.IsZero() {
		h.StartedAt = now
	}
	if h.FinishedAt.IsZero() {
		h.FinishedAt = now
	}
	if strings.EqualFold(result.Status, "success") {
		if strings.TrimSpace(result.ResponseText) != "" && !strings.Contains(result.ResponseText, challenge) {
			h.ErrorMessage = "challenge_mismatch"
			return h
		}
		h.Status = ChannelMonitorStatusSuccess
		if h.LatencyMs > channelMonitorDegradedThreshold {
			h.Status = ChannelMonitorStatusDegraded
		}
	}
	return h
}

type channelMonitorCheckResult struct {
	Status       string
	ResponseText string
	ErrorMessage string
	HTTPStatus   *int
	LatencyMs    int64
	StartedAt    time.Time
	FinishedAt   time.Time
}
