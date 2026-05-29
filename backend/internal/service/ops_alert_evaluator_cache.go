package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type opsAlertEvaluationCache struct {
	overview     map[string]opsAlertOverviewCacheEntry
	availability map[string]opsAlertAvailabilityCacheEntry
	stats        opsAlertEvaluationCacheStats
}

type opsAlertEvaluationCacheStats struct {
	OverviewHits       int
	OverviewMisses     int
	AvailabilityHits   int
	AvailabilityMisses int
}

type opsAlertOverviewCacheEntry struct {
	overview *OpsDashboardOverview
	err      error
}

type opsAlertAvailabilityCacheEntry struct {
	availability *OpsAccountAvailability
	err          error
}

func newOpsAlertEvaluationCache() *opsAlertEvaluationCache {
	return &opsAlertEvaluationCache{
		overview:     make(map[string]opsAlertOverviewCacheEntry),
		availability: make(map[string]opsAlertAvailabilityCacheEntry),
	}
}

func (c *opsAlertEvaluationCache) getOverview(
	key string,
	load func() (*OpsDashboardOverview, error),
) (*OpsDashboardOverview, error) {
	if c == nil {
		return load()
	}
	if entry, ok := c.overview[key]; ok {
		c.stats.OverviewHits++
		return entry.overview, entry.err
	}

	overview, err := load()
	c.overview[key] = opsAlertOverviewCacheEntry{
		overview: overview,
		err:      err,
	}
	c.stats.OverviewMisses++
	return overview, err
}

func (c *opsAlertEvaluationCache) getAvailability(
	key string,
	load func() (*OpsAccountAvailability, error),
) (*OpsAccountAvailability, error) {
	if c == nil {
		return load()
	}
	if entry, ok := c.availability[key]; ok {
		c.stats.AvailabilityHits++
		return entry.availability, entry.err
	}

	availability, err := load()
	c.availability[key] = opsAlertAvailabilityCacheEntry{
		availability: availability,
		err:          err,
	}
	c.stats.AvailabilityMisses++
	return availability, err
}

func (c *opsAlertEvaluationCache) formatStats() string {
	if c == nil {
		return "overview_cache_hits=0 overview_cache_misses=0 availability_cache_hits=0 availability_cache_misses=0"
	}
	return fmt.Sprintf(
		"overview_cache_hits=%d overview_cache_misses=%d availability_cache_hits=%d availability_cache_misses=%d",
		c.stats.OverviewHits,
		c.stats.OverviewMisses,
		c.stats.AvailabilityHits,
		c.stats.AvailabilityMisses,
	)
}

func buildOpsAlertOverviewCacheKey(start, end time.Time, platform string, groupID *int64) string {
	return fmt.Sprintf(
		"%s|%s|%s|%d",
		start.UTC().Format(time.RFC3339),
		end.UTC().Format(time.RFC3339),
		strings.TrimSpace(strings.ToLower(platform)),
		normalizedOpsAlertCacheGroupID(groupID),
	)
}

func buildOpsAlertAvailabilityCacheKey(platform string, groupID *int64) string {
	return fmt.Sprintf(
		"%s|%d",
		strings.TrimSpace(strings.ToLower(platform)),
		normalizedOpsAlertCacheGroupID(groupID),
	)
}

func normalizedOpsAlertCacheGroupID(groupID *int64) int64 {
	if groupID == nil || *groupID <= 0 {
		return 0
	}
	return *groupID
}

func (s *OpsAlertEvaluatorService) getCachedAccountAvailability(
	ctx context.Context,
	platform string,
	groupID *int64,
	evalCache *opsAlertEvaluationCache,
) (*OpsAccountAvailability, error) {
	if s == nil || s.opsService == nil {
		return nil, nil
	}

	cacheKey := buildOpsAlertAvailabilityCacheKey(platform, groupID)
	return evalCache.getAvailability(cacheKey, func() (*OpsAccountAvailability, error) {
		return s.opsService.GetAccountAvailability(ctx, platform, groupID)
	})
}

func (s *OpsAlertEvaluatorService) getCachedDashboardOverview(
	ctx context.Context,
	start time.Time,
	end time.Time,
	platform string,
	groupID *int64,
	evalCache *opsAlertEvaluationCache,
) (*OpsDashboardOverview, error) {
	if s == nil || s.opsRepo == nil {
		return nil, nil
	}

	cacheKey := buildOpsAlertOverviewCacheKey(start, end, platform, groupID)
	return evalCache.getOverview(cacheKey, func() (*OpsDashboardOverview, error) {
		return s.opsRepo.GetDashboardOverview(ctx, &OpsDashboardFilter{
			StartTime: start,
			EndTime:   end,
			Platform:  platform,
			GroupID:   groupID,
			QueryMode: OpsQueryModeAuto,
		})
	})
}
