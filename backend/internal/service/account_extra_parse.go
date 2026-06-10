package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// getExtraFloat64 从 Extra 中读取指定 key 的 float64 值
func (a *Account) getExtraFloat64(key string) float64 {
	if a.Extra == nil {
		return 0
	}
	if v, ok := a.Extra[key]; ok {
		return parseExtraFloat64(v)
	}
	return 0
}

func (a *Account) getExtraCurrencyMap(key string) map[string]float64 {
	if a == nil || a.Extra == nil {
		return nil
	}
	raw, ok := a.Extra[key]
	if !ok {
		return nil
	}
	values := map[string]float64{}
	switch typed := raw.(type) {
	case map[string]float64:
		values = typed
	case map[string]any:
		for currency, value := range typed {
			values[currency] = parseExtraFloat64(value)
		}
	default:
		return nil
	}
	return cloneBillingStringMapFloat64(values)
}

// getExtraTime 从 Extra 中读取 RFC3339 时间戳
func (a *Account) getExtraTime(key string) time.Time {
	if a.Extra == nil {
		return time.Time{}
	}
	if v, ok := a.Extra[key]; ok {
		if s, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t
			}
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// GetExtraTime returns a RFC3339/RFC3339Nano timestamp from Extra.
func (a *Account) GetExtraTime(key string) time.Time {
	if a == nil {
		return time.Time{}
	}
	return a.getExtraTime(key)
}

// getExtraString 从 Extra 中读取指定 key 的字符串值
func (a *Account) getExtraString(key string) string {
	if a.Extra == nil {
		return ""
	}
	if v, ok := a.Extra[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getExtraInt 从 Extra 中读取指定 key 的 int 值
func (a *Account) getExtraInt(key string) int {
	if a.Extra == nil {
		return 0
	}
	if v, ok := a.Extra[key]; ok {
		return int(parseExtraFloat64(v))
	}
	return 0
}

// parseExtraFloat64 从 extra 字段解析 float64 值
func parseExtraFloat64(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return 0
}

// ParseExtraInt 从 extra 字段的 any 值解析为 int。
// 支持 int, int64, float64, json.Number, string 类型，无法解析时返回 0。
func ParseExtraInt(value any) int {
	return parseExtraInt(value)
}

func parseExtraInt(value any) int {
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
