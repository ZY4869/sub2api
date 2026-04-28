package service

import "strings"

const (
	ImageCountWeightTier1K = OpenAIImageSizeTier1K
	ImageCountWeightTier2K = OpenAIImageSizeTier2K
	ImageCountWeightTier4K = OpenAIImageSizeTier4K
)

func DefaultAPIKeyImageCountWeights() map[string]int {
	return map[string]int{
		ImageCountWeightTier1K: 1,
		ImageCountWeightTier2K: 1,
		ImageCountWeightTier4K: 2,
	}
}

func NormalizeAPIKeyImageCountWeights(input map[string]int) map[string]int {
	normalized := DefaultAPIKeyImageCountWeights()
	for key, value := range input {
		tier := normalizeImageCountWeightTier(key)
		if tier == "" || value <= 0 {
			continue
		}
		normalized[tier] = value
	}
	return normalized
}

func CloneAPIKeyImageCountWeights(input map[string]int) map[string]int {
	return NormalizeAPIKeyImageCountWeights(input)
}

func (k *APIKey) ImageCountWeightForTier(tier string) int {
	weights := DefaultAPIKeyImageCountWeights()
	if k != nil {
		weights = NormalizeAPIKeyImageCountWeights(k.ImageCountWeights)
	}
	normalizedTier := normalizeImageCountWeightTier(tier)
	if normalizedTier == "" {
		normalizedTier = ImageCountWeightTier2K
	}
	weight := weights[normalizedTier]
	if weight <= 0 {
		return 1
	}
	return weight
}

func (k *APIKey) ImageCountUnitsForTier(count int, tier string) int {
	if count <= 0 {
		return 0
	}
	return count * k.ImageCountWeightForTier(tier)
}

func normalizeImageCountWeightTier(tier string) string {
	switch strings.ToUpper(strings.TrimSpace(tier)) {
	case ImageCountWeightTier1K:
		return ImageCountWeightTier1K
	case ImageCountWeightTier2K, "":
		return ImageCountWeightTier2K
	case ImageCountWeightTier4K:
		return ImageCountWeightTier4K
	default:
		return ""
	}
}
