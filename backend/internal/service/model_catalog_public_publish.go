package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const defaultPublicModelCatalogPageSize = 10

func normalizePublicModelCatalogPageSize(value int) int {
	if value <= 0 {
		return defaultPublicModelCatalogPageSize
	}
	if value > 100 {
		return 100
	}
	return value
}

func normalizePublicModelCatalogDraft(input *PublicModelCatalogDraft) PublicModelCatalogDraft {
	normalized := PublicModelCatalogDraft{
		PageSize: normalizePublicModelCatalogPageSize(defaultPublicModelCatalogPageSize),
	}
	if input == nil {
		return normalized
	}
	normalized.PageSize = normalizePublicModelCatalogPageSize(input.PageSize)
	normalized.UpdatedAt = strings.TrimSpace(input.UpdatedAt)
	seen := map[string]struct{}{}
	for _, model := range input.SelectedModels {
		normalizedModel := NormalizeModelCatalogModelID(model)
		if normalizedModel == "" {
			continue
		}
		if _, ok := seen[normalizedModel]; ok {
			continue
		}
		seen[normalizedModel] = struct{}{}
		normalized.SelectedModels = append(normalized.SelectedModels, normalizedModel)
	}
	return normalized
}

func clonePublicModelCatalogDetail(detail PublicModelCatalogDetail) PublicModelCatalogDetail {
	cloned := detail
	cloned.Item = clonePublicModelCatalogItem(detail.Item)
	return cloned
}

func clonePublicModelCatalogPublishedSnapshot(snapshot *PublicModelCatalogPublishedSnapshot) *PublicModelCatalogPublishedSnapshot {
	if snapshot == nil {
		return nil
	}
	cloned := &PublicModelCatalogPublishedSnapshot{
		Snapshot: *clonePublicModelCatalogSnapshot(&snapshot.Snapshot),
	}
	if len(snapshot.Details) > 0 {
		cloned.Details = make(map[string]PublicModelCatalogDetail, len(snapshot.Details))
		keys := make([]string, 0, len(snapshot.Details))
		for key := range snapshot.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			cloned.Details[key] = clonePublicModelCatalogDetail(snapshot.Details[key])
		}
	}
	return cloned
}

func publicModelCatalogPublishedSummary(snapshot *PublicModelCatalogPublishedSnapshot) *PublicModelCatalogPublishedSummary {
	if snapshot == nil {
		return nil
	}
	return &PublicModelCatalogPublishedSummary{
		ETag:       snapshot.Snapshot.ETag,
		UpdatedAt:  snapshot.Snapshot.UpdatedAt,
		PageSize:   normalizePublicModelCatalogPageSize(snapshot.Snapshot.PageSize),
		ModelCount: len(snapshot.Snapshot.Items),
	}
}

func loadPublicModelCatalogDraftBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
) *PublicModelCatalogDraft {
	if settingRepo == nil {
		return nil
	}
	raw, err := settingRepo.GetValue(ctx, settingKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var draft PublicModelCatalogDraft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		logger.FromContext(ctx).Warn(
			"public model catalog: invalid draft json",
			zap.String("setting_key", settingKey),
			zap.Error(err),
		)
		return nil
	}
	normalized := normalizePublicModelCatalogDraft(&draft)
	return &normalized
}

func persistPublicModelCatalogDraftBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
	draft *PublicModelCatalogDraft,
) error {
	if settingRepo == nil {
		return nil
	}
	normalized := normalizePublicModelCatalogDraft(draft)
	if normalized.UpdatedAt == "" {
		normalized.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return settingRepo.Set(ctx, settingKey, string(payload))
}

func loadPublicModelCatalogPublishedSnapshotBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
) *PublicModelCatalogPublishedSnapshot {
	if settingRepo == nil {
		return nil
	}
	raw, err := settingRepo.GetValue(ctx, settingKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var snapshot PublicModelCatalogPublishedSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		logger.FromContext(ctx).Warn(
			"public model catalog: invalid published snapshot json",
			zap.String("setting_key", settingKey),
			zap.Error(err),
		)
		return nil
	}
	normalized := clonePublicModelCatalogPublishedSnapshot(&snapshot)
	if normalized == nil {
		return nil
	}
	normalized.Snapshot.PageSize = normalizePublicModelCatalogPageSize(normalized.Snapshot.PageSize)
	return normalized
}

