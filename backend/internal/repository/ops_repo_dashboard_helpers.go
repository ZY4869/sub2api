package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsPercentileSegment struct {
	weight int64
	p      service.OpsPercentiles
}

func combineApproxPercentiles(segments []opsPercentileSegment) service.OpsPercentiles {
	weightedInt := func(get func(service.OpsPercentiles) *int) *int {
		var sum float64
		var w int64
		for _, seg := range segments {
			if seg.weight <= 0 {
				continue
			}
			v := get(seg.p)
			if v == nil {
				continue
			}
			sum += float64(*v) * float64(seg.weight)
			w += seg.weight
		}
		if w <= 0 {
			return nil
		}
		out := int(math.Round(sum / float64(w)))
		return &out
	}

	maxInt := func(get func(service.OpsPercentiles) *int) *int {
		var max *int
		for _, seg := range segments {
			v := get(seg.p)
			if v == nil {
				continue
			}
			if max == nil || *v > *max {
				c := *v
				max = &c
			}
		}
		return max
	}

	return service.OpsPercentiles{
		P50: weightedInt(func(p service.OpsPercentiles) *int { return p.P50 }),
		P90: weightedInt(func(p service.OpsPercentiles) *int { return p.P90 }),
		P95: maxInt(func(p service.OpsPercentiles) *int { return p.P95 }),
		P99: maxInt(func(p service.OpsPercentiles) *int { return p.P99 }),
		Avg: weightedInt(func(p service.OpsPercentiles) *int { return p.Avg }),
		Max: maxInt(func(p service.OpsPercentiles) *int { return p.Max }),
	}
}

func preaggSafeEnd(endTime time.Time) time.Time {
	now := time.Now().UTC()
	cutoff := now.Add(-5 * time.Minute)
	if endTime.After(cutoff) {
		return cutoff
	}
	return endTime
}

func utcCeilToHour(t time.Time) time.Time {
	u := t.UTC()
	f := u.Truncate(time.Hour)
	if f.Equal(u) {
		return f
	}
	return f.Add(time.Hour)
}

func utcFloorToHour(t time.Time) time.Time {
	return t.UTC().Truncate(time.Hour)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func isQueryTimeoutErr(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

func floatToIntPtr(v sql.NullFloat64) *int {
	if !v.Valid {
		return nil
	}
	n := int(math.Round(v.Float64))
	return &n
}

func safeDivideFloat64(numerator float64, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func roundTo1DP(v float64) float64 {
	return math.Round(v*10) / 10
}

func roundTo4DP(v float64) float64 {
	return math.Round(v*10000) / 10000
}
