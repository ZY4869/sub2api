package modelregistry

import (
	"regexp"
	"strings"
)

var (
	modelDateVersionSuffixPattern = regexp.MustCompile(`-(?:\d{8}|\d{4}-\d{2}-\d{2})(?:-[^-\s]+:\d+)?$`)
	versionPairPattern            = regexp.MustCompile(`-(\d+)-(\d+)`)
	modelSeparatorPattern         = regexp.MustCompile(`[\s_]+`)
	modelDashPattern              = regexp.MustCompile(`-+`)
)

func NormalizeID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "/")
	value = strings.TrimPrefix(value, "models/")
	value = strings.TrimPrefix(value, "publishers/google/models/")
	if idx := strings.LastIndex(value, "/publishers/google/models/"); idx != -1 {
		value = value[idx+len("/publishers/google/models/"):]
	}
	if idx := strings.LastIndex(value, "/models/"); idx != -1 {
		value = value[idx+len("/models/"):]
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizePlatform(platform string) string {
	platform = strings.TrimSpace(strings.ToLower(platform))
	switch platform {
	case "claude":
		return "anthropic"
	case "xai":
		return "grok"
	default:
		return platform
	}
}

func NormalizePlatformFamily(platform string) string {
	platform = NormalizePlatform(platform)
	switch platform {
	case "kiro":
		return "anthropic"
	case "baidu", "baidu_document_ai":
		return "baidu_document_ai"
	default:
		return platform
	}
}

func StripDateVersionSuffix(value string) string {
	normalized := NormalizeID(value)
	return modelDateVersionSuffixPattern.ReplaceAllString(normalized, "")
}

func AlternateVersionVariants(value string) []string {
	normalized := NormalizeID(value)
	if normalized == "" {
		return nil
	}
	items := make([]string, 0, 12)
	add := func(item string) {
		item = NormalizeID(item)
		if item == "" {
			return
		}
		for _, existing := range items {
			if existing == item {
				return
			}
		}
		items = append(items, item)
	}
	add(normalized)
	add(strings.ReplaceAll(normalized, ".", "-"))
	for _, item := range modelShapeVariants(normalized) {
		add(item)
	}
	add(versionPairPattern.ReplaceAllString(normalized, "-$1.$2"))
	base := StripDateVersionSuffix(normalized)
	if base != normalized {
		add(base)
		add(strings.ReplaceAll(base, ".", "-"))
		for _, item := range modelShapeVariants(base) {
			add(item)
		}
		add(versionPairPattern.ReplaceAllString(base, "-$1.$2"))
	}
	return items
}

func modelShapeVariants(value string) []string {
	items := make([]string, 0, 8)
	add := func(item string) {
		item = NormalizeID(item)
		item = modelSeparatorPattern.ReplaceAllString(item, "-")
		item = modelDashPattern.ReplaceAllString(item, "-")
		item = strings.Trim(item, "-")
		if item == "" {
			return
		}
		for _, existing := range items {
			if existing == item {
				return
			}
		}
		items = append(items, item)
	}

	add(value)
	if idx := strings.LastIndex(value, "/"); idx != -1 && idx+1 < len(value) {
		add(value[idx+1:])
	}
	for _, item := range append([]string(nil), items...) {
		switch {
		case strings.HasSuffix(item, ":free"):
			add(strings.TrimSuffix(item, ":free"))
			add(strings.TrimSuffix(item, ":free") + "-free")
		case strings.HasSuffix(item, "-free"):
			add(strings.TrimSuffix(item, "-free"))
		}
	}
	return items
}