func persistPublicModelCatalogPublishedSnapshotBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
	snapshot *PublicModelCatalogPublishedSnapshot,
) error {
	if settingRepo == nil {
		return nil
	}
	if snapshot == nil || len(snapshot.Snapshot.Items) == 0 {
		return settingRepo.Delete(ctx, settingKey)
	}
	normalized := clonePublicModelCatalogPublishedSnapshot(snapshot)
	normalized.Snapshot.PageSize = normalizePublicModelCatalogPageSize(normalized.Snapshot.PageSize)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return settingRepo.Set(ctx, settingKey, string(payload))
}

func (s *ModelCatalogService) loadPublicModelCatalogDraft(ctx context.Context) *PublicModelCatalogDraft {
	if s == nil {
		return nil
	}
	return loadPublicModelCatalogDraftBySetting(ctx, s.settingRepo, SettingKeyPublicModelCatalogDraft)
}

func (s *ModelCatalogService) persistPublicModelCatalogDraft(ctx context.Context, draft *PublicModelCatalogDraft) error {
	if s == nil {
		return nil
	}
	return persistPublicModelCatalogDraftBySetting(ctx, s.settingRepo, SettingKeyPublicModelCatalogDraft, draft)
}

func (s *ModelCatalogService) loadPublishedPublicModelCatalogSnapshot(ctx context.Context) *PublicModelCatalogPublishedSnapshot {
	if s == nil {
		return nil
	}
	return loadPublicModelCatalogPublishedSnapshotBySetting(ctx, s.settingRepo, SettingKeyPublicModelCatalogPublishedSnapshot)
}

func (s *ModelCatalogService) persistPublishedPublicModelCatalogSnapshot(ctx context.Context, snapshot *PublicModelCatalogPublishedSnapshot) error {
	if s == nil {
		return nil
	}
	return persistPublicModelCatalogPublishedSnapshotBySetting(ctx, s.settingRepo, SettingKeyPublicModelCatalogPublishedSnapshot, snapshot)
}

func selectPublicModelCatalogPublishItems(draft PublicModelCatalogDraft, items []PublicModelCatalogItem) []PublicModelCatalogItem {
	if len(items) == 0 {
		return []PublicModelCatalogItem{}
	}
	if len(draft.SelectedModels) == 0 {
		selected := make([]PublicModelCatalogItem, 0, len(items))
		for _, item := range items {
			selected = append(selected, clonePublicModelCatalogItem(item))
		}
		return selected
	}
	itemsByModel := make(map[string]PublicModelCatalogItem, len(items))
	for _, item := range items {
		modelID := NormalizeModelCatalogModelID(item.Model)
		if modelID == "" {
			continue
		}
		itemsByModel[modelID] = item
	}
	selected := make([]PublicModelCatalogItem, 0, len(draft.SelectedModels))
	for _, modelID := range draft.SelectedModels {
		item, ok := itemsByModel[NormalizeModelCatalogModelID(modelID)]
		if !ok {
			continue
		}
		selected = append(selected, clonePublicModelCatalogItem(item))
	}
	return selected
}

func (s *ModelCatalogService) GetPublicModelCatalogDraftPayload(ctx context.Context) (*PublicModelCatalogDraftPayload, error) {
	draft := normalizePublicModelCatalogDraft(s.loadPublicModelCatalogDraft(ctx))
	liveSnapshot, err := s.buildLivePublicModelCatalogSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return &PublicModelCatalogDraftPayload{
		Draft:          draft,
		AvailableItems: append([]PublicModelCatalogItem(nil), liveSnapshot.Items...),
		Published:      publicModelCatalogPublishedSummary(s.loadPublishedPublicModelCatalogSnapshot(ctx)),
	}, nil
}

