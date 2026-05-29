package service

import (
	"strconv"
	"strings"
)

func publicModelCatalogPublishItemLookups(items []PublicModelCatalogItem) (map[string]PublicModelCatalogItem, map[string]PublicModelCatalogItem) {
	byEntryID := make(map[string]PublicModelCatalogItem, len(items))
	bySource := make(map[string]PublicModelCatalogItem, len(items))
	ambiguousSources := map[string]struct{}{}
	for _, item := range items {
		if entryID := strings.TrimSpace(item.EntryID); entryID != "" {
			byEntryID[entryID] = item
		}
		for _, key := range publicModelCatalogItemSourceKeys(item) {
			if _, ambiguous := ambiguousSources[key]; ambiguous {
				continue
			}
			if existing, exists := bySource[key]; exists && !samePublicModelCatalogSourceItem(existing, item) {
				delete(bySource, key)
				ambiguousSources[key] = struct{}{}
				continue
			}
			bySource[key] = item
		}
	}
	return byEntryID, bySource
}

func resolvePublicModelCatalogPublishItem(
	entry PublicModelCatalogEntryDraft,
	byEntryID map[string]PublicModelCatalogItem,
	bySource map[string]PublicModelCatalogItem,
) (PublicModelCatalogItem, bool) {
	if item, ok := byEntryID[strings.TrimSpace(entry.EntryID)]; ok {
		return item, true
	}
	for _, key := range publicModelCatalogDraftSourceKeys(entry) {
		if item, ok := bySource[key]; ok {
			return item, true
		}
	}
	return PublicModelCatalogItem{}, false
}

func publicModelCatalogDraftSourceKeys(entry PublicModelCatalogEntryDraft) []string {
	return publicModelCatalogSourceKeys(
		entry.SourceAccountID,
		entry.SourceProtocol,
		entry.SourceModelID,
		entry.BaseModel,
	)
}

func publicModelCatalogItemSourceKeys(item PublicModelCatalogItem) []string {
	models := append([]string{item.SourceModelID, item.BaseModel}, item.SourceIDs...)
	protocols := append([]string{item.SourceProtocol}, item.RequestProtocols...)
	return publicModelCatalogSourceKeys(item.SourceAccountID, protocols, models...)
}

func publicModelCatalogSourceKeys(accountID int64, protocols any, models ...string) []string {
	protocolValues := publicModelCatalogSourceProtocolValues(protocols)
	accountValues := []int64{accountID}
	if accountID > 0 {
		accountValues = append(accountValues, 0)
	}
	keys := make([]string, 0, len(accountValues)*len(protocolValues)*len(models))
	for _, currentAccountID := range accountValues {
		for _, protocol := range protocolValues {
			for _, model := range models {
				key := publicModelCatalogSourceKey(currentAccountID, protocol, model)
				if key == "" || containsString(keys, key) {
					continue
				}
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func publicModelCatalogSourceProtocolValues(protocols any) []string {
	seen := map[string]struct{}{}
	values := []string{}
	appendProtocol := func(protocol string) {
		normalized := strings.TrimSpace(strings.ToLower(protocol))
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		values = append(values, normalized)
	}
	switch typed := protocols.(type) {
	case string:
		appendProtocol(typed)
	case []string:
		for _, protocol := range typed {
			appendProtocol(protocol)
		}
	}
	appendProtocol("")
	return values
}

func publicModelCatalogSourceKey(accountID int64, protocol string, model string) string {
	sourceModel := NormalizeModelCatalogModelID(model)
	if sourceModel == "" {
		return ""
	}
	return strings.Join([]string{
		strconv.FormatInt(accountID, 10),
		strings.TrimSpace(strings.ToLower(protocol)),
		sourceModel,
	}, ":")
}

func samePublicModelCatalogSourceItem(left PublicModelCatalogItem, right PublicModelCatalogItem) bool {
	leftEntryID := strings.TrimSpace(left.EntryID)
	rightEntryID := strings.TrimSpace(right.EntryID)
	if leftEntryID != "" || rightEntryID != "" {
		return leftEntryID != "" && leftEntryID == rightEntryID
	}
	return left.SourceAccountID == right.SourceAccountID &&
		strings.EqualFold(strings.TrimSpace(left.SourceProtocol), strings.TrimSpace(right.SourceProtocol)) &&
		NormalizeModelCatalogModelID(firstNonEmptyTrimmed(left.SourceModelID, left.BaseModel, left.Model)) ==
			NormalizeModelCatalogModelID(firstNonEmptyTrimmed(right.SourceModelID, right.BaseModel, right.Model))
}
