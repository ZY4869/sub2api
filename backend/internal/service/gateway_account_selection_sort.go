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
	minPriority := accounts[0].account.Priority
	for _, acc := range accounts[1:] {
		if acc.account.Priority < minPriority {
			minPriority = acc.account.Priority
		}
	}
	result := make([]accountWithLoad, 0, len(accounts))
	for _, acc := range accounts {
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
	var minTime *time.Time
	hasNil := false
	for _, acc := range accounts {
		if acc.account.LastUsedAt == nil {
			hasNil = true
			break
		}
		if minTime == nil || acc.account.LastUsedAt.Before(*minTime) {
			minTime = acc.account.LastUsedAt
		}
	}
	var candidateIdxs []int
	for i, acc := range accounts {
		if hasNil {
			if acc.account.LastUsedAt == nil {
				candidateIdxs = append(candidateIdxs, i)
			}
		} else {
			if acc.account.LastUsedAt != nil && acc.account.LastUsedAt.Equal(*minTime) {
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
func sortAccountsByPriorityAndLastUsed(accounts []*Account, preferOAuth bool) {
	sort.SliceStable(accounts, func(i, j int) bool {
		a, b := accounts[i], accounts[j]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if aPenalty, bPenalty := geminiRegionalPenalty(a, preferOAuth), geminiRegionalPenalty(b, preferOAuth); aPenalty != bPenalty {
			return aPenalty < bPenalty
		}
		switch {
		case a.LastUsedAt == nil && b.LastUsedAt != nil:
			return true
		case a.LastUsedAt != nil && b.LastUsedAt == nil:
			return false
		case a.LastUsedAt == nil && b.LastUsedAt == nil:
			if preferOAuth && a.Type != b.Type {
				return a.Type == AccountTypeOAuth
			}
			return false
		default:
			return a.LastUsedAt.Before(*b.LastUsedAt)
		}
	})
	shuffleWithinPriorityAndLastUsed(accounts, preferOAuth)
}
func shuffleWithinSortGroups(accounts []accountWithLoad) {
	if len(accounts) <= 1 {
		return
	}
	i := 0
	for i < len(accounts) {
		j := i + 1
		for j < len(accounts) && sameAccountWithLoadGroup(accounts[i], accounts[j]) {
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
func sameAccountWithLoadGroup(a, b accountWithLoad) bool {
	if a.account.Priority != b.account.Priority {
		return false
	}
	if geminiRegionalPenalty(a.account, true) != geminiRegionalPenalty(b.account, true) {
		return false
	}
	if a.loadInfo.LoadRate != b.loadInfo.LoadRate {
		return false
	}
	return sameLastUsedAt(a.account.LastUsedAt, b.account.LastUsedAt)
}
func shuffleWithinPriorityAndLastUsed(accounts []*Account, preferOAuth bool) {
	if len(accounts) <= 1 {
		return
	}
	i := 0
	for i < len(accounts) {
		j := i + 1
		for j < len(accounts) && sameAccountGroup(accounts[i], accounts[j]) {
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
func sameAccountGroup(a, b *Account) bool {
	if a.Priority != b.Priority {
		return false
	}
	if geminiRegionalPenalty(a, true) != geminiRegionalPenalty(b, true) {
		return false
	}
	return sameLastUsedAt(a.LastUsedAt, b.LastUsedAt)
}
func sameLastUsedAt(a, b *time.Time) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.Unix() == b.Unix()
	}
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
	sort.SliceStable(accounts, func(i, j int) bool {
		a, b := accounts[i], accounts[j]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if aPenalty, bPenalty := geminiRegionalPenalty(a, preferOAuth), geminiRegionalPenalty(b, preferOAuth); aPenalty != bPenalty {
			return aPenalty < bPenalty
		}
		if preferOAuth && a.Type != b.Type {
			return a.Type == AccountTypeOAuth
		}
		return false
	})
}
func shuffleWithinPriority(accounts []*Account) {
	if len(accounts) <= 1 {
		return
	}
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	start := 0
	for start < len(accounts) {
		priority := accounts[start].Priority
		penalty := geminiRegionalPenalty(accounts[start], true)
		end := start + 1
		for end < len(accounts) && accounts[end].Priority == priority && geminiRegionalPenalty(accounts[end], true) == penalty {
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
