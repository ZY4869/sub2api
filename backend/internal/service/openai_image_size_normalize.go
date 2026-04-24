package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type openAIImageSizeShorthand struct {
	ImageSizeTier string
	AspectRatio   string
}

func normalizeOpenAIImageSizeWithAspect(sizeRaw string, imageSizeRaw string, aspectRatioRaw string) (string, bool, string, string) {
	sizeRaw = strings.TrimSpace(sizeRaw)
	imageSizeRaw = strings.TrimSpace(imageSizeRaw)
	aspectRatioRaw = strings.TrimSpace(aspectRatioRaw)

	if normalized, ok := normalizeOpenAIExplicitSize(sizeRaw); ok {
		return normalized, normalized != sizeRaw, "", ""
	}

	shorthand, usedShorthand := parseOpenAIImageSizeShorthand(sizeRaw)
	if usedShorthand {
		if shorthand.ImageSizeTier != "" {
			imageSizeRaw = shorthand.ImageSizeTier
		}
		if shorthand.AspectRatio != "" {
			aspectRatioRaw = shorthand.AspectRatio
		}
	}

	if imageSizeRaw == "" && aspectRatioRaw == "" {
		return sizeRaw, false, "", ""
	}

	if imageSizeRaw == "" {
		imageSizeRaw = OpenAIImageSizeTier2K
	}
	if aspectRatioRaw == "" {
		aspectRatioRaw = "1:1"
	}

	maxSide, ok := parseOpenAIImageSizeTierMaxSide(imageSizeRaw)
	if !ok {
		return "", false, "image_size_invalid", "image_size must be 1K, 2K, or 4K"
	}
	ratioW, ratioH, ok := parseOpenAIImageAspectRatio(aspectRatioRaw)
	if !ok {
		return "", false, "image_aspect_ratio_invalid", "aspect_ratio must be in the form W:H, e.g. 16:9"
	}

	width, height := resolveOpenAIImageSizeForAspect(maxSide, ratioW, ratioH)
	if width <= 0 || height <= 0 {
		return "", false, "image_size_invalid", "resolved size dimensions must be positive integers"
	}
	if width > 3840 || height > 3840 {
		return "", false, "image_size_too_large", "custom image size cannot exceed 3840px on either side"
	}
	normalized := fmt.Sprintf("%dx%d", width, height)
	return normalized, normalized != sizeRaw, "", ""
}

func normalizeOpenAIExplicitSize(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	if strings.EqualFold(trimmed, "auto") {
		return "auto", true
	}
	if width, height, ok := parseOpenAIImageSizeDimensions(trimmed); ok {
		return fmt.Sprintf("%dx%d", width, height), true
	}
	return "", false
}

func parseOpenAIImageSizeShorthand(value string) (openAIImageSizeShorthand, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return openAIImageSizeShorthand{}, false
	}

	parts := strings.Fields(trimmed)
	switch len(parts) {
	case 1:
		if _, ok := parseOpenAIImageSizeTierMaxSide(parts[0]); ok {
			return openAIImageSizeShorthand{ImageSizeTier: parts[0], AspectRatio: "1:1"}, true
		}
		if _, _, ok := parseOpenAIImageAspectRatio(parts[0]); ok {
			return openAIImageSizeShorthand{ImageSizeTier: OpenAIImageSizeTier2K, AspectRatio: parts[0]}, true
		}
		return openAIImageSizeShorthand{}, false
	case 2:
		first := parts[0]
		second := parts[1]

		_, firstIsTier := parseOpenAIImageSizeTierMaxSide(first)
		_, secondIsTier := parseOpenAIImageSizeTierMaxSide(second)
		_, _, firstIsRatio := parseOpenAIImageAspectRatio(first)
		_, _, secondIsRatio := parseOpenAIImageAspectRatio(second)

		switch {
		case firstIsTier && secondIsRatio:
			return openAIImageSizeShorthand{ImageSizeTier: first, AspectRatio: second}, true
		case firstIsRatio && secondIsTier:
			return openAIImageSizeShorthand{ImageSizeTier: second, AspectRatio: first}, true
		case firstIsTier && !secondIsTier && strings.EqualFold(second, "auto"):
			return openAIImageSizeShorthand{}, false
		default:
			return openAIImageSizeShorthand{}, true
		}
	default:
		return openAIImageSizeShorthand{}, true
	}
}

func parseOpenAIImageSizeTierMaxSide(value string) (int, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "1k":
		return 1024, true
	case "2k":
		return 2048, true
	case "4k":
		return 3840, true
	default:
		return 0, false
	}
}

func parseOpenAIImageAspectRatio(value string) (int, int, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return 0, 0, false
	}

	normalized = strings.NewReplacer("/", ":", " ", "").Replace(normalized)
	parts := strings.Split(normalized, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	left, errLeft := strconv.Atoi(strings.TrimSpace(parts[0]))
	right, errRight := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errLeft != nil || errRight != nil {
		return 0, 0, false
	}
	if left <= 0 || right <= 0 {
		return 0, 0, false
	}
	return left, right, true
}

func resolveOpenAIImageSizeForAspect(maxSide int, ratioW int, ratioH int) (int, int) {
	maxSide = max(maxSide, 1)
	if ratioW <= 0 || ratioH <= 0 {
		return maxSide, maxSide
	}

	if ratioW >= ratioH {
		height := int(math.Round(float64(maxSide) * float64(ratioH) / float64(ratioW)))
		if height <= 0 {
			height = 1
		}
		return maxSide, height
	}
	width := int(math.Round(float64(maxSide) * float64(ratioW) / float64(ratioH)))
	if width <= 0 {
		width = 1
	}
	return width, maxSide
}

