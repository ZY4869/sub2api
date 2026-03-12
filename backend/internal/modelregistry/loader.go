package modelregistry

import (
	_ "embed"
	"encoding/json"
)

//go:embed registry_seed.json
var registrySeedJSON []byte

//go:embed preset_overlay.json
var presetOverlayJSON []byte

var (
	seedModels   []ModelEntry
	seedPresets  []PresetMapping
	seedModelMap map[string]ModelEntry
)

func init() {
	if err := json.Unmarshal(registrySeedJSON, &seedModels); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(presetOverlayJSON, &seedPresets); err != nil {
		panic(err)
	}
	seedModelMap = make(map[string]ModelEntry, len(seedModels))
	for _, entry := range seedModels {
		seedModelMap[entry.ID] = cloneEntry(entry)
	}
	for index := range seedPresets {
		if seedPresets[index].Order == 0 {
			seedPresets[index].Order = index + 1
		}
	}
	sortEntries(seedModels)
	sortPresets(seedPresets)
}

func SeedModels() []ModelEntry {
	items := make([]ModelEntry, len(seedModels))
	for index, entry := range seedModels {
		items[index] = cloneEntry(entry)
	}
	return items
}

func SeedPresets() []PresetMapping {
	items := make([]PresetMapping, len(seedPresets))
	copy(items, seedPresets)
	return items
}

func SeedModelByID(id string) (ModelEntry, bool) {
	entry, ok := seedModelMap[id]
	if !ok {
		return ModelEntry{}, false
	}
	return cloneEntry(entry), true
}

func cloneEntry(entry ModelEntry) ModelEntry {
	entry.Platforms = append([]string(nil), entry.Platforms...)
	entry.ProtocolIDs = append([]string(nil), entry.ProtocolIDs...)
	entry.Aliases = append([]string(nil), entry.Aliases...)
	entry.PricingLookupIDs = append([]string(nil), entry.PricingLookupIDs...)
	entry.Modalities = append([]string(nil), entry.Modalities...)
	entry.Capabilities = append([]string(nil), entry.Capabilities...)
	entry.ExposedIn = append([]string(nil), entry.ExposedIn...)
	return entry
}
