package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

const contentModerationCompiledRuleCacheMaxEntries = 64

type compiledContentModerationKeywordRule struct {
	Keyword    string
	Comparable string
	Reason     string
}

type compiledContentModerationCyberCategory struct {
	ID       string
	Reason   string
	Keywords []compiledContentModerationKeywordRule
}

var contentModerationKeywordRulesCache = struct {
	sync.RWMutex
	items map[string][]compiledContentModerationKeywordRule
}{items: map[string][]compiledContentModerationKeywordRule{}}

var contentModerationCyberRulesCache = struct {
	sync.RWMutex
	items map[string][]compiledContentModerationCyberCategory
}{items: map[string][]compiledContentModerationCyberCategory{}}

func compiledContentModerationKeywordRules(keywords []string) []compiledContentModerationKeywordRule {
	normalized := NormalizeContentModerationKeywordList(keywords)
	if len(normalized) == 0 {
		return nil
	}
	key := contentModerationCompiledRulesCacheKey(normalized)
	contentModerationKeywordRulesCache.RLock()
	if cached, ok := contentModerationKeywordRulesCache.items[key]; ok {
		contentModerationKeywordRulesCache.RUnlock()
		return cached
	}
	contentModerationKeywordRulesCache.RUnlock()

	compiled := make([]compiledContentModerationKeywordRule, 0, len(normalized))
	for _, keyword := range normalized {
		comparable := normalizeContentModerationKeywordComparable(keyword)
		if comparable == "" {
			continue
		}
		sum := sha256.Sum256([]byte(comparable))
		compiled = append(compiled, compiledContentModerationKeywordRule{
			Keyword:    keyword,
			Comparable: comparable,
			Reason:     fmt.Sprintf("keyword_blocked:%s", hex.EncodeToString(sum[:])[:12]),
		})
	}
	if len(compiled) == 0 {
		return nil
	}

	contentModerationKeywordRulesCache.Lock()
	if cached, ok := contentModerationKeywordRulesCache.items[key]; ok {
		contentModerationKeywordRulesCache.Unlock()
		return cached
	}
	if len(contentModerationKeywordRulesCache.items) >= contentModerationCompiledRuleCacheMaxEntries {
		contentModerationKeywordRulesCache.items = map[string][]compiledContentModerationKeywordRule{}
	}
	contentModerationKeywordRulesCache.items[key] = compiled
	contentModerationKeywordRulesCache.Unlock()
	return compiled
}

func compiledContentModerationCyberRules(categories []ContentModerationCyberCategory) []compiledContentModerationCyberCategory {
	normalized := NormalizeContentModerationCyberCategoryList(categories)
	if len(normalized) == 0 {
		return nil
	}
	key := contentModerationCompiledRulesCacheKey(normalized)
	contentModerationCyberRulesCache.RLock()
	if cached, ok := contentModerationCyberRulesCache.items[key]; ok {
		contentModerationCyberRulesCache.RUnlock()
		return cached
	}
	contentModerationCyberRulesCache.RUnlock()

	compiled := make([]compiledContentModerationCyberCategory, 0, len(normalized))
	for _, category := range normalized {
		item := compiledContentModerationCyberCategory{
			ID:     category.ID,
			Reason: "cyber_policy:" + category.ID,
		}
		for _, keyword := range NormalizeContentModerationKeywordList(category.Keywords) {
			comparable := normalizeContentModerationKeywordComparable(keyword)
			if comparable == "" {
				continue
			}
			item.Keywords = append(item.Keywords, compiledContentModerationKeywordRule{
				Keyword:    keyword,
				Comparable: comparable,
			})
		}
		if len(item.Keywords) > 0 {
			compiled = append(compiled, item)
		}
	}
	if len(compiled) == 0 {
		return nil
	}

	contentModerationCyberRulesCache.Lock()
	if cached, ok := contentModerationCyberRulesCache.items[key]; ok {
		contentModerationCyberRulesCache.Unlock()
		return cached
	}
	if len(contentModerationCyberRulesCache.items) >= contentModerationCompiledRuleCacheMaxEntries {
		contentModerationCyberRulesCache.items = map[string][]compiledContentModerationCyberCategory{}
	}
	contentModerationCyberRulesCache.items[key] = compiled
	contentModerationCyberRulesCache.Unlock()
	return compiled
}

func contentModerationCompiledRulesCacheKey(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", value))
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func resetContentModerationCompiledRuleCachesForTest() {
	contentModerationKeywordRulesCache.Lock()
	contentModerationKeywordRulesCache.items = map[string][]compiledContentModerationKeywordRule{}
	contentModerationKeywordRulesCache.Unlock()

	contentModerationCyberRulesCache.Lock()
	contentModerationCyberRulesCache.items = map[string][]compiledContentModerationCyberCategory{}
	contentModerationCyberRulesCache.Unlock()
}
