package service

import (
	"context"
	"errors"
)

func (s *defaultOpenAIAccountScheduler) selectByLoadBalance(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, int, int, float64, error) {
	accounts, err := s.service.listSchedulableAccounts(ctx, req.GroupID)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if len(accounts) == 0 {
		return nil, 0, 0, 0, errors.New("no available OpenAI accounts")
	}

	candidates, loadSkew, err := s.buildOpenAILoadBalanceCandidates(ctx, req, accounts)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	selectionCandidates := candidates
	subscriptionFallback := []openAIAccountCandidateScore(nil)
	if req.SchedulerRuntime.SubscriptionPriorityEnabled {
		if subscriptionCandidates, otherCandidates := splitOpenAISubscriptionPriorityCandidates(candidates); len(subscriptionCandidates) > 0 && len(otherCandidates) > 0 {
			selectionCandidates = subscriptionCandidates
			subscriptionFallback = otherCandidates
		}
	}

	topK := normalizeOpenAILoadBalanceTopK(req.SchedulerRuntime.LBTopK, len(selectionCandidates))
	rankedCandidates := selectTopKOpenAICandidates(selectionCandidates, topK)
	selectionOrder := buildOpenAIWeightedSelectionOrder(rankedCandidates, req)
	topKAccountIDs := make(map[int64]struct{}, len(selectionOrder))
	for _, candidate := range selectionOrder {
		if candidate.account != nil {
			topKAccountIDs[candidate.account.ID] = struct{}{}
		}
	}

	if selection, ok, err := s.tryOpenAILoadBalanceAcquire(ctx, req, selectionOrder, len(candidates), topK, loadSkew, "acquired"); ok || err != nil {
		return selection, len(candidates), topK, loadSkew, err
	}

	fallbackOrder := make([]openAIAccountCandidateScore, 0, len(candidates))
	fallbackSources := [][]openAIAccountCandidateScore{candidates}
	if len(subscriptionFallback) > 0 {
		fallbackSources = [][]openAIAccountCandidateScore{selectionCandidates, subscriptionFallback}
	}
	for _, source := range fallbackSources {
		for _, candidate := range selectTopKOpenAICandidates(source, len(source)) {
			if candidate.account == nil {
				continue
			}
			if _, alreadyTried := topKAccountIDs[candidate.account.ID]; alreadyTried {
				continue
			}
			fallbackOrder = append(fallbackOrder, candidate)
		}
	}

	if selection, ok, err := s.tryOpenAILoadBalanceAcquire(ctx, req, fallbackOrder, len(candidates), topK, loadSkew, "fallback_all_acquire"); ok || err != nil {
		return selection, len(candidates), topK, loadSkew, err
	}

	waitOrder := append(append([]openAIAccountCandidateScore(nil), selectionOrder...), fallbackOrder...)
	if selection, ok, err := s.buildOpenAILoadBalanceWaitSelection(ctx, req, waitOrder, len(candidates), topK, loadSkew); ok || err != nil {
		return selection, len(candidates), topK, loadSkew, err
	}

	return nil, len(candidates), topK, loadSkew, ErrNoAvailableAccounts
}

func splitOpenAISubscriptionPriorityCandidates(candidates []openAIAccountCandidateScore) ([]openAIAccountCandidateScore, []openAIAccountCandidateScore) {
	subscriptionCandidates := make([]openAIAccountCandidateScore, 0, len(candidates))
	otherCandidates := make([]openAIAccountCandidateScore, 0, len(candidates))
	for _, candidate := range candidates {
		if isOpenAIChatGPTSubscriptionCandidate(candidate.account) {
			subscriptionCandidates = append(subscriptionCandidates, candidate)
			continue
		}
		otherCandidates = append(otherCandidates, candidate)
	}
	return subscriptionCandidates, otherCandidates
}

func isOpenAIChatGPTSubscriptionCandidate(account *Account) bool {
	rank, ok := resolveOpenAIAccountPlanRank(account)
	return ok && rank >= 0 && rank < 3
}
