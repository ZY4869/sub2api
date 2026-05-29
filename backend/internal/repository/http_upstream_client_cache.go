package repository

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *httpUpstreamService) acquireClient(proxyURL string, accountID int64, accountConcurrency int) (*upstreamClientEntry, error) {
	return s.getClientEntryWithOptions(proxyURL, accountID, accountConcurrency, true, true, upstreamRequestOptions{profile: service.HTTPUpstreamProfileDefault})
}

func (s *httpUpstreamService) acquireClientForRequest(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*upstreamClientEntry, error) {
	return s.getClientEntryWithOptions(proxyURL, accountID, accountConcurrency, true, true, s.requestOptions(req, proxyURL))
}

func (s *httpUpstreamService) requestOptions(req *http.Request, proxyURL string) upstreamRequestOptions {
	opts := upstreamRequestOptions{profile: service.HTTPUpstreamProfileDefault}
	if req == nil {
		return opts
	}
	opts.profile = service.HTTPUpstreamProfileFromContext(req.Context())
	if opts.profile != service.HTTPUpstreamProfileOpenAI {
		opts.profile = service.HTTPUpstreamProfileDefault
		return opts
	}
	opts.http2 = s.openAIHTTP2Enabled()
	if opts.http2 && s.shouldFallbackOpenAIHTTP2Proxy(proxyURL) {
		opts.http2 = false
	}
	return opts
}

func (s *httpUpstreamService) requestOptionsCacheSuffix(opts upstreamRequestOptions) string {
	if opts.profile == service.HTTPUpstreamProfileDefault {
		return ""
	}
	return fmt.Sprintf("|profile:%s|http2:%t", opts.profile, opts.http2)
}

// getOrCreateClient 获取或创建客户端
// 根据隔离策略和参数决定缓存键，处理代理变更和配置变更
//
// 参数:
//   - proxyURL: 代理地址
//   - accountID: 账户 ID
//   - accountConcurrency: 账户并发限制
//
// 返回:
//   - *upstreamClientEntry: 客户端缓存条目
//
// 隔离策略说明:
//   - proxy: 按代理地址隔离，同一代理共享客户端
//   - account: 按账户隔离，同一账户共享客户端（代理变更时重建）
//   - account_proxy: 按账户+代理组合隔离，最细粒度

func (s *httpUpstreamService) getOrCreateClient(proxyURL string, accountID int64, accountConcurrency int) (*upstreamClientEntry, error) {
	return s.getClientEntryWithOptions(proxyURL, accountID, accountConcurrency, false, false, upstreamRequestOptions{profile: service.HTTPUpstreamProfileDefault})
}

// getClientEntry 获取或创建客户端条目
// markInFlight=true 时会标记进行中请求，用于请求路径防止被淘汰
// enforceLimit=true 时会限制客户端数量，超限且无法淘汰时返回错误

func (s *httpUpstreamService) getClientEntry(proxyURL string, accountID int64, accountConcurrency int, markInFlight bool, enforceLimit bool) (*upstreamClientEntry, error) {
	return s.getClientEntryWithOptions(proxyURL, accountID, accountConcurrency, markInFlight, enforceLimit, upstreamRequestOptions{profile: service.HTTPUpstreamProfileDefault})
}

func (s *httpUpstreamService) getClientEntryWithOptions(proxyURL string, accountID int64, accountConcurrency int, markInFlight bool, enforceLimit bool, opts upstreamRequestOptions) (*upstreamClientEntry, error) {
	// 获取隔离模式
	isolation := s.getIsolationMode()
	// 标准化代理 URL 并解析
	proxyKey, parsedProxy, err := normalizeProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}
	// 构建缓存键（根据隔离策略不同）
	cacheKey := buildCacheKey(isolation, proxyKey, accountID) + s.requestOptionsCacheSuffix(opts)
	// 构建连接池配置键（用于检测配置变更）
	poolKey := s.buildPoolKeyWithOptions(isolation, accountConcurrency, opts)

	now := time.Now()
	nowUnix := now.UnixNano()

	// 读锁快速路径：命中缓存直接返回，减少锁竞争
	s.mu.RLock()
	if entry, ok := s.clients[cacheKey]; ok && s.shouldReuseEntry(entry, isolation, proxyKey, poolKey) {
		atomic.StoreInt64(&entry.lastUsed, nowUnix)
		if markInFlight {
			atomic.AddInt64(&entry.inFlight, 1)
		}
		s.mu.RUnlock()
		return entry, nil
	}
	s.mu.RUnlock()

	// 写锁慢路径：创建或重建客户端
	s.mu.Lock()
	if entry, ok := s.clients[cacheKey]; ok {
		if s.shouldReuseEntry(entry, isolation, proxyKey, poolKey) {
			atomic.StoreInt64(&entry.lastUsed, nowUnix)
			if markInFlight {
				atomic.AddInt64(&entry.inFlight, 1)
			}
			s.mu.Unlock()
			return entry, nil
		}
		s.removeClientLocked(cacheKey, entry)
	}

	// 超出缓存上限时尝试淘汰，无法淘汰则拒绝新建
	if enforceLimit && s.maxUpstreamClients() > 0 {
		s.evictIdleLocked(now)
		if len(s.clients) >= s.maxUpstreamClients() {
			if !s.evictOldestIdleLocked() {
				s.mu.Unlock()
				return nil, errUpstreamClientLimitReached
			}
		}
	}

	// 缓存未命中或需要重建，创建新客户端
	settings := s.resolvePoolSettingsWithOptions(isolation, accountConcurrency, opts)
	transport, err := buildUpstreamTransport(settings, parsedProxy)
	if err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("build transport: %w", err)
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

	// 执行淘汰策略：先淘汰空闲超时的，再淘汰超出数量限制的
	s.evictIdleLocked(now)
	s.evictOverLimitLocked()
	s.mu.Unlock()
	return entry, nil
}

// shouldReuseEntry 判断缓存条目是否可复用
// 若代理或连接池配置发生变化，则需要重建客户端

func (s *httpUpstreamService) shouldReuseEntry(entry *upstreamClientEntry, isolation, proxyKey, poolKey string) bool {
	if entry == nil {
		return false
	}
	if isolation == config.ConnectionPoolIsolationAccount && entry.proxyKey != proxyKey {
		return false
	}
	if entry.poolKey != poolKey {
		return false
	}
	return true
}

// removeClientLocked 移除客户端（需持有锁）
// 从缓存中删除并关闭空闲连接
//
// 参数:
//   - key: 缓存键
//   - entry: 客户端条目

func (s *httpUpstreamService) removeClientLocked(key string, entry *upstreamClientEntry) {
	delete(s.clients, key)
	if entry != nil && entry.client != nil {
		// 关闭空闲连接，释放系统资源
		// 注意：这不会中断活跃连接
		entry.client.CloseIdleConnections()
	}
}
