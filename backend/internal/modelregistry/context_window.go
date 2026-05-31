package modelregistry

import (
	"encoding/json"
	"sync"

	pricingbundle "github.com/Wei-Shaw/sub2api/resources/model-pricing"
)

type pricingContextWindowEntry struct {
	MaxInputTokens int64 `json:"max_input_tokens"`
}

const ContextWindowSourcePricingCatalog = "pricing_catalog"

type ContextWindowResolution struct {
	Tokens int64
	Source string
}

var (
	contextWindowLookupOnce sync.Once
	contextWindowLookup     map[string]ContextWindowResolution
)

func ResolveContextWindow(ids ...string) (ContextWindowResolution, bool) {
	lookup := loadContextWindowLookup()
	for _, id := range ids {
		for _, variant := range AlternateVersionVariants(id) {
			if resolution, ok := lookup[variant]; ok && resolution.Tokens > 0 {
				if resolution.Source == "" {
					resolution.Source = ContextWindowSourcePricingCatalog
				}
				return resolution, true
			}
		}
	}
	return ContextWindowResolution{}, false
}

func ResolveContextWindowTokens(ids ...string) (int64, bool) {
	resolution, ok := ResolveContextWindow(ids...)
	if !ok {
		return 0, false
	}
	return resolution.Tokens, true
}

func hydrateContextWindowTokens(entry ModelEntry) ModelEntry {
	if tokens, ok := ResolveContextWindowTokens(append(append([]string{}, entry.PricingLookupIDs...), entry.ID)...); ok {
		entry.ContextWindowTokens = tokens
	} else {
		entry.ContextWindowTokens = 0
	}
	return entry
}

func loadContextWindowLookup() map[string]ContextWindowResolution {
	contextWindowLookupOnce.Do(func() {
		contextWindowLookup = make(map[string]ContextWindowResolution)
		if len(pricingbundle.FallbackPricingJSON) == 0 {
			return
		}
		raw := map[string]pricingContextWindowEntry{}
		if err := json.Unmarshal(pricingbundle.FallbackPricingJSON, &raw); err != nil {
			return
		}
		for key, value := range raw {
			if value.MaxInputTokens <= 0 {
				continue
			}
			normalized := NormalizeID(key)
			if normalized == "" {
				continue
			}
			contextWindowLookup[normalized] = ContextWindowResolution{
				Tokens: value.MaxInputTokens,
				Source: ContextWindowSourcePricingCatalog,
			}
		}
	})
	return contextWindowLookup
}
