package service

import (
	"context"
	"time"
)

func (s *OpenAIGatewayService) recheckSelectedOpenAIAccountFromDB(ctx context.Context, account *Account, requestedModel string) *Account {
	if account == nil {
		return nil
	}
	if s == nil || s.schedulerSnapshot == nil || s.accountRepo == nil {
		return account
	}

	latest, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil || latest == nil {
		return nil
	}
	syncOpenAICodexRateLimitFromExtra(ctx, s.accountRepo, latest, time.Now())
	if !latest.IsSchedulable() || !latest.IsOpenAI() {
		return nil
	}
	if requestedModel != "" && !latest.IsModelSupported(requestedModel) {
		return nil
	}
	return latest
}
