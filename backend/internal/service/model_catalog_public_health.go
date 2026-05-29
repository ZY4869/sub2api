package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func (s *ModelCatalogService) PublicModelCatalogStatusSnapshot(ctx context.Context) (*PublicModelCatalogStatusSnapshot, error) {
	snapshot, err := s.internalPublicModelCatalogSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	items := snapshot.Items
	if len(items) == 0 {
		return &PublicModelCatalogStatusSnapshot{
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			Items:     []PublicModelCatalogStatusItem{},
		}, nil
	}
	statuses, err := s.publicModelCatalogHealthStatuses(ctx, items)
	if err != nil {
		return nil, err
	}
	rateLimits := s.publicModelCatalogRateLimitSummaries(ctx, items)
	out := make([]PublicModelCatalogStatusItem, 0, len(items))
	for _, item := range items {
		modelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if modelID == "" {
			continue
		}
		if status, ok := statuses[modelID]; ok {
			status.RateLimit = clonePublicModelCatalogRateLimitSummary(rateLimits[modelID])
			out = append(out, status)
			continue
		}
		status := pendingPublicModelCatalogStatus(modelID)
		status.RateLimit = clonePublicModelCatalogRateLimitSummary(rateLimits[modelID])
		out = append(out, status)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Model < out[j].Model
	})
	fields := []zap.Field{
		zap.String("component", "service.model_catalog"),
		zap.Int("model_count", len(out)),
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	logger.FromContext(ctx).Info("public model catalog status snapshot built", fields...)
	return &PublicModelCatalogStatusSnapshot{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     out,
	}, nil
}

func (s *ModelCatalogService) publicModelCatalogHealthStatuses(ctx context.Context, items []PublicModelCatalogItem) (map[string]PublicModelCatalogStatusItem, error) {
	if s == nil || s.channelMonitorService == nil || !s.publicCatalogHasConfirmedPublishedOrRoutableItems(ctx, items) {
		return map[string]PublicModelCatalogStatusItem{}, nil
	}
	return s.channelMonitorService.PublicModelCatalogHealth(ctx, items)
}

func (s *ModelCatalogService) publicCatalogHasConfirmedPublishedOrRoutableItems(ctx context.Context, items []PublicModelCatalogItem) bool {
	for _, item := range items {
		if s.publicModelCatalogItemRouteConfirmed(ctx, item) {
			return true
		}
	}
	return false
}

func pendingPublicModelCatalogStatus(modelID string) PublicModelCatalogStatusItem {
	return PublicModelCatalogStatusItem{
		Model:  modelID,
		Status: PublicModelHealthStatusPending,
		Daily:  []PublicModelCatalogDailyStatus{},
		Trend:  []PublicModelCatalogTrendPoint{},
	}
}
