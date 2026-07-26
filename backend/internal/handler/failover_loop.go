package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

// TempUnscheduler 用于 HandleFailoverError 中同账号重试耗尽后的临时封禁。
// GatewayService 隐式实现此接口。
type TempUnscheduler interface {
	TempUnscheduleRetryableError(ctx context.Context, accountID int64, failoverErr *service.UpstreamFailoverError)
}

// FailoverAction 表示 failover 错误处理后的下一步动作
type FailoverAction int

const (
	// FailoverContinue 继续循环（同账号重试或切换账号，调用方统一 continue）
	FailoverContinue FailoverAction = iota
	// FailoverExhausted 切换次数耗尽（调用方应返回错误响应）
	FailoverExhausted
	// FailoverCanceled context 已取消（调用方应直接 return）
	FailoverCanceled
)

const (
	// maxSameAccountRetries 同账号重试次数上限（针对 RetryableOnSameAccount 错误）
	maxSameAccountRetries = 3
	// sameAccountRetryDelay 同账号重试间隔
	sameAccountRetryDelay = 500 * time.Millisecond
	// singleAccountBackoffDelay 单账号分组 503 退避重试固定延时。
	// Service 层在 SingleAccountRetry 模式下已做充分原地重试（最多 3 次、总等待 30s），
	// Handler 层只需短暂间隔后重新进入 Service 层即可。
	singleAccountBackoffDelay = 2 * time.Second
	// maxSelectionBackoffs 选号耗尽后的退避重试轮数上限。
	// 容量类错误多为瞬时抖动，重试两轮仍不可用即认为该分组确实不可用，
	// 避免把切换额度耗在同一批坏号上、让客户端白等。
	maxSelectionBackoffs = 2
)

// isCapacityFailoverStatus 判断上游状态码是否属于"容量/瞬时不可用"类错误。
// 这类错误值得退避后重试同一账号；而 401/403/429 属确定性失败，
// 在同一次请求内重试只会白白消耗切换额度。
func isCapacityFailoverStatus(statusCode int) bool {
	return statusCode == http.StatusServiceUnavailable || statusCode == 529
}

// FailoverState 跨循环迭代共享的 failover 状态
type FailoverState struct {
	SwitchCount           int
	MaxSwitches           int
	FailedAccountIDs      map[int64]struct{}
	SameAccountRetryCount map[int64]int
	// LastStatusByAccount 记录每个账号最后一次 failover 的上游状态码，
	// 用于选号耗尽后只放回"容量类"失败的账号。
	LastStatusByAccount   map[int64]int
	SelectionBackoffCount int
	LastFailoverErr       *service.UpstreamFailoverError
	ForceCacheBilling     bool
	hasBoundSession       bool
}

// NewFailoverState 创建 failover 状态
func NewFailoverState(maxSwitches int, hasBoundSession bool) *FailoverState {
	return &FailoverState{
		MaxSwitches:           maxSwitches,
		FailedAccountIDs:      make(map[int64]struct{}),
		SameAccountRetryCount: make(map[int64]int),
		LastStatusByAccount:   make(map[int64]int),
		hasBoundSession:       hasBoundSession,
	}
}

