package admin

import (
	"encoding/json"
	"time"
)

func firstCodexTime(obj map[string]any, paths ...[]string) (time.Time, bool) {
	for _, path := range paths {
		value, ok := codexPathValue(obj, path)
		if !ok {
			continue
		}
		if parsed, ok := parseCodexTimeValue(value); ok {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func parseCodexTimeValue(value any) (time.Time, bool) {
	switch v := value.(type) {
	case string:
		return parseCodexTimeString(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return codexUnixTime(n), true
		}
		if f, err := v.Float64(); err == nil {
			return codexUnixTime(int64(f)), true
		}
	case float64:
		return codexUnixTime(int64(v)), true
	case int:
		return codexUnixTime(int64(v)), true
	case int64:
		return codexUnixTime(v), true
	}
	return time.Time{}, false
}
