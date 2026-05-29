package repository

import (
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

func (s *httpUpstreamService) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	if err := s.validateRequestHost(req); err != nil {
		return nil, err
	}

	// 获取或创建对应的客户端，并标记请求占用
	entry, err := s.acquireClientForRequest(req, proxyURL, accountID, accountConcurrency)
	if err != nil {
		return nil, err
	}

	// 执行请求
	req = withUpstreamRequestScheme(req)
	resp, err := entry.client.Do(req)
	if err != nil {
		// 请求失败，立即减少计数
		atomic.AddInt64(&entry.inFlight, -1)
		atomic.StoreInt64(&entry.lastUsed, time.Now().UnixNano())
		s.recordOpenAIHTTP2Failure(req, proxyURL, err)
		return nil, err
	}

	return s.finalizeResponse(resp, entry), nil
}

// DoWithTLS 执行带 TLS 指纹伪装的 HTTP 请求
// 根据 enableTLSFingerprint 参数决定是否使用 TLS 指纹
//
// 参数:
//   - req: HTTP 请求对象
//   - proxyURL: 代理地址，空字符串表示直连
//   - accountID: 账户 ID，用于账户级隔离和 TLS 指纹模板选择
//   - accountConcurrency: 账户并发限制，用于动态调整连接池大小
//   - enableTLSFingerprint: 是否启用 TLS 指纹伪装
//
// TLS 指纹说明:
//   - 当 enableTLSFingerprint=true 时，使用 utls 库模拟 Claude CLI 的 TLS 指纹
//   - 指纹模板根据 accountID % len(profiles) 自动选择
//   - 支持直连、HTTP/HTTPS 代理、SOCKS5 代理三种场景

func (s *httpUpstreamService) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *tlsfingerprint.Profile) (*http.Response, error) {
	// 如果未提供 TLS 指纹 profile，直接使用标准请求路径
	if tlsProfile == nil {
		return s.Do(req, proxyURL, accountID, accountConcurrency)
	}

	// TLS 指纹已启用，记录调试日志
	targetHost := ""
	if req != nil && req.URL != nil {
		targetHost = req.URL.Host
	}
	proxyInfo := "direct"
	if proxyURL != "" {
		proxyInfo = proxyURL
	}
	slog.Debug("tls_fingerprint_enabled", "account_id", accountID, "target", targetHost, "proxy", proxyInfo)

	if err := s.validateRequestHost(req); err != nil {
		return nil, err
	}

	slog.Debug("tls_fingerprint_using_profile", "account_id", accountID, "profile", tlsProfile.Name, "grease", tlsProfile.EnableGREASE)

	// 获取或创建带 TLS 指纹的客户端
	entry, err := s.acquireClientWithTLS(proxyURL, accountID, accountConcurrency, tlsProfile)
	if err != nil {
		slog.Debug("tls_fingerprint_acquire_client_failed", "account_id", accountID, "error", err)
		return nil, err
	}

	// 执行请求
	req = withUpstreamRequestScheme(req)
	resp, err := entry.client.Do(req)
	if err != nil {
		// 请求失败，立即减少计数
		atomic.AddInt64(&entry.inFlight, -1)
		atomic.StoreInt64(&entry.lastUsed, time.Now().UnixNano())
		slog.Debug("tls_fingerprint_request_failed", "account_id", accountID, "error", err)
		s.recordOpenAIHTTP2Failure(req, proxyURL, err)
		return nil, err
	}

	slog.Debug("tls_fingerprint_request_success", "account_id", accountID, "status", resp.StatusCode)

	return s.finalizeResponse(resp, entry), nil
}

// acquireClientWithTLS 获取或创建带 TLS 指纹的客户端