// HandleFailoverError 处理 UpstreamFailoverError，返回下一步动作。
// 包含：缓存计费判断、同账号重试、临时封禁、切换计数、Antigravity 延时。
func (s *FailoverState) HandleFailoverError(
	ctx context.Context,
	gatewayService TempUnscheduler,
	accountID int64,
	platform string,
	failoverErr *service.UpstreamFailoverError,
) FailoverAction {
	s.LastFailoverErr = failoverErr

	// 同账号重试：对 RetryableOnSameAccount 的临时性错误，先在同一账号上重试
	if failoverErr.RetryableOnSameAccount && s.SameAccountRetryCount[accountID] < maxSameAccountRetries {
		s.SameAccountRetryCount[accountID]++
		logger.FromContext(ctx).Warn("gateway.failover_same_account_retry",
			zap.Int64("account_id", accountID),
			zap.Int("upstream_status", failoverErr.StatusCode),
			zap.Int("same_account_retry_count", s.SameAccountRetryCount[accountID]),
			zap.Int("same_account_retry_max", maxSameAccountRetries),
		)
		if !sleepWithContext(ctx, sameAccountRetryDelay) {
			return FailoverCanceled
		}
		return FailoverContinue
	}

	// 同账号重试用尽，执行临时封禁
	if failoverErr.RetryableOnSameAccount {
		gatewayService.TempUnscheduleRetryableError(ctx, accountID, failoverErr)
	}

	// 加入失败列表，并记录该账号最后一次失败的上游状态码
	s.FailedAccountIDs[accountID] = struct{}{}
	if s.LastStatusByAccount == nil {
		s.LastStatusByAccount = make(map[int64]int)
	}
	s.LastStatusByAccount[accountID] = failoverErr.StatusCode

	// 缓存计费判断只在真实跨账号切换或耗尽路径上生效。
	if needForceCacheBilling(s.hasBoundSession, failoverErr) {
		s.ForceCacheBilling = true
	}

	// 检查是否耗尽
	if s.SwitchCount >= s.MaxSwitches {
		return FailoverExhausted
	}

	// 递增切换计数
	s.SwitchCount++
	logger.FromContext(ctx).Warn("gateway.failover_switch_account",
		zap.Int64("account_id", accountID),
		zap.Int("upstream_status", failoverErr.StatusCode),
		zap.Int("switch_count", s.SwitchCount),
		zap.Int("max_switches", s.MaxSwitches),
	)

	// Antigravity 平台换号线性递增延时
	if platform == service.PlatformAntigravity {
		delay := time.Duration(s.SwitchCount-1) * time.Second
		if !sleepWithContext(ctx, delay) {
			return FailoverCanceled
		}
	}

	return FailoverContinue
}

// HandleSelectionExhausted 处理选号失败（所有候选账号都在排除列表中）时的退避重试决策。
// 触发条件是失败账号中存在最后一次失败为容量类错误（503 MODEL_CAPACITY_EXHAUSTED /
// 529 overloaded）的账号。按逐账号状态判定而非全局最后一次错误，避免
// "A 账号 503 可退避、B 账号最后 429"时因全局错误是 429 而提前放弃 A。
// 典型场景是 Antigravity 单账号分组，但对所有平台同样适用。
//
// 退避后只把"最后一次失败属容量类"的账号放回候选池；
// 401/403/429 等确定性失败在本次请求内保持排除，避免与坏号来回乒乓耗尽切换额度。
//
// 返回 FailoverContinue 时，调用方应设置 SingleAccountRetry context 并 continue。
// 返回 FailoverExhausted 时，调用方应返回错误响应。
// 返回 FailoverCanceled 时，调用方应直接 return。
func (s *FailoverState) HandleSelectionExhausted(ctx context.Context) FailoverAction {
	if s.LastFailoverErr == nil || s.SelectionBackoffCount >= maxSelectionBackoffs {
		return FailoverExhausted
	}

	retryable := make([]int64, 0, len(s.FailedAccountIDs))
	for accountID := range s.FailedAccountIDs {
		if isCapacityFailoverStatus(s.LastStatusByAccount[accountID]) {
			retryable = append(retryable, accountID)
		}
	}
	if len(retryable) == 0 {
		return FailoverExhausted
	}

	logger.FromContext(ctx).Warn("gateway.failover_single_account_backoff",
		zap.Duration("backoff_delay", singleAccountBackoffDelay),
		zap.Int("switch_count", s.SwitchCount),
		zap.Int("max_switches", s.MaxSwitches),
		zap.Int("selection_backoff_count", s.SelectionBackoffCount),
		zap.Int("retryable_accounts", len(retryable)),
	)
	if !sleepWithContext(ctx, singleAccountBackoffDelay) {
		return FailoverCanceled
	}
	s.SelectionBackoffCount++
	logger.FromContext(ctx).Warn("gateway.failover_single_account_retry",
		zap.Int("switch_count", s.SwitchCount),
		zap.Int("max_switches", s.MaxSwitches),
		zap.Int("selection_backoff_count", s.SelectionBackoffCount),
		zap.Int("selection_backoff_max", maxSelectionBackoffs),
	)
	for _, accountID := range retryable {
		delete(s.FailedAccountIDs, accountID)
		delete(s.LastStatusByAccount, accountID)
	}
	return FailoverContinue
}

// needForceCacheBilling 判断 failover 时是否需要强制缓存计费。
// 粘性会话切换账号、或上游明确标记时，将 input_tokens 转为 cache_read 计费。
func needForceCacheBilling(hasBoundSession bool, failoverErr *service.UpstreamFailoverError) bool {
	return hasBoundSession || (failoverErr != nil && failoverErr.ForceCacheBilling)
}

// sleepWithContext 等待指定时长，返回 false 表示 context 已取消。
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
