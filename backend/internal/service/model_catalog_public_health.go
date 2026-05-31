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

type publicModelCatalogTrafficHealthRepository interface {
	PublicModelCatalogTrafficHealth(ctx context.Context, items []PublicModelCatalogItem, start time.Time, end time.Time) (map[string]PublicModelCatalogStatusItem, error)
}

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
	now := time.Now().UTC()
	start7 := channelMonitorStartDay(now, 7)
	statuses, err := s.publicModelCatalogHealthStatuses(ctx, items, start7, now)
	if err != nil {
		return nil, err
	}
	out := make([]PublicModelCatalogStatusItem, 0, len(items))
	for _, item := range items {
		modelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if modelID == "" {
			continue
		}
		aliases := publicModelCatalogPublicStatusAliases(item)
		if status, ok := statuses[modelID]; ok {
			status = completePublicModelCatalogStatusIdentity(status, modelID, aliases)
			status.RateLimit = nil
			out = append(out, status)
			continue
		}
		status := pendingPublicModelCatalogStatusWithReason(modelID, aliases, PublicModelHealthReasonChecking)
		out = append(out, status)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Model < out[j].Model
	})
	fields := []zap.Field{
		zap.String("component", "service.model_catalog"),
		zap.Int("model_count", len(out)),
		zap.Any("health_source_counts", publicModelCatalogStatusFieldCounts(out, func(item PublicModelCatalogStatusItem) string {
			return item.HealthSource
		})),
		zap.Any("status_reason_counts", publicModelCatalogStatusFieldCounts(out, func(item PublicModelCatalogStatusItem) string {
			return item.StatusReason
		})),
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	logger.FromContext(ctx).Info("public model catalog status snapshot built", fields...)
	return &PublicModelCatalogStatusSnapshot{
		UpdatedAt: now.Format(time.RFC3339),
		Items:     out,
	}, nil
}

func (s *ModelCatalogService) publicModelCatalogHealthStatuses(ctx context.Context, items []PublicModelCatalogItem, start time.Time, end time.Time) (map[string]PublicModelCatalogStatusItem, error) {
	if s == nil || !s.publicCatalogHasConfirmedPublishedOrRoutableItems(ctx, items) {
		return map[string]PublicModelCatalogStatusItem{}, nil
	}
	traffic := map[string]PublicModelCatalogStatusItem{}
	if s.usageHealthRepo != nil {
		statuses, err := s.usageHealthRepo.PublicModelCatalogTrafficHealth(ctx, items, start, end)
		if err != nil {
			return nil, err
		}
		traffic = statuses
	}
	probe, err := s.publicModelCatalogProbeHealthStatuses(ctx, items)
	if err != nil {
		return nil, err
	}
	merged := make(map[string]PublicModelCatalogStatusItem, len(items))
	for _, item := range items {
		modelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if modelID == "" {
			continue
		}
		aliases := publicModelCatalogPublicStatusAliases(item)
		if status, ok := traffic[modelID]; ok && statusHasHealthHistory(status) {
			merged[modelID] = completePublicModelCatalogStatusIdentity(status, modelID, aliases)
			continue
		}
		if status, ok := probe[modelID]; ok {
			merged[modelID] = completePublicModelCatalogStatusIdentity(status, modelID, aliases)
			continue
		}
		if status, ok := traffic[modelID]; ok {
			merged[modelID] = completePublicModelCatalogStatusIdentity(status, modelID, aliases)
		}
	}
	return merged, nil
}

func (s *ModelCatalogService) publicModelCatalogProbeHealthStatuses(ctx context.Context, items []PublicModelCatalogItem) (map[string]PublicModelCatalogStatusItem, error) {
	if s == nil || s.channelMonitorService == nil {
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
	return pendingPublicModelCatalogStatusWithReason(modelID, nil, PublicModelHealthReasonChecking)
}

func pendingPublicModelCatalogStatusWithReason(modelID string, aliases []string, reason string) PublicModelCatalogStatusItem {
	if strings.TrimSpace(reason) == "" {
		reason = PublicModelHealthReasonChecking
	}
	return PublicModelCatalogStatusItem{
		PublicModelID: modelID,
		Model:         modelID,
		Aliases:       clonePublicModelStatusAliases(aliases),
		Status:        PublicModelHealthStatusPending,
		HealthStatus:  PublicModelHealthStatusPending,
		HealthSource:  PublicModelHealthSourceNone,
		StatusReason:  reason,
		Daily:         []PublicModelCatalogDailyStatus{},
		Trend:         []PublicModelCatalogTrendPoint{},
	}
}

func completePublicModelCatalogStatusIdentity(status PublicModelCatalogStatusItem, modelID string, aliases []string) PublicModelCatalogStatusItem {
	status.PublicModelID = modelID
	status.Model = modelID
	status.Aliases = clonePublicModelStatusAliases(aliases)
	if strings.TrimSpace(status.HealthStatus) == "" {
		status.HealthStatus = firstNonEmptyTrimmed(status.Status, PublicModelHealthStatusPending)
	}
	if strings.TrimSpace(status.Status) == "" {
		status.Status = status.HealthStatus
	}
	if strings.TrimSpace(status.HealthSource) == "" {
		status.HealthSource = PublicModelHealthSourceNone
	}
	if strings.TrimSpace(status.StatusReason) == "" {
		status.StatusReason = PublicModelHealthReasonChecking
	}
	if status.Daily == nil {
		status.Daily = []PublicModelCatalogDailyStatus{}
	}
	if status.Trend == nil {
		status.Trend = []PublicModelCatalogTrendPoint{}
	}
	return status
}

func publicModelCatalogPublicStatusAliases(item PublicModelCatalogItem) []string {
	publicModelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	values := []string{publicModelID}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		alias := NormalizeModelCatalogModelID(value)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}
	return out
}

func clonePublicModelStatusAliases(aliases []string) []string {
	if len(aliases) == 0 {
		return []string{}
	}
	out := append([]string(nil), aliases...)
	sort.Strings(out)
	return out
}

func statusHasHealthHistory(status PublicModelCatalogStatusItem) bool {
	if status.SuccessRateToday != nil ||
		status.SuccessRate7d != nil ||
		status.LatencyMs != nil ||
		len(status.Trend) > 0 {
		return true
	}
	for _, day := range status.Daily {
		if day.SuccessRate != nil || day.LatencyMs != nil || day.Status != PublicModelHealthStatusPending {
			return true
		}
	}
	return false
}

func publicModelCatalogStatusFieldCounts(items []PublicModelCatalogStatusItem, field func(PublicModelCatalogStatusItem) string) map[string]int {
	out := map[string]int{}
	for _, item := range items {
		value := strings.TrimSpace(field(item))
		if value == "" {
			value = "unknown"
		}
		out[value]++
	}
	return out
}
