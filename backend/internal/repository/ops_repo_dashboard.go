package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	opsRawLatencyQueryTimeout = 2 * time.Second
	opsRawPeakQueryTimeout    = 1500 * time.Millisecond
)

func (r *opsRepository) GetDashboardOverview(ctx context.Context, filter *service.OpsDashboardFilter) (*service.OpsDashboardOverview, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		return nil, fmt.Errorf("nil filter")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time/end_time required")
	}

	mode := filter.QueryMode
	if !mode.IsValid() {
		mode = service.OpsQueryModeRaw
	}

	switch mode {
	case service.OpsQueryModePreagg:
		return r.getDashboardOverviewPreaggregated(ctx, filter)
	case service.OpsQueryModeAuto:
		out, err := r.getDashboardOverviewPreaggregated(ctx, filter)
		if err != nil && errors.Is(err, service.ErrOpsPreaggregatedNotPopulated) {
			return r.getDashboardOverviewRaw(ctx, filter)
		}
		return out, err
	default:
		return r.getDashboardOverviewRaw(ctx, filter)
	}
}

func (r *opsRepository) getDashboardOverviewRaw(ctx context.Context, filter *service.OpsDashboardFilter) (*service.OpsDashboardOverview, error) {
	start := filter.StartTime.UTC()
	end := filter.EndTime.UTC()
	degraded := false

	successCount, tokenConsumed, err := r.queryUsageCounts(ctx, filter, start, end)
	if err != nil {
		return nil, err
	}

	latencyCtx, cancelLatency := context.WithTimeout(ctx, opsRawLatencyQueryTimeout)
	duration, ttft, err := r.queryUsageLatency(latencyCtx, filter, start, end)
	cancelLatency()
	if err != nil {
		if isQueryTimeoutErr(err) {
			degraded = true
			duration = service.OpsPercentiles{}
			ttft = service.OpsPercentiles{}
		} else {
			return nil, err
		}
	}

	errorTotal, businessLimited, errorCountSLA, upstreamExcl, upstream429, upstream529, err := r.queryErrorCounts(ctx, filter, start, end)
	if err != nil {
		return nil, err
	}

	windowSeconds := end.Sub(start).Seconds()
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	requestCountTotal := successCount + errorTotal
	requestCountSLA := successCount + errorCountSLA

	sla := safeDivideFloat64(float64(successCount), float64(requestCountSLA))
	errorRate := safeDivideFloat64(float64(errorCountSLA), float64(requestCountSLA))
	upstreamErrorRate := safeDivideFloat64(float64(upstreamExcl), float64(requestCountSLA))

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
