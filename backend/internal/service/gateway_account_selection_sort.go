package service

import (
	mathrand "math/rand"
	"sort"
	"time"
)

func filterByMinPriority(accounts []accountWithLoad) []accountWithLoad {
	if len(accounts) == 0 {
		return accounts
	}
	now := time.Now().UTC()
	bestHasBoost := AccountHasActiveExpiryProbePriority(accounts[0].account, now)
	minPriority := accounts[0].account.Priority
	for _, acc := range accounts[1:] {
		currentHasBoost := AccountHasActiveExpiryProbePriority(acc.account, now)
		if currentHasBoost != bestHasBoost {
			if currentHasBoost {
				bestHasBoost = true
				minPriority = acc.account.Priority
			}
			continue
		}
		if acc.account.Priority < minPriority {
			minPriority = acc.account.Priority
		}
	}
	result := make([]accountWithLoad, 0, len(accounts))
	for _, acc := range accounts {
		if AccountHasActiveExpiryProbePriority(acc.account, now) != bestHasBoost {
			continue
		}
		if acc.account.Priority == minPriority {
			result = append(result, acc)
		}
	}
	return result
}
func filterByMinGeminiRegionalPenalty(accounts []accountWithLoad, preferOAuth bool) []accountWithLoad {
	if len(accounts) == 0 {
		return accounts
	}
	minPenalty := geminiRegionalPenalty(accounts[0].account, preferOAuth)
	for _, acc := range accounts[1:] {
		if penalty := geminiRegionalPenalty(acc.account, preferOAuth); penalty < minPenalty {
			minPenalty = penalty
		}
	}
	result := make([]accountWithLoad, 0, len(accounts))
	for _, acc := range accounts {
		if geminiRegionalPenalty(acc.account, preferOAuth) == minPenalty {
			result = append(result, acc)
		}
	}
	return result
}

func filterByMinConcurrencyUtilization(accounts []accountWithLoad) []accountWithLoad {
	if len(accounts) == 0 {
		return accounts
	}
	minNum, minDen := concurrencyUtilizationFraction(accounts[0])
	for _, acc := range accounts[1:] {
		num, den := concurrencyUtilizationFraction(acc)
		if num*minDen < minNum*den {
			minNum, minDen = num, den
		}
	}
	result := make([]accountWithLoad, 0, len(accounts))
	for _, acc := range accounts {
		num, den := concurrencyUtilizationFraction(acc)
		if num*minDen == minNum*den {
			result = append(result, acc)
		}
	}
	return result
}

func concurrencyUtilizationFraction(acc accountWithLoad) (int64, int64) {
	if acc.account == nil || acc.account.Concurrency <= 0 {
		return 0, 1
	}
	current := 0
	if acc.loadInfo != nil {
		current = acc.loadInfo.CurrentConcurrency
	}
	if current <= 0 {
		return 0, int64(acc.account.Concurrency)
	}
	return int64(current), int64(acc.account.Concurrency)
}
func filterByMinLoadRate(accounts []accountWithLoad) []accountWithLoad {
	if len(accounts) == 0 {
		return accounts
	}
	minLoadRate := accounts[0].loadInfo.LoadRate
	for _, acc := range accounts[1:] {
		if acc.loadInfo.LoadRate < minLoadRate {
			minLoadRate = acc.loadInfo.LoadRate
		}
	}
	result := make([]accountWithLoad, 0, len(accounts))
	for _, acc := range accounts {
		if acc.loadInfo.LoadRate == minLoadRate {
			result = append(result, acc)
		}
	}
	return result
}
func selectByLRU(accounts []accountWithLoad, preferOAuth bool) *accountWithLoad {
	if len(accounts) == 0 {
		return nil
	}
	if len(accounts) == 1 {
		return &accounts[0]
	}
	now := time.Now()
	var minTime *time.Time
	hasNil := false
	for _, acc := range accounts {
		lastUsedAt := normalizedSchedulerLastUsedAt(acc.account.LastUsedAt, now)
		if lastUsedAt == nil {
			hasNil = true
			break
		}
		if minTime == nil || lastUsedAt.Before(*minTime) {
			minTime = lastUsedAt
		}
	}
	var candidateIdxs []int
	for i, acc := range accounts {
		lastUsedAt := normalizedSchedulerLastUsedAt(acc.account.LastUsedAt, now)
		if hasNil {
			if lastUsedAt == nil {
				candidateIdxs = append(candidateIdxs, i)
			}
		} else {
			if lastUsedAt != nil && lastUsedAt.Equal(*minTime) {
				candidateIdxs = append(candidateIdxs, i)
			}
		}
	}
	if len(candidateIdxs) == 1 {
		return &accounts[candidateIdxs[0]]
	}
	if preferOAuth {
		var oauthIdxs []int
		for _, idx := range candidateIdxs {
			if accounts[idx].account.Type == AccountTypeOAuth {
				oauthIdxs = append(oauthIdxs, idx)
			}
		}
		if len(oauthIdxs) > 0 {
			candidateIdxs = oauthIdxs
		}
	}
	selectedIdx := candidateIdxs[mathrand.Intn(len(candidateIdxs))]
	return &accounts[selectedIdx]
}

