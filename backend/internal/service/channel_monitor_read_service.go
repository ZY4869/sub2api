package service

import (
	"context"
	"errors"
	"time"
)

func (s *ChannelMonitorService) ListHistory(ctx context.Context, monitorID int64, limit int) ([]*ChannelMonitorHistory, error) {
	if s.historyRepo == nil {
		return nil, errors.New("history repo not configured")
	}
	if _, err := s.repo.GetByID(ctx, monitorID); err != nil {
		return nil, err
	}
	return s.historyRepo.ListByMonitorID(ctx, monitorID, limit)
}

func (s *ChannelMonitorService) ListUserView(ctx context.Context) ([]*ChannelMonitorUserListItem, error) {
	if !channelMonitorRequireEnabled(ctx, s.settingSvc) {
		return []*ChannelMonitorUserListItem{}, nil
	}

	monitors, err := s.repo.ListEnabled(ctx)
	if err != nil || len(monitors) == 0 {
		return []*ChannelMonitorUserListItem{}, err
	}

	ids := make([]int64, 0, len(monitors))
	for _, m := range monitors {
		if m == nil {
			continue
		}
		ids = append(ids, m.ID)
	}

	latest, err := s.historyRepo.ListLatestByMonitorIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	timeline, err := s.historyRepo.ListPrimaryTimelineByMonitorIDs(ctx, ids, 20)
	if err != nil {
		return nil, err
	}
	rollups, err := s.rollupRepo.SumAvailability(ctx, ids, channelMonitorStartDay(time.Now(), 7))
	if err != nil {
		return nil, err
	}

	latestIndex := indexLatestHistories(latest)
	timelineIndex := indexTimeline(timeline)

	out := make([]*ChannelMonitorUserListItem, 0, len(monitors))
	for _, m := range monitors {
		if m == nil {
			continue
		}
		item := &ChannelMonitorUserListItem{
			ID:                    m.ID,
			Name:                  m.Name,
			Provider:              m.Provider,
			PrimaryModelID:        m.PrimaryModelID,
			Timeline:              timelineIndex[m.ID],
			AdditionalLast:        []ChannelMonitorModelLastStatus{},
			PrimaryAvailability7d: nil,
		}

		if last := latestIndex[m.ID][m.PrimaryModelID]; last != nil {
			item.PrimaryLast = toModelLastStatus(m.PrimaryModelID, last)
		}
		if sum := rollups[m.ID][m.PrimaryModelID]; sum != nil {
			item.PrimaryAvailability7d = availabilityRate(sum.AvailableChecks, sum.TotalChecks)
		}

		for _, modelID := range m.AdditionalModelIDs {
			if last := latestIndex[m.ID][modelID]; last != nil {
				item.AdditionalLast = append(item.AdditionalLast, *toModelLastStatus(modelID, last))
			}
		}

		out = append(out, item)
	}
	return out, nil
}

func (s *ChannelMonitorService) GetUserDetail(ctx context.Context, monitorID int64) (*ChannelMonitorUserDetail, error) {
	if !channelMonitorRequireEnabled(ctx, s.settingSvc) {
		return nil, ErrChannelMonitorNotFound
	}
	monitor, err := s.repo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	if !monitor.Enabled {
		return nil, ErrChannelMonitorNotFound
	}

	latest, err := s.historyRepo.ListLatestByMonitorID(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	latestByModel := map[string]*ChannelMonitorHistory{}
	for _, h := range latest {
		if h == nil || h.ModelID == "" {
			continue
		}
		latestByModel[h.ModelID] = h
	}

	now := time.Now()
	start7 := channelMonitorStartDay(now, 7)
	start15 := channelMonitorStartDay(now, 15)
	start30 := channelMonitorStartDay(now, 30)
	windows, err := s.rollupRepo.SumAvailabilityWindows(ctx, monitorID, start7, start15, start30)
	if err != nil {
		return nil, err
	}

	modelIDs := append([]string{monitor.PrimaryModelID}, monitor.AdditionalModelIDs...)
	modelIDs = dedupeNonEmptyStrings(modelIDs)

	out := &ChannelMonitorUserDetail{
		ID:             monitor.ID,
		Name:           monitor.Name,
		Provider:       monitor.Provider,
		PrimaryModelID: monitor.PrimaryModelID,
		Models:         make([]ChannelMonitorUserModelDetail, 0, len(modelIDs)),
	}

	for _, modelID := range modelIDs {
		d := ChannelMonitorUserModelDetail{ModelID: modelID}
		if last := latestByModel[modelID]; last != nil {
			d.Last = toModelLastStatus(modelID, last)
		}
		if w := windows[modelID]; w != nil {
			d.Availability7d = availabilityRate(w.Last7.AvailableChecks, w.Last7.TotalChecks)
			d.Availability15d = availabilityRate(w.Last15.AvailableChecks, w.Last15.TotalChecks)
			d.Availability30d = availabilityRate(w.Last30.AvailableChecks, w.Last30.TotalChecks)
		}
		out.Models = append(out.Models, d)
	}
	return out, nil
}
