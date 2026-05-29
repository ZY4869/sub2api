package repository

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
	"github.com/Wei-Shaw/sub2api/internal/pkg/netguard"
	"github.com/Wei-Shaw/sub2api/internal/pkg/proxyurl"
	"github.com/Wei-Shaw/sub2api/internal/pkg/proxyutil"
)

func buildCacheKey(isolation, proxyKey string, accountID int64) string {
	switch isolation {
	case config.ConnectionPoolIsolationAccount:
		return fmt.Sprintf("account:%d", accountID)
	case config.ConnectionPoolIsolationAccountProxy:
		return fmt.Sprintf("account:%d|proxy:%s", accountID, proxyKey)
	default:
		return fmt.Sprintf("proxy:%s", proxyKey)
	}
}

// normalizeProxyURL 标准化代理 URL
// 处理空值和解析错误，返回标准化的键和解析后的 URL
//
// 参数:
//   - raw: 原始代理 URL 字符串
//
// 返回:
//   - string: 标准化的代理键（空返回 "direct"）
//   - *url.URL: 解析后的 URL（空返回 nil）
//   - error: 非空代理 URL 解析失败时返回错误（禁止回退到直连）

func normalizeProxyURL(raw string) (string, *url.URL, error) {
	_, parsed, err := proxyurl.Parse(raw)
	if err != nil {
		return "", nil, err
	}
	if parsed == nil {
		return directProxyKey, nil, nil
	}
	// 规范化：小写 scheme/host，去除路径和查询参数
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.ForceQuery = false
	if hostname := parsed.Hostname(); hostname != "" {
		port := parsed.Port()
		if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
			port = ""
		}
		hostname = strings.ToLower(hostname)
		if port != "" {
			parsed.Host = net.JoinHostPort(hostname, port)
		} else {
			parsed.Host = hostname
		}
	}
	return parsed.String(), parsed, nil
}

// defaultPoolSettings 获取默认连接池配置
// 从全局配置中读取，无效值使用常量默认值
//
// 参数:
//   - cfg: 全局配置
//
// 返回:
//   - poolSettings: 连接池配置

func defaultPoolSettings(cfg *config.Config) poolSettings {
	maxIdleConns := defaultMaxIdleConns
	maxIdleConnsPerHost := defaultMaxIdleConnsPerHost
	maxConnsPerHost := defaultMaxConnsPerHost
	idleConnTimeout := defaultIdleConnTimeout
	responseHeaderTimeout := defaultResponseHeaderTimeout

	if cfg != nil {
		if cfg.Gateway.MaxIdleConns > 0 {
			maxIdleConns = cfg.Gateway.MaxIdleConns
		}
		if cfg.Gateway.MaxIdleConnsPerHost > 0 {
			maxIdleConnsPerHost = cfg.Gateway.MaxIdleConnsPerHost
		}
		if cfg.Gateway.MaxConnsPerHost >= 0 {
			maxConnsPerHost = cfg.Gateway.MaxConnsPerHost
		}
		if cfg.Gateway.IdleConnTimeoutSeconds > 0 {
			idleConnTimeout = time.Duration(cfg.Gateway.IdleConnTimeoutSeconds) * time.Second
		}
		if cfg.Gateway.ResponseHeaderTimeout > 0 {
			responseHeaderTimeout = time.Duration(cfg.Gateway.ResponseHeaderTimeout) * time.Second
		}
	}

	return poolSettings{
		maxIdleConns:          maxIdleConns,
		maxIdleConnsPerHost:   maxIdleConnsPerHost,
		maxConnsPerHost:       maxConnsPerHost,
		idleConnTimeout:       idleConnTimeout,
		responseHeaderTimeout: responseHeaderTimeout,
		forceAttemptHTTP2:     false,
	}
}

// buildUpstreamTransport 构建上游请求的 Transport
// 使用配置文件中的连接池参数，支持生产环境调优
//
// 参数:
//   - settings: 连接池配置
//   - proxyURL: 代理 URL（nil 表示直连）
//
// 返回:
//   - *http.Transport: 配置好的 Transport 实例
//   - error: 代理配置错误
//
// Transport 参数说明:
//   - MaxIdleConns: 所有主机的最大空闲连接总数
//   - MaxIdleConnsPerHost: 每主机最大空闲连接数（影响连接复用率）
//   - MaxConnsPerHost: 每主机最大连接数（达到后新请求等待）
//   - IdleConnTimeout: 空闲连接超时（超时后关闭）
//   - ResponseHeaderTimeout: 等待响应头超时（不影响流式传输）

func buildUpstreamTransport(settings poolSettings, proxyURL *url.URL) (*http.Transport, error) {
	baseDialer := &net.Dialer{}
	targetDialer := newUpstreamGuardedDialer(settings, baseDialer, hostexceptions.ScopeUpstreamBaseURL, "")
	proxyDialer := newUpstreamGuardedDialer(settings, baseDialer, hostexceptions.ScopeProxyHost, proxyScheme(proxyURL))
	transport := &http.Transport{
		DialContext:           targetDialer.DialContext,
		MaxIdleConns:          settings.maxIdleConns,
		MaxIdleConnsPerHost:   settings.maxIdleConnsPerHost,
		MaxConnsPerHost:       settings.maxConnsPerHost,
		IdleConnTimeout:       settings.idleConnTimeout,
		ResponseHeaderTimeout: settings.responseHeaderTimeout,
		ForceAttemptHTTP2:     settings.forceAttemptHTTP2,
	}
	if err := proxyutil.ConfigureTransportProxyWithDialer(transport, proxyURL, proxyDialer); err != nil {
		return nil, err
	}
	return transport, nil
}

func newUpstreamGuardedDialer(settings poolSettings, base *net.Dialer, scope, scheme string) *netguard.Dialer {
	return netguard.NewDialer(netguard.Options{
		Base:               base,
		ValidateResolvedIP: settings.validateResolvedIP,
		AllowPrivateHosts:  settings.allowPrivateHosts,
		AllowResolvedIPWithContext: func(ctx context.Context, host string, port int, ip net.IP) bool {
			effectiveScheme := scheme
			if effectiveScheme == "" {
				if value, _ := ctx.Value(upstreamRequestSchemeContextKey{}).(string); strings.TrimSpace(value) != "" {
					effectiveScheme = strings.TrimSpace(value)
				}
			}
			match, ok := hostexceptions.IsResolvedIPAllowed(settings.privateHostConfig, scope, effectiveScheme, host, port, ip)
			if ok {
				hostexceptions.LogMatch(ctx, match)
			}
			return ok
		},
	})
}

func proxyScheme(proxyURL *url.URL) string {
	if proxyURL == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(proxyURL.Scheme))
}

// trackedBody 带跟踪功能的响应体包装器
// 在 Close 时执行回调，用于更新请求计数
