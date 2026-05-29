package repository

import (
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

func (s *httpUpstreamService) acquireClientWithTLS(proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*upstreamClientEntry, error) {
	return s.getClientEntryWithTLS(proxyURL, accountID, accountConcurrency, profile, true, true)
}

// getClientEntryWithTLS 获取或创建带 TLS 指纹的客户端条目
// TLS 指纹客户端使用独立的缓存键，与普通客户端隔离

func (s *httpUpstreamService) getClientEntryWithTLS(proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile, markInFlight bool, enforceLimit bool) (*upstreamClientEntry, error) {
	isolation := s.getIsolationMode()
	proxyKey, parsedProxy, err := normalizeProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}
	// TLS 指纹客户端使用独立的缓存键，加 "tls:" 前缀
	profileKey := buildTLSProfileKey(profile)
	cacheKey := "tls:" + buildCacheKey(isolation, proxyKey, accountID) + ":" + profileKey
	poolKey := s.buildPoolKey(isolation, accountConcurrency) + ":tls:" + profileKey

	now := time.Now()
	nowUnix := now.UnixNano()

	// 读锁快速路径
	s.mu.RLock()
	if entry, ok := s.clients[cacheKey]; ok && s.shouldReuseEntry(entry, isolation, proxyKey, poolKey) {
		atomic.StoreInt64(&entry.lastUsed, nowUnix)
		if markInFlight {
			atomic.AddInt64(&entry.inFlight, 1)
		}
		s.mu.RUnlock()
		slog.Debug("tls_fingerprint_reusing_client", "account_id", accountID, "cache_key", cacheKey)
		return entry, nil
	}
	s.mu.RUnlock()

	// 写锁慢路径
	s.mu.Lock()
	if entry, ok := s.clients[cacheKey]; ok {
		if s.shouldReuseEntry(entry, isolation, proxyKey, poolKey) {
			atomic.StoreInt64(&entry.lastUsed, nowUnix)
			if markInFlight {
				atomic.AddInt64(&entry.inFlight, 1)
			}
			s.mu.Unlock()
			slog.Debug("tls_fingerprint_reusing_client", "account_id", accountID, "cache_key", cacheKey)
			return entry, nil
		}
		slog.Debug("tls_fingerprint_evicting_stale_client",
			"account_id", accountID,
			"cache_key", cacheKey,
			"proxy_changed", entry.proxyKey != proxyKey,
			"pool_changed", entry.poolKey != poolKey)
		s.removeClientLocked(cacheKey, entry)
	}

	// 超出缓存上限时尝试淘汰
	if enforceLimit && s.maxUpstreamClients() > 0 {
		s.evictIdleLocked(now)
		if len(s.clients) >= s.maxUpstreamClients() {
			if !s.evictOldestIdleLocked() {
				s.mu.Unlock()
				return nil, errUpstreamClientLimitReached
			}
		}
	}

	// 创建带 TLS 指纹的 Transport
	slog.Debug("tls_fingerprint_creating_new_client", "account_id", accountID, "cache_key", cacheKey, "proxy", proxyKey)
	settings := s.resolvePoolSettings(isolation, accountConcurrency)
	transport, err := buildUpstreamTransportWithTLSFingerprint(settings, parsedProxy, profile)
	if err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("build TLS fingerprint transport: %w", err)
	}

	client := &http.Client{Transport: transport, CheckRedirect: s.redirectChecker}

	entry := &upstreamClientEntry{
		client:   client,
		proxyKey: proxyKey,
		poolKey:  poolKey,
	}
	atomic.StoreInt64(&entry.lastUsed, nowUnix)
	if markInFlight {
		atomic.StoreInt64(&entry.inFlight, 1)
	}
	s.clients[cacheKey] = entry

	s.evictIdleLocked(now)
	s.evictOverLimitLocked()
	s.mu.Unlock()
	return entry, nil
}

func buildTLSProfileKey(profile *tlsfingerprint.Profile) string {
	if profile == nil {
		return "default"
	}
	return fmt.Sprintf("%s|%t|%v|%v|%v|%v|%v|%v|%v|%v|%v",
		profile.Name,
		profile.EnableGREASE,
		profile.CipherSuites,
		profile.Curves,
		profile.PointFormats,
		profile.SignatureAlgorithms,
		profile.ALPNProtocols,
		profile.SupportedVersions,
		profile.KeyShareGroups,
		profile.PSKModes,
		profile.Extensions,
	)
}
