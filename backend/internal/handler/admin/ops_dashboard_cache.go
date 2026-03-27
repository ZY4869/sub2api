package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var (
	opsDashboardThroughputTrendCache   = newSnapshotCache(30 * time.Second)
	opsDashboardLatencyHistogramCache  = newSnapshotCache(30 * time.Second)
	opsDashboardErrorDistributionCache = newSnapshotCache(30 * time.Second)
)

type opsDashboardReadCacheKey struct {
	StartTime     string               `json:"start_time"`
	EndTime       string               `json:"end_time"`
	TimeRange     string               `json:"time_range"`
	Platform      string               `json:"platform"`
	GroupID       *int64               `json:"group_id"`
	QueryMode     service.OpsQueryMode `json:"mode"`
	BucketSeconds int                  `json:"bucket_seconds"`
}

func buildOpsDashboardFilterFromRequest(
	c *gin.Context,
	defaultRange string,
) (*service.OpsDashboardFilter, time.Time, time.Time, error) {
	startTime, endTime, err := parseOpsTimeRange(c, defaultRange)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	filter := &service.OpsDashboardFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Platform:  strings.TrimSpace(c.Query("platform")),
		QueryMode: parseOpsQueryMode(c),
	}

	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return nil, time.Time{}, time.Time{}, errInvalidGroupID
		}
		filter.GroupID = &id
	}

	return filter, startTime, endTime, nil
}

func buildOpsDashboardReadCacheKey(
	c *gin.Context,
	defaultRange string,
	filter *service.OpsDashboardFilter,
	bucketSeconds int,
) string {
	if c == nil || filter == nil {
		return ""
	}

	startTime := strings.TrimSpace(c.Query("start_time"))
	endTime := strings.TrimSpace(c.Query("end_time"))
	timeRange := strings.TrimSpace(c.Query("time_range"))
	if startTime == "" && endTime == "" && timeRange == "" {
		timeRange = defaultRange
	}

	keyRaw, _ := json.Marshal(opsDashboardReadCacheKey{
		StartTime:     startTime,
		EndTime:       endTime,
		TimeRange:     timeRange,
		Platform:      filter.Platform,
		GroupID:       filter.GroupID,
		QueryMode:     filter.QueryMode,
		BucketSeconds: bucketSeconds,
	})
	return string(keyRaw)
}

func respondCachedOpsDashboardRead(
	c *gin.Context,
	cache *snapshotCache,
	cacheKey string,
	load func() (any, error),
) {
	entry, hit, err := cache.GetOrLoad(cacheKey, load)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if entry.ETag != "" {
		c.Header("ETag", entry.ETag)
		c.Header("Vary", "If-None-Match")
		if ifNoneMatchMatched(c.GetHeader("If-None-Match"), entry.ETag) {
			c.Status(http.StatusNotModified)
			return
		}
	}

	if hit {
		c.Header("X-Snapshot-Cache", "hit")
	} else {
		c.Header("X-Snapshot-Cache", "miss")
	}
	response.Success(c, entry.Payload)
}

var errInvalidGroupID = responseError("Invalid group_id")

type responseError string

func (e responseError) Error() string {
	return string(e)
}
