package repository

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const publicModelCatalogTrafficStaleLookbackDays = 30

type publicModelCatalogTrafficCandidate struct {
	PublicModelID string
	AccountID     int64
	Protocol      string
	PublicAliases []string
	SourceAliases []string
}

type publicModelCatalogTrafficDaily struct {
	Date         string
	Total        int64
	Success      int64
	LatencyTotal int64
	LatencyCount int64
}

type publicModelCatalogTrafficAggregate struct {
	PublicModelID string
	LatestAt      time.Time
	LatestLatency sql.NullInt64
	Days          map[string]*publicModelCatalogTrafficDaily
	SevenTotal    int64
	SevenSuccess  int64
	TodayTotal    int64
	TodaySuccess  int64
}

func (r *usageLogRepository) PublicModelCatalogTrafficHealth(
	ctx context.Context,
	items []service.PublicModelCatalogItem,
	start time.Time,
	end time.Time,
) (map[string]service.PublicModelCatalogStatusItem, error) {
	out := make(map[string]service.PublicModelCatalogStatusItem, len(items))
	candidates := buildPublicModelCatalogTrafficCandidates(items)
	for _, candidate := range candidates {
		out[candidate.PublicModelID] = service.PublicModelCatalogStatusItem{
			PublicModelID: candidate.PublicModelID,
			Model:         candidate.PublicModelID,
			Aliases:       append([]string(nil), candidate.PublicAliases...),
			Status:        service.PublicModelHealthStatusPending,
			HealthStatus:  service.PublicModelHealthStatusPending,
			HealthSource:  service.PublicModelHealthSourceNone,
			StatusReason:  service.PublicModelHealthReasonNoHistory,
			Daily:         []service.PublicModelCatalogDailyStatus{},
			Trend:         []service.PublicModelCatalogTrendPoint{},
		}
	}
	if r == nil || r.sql == nil || len(candidates) == 0 {
		return out, nil
	}
	if start.IsZero() {
		start = time.Now().UTC().AddDate(0, 0, -6)
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}
	aggregates, err := r.queryPublicModelCatalogTrafficAggregates(ctx, candidates, start.UTC(), end.UTC())
	if err != nil {
		return nil, err
	}
	todayKey := utcDayString(end.UTC())
	for _, candidate := range candidates {
		aggregate := aggregates[candidate.PublicModelID]
		if aggregate == nil || aggregate.SevenTotal == 0 {
			continue
		}
		out[candidate.PublicModelID] = buildPublicModelCatalogTrafficStatus(candidate, aggregate, start.UTC(), end.UTC(), todayKey)
	}
	staleCandidates := publicModelCatalogTrafficStaleCandidates(candidates, out)
	if len(staleCandidates) == 0 {
		return out, nil
	}
	staleStart := end.UTC().AddDate(0, 0, -publicModelCatalogTrafficStaleLookbackDays)
	if staleStart.After(start.UTC()) {
		return out, nil
	}
	staleEnd := start.UTC().Add(-time.Nanosecond)
	staleAggregates, err := r.queryPublicModelCatalogTrafficAggregates(ctx, staleCandidates, staleStart, staleEnd)
	if err != nil {
		return nil, err
	}
	for _, candidate := range staleCandidates {
		aggregate := staleAggregates[candidate.PublicModelID]
		if aggregate == nil || aggregate.SevenTotal == 0 || aggregate.LatestAt.IsZero() {
			continue
		}
		out[candidate.PublicModelID] = buildPublicModelCatalogStaleTrafficStatus(candidate, aggregate.LatestAt)
	}
	return out, nil
}

