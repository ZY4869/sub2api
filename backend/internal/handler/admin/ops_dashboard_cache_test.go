package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type opsDashboardRepoCacheProbe struct {
	service.OpsRepository
	throughputCalls        atomic.Int32
	latencyHistogramCalls  atomic.Int32
	errorDistributionCalls atomic.Int32
}

func (r *opsDashboardRepoCacheProbe) GetThroughputTrend(
	ctx context.Context,
	filter *service.OpsDashboardFilter,
	bucketSeconds int,
) (*service.OpsThroughputTrendResponse, error) {
	r.throughputCalls.Add(1)
	return &service.OpsThroughputTrendResponse{
		Bucket: "5m",
		Points: []*service.OpsThroughputTrendPoint{{
			BucketStart:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			RequestCount:  3,
			TokenConsumed: 9,
			SwitchCount:   1,
		}},
	}, nil
}

func (r *opsDashboardRepoCacheProbe) GetLatencyHistogram(
	ctx context.Context,
	filter *service.OpsDashboardFilter,
) (*service.OpsLatencyHistogramResponse, error) {
	r.latencyHistogramCalls.Add(1)
	return &service.OpsLatencyHistogramResponse{
		TotalRequests: 10,
		Buckets: []*service.OpsLatencyHistogramBucket{{
			Range: "0-1s",
			Count: 8,
		}},
	}, nil
}

func (r *opsDashboardRepoCacheProbe) GetErrorDistribution(
	ctx context.Context,
	filter *service.OpsDashboardFilter,
) (*service.OpsErrorDistributionResponse, error) {
	r.errorDistributionCalls.Add(1)
	return &service.OpsErrorDistributionResponse{
		Total: 4,
		Items: []*service.OpsErrorDistributionItem{{
			StatusCode: 500,
			Total:      4,
		}},
	}, nil
}

func resetOpsDashboardReadCachesForTest() {
	opsDashboardThroughputTrendCache = newSnapshotCache(30 * time.Second)
	opsDashboardLatencyHistogramCache = newSnapshotCache(30 * time.Second)
	opsDashboardErrorDistributionCache = newSnapshotCache(30 * time.Second)
}

func newTestOpsHandler(repo service.OpsRepository) *OpsHandler {
	return NewOpsHandler(service.NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))
}

func TestOpsHandler_GetDashboardLatencyHistogram_UsesCacheAndETag(t *testing.T) {
	t.Cleanup(resetOpsDashboardReadCachesForTest)
	resetOpsDashboardReadCachesForTest()

	gin.SetMode(gin.TestMode)
	repo := &opsDashboardRepoCacheProbe{}
	handler := newTestOpsHandler(repo)
	router := gin.New()
	router.GET("/admin/ops/dashboard/latency-histogram", handler.GetDashboardLatencyHistogram)

	url := "/admin/ops/dashboard/latency-histogram?time_range=1h&platform=openai&group_id=7&mode=auto"

	req1 := httptest.NewRequest(http.MethodGet, url, nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	require.Equal(t, "miss", rec1.Header().Get("X-Snapshot-Cache"))

	req2 := httptest.NewRequest(http.MethodGet, url, nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "hit", rec2.Header().Get("X-Snapshot-Cache"))
	require.Equal(t, int32(1), repo.latencyHistogramCalls.Load())

	etag := rec2.Header().Get("ETag")
	require.NotEmpty(t, etag)

	req3 := httptest.NewRequest(http.MethodGet, url, nil)
	req3.Header.Set("If-None-Match", etag)
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req3)
	require.Equal(t, http.StatusNotModified, rec3.Code)
	require.Equal(t, int32(1), repo.latencyHistogramCalls.Load())
}

func TestOpsHandler_GetDashboardErrorDistribution_UsesCacheAndETag(t *testing.T) {
	t.Cleanup(resetOpsDashboardReadCachesForTest)
	resetOpsDashboardReadCachesForTest()

	gin.SetMode(gin.TestMode)
	repo := &opsDashboardRepoCacheProbe{}
	handler := newTestOpsHandler(repo)
	router := gin.New()
	router.GET("/admin/ops/dashboard/error-distribution", handler.GetDashboardErrorDistribution)

	url := "/admin/ops/dashboard/error-distribution?time_range=6h&platform=openai&mode=auto"

	req1 := httptest.NewRequest(http.MethodGet, url, nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	require.Equal(t, "miss", rec1.Header().Get("X-Snapshot-Cache"))

	req2 := httptest.NewRequest(http.MethodGet, url, nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "hit", rec2.Header().Get("X-Snapshot-Cache"))
	require.Equal(t, int32(1), repo.errorDistributionCalls.Load())

	etag := rec2.Header().Get("ETag")
	require.NotEmpty(t, etag)

	req3 := httptest.NewRequest(http.MethodGet, url, nil)
	req3.Header.Set("If-None-Match", etag)
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req3)
	require.Equal(t, http.StatusNotModified, rec3.Code)
	require.Equal(t, int32(1), repo.errorDistributionCalls.Load())
}

func TestOpsHandler_GetDashboardThroughputTrend_UsesCacheAndETag(t *testing.T) {
	t.Cleanup(resetOpsDashboardReadCachesForTest)
	resetOpsDashboardReadCachesForTest()

	gin.SetMode(gin.TestMode)
	repo := &opsDashboardRepoCacheProbe{}
	handler := newTestOpsHandler(repo)
	router := gin.New()
	router.GET("/admin/ops/dashboard/throughput-trend", handler.GetDashboardThroughputTrend)

	url := "/admin/ops/dashboard/throughput-trend?start_time=2026-03-01T00:00:00Z&end_time=2026-03-01T05:00:00Z&platform=openai&mode=auto"

	req1 := httptest.NewRequest(http.MethodGet, url, nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	require.Equal(t, "miss", rec1.Header().Get("X-Snapshot-Cache"))

	req2 := httptest.NewRequest(http.MethodGet, url, nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "hit", rec2.Header().Get("X-Snapshot-Cache"))
	require.Equal(t, int32(1), repo.throughputCalls.Load())

	etag := rec2.Header().Get("ETag")
	require.NotEmpty(t, etag)

	req3 := httptest.NewRequest(http.MethodGet, url, nil)
	req3.Header.Set("If-None-Match", etag)
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req3)
	require.Equal(t, http.StatusNotModified, rec3.Code)
	require.Equal(t, int32(1), repo.throughputCalls.Load())
}
