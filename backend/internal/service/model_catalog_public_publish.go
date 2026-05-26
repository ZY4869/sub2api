package service

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const defaultPublicModelCatalogPageSize = 10
const publicModelCatalogDraftLiveTTL = 10 * time.Minute

const (
	publicModelCatalogDraftAvailableSourcePersisted = "persisted_snapshot"
	publicModelCatalogDraftAvailableSourceRefreshed = "refreshed_snapshot"
	publicModelCatalogDraftAvailableSourceBootstrap = "bootstrap_snapshot"
	publicModelCatalogDraftAvailableSourceCache     = "cache_snapshot"
)

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
	entrySeen := map[string]struct{}{}
	for _, entry := range input.SelectedEntries {
		normalizedEntry := normalizePublicModelCatalogEntryDraft(entry)
		if normalizedEntry.EntryID == "" {
			continue
		}
		if _, ok := entrySeen[normalizedEntry.EntryID]; ok {
			continue
		}
		entrySeen[normalizedEntry.EntryID] = struct{}{}
		normalized.SelectedEntries = append(normalized.SelectedEntries, normalizedEntry)
	}
	if len(normalized.SelectedModels) == 0 && len(normalized.SelectedEntries) > 0 {
		for _, entry := range normalized.SelectedEntries {
			if entry.PublicModelID != "" {
				normalized.SelectedModels = append(normalized.SelectedModels, entry.PublicModelID)
			}
		}
	}
	return normalized
}

func normalizePublicModelCatalogEntryDraft(entry PublicModelCatalogEntryDraft) PublicModelCatalogEntryDraft {
	normalized := PublicModelCatalogEntryDraft{
		EntryID:         strings.TrimSpace(entry.EntryID),
		PublicModelID:   NormalizeModelCatalogModelID(entry.PublicModelID),
		SourceAccountID: entry.SourceAccountID,
		SourceAlias:     strings.TrimSpace(entry.SourceAlias),
		SourceModelID:   NormalizeModelCatalogModelID(entry.SourceModelID),
		BaseModel:       NormalizeModelCatalogModelID(entry.BaseModel),
		SourceProtocol:  strings.TrimSpace(strings.ToLower(entry.SourceProtocol)),
	}
	normalized.SalePriceDisplay = normalizePublicModelCatalogPriceDisplay(entry.SalePriceDisplay)
	return normalized
}

func normalizePublicModelCatalogPriceDisplay(display PublicModelCatalogPriceDisplay) PublicModelCatalogPriceDisplay {
	return PublicModelCatalogPriceDisplay{
		Primary:   normalizePublicModelCatalogPriceEntries(display.Primary),
		Secondary: normalizePublicModelCatalogPriceEntries(display.Secondary),
	}
}

func normalizePublicModelCatalogPriceEntries(entries []PublicModelCatalogPriceEntry) []PublicModelCatalogPriceEntry {
	if len(entries) == 0 {
		return nil
	}
	normalized := make([]PublicModelCatalogPriceEntry, 0, len(entries))
	for _, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			continue
		}
		normalized = append(normalized, PublicModelCatalogPriceEntry{
			ID:    id,
			Unit:  strings.TrimSpace(entry.Unit),
			Value: entry.Value,
		})
	}
	if len(normalized) == 0 {
		return nil
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

func loadPublicModelCatalogSnapshotBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
) *PublicModelCatalogSnapshot {
	if settingRepo == nil {
		return nil
	}
	raw, err := settingRepo.GetValue(ctx, settingKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var snapshot PublicModelCatalogSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		logger.FromContext(ctx).Warn(
			"public model catalog: invalid snapshot json",
			zap.String("setting_key", settingKey),
			zap.Error(err),
		)
		return nil
	}
	normalized := clonePublicModelCatalogSnapshot(&snapshot)
	if normalized == nil {
		return nil
	}
	normalized.PageSize = normalizePublicModelCatalogPageSize(normalized.PageSize)
	return normalized
}

func persistPublicModelCatalogSnapshotBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
	snapshot *PublicModelCatalogSnapshot,
) error {
	if settingRepo == nil {
		return nil
	}
	if snapshot == nil {
		return settingRepo.Delete(ctx, settingKey)
	}
	normalized := clonePublicModelCatalogSnapshot(snapshot)
	normalized.PageSize = normalizePublicModelCatalogPageSize(normalized.PageSize)
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

func (s *ModelCatalogService) loadPublicModelCatalogDraftCandidateSnapshot(ctx context.Context) *PublicModelCatalogSnapshot {
	if s == nil {
		return nil
	}
	return loadPublicModelCatalogSnapshotBySetting(ctx, s.settingRepo, SettingKeyPublicModelCatalogDraftCandidateSnapshot)
}

func (s *ModelCatalogService) persistPublicModelCatalogDraftCandidateSnapshot(ctx context.Context, snapshot *PublicModelCatalogSnapshot) error {
	if s == nil {
		return nil
	}
	return persistPublicModelCatalogSnapshotBySetting(ctx, s.settingRepo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, snapshot)
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

func selectPublicModelCatalogPublishItems(draft PublicModelCatalogDraft, items []PublicModelCatalogItem) ([]PublicModelCatalogItem, error) {
	if len(items) == 0 {
		if len(draft.SelectedEntries) > 0 {
			return nil, infraerrors.BadRequest("PUBLIC_MODEL_ENTRY_UNAVAILABLE", "selected public model entry is no longer available")
		}
		return []PublicModelCatalogItem{}, nil
	}
	if len(draft.SelectedEntries) > 0 {
		return selectPublicModelCatalogPublishEntryItems(draft.SelectedEntries, items)
	}
	if len(draft.SelectedModels) == 0 {
		selected := make([]PublicModelCatalogItem, 0, len(items))
		for _, item := range items {
			selected = append(selected, clonePublicModelCatalogItem(item))
		}
		return selected, nil
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
	return selected, nil
}

func selectPublicModelCatalogPublishEntryItems(entries []PublicModelCatalogEntryDraft, items []PublicModelCatalogItem) ([]PublicModelCatalogItem, error) {
	itemsByEntryID := make(map[string]PublicModelCatalogItem, len(items))
	for _, item := range items {
		entryID := strings.TrimSpace(item.EntryID)
		if entryID == "" {
			continue
		}
		itemsByEntryID[entryID] = item
	}

	selected := make([]PublicModelCatalogItem, 0, len(entries))
	seenPublicIDs := map[string]struct{}{}
	for _, draftEntry := range entries {
		entry := normalizePublicModelCatalogEntryDraft(draftEntry)
		item, ok := itemsByEntryID[entry.EntryID]
		if !ok {
			return nil, infraerrors.BadRequest("PUBLIC_MODEL_ENTRY_UNAVAILABLE", "selected public model entry is no longer available")
		}
		publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(entry.PublicModelID, item.PublicModelID, item.Model))
		if publicID == "" {
			return nil, infraerrors.BadRequest("PUBLIC_MODEL_ID_REQUIRED", "public_model_id is required")
		}
		if _, exists := seenPublicIDs[publicID]; exists {
			return nil, infraerrors.BadRequest("PUBLIC_MODEL_ID_DUPLICATE", "public_model_id must be unique")
		}
		seenPublicIDs[publicID] = struct{}{}
		if err := validatePublicModelCatalogPriceDisplay(entry.SalePriceDisplay); err != nil {
			return nil, err
		}
		next := clonePublicModelCatalogItem(item)
		next.PublicModelID = publicID
		next.Model = publicID
		next.SourceAlias = firstNonEmptyTrimmed(entry.SourceAlias, item.SourceAlias)
		next.SourceModelID = NormalizeModelCatalogModelID(firstNonEmptyTrimmed(entry.SourceModelID, item.SourceModelID, item.BaseModel))
		next.BaseModel = NormalizeModelCatalogModelID(firstNonEmptyTrimmed(entry.BaseModel, item.BaseModel, next.SourceModelID))
		next.SourceProtocol = firstNonEmptyTrimmed(entry.SourceProtocol, item.SourceProtocol)
		if len(entry.SalePriceDisplay.Primary) > 0 || len(entry.SalePriceDisplay.Secondary) > 0 {
			next.SalePriceDisplay = normalizePublicModelCatalogPriceDisplay(entry.SalePriceDisplay)
			next.PriceDisplay = next.SalePriceDisplay
			next.MultiplierSummary = PublicModelCatalogMultiplierSummary{
				Enabled: false,
				Kind:    publicModelCatalogMultiplierDisabled,
			}
		}
		selected = append(selected, next)
	}
	return selected, nil
}

func validatePublicModelCatalogPriceDisplay(display PublicModelCatalogPriceDisplay) error {
	for _, entry := range append(append([]PublicModelCatalogPriceEntry(nil), display.Primary...), display.Secondary...) {
		if entry.Value < 0 {
			return infraerrors.BadRequest("PUBLIC_MODEL_PRICE_INVALID", "sale price must be non-negative")
		}
	}
	return nil
}

func publicModelCatalogActor(actors []ModelCatalogActor) ModelCatalogActor {
	if len(actors) == 0 {
		return ModelCatalogActor{}
	}
	return actors[0]
}

func publicModelCatalogAuditFields(ctx context.Context, actor ModelCatalogActor) []zap.Field {
	fields := []zap.Field{
		zap.String("component", "service.model_catalog"),
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	if actor.UserID > 0 {
		fields = append(fields, zap.Int64("actor_user_id", actor.UserID))
	}
	if strings.TrimSpace(actor.Email) != "" {
		fields = append(fields, zap.String("actor_email", strings.TrimSpace(actor.Email)))
	}
	return fields
}

func publicModelCatalogDraftEntryLogFields(entry PublicModelCatalogEntryDraft) []zap.Field {
	return []zap.Field{
		zap.String("entry_id", strings.TrimSpace(entry.EntryID)),
		zap.String("public_model_id", strings.TrimSpace(entry.PublicModelID)),
		zap.Int64("source_account_id", entry.SourceAccountID),
		zap.String("source_alias", strings.TrimSpace(entry.SourceAlias)),
		zap.String("source_model_id", strings.TrimSpace(entry.SourceModelID)),
		zap.Int("sale_primary_count", len(entry.SalePriceDisplay.Primary)),
		zap.Int("sale_secondary_count", len(entry.SalePriceDisplay.Secondary)),
	}
}

func publicModelCatalogItemLogFields(item PublicModelCatalogItem) []zap.Field {
	return []zap.Field{
		zap.String("entry_id", strings.TrimSpace(item.EntryID)),
		zap.String("public_model_id", strings.TrimSpace(firstNonEmptyTrimmed(item.PublicModelID, item.Model))),
		zap.Int64("source_account_id", item.SourceAccountID),
		zap.String("source_alias", strings.TrimSpace(item.SourceAlias)),
		zap.String("source_model_id", strings.TrimSpace(item.SourceModelID)),
		zap.Int("sale_primary_count", len(item.SalePriceDisplay.Primary)),
		zap.Int("sale_secondary_count", len(item.SalePriceDisplay.Secondary)),
	}
}

func logPublicModelCatalogDraftSalePriceChanges(
	ctx context.Context,
	actor ModelCatalogActor,
	previous *PublicModelCatalogDraft,
	next PublicModelCatalogDraft,
) {
	if len(next.SelectedEntries) == 0 {
		return
	}
	previousEntries := map[string]PublicModelCatalogEntryDraft{}
	if previous != nil {
		for _, entry := range previous.SelectedEntries {
			normalized := normalizePublicModelCatalogEntryDraft(entry)
			if normalized.EntryID != "" {
				previousEntries[normalized.EntryID] = normalized
			}
		}
	}
	baseFields := publicModelCatalogAuditFields(ctx, actor)
	for _, entry := range next.SelectedEntries {
		previousEntry, existed := previousEntries[entry.EntryID]
		if existed && reflect.DeepEqual(previousEntry.SalePriceDisplay, entry.SalePriceDisplay) {
			continue
		}
		fields := append([]zap.Field{}, baseFields...)
		fields = append(fields, publicModelCatalogDraftEntryLogFields(entry)...)
		fields = append(fields, zap.Bool("new_entry", !existed))
		logger.FromContext(ctx).Info("public model catalog draft sale price updated", fields...)
	}
}

func (s *ModelCatalogService) GetPublicModelCatalogDraftPayload(ctx context.Context, force bool) (*PublicModelCatalogDraftPayload, error) {
	draft := normalizePublicModelCatalogDraft(s.loadPublicModelCatalogDraft(ctx))
	availableSnapshot, availableSource, err := s.publicModelCatalogDraftCandidateSnapshot(ctx, force)
	if err != nil {
		return nil, err
	}
	return &PublicModelCatalogDraftPayload{
		Draft:              draft,
		AvailableItems:     append([]PublicModelCatalogItem(nil), availableSnapshot.Items...),
		AvailableEntries:   append([]PublicModelCatalogItem(nil), availableSnapshot.Items...),
		AvailableUpdatedAt: availableSnapshot.UpdatedAt,
		AvailableSource:    availableSource,
		Published:          publicModelCatalogPublishedSummary(s.loadPublishedPublicModelCatalogSnapshot(ctx)),
	}, nil
}

func (s *ModelCatalogService) publicModelCatalogDraftCandidateSnapshot(
	ctx context.Context,
	force bool,
) (*PublicModelCatalogSnapshot, string, error) {
	if !force {
		if persisted := s.loadPublicModelCatalogDraftCandidateSnapshot(ctx); persisted != nil {
			logger.FromContext(ctx).Info(
				"public model catalog draft candidate snapshot loaded",
				zap.String("component", "service.model_catalog"),
				zap.Int("model_count", len(persisted.Items)),
				zap.String("updated_at", persisted.UpdatedAt),
			)
			return persisted, publicModelCatalogDraftAvailableSourcePersisted, nil
		}
		if cached, age, ok := s.getFreshPublicModelCatalogSnapshotWithTTL(publicModelCatalogDraftLiveTTL); ok {
			logger.FromContext(ctx).Info(
				"public model catalog draft candidate cache hit",
				zap.String("component", "service.model_catalog"),
				zap.Duration("cache_age", age),
				zap.Int("model_count", len(cached.Items)),
			)
			return cached, publicModelCatalogDraftAvailableSourceCache, nil
		}
	}

	availableSource := publicModelCatalogDraftAvailableSourceRefreshed
	if !force {
		availableSource = publicModelCatalogDraftAvailableSourceBootstrap
	}
	liveSnapshot, err := s.buildLivePublicModelCatalogSnapshot(ctx)
	if err != nil {
		if fallback, age, ok := s.getFreshPublicModelCatalogSnapshotWithTTL(publicModelCatalogDraftLiveTTL); ok {
			logger.FromContext(ctx).Warn(
				"public model catalog draft candidate cache fallback",
				zap.String("component", "service.model_catalog"),
				zap.Duration("cache_age", age),
				zap.Int("model_count", len(fallback.Items)),
				zap.Error(err),
			)
			return fallback, publicModelCatalogDraftAvailableSourceCache, nil
		}
		return nil, "", err
	}
	s.storePublicModelCatalogSnapshot(liveSnapshot)
	liveSnapshot = clonePublicModelCatalogSnapshot(liveSnapshot)
	if err := s.persistPublicModelCatalogDraftCandidateSnapshot(ctx, liveSnapshot); err != nil {
		return nil, "", err
	}
	logger.FromContext(ctx).Info(
		"public model catalog draft candidate snapshot refreshed",
		zap.String("component", "service.model_catalog"),
		zap.Bool("force_refresh", force),
		zap.Int("model_count", len(liveSnapshot.Items)),
		zap.String("updated_at", liveSnapshot.UpdatedAt),
	)
	return liveSnapshot, availableSource, nil
}

func (s *ModelCatalogService) publicModelCatalogPublishCandidateSnapshot(ctx context.Context) (*PublicModelCatalogSnapshot, string, error) {
	persisted := s.loadPublicModelCatalogDraftCandidateSnapshot(ctx)
	liveSnapshot, err := s.buildLivePublicModelCatalogSnapshot(ctx)
	if err == nil && liveSnapshot != nil && len(liveSnapshot.Items) > 0 {
		s.storePublicModelCatalogSnapshot(liveSnapshot)
		liveSnapshot = clonePublicModelCatalogSnapshot(liveSnapshot)
		if persistErr := s.persistPublicModelCatalogDraftCandidateSnapshot(ctx, liveSnapshot); persistErr != nil {
			return nil, "", persistErr
		}
		logger.FromContext(ctx).Info(
			"public model catalog publish candidate snapshot refreshed",
			zap.String("component", "service.model_catalog"),
			zap.Int("model_count", len(liveSnapshot.Items)),
			zap.String("updated_at", liveSnapshot.UpdatedAt),
		)
		return liveSnapshot, publicModelCatalogDraftAvailableSourceRefreshed, nil
	}
	if persisted != nil && publicModelCatalogSnapshotFresh(persisted, publicModelCatalogDraftLiveTTL) {
		fields := []zap.Field{
			zap.String("component", "service.model_catalog"),
			zap.Int("model_count", len(persisted.Items)),
			zap.String("updated_at", persisted.UpdatedAt),
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
		}
		logger.FromContext(ctx).Warn("public model catalog publish using fresh persisted candidate snapshot", fields...)
		return persisted, publicModelCatalogDraftAvailableSourcePersisted, nil
	}
	if err != nil {
		return nil, "", err
	}
	return liveSnapshot, publicModelCatalogDraftAvailableSourceRefreshed, nil
}

func publicModelCatalogSnapshotFresh(snapshot *PublicModelCatalogSnapshot, ttl time.Duration) bool {
	if snapshot == nil || ttl <= 0 {
		return false
	}
	updatedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(snapshot.UpdatedAt))
	if err != nil {
		return false
	}
	return time.Since(updatedAt) <= ttl
}

func (s *ModelCatalogService) SavePublicModelCatalogDraft(ctx context.Context, draft PublicModelCatalogDraft, actors ...ModelCatalogActor) (*PublicModelCatalogDraft, error) {
	actor := publicModelCatalogActor(actors)
	previous := s.loadPublicModelCatalogDraft(ctx)
	normalized := normalizePublicModelCatalogDraft(&draft)
	normalized.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.persistPublicModelCatalogDraft(ctx, &normalized); err != nil {
		return nil, err
	}
	fields := publicModelCatalogAuditFields(ctx, actor)
	fields = append(fields,
		zap.Int("selected_model_count", len(normalized.SelectedModels)),
		zap.Int("selected_entry_count", len(normalized.SelectedEntries)),
		zap.Int("page_size", normalized.PageSize),
	)
	logger.FromContext(ctx).Info(
		"public model catalog draft saved",
		fields...,
	)
	logPublicModelCatalogDraftSalePriceChanges(ctx, actor, previous, normalized)
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
	availableSnapshot, availableSource, err := s.publicModelCatalogPublishCandidateSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	selectedItems, err := selectPublicModelCatalogPublishItems(draft, availableSnapshot.Items)
	if err != nil {
		return nil, err
	}
	if len(selectedItems) == 0 && len(availableSnapshot.Items) > 0 {
		return nil, infraerrors.BadRequest("PUBLIC_MODEL_CATALOG_EMPTY", "no models selected for publish")
	}
	details := make(map[string]PublicModelCatalogDetail, len(selectedItems))
	for _, item := range selectedItems {
		exampleSource, exampleProtocol, examplePageID, exampleMarkdown, exampleOverrideID := s.buildPublicModelCatalogDetailExample(ctx, item)
		publicID := firstNonEmptyTrimmed(item.PublicModelID, item.Model)
		details[publicID] = PublicModelCatalogDetail{
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
			ETag:      availableSnapshot.ETag,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			PageSize:  normalizePublicModelCatalogPageSize(draft.PageSize),
			Items:     clonePublicModelCatalogItems(selectedItems),
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
	fields := publicModelCatalogAuditFields(ctx, actor)
	fields = append(fields,
		zap.String("etag", summary.ETag),
		zap.Int("model_count", summary.ModelCount),
		zap.Int("entry_count", len(draft.SelectedEntries)),
		zap.Int("page_size", summary.PageSize),
		zap.String("available_source", availableSource),
		zap.String("available_updated_at", availableSnapshot.UpdatedAt),
	)
	logger.FromContext(ctx).Info(
		"public model catalog published",
		fields...,
	)
	publishEntryFields := publicModelCatalogAuditFields(ctx, actor)
	for _, item := range selectedItems {
		fields := append([]zap.Field{}, publishEntryFields...)
		fields = append(fields, publicModelCatalogItemLogFields(item)...)
		logger.FromContext(ctx).Info("public model catalog published entry", fields...)
	}
	return summary, nil
}

func (s *ModelCatalogService) GetPublishedPublicModelCatalogSummary(ctx context.Context) (*PublicModelCatalogPublishedSummary, error) {
	return publicModelCatalogPublishedSummary(s.loadPublishedPublicModelCatalogSnapshot(ctx)), nil
}
