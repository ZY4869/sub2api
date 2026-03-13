package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

func (s *ModelRegistryService) ResolveModel(ctx context.Context, input string) (string, bool, error) {
	resolution, err := s.ExplainResolution(ctx, input)
	if err != nil || resolution == nil {
		return "", false, err
	}
	if resolution.EffectiveID != "" {
		return resolution.EffectiveID, true, nil
	}
	return resolution.CanonicalID, true, nil
}

func (s *ModelRegistryService) ResolveProtocolModel(ctx context.Context, input string, route string) (string, bool, error) {
	entries, err := s.resolutionEntries(ctx)
	if err != nil {
		return "", false, err
	}
	index := modelregistry.BuildIndex(entries)
	value, ok := index.ResolveProtocolID(input, route)
	return value, ok, nil
}

func (s *ModelRegistryService) ResolvePricingModel(ctx context.Context, input string) (string, bool, error) {
	entries, err := s.pricingEntries(ctx)
	if err != nil {
		return "", false, err
	}
	index := modelregistry.BuildIndex(entries)
	value, ok := index.ResolvePricingID(input)
	return value, ok, nil
}

func (s *ModelRegistryService) ExplainResolution(ctx context.Context, input string) (*modelregistry.Resolution, error) {
	entries, err := s.resolutionEntries(ctx)
	if err != nil {
		return nil, err
	}
	resolution, ok := modelregistry.ExplainResolution(entries, input)
	if !ok {
		return nil, nil
	}
	if resolution.ReplacementEntry == nil && resolution.Entry.ReplacedBy != "" {
		if replacement, found := modelregistry.FindModel(entries, resolution.Entry.ReplacedBy); found {
			cloned := replacement
			resolution.ReplacementEntry = &cloned
		}
	}
	return resolution, nil
}

func (s *ModelRegistryService) resolutionEntries(ctx context.Context) ([]modelregistry.ModelEntry, error) {
	availableSet, err := s.loadAvailableModelSet(ctx)
	if err != nil {
		return nil, err
	}
	entries, _, _, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]modelregistry.ModelEntry, 0, len(entries))
	for id, entry := range entries {
		if _, tombstoned := tombstones[id]; tombstoned {
			continue
		}
		if _, available := availableSet[id]; !available {
			continue
		}
		items = append(items, entry)
	}
	return items, nil
}
