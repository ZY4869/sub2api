package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

var ErrContentModerationUsageAccountNotFound = errors.New("content moderation usage account not found")

type ContentModerationUsageAccount struct {
	APIKey       *APIKey
	Subscription *UserSubscription
	Account      *Account
}

func (s *OpenAIGatewayService) ResolveContentModerationUsageAccount(
	ctx context.Context,
	apiKey *APIKey,
	allowedPlatforms []string,
	requestedModel string,
) (*ContentModerationUsageAccount, error) {
	if s == nil || apiKey == nil {
		return nil, ErrContentModerationUsageAccountNotFound
	}
	apiKey, err := s.resolveContentModerationAPIKey(ctx, apiKey, allowedPlatforms, requestedModel)
	if err != nil {
		return nil, err
	}
	account := s.resolveContentModerationOpenAIAccount(ctx, apiKey.GroupID, requestedModel)
	if account == nil {
		return nil, ErrContentModerationUsageAccountNotFound
	}
	subscription, err := s.resolveContentModerationSubscription(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	return &ContentModerationUsageAccount{APIKey: apiKey, Subscription: subscription, Account: account}, nil
}

func (s *GatewayService) ResolveContentModerationUsageAccount(
	ctx context.Context,
	apiKey *APIKey,
	allowedPlatforms []string,
	requestedModel string,
) (*ContentModerationUsageAccount, error) {
	if s == nil || apiKey == nil {
		return nil, ErrContentModerationUsageAccountNotFound
	}
	apiKey, err := s.resolveContentModerationAPIKey(ctx, apiKey, allowedPlatforms, requestedModel)
	if err != nil {
		return nil, err
	}
	account := s.resolveContentModerationGatewayAccount(ctx, apiKey.GroupID, requestedModel)
	if account == nil {
		return nil, ErrContentModerationUsageAccountNotFound
	}
	subscription, err := s.resolveContentModerationSubscription(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	return &ContentModerationUsageAccount{APIKey: apiKey, Subscription: subscription, Account: account}, nil
}

func (s *OpenAIGatewayService) resolveContentModerationAPIKey(
	ctx context.Context,
	apiKey *APIKey,
	allowedPlatforms []string,
	requestedModel string,
) (*APIKey, error) {
	binding, err := s.SelectGroupForAllowedPlatforms(ctx, apiKey, allowedPlatforms, requestedModel, nil)
	if err != nil {
		return nil, err
	}
	return CloneAPIKeyWithSelectedGroup(apiKey, binding), nil
}

func (s *GatewayService) resolveContentModerationAPIKey(
	ctx context.Context,
	apiKey *APIKey,
	allowedPlatforms []string,
	requestedModel string,
) (*APIKey, error) {
	binding, err := s.SelectGroupForAllowedPlatforms(ctx, apiKey, allowedPlatforms, requestedModel, nil)
	if err != nil {
		return nil, err
	}
	return CloneAPIKeyWithSelectedGroup(apiKey, binding), nil
}

func (s *OpenAIGatewayService) resolveContentModerationOpenAIAccount(ctx context.Context, groupID *int64, requestedModel string) *Account {
	accounts, err := s.listSchedulableAccounts(ctx, groupID)
	if err != nil || len(accounts) == 0 {
		return nil
	}
	candidates := make([]*Account, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, account, requestedModel); fresh != nil {
			candidates = append(candidates, fresh)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	now := time.Now()
	sort.SliceStable(candidates, func(i, j int) bool {
		return compareOpenAIAccountsForSelection(candidates[i], candidates[j], requestedModel, now) < 0
	})
	return candidates[0]
}

func (s *GatewayService) resolveContentModerationGatewayAccount(ctx context.Context, groupID *int64, requestedModel string) *Account {
	group, resolvedGroupID, err := s.checkClaudeCodeRestriction(ctx, groupID)
	if err != nil {
		return nil
	}
	ctx = s.withGroupContext(ctx, group)
	platform, hasForcePlatform, err := s.resolvePlatform(ctx, resolvedGroupID, group)
	if err != nil {
		return nil
	}
	accounts, useMixed, err := s.listSchedulableAccounts(ctx, resolvedGroupID, platform, hasForcePlatform)
	if err != nil || len(accounts) == 0 {
		return nil
	}
	ctx = s.withWindowCostPrefetch(ctx, accounts)
	ctx = s.withRPMPrefetch(ctx, accounts)
	preferOAuth := platform == PlatformGemini
	candidates := make([]*Account, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if fresh := s.resolveFreshSelectionAccount(ctx, account, platform, useMixed, requestedModel, false); fresh != nil {
			candidates = append(candidates, fresh)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sortAccountsByPriorityAndLastUsed(candidates, preferOAuth)
	stableSortAccountsByGeminiPublicProtocolRank(ctx, candidates)
	return candidates[0]
}

func (s *OpenAIGatewayService) resolveContentModerationSubscription(
	ctx context.Context,
	apiKey *APIKey,
) (*UserSubscription, error) {
	return resolveContentModerationSubscription(ctx, apiKey, s.GetActiveSubscriptionForGroup)
}

func (s *GatewayService) resolveContentModerationSubscription(
	ctx context.Context,
	apiKey *APIKey,
) (*UserSubscription, error) {
	return resolveContentModerationSubscription(ctx, apiKey, s.GetActiveSubscriptionForGroup)
}

func resolveContentModerationSubscription(
	ctx context.Context,
	apiKey *APIKey,
	loader func(context.Context, int64, int64) (*UserSubscription, error),
) (*UserSubscription, error) {
	if apiKey == nil || apiKey.Group == nil || !apiKey.Group.IsSubscriptionType() || loader == nil {
		return nil, nil
	}
	if apiKey.GroupID == nil {
		return nil, ErrContentModerationUsageAccountNotFound
	}
	userID := apiKey.UserID
	if apiKey.User != nil && apiKey.User.ID > 0 {
		userID = apiKey.User.ID
	}
	subscription, err := loader(ctx, userID, *apiKey.GroupID)
	if err != nil {
		if !errors.Is(err, ErrSubscriptionNotFound) {
			return nil, nil
		}
		return nil, nil
	}
	return subscription, nil
}

func ContentModerationRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return strings.TrimSpace(requestID)
	}
	return ""
}

func ContentModerationClientRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(requestID) != "" {
		return strings.TrimSpace(requestID)
	}
	return ""
}