func compareAccountsByPriorityAndLastUsed(left, right *Account, preferOAuth bool, now time.Time) int {
	if left == nil || right == nil {
		return 0
	}
	if leftBoost, rightBoost := AccountHasActiveExpiryProbePriority(left, now), AccountHasActiveExpiryProbePriority(right, now); leftBoost != rightBoost {
		if leftBoost {
			return -1
		}
		return 1
	}
	if left.Priority != right.Priority {
		if left.Priority < right.Priority {
			return -1
		}
		return 1
	}
	if leftPenalty, rightPenalty := geminiRegionalPenalty(left, preferOAuth), geminiRegionalPenalty(right, preferOAuth); leftPenalty != rightPenalty {
		if leftPenalty < rightPenalty {
			return -1
		}
		return 1
	}
	if pressureCmp := compareAccountUsagePressure(left, right, now); pressureCmp != 0 {
		return pressureCmp
	}
	switch {
	case normalizedSchedulerLastUsedAt(left.LastUsedAt, now) == nil && normalizedSchedulerLastUsedAt(right.LastUsedAt, now) != nil:
		return -1
	case normalizedSchedulerLastUsedAt(left.LastUsedAt, now) != nil && normalizedSchedulerLastUsedAt(right.LastUsedAt, now) == nil:
		return 1
	case normalizedSchedulerLastUsedAt(left.LastUsedAt, now) == nil && normalizedSchedulerLastUsedAt(right.LastUsedAt, now) == nil:
		if preferOAuth && left.Type != right.Type {
			if left.Type == AccountTypeOAuth {
				return -1
			}
			return 1
		}
		return 0
	default:
		leftLastUsedAt := normalizedSchedulerLastUsedAt(left.LastUsedAt, now)
		rightLastUsedAt := normalizedSchedulerLastUsedAt(right.LastUsedAt, now)
		if leftLastUsedAt.Before(*rightLastUsedAt) {
			return -1
		}
		if rightLastUsedAt.Before(*leftLastUsedAt) {
			return 1
		}
		return 0
	}
}

func compareAccountsWithLoad(left, right accountWithLoad, preferOAuth bool, now time.Time) int {
	if left.account == nil || right.account == nil {
		return 0
	}
	if accountCmp := compareAccountsByPriorityAndLastUsed(left.account, right.account, preferOAuth, now); accountCmp != 0 {
		if left.account.Priority != right.account.Priority {
			return accountCmp
		}
		if geminiRegionalPenalty(left.account, preferOAuth) != geminiRegionalPenalty(right.account, preferOAuth) {
			return accountCmp
		}
		if pressureCmp := compareAccountUsagePressure(left.account, right.account, now); pressureCmp != 0 {
			return pressureCmp
		}
	}
	if left.loadInfo.LoadRate != right.loadInfo.LoadRate {
		if left.loadInfo.LoadRate < right.loadInfo.LoadRate {
			return -1
		}
		return 1
	}
	return compareAccountsByPriorityAndLastUsed(left.account, right.account, preferOAuth, now)
}

