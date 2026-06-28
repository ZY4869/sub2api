package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	openAIImageMaxEdge   = 3840
	openAIImageMinPixels = 655360
	openAIImageMaxPixels = 8294400
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
		aspectRatioRaw = defaultOpenAIImageAspectRatioForTier(imageSizeRaw)
	}

	_, ok := parseOpenAIImageSizeTierMaxSide(imageSizeRaw)
	if !ok {
		return "", false, "image_size_invalid", "image_size must be 1K, 2K, or 4K"
	}
	ratioW, ratioH, ok := parseOpenAIImageAspectRatio(aspectRatioRaw)
	if !ok {
		return "", false, "image_aspect_ratio_invalid", "aspect_ratio must be in the form W:H, e.g. 16:9"
	}

	maxSide := resolveOpenAIImageSizeTierLongEdge(imageSizeRaw, ratioW, ratioH)
	width, height := resolveOpenAIImageSizeForAspect(maxSide, ratioW, ratioH)
	if errCode, errMessage := validateOpenAIImageSizeDimensions(width, height); errCode != "" {
		return "", false, errCode, errMessage
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
			return openAIImageSizeShorthand{ImageSizeTier: parts[0], AspectRatio: defaultOpenAIImageAspectRatioForTier(parts[0])}, true
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

func resolveOpenAIImageSizeTierLongEdge(value string, ratioW int, ratioH int) int {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "1k":
		if ratioW == ratioH {
			return 1024
		}
		return 1536
	case "2k":
		return 2048
	case "4k":
		return 3840
	default:
		return 2048
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
	maxSide = max(maxSide, 16)
	if ratioW <= 0 || ratioH <= 0 {
		return maxSide, maxSide
	}
	maxSide = resolveOpenAIImageMaxSideForPixelLimit(maxSide, ratioW, ratioH)

	if ratioW >= ratioH {
		height := roundOpenAIImageDimensionToMultipleOf16(float64(maxSide) * float64(ratioH) / float64(ratioW))
		if height <= 0 {
			height = 16
		}
		return maxSide, height
	}
	width := roundOpenAIImageDimensionToMultipleOf16(float64(maxSide) * float64(ratioW) / float64(ratioH))
	if width <= 0 {
		width = 16
	}
	return width, maxSide
}

func defaultOpenAIImageAspectRatioForTier(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), OpenAIImageSizeTier4K) {
		return "16:9"
	}
	return "1:1"
}

func validateOpenAIImageSizeDimensions(width int, height int) (string, string) {
	if width <= 0 || height <= 0 {
		return "image_size_invalid", "size dimensions must be positive integers"
	}
	if width > openAIImageMaxEdge || height > openAIImageMaxEdge {
		return "image_size_too_large", "custom image size cannot exceed 3840px on either side"
	}
	if width%16 != 0 || height%16 != 0 {
		return "image_size_invalid", "size width and height must both be divisible by 16"
	}
	longEdge := max(width, height)
	shortEdge := min(width, height)
	if longEdge > shortEdge*3 {
		return "image_aspect_ratio_invalid", "image size aspect ratio must be between 1:3 and 3:1"
	}
	pixels := width * height
	if pixels < openAIImageMinPixels {
		return "image_size_invalid", "image size must be at least 655360 total pixels"
	}
	if pixels > openAIImageMaxPixels {
		return "image_size_too_large", "image size cannot exceed 8294400 total pixels"
	}
	return "", ""
}

func resolveOpenAIImageMaxSideForPixelLimit(maxSide int, ratioW int, ratioH int) int {
	if maxSide <= 0 || ratioW <= 0 || ratioH <= 0 {
		return maxSide
	}
	longRatio := max(ratioW, ratioH)
	shortRatio := min(ratioW, ratioH)
	if longRatio <= 0 || shortRatio <= 0 {
		return maxSide
	}
	pixelLimited := math.Sqrt(float64(openAIImageMaxPixels) * float64(longRatio) / float64(shortRatio))
	if pixelLimited <= 0 || pixelLimited >= float64(maxSide) {
		return floorOpenAIImageDimensionToMultipleOf16(maxSide)
	}
	return floorOpenAIImageDimensionToMultipleOf16(int(math.Floor(pixelLimited)))
}

func roundOpenAIImageDimensionToMultipleOf16(value float64) int {
	if value <= 0 {
		return 0
	}
	return max(16, int(math.Round(value/16))*16)
}

func floorOpenAIImageDimensionToMultipleOf16(value int) int {
	if value <= 0 {
		return 0
	}
	return max(16, (value/16)*16)
}
