package service

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	channelMonitorRunnerConcurrency = 5
)

type ChannelMonitorRunnerService struct {
	db          *sql.DB
	repo        ChannelMonitorRepository
	historyRepo ChannelMonitorHistoryRepository
	rollupRepo  ChannelMonitorRollupRepository
	aggRepo     ChannelMonitorAggregationRepository
	settingSvc  *SettingService
	executor    *channelMonitorExecutor

	stopCh chan struct{}
	doneCh chan struct{}

	mu          sync.Mutex
	lockRelease func()
}

func NewChannelMonitorRunnerService(
	db *sql.DB,
	repo ChannelMonitorRepository,
	historyRepo ChannelMonitorHistoryRepository,
	rollupRepo ChannelMonitorRollupRepository,
	aggRepo ChannelMonitorAggregationRepository,
	settingSvc *SettingService,
	encryptor SecretEncryptor,
	cfg *config.Config,
) *ChannelMonitorRunnerService {
	checker := newChannelMonitorHTTPChecker(cfg)
	executor := newChannelMonitorExecutor(encryptor, cfg, checker, historyRepo)
	return &ChannelMonitorRunnerService{
		db:          db,
		repo:        repo,
		historyRepo: historyRepo,
		rollupRepo:  rollupRepo,
		aggRepo:     aggRepo,
		settingSvc:  settingSvc,
		executor:    executor,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

func (s *ChannelMonitorRunnerService) Start() {
	go s.run()
}

func (s *ChannelMonitorRunnerService) Stop() {
	select {
	case <-s.stopCh:
		// already stopped
	default:
		close(s.stopCh)
	}
	<-s.doneCh
}

func (s *ChannelMonitorRunnerService) run() {
	defer close(s.doneCh)

	sem := make(chan struct{}, channelMonitorRunnerConcurrency)

	dueTicker := time.NewTicker(5 * time.Second)
	aggTicker := time.NewTicker(10 * time.Second)
	retentionTicker := time.NewTicker(1 * time.Hour)
	defer func() {
		dueTicker.Stop()
		aggTicker.Stop()
		retentionTicker.Stop()
	}()

	for !s.tryAcquireLeaderLock() {
		select {
		case <-s.stopCh:
			return
		case <-time.After(10 * time.Second):
		}
	}

	for {
		select {
		case <-s.stopCh:
			s.releaseLeaderLock()
			return
		case <-dueTicker.C:
			if !channelMonitorRequireEnabled(context.Background(), s.settingSvc) {
				continue
			}
			s.runDueOnce(sem)
		case <-aggTicker.C:
			s.aggregateOnce(context.Background())
		case <-retentionTicker.C:
			s.pruneOnce(context.Background())
		}
	}
}

func (s *ChannelMonitorRunnerService) runDueOnce(sem chan struct{}) {
	now := time.Now()
	claimed, err := s.repo.ClaimDue(context.Background(), now, 20)
	if err != nil || len(claimed) == 0 {
		return
	}

	for _, monitor := range claimed {
		m := monitor
		select {
		case sem <- struct{}{}:
		default:
			// if saturated, skip and let next tick pick it up again
			continue
		}
		go func() {
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
			defer cancel()
			if _, err := s.executor.Execute(ctx, m); err != nil {
				logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitorRunner] execute failed: monitor_id=%d err=%v", m.ID, err)
			}
		}()
	}
}

func (s *ChannelMonitorRunnerService) tryAcquireLeaderLock() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lockRelease != nil {
		return true
	}
	key := "service.channel_monitor_runner"
	release, ok := tryAcquireDBAdvisoryLock(context.Background(), s.db, hashAdvisoryLockID(key))
	if !ok {
		return false
	}
	s.lockRelease = release
	logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitorRunner] leader lock acquired")
	return true
}

func (s *ChannelMonitorRunnerService) releaseLeaderLock() {
	s.mu.Lock()
	release := s.lockRelease
	s.lockRelease = nil
	s.mu.Unlock()
	if release != nil {
		release()
		logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitorRunner] leader lock released")
	}
}