func sortAccountsByPriorityAndLastUsed(accounts []*Account, preferOAuth bool) {
	now := time.Now()
	sort.SliceStable(accounts, func(i, j int) bool {
		return compareAccountsByPriorityAndLastUsed(accounts[i], accounts[j], preferOAuth, now) < 0
	})
	shuffleWithinPriorityAndLastUsed(accounts, preferOAuth, now)
}
func shuffleWithinSortGroups(accounts []accountWithLoad) {
	if len(accounts) <= 1 {
		return
	}
	now := time.Now()
	i := 0
	for i < len(accounts) {
		j := i + 1
		for j < len(accounts) && sameAccountWithLoadGroupAtTime(accounts[i], accounts[j], now) {
			j++
		}
		if j-i > 1 {
			mathrand.Shuffle(j-i, func(a, b int) {
				accounts[i+a], accounts[i+b] = accounts[i+b], accounts[i+a]
			})
		}
		i = j
	}
}
func sameAccountWithLoadGroupAtTime(a, b accountWithLoad, now time.Time) bool {
	if AccountHasActiveExpiryProbePriority(a.account, now) != AccountHasActiveExpiryProbePriority(b.account, now) {
		return false
	}
	if a.account.Priority != b.account.Priority {
		return false
	}
	if geminiRegionalPenalty(a.account, true) != geminiRegionalPenalty(b.account, true) {
		return false
	}
	if compareAccountUsagePressure(a.account, b.account, now) != 0 {
		return false
	}
	if a.loadInfo.LoadRate != b.loadInfo.LoadRate {
		return false
	}
	return sameLastUsedAtAtTime(a.account.LastUsedAt, b.account.LastUsedAt, now)
}
func shuffleWithinPriorityAndLastUsed(accounts []*Account, preferOAuth bool, now time.Time) {
	if len(accounts) <= 1 {
		return
	}
	i := 0
	for i < len(accounts) {
		j := i + 1
		for j < len(accounts) && sameAccountGroupAtTime(accounts[i], accounts[j], now) {
			j++
		}
		if j-i > 1 {
			if preferOAuth {
				oauth := make([]*Account, 0, j-i)
				others := make([]*Account, 0, j-i)
				for _, acc := range accounts[i:j] {
					if acc.Type == AccountTypeOAuth {
						oauth = append(oauth, acc)
					} else {
						others = append(others, acc)
					}
				}
				if len(oauth) > 1 {
					mathrand.Shuffle(len(oauth), func(a, b int) {
						oauth[a], oauth[b] = oauth[b], oauth[a]
					})
				}
				if len(others) > 1 {
					mathrand.Shuffle(len(others), func(a, b int) {
						others[a], others[b] = others[b], others[a]
					})
				}
				copy(accounts[i:], oauth)
				copy(accounts[i+len(oauth):], others)
			} else {
				mathrand.Shuffle(j-i, func(a, b int) {
					accounts[i+a], accounts[i+b] = accounts[i+b], accounts[i+a]
				})
			}
		}
		i = j
	}
}
func sameAccountGroupAtTime(a, b *Account, now time.Time) bool {
	if AccountHasActiveExpiryProbePriority(a, now) != AccountHasActiveExpiryProbePriority(b, now) {
		return false
	}
	if a.Priority != b.Priority {
		return false
	}
	if geminiRegionalPenalty(a, true) != geminiRegionalPenalty(b, true) {
		return false
	}
	if compareAccountUsagePressure(a, b, now) != 0 {
		return false
	}
	return sameLastUsedAtAtTime(a.LastUsedAt, b.LastUsedAt, now)
}
func sameLastUsedAt(a, b *time.Time) bool {
	return sameLastUsedAtAtTime(a, b, time.Now())
}
func sameLastUsedAtAtTime(a, b *time.Time, now time.Time) bool {
	a = normalizedSchedulerLastUsedAt(a, now)
	b = normalizedSchedulerLastUsedAt(b, now)
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.Unix() == b.Unix()
	}
}
func normalizedSchedulerLastUsedAt(value *time.Time, now time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalizedNow := now.UTC()
	normalized := value.UTC()
	if normalized.After(normalizedNow) {
		normalized = normalizedNow
	}
	return &normalized
}
func (s *GatewayService) sortCandidatesForFallback(accounts []*Account, preferOAuth bool, mode string) {
	if mode == "random" {
		sortAccountsByPriorityOnly(accounts, preferOAuth)
		shuffleWithinPriority(accounts)
	} else {
		sortAccountsByPriorityAndLastUsed(accounts, preferOAuth)
	}
}
func sortAccountsByPriorityOnly(accounts []*Account, preferOAuth bool) {
	now := time.Now()
	sort.SliceStable(accounts, func(i, j int) bool {
		return compareAccountsByPriorityAndLastUsed(accounts[i], accounts[j], preferOAuth, now) < 0
	})
}
func shuffleWithinPriority(accounts []*Account) {
	if len(accounts) <= 1 {
		return
	}
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	now := time.Now()
	start := 0
	for start < len(accounts) {
		boosted := AccountHasActiveExpiryProbePriority(accounts[start], now)
		priority := accounts[start].Priority
		penalty := geminiRegionalPenalty(accounts[start], true)
		end := start + 1
		for end < len(accounts) &&
			AccountHasActiveExpiryProbePriority(accounts[end], now) == boosted &&
			accounts[end].Priority == priority &&
			geminiRegionalPenalty(accounts[end], true) == penalty &&
			compareAccountUsagePressure(accounts[start], accounts[end], now) == 0 {
			end++
		}
		if end-start > 1 {
			r.Shuffle(end-start, func(i, j int) {
				accounts[start+i], accounts[start+j] = accounts[start+j], accounts[start+i]
			})
		}
		start = end
	}
}
