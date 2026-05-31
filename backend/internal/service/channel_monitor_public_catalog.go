package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

const publicModelCatalogProbeHistoryTTL = 30 * time.Minute

type publicCatalogHealthAggregate struct {
	Model     string
	Aliases   map[string]struct{}
	Latest    []*ChannelMonitorHistory
	Daily     map[string]*ChannelMonitorDailyRollup
	Seven     ChannelMonitorDailyRollup
	Today     ChannelMonitorDailyRollup
	UpdatedAt time.Time
	Enabled   bool
	HasConfig bool
}

func (s *ChannelMonitorService) PublicModelCatalogHealth(ctx context.Context, items []PublicModelCatalogItem) (map[string]PublicModelCatalogStatusItem, error) {
	out := make(map[string]PublicModelCatalogStatusItem, len(items))
	aliasIndex := buildPublicModelCatalogHealthAliasIndex(items)
	if len(aliasIndex) == 0 {
		return out, nil
	}
	for _, modelID := range publicModelCatalogHealthModels(items) {
		out[modelID] = pendingPublicModelCatalogStatusWithReason(modelID, nil, PublicModelHealthReasonChecking)
	}
	if s == nil {
		for modelID := range out {
			out[modelID] = pendingPublicModelCatalogStatusWithReason(modelID, nil, PublicModelHealthReasonMonitorDisabled)
		}
		return out, nil
	}
	enabled := true
	if s.settingSvc != nil {
		enabled = channelMonitorRequireEnabled(ctx, s.settingSvc)
	}
	if s.repo == nil || s.historyRepo == nil || s.rollupRepo == nil || !enabled {
		for modelID := range out {
			out[modelID] = pendingPublicModelCatalogStatusWithReason(modelID, nil, PublicModelHealthReasonMonitorDisabled)
		}
		return out, nil
	}
	monitors, err := s.repo.ListEnabled(ctx)
	if err != nil || len(monitors) == 0 {
		if err != nil && errors.Is(err, ErrChannelMonitorNotFound) {
			for modelID := range out {
				out[modelID] = pendingPublicModelCatalogStatusWithReason(modelID, nil, PublicModelHealthReasonNoHistory)
			}
			return out, nil
		}
		return out, err
	}
	monitorIDs := make([]int64, 0, len(monitors))
	for _, monitor := range monitors {
		if monitor == nil || monitor.ID <= 0 {
			continue
		}
		monitorIDs = append(monitorIDs, monitor.ID)
	}
	if len(monitorIDs) == 0 {
		for modelID := range out {
			out[modelID] = pendingPublicModelCatalogStatusWithReason(modelID, nil, PublicModelHealthReasonNoHistory)
		}
		return out, nil
	}

	latest, err := s.historyRepo.ListLatestByMonitorIDs(ctx, monitorIDs)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	start7 := channelMonitorStartDay(now, 7)
	daily, err := s.rollupRepo.ListDailyByMonitorIDs(ctx, monitorIDs, start7)
	if err != nil {
		return nil, err
	}

	aggregates := make(map[string]*publicCatalogHealthAggregate, len(out))
	for modelID := range out {
		aggregates[modelID] = &publicCatalogHealthAggregate{
			Model:     modelID,
			Aliases:   map[string]struct{}{},
			Daily:     map[string]*ChannelMonitorDailyRollup{},
			Enabled:   enabled,
			HasConfig: len(monitorIDs) > 0,
		}
	}
	for alias, modelID := range aliasIndex {
		if aggregate := aggregates[modelID]; aggregate != nil {
			aggregate.Aliases[alias] = struct{}{}
		}
	}

	for _, history := range latest {
		if history == nil {
			continue
		}
		modelID, ok := aliasIndex[NormalizeModelCatalogModelID(history.ModelID)]
		if !ok {
			continue
		}
		aggregate := aggregates[modelID]
		if aggregate == nil {
			continue
		}
		aggregate.Latest = append(aggregate.Latest, history)
		if history.CreatedAt.After(aggregate.UpdatedAt) {
			aggregate.UpdatedAt = history.CreatedAt
		}
	}

	todayKey := utcDayStart(now).Format("2006-01-02")
	for _, rollup := range daily {
		if rollup == nil {
			continue
		}
		modelID, ok := aliasIndex[NormalizeModelCatalogModelID(rollup.ModelID)]
		if !ok {
			continue
		}
		aggregate := aggregates[modelID]
		if aggregate == nil {
			continue
		}
		dayKey := utcDayStart(rollup.Day).Format("2006-01-02")
		target := aggregate.Daily[dayKey]
		if target == nil {
			target = &ChannelMonitorDailyRollup{Day: utcDayStart(rollup.Day)}
			aggregate.Daily[dayKey] = target
		}
		addChannelMonitorRollup(target, rollup)
		addChannelMonitorRollup(&aggregate.Seven, rollup)
		if dayKey == todayKey {
			addChannelMonitorRollup(&aggregate.Today, rollup)
		}
	}

	for modelID, aggregate := range aggregates {
		out[modelID] = buildPublicModelCatalogStatusFromAggregate(aggregate, start7, now)
	}
	return out, nil
}

