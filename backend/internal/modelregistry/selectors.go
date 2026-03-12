package modelregistry

import (
	"sort"
	"strings"
)

func HasExposure(entry ModelEntry, exposures ...string) bool {
	if len(exposures) == 0 {
		return true
	}
	for _, exposure := range exposures {
		for _, current := range entry.ExposedIn {
			if current == exposure {
				return true
			}
		}
	}
	return false
}

func SupportsPlatform(entry ModelEntry, platform string) bool {
	platform = normalizePlatform(platform)
	for _, current := range entry.Platforms {
		if normalizePlatform(current) == platform {
			return true
		}
	}
	return false
}

func ModelsByPlatform(entries []ModelEntry, platform string, exposures ...string) []ModelEntry {
	items := make([]ModelEntry, 0)
	for _, entry := range entries {
		if !SupportsPlatform(entry, platform) {
			continue
		}
		if !HasExposure(entry, exposures...) {
			continue
		}
		items = append(items, cloneEntry(entry))
	}
	sortEntries(items)
	return items
}

func PresetsByPlatform(presets []PresetMapping, platform string) []PresetMapping {
	platform = normalizePlatform(platform)
	items := make([]PresetMapping, 0)
	for _, preset := range presets {
		if normalizePlatform(preset.Platform) != platform {
			continue
		}
		items = append(items, preset)
	}
	sortPresets(items)
	return items
}

func ModelIDs(entries []ModelEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func FindModel(entries []ModelEntry, modelID string) (ModelEntry, bool) {
	modelID = strings.TrimSpace(modelID)
	for _, entry := range entries {
		if entry.ID == modelID {
			return cloneEntry(entry), true
		}
		for _, alias := range entry.Aliases {
			if alias == modelID {
				return cloneEntry(entry), true
			}
		}
		for _, protocolID := range entry.ProtocolIDs {
			if protocolID == modelID {
				return cloneEntry(entry), true
			}
		}
	}
	return ModelEntry{}, false
}

func normalizePlatform(platform string) string {
	platform = strings.TrimSpace(strings.ToLower(platform))
	switch platform {
	case "claude":
		return "anthropic"
	default:
		return platform
	}
}

func sortEntries(entries []ModelEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].UIPriority == entries[j].UIPriority {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].UIPriority < entries[j].UIPriority
	})
}

func sortPresets(presets []PresetMapping) {
	sort.Slice(presets, func(i, j int) bool {
		if presets[i].Order == presets[j].Order {
			if presets[i].Platform == presets[j].Platform {
				return presets[i].Label < presets[j].Label
			}
			return presets[i].Platform < presets[j].Platform
		}
		return presets[i].Order < presets[j].Order
	})
}
