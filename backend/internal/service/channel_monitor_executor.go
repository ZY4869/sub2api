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

type channelMonitorCheckResult struct {
	Status       string
	ResponseText string
	ErrorMessage string
	HTTPStatus   *int
	LatencyMs    int64
	StartedAt    time.Time
	FinishedAt   time.Time
}
