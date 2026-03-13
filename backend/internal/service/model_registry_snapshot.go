package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *ModelRegistryService) PublicSnapshot(ctx context.Context) (*modelregistry.PublicSnapshot, error) {
	models, presets, err := s.visibleSnapshotData(ctx)
	if err != nil {
		return nil, err
	}
	etag, err := computeRegistryETag(models, presets)
	if err != nil {
		return nil, err
	}
	return &modelregistry.PublicSnapshot{
		ETag:      etag,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Models:    models,
		Presets:   presets,
	}, nil
}

func (s *ModelRegistryService) GetModelsByPlatform(ctx context.Context, platform string, exposures ...string) ([]modelregistry.ModelEntry, error) {
	snapshot, err := s.PublicSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return modelregistry.ModelsByPlatform(snapshot.Models, platform, exposures...), nil
}

func (s *ModelRegistryService) GetModel(ctx context.Context, modelID string) (*modelregistry.ModelEntry, error) {
	snapshot, err := s.PublicSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	entry, ok := modelregistry.FindModel(snapshot.Models, normalizeRegistryID(modelID))
	if !ok {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	return &entry, nil
}

func (s *ModelRegistryService) visibleSnapshotData(ctx context.Context) ([]modelregistry.ModelEntry, []modelregistry.PresetMapping, error) {
	availableSet, err := s.loadAvailableModelSet(ctx)
	if err != nil {
		return nil, nil, err
	}
	entries, _, hidden, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, nil, err
	}
	models := make([]modelregistry.ModelEntry, 0, len(entries))
	for id, entry := range entries {
		if _, isHidden := hidden[id]; isHidden {
			continue
		}
		if _, isTombstoned := tombstones[id]; isTombstoned {
			continue
		}
		if _, available := availableSet[id]; !available {
			continue
		}
		models = append(models, entry)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].UIPriority == models[j].UIPriority {
			return models[i].ID < models[j].ID
		}
		return models[i].UIPriority < models[j].UIPriority
	})
	presets := make([]modelregistry.PresetMapping, 0)
	for _, preset := range modelregistry.SeedPresets() {
		if _, hiddenFrom := hidden[normalizeRegistryID(preset.From)]; hiddenFrom {
			continue
		}
		if _, hiddenTo := hidden[normalizeRegistryID(preset.To)]; hiddenTo {
			continue
		}
		if _, tombstoneFrom := tombstones[normalizeRegistryID(preset.From)]; tombstoneFrom {
			continue
		}
		if _, tombstoneTo := tombstones[normalizeRegistryID(preset.To)]; tombstoneTo {
			continue
		}
		if _, availableFrom := availableSet[normalizeRegistryID(preset.From)]; !availableFrom {
			continue
		}
		if _, availableTo := availableSet[normalizeRegistryID(preset.To)]; !availableTo {
			continue
		}
		presets = append(presets, preset)
	}
	return models, presets, nil
}

func computeRegistryETag(models []modelregistry.ModelEntry, presets []modelregistry.PresetMapping) (string, error) {
	payload, err := json.Marshal(struct {
		Models  []modelregistry.ModelEntry    `json:"models"`
		Presets []modelregistry.PresetMapping `json:"presets"`
	}{Models: models, Presets: presets})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "W/\"" + hex.EncodeToString(sum[:]) + "\"", nil
}
