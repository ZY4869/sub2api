package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

func (s *ModelCatalogService) GetPublicModelCatalogRevalidationState(ctx context.Context) PublicModelCatalogRevalidationState {
	return PublicModelCatalogRevalidationState{AutoEnabled: s.IsPublicModelCatalogAutoRevalidationEnabled(ctx)}
}

func (s *ModelCatalogService) IsPublicModelCatalogAutoRevalidationEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyPublicModelCatalogAutoRevalidateEnabled)
	if err != nil {
		return false
	}
	return parseSettingBool(raw, false)
}

func (s *ModelCatalogService) UpdatePublicModelCatalogRevalidationState(
	ctx context.Context,
	input PublicModelCatalogRevalidationInput,
	actor ModelCatalogActor,
) (PublicModelCatalogRevalidationState, error) {
	if s == nil || s.settingRepo == nil {
		return PublicModelCatalogRevalidationState{}, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "model catalog service unavailable")
	}
	if input.AutoEnabled != nil {
		if err := s.settingRepo.Set(ctx, SettingKeyPublicModelCatalogAutoRevalidateEnabled, fmt.Sprintf("%t", *input.AutoEnabled)); err != nil {
			return PublicModelCatalogRevalidationState{}, err
		}
		fields := publicModelCatalogAuditFields(ctx, actor)
		fields = append(fields, zap.Bool("auto_enabled", *input.AutoEnabled))
		logger.FromContext(ctx).Info("public model catalog revalidation setting updated", fields...)
	}
	return s.GetPublicModelCatalogRevalidationState(ctx), nil
}

func (s *ModelCatalogService) RevalidatePublishedPublicModelCatalog(
	ctx context.Context,
	actor ModelCatalogActor,
) (*PublicModelCatalogRevalidationResult, error) {
	if s == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "model catalog service unavailable")
	}
	published := s.loadPublishedPublicModelCatalogSnapshot(ctx)
	if published == nil {
		return nil, infraerrors.NotFound("PUBLIC_MODEL_CATALOG_NOT_PUBLISHED", "published public model catalog not found")
	}
	fields := publicModelCatalogAuditFields(ctx, actor)
	fields = append(fields, zap.Int("model_count", len(published.Snapshot.Items)))
	logger.FromContext(ctx).Info("public model catalog revalidation started", fields...)

	reasons := map[string]int{}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	for index, item := range published.Snapshot.Items {
		routeConfirmed := s.publicModelCatalogItemRouteConfirmed(ctx, item)
		if routeConfirmed {
			protocolruntime.RecordModelCapabilityVerification("success")
		} else {
			protocolruntime.RecordModelCapabilityVerification("failure")
		}
		item = markPublicModelCatalogItemRevalidated(item, checkedAt, routeConfirmed)
		published.Snapshot.Items[index] = item
		publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if publicID != "" {
			if detail, ok := published.Details[publicID]; ok {
				detail.Item = clonePublicModelCatalogItem(item)
				published.Details[publicID] = detail
			}
		}
		itemFields := publicModelCatalogAuditFields(ctx, actor)
		itemFields = append(itemFields, publicModelCatalogItemLogFields(item)...)
		itemFields = append(itemFields,
			zap.Int64("account_id", item.SourceAccountID),
			zap.String("public_model_id", strings.TrimSpace(firstNonEmptyTrimmed(item.PublicModelID, item.Model))),
			zap.String("source_model_id", strings.TrimSpace(item.SourceModelID)),
			zap.Strings("protocols", append([]string(nil), item.RequestProtocols...)),
			zap.Bool("route_confirmed", routeConfirmed),
			zap.String("result", ternaryString(routeConfirmed, "success", "stale")),
			zap.Int("protocol_endpoint_count", len(item.ProtocolEndpoints)),
			zap.Int("capability_matrix_count", len(item.CapabilityMatrix)),
			zap.String("checked_at", checkedAt),
		)
		logger.FromContext(ctx).Info("public model catalog item revalidated", itemFields...)
		if routeConfirmed {
			continue
		}
		reasons[PublicModelUnavailableReasonPublishedSourceUnavailable]++
	}
	published.Snapshot.LastRevalidatedAt = checkedAt
	published.Snapshot.StaleReason = publicModelCatalogStaleReasonSummary(reasons)
	if etag, err := computePublicModelCatalogETag(&published.Snapshot); err != nil {
		fields := publicModelCatalogAuditFields(ctx, actor)
		fields = append(fields, zap.Error(err))
		logger.FromContext(ctx).Warn("public model catalog revalidation failed", fields...)
		return nil, err
	} else {
		published.Snapshot.ETag = etag
	}
	if err := s.persistPublishedPublicModelCatalogSnapshot(ctx, published); err != nil {
		fields := publicModelCatalogAuditFields(ctx, actor)
		fields = append(fields, zap.Error(err))
		logger.FromContext(ctx).Warn("public model catalog revalidation failed", fields...)
		return nil, err
	}
	summary := publicModelCatalogPublishedSummary(published)
	result := &PublicModelCatalogRevalidationResult{
		Published:  *summary,
		CheckedAt:  checkedAt,
		ModelCount: len(published.Snapshot.Items),
		StaleCount: sumStringIntMap(reasons),
		Reasons:    reasons,
	}
	if len(result.Reasons) == 0 {
		result.Reasons = nil
	}
	fields = publicModelCatalogAuditFields(ctx, actor)
	fields = append(fields,
		zap.Int("model_count", result.ModelCount),
		zap.Int("stale_count", result.StaleCount),
		zap.Any("reason_counts", reasons),
	)
	logger.FromContext(ctx).Info("public model catalog revalidation completed", fields...)
	return result, nil
}

func publicModelCatalogStaleReasonSummary(reasons map[string]int) string {
	if len(reasons) == 0 {
		return ""
	}
	keys := make([]string, 0, len(reasons))
	for key := range reasons {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if reasons[key] <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%d", key, reasons[key]))
	}
	return strings.Join(parts, ",")
}

func markPublicModelCatalogItemRevalidated(item PublicModelCatalogItem, checkedAt string, routeConfirmed bool) PublicModelCatalogItem {
	if routeConfirmed {
		item.AvailabilityState = AccountModelAvailabilityVerified
		item.StaleState = AccountModelStaleStateFresh
	} else {
		item.AvailabilityState = AccountModelAvailabilityUnavailable
		item.StaleState = AccountModelStaleStateStale
	}
	item = enrichPublicModelCatalogItemMetadata(item, publicModelCatalogMetadataSourceForPublished(checkedAt))
	for index := range item.ProtocolEndpoints {
		item.ProtocolEndpoints[index].LastCheckedAt = checkedAt
		if routeConfirmed && item.ProtocolEndpoints[index].Support == PublicModelSupportUnknown {
			item.ProtocolEndpoints[index].Support = PublicModelSupportPartial
		}
	}
	for index := range item.CapabilityMatrix {
		item.CapabilityMatrix[index].LastCheckedAt = checkedAt
		if routeConfirmed && item.CapabilityMatrix[index].Support == PublicModelSupportUnknown {
			item.CapabilityMatrix[index].Support = PublicModelSupportPartial
		}
	}
	item.ContextWindow.LastCheckedAt = checkedAt
	return item
}

func sumStringIntMap(values map[string]int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func parseSettingBool(raw string, def bool) bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		return def
	}
}
