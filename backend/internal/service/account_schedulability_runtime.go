package service

import "time"

// WindowCostSchedulability 窗口费用调度状态
type WindowCostSchedulability int

const (
	// WindowCostSchedulable 可正常调度
	WindowCostSchedulable WindowCostSchedulability = iota
	// WindowCostStickyOnly 仅允许粘性会话
	WindowCostStickyOnly
	// WindowCostNotSchedulable 完全不可调度
	WindowCostNotSchedulable
)

// GetWindowCostLimit 获取 5h 窗口费用阈值（美元）
// 返回 0 表示未启用
func (a *Account) GetWindowCostLimit() float64 {
	if a.Extra == nil {
		return 0
	}
	if v, ok := a.Extra["window_cost_limit"]; ok {
		return parseExtraFloat64(v)
	}
	return 0
}

// GetWindowCostStickyReserve 获取粘性会话预留额度（美元）
// 默认值为 10
func (a *Account) GetWindowCostStickyReserve() float64 {
	if a.Extra == nil {
		return 10.0
	}
	if v, ok := a.Extra["window_cost_sticky_reserve"]; ok {
		val := parseExtraFloat64(v)
		if val > 0 {
			return val
		}
	}
	return 10.0
}

// GetMaxSessions 获取最大并发会话数
// 返回 0 表示未启用
func (a *Account) GetMaxSessions() int {
	if a.Extra == nil {
		return 0
	}
	if v, ok := a.Extra["max_sessions"]; ok {
		return parseExtraInt(v)
	}
	return 0
}

// GetSessionIdleTimeoutMinutes 获取会话空闲超时分钟数
// 默认值为 5 分钟
func (a *Account) GetSessionIdleTimeoutMinutes() int {
	if a.Extra == nil {
		return 5
	}
	if v, ok := a.Extra["session_idle_timeout_minutes"]; ok {
		val := parseExtraInt(v)
		if val > 0 {
			return val
		}
	}
	return 5
}

// GetBaseRPM 获取基础 RPM 限制
// 返回 0 表示未启用（负数视为无效配置，按 0 处理）
func (a *Account) GetBaseRPM() int {
	if a.Extra == nil {
		return 0
	}
	if v, ok := a.Extra["base_rpm"]; ok {
		val := parseExtraInt(v)
		if val > 0 {
			return val
		}
	}
	return 0
}

// GetRPMStrategy 获取 RPM 策略
// "tiered" = 三区模型（默认）, "sticky_exempt" = 粘性豁免
func (a *Account) GetRPMStrategy() string {
	if a.Extra == nil {
		return "tiered"
	}
	if v, ok := a.Extra["rpm_strategy"]; ok {
		if s, ok := v.(string); ok && s == "sticky_exempt" {
			return "sticky_exempt"
		}
	}
	return "tiered"
}

// GetRPMStickyBuffer 获取 RPM 粘性缓冲数量
// tiered 模式下的黄区大小，默认为 base_rpm 的 20%（至少 1）
func (a *Account) GetRPMStickyBuffer() int {
	if a.Extra == nil {
		return 0
	}
	if v, ok := a.Extra["rpm_sticky_buffer"]; ok {
		val := parseExtraInt(v)
		if val > 0 {
			return val
		}
	}
	base := a.GetBaseRPM()
	buffer := base / 5
	if buffer < 1 && base > 0 {
		buffer = 1
	}
	return buffer
}

// CheckRPMSchedulability 根据当前 RPM 计数检查调度状态
// 复用 WindowCostSchedulability 三态：Schedulable / StickyOnly / NotSchedulable
func (a *Account) CheckRPMSchedulability(currentRPM int) WindowCostSchedulability {
	baseRPM := a.GetBaseRPM()
	if baseRPM <= 0 {
		return WindowCostSchedulable
	}

	if currentRPM < baseRPM {
		return WindowCostSchedulable
	}

	strategy := a.GetRPMStrategy()
	if strategy == "sticky_exempt" {
		return WindowCostStickyOnly // 粘性豁免无红区
	}

	// tiered: 黄区 + 红区
	buffer := a.GetRPMStickyBuffer()
	if currentRPM < baseRPM+buffer {
		return WindowCostStickyOnly
	}
	return WindowCostNotSchedulable
}

// CheckWindowCostSchedulability 根据当前窗口费用检查调度状态
// - 费用 < 阈值: WindowCostSchedulable（可正常调度）
// - 费用 >= 阈值 且 < 阈值+预留: WindowCostStickyOnly（仅粘性会话）
// - 费用 >= 阈值+预留: WindowCostNotSchedulable（不可调度）
func (a *Account) CheckWindowCostSchedulability(currentWindowCost float64) WindowCostSchedulability {
	limit := a.GetWindowCostLimit()
	if limit <= 0 {
		return WindowCostSchedulable
	}

	if currentWindowCost < limit {
		return WindowCostSchedulable
	}

	stickyReserve := a.GetWindowCostStickyReserve()
	if currentWindowCost < limit+stickyReserve {
		return WindowCostStickyOnly
	}

	return WindowCostNotSchedulable
}

// GetCurrentWindowStartTime 获取当前有效的窗口开始时间
// 逻辑：
// 1. 如果窗口未过期（SessionWindowEnd 存在且在当前时间之后），使用记录的 SessionWindowStart
// 2. 否则（窗口过期或未设置），使用新的预测窗口开始时间（从当前整点开始）
func (a *Account) GetCurrentWindowStartTime() time.Time {
	now := time.Now()

	// 窗口未过期，使用记录的窗口开始时间
	if a.SessionWindowStart != nil && a.SessionWindowEnd != nil && now.Before(*a.SessionWindowEnd) {
		return *a.SessionWindowStart
	}

	// 窗口已过期或未设置，预测新的窗口开始时间（从当前整点开始）
	// 与 ratelimit_service.go 中 UpdateSessionWindow 的预测逻辑保持一致
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
}
