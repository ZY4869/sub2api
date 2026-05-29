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

	topK := normalizeOpenAILoadBalanceTopK(s.service.openAIWSLBTopK(), len(candidates))
	rankedCandidates := selectTopKOpenAICandidates(candidates, topK)
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
	for _, candidate := range selectTopKOpenAICandidates(candidates, len(candidates)) {
		if candidate.account == nil {
			continue
		}
		if _, alreadyTried := topKAccountIDs[candidate.account.ID]; alreadyTried {
			continue
		}
		fallbackOrder = append(fallbackOrder, candidate)
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
