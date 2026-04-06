package modelregistry

type Index struct {
	byID         map[string]ModelEntry
	aliasToID    map[string]string
	protocolToID map[string]string
	pricingToID  map[string]string
}

func BuildIndex(entries []ModelEntry) *Index {
	index := &Index{
		byID:         make(map[string]ModelEntry, len(entries)),
		aliasToID:    make(map[string]string, len(entries)*2),
		protocolToID: make(map[string]string, len(entries)*2),
		pricingToID:  make(map[string]string, len(entries)*2),
	}

	for _, original := range entries {
		entry := cloneEntry(original)
		index.byID[NormalizeID(entry.ID)] = entry
		index.registerVariants(index.aliasToID, entry.ID, entry.ID)
		for _, alias := range entry.Aliases {
			index.registerVariants(index.aliasToID, alias, entry.ID)
		}
		for _, protocolID := range entry.ProtocolIDs {
			index.registerVariants(index.protocolToID, protocolID, entry.ID)
			index.registerVariants(index.aliasToID, protocolID, entry.ID)
		}
		for _, pricingID := range entry.PricingLookupIDs {
			index.registerVariants(index.pricingToID, pricingID, entry.ID)
		}
	}

	return index
}

func (i *Index) Resolve(input string) (*Resolution, bool) {
	if i == nil {
		return nil, false
	}
	normalizedInput := NormalizeID(input)
	if normalizedInput == "" {
		return nil, false
	}

	resolveBy := func(source string, canonicalID string, matchedValue string) (*Resolution, bool) {
		entry, ok := i.byID[NormalizeID(canonicalID)]
		if !ok {
			return nil, false
		}
		resolution := &Resolution{
			Input:           input,
			NormalizedInput: normalizedInput,
			CanonicalID:     entry.ID,
			EffectiveID:     entry.ID,
			PricingID:       firstNonEmpty(entry.PricingLookupIDs...),
			MatchedBy:       source,
			MatchedValue:    matchedValue,
			Entry:           cloneEntry(entry),
		}
		if resolution.PricingID == "" {
			resolution.PricingID = entry.ID
		}
		if entry.Status == "deprecated" && entry.ReplacedBy != "" {
			resolution.Deprecated = true
			resolution.EffectiveID = entry.ReplacedBy
			if replacement, ok := i.byID[NormalizeID(entry.ReplacedBy)]; ok {
				cloned := cloneEntry(replacement)
				resolution.ReplacementEntry = &cloned
			}
		}
		return resolution, true
	}

	if _, ok := i.byID[normalizedInput]; ok {
		return resolveBy("id", normalizedInput, normalizedInput)
	}
	if canonicalID, ok := i.aliasToID[normalizedInput]; ok {
		return resolveBy("alias", canonicalID, normalizedInput)
	}
	if canonicalID, ok := i.protocolToID[normalizedInput]; ok {
		return resolveBy("protocol", canonicalID, normalizedInput)
	}
	if canonicalID, ok := i.pricingToID[normalizedInput]; ok {
		return resolveBy("pricing", canonicalID, normalizedInput)
	}
	return nil, false
}

func (i *Index) ResolveCanonicalID(input string) (string, bool) {
	resolution, ok := i.Resolve(input)
	if !ok {
		return "", false
	}
	if resolution.EffectiveID != "" {
		return resolution.EffectiveID, true
	}
	return resolution.CanonicalID, true
}

func (i *Index) ResolvePricingID(input string) (string, bool) {
	resolution, ok := i.Resolve(input)
	if !ok {
		return "", false
	}
	if resolution.PricingID != "" {
		return resolution.PricingID, true
	}
	return resolution.CanonicalID, true
}

func (i *Index) ResolveProtocolID(input string, route string) (string, bool) {
	resolution, ok := i.Resolve(input)
	if !ok {
		return "", false
	}
	entry := resolution.Entry
	route = NormalizePlatform(route)
	familyRoute := ""
	switch NormalizePlatformFamily(route) {
	case "anthropic":
		familyRoute = "anthropic_oauth"
	case "openai":
		familyRoute = "openai"
	}
	if value := firstNonEmpty(
		entry.PreferredProtocolIDs[route],
		entry.PreferredProtocolIDs[familyRoute],
		entry.PreferredProtocolIDs["default"],
	); value != "" {
		return NormalizeID(value), true
	}

	switch route {
	case "anthropic_oauth":
		if value := firstNonEmpty(entry.ProtocolIDs...); value != "" {
			return NormalizeID(value), true
		}
	case "kiro":
		values := append([]string{entry.PreferredProtocolIDs["anthropic_oauth"]}, entry.ProtocolIDs...)
		if value := firstNonEmpty(values...); value != "" {
			return NormalizeID(value), true
		}
	case "copilot":
		if value := firstNonEmpty(entry.PreferredProtocolIDs["openai"], entry.ID); value != "" {
			return NormalizeID(value), true
		}
	case "anthropic_apikey", "openai", "gemini", "antigravity":
		if entry.ID != "" {
			return entry.ID, true
		}
	}

	if value := firstNonEmpty(entry.ProtocolIDs...); value != "" {
		return NormalizeID(value), true
	}
	if resolution.EffectiveID != "" {
		return resolution.EffectiveID, true
	}
	return resolution.CanonicalID, true
}

func (i *Index) registerVariants(target map[string]string, input string, canonicalID string) {
	canonicalID = NormalizeID(canonicalID)
	if canonicalID == "" {
		return
	}
	for _, variant := range AlternateVersionVariants(input) {
		target[variant] = canonicalID
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = NormalizeID(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func ExplainResolution(entries []ModelEntry, input string) (*Resolution, bool) {
	return BuildIndex(entries).Resolve(input)
}

func ResolveToCanonicalID(input string) (string, bool) {
	if seedIndex == nil {
		return "", false
	}
	return seedIndex.ResolveCanonicalID(input)
}

func ResolveToPricingID(input string) (string, bool) {
	if seedIndex == nil {
		return "", false
	}
	return seedIndex.ResolvePricingID(input)
}

func ResolveToProtocolID(input string, route string) (string, bool) {
	if seedIndex == nil {
		return "", false
	}
	return seedIndex.ResolveProtocolID(input, route)
}

func ExplainSeedResolution(input string) (*Resolution, bool) {
	if seedIndex == nil {
		return nil, false
	}
	return seedIndex.Resolve(input)
}
