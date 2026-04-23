package service

import (
	"strconv"
	"strings"
)

const (
	OpenAIImageSizeTier1K = "1K"
	OpenAIImageSizeTier2K = "2K"
	OpenAIImageSizeTier4K = "4K"
)

func ResolveOpenAIImageSizeTier(size string) string {
	trimmed := strings.TrimSpace(strings.ToLower(size))
	switch trimmed {
	case "", "auto":
		return OpenAIImageSizeTier2K
	case "1k":
		return OpenAIImageSizeTier1K
	case "2k":
		return OpenAIImageSizeTier2K
	case "4k":
		return OpenAIImageSizeTier4K
	}

	parts := strings.Split(trimmed, "x")
	if len(parts) != 2 {
		return OpenAIImageSizeTier2K
	}
	width, errWidth := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errHeight := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errWidth != nil || errHeight != nil {
		return OpenAIImageSizeTier2K
	}
	maxSide := width
	if height > maxSide {
		maxSide = height
	}
	switch {
	case maxSide <= 1536:
		return OpenAIImageSizeTier1K
	case maxSide <= 2048:
		return OpenAIImageSizeTier2K
	case maxSide <= 3840:
		return OpenAIImageSizeTier4K
	default:
		return OpenAIImageSizeTier4K
	}
}
