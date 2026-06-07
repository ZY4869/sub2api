package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const defaultAccountBlacklistCleanupInterval = time.Hour
const defaultAccountBlacklistCleanupBatchSize = 100
const accountBlacklistCleanupJobName = "account_blacklist_cleanup"

type AccountBlacklistCleanupService struct {
	accountRepo AccountRepository
	leaderGate  PeriodicJobLeaderGate
	interval    time.Duration
	stopCh      chan struct{}
	stopOnce    sync.Once
}

func NewAccountBlacklistCleanupService(accountRepo AccountRepository, interval time.Duration) *AccountBlacklistCleanupService {
	if interval <= 0 {
		interval = defaultAccountBlacklistCleanupInterval
	}
	return &AccountBlacklistCleanupService{
		accountRepo: accountRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

func (s *AccountBlacklistCleanupService) SetLeaderGate(gate PeriodicJobLeaderGate) {
	if s == nil {
		return
	}
	s.leaderGate = gate
}

func (s *AccountBlacklistCleanupService) Start() {
	if s == nil || s.accountRepo == nil {
		return
	}
	go s.loop()
}

func (s *AccountBlacklistCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *AccountBlacklistCleanupService) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		s.runLeaderOnce(context.Background())
		select {
		case <-ticker.C:
		case <-s.stopCh:
			return
		}
	}
}

func (s *AccountBlacklistCleanupService) runLeaderOnce(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.leaderGate == nil {
		s.runOnce(ctx)
		return true
	}
	return s.leaderGate.RunIfLeader(ctx, accountBlacklistCleanupJobName, periodicJobLeaderTTL(s.interval), s.runOnce)
}

func (s *AccountBlacklistCleanupService) runOnce(ctx context.Context) {
	if s == nil || s.accountRepo == nil {
		return
	}
	accounts, err := s.accountRepo.ListBlacklistedForPurge(ctx, time.Now(), defaultAccountBlacklistCleanupBatchSize)
	if err != nil {
		slog.Warn("account_blacklist_cleanup_list_failed", "error", err)
		return
	}
	for _, account := range accounts {
		slog.Info("account_blacklist_cleanup_delete_start", "account_id", account.ID, "purge_at", account.BlacklistPurgeAt)
		if err := s.accountRepo.Delete(ctx, account.ID); err != nil {
			slog.Warn("account_blacklist_cleanup_delete_failed", "account_id", account.ID, "error", err)
			continue
		}
		slog.Info("account_blacklist_cleanup_delete_done", "account_id", account.ID)
	}
}
