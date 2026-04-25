package service

import (
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/model"
)

type pricedModelEntry struct {
	displayName string
	pricing     *SupportedModelPricing
}

type supportedModelKey struct {
	platform string
	name     string
}

func buildChannelSupportedModels(ch *model.Channel) []SupportedModel {
	if ch == nil {
		return nil
	}

	priced := indexChannelPricedModels(ch)
	seen := make(map[supportedModelKey]struct{})
	out := make([]SupportedModel, 0)

	add := func(platform, name string, pricing *SupportedModelPricing) {
		platform = strings.ToLower(strings.TrimSpace(platform))
		name = strings.TrimSpace(name)
		if platform == "" || name == "" {
			return
		}
		key := supportedModelKey{platform: platform, name: strings.ToLower(name)}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, SupportedModel{
			Name:     name,
			Platform: platform,
			Pricing:  pricing,
		})
	}

	// Pass A: mapping-driven entries (exact src only, no wildcards).
	for platformKey, mapping := range ch.ModelMapping {
		platform := strings.ToLower(strings.TrimSpace(platformKey))
		if platform == "" || platform == "*" {
			continue
		}
		for src, target := range mapping {
			src = strings.TrimSpace(src)
			if src == "" || strings.Contains(src, "*") {
				continue
			}
			pricingModel := strings.TrimSpace(target)
			if pricingModel == "" || strings.Contains(pricingModel, "*") {
				pricingModel = src
			}

			displayName := src
			if entry, ok := priced[platform][strings.ToLower(src)]; ok && entry.displayName != "" {
				displayName = entry.displayName
			}

			var pricing *SupportedModelPricing
			if entry, ok := priced[platform][strings.ToLower(pricingModel)]; ok {
				pricing = entry.pricing
			}
			add(platform, displayName, pricing)
		}
	}

	// Pass B: pricing-only entries (exact models only, no wildcards).
	platforms := make([]string, 0, len(priced))
	for platform := range priced {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)
	for _, platform := range platforms {
		names := make([]string, 0, len(priced[platform]))
		for lower := range priced[platform] {
			names = append(names, lower)
		}
		sort.Strings(names)
		for _, lower := range names {
			entry := priced[platform][lower]
			add(platform, entry.displayName, entry.pricing)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func indexChannelPricedModels(ch *model.Channel) map[string]map[string]pricedModelEntry {
	result := make(map[string]map[string]pricedModelEntry)
	if ch == nil {
		return result
	}

	for i := range ch.ModelPricing {
		p := ch.ModelPricing[i]
		platform := strings.ToLower(strings.TrimSpace(p.Platform))
		if platform == "" {
			continue
		}
		if _, ok := result[platform]; !ok {
			result[platform] = make(map[string]pricedModelEntry)
		}

		pricingView := supportedPricingFromChannelPricing(p)
		for _, modelName := range p.Models {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" || strings.Contains(modelName, "*") {
				continue
			}
			lower := strings.ToLower(modelName)
			if _, exists := result[platform][lower]; exists {
				continue
			}
			result[platform][lower] = pricedModelEntry{
				displayName: modelName,
				pricing:     pricingView,
			}
		}
	}

	return result
}

func supportedPricingFromChannelPricing(p model.ChannelModelPricing) *SupportedModelPricing {
	intervals := make([]SupportedModelPricingInterval, 0, len(p.Intervals))
	for i := range p.Intervals {
		iv := p.Intervals[i]
		intervals = append(intervals, SupportedModelPricingInterval{
			MinTokens:       int(iv.MinTokens),
			MaxTokens:       int64PtrToIntPtr(iv.MaxTokens),
			TierLabel:       strings.TrimSpace(iv.TierLabel),
			InputPrice:      cloneFloatPtr(iv.InputPrice),
			OutputPrice:     cloneFloatPtr(iv.OutputPrice),
			CacheWritePrice: cloneFloatPtr(iv.CacheWritePrice),
			CacheReadPrice:  cloneFloatPtr(iv.CacheReadPrice),
			PerRequestPrice: cloneFloatPtr(iv.PerRequestPrice),
		})
	}

	return &SupportedModelPricing{
		BillingMode:      p.BillingMode,
		InputPrice:       cloneFloatPtr(p.InputPrice),
		OutputPrice:      cloneFloatPtr(p.OutputPrice),
		CacheWritePrice:  cloneFloatPtr(p.CacheWritePrice),
		CacheReadPrice:   cloneFloatPtr(p.CacheReadPrice),
		ImageOutputPrice: cloneFloatPtr(p.ImageOutputPrice),
		PerRequestPrice:  cloneFloatPtr(p.PerRequestPrice),
		Intervals:        intervals,
	}
}

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	clone := *v
	return &clone
}

func int64PtrToIntPtr(v *int64) *int {
	if v == nil {
		return nil
	}
	c := int(*v)
	return &c
}
