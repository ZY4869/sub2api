package repository

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"net/http"
	"strings"
	"sync"
	"time"
)

func (s *httpUpstreamService) openAIResponseHeaderTimeout() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	if seconds := s.cfg.Gateway.OpenAIResponseHeaderTimeout; seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return 0
}

func (s *httpUpstreamService) openAIHTTP2Enabled() bool {
	if s == nil || s.cfg == nil {
		return true
	}
	return s.cfg.Gateway.OpenAIHTTP2.Enabled
}

func (s *httpUpstreamService) allowOpenAIProxyHTTP2Fallback() bool {
	if s == nil || s.cfg == nil {
		return true
	}
	return s.cfg.Gateway.OpenAIHTTP2.AllowProxyFallbackToHTTP1
}

func (s *httpUpstreamService) shouldFallbackOpenAIHTTP2Proxy(proxyURL string) bool {
	if !s.allowOpenAIProxyHTTP2Fallback() {
		return false
	}
	proxyKey, _, err := normalizeProxyURL(proxyURL)
	if err != nil || proxyKey == directProxyKey {
		return false
	}
	return s.fallbacks.shouldFallback(proxyKey, time.Now())
}

func (s *httpUpstreamService) recordOpenAIHTTP2Failure(req *http.Request, proxyURL string, err error) {
	if err == nil || req == nil {
		return
	}
	if service.HTTPUpstreamProfileFromContext(req.Context()) != service.HTTPUpstreamProfileOpenAI {
		return
	}
	if !s.openAIHTTP2Enabled() || !s.allowOpenAIProxyHTTP2Fallback() {
		return
	}
	if !isOpenAIHTTP2ProxyCompatibilityError(err) {
		return
	}
	proxyKey, _, parseErr := normalizeProxyURL(proxyURL)
	if parseErr != nil || proxyKey == directProxyKey {
		return
	}
	s.fallbacks.record(proxyKey, s.openAIHTTP2FallbackSettings(), time.Now())
}

type openAIHTTP2FallbackSettings struct {
	threshold int
	window    time.Duration
	ttl       time.Duration
}

func (s *httpUpstreamService) openAIHTTP2FallbackSettings() openAIHTTP2FallbackSettings {
	settings := openAIHTTP2FallbackSettings{
		threshold: 2,
		window:    60 * time.Second,
		ttl:       600 * time.Second,
	}
	if s == nil || s.cfg == nil {
		return settings
	}
	if value := s.cfg.Gateway.OpenAIHTTP2.FallbackErrorThreshold; value > 0 {
		settings.threshold = value
	}
	if value := s.cfg.Gateway.OpenAIHTTP2.FallbackWindowSeconds; value > 0 {
		settings.window = time.Duration(value) * time.Second
	}
	if value := s.cfg.Gateway.OpenAIHTTP2.FallbackTTLSeconds; value > 0 {
		settings.ttl = time.Duration(value) * time.Second
	}
	return settings
}

type openAIHTTP2FallbackTracker struct {
	mu      sync.Mutex
	entries map[string]openAIHTTP2FallbackEntry
}

type openAIHTTP2FallbackEntry struct {
	firstFailure  time.Time
	failures      int
	fallbackUntil time.Time
}

func (t *openAIHTTP2FallbackTracker) shouldFallback(proxyKey string, now time.Time) bool {
	if strings.TrimSpace(proxyKey) == "" {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.entries[proxyKey]
	if !ok {
		return false
	}
	if entry.fallbackUntil.IsZero() || !now.Before(entry.fallbackUntil) {
		if !entry.fallbackUntil.IsZero() {
			delete(t.entries, proxyKey)
		}
		return false
	}
	return true
}

func (t *openAIHTTP2FallbackTracker) record(proxyKey string, settings openAIHTTP2FallbackSettings, now time.Time) {
	if strings.TrimSpace(proxyKey) == "" || settings.threshold <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.entries == nil {
		t.entries = make(map[string]openAIHTTP2FallbackEntry)
	}
	entry := t.entries[proxyKey]
	if entry.firstFailure.IsZero() || settings.window <= 0 || now.Sub(entry.firstFailure) > settings.window {
		entry.firstFailure = now
		entry.failures = 0
	}
	entry.failures++
	if entry.failures >= settings.threshold {
		entry.fallbackUntil = now.Add(settings.ttl)
		entry.firstFailure = now
		entry.failures = 0
	}
	t.entries[proxyKey] = entry
}

func isOpenAIHTTP2ProxyCompatibilityError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "timeout awaiting response headers") ||
		strings.Contains(text, "response header timeout") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "client.timeout exceeded") {
		return false
	}
	patterns := []string{
		"http2:",
		"http/2",
		"protocol error",
		"malformed http response",
		"server sent goaway",
		"client connection lost",
		"unexpected eof",
		"connection reset by peer",
		"connection was forcibly closed",
		"stream error",
	}
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// buildCacheKey 构建客户端缓存键
// 根据隔离策略决定缓存键的组成
//
// 参数:
//   - isolation: 隔离模式
//   - proxyKey: 代理标识
//   - accountID: 账户 ID
//
// 返回:
//   - string: 缓存键
//
// 缓存键格式:
//   - proxy 模式: "proxy:{proxyKey}"
//   - account 模式: "account:{accountID}"
//   - account_proxy 模式: "account:{accountID}|proxy:{proxyKey}"
