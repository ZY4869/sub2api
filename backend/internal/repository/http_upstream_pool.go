package repository

import (
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"strings"
	"time"
)

func (s *httpUpstreamService) getIsolationMode() string {
	if s.cfg == nil {
		return config.ConnectionPoolIsolationAccountProxy
	}
	mode := strings.ToLower(strings.TrimSpace(s.cfg.Gateway.ConnectionPoolIsolation))
	if mode == "" {
		return config.ConnectionPoolIsolationAccountProxy
	}
	switch mode {
	case config.ConnectionPoolIsolationProxy, config.ConnectionPoolIsolationAccount, config.ConnectionPoolIsolationAccountProxy:
		return mode
	default:
		return config.ConnectionPoolIsolationAccountProxy
	}
}

// maxUpstreamClients 获取最大客户端缓存数量
// 从配置中读取，无效值使用默认值

func (s *httpUpstreamService) maxUpstreamClients() int {
	if s.cfg == nil {
		return defaultMaxUpstreamClients
	}
	if s.cfg.Gateway.MaxUpstreamClients > 0 {
		return s.cfg.Gateway.MaxUpstreamClients
	}
	return defaultMaxUpstreamClients
}

// clientIdleTTL 获取客户端空闲回收阈值
// 从配置中读取，无效值使用默认值

func (s *httpUpstreamService) clientIdleTTL() time.Duration {
	if s.cfg == nil {
		return time.Duration(defaultClientIdleTTLSeconds) * time.Second
	}
	if s.cfg.Gateway.ClientIdleTTLSeconds > 0 {
		return time.Duration(s.cfg.Gateway.ClientIdleTTLSeconds) * time.Second
	}
	return time.Duration(defaultClientIdleTTLSeconds) * time.Second
}

// resolvePoolSettings 解析连接池配置
// 根据隔离策略和账户并发数动态调整连接池参数
//
// 参数:
//   - isolation: 隔离模式
//   - accountConcurrency: 账户并发限制
//
// 返回:
//   - poolSettings: 连接池配置
//
// 说明:
//   - 账户隔离模式下，连接池大小与账户并发数对应
//   - 这确保了单账户不会占用过多连接资源

func (s *httpUpstreamService) resolvePoolSettings(isolation string, accountConcurrency int) poolSettings {
	return s.resolvePoolSettingsWithOptions(isolation, accountConcurrency, upstreamRequestOptions{profile: service.HTTPUpstreamProfileDefault})
}

func (s *httpUpstreamService) resolvePoolSettingsWithOptions(isolation string, accountConcurrency int, opts upstreamRequestOptions) poolSettings {
	settings := defaultPoolSettings(s.cfg)
	if opts.profile == service.HTTPUpstreamProfileOpenAI {
		settings.responseHeaderTimeout = s.openAIResponseHeaderTimeout()
		settings.forceAttemptHTTP2 = opts.http2
	}
	settings.validateResolvedIP = s.shouldValidateResolvedIP()
	if s.cfg != nil {
		settings.allowPrivateHosts = s.cfg.Security.URLAllowlist.AllowPrivateHosts
		settings.privateHostConfig = s.cfg
	}
	// 账户隔离模式下，根据账户并发数调整连接池大小
	if (isolation == config.ConnectionPoolIsolationAccount || isolation == config.ConnectionPoolIsolationAccountProxy) && accountConcurrency > 0 {
		settings.maxIdleConns = accountConcurrency
		settings.maxIdleConnsPerHost = accountConcurrency
		settings.maxConnsPerHost = accountConcurrency
	}
	return settings
}

// buildPoolKey 构建连接池配置键
// 用于检测配置变更，配置变更时需要重建客户端
//
// 参数:
//   - isolation: 隔离模式
//   - accountConcurrency: 账户并发限制
//
// 返回:
//   - string: 配置键

func (s *httpUpstreamService) buildPoolKey(isolation string, accountConcurrency int) string {
	return s.buildPoolKeyWithOptions(isolation, accountConcurrency, upstreamRequestOptions{profile: service.HTTPUpstreamProfileDefault})
}

func (s *httpUpstreamService) buildPoolKeyWithOptions(isolation string, accountConcurrency int, opts upstreamRequestOptions) string {
	ssrfKey := "ssrf:true:false"
	if s.cfg != nil {
		ssrfKey = fmt.Sprintf("ssrf:%t:%t|%s",
			!s.cfg.Security.URLAllowlist.AllowPrivateHosts,
			s.cfg.Security.URLAllowlist.AllowPrivateHosts,
			hostexceptions.Key(s.cfg))
	}
	profileKey := fmt.Sprintf("|profile:%s|http2:%t|rht:%d", opts.profile, opts.http2, s.responseHeaderTimeoutKey(opts))
	if isolation == config.ConnectionPoolIsolationAccount || isolation == config.ConnectionPoolIsolationAccountProxy {
		if accountConcurrency > 0 {
			return fmt.Sprintf("account:%d|%s%s", accountConcurrency, ssrfKey, profileKey)
		}
	}
	return "default|" + ssrfKey + profileKey
}

func (s *httpUpstreamService) responseHeaderTimeoutKey(opts upstreamRequestOptions) int64 {
	if opts.profile == service.HTTPUpstreamProfileOpenAI {
		return int64(s.openAIResponseHeaderTimeout())
	}
	return int64(defaultPoolSettings(s.cfg).responseHeaderTimeout)
}
