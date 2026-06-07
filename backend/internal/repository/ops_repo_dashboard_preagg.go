package repository

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsDashboardPartial struct {
	successCount         int64
	errorCountTotal      int64
	businessLimitedCount int64
	errorCountSLA        int64

	upstreamErrorCountExcl429529 int64
	upstream429Count             int64
	upstream529Count             int64

	tokenConsumed int64

	ttftSampleCount int64
	duration        service.OpsPercentiles
	ttft            service.OpsPercentiles
}

func (r *opsRepository) getDashboardOverviewPreaggregated(ctx context.Context, filter *service.OpsDashboardFilter) (*service.OpsDashboardOverview, error) {
	if filter == nil {
		return nil, fmt.Errorf("nil filter")
	}

	start := filter.StartTime.UTC()
	end := filter.EndTime.UTC()

	// Stable full-hour range covered by pre-aggregation.
	aggSafeEnd := preaggSafeEnd(end)
	aggFullStart := utcCeilToHour(start)
	aggFullEnd := utcFloorToHour(aggSafeEnd)

	// If there are no stable full-hour buckets, use raw directly (short windows).
	if !aggFullStart.Before(aggFullEnd) {
		return r.getDashboardOverviewRaw(ctx, filter)
	}

	// 1) Pre-aggregated stable segment.
	preaggRows, err := r.listHourlyMetricsRows(ctx, filter, aggFullStart, aggFullEnd)
	if err != nil {
		return nil, err
	}
	if len(preaggRows) == 0 {
		// Distinguish "no data" vs "preagg not populated yet".
		if exists, err := r.rawOpsDataExists(ctx, filter, aggFullStart, aggFullEnd); err == nil && exists {
			return nil, service.ErrOpsPreaggregatedNotPopulated
		}
	}
	preagg := aggregateHourlyRows(preaggRows)

	// 2) Raw head/tail fragments (at most ~1 hour each).
	head := opsDashboardPartial{}
	tail := opsDashboardPartial{}

	if start.Before(aggFullStart) {
		part, err := r.queryRawPartial(ctx, filter, start, minTime(end, aggFullStart))
		if err != nil {
			return nil, err
		}
		head = *part
	}
	if aggFullEnd.Before(end) {
		part, err := r.queryRawPartial(ctx, filter, maxTime(start, aggFullEnd), end)
		if err != nil {
			return nil, err
		}
		tail = *part
	}

	// Merge counts.
	successCount := preagg.successCount + head.successCount + tail.successCount
	errorTotal := preagg.errorCountTotal + head.errorCountTotal + tail.errorCountTotal
	businessLimited := preagg.businessLimitedCount + head.businessLimitedCount + tail.businessLimitedCount
	errorCountSLA := preagg.errorCountSLA + head.errorCountSLA + tail.errorCountSLA

	upstreamExcl := preagg.upstreamErrorCountExcl429529 + head.upstreamErrorCountExcl429529 + tail.upstreamErrorCountExcl429529
	upstream429 := preagg.upstream429Count + head.upstream429Count + tail.upstream429Count
	upstream529 := preagg.upstream529Count + head.upstream529Count + tail.upstream529Count

	tokenConsumed := preagg.tokenConsumed + head.tokenConsumed + tail.tokenConsumed

	// Approximate percentiles across segments:
	// - p50/p90/avg: weighted average by success_count
	// - p95/p99/max: max (conservative tail)
	duration := combineApproxPercentiles([]opsPercentileSegment{
		{weight: preagg.successCount, p: preagg.duration},
		{weight: head.successCount, p: head.duration},
		{weight: tail.successCount, p: tail.duration},
	})
	ttft := combineApproxPercentiles([]opsPercentileSegment{
		{weight: preagg.ttftSampleCount, p: preagg.ttft},
		{weight: head.ttftSampleCount, p: head.ttft},
		{weight: tail.ttftSampleCount, p: tail.ttft},
	})

	windowSeconds := end.Sub(start).Seconds()
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	requestCountTotal := successCount + errorTotal
	requestCountSLA := successCount + errorCountSLA

	sla := safeDivideFloat64(float64(successCount), float64(requestCountSLA))
	errorRate := safeDivideFloat64(float64(errorCountSLA), float64(requestCountSLA))
	upstreamErrorRate := safeDivideFloat64(float64(upstreamExcl), float64(requestCountSLA))
	degraded := false

	// Keep "current" rates as raw, to preserve realtime semantics.
	qpsCurrent, tpsCurrent, err := r.queryCurrentRates(ctx, filter, end)
	if err != nil {
		if isQueryTimeoutErr(err) {
			degraded = true
		} else {
			return nil, err
		}
	}

	peakCtx, cancelPeak := context.WithTimeout(ctx, opsRawPeakQueryTimeout)
	qpsPeak, tpsPeak, err := r.queryPeakRates(peakCtx, filter, start, end)
	cancelPeak()
	if err != nil {
		if isQueryTimeoutErr(err) {
			degraded = true
		} else {
			return nil, err
		}
	}

	qpsAvg := roundTo1DP(float64(requestCountTotal) / windowSeconds)
	tpsAvg := roundTo1DP(float64(tokenConsumed) / windowSeconds)
	if degraded {
		if qpsCurrent <= 0 {
			qpsCurrent = qpsAvg
		}
		if tpsCurrent <= 0 {
			tpsCurrent = tpsAvg
		}
		if qpsPeak <= 0 {
			qpsPeak = roundTo1DP(math.Max(qpsCurrent, qpsAvg))
		}
		if tpsPeak <= 0 {
			tpsPeak = roundTo1DP(math.Max(tpsCurrent, tpsAvg))
		}
	}

	return &service.OpsDashboardOverview{
		StartTime: start,
		EndTime:   end,
		Platform:  strings.TrimSpace(filter.Platform),
		GroupID:   filter.GroupID,
		ChannelID: filter.ChannelID,

		SuccessCount:         successCount,
		ErrorCountTotal:      errorTotal,
		BusinessLimitedCount: businessLimited,
		ErrorCountSLA:        errorCountSLA,
		RequestCountTotal:    requestCountTotal,
		RequestCountSLA:      requestCountSLA,
		TokenConsumed:        tokenConsumed,

		SLA:                          roundTo4DP(sla),
		ErrorRate:                    roundTo4DP(errorRate),
		UpstreamErrorRate:            roundTo4DP(upstreamErrorRate),
		UpstreamErrorCountExcl429529: upstreamExcl,
		Upstream429Count:             upstream429,
		Upstream529Count:             upstream529,

		QPS: service.OpsRateSummary{
			Current: qpsCurrent,
			Peak:    qpsPeak,
			Avg:     qpsAvg,
		},
		TPS: service.OpsRateSummary{
			Current: tpsCurrent,
			Peak:    tpsPeak,
			Avg:     tpsAvg,
		},

		Duration: duration,
		TTFT:     ttft,
	}, nil
}