func (s *ModelCatalogService) SavePublicModelCatalogDraft(ctx context.Context, draft PublicModelCatalogDraft) (*PublicModelCatalogDraft, error) {
	normalized := normalizePublicModelCatalogDraft(&draft)
	normalized.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.persistPublicModelCatalogDraft(ctx, &normalized); err != nil {
		return nil, err
	}
	logger.FromContext(ctx).Info(
		"public model catalog draft saved",
		zap.String("component", "service.model_catalog"),
		zap.Int("selected_model_count", len(normalized.SelectedModels)),
		zap.Int("page_size", normalized.PageSize),
	)
	return &normalized, nil
}

func (s *ModelCatalogService) PublishPublicModelCatalog(
	ctx context.Context,
	actor ModelCatalogActor,
	draftInput *PublicModelCatalogDraft,
) (*PublicModelCatalogPublishedSummary, error) {
	if s == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "model catalog service unavailable")
	}
	draft := normalizePublicModelCatalogDraft(draftInput)
	if draftInput != nil {
		draft.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := s.persistPublicModelCatalogDraft(ctx, &draft); err != nil {
			return nil, err
		}
	} else {
		draft = normalizePublicModelCatalogDraft(s.loadPublicModelCatalogDraft(ctx))
	}
	liveSnapshot, err := s.buildLivePublicModelCatalogSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	selectedItems := selectPublicModelCatalogPublishItems(draft, liveSnapshot.Items)
	if len(selectedItems) == 0 && len(liveSnapshot.Items) > 0 {
		return nil, infraerrors.BadRequest("PUBLIC_MODEL_CATALOG_EMPTY", "no models selected for publish")
	}
	details := make(map[string]PublicModelCatalogDetail, len(selectedItems))
	for _, item := range selectedItems {
		exampleSource, exampleProtocol, examplePageID, exampleMarkdown, exampleOverrideID := s.buildPublicModelCatalogDetailExample(ctx, item)
		details[item.Model] = PublicModelCatalogDetail{
			Item:              clonePublicModelCatalogItem(item),
			ExampleSource:     exampleSource,
			ExampleProtocol:   exampleProtocol,
			ExamplePageID:     examplePageID,
			ExampleMarkdown:   exampleMarkdown,
			ExampleOverrideID: exampleOverrideID,
		}
	}
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:      liveSnapshot.ETag,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			PageSize:  normalizePublicModelCatalogPageSize(draft.PageSize),
			Items:     selectedItems,
		},
		Details: details,
	}
	etag, err := computePublicModelCatalogETagWithPageSize(published.Snapshot.PageSize, published.Snapshot.Items)
	if err != nil {
		return nil, err
	}
	published.Snapshot.ETag = etag
	if err := s.persistPublishedPublicModelCatalogSnapshot(ctx, published); err != nil {
		return nil, err
	}
	summary := publicModelCatalogPublishedSummary(published)
	logger.FromContext(ctx).Info(
		"public model catalog published",
		zap.String("component", "service.model_catalog"),
		zap.String("etag", summary.ETag),
		zap.Int("model_count", summary.ModelCount),
		zap.Int("page_size", summary.PageSize),
		zap.Int64("actor_user_id", actor.UserID),
		zap.String("actor_email", strings.TrimSpace(actor.Email)),
	)
	return summary, nil
}

func (s *ModelCatalogService) GetPublishedPublicModelCatalogSummary(ctx context.Context) (*PublicModelCatalogPublishedSummary, error) {
	return publicModelCatalogPublishedSummary(s.loadPublishedPublicModelCatalogSnapshot(ctx)), nil
}
