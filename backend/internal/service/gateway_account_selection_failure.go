package service

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"strings"
	"time"
)

type selectionFailureStats struct {
	Total              int
	Eligible           int
	Excluded           int
	Unschedulable      int
	PlatformFiltered   int
	ModelUnsupported   int
	ModelRateLimited   int
	SamplePlatformIDs  []int64
	SampleMappingIDs   []int64
	SampleRateLimitIDs []string
}
type selectionFailureDiagnosis struct {
	Category string
	Detail   string
}

func (s *GatewayService) logDetailedSelectionFailure(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, platform string, accounts []Account, excludedIDs map[int64]struct{}, allowMixedScheduling bool) selectionFailureStats {
	stats := s.collectSelectionFailureStats(ctx, accounts, requestedModel, platform, excludedIDs, allowMixedScheduling)
	logger.LegacyPrintf("service.gateway", "[SelectAccountDetailed] group_id=%v model=%s platform=%s session=%s total=%d eligible=%d excluded=%d unschedulable=%d platform_filtered=%d model_unsupported=%d model_rate_limited=%d sample_platform_filtered=%v sample_model_unsupported=%v sample_model_rate_limited=%v", derefGroupID(groupID), requestedModel, platform, shortSessionHash(sessionHash), stats.Total, stats.Eligible, stats.Excluded, stats.Unschedulable, stats.PlatformFiltered, stats.ModelUnsupported, stats.ModelRateLimited, stats.SamplePlatformIDs, stats.SampleMappingIDs, stats.SampleRateLimitIDs)
	return stats
}
func (s *GatewayService) collectSelectionFailureStats(ctx context.Context, accounts []Account, requestedModel string, platform string, excludedIDs map[int64]struct{}, allowMixedScheduling bool) selectionFailureStats {
	stats := selectionFailureStats{Total: len(accounts)}
	for i := range accounts {
		acc := &accounts[i]
		diagnosis := s.diagnoseSelectionFailure(ctx, acc, requestedModel, platform, excludedIDs, allowMixedScheduling)
		switch diagnosis.Category {
		case "excluded":
			stats.Excluded++
		case "unschedulable":
			stats.Unschedulable++
		case "platform_filtered":
			stats.PlatformFiltered++
			stats.SamplePlatformIDs = appendSelectionFailureSampleID(stats.SamplePlatformIDs, acc.ID)
		case "model_unsupported":
			stats.ModelUnsupported++
			stats.SampleMappingIDs = appendSelectionFailureSampleID(stats.SampleMappingIDs, acc.ID)
		case "model_rate_limited":
			stats.ModelRateLimited++
			remaining := acc.GetRateLimitRemainingTimeWithContext(ctx, requestedModel).Truncate(time.Second)
			stats.SampleRateLimitIDs = appendSelectionFailureRateSample(stats.SampleRateLimitIDs, acc.ID, remaining)
		default:
			stats.Eligible++
		}
	}
	return stats
}
func (s *GatewayService) diagnoseSelectionFailure(ctx context.Context, acc *Account, requestedModel string, platform string, excludedIDs map[int64]struct{}, allowMixedScheduling bool) selectionFailureDiagnosis {
	if acc == nil {
		return selectionFailureDiagnosis{Category: "unschedulable", Detail: "account_nil"}
	}
	if _, excluded := excludedIDs[acc.ID]; excluded {
		return selectionFailureDiagnosis{Category: "excluded"}
	}
	if !s.isAccountSchedulableForSelection(acc) {
		return selectionFailureDiagnosis{Category: "unschedulable", Detail: "generic_unschedulable"}
	}
	if isPlatformFilteredForSelection(acc, platform, allowMixedScheduling) {
		return selectionFailureDiagnosis{Category: "platform_filtered", Detail: fmt.Sprintf("account_platform=%s requested_platform=%s", acc.Platform, strings.TrimSpace(platform))}
	}
	if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, acc, requestedModel) {
		return selectionFailureDiagnosis{Category: "model_unsupported", Detail: fmt.Sprintf("model=%s", requestedModel)}
	}
	if !s.isAccountSchedulableForModelSelection(ctx, acc, requestedModel) {
		remaining := acc.GetRateLimitRemainingTimeWithContext(ctx, requestedModel).Truncate(time.Second)
		return selectionFailureDiagnosis{Category: "model_rate_limited", Detail: fmt.Sprintf("remaining=%s", remaining)}
	}
	return selectionFailureDiagnosis{Category: "eligible"}
}
func isPlatformFilteredForSelection(acc *Account, platform string, allowMixedScheduling bool) bool {
	if acc == nil {
		return true
	}
	if allowMixedScheduling {
		if acc.Platform == PlatformAntigravity {
			return !acc.IsMixedSchedulingEnabled()
		}
		return acc.Platform != platform
	}
	if strings.TrimSpace(platform) == "" {
		return false
	}
	return acc.Platform != platform
}
func appendSelectionFailureSampleID(samples []int64, id int64) []int64 {
	const limit = 5
	if len(samples) >= limit {
		return samples
	}
	return append(samples, id)
}
func appendSelectionFailureRateSample(samples []string, accountID int64, remaining time.Duration) []string {
	const limit = 5
	if len(samples) >= limit {
		return samples
	}
	return append(samples, fmt.Sprintf("%d(%s)", accountID, remaining))
}
func summarizeSelectionFailureStats(stats selectionFailureStats) string {
	return fmt.Sprintf("total=%d eligible=%d excluded=%d unschedulable=%d platform_filtered=%d model_unsupported=%d model_rate_limited=%d", stats.Total, stats.Eligible, stats.Excluded, stats.Unschedulable, stats.PlatformFiltered, stats.ModelUnsupported, stats.ModelRateLimited)
}
