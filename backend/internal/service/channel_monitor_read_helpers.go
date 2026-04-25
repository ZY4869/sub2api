package service

func indexLatestHistories(items []*ChannelMonitorHistory) map[int64]map[string]*ChannelMonitorHistory {
	out := map[int64]map[string]*ChannelMonitorHistory{}
	for _, h := range items {
		if h == nil || h.MonitorID <= 0 || h.ModelID == "" {
			continue
		}
		if out[h.MonitorID] == nil {
			out[h.MonitorID] = map[string]*ChannelMonitorHistory{}
		}
		out[h.MonitorID][h.ModelID] = h
	}
	return out
}

func indexTimeline(items []*ChannelMonitorHistory) map[int64][]ChannelMonitorTimelineItem {
	out := map[int64][]ChannelMonitorTimelineItem{}
	for _, h := range items {
		if h == nil || h.MonitorID <= 0 {
			continue
		}
		out[h.MonitorID] = append(out[h.MonitorID], ChannelMonitorTimelineItem{
			Status:    h.Status,
			LatencyMs: h.LatencyMs,
			CheckedAt: h.CreatedAt,
		})
	}
	return out
}

func toModelLastStatus(modelID string, h *ChannelMonitorHistory) *ChannelMonitorModelLastStatus {
	if h == nil {
		return nil
	}
	checkedAt := h.CreatedAt
	return &ChannelMonitorModelLastStatus{
		ModelID:    modelID,
		Status:     h.Status,
		LatencyMs:  h.LatencyMs,
		CheckedAt:  &checkedAt,
		HTTPStatus: h.HTTPStatus,
	}
}

func availabilityRate(available int64, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	v := float64(available) / float64(total)
	return &v
}
