package service

import (
	"strconv"
	"strings"
)

type soraWatermarkOptions struct {
	Enabled           bool
	ParseMethod       string
	ParseURL          string
	ParseToken        string
	FallbackOnFailure bool
	DeletePost        bool
}

type soraCharacterOptions struct {
	SetPublic           bool
	DeleteAfterGenerate bool
}

func parseSoraWatermarkOptions(body map[string]any) soraWatermarkOptions {
	opts := soraWatermarkOptions{
		Enabled:           parseBoolWithDefault(body, "watermark_free", false),
		ParseMethod:       strings.ToLower(strings.TrimSpace(parseStringWithDefault(body, "watermark_parse_method", "third_party"))),
		ParseURL:          strings.TrimSpace(parseStringWithDefault(body, "watermark_parse_url", "")),
		ParseToken:        strings.TrimSpace(parseStringWithDefault(body, "watermark_parse_token", "")),
		FallbackOnFailure: parseBoolWithDefault(body, "watermark_fallback_on_failure", true),
		DeletePost:        parseBoolWithDefault(body, "watermark_delete_post", false),
	}
	if opts.ParseMethod == "" {
		opts.ParseMethod = "third_party"
	}
	return opts
}

func parseSoraCharacterOptions(body map[string]any) soraCharacterOptions {
	return soraCharacterOptions{
		SetPublic:           parseBoolWithDefault(body, "character_set_public", true),
		DeleteAfterGenerate: parseBoolWithDefault(body, "character_delete_after_generate", true),
	}
}

func parseSoraVideoCount(body map[string]any) int {
	if body == nil {
		return 1
	}
	keys := []string{"video_count", "videos", "n_variants"}
	for _, key := range keys {
		count := parseIntWithDefault(body, key, 0)
		if count > 0 {
			return clampInt(count, 1, 3)
		}
	}
	return 1
}

func parseBoolWithDefault(body map[string]any, key string, def bool) bool {
	if body == nil {
		return def
	}
	val, ok := body[key]
	if !ok {
		return def
	}
	switch typed := val.(type) {
	case bool:
		return typed
	case int:
		return typed != 0
	case int32:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	case string:
		typed = strings.ToLower(strings.TrimSpace(typed))
		if typed == "true" || typed == "1" || typed == "yes" {
			return true
		}
		if typed == "false" || typed == "0" || typed == "no" {
			return false
		}
	}
	return def
}

func parseStringWithDefault(body map[string]any, key, def string) string {
	if body == nil {
		return def
	}
	val, ok := body[key]
	if !ok {
		return def
	}
	if str, ok := val.(string); ok {
		return str
	}
	return def
}

func parseIntWithDefault(body map[string]any, key string, def int) int {
	if body == nil {
		return def
	}
	val, ok := body[key]
	if !ok {
		return def
	}
	switch typed := val.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return def
}

func clampInt(v, minVal, maxVal int) int {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

func extractSoraCameoIDs(body map[string]any) []string {
	if body == nil {
		return nil
	}
	raw, ok := body["cameo_ids"]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			str, ok := item.(string)
			if !ok {
				continue
			}
			str = strings.TrimSpace(str)
			if str != "" {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}
