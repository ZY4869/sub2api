package repository

import "math"

func aggregateHourlyRows(rows []opsHourlyMetricsRow) opsDashboardPartial {
	out := opsDashboardPartial{}
	if len(rows) == 0 {
		return out
	}

	var (
		p50Sum float64
		p50W   int64
		p90Sum float64
		p90W   int64
		avgSum float64
		avgW   int64
	)
	var (
		ttftP50Sum float64
		ttftP50W   int64
		ttftP90Sum float64
		ttftP90W   int64
		ttftAvgSum float64
		ttftAvgW   int64
	)

	var (
		p95Max *int
		p99Max *int
		maxMax *int

		ttftP95Max *int
		ttftP99Max *int
		ttftMaxMax *int
	)

	for _, row := range rows {
		out.successCount += row.successCount
		out.errorCountTotal += row.errorCountTotal
		out.businessLimitedCount += row.businessLimitedCount
		out.errorCountSLA += row.errorCountSLA

		out.upstreamErrorCountExcl429529 += row.upstreamErrorCountExcl429529
		out.upstream429Count += row.upstream429Count
		out.upstream529Count += row.upstream529Count

		out.tokenConsumed += row.tokenConsumed
		out.ttftSampleCount += row.ttftSampleCount

		if row.successCount > 0 {
			if row.durationP50.Valid {
				p50Sum += float64(row.durationP50.Int64) * float64(row.successCount)
				p50W += row.successCount
			}
			if row.durationP90.Valid {
				p90Sum += float64(row.durationP90.Int64) * float64(row.successCount)
				p90W += row.successCount
			}
			if row.durationAvg.Valid {
				avgSum += row.durationAvg.Float64 * float64(row.successCount)
				avgW += row.successCount
			}
		}
		if row.ttftSampleCount > 0 {
			if row.ttftP50.Valid {
				ttftP50Sum += float64(row.ttftP50.Int64) * float64(row.ttftSampleCount)
				ttftP50W += row.ttftSampleCount
			}
			if row.ttftP90.Valid {
				ttftP90Sum += float64(row.ttftP90.Int64) * float64(row.ttftSampleCount)
				ttftP90W += row.ttftSampleCount
			}
			if row.ttftAvg.Valid {
				ttftAvgSum += row.ttftAvg.Float64 * float64(row.ttftSampleCount)
				ttftAvgW += row.ttftSampleCount
			}
		}

		if row.durationP95.Valid {
			v := int(row.durationP95.Int64)
			if p95Max == nil || v > *p95Max {
				p95Max = &v
			}
		}
		if row.durationP99.Valid {
			v := int(row.durationP99.Int64)
			if p99Max == nil || v > *p99Max {
				p99Max = &v
			}
		}
		if row.durationMax.Valid {
			v := int(row.durationMax.Int64)
			if maxMax == nil || v > *maxMax {
				maxMax = &v
			}
		}

		if row.ttftP95.Valid {
			v := int(row.ttftP95.Int64)
			if ttftP95Max == nil || v > *ttftP95Max {
				ttftP95Max = &v
			}
		}
		if row.ttftP99.Valid {
			v := int(row.ttftP99.Int64)
			if ttftP99Max == nil || v > *ttftP99Max {
				ttftP99Max = &v
			}
		}
		if row.ttftMax.Valid {
			v := int(row.ttftMax.Int64)
			if ttftMaxMax == nil || v > *ttftMaxMax {
				ttftMaxMax = &v
			}
		}
	}

	if p50W > 0 {
		v := int(math.Round(p50Sum / float64(p50W)))
		out.duration.P50 = &v
	}
	if p90W > 0 {
		v := int(math.Round(p90Sum / float64(p90W)))
		out.duration.P90 = &v
	}
	out.duration.P95 = p95Max
	out.duration.P99 = p99Max
	if avgW > 0 {
		v := int(math.Round(avgSum / float64(avgW)))
		out.duration.Avg = &v
	}
	out.duration.Max = maxMax

	if ttftP50W > 0 {
		v := int(math.Round(ttftP50Sum / float64(ttftP50W)))
		out.ttft.P50 = &v
	}
	if ttftP90W > 0 {
		v := int(math.Round(ttftP90Sum / float64(ttftP90W)))
		out.ttft.P90 = &v
	}
	out.ttft.P95 = ttftP95Max
	out.ttft.P99 = ttftP99Max
	if ttftAvgW > 0 {
		v := int(math.Round(ttftAvgSum / float64(ttftAvgW)))
		out.ttft.Avg = &v
	}
	out.ttft.Max = ttftMaxMax

	return out
}
