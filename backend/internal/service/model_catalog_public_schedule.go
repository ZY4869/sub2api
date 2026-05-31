package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

func publicModelCatalogItemTimeAccessPolicy(item PublicModelCatalogItem) *TimeAccessPolicy {
	policy := cloneTimeAccessPolicy(item.AccessTimePolicy)
	if strings.TrimSpace(item.AvailableFrom) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(item.AvailableFrom)); err == nil {
			if policy == nil {
				policy = &TimeAccessPolicy{}
			}
			policy.NotBefore = &t
		}
	}
	if strings.TrimSpace(item.AvailableUntil) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(item.AvailableUntil)); err == nil {
			if policy == nil {
				policy = &TimeAccessPolicy{}
			}
			policy.NotAfter = &t
		}
	}
	if policy != nil {
		policy.Enabled = true
	}
	return policy
}

func publicModelCatalogItemScheduleStatus(item PublicModelCatalogItem, now time.Time) string {
	start := time.Now()
	eval := EvaluateTimeAccessPolicy(publicModelCatalogItemTimeAccessPolicy(item), now)
	protocolruntime.RecordTimePolicyDecision("model", eval.Allowed, eval.Reason, time.Since(start).Milliseconds())
	if eval.Allowed {
		return ModelRegistryScheduleActive
	}
	switch eval.Reason {
	case TimeAccessDecisionNotBefore:
		return ModelRegistryScheduleScheduled
	case TimeAccessDecisionNotAfter:
		return ModelRegistryScheduleExpired
	case TimeAccessDecisionOutsideWindow:
		return ModelRegistryScheduleOutOfWindow
	default:
		return ModelRegistryScheduleInvalid
	}
}

func publicModelCatalogItemCurrentlyAvailable(item PublicModelCatalogItem, now time.Time) bool {
	return publicModelCatalogItemScheduleStatus(item, now) == ModelRegistryScheduleActive
}

func (s *ModelCatalogService) publicModelCatalogPublishedItemExists(ctx context.Context, publicModelID string) bool {
	_, ok := s.publicModelCatalogPublishedItemByPublicID(ctx, publicModelID)
	return ok
}

func (s *ModelCatalogService) publicModelCatalogPublishedItemCurrentlyAvailable(ctx context.Context, publicModelID string) bool {
	item, ok := s.publicModelCatalogPublishedItemByPublicID(ctx, publicModelID)
	if !ok {
		return false
	}
	return publicModelCatalogItemCurrentlyAvailable(item, time.Now())
}

func (s *ModelCatalogService) publicModelCatalogPublishedItemByPublicID(ctx context.Context, publicModelID string) (PublicModelCatalogItem, bool) {
	if s == nil {
		return PublicModelCatalogItem{}, false
	}
	normalizedPublicID := NormalizeModelCatalogModelID(publicModelID)
	if normalizedPublicID == "" {
		return PublicModelCatalogItem{}, false
	}
	published := s.loadPublishedPublicModelCatalogSnapshot(ctx)
	if published == nil {
		return PublicModelCatalogItem{}, false
	}
	for _, item := range published.Snapshot.Items {
		if publicModelCatalogItemMatchesPublicID(item, normalizedPublicID) {
			return item, true
		}
	}
	for _, detail := range published.Details {
		if publicModelCatalogItemMatchesPublicID(detail.Item, normalizedPublicID) {
			return detail.Item, true
		}
	}
	return PublicModelCatalogItem{}, false
}

func applyPublicModelCatalogDraftSchedule(item PublicModelCatalogItem, draft PublicModelCatalogEntryDraft) PublicModelCatalogItem {
	item.AvailableFrom = normalizeRegistryOptionalRFC3339(draft.AvailableFrom)
	item.AvailableUntil = normalizeRegistryOptionalRFC3339(draft.AvailableUntil)
	item.AccessTimePolicy = cloneTimeAccessPolicy(draft.AccessTimePolicy)
	item.ScheduleStatus = publicModelCatalogItemScheduleStatus(item, time.Now())
	return item
}
