package service

import (
	"context"
	"strings"
)

func (s *GatewayService) publishedPublicCatalogItemForBinding(
	ctx context.Context,
	binding APIKeyGroupBinding,
	requestedPlatform string,
	item PublicModelCatalogItem,
) (apiKeyPublishedPublicCatalogMatch, bool, error) {
	publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	if s == nil || publicID == "" || binding.Group == nil || !binding.Group.IsActive() {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	if !publicModelCatalogItemConfirmedAvailable(item) {
		recordPublicCatalogRouteMiss(ctx, nil, bindingGroupIDPtr(binding), publicID, requestedPlatform)
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	if _, matched := bindingMatchesModel(binding.ModelPatterns, publicID); !matched {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	projectionPlatform, ok := publishedCatalogBindingItemPlatform(binding, requestedPlatform, item)
	if !ok {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}

	var resolvedAccount *Account
	if item.SourceAccountID > 0 {
		account, err := s.accountRepo.GetByID(ctx, item.SourceAccountID)
		if err != nil || account == nil {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_found", err)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		if !publishedCatalogAccountUsableForBinding(ctx, s.modelRegistryService, account, binding.GroupID, projectionPlatform, item) {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_available_for_group", nil)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		resolvedAccount = ResolveProtocolGatewayInboundAccount(account, firstNonEmptyTrimmed(item.SourceProtocol, projectionPlatform, RoutingPlatformForAccount(account)))
		if resolvedAccount == nil {
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
	}

	entry := APIKeyPublicModelEntry{
		PublicID:          publicID,
		AliasID:           publicID,
		SourceID:          NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel)),
		DisplayName:       firstNonEmptyTrimmed(item.DisplayName, item.BaseModel, publicID),
		Platform:          firstNonEmptyTrimmed(item.SourceProtocol, item.Provider, projectionPlatform),
		AvailabilityState: firstNonEmptyTrimmed(item.AvailabilityState, AccountModelAvailabilityUnknown),
		StaleState:        firstNonEmptyTrimmed(item.StaleState, AccountModelStaleStateUnverified),
		LifecycleStatus:   normalizePublicModelLifecycleStatus(item.LifecycleStatus, item.DisplayName, publicID),
	}
	if !binding.Group.AllowsVisibleModel(entry.PublicID, entry.AliasID, entry.SourceID, entry.DisplayName) {
		recordPublicCatalogRouteMiss(ctx, nil, bindingGroupIDPtr(binding), publicID, requestedPlatform)
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	return apiKeyPublishedPublicCatalogMatch{
		Entry:      entry,
		Catalog:    publishedPublicCatalogEntryFromItem(item),
		Binding:    binding,
		GroupID:    bindingGroupIDPtr(binding),
		Account:    resolvedAccount,
		SourceItem: clonePublicModelCatalogItem(item),
	}, true, nil
}

func (s *OpenAIGatewayService) publishedPublicCatalogItemForBinding(
	ctx context.Context,
	binding APIKeyGroupBinding,
	requestedPlatform string,
	item PublicModelCatalogItem,
) (apiKeyPublishedPublicCatalogMatch, bool, error) {
	publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	if s == nil || publicID == "" || binding.Group == nil || !binding.Group.IsActive() {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	if !publicModelCatalogItemConfirmedAvailable(item) {
		recordPublicCatalogRouteMiss(ctx, nil, bindingGroupIDPtr(binding), publicID, requestedPlatform)
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	if _, matched := bindingMatchesModel(binding.ModelPatterns, publicID); !matched {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	projectionPlatform, ok := publishedCatalogBindingItemPlatform(binding, requestedPlatform, item)
	if !ok {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}

	var resolvedAccount *Account
	if item.SourceAccountID > 0 {
		account, err := s.accountRepo.GetByID(ctx, item.SourceAccountID)
		if err != nil || account == nil {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_found", err)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		if !publishedCatalogAccountUsableForBinding(ctx, s.modelRegistryService, account, binding.GroupID, projectionPlatform, item) {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_available_for_group", nil)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		resolvedAccount = ResolveProtocolGatewayInboundAccount(account, firstNonEmptyTrimmed(item.SourceProtocol, projectionPlatform, OpenAIPlatformFromContext(ctx)))
		if resolvedAccount == nil || !isOpenAITextRuntimeAccount(resolvedAccount) {
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
	}

	entry := APIKeyPublicModelEntry{
		PublicID:          publicID,
		AliasID:           publicID,
		SourceID:          NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel)),
		DisplayName:       firstNonEmptyTrimmed(item.DisplayName, item.BaseModel, publicID),
		Platform:          firstNonEmptyTrimmed(item.SourceProtocol, item.Provider, projectionPlatform),
		AvailabilityState: firstNonEmptyTrimmed(item.AvailabilityState, AccountModelAvailabilityUnknown),
		StaleState:        firstNonEmptyTrimmed(item.StaleState, AccountModelStaleStateUnverified),
		LifecycleStatus:   normalizePublicModelLifecycleStatus(item.LifecycleStatus, item.DisplayName, publicID),
	}
	if !binding.Group.AllowsVisibleModel(entry.PublicID, entry.AliasID, entry.SourceID, entry.DisplayName) {
		recordPublicCatalogRouteMiss(ctx, nil, bindingGroupIDPtr(binding), publicID, requestedPlatform)
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	return apiKeyPublishedPublicCatalogMatch{
		Entry:      entry,
		Catalog:    publishedPublicCatalogEntryFromItem(item),
		Binding:    binding,
		GroupID:    bindingGroupIDPtr(binding),
		Account:    resolvedAccount,
		SourceItem: clonePublicModelCatalogItem(item),
	}, true, nil
}

func publishedCatalogBindingItemPlatform(binding APIKeyGroupBinding, requestedPlatform string, item PublicModelCatalogItem) (string, bool) {
	if binding.Group == nil {
		return "", false
	}
	bindingPlatform := strings.TrimSpace(strings.ToLower(binding.Group.Platform))
	requestedPlatform = strings.TrimSpace(strings.ToLower(requestedPlatform))
	if requestedPlatform != "" {
		projectionPlatform := apiKeyPublicProjectionPlatform(bindingPlatform, requestedPlatform)
		if projectionPlatform == "" {
			return "", false
		}
		if publicCatalogItemMatchesProtocol(item, requestedPlatform) || publicCatalogItemMatchesProtocol(item, projectionPlatform) {
			return projectionPlatform, true
		}
		return "", false
	}
	if bindingPlatform == PlatformProtocolGateway {
		for _, protocol := range []string{PlatformOpenAI, PlatformAnthropic, PlatformGemini} {
			if publicCatalogItemMatchesProtocol(item, protocol) {
				return protocol, true
			}
		}
		return "", false
	}
	if publicCatalogItemMatchesProtocol(item, bindingPlatform) {
		return bindingPlatform, true
	}
	return "", false
}

func publicCatalogItemMatchesProtocol(item PublicModelCatalogItem, protocol string) bool {
	protocol = strings.TrimSpace(strings.ToLower(protocol))
	if protocol == "" {
		return false
	}
	for _, candidate := range append([]string{item.SourceProtocol, item.Provider}, item.RequestProtocols...) {
		if strings.TrimSpace(strings.ToLower(candidate)) == protocol {
			return true
		}
	}
	return false
}
