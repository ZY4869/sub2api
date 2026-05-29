package service

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// apiUsageCache 缓存从 Anthropic API 获取的使用率数据（utilization, resets_at）
// 同时支持缓存错误响应（负缓存），防止 429 等错误导致的重试风暴
type apiUsageCache struct {
	response  *ClaudeUsageResponse
	err       error // 非 nil 表示缓存的错误（负缓存）
	timestamp time.Time
}

// windowStatsCache 缓存从本地数据库查询的窗口统计（requests, tokens, cost）
type windowStatsCache struct {
	stats     *WindowStats
	timestamp time.Time
}

// antigravityUsageCache 缓存 Antigravity 额度数据
type antigravityUsageCache struct {
	usageInfo *UsageInfo
	timestamp time.Time
}

const (
	apiCacheTTL         = 3 * time.Minute
	apiErrorCacheTTL    = 1 * time.Minute        // 负缓存 TTL：429 等错误缓存 1 分钟
	antigravityErrorTTL = 1 * time.Minute        // Antigravity 错误缓存 TTL（可恢复错误）
	apiQueryMaxJitter   = 800 * time.Millisecond // 用量查询最大随机延迟
	windowStatsCacheTTL = 1 * time.Minute
	openAIProbeCacheTTL = 10 * time.Minute
)

// UsageCache 封装账户使用量相关的缓存
type UsageCache struct {
	apiCache          sync.Map           // accountID -> *apiUsageCache
	windowStatsCache  sync.Map           // accountID -> *windowStatsCache
	antigravityCache  sync.Map           // accountID -> *antigravityUsageCache
	apiFlight         singleflight.Group // 防止同一账号的并发请求击穿缓存（Anthropic）
	antigravityFlight singleflight.Group // 防止同一 Antigravity 账号的并发请求击穿缓存
	openAIProbeCache  sync.Map           // accountID -> time.Time
}

// NewUsageCache 创建 UsageCache 实例
func NewUsageCache() *UsageCache {
	return &UsageCache{}
}
