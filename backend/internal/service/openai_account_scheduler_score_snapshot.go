package service

import "time"

type OpenAIAccountSchedulerScoreSnapshot struct {
	BaseScore             float64
	StickyScore           float64
	StickyWeightedEnabled bool
}

func BuildOpenAIAccountSchedulerScoreSnapshot(
	accounts []Account,
	loadMap map[int64]*AccountLoadInfo,
	runtime OpenAIAdvancedSchedulerRuntimeSettings,
) map[int64]OpenAIAccountSchedulerScoreSnapshot {
	if len(accounts) == 0 {
		return map[int64]OpenAIAccountSchedulerScoreSnapshot{}
	}
	if runtime.LBTopK <= 0 {
		runtime = DefaultOpenAIAdvancedSchedulerRuntimeSettings()
	}
	filtered := make([]*Account, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if !account.IsSchedulable() || !isOpenAITextRuntimeAccount(account) {
			continue
		}
		filtered = append(filtered, account)
	}
	if len(filtered) == 0 {
		return map[int64]OpenAIAccountSchedulerScoreSnapshot{}
	}

	factors := openAILoadBalanceScoreFactors{
		minPriority: filtered[0].Priority,
		maxPriority: filtered[0].Priority,
		maxWaiting:  1,
	}
	candidates := make([]openAIAccountCandidateScore, 0, len(filtered))
	now := time.Now()
	weights := GatewayOpenAIWSSchedulerScoreWeightsFromConfig(runtime.Weights)
	for _, account := range filtered {
		loadInfo := loadMap[account.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: account.ID}
		}
		updateOpenAILoadBalanceScoreFactors(&factors, account, loadInfo)
		pressure := buildOpenAIAccountUsagePressure(account, "", now)
		quotaHeadroom, quotaHeadroomKnown := resolveOpenAIQuotaHeadroomFactor(account, "", now)
		candidates = append(candidates, openAIAccountCandidateScore{
			account:             account,
			loadInfo:            loadInfo,
			pressure:            pressure,
			pressureScope:       resolveOpenAIAccountUsagePressureScope(account, ""),
			quotaHeadroom:       quotaHeadroom,
			quotaHeadroomKnown:  quotaHeadroomKnown,
			quotaHeadroomWeight: weights.QuotaHeadroom,
			expiryBoost:         AccountHasActiveExpiryProbePriority(account, now),
			planType:            openAIAccountPlanType(account),
			planRank:            resolveOpenAIAccountPlanRankForLog(account),
		})
	}
	scoreOpenAIAccountSchedulerSnapshotCandidates(candidates, factors, runtime)

	out := make(map[int64]OpenAIAccountSchedulerScoreSnapshot, len(candidates))
	for _, candidate := range candidates {
		if candidate.account == nil {
			continue
		}
		stickyScore := candidate.score
		if runtime.StickyWeightedEnabled {
			stickyScore += weights.SessionSticky
		}
		out[candidate.account.ID] = OpenAIAccountSchedulerScoreSnapshot{
			BaseScore:             candidate.score,
			StickyScore:           stickyScore,
			StickyWeightedEnabled: runtime.StickyWeightedEnabled,
		}
	}
	return out
}

func scoreOpenAIAccountSchedulerSnapshotCandidates(
	candidates []openAIAccountCandidateScore,
	factors openAILoadBalanceScoreFactors,
	runtime OpenAIAdvancedSchedulerRuntimeSettings,
) {
	weights := GatewayOpenAIWSSchedulerScoreWeightsFromConfig(runtime.Weights)
	for i := range candidates {
		item := &candidates[i]
		priorityFactor := 1.0
		if !item.expiryBoost && factors.maxPriority > factors.minPriority {
			priorityFactor = 1 - float64(item.account.Priority-factors.minPriority)/float64(factors.maxPriority-factors.minPriority)
		}
		loadFactor := 1 - clamp01(calcConcurrencyUtilization(item.loadInfo.CurrentConcurrency, item.account.Concurrency))
		queueFactor := 1 - clamp01(float64(item.loadInfo.WaitingCount)/float64(factors.maxWaiting))
		quotaHeadroomFactor := 0.5
		if item.quotaHeadroomKnown {
			quotaHeadroomFactor = clamp01(item.quotaHeadroom)
		}
		item.score = weights.Priority*priorityFactor +
			weights.Load*loadFactor +
			weights.Queue*queueFactor +
			weights.ErrorRate +
			weights.TTFT*0.5 +
			weights.QuotaHeadroom*quotaHeadroomFactor
	}
}
