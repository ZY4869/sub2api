package service

import (
	"encoding/json"
	"strconv"
	"strings"
)

func (a *Account) IsCustomErrorCodesEnabled() bool {
	if a.Type != AccountTypeAPIKey || a.Credentials == nil {
		return false
	}
	if v, ok := a.Credentials["custom_error_codes_enabled"]; ok {
		if enabled, ok := v.(bool); ok {
			return enabled
		}
	}
	return false
}

// IsPoolMode 检查 API Key 账号是否启用池模式。
// 池模式下，上游错误不标记本地账号状态，而是在同一账号上重试。
func (a *Account) IsPoolMode() bool {
	if !a.IsAPIKeyOrBedrock() || a.Credentials == nil {
		return false
	}
	if v, ok := a.Credentials["pool_mode"]; ok {
		if enabled, ok := v.(bool); ok {
			return enabled
		}
	}
	return false
}

const (
	defaultPoolModeRetryCount = 3
	maxPoolModeRetryCount     = 10
)

var defaultPoolModeRetryStatusCodes = []int{401, 403, 429}

// GetPoolModeRetryCount 返回池模式同账号重试次数。
// 未配置或配置非法时回退为默认值 3；小于 0 按 0 处理；过大则截断到 10。
func (a *Account) GetPoolModeRetryCount() int {
	if a == nil || !a.IsPoolMode() || a.Credentials == nil {
		return defaultPoolModeRetryCount
	}
	raw, ok := a.Credentials["pool_mode_retry_count"]
	if !ok || raw == nil {
		return defaultPoolModeRetryCount
	}
	count := parsePoolModeRetryCount(raw)
	if count < 0 {
		return 0
	}
	if count > maxPoolModeRetryCount {
		return maxPoolModeRetryCount
	}
	return count
}

func parsePoolModeRetryCount(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
	}
	return defaultPoolModeRetryCount
}

// GetPoolModeRetryStatusCodes returns same-account retryable statuses for pool mode.
// Missing or invalid config preserves the legacy default: 401, 403 and 429.
func (a *Account) GetPoolModeRetryStatusCodes() []int {
	if a == nil || !a.IsPoolMode() || a.Credentials == nil {
		return append([]int(nil), defaultPoolModeRetryStatusCodes...)
	}
	raw, ok := a.Credentials["pool_mode_retry_status_codes"]
	if !ok || raw == nil {
		return append([]int(nil), defaultPoolModeRetryStatusCodes...)
	}
	codes := parsePoolModeRetryStatusCodes(raw)
	if len(codes) == 0 {
		return append([]int(nil), defaultPoolModeRetryStatusCodes...)
	}
	return codes
}

func parsePoolModeRetryStatusCodes(value any) []int {
	appendCode := func(out []int, seen map[int]struct{}, code int) []int {
		if code < 100 || code > 599 {
			return out
		}
		if _, ok := seen[code]; ok {
			return out
		}
		seen[code] = struct{}{}
		return append(out, code)
	}

	seen := map[int]struct{}{}
	out := []int{}
	switch v := value.(type) {
	case []int:
		for _, item := range v {
			out = appendCode(out, seen, item)
		}
	case []int64:
		for _, item := range v {
			out = appendCode(out, seen, int(item))
		}
	case []float64:
		for _, item := range v {
			out = appendCode(out, seen, int(item))
		}
	case []any:
		for _, item := range v {
			out = appendCode(out, seen, parsePoolModeRetryStatusCode(item))
		}
	case string:
		for _, part := range strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
		}) {
			out = appendCode(out, seen, parsePoolModeRetryStatusCode(part))
		}
	}
	return out
}

func parsePoolModeRetryStatusCode(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
	}
	return 0
}

// IsPoolModeRetryableStatus checks whether a status should retry on the same account.
func (a *Account) IsPoolModeRetryableStatus(statusCode int) bool {
	if a == nil || !a.IsPoolMode() {
		return false
	}
	for _, code := range a.GetPoolModeRetryStatusCodes() {
		if code == statusCode {
			return true
		}
	}
	return false
}

func (a *Account) GetCustomErrorCodes() []int {
	if a.Credentials == nil {
		return nil
	}
	raw, ok := a.Credentials["custom_error_codes"]
	if !ok || raw == nil {
		return nil
	}
	if arr, ok := raw.([]any); ok {
		result := make([]int, 0, len(arr))
		for _, v := range arr {
			if f, ok := v.(float64); ok {
				result = append(result, int(f))
			}
		}
		return result
	}
	return nil
}

func (a *Account) ShouldHandleErrorCode(statusCode int) bool {
	if !a.IsCustomErrorCodesEnabled() {
		return true
	}
	codes := a.GetCustomErrorCodes()
	if len(codes) == 0 {
		return true
	}
	for _, code := range codes {
		if code == statusCode {
			return true
		}
	}
	return false
}

func (a *Account) IsInterceptWarmupEnabled() bool {
	if a.Credentials == nil {
		return false
	}
	if v, ok := a.Credentials["intercept_warmup_requests"]; ok {
		if enabled, ok := v.(bool); ok {
			return enabled
		}
	}
	return false
}
