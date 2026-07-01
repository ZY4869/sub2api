package service

import (
	"hash/fnv"
	"math"
	"strconv"
	"strings"
	"time"
)

type openAISelectionRNG struct {
	state uint64
}

func newOpenAISelectionRNG(seed uint64) openAISelectionRNG {
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}
	return openAISelectionRNG{state: seed}
}

func (r *openAISelectionRNG) nextUint64() uint64 {
	// xorshift64*
	x := r.state
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	r.state = x
	return x * 2685821657736338717
}

func (r *openAISelectionRNG) nextFloat64() float64 {
	return float64(r.nextUint64()>>11) / (1 << 53)
}

func deriveOpenAISelectionSeed(req OpenAIAccountScheduleRequest) uint64 {
	hasher := fnv.New64a()
	writeValue := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		_, _ = hasher.Write([]byte(trimmed))
		_, _ = hasher.Write([]byte{0})
	}

	writeValue(req.SessionHash)
	writeValue(req.PreviousResponseID)
	writeValue(req.RequestedModel)
	if req.GroupID != nil {
		_, _ = hasher.Write([]byte(strconv.FormatInt(*req.GroupID, 10)))
	}

	seed := hasher.Sum64()
	// 对“无会话锚点”的纯负载均衡请求引入时间熵，避免固定命中同一账号。
	if strings.TrimSpace(req.SessionHash) == "" && strings.TrimSpace(req.PreviousResponseID) == "" {
		seed ^= uint64(time.Now().UnixNano())
	}
	if seed == 0 {
		seed = uint64(time.Now().UnixNano()) ^ 0x9e3779b97f4a7c15
	}
	return seed
}

func buildOpenAIWeightedSelectionOrder(
	candidates []openAIAccountCandidateScore,
	req OpenAIAccountScheduleRequest,
) []openAIAccountCandidateScore {
	if len(candidates) <= 1 {
		return append([]openAIAccountCandidateScore(nil), candidates...)
	}

	order := make([]openAIAccountCandidateScore, 0, len(candidates))
	rng := newOpenAISelectionRNG(deriveOpenAISelectionSeed(req))
	start := 0
	for start < len(candidates) {
		end := start + 1
		for end < len(candidates) && sameOpenAIWeightedSelectionGroup(candidates[start], candidates[end]) {
			end++
		}
		order = append(order, buildOpenAIWeightedSelectionGroupOrder(candidates[start:end], &rng)...)
		start = end
	}
	return order
}

func sameOpenAIWeightedSelectionGroup(left, right openAIAccountCandidateScore) bool {
	if left.account == nil || right.account == nil {
		return false
	}
	if left.expiryBoost != right.expiryBoost {
		return false
	}
	if left.account.Priority != right.account.Priority {
		return false
	}
	if compareOpenAIAccountCandidatePlanRank(left, right) != 0 {
		return false
	}
	if compareOpenAIAccountCandidateQuotaHeadroom(left, right) != 0 {
		return false
	}
	return compareResolvedAccountUsagePressure(left.pressure, right.pressure) == 0
}

func buildOpenAIWeightedSelectionGroupOrder(
	candidates []openAIAccountCandidateScore,
	rng *openAISelectionRNG,
) []openAIAccountCandidateScore {
	if len(candidates) <= 1 {
		return append([]openAIAccountCandidateScore(nil), candidates...)
	}

	pool := append([]openAIAccountCandidateScore(nil), candidates...)
	weights := make([]float64, len(pool))
	minScore := pool[0].score
	for i := 1; i < len(pool); i++ {
		if pool[i].score < minScore {
			minScore = pool[i].score
		}
	}
	for i := range pool {
		weight := (pool[i].score - minScore) + 1.0
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight <= 0 {
			weight = 1.0
		}
		weights[i] = weight
	}

	order := make([]openAIAccountCandidateScore, 0, len(pool))
	for len(pool) > 0 {
		total := 0.0
		for _, weight := range weights {
			total += weight
		}

		selectedIdx := 0
		if total > 0 {
			r := rng.nextFloat64() * total
			acc := 0.0
			for i, weight := range weights {
				acc += weight
				if r <= acc {
					selectedIdx = i
					break
				}
			}
		} else {
			selectedIdx = int(rng.nextUint64() % uint64(len(pool)))
		}

		order = append(order, pool[selectedIdx])
		pool = append(pool[:selectedIdx], pool[selectedIdx+1:]...)
		weights = append(weights[:selectedIdx], weights[selectedIdx+1:]...)
	}
	return order
}
