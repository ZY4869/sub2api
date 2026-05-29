package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

func (s *defaultOpenAIAccountScheduler) buildOpenAILoadBalanceCandidates(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
	accounts []Account,
) ([]openAIAccountCandidateScore, float64, error) {
	filtered := make([]*Account, 0, len(accounts))
	loadReq := make([]AccountWithConcurrency, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if req.ExcludedIDs != nil {
			if _, excluded := req.ExcludedIDs[account.ID]; excluded {
				continue
			}
		}
		if !account.IsSchedulable() || !isOpenAITextRuntimeAccount(account) {
			continue
		}
		if req.RequestedModel != "" && !s.service.isModelSupportedByAccountWithContext(ctx, account, req.RequestedModel) {
			continue
		}
		if !account.IsSchedulableForModelWithContext(ctx, req.RequestedModel) {
			continue
		}
		if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
			continue
		}
		filtered = append(filtered, account)
		loadReq = append(loadReq, AccountWithConcurrency{
			ID:             account.ID,
			MaxConcurrency: account.EffectiveLoadFactor(),
		})
	}
	if len(filtered) == 0 {
		return nil, 0, errors.New("no available OpenAI accounts")
	}

	loadMap := map[int64]*AccountLoadInfo{}
	if s.service.concurrencyService != nil {
		if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
			loadMap = batchLoad
		}
	}

	candidates, factors := s.collectOpenAILoadBalanceCandidateStats(req, filtered, loadMap)
	s.scoreOpenAILoadBalanceCandidates(candidates, factors)
	return candidates, factors.loadSkew, nil
}

type openAILoadBalanceScoreFactors struct {
	minPriority   int
	maxPriority   int
	maxWaiting    int
	minTTFT       float64
	maxTTFT       float64
	hasTTFTSample bool
	loadSkew      float64
}

func (s *defaultOpenAIAccountScheduler) collectOpenAILoadBalanceCandidateStats(
	req OpenAIAccountScheduleRequest,
	filtered []*Account,
	loadMap map[int64]*AccountLoadInfo,
) ([]openAIAccountCandidateScore, openAILoadBalanceScoreFactors) {
	factors := openAILoadBalanceScoreFactors{
		minPriority: filtered[0].Priority,
		maxPriority: filtered[0].Priority,
		maxWaiting:  1,
	}
	loadRateSum := 0.0
	loadRateSumSquares := 0.0
	now := time.Now()
	candidates := make([]openAIAccountCandidateScore, 0, len(filtered))

	for _, account := range filtered {
		loadInfo := loadMap[account.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: account.ID}
		}
		updateOpenAILoadBalanceScoreFactors(&factors, account, loadInfo)
		errorRate, ttft, hasTTFT := s.stats.snapshot(account.ID)
		updateOpenAILoadBalanceTTFTFactors(&factors, ttft, hasTTFT)
		loadRate := float64(loadInfo.LoadRate)
		loadRateSum += loadRate
		loadRateSumSquares += loadRate * loadRate
		pressure := buildOpenAIAccountUsagePressure(account, req.RequestedModel, now)
		candidates = append(candidates, openAIAccountCandidateScore{
			account:       account,
			loadInfo:      loadInfo,
			pressure:      pressure,
			pressureScope: resolveOpenAIAccountUsagePressureScope(account, req.RequestedModel),
			expiryBoost:   AccountHasActiveExpiryProbePriority(account, now),
			planType:      openAIAccountPlanType(account),
			planRank:      resolveOpenAIAccountPlanRankForLog(account),
			errorRate:     errorRate,
			ttft:          ttft,
			hasTTFT:       hasTTFT,
		})
		if pressure != nil && strings.TrimSpace(pressure.scope) != "" {
			candidates[len(candidates)-1].pressureScope = pressure.scope
		}
	}
	factors.loadSkew = calcLoadSkewByMoments(loadRateSum, loadRateSumSquares, len(candidates))
	return candidates, factors
}

func updateOpenAILoadBalanceScoreFactors(factors *openAILoadBalanceScoreFactors, account *Account, loadInfo *AccountLoadInfo) {
	if account.Priority < factors.minPriority {
		factors.minPriority = account.Priority
	}
	if account.Priority > factors.maxPriority {
		factors.maxPriority = account.Priority
	}
	if loadInfo.WaitingCount > factors.maxWaiting {
		factors.maxWaiting = loadInfo.WaitingCount
	}
}

func updateOpenAILoadBalanceTTFTFactors(factors *openAILoadBalanceScoreFactors, ttft float64, hasTTFT bool) {
	if !hasTTFT || ttft <= 0 {
		return
	}
	if !factors.hasTTFTSample {
		factors.minTTFT = ttft
		factors.maxTTFT = ttft
		factors.hasTTFTSample = true
		return
	}
	if ttft < factors.minTTFT {
		factors.minTTFT = ttft
	}
	if ttft > factors.maxTTFT {
		factors.maxTTFT = ttft
	}
}

func (s *defaultOpenAIAccountScheduler) scoreOpenAILoadBalanceCandidates(
	candidates []openAIAccountCandidateScore,
	factors openAILoadBalanceScoreFactors,
) {
	weights := s.service.openAIWSSchedulerWeights()
	for i := range candidates {
		item := &candidates[i]
		priorityFactor := 1.0
		if item.expiryBoost {
			priorityFactor = 1.0
		} else if factors.maxPriority > factors.minPriority {
			priorityFactor = 1 - float64(item.account.Priority-factors.minPriority)/float64(factors.maxPriority-factors.minPriority)
		}
		loadFactor := 1 - clamp01(calcConcurrencyUtilization(item.loadInfo.CurrentConcurrency, item.account.Concurrency))
		queueFactor := 1 - clamp01(float64(item.loadInfo.WaitingCount)/float64(factors.maxWaiting))
		errorFactor := 1 - clamp01(item.errorRate)
		ttftFactor := 0.5
		if item.hasTTFT && factors.hasTTFTSample && factors.maxTTFT > factors.minTTFT {
			ttftFactor = 1 - clamp01((item.ttft-factors.minTTFT)/(factors.maxTTFT-factors.minTTFT))
		}

		item.score = weights.Priority*priorityFactor +
			weights.Load*loadFactor +
			weights.Queue*queueFactor +
			weights.ErrorRate*errorFactor +
			weights.TTFT*ttftFactor
	}
}

func normalizeOpenAILoadBalanceTopK(topK int, candidateCount int) int {
	if topK > candidateCount {
		topK = candidateCount
	}
	if topK <= 0 {
		return 1
	}
	return topK
}
