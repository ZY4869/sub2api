package modelregistry

import (
	"encoding/json"
	"sync"

	pricingbundle "github.com/Wei-Shaw/sub2api/resources/model-pricing"
)

type pricingContextWindowEntry struct {
	MaxInputTokens int64 `json:"max_input_tokens"`
}

var (
	contextWindowLookupOnce sync.Once
	contextWindowLookup     map[string]int64
)

func ResolveContextWindowTokens(ids ...string) (int64, bool) {
	lookup := loadContextWindowLookup()
	for _, id := range ids {
		for _, variant := range AlternateVersionVariants(id) {
			if tokens, ok := lookup[variant]; ok && tokens > 0 {
				return tokens, true
			}
		}
	}
	return 0, false
}

func hydrateContextWindowTokens(entry ModelEntry) ModelEntry {
	if tokens, ok := ResolveContextWindowTokens(append(append([]string{}, entry.PricingLookupIDs...), entry.ID)...); ok {
		entry.ContextWindowTokens = tokens
	} else {
		entry.ContextWindowTokens = 0
	}
	return entry
}

func loadContextWindowLookup() map[string]int64 {
	contextWindowLookupOnce.Do(func() {
		contextWindowLookup = make(map[string]int64)
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
			contextWindowLookup[normalized] = value.MaxInputTokens
		}
	})
	return contextWindowLookup
}