func buildPublicModelCatalogHealthAliasIndex(items []PublicModelCatalogItem) map[string]string {
	index := map[string]string{}
	for _, item := range items {
		modelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if modelID == "" {
			continue
		}
		for _, alias := range publicModelCatalogHealthAliases(item) {
			if alias == "" {
				continue
			}
			index[alias] = modelID
		}
	}
	return index
}

func publicModelCatalogHealthModels(items []PublicModelCatalogItem) []string {
	seen := map[string]struct{}{}
	models := make([]string, 0, len(items))
	for _, item := range items {
		modelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		models = append(models, modelID)
	}
	sort.Strings(models)
	return models
}

func publicModelCatalogHealthAliases(item PublicModelCatalogItem) []string {
	aliases := []string{
		item.PublicModelID,
		item.Model,
		item.BaseModel,
		item.SourceModelID,
	}
	aliases = append(aliases, item.SourceIDs...)
	normalized := make([]string, 0, len(aliases))
	seen := map[string]struct{}{}
	for _, alias := range aliases {
		id := NormalizeModelCatalogModelID(alias)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}

func addChannelMonitorRollup(target *ChannelMonitorDailyRollup, delta *ChannelMonitorDailyRollup) {
	if target == nil || delta == nil {
		return
	}
	if target.Day.IsZero() {
		target.Day = utcDayStart(delta.Day)
	}
	target.TotalChecks += delta.TotalChecks
	target.AvailableChecks += delta.AvailableChecks
	target.DegradedChecks += delta.DegradedChecks
	target.TotalLatencyMs += delta.TotalLatencyMs
	if delta.MaxLatencyMs > target.MaxLatencyMs {
		target.MaxLatencyMs = delta.MaxLatencyMs
	}
}

func buildPublicModelCatalogStatusFromAggregate(
	aggregate *publicCatalogHealthAggregate,
	startDay time.Time,
	now time.Time,
) PublicModelCatalogStatusItem {
	if aggregate == nil || aggregate.Model == "" {
		return pendingPublicModelCatalogStatus("")
	}
	item := pendingPublicModelCatalogStatus(aggregate.Model)
	item.HealthSource = PublicModelHealthSourceProbe
	item.StatusReason = PublicModelHealthReasonProbeRecent
	item.SuccessRateToday = availabilityRate(aggregate.Today.AvailableChecks, aggregate.Today.TotalChecks)
	item.SuccessRate7d = availabilityRate(aggregate.Seven.AvailableChecks, aggregate.Seven.TotalChecks)
	item.LatencyMs = publicModelCatalogLatestLatency(aggregate.Latest)
	if !aggregate.UpdatedAt.IsZero() {
		item.LastCheckedAt = aggregate.UpdatedAt.UTC().Format(time.RFC3339)
	}
	item.Daily = buildPublicModelCatalogDailyStatuses(aggregate, startDay, now)
	item.Trend = buildPublicModelCatalogTrend(item.Daily)
	item.Status = publicModelCatalogHealthStatus(aggregate, item.SuccessRateToday, item.SuccessRate7d)
	item.HealthStatus = item.Status
	if publicModelCatalogProbeHistoryStale(aggregate, now) {
		return stalePublicModelCatalogProbeStatus(item)
	}
	if !statusHasHealthHistory(item) {
		item.HealthSource = PublicModelHealthSourceNone
		item.StatusReason = publicModelCatalogProbePendingReason(aggregate)
	}
	return item
}

func publicModelCatalogProbeHistoryStale(aggregate *publicCatalogHealthAggregate, now time.Time) bool {
	if aggregate == nil || aggregate.UpdatedAt.IsZero() {
		return false
	}
	return now.UTC().Sub(aggregate.UpdatedAt.UTC()) > publicModelCatalogProbeHistoryTTL
}

func stalePublicModelCatalogProbeStatus(item PublicModelCatalogStatusItem) PublicModelCatalogStatusItem {
	return PublicModelCatalogStatusItem{
		PublicModelID: item.PublicModelID,
		Model:         item.Model,
		Aliases:       clonePublicModelStatusAliases(item.Aliases),
		Status:        PublicModelHealthStatusPending,
		HealthStatus:  PublicModelHealthStatusPending,
		HealthSource:  PublicModelHealthSourceNone,
		StatusReason:  PublicModelHealthReasonStaleHistory,
		LastCheckedAt: item.LastCheckedAt,
		RateLimit:     clonePublicModelCatalogRateLimitSummary(item.RateLimit),
		Daily:         []PublicModelCatalogDailyStatus{},
		Trend:         []PublicModelCatalogTrendPoint{},
	}
}

func publicModelCatalogProbePendingReason(aggregate *publicCatalogHealthAggregate) string {
	if aggregate == nil || !aggregate.Enabled {
		return PublicModelHealthReasonMonitorDisabled
	}
	if !aggregate.HasConfig {
		return PublicModelHealthReasonNoHistory
	}
	if aggregate.UpdatedAt.IsZero() {
		return PublicModelHealthReasonChecking
	}
	return PublicModelHealthReasonStaleHistory
}

func publicModelCatalogLatestLatency(items []*ChannelMonitorHistory) *int64 {
	var (
		latest *ChannelMonitorHistory
		sum    int64
		count  int64
	)
	for _, item := range items {
		if item == nil || item.LatencyMs <= 0 {
			continue
		}
		if latest == nil || item.CreatedAt.After(latest.CreatedAt) {
			latest = item
		}
		sum += item.LatencyMs
		count++
	}
	if latest != nil {
		value := latest.LatencyMs
		return &value
	}
	if count == 0 {
		return nil
	}
	value := sum / count
	return &value
}

func buildPublicModelCatalogDailyStatuses(
	aggregate *publicCatalogHealthAggregate,
	startDay time.Time,
	now time.Time,
) []PublicModelCatalogDailyStatus {
	start := utcDayStart(startDay)
	end := utcDayStart(now)
	days := make([]PublicModelCatalogDailyStatus, 0, 7)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		rollup := aggregate.Daily[key]
		status := PublicModelHealthStatusPending
		var rate *float64
		var latency *int64
		if rollup != nil && rollup.TotalChecks > 0 {
			rate = availabilityRate(rollup.AvailableChecks, rollup.TotalChecks)
			status = publicModelCatalogHealthStatusForRollup(rollup)
			if rollup.AvailableChecks > 0 {
				value := rollup.TotalLatencyMs / rollup.AvailableChecks
				latency = &value
			}
		}
		days = append(days, PublicModelCatalogDailyStatus{
			Date:        key,
			Status:      status,
			SuccessRate: rate,
			LatencyMs:   latency,
		})
	}
	return days
}