func (r *opsRepository) queryRawPartial(ctx context.Context, filter *service.OpsDashboardFilter, start, end time.Time) (*opsDashboardPartial, error) {
	successCount, tokenConsumed, err := r.queryUsageCounts(ctx, filter, start, end)
	if err != nil {
		return nil, err
	}

	latencyCtx, cancelLatency := context.WithTimeout(ctx, opsRawLatencyQueryTimeout)
	duration, ttft, ttftSampleCount, err := r.queryUsageLatency(latencyCtx, filter, start, end)
	cancelLatency()
	if err != nil {
		if isQueryTimeoutErr(err) {
			duration = service.OpsPercentiles{}
			ttft = service.OpsPercentiles{}
			ttftSampleCount = 0
		} else {
			return nil, err
		}
	}

	errorTotal, businessLimited, errorCountSLA, upstreamExcl, upstream429, upstream529, err := r.queryErrorCounts(ctx, filter, start, end)
	if err != nil {
		return nil, err
	}

	return &opsDashboardPartial{
		successCount:                 successCount,
		errorCountTotal:              errorTotal,
		businessLimitedCount:         businessLimited,
		errorCountSLA:                errorCountSLA,
		upstreamErrorCountExcl429529: upstreamExcl,
		upstream429Count:             upstream429,
		upstream529Count:             upstream529,
		tokenConsumed:                tokenConsumed,
		ttftSampleCount:              ttftSampleCount,
		duration:                     duration,
		ttft:                         ttft,
	}, nil
}

func (r *opsRepository) rawOpsDataExists(ctx context.Context, filter *service.OpsDashboardFilter, start, end time.Time) (bool, error) {
	{
		join, where, args, _ := buildUsageWhere(filter, start, end, 1)
		q := `SELECT EXISTS(SELECT 1 FROM usage_logs ul ` + join + ` ` + where + ` LIMIT 1)`
		var exists bool
		if err := r.db.QueryRowContext(ctx, q, args...).Scan(&exists); err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}

	{
		where, args, _ := buildErrorWhere(filter, start, end, 1)
		q := `SELECT EXISTS(SELECT 1 FROM ops_error_logs ` + where + ` LIMIT 1)`
		var exists bool
		if err := r.db.QueryRowContext(ctx, q, args...).Scan(&exists); err != nil {
			return false, err
		}
		return exists, nil
	}
}
