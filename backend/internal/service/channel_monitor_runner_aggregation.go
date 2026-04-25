package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type rollupKey struct {
	MonitorID int64
	ModelID   string
	Day       time.Time
}

type rollupDelta struct {
	Total      int64
	Available  int64
	Degraded   int64
	Latency    int64
	MaxLatency int64
}

func (s *ChannelMonitorRunnerService) aggregateOnce(ctx context.Context) {
	if s.aggRepo == nil || s.historyRepo == nil || s.rollupRepo == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	for i := 0; i < 5; i++ {
		watermark, err := s.aggRepo.GetWatermark(ctx)
		if err != nil {
			return
		}
		histories, err := s.historyRepo.ListForAggregation(ctx, watermark, 1000)
		if err != nil || len(histories) == 0 {
			return
		}

		maxID := watermark
		deltas := map[rollupKey]*rollupDelta{}

		for _, h := range histories {
			if h == nil {
				continue
			}
			if h.ID > maxID {
				maxID = h.ID
			}
			day := utcDayStart(h.StartedAt)
			k := rollupKey{MonitorID: h.MonitorID, ModelID: h.ModelID, Day: day}
			d := deltas[k]
			if d == nil {
				d = &rollupDelta{}
				deltas[k] = d
			}

			d.Total++
			if h.Status != ChannelMonitorStatusFailure {
				d.Available++
				d.Latency += h.LatencyMs
				if h.LatencyMs > d.MaxLatency {
					d.MaxLatency = h.LatencyMs
				}
			}
			if h.Status == ChannelMonitorStatusDegraded {
				d.Degraded++
			}
		}

		for k, d := range deltas {
			if d == nil || d.Total == 0 {
				continue
			}
			if err := s.rollupRepo.UpsertIncrement(ctx, k.MonitorID, k.ModelID, k.Day, d.Total, d.Available, d.Degraded, d.Latency, d.MaxLatency); err != nil {
				logger.LegacyPrintf("service.channel_monitor", "[ChannelMonitorRunner] rollup upsert failed: monitor_id=%d model=%s err=%v", k.MonitorID, k.ModelID, err)
				return
			}
		}

		if maxID > watermark {
			if err := s.aggRepo.SetWatermark(ctx, maxID); err != nil {
				return
			}
		}
		if len(histories) < 1000 {
			return
		}
	}
}

func (s *ChannelMonitorRunnerService) pruneOnce(ctx context.Context) {
	if s.historyRepo == nil || s.rollupRepo == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -channelMonitorHistoryKeepDays)
	if _, err := s.historyRepo.PruneBefore(ctx, cutoff); err != nil {
		return
	}
	if _, err := s.rollupRepo.PruneBeforeDay(ctx, utcDayStart(cutoff)); err != nil {
		return
	}
}
