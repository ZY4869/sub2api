package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type ChannelMonitorService struct {
	repo        ChannelMonitorRepository
	historyRepo ChannelMonitorHistoryRepository
	rollupRepo  ChannelMonitorRollupRepository
	settingSvc  *SettingService
	encryptor   SecretEncryptor
	cfg         *config.Config
	checker     *channelMonitorHTTPChecker
}

func NewChannelMonitorService(
	repo ChannelMonitorRepository,
	historyRepo ChannelMonitorHistoryRepository,
	rollupRepo ChannelMonitorRollupRepository,
	settingSvc *SettingService,
	encryptor SecretEncryptor,
	cfg *config.Config,
) *ChannelMonitorService {
	return &ChannelMonitorService{
		repo:        repo,
		historyRepo: historyRepo,
		rollupRepo:  rollupRepo,
		settingSvc:  settingSvc,
		encryptor:   encryptor,
		cfg:         cfg,
		checker:     newChannelMonitorHTTPChecker(cfg),
	}
}

func (s *ChannelMonitorService) ListAll(ctx context.Context) ([]*ChannelMonitor, error) {
	return s.repo.ListAll(ctx)
}

func (s *ChannelMonitorService) GetByID(ctx context.Context, id int64) (*ChannelMonitor, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ChannelMonitorService) Create(ctx context.Context, monitor *ChannelMonitor, plaintextAPIKey *string) (*ChannelMonitor, error) {
	if monitor == nil {
		return nil, errors.New("nil monitor")
	}

	runtime, err := s.settingSvc.GetChannelMonitorRuntime(ctx)
	defaultInterval := 60
	if err == nil && runtime != nil && runtime.DefaultIntervalSeconds > 0 {
		defaultInterval = runtime.DefaultIntervalSeconds
	}
	if monitor.IntervalSeconds <= 0 {
		monitor.IntervalSeconds = defaultInterval
	}

	normalized, err := normalizeChannelMonitor(monitor)
	if err != nil {
		return nil, err
	}
	*monitor = *normalized

	endpoint, err := validateChannelMonitorEndpointForSave(s.cfg, monitor.Endpoint, monitor.Enabled)
	if err != nil {
		return nil, err
	}
	monitor.Endpoint = endpoint

	if plaintextAPIKey != nil {
		key := strings.TrimSpace(*plaintextAPIKey)
		if key != "" {
			enc, err := s.encryptor.Encrypt(key)
			if err != nil {
				return nil, err
			}
			monitor.APIKeyEncrypted = &enc
		}
	}
	if monitor.Enabled && (monitor.APIKeyEncrypted == nil || strings.TrimSpace(*monitor.APIKeyEncrypted) == "") {
		return nil, ErrChannelMonitorAPIKeyRequired
	}

	ensureNextRunAtOnEnable(monitor, time.Now())
	return s.repo.Create(ctx, monitor)
}

func (s *ChannelMonitorService) Update(ctx context.Context, monitor *ChannelMonitor, plaintextAPIKey *string) (*ChannelMonitor, error) {
	if monitor == nil {
		return nil, errors.New("nil monitor")
	}
	existing, err := s.repo.GetByID(ctx, monitor.ID)
	if err != nil {
		return nil, err
	}

	merged := *existing
	merged.Name = monitor.Name
	merged.Provider = monitor.Provider
	merged.Endpoint = monitor.Endpoint
	merged.IntervalSeconds = monitor.IntervalSeconds
	merged.Enabled = monitor.Enabled
	merged.PrimaryModelID = monitor.PrimaryModelID
	merged.AdditionalModelIDs = monitor.AdditionalModelIDs
	merged.TemplateID = monitor.TemplateID
	merged.ExtraHeaders = monitor.ExtraHeaders
	merged.BodyOverrideMode = monitor.BodyOverrideMode
	merged.BodyOverride = monitor.BodyOverride

	runtime, err := s.settingSvc.GetChannelMonitorRuntime(ctx)
	defaultInterval := 60
	if err == nil && runtime != nil && runtime.DefaultIntervalSeconds > 0 {
		defaultInterval = runtime.DefaultIntervalSeconds
	}
	if merged.IntervalSeconds <= 0 {
		merged.IntervalSeconds = defaultInterval
	}

	normalized, err := normalizeChannelMonitor(&merged)
	if err != nil {
		return nil, err
	}
	merged = *normalized

	endpoint, err := validateChannelMonitorEndpointForSave(s.cfg, merged.Endpoint, merged.Enabled)
	if err != nil {
		return nil, err
	}
	merged.Endpoint = endpoint

	if plaintextAPIKey != nil {
		key := strings.TrimSpace(*plaintextAPIKey)
		if key == "" {
			merged.APIKeyEncrypted = nil
		} else {
			enc, err := s.encryptor.Encrypt(key)
			if err != nil {
				return nil, err
			}
			merged.APIKeyEncrypted = &enc
		}
	}

	if merged.Enabled && (merged.APIKeyEncrypted == nil || strings.TrimSpace(*merged.APIKeyEncrypted) == "") {
		return nil, ErrChannelMonitorAPIKeyRequired
	}

	ensureNextRunAtOnEnable(&merged, time.Now())
	return s.repo.Update(ctx, &merged)
}

func (s *ChannelMonitorService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *ChannelMonitorService) IsAPIKeyDecryptFailed(monitor *ChannelMonitor) bool {
	if monitor == nil || monitor.APIKeyEncrypted == nil || strings.TrimSpace(*monitor.APIKeyEncrypted) == "" {
		return false
	}
	if s.encryptor == nil {
		return true
	}
	_, err := s.encryptor.Decrypt(*monitor.APIKeyEncrypted)
	return err != nil
}

func (s *ChannelMonitorService) RunCheckNow(ctx context.Context, monitorID int64) ([]*ChannelMonitorHistory, error) {
	monitor, err := s.repo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	last := now
	next := now.Add(time.Duration(monitor.IntervalSeconds) * time.Second)
	monitor.LastRunAt = &last
	monitor.NextRunAt = &next
	if _, err := s.repo.Update(ctx, monitor); err != nil {
		return nil, err
	}

	exec := newChannelMonitorExecutor(s.encryptor, s.cfg, s.checker, s.historyRepo)
	return exec.Execute(ctx, monitor)
}
