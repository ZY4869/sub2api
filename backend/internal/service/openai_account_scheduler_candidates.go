package service

import (
	"container/heap"
	"sort"
	"strings"
)

type openAIAccountCandidateScore struct {
	account             *Account
	loadInfo            *AccountLoadInfo
	pressure            *accountUsagePressure
	pressureScope       string
	quotaHeadroom       float64
	quotaHeadroomKnown  bool
	quotaHeadroomWeight float64
	expiryBoost         bool
	planType            string
	planRank            int
	score               float64
	errorRate           float64
	ttft                float64
	hasTTFT             bool
}

type openAIAccountCandidateHeap []openAIAccountCandidateScore

func (h openAIAccountCandidateHeap) Len() int {
	return len(h)
}

func (h openAIAccountCandidateHeap) Less(i, j int) bool {
	// 最小堆根节点保存“最差”候选，便于 O(log k) 维护 topK。
	return isOpenAIAccountCandidateBetter(h[j], h[i])
}

func (h openAIAccountCandidateHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *openAIAccountCandidateHeap) Push(x any) {
	candidate, ok := x.(openAIAccountCandidateScore)
	if !ok {
		panic("openAIAccountCandidateHeap: invalid element type")
	}
	*h = append(*h, candidate)
}

func (h *openAIAccountCandidateHeap) Pop() any {
	old := *h
	n := len(old)
	last := old[n-1]
	*h = old[:n-1]
	return last
}

func isOpenAIAccountCandidateBetter(left openAIAccountCandidateScore, right openAIAccountCandidateScore) bool {
	if left.expiryBoost != right.expiryBoost {
		return left.expiryBoost
	}
	if left.account.Priority != right.account.Priority {
		return left.account.Priority < right.account.Priority
	}
	if planCmp := compareOpenAIAccountCandidatePlanRank(left, right); planCmp != 0 {
		return planCmp < 0
	}
	if headroomCmp := compareOpenAIAccountCandidateQuotaHeadroom(left, right); headroomCmp != 0 {
		return headroomCmp < 0
	}
	if pressureCmp := compareResolvedAccountUsagePressure(left.pressure, right.pressure); pressureCmp != 0 {
		return pressureCmp < 0
	}
	if left.score != right.score {
		return left.score > right.score
	}
	leftUtil := clamp01(calcConcurrencyUtilization(left.loadInfo.CurrentConcurrency, left.account.Concurrency))
	rightUtil := clamp01(calcConcurrencyUtilization(right.loadInfo.CurrentConcurrency, right.account.Concurrency))
	if leftUtil != rightUtil {
		return leftUtil < rightUtil
	}
	if left.loadInfo.LoadRate != right.loadInfo.LoadRate {
		return left.loadInfo.LoadRate < right.loadInfo.LoadRate
	}
	if left.loadInfo.WaitingCount != right.loadInfo.WaitingCount {
		return left.loadInfo.WaitingCount < right.loadInfo.WaitingCount
	}
	return left.account.ID < right.account.ID
}

func compareOpenAIAccountCandidateQuotaHeadroom(left, right openAIAccountCandidateScore) int {
	weight := left.quotaHeadroomWeight
	if right.quotaHeadroomWeight > weight {
		weight = right.quotaHeadroomWeight
	}
	if weight <= 0 {
		return 0
	}
	leftHeadroom := 0.5
	if left.quotaHeadroomKnown {
		leftHeadroom = clamp01(left.quotaHeadroom)
	}
	rightHeadroom := 0.5
	if right.quotaHeadroomKnown {
		rightHeadroom = clamp01(right.quotaHeadroom)
	}
	if leftHeadroom > rightHeadroom {
		return -1
	}
	if leftHeadroom < rightHeadroom {
		return 1
	}
	return 0
}

func compareOpenAIAccountCandidatePlanRank(left, right openAIAccountCandidateScore) int {
	if strings.TrimSpace(left.planType) == "" || strings.TrimSpace(right.planType) == "" {
		return 0
	}
	return compareOpenAIAccountPlanRankValues(left.planRank, right.planRank)
}

func selectTopKOpenAICandidates(candidates []openAIAccountCandidateScore, topK int) []openAIAccountCandidateScore {
	if len(candidates) == 0 {
		return nil
	}
	if topK <= 0 {
		topK = 1
	}
	if topK >= len(candidates) {
		ranked := append([]openAIAccountCandidateScore(nil), candidates...)
		sort.Slice(ranked, func(i, j int) bool {
			return isOpenAIAccountCandidateBetter(ranked[i], ranked[j])
		})
		return ranked
	}

	best := make(openAIAccountCandidateHeap, 0, topK)
	for _, candidate := range candidates {
		if len(best) < topK {
			heap.Push(&best, candidate)
			continue
		}
		if isOpenAIAccountCandidateBetter(candidate, best[0]) {
			best[0] = candidate
			heap.Fix(&best, 0)
		}
	}

	ranked := make([]openAIAccountCandidateScore, len(best))
	copy(ranked, best)
	sort.Slice(ranked, func(i, j int) bool {
		return isOpenAIAccountCandidateBetter(ranked[i], ranked[j])
	})
	return ranked
}
