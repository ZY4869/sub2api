package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const proxyExpiryJobName = "proxy_expiry_fallback"

type ProxyExpiryService struct {
	proxyRepo   ProxyExpiryRepository
	accountRepo AccountProxyFallbackRepository
	proxyReader ProxyRepository
	leaderGate  PeriodicJobLeaderGate
	interval    time.Duration
	now         func() time.Time
	stopCh      chan struct{}
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func NewProxyExpiryService(
	proxyRepo ProxyExpiryRepository,
	accountRepo AccountProxyFallbackRepository,
	proxyReader ProxyRepository,
	interval time.Duration,
) *ProxyExpiryService {
	if interval <= 0 {
		interval = time.Minute
	}
	return &ProxyExpiryService{
		proxyRepo:   proxyRepo,
		accountRepo: accountRepo,
		proxyReader: proxyReader,
		interval:    interval,
		now:         time.Now,
		stopCh:      make(chan struct{}),
	}
}

func (s *ProxyExpiryService) SetLeaderGate(gate PeriodicJobLeaderGate) {
	if s == nil {
		return
	}
	s.leaderGate = gate
}

func (s *ProxyExpiryService) SetNow(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = now
}

func (s *ProxyExpiryService) Start() {
	if s == nil || s.proxyRepo == nil || s.accountRepo == nil || s.proxyReader == nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runLeaderOnce(context.Background())
		for {
			select {
			case <-ticker.C:
				s.runLeaderOnce(context.Background())
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *ProxyExpiryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *ProxyExpiryService) runLeaderOnce(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.leaderGate == nil {
		s.runOnce(ctx)
		return true
	}
	return s.leaderGate.RunIfLeader(ctx, proxyExpiryJobName, periodicJobLeaderTTL(s.interval), s.runOnce)
}

func (s *ProxyExpiryService) runOnce(ctx context.Context) {
	now := s.now().UTC()
	proxies, err := s.proxyRepo.ListExpiredProxies(ctx, now)
	if err != nil {
		slog.Warn("proxy_expiry_scan_failed", "error", err)
		return
	}
	for i := range proxies {
		s.handleExpiredProxy(ctx, proxies[i], now)
	}
}

func (s *ProxyExpiryService) handleExpiredProxy(ctx context.Context, expired Proxy, now time.Time) {
	if expired.FallbackProxyID == nil || *expired.FallbackProxyID <= 0 {
		slog.Warn(
			"proxy_expired_without_fallback",
			"proxy_id", expired.ID,
			"expires_at", expired.ExpiresAt,
			"checked_at", now,
		)
		return
	}
	fallback, err := s.proxyReader.GetByID(ctx, *expired.FallbackProxyID)
	if err != nil || fallback == nil || fallback.Status != StatusActive {
		slog.Warn(
			"proxy_expiry_fallback_unavailable",
			"proxy_id", expired.ID,
			"fallback_proxy_id", *expired.FallbackProxyID,
			"checked_at", now,
			"error", err,
		)
		return
	}
	accountIDs, err := s.accountRepo.SwitchExpiredProxyAccounts(ctx, expired, *fallback, now)
	if err != nil {
		slog.Warn(
			"proxy_expiry_fallback_failed",
			"proxy_id", expired.ID,
			"fallback_proxy_id", fallback.ID,
			"checked_at", now,
			"error", err,
		)
		return
	}
	slog.Info(
		"proxy_expiry_fallback_applied",
		"proxy_id", expired.ID,
		"fallback_proxy_id", fallback.ID,
		"account_count", len(accountIDs),
		"checked_at", now,
	)
}

func ProvideProxyExpiryService(
	proxyRepo ProxyRepository,
	accountRepo AccountRepository,
	leaderGate PeriodicJobLeaderGate,
) *ProxyExpiryService {
	expiryRepo, ok := proxyRepo.(ProxyExpiryRepository)
	if !ok {
		return nil
	}
	fallbackRepo, ok := accountRepo.(AccountProxyFallbackRepository)
	if !ok {
		return nil
	}
	svc := NewProxyExpiryService(expiryRepo, fallbackRepo, proxyRepo, time.Minute)
	svc.SetLeaderGate(leaderGate)
	svc.Start()
	return svc
}
