package service

import (
	"strings"
	"time"
)

const (
	AccountAutoRenewPeriodMonth   = "month"
	AccountAutoRenewPeriodQuarter = "quarter"
	AccountAutoRenewPeriodYear    = "year"
)

type Account struct {
	ID          int64
	Name        string
	Notes       *string
	Platform    string
	Type        string
	Credentials map[string]any
	Extra       map[string]any
	ProxyID     *int64
	Concurrency int
	Priority    int
	// RateMultiplier 账号计费倍率（>=0，允许 0 表示该账号计费为 0）。
	// 使用指针用于兼容旧版本调度缓存（Redis）中缺字段的情况：nil 表示按 1.0 处理。
	RateMultiplier         *float64
	LoadFactor             *int // 调度负载因子；nil 表示使用 Concurrency
	Status                 string
	LifecycleState         string
	LifecycleReasonCode    string
	LifecycleReasonMessage string
	ErrorMessage           string
	LastUsedAt             *time.Time
	ExpiresAt              *time.Time
	AutoPauseOnExpired     bool
	AutoRenewEnabled       bool
	AutoRenewPeriod        string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	BlacklistedAt          *time.Time
	BlacklistPurgeAt       *time.Time

	Schedulable bool

	RateLimitedAt    *time.Time
	RateLimitResetAt *time.Time
	OverloadUntil    *time.Time

	TempUnschedulableUntil  *time.Time
	TempUnschedulableReason string

	SessionWindowStart  *time.Time
	SessionWindowEnd    *time.Time
	SessionWindowStatus string

	Proxy         *Proxy
	AccountGroups []AccountGroup
	GroupIDs      []int64
	Groups        []*Group

	// model_mapping 热路径缓存（非持久化字段）
	modelMappingCache               map[string]string
	modelMappingCacheReady          bool
	modelMappingCacheCredentialsPtr uintptr
	modelMappingCacheRawPtr         uintptr
	modelMappingCacheRawLen         int
	modelMappingCacheRawSig         uint64
}

type TempUnschedulableRule struct {
	ErrorCode       int      `json:"error_code"`
	Keywords        []string `json:"keywords"`
	DurationMinutes int      `json:"duration_minutes"`
	Description     string   `json:"description"`
}

func normalizeAccountNotes(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