func buildPublicModelCatalogTrend(days []PublicModelCatalogDailyStatus) []PublicModelCatalogTrendPoint {
	trend := make([]PublicModelCatalogTrendPoint, 0, len(days))
	for _, day := range days {
		if day.SuccessRate == nil && day.LatencyMs == nil {
			continue
		}
		trend = append(trend, PublicModelCatalogTrendPoint{
			Timestamp:   day.Date,
			SuccessRate: day.SuccessRate,
			LatencyMs:   day.LatencyMs,
		})
	}
	return trend
}

func publicModelCatalogHealthStatus(
	aggregate *publicCatalogHealthAggregate,
	today *float64,
	week *float64,
) string {
	if aggregate == nil || (len(aggregate.Latest) == 0 && aggregate.Seven.TotalChecks == 0) {
		return PublicModelHealthStatusPending
	}
	hasSuccess := false
	hasDegraded := false
	hasFailure := false
	for _, latest := range aggregate.Latest {
		switch strings.TrimSpace(latest.Status) {
		case ChannelMonitorStatusSuccess:
			hasSuccess = true
		case ChannelMonitorStatusDegraded:
			hasDegraded = true
		case ChannelMonitorStatusFailure:
			hasFailure = true
		}
	}
	if hasFailure && !hasSuccess {
		return PublicModelHealthStatusError
	}
	if hasDegraded || (today != nil && *today < 0.95) || (week != nil && *week < 0.98) {
		return PublicModelHealthStatusWarning
	}
	return PublicModelHealthStatusHealthy
}

func publicModelCatalogHealthStatusForRollup(rollup *ChannelMonitorDailyRollup) string {
	if rollup == nil || rollup.TotalChecks == 0 {
		return PublicModelHealthStatusPending
	}
	if rollup.AvailableChecks == 0 {
		return PublicModelHealthStatusError
	}
	rate := float64(rollup.AvailableChecks) / float64(rollup.TotalChecks)
	if rollup.DegradedChecks > 0 || rate < 0.98 {
		return PublicModelHealthStatusWarning
	}
	return PublicModelHealthStatusHealthy
}
