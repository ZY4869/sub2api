package securityaudit

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type PromptService struct {
	config  *ConfigManager
	repo    promptRepository
	payload PayloadStore
	scanner PromptScanner
	metrics *AtomicMetrics
	worker  *Worker

	invalidator  ConfigInvalidator
	lifecycleMu  sync.Mutex
	cancel       context.CancelFunc
	background   context.Context
	enqueueWG    sync.WaitGroup
	enqueueSlots chan struct{}
	probeMu      sync.RWMutex
	probes       map[string]ProbeResult
}

func NewPromptService(config *ConfigManager, repo promptRepository, payload PayloadStore, scanner PromptScanner, metrics *AtomicMetrics) *PromptService {
	return &PromptService{
		config: config, repo: repo, payload: payload, scanner: scanner, metrics: metrics,
		enqueueSlots: make(chan struct{}, DefaultAsyncEnqueueSlots),
		probes:       map[string]ProbeResult{},
	}
}

func (s *PromptService) SetConfigInvalidator(invalidator ConfigInvalidator) {
	if s == nil {
		return
	}
	s.invalidator = invalidator
	if s.config != nil {
		s.config.SetInvalidator(invalidator)
	}
}

func (s *PromptService) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.lifecycleMu.Lock()
	if s.cancel != nil {
		s.lifecycleMu.Unlock()
		return
	}
	background, cancel := context.WithCancel(ctx)
	s.background = background
	s.cancel = cancel
	s.lifecycleMu.Unlock()
	if err := s.config.Load(ctx); err != nil {
		promptAuditLog(ctx).Warn("prompt_audit_config_load_failed", zap.String("error_code", ErrorCodeConfigConflict))
	}
	s.worker = NewWorker(s.config, s.repo, s.payload, s.scanner, s.metrics)
	s.worker.Start(background)
	s.startConfigInvalidationSubscriber(background)
}

func (s *PromptService) Stop() {
	if s == nil {
		return
	}
	s.lifecycleMu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.background = nil
	s.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if s.worker != nil {
		s.worker.Stop()
	}
	s.enqueueWG.Wait()
}

func (s *PromptService) GetConfig() PublicConfig { return s.config.Public() }

func (s *PromptService) SaveConfig(ctx context.Context, req UpdateConfigRequest, actorID int64) (PublicConfig, error) {
	cfg, err := s.config.Save(ctx, req, actorID)
	if err != nil {
		promptAuditLog(ctx).Warn("prompt_audit_config_save_failed",
			zap.Int64("actor_user_id", actorID),
			zap.String("error_code", errorReason(err)),
		)
		return PublicConfig{}, err
	}
	promptAuditLog(ctx).Info("prompt_audit_config_saved",
		zap.Int64("actor_user_id", actorID),
		zap.Int64("config_version", cfg.ConfigVersion),
		zap.Bool("enabled", cfg.Enabled),
		zap.Bool("blocking_enabled", cfg.BlockingEnabled),
		zap.Int("endpoint_count", len(cfg.Endpoints)),
	)
	return cfg, nil
}

func (s *PromptService) startConfigInvalidationSubscriber(ctx context.Context) {
	if s == nil || s.invalidator == nil {
		return
	}
	go func() {
		err := s.invalidator.Subscribe(ctx, func() {
			if loadErr := s.config.Load(ctx); loadErr != nil {
				promptAuditLog(ctx).Warn("prompt_audit_config_reload_failed", zap.String("error_code", ErrorCodeConfigConflict))
			}
		})
		if err != nil && ctx.Err() == nil {
			promptAuditLog(ctx).Warn("prompt_audit_config_subscribe_failed", zap.String("error_code", "prompt_audit_config_subscribe_failed"))
		}
	}()
}