func publicModelCatalogTrafficStaleCandidates(
	candidates []publicModelCatalogTrafficCandidate,
	statuses map[string]service.PublicModelCatalogStatusItem,
) []publicModelCatalogTrafficCandidate {
	out := make([]publicModelCatalogTrafficCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		status := statuses[candidate.PublicModelID]
		if status.HealthSource == service.PublicModelHealthSourceTraffic {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func buildPublicModelCatalogTrafficCandidates(items []service.PublicModelCatalogItem) []publicModelCatalogTrafficCandidate {
	seen := map[string]struct{}{}
	out := make([]publicModelCatalogTrafficCandidate, 0, len(items))
	for _, item := range items {
		publicModelID := service.NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if publicModelID == "" {
			continue
		}
		if _, ok := seen[publicModelID]; ok {
			continue
		}
		seen[publicModelID] = struct{}{}
		publicAliases, sourceAliases := publicModelCatalogTrafficAliases(item)
		out = append(out, publicModelCatalogTrafficCandidate{
			PublicModelID: publicModelID,
			AccountID:     item.SourceAccountID,
			Protocol:      publicModelCatalogTrafficProtocol(item.SourceProtocol),
			PublicAliases: publicAliases,
			SourceAliases: sourceAliases,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].PublicModelID < out[j].PublicModelID
	})
	return out
}

func publicModelCatalogTrafficAliases(item service.PublicModelCatalogItem) ([]string, []string) {
	publicModelID := service.NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	publicAliases := normalizePublicModelCatalogTrafficAliasSet([]string{publicModelID})
	sourceValues := append([]string{
		item.BaseModel,
		item.SourceModelID,
	}, item.SourceIDs...)
	if service.NormalizeModelCatalogModelID(item.PublicModelID) != "" &&
		service.NormalizeModelCatalogModelID(item.Model) != publicModelID {
		sourceValues = append(sourceValues, item.Model)
	}
	sourceAliases := normalizePublicModelCatalogTrafficAliasSet(sourceValues)
	return publicAliases, sourceAliases
}

func normalizePublicModelCatalogTrafficAliasSet(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		alias := service.NormalizeModelCatalogModelID(value)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

func publicModelCatalogTrafficProtocol(protocol string) string {
	return strings.TrimSpace(strings.ToLower(protocol))
}

func (r *usageLogRepository) queryPublicModelCatalogTrafficAggregates(
	ctx context.Context,
	candidates []publicModelCatalogTrafficCandidate,
	start time.Time,
	end time.Time,
) (map[string]*publicModelCatalogTrafficAggregate, error) {
	out := make(map[string]*publicModelCatalogTrafficAggregate, len(candidates))
	publicModelToCandidates := map[string][]publicModelCatalogTrafficCandidate{}
	sourceModelToCandidates := map[string][]publicModelCatalogTrafficCandidate{}
	modelSet := map[string]struct{}{}
	accountSet := map[int64]struct{}{}
	for _, candidate := range candidates {
		out[candidate.PublicModelID] = &publicModelCatalogTrafficAggregate{
			PublicModelID: candidate.PublicModelID,
			Days:          map[string]*publicModelCatalogTrafficDaily{},
		}
		if candidate.AccountID > 0 {
			accountSet[candidate.AccountID] = struct{}{}
		}
		for _, alias := range candidate.PublicAliases {
			modelSet[alias] = struct{}{}
			publicModelToCandidates[alias] = append(publicModelToCandidates[alias], candidate)
		}
		for _, alias := range candidate.SourceAliases {
			modelSet[alias] = struct{}{}
			sourceModelToCandidates[alias] = append(sourceModelToCandidates[alias], candidate)
		}
	}
	models := sortedStringKeys(modelSet)
	accountIDs := sortedInt64Keys(accountSet)
	if len(models) == 0 {
		return out, nil
	}

	query := `
		SELECT
			account_id,
			COALESCE(NULLIF(TRIM(requested_model), ''), model) AS requested_model,
			COALESCE(NULLIF(TRIM(upstream_model), ''), COALESCE(NULLIF(TRIM(requested_model), ''), model)) AS upstream_model,
			COALESCE(NULLIF(TRIM(upstream_service), ''), NULLIF(TRIM(inbound_endpoint), '')) AS protocol,
			status,
			duration_ms,
			created_at
		FROM usage_logs
		WHERE created_at >= $1
		  AND created_at <= $2
		  AND COALESCE(operation_type, '') NOT IN ($3, $4, $5, $6)
		  AND (
		    COALESCE(NULLIF(TRIM(requested_model), ''), model) = ANY($7)
		    OR COALESCE(NULLIF(TRIM(upstream_model), ''), COALESCE(NULLIF(TRIM(requested_model), ''), model)) = ANY($7)
		  )
	`
	args := []any{
		start,
		end,
		service.UsageOperationTypeAccountTest,
		service.UsageOperationTypeBatchTest,
		service.UsageOperationTypeScheduledTest,
		service.UsageOperationTypeAutoRecoveryTest,
		pq.Array(models),
	}
	if len(accountIDs) > 0 {
		query += " AND (account_id = ANY($8) OR COALESCE(NULLIF(TRIM(requested_model), ''), model) = ANY($7))"
		args = append(args, pq.Array(accountIDs))
	}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var (
			accountID      int64
			requestedModel string
			upstreamModel  string
			protocol       sql.NullString
			status         string
			duration       sql.NullInt64
			createdAt      time.Time
		)
		if err := rows.Scan(&accountID, &requestedModel, &upstreamModel, &protocol, &status, &duration, &createdAt); err != nil {
			return nil, err
		}
		for _, candidate := range matchPublicModelCatalogTrafficCandidates(publicModelToCandidates, sourceModelToCandidates, accountID, requestedModel, upstreamModel, protocol.String) {
			addPublicModelCatalogTrafficRow(out[candidate.PublicModelID], status, duration, createdAt)
		}
	}
	return out, rows.Err()
}

func matchPublicModelCatalogTrafficCandidates(
	publicIndex map[string][]publicModelCatalogTrafficCandidate,
	sourceIndex map[string][]publicModelCatalogTrafficCandidate,
	accountID int64,
	requestedModel string,
	upstreamModel string,
	protocol string,
) []publicModelCatalogTrafficCandidate {
	seen := map[string]struct{}{}
	out := []publicModelCatalogTrafficCandidate{}
	add := func(index map[string][]publicModelCatalogTrafficCandidate, model string, requireAccount bool) {
		alias := service.NormalizeModelCatalogModelID(model)
		if alias == "" {
			return
		}
		for _, candidate := range index[alias] {
			if requireAccount && candidate.AccountID > 0 && candidate.AccountID != accountID {
				continue
			}
			if requireAccount && !publicModelCatalogTrafficProtocolMatches(candidate.Protocol, protocol) {
				continue
			}
			if _, ok := seen[candidate.PublicModelID]; ok {
				continue
			}
			seen[candidate.PublicModelID] = struct{}{}
			out = append(out, candidate)
		}
	}
	add(publicIndex, requestedModel, false)
	add(sourceIndex, requestedModel, true)
	add(sourceIndex, upstreamModel, true)
	return out
}

func publicModelCatalogTrafficProtocolMatches(candidateProtocol string, logProtocol string) bool {
	candidateProtocol = publicModelCatalogTrafficProtocol(candidateProtocol)
	if candidateProtocol == "" {
		return true
	}
	logProtocol = publicModelCatalogTrafficProtocol(logProtocol)
	if logProtocol == "" {
		return true
	}
	return logProtocol == candidateProtocol || strings.Contains(logProtocol, candidateProtocol)
}

func addPublicModelCatalogTrafficRow(
	aggregate *publicModelCatalogTrafficAggregate,
	status string,
	duration sql.NullInt64,
	createdAt time.Time,
) {
	if aggregate == nil {
		return
	}
	createdAt = createdAt.UTC()
	dayKey := utcDayString(createdAt)
	day := aggregate.Days[dayKey]
	if day == nil {
		day = &publicModelCatalogTrafficDaily{Date: dayKey}
		aggregate.Days[dayKey] = day
	}
	success := strings.EqualFold(strings.TrimSpace(status), service.UsageLogStatusSucceeded)
	day.Total++
	aggregate.SevenTotal++
	if success {
		day.Success++
		aggregate.SevenSuccess++
	}
	if duration.Valid && duration.Int64 > 0 && success {
		day.LatencyTotal += duration.Int64
		day.LatencyCount++
		if createdAt.After(aggregate.LatestAt) {
			aggregate.LatestAt = createdAt
			aggregate.LatestLatency = duration
		}
	} else if createdAt.After(aggregate.LatestAt) {
		aggregate.LatestAt = createdAt
		aggregate.LatestLatency = sql.NullInt64{}
	}
}

func buildPublicModelCatalogTrafficStatus(
	candidate publicModelCatalogTrafficCandidate,
	aggregate *publicModelCatalogTrafficAggregate,
	start time.Time,
	end time.Time,
	todayKey string,
) service.PublicModelCatalogStatusItem {
	status := service.PublicModelCatalogStatusItem{
		PublicModelID: candidate.PublicModelID,
		Model:         candidate.PublicModelID,
		Aliases:       publicModelCatalogPublicAliases(candidate),
		Status:        publicModelCatalogTrafficHealthStatus(aggregate),
		HealthStatus:  publicModelCatalogTrafficHealthStatus(aggregate),
		HealthSource:  service.PublicModelHealthSourceTraffic,
		StatusReason:  service.PublicModelHealthReasonTrafficRecent,
		Daily:         []service.PublicModelCatalogDailyStatus{},
		Trend:         []service.PublicModelCatalogTrendPoint{},
	}
	if !aggregate.LatestAt.IsZero() {
		status.LastCheckedAt = aggregate.LatestAt.Format(time.RFC3339)
	}
	if aggregate.LatestLatency.Valid && aggregate.LatestLatency.Int64 > 0 {
		value := aggregate.LatestLatency.Int64
		status.LatencyMs = &value
	}
	for _, day := range publicModelCatalogTrafficDailyStatuses(aggregate, start, end) {
		status.Daily = append(status.Daily, day)
		if day.SuccessRate != nil || day.LatencyMs != nil {
			status.Trend = append(status.Trend, service.PublicModelCatalogTrendPoint{
				Timestamp:   day.Date,
				SuccessRate: day.SuccessRate,
				LatencyMs:   day.LatencyMs,
			})
		}
	}
	if today := aggregate.Days[todayKey]; today != nil {
		aggregate.TodayTotal = today.Total
		aggregate.TodaySuccess = today.Success
		status.SuccessRateToday = availabilityRatePtr(today.Success, today.Total)
	}
	status.SuccessRate7d = availabilityRatePtr(aggregate.SevenSuccess, aggregate.SevenTotal)
	status.Status = publicModelCatalogTrafficHealthStatus(aggregate)
	status.HealthStatus = status.Status
	return status
}

func buildPublicModelCatalogStaleTrafficStatus(
	candidate publicModelCatalogTrafficCandidate,
	lastCheckedAt time.Time,
) service.PublicModelCatalogStatusItem {
	status := service.PublicModelCatalogStatusItem{
		PublicModelID: candidate.PublicModelID,
		Model:         candidate.PublicModelID,
		Aliases:       publicModelCatalogPublicAliases(candidate),
		Status:        service.PublicModelHealthStatusPending,
		HealthStatus:  service.PublicModelHealthStatusPending,
		HealthSource:  service.PublicModelHealthSourceNone,
		StatusReason:  service.PublicModelHealthReasonStaleHistory,
		Daily:         []service.PublicModelCatalogDailyStatus{},
		Trend:         []service.PublicModelCatalogTrendPoint{},
	}
	if !lastCheckedAt.IsZero() {
		status.LastCheckedAt = lastCheckedAt.UTC().Format(time.RFC3339)
	}
	return status
}

func publicModelCatalogPublicAliases(candidate publicModelCatalogTrafficCandidate) []string {
	aliases := []string{}
	for _, alias := range candidate.PublicAliases {
		if alias == candidate.PublicModelID {
			aliases = append(aliases, alias)
		}
	}
	if len(aliases) == 0 {
		aliases = append(aliases, candidate.PublicModelID)
	}
	return aliases
}

func publicModelCatalogTrafficDailyStatuses(
	aggregate *publicModelCatalogTrafficAggregate,
	start time.Time,
	end time.Time,
) []service.PublicModelCatalogDailyStatus {
	startDay := truncateUTCDay(start)
	endDay := truncateUTCDay(end)
	out := make([]service.PublicModelCatalogDailyStatus, 0, 7)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := utcDayString(day)
		rollup := aggregate.Days[key]
		item := service.PublicModelCatalogDailyStatus{
			Date:   key,
			Status: service.PublicModelHealthStatusPending,
		}
		if rollup != nil && rollup.Total > 0 {
			item.SuccessRate = availabilityRatePtr(rollup.Success, rollup.Total)
			item.Status = publicModelCatalogTrafficDayStatus(rollup)
			if rollup.LatencyCount > 0 {
				value := rollup.LatencyTotal / rollup.LatencyCount
				item.LatencyMs = &value
			}
		}
		out = append(out, item)
	}
	return out
}

func publicModelCatalogTrafficHealthStatus(aggregate *publicModelCatalogTrafficAggregate) string {
	if aggregate == nil || aggregate.SevenTotal == 0 {
		return service.PublicModelHealthStatusPending
	}
	if aggregate.TodayTotal > 0 && aggregate.TodaySuccess == 0 {
		return service.PublicModelHealthStatusError
	}
	rate := float64(aggregate.SevenSuccess) / float64(aggregate.SevenTotal)
	if rate < 0.9 {
		return service.PublicModelHealthStatusError
	}
	if rate < 0.98 {
		return service.PublicModelHealthStatusWarning
	}
	return service.PublicModelHealthStatusHealthy
}

func publicModelCatalogTrafficDayStatus(day *publicModelCatalogTrafficDaily) string {
	if day == nil || day.Total == 0 {
		return service.PublicModelHealthStatusPending
	}
	if day.Success == 0 {
		return service.PublicModelHealthStatusError
	}
	rate := float64(day.Success) / float64(day.Total)
	if rate < 0.98 {
		return service.PublicModelHealthStatusWarning
	}
	return service.PublicModelHealthStatusHealthy
}

func availabilityRatePtr(success int64, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	value := float64(success) / float64(total)
	return &value
}

func utcDayString(t time.Time) string {
	return truncateUTCDay(t).Format("2006-01-02")
}

func truncateUTCDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func sortedStringKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedInt64Keys(values map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
