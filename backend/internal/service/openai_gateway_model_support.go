package service

import "context"

func (s *OpenAIGatewayService) isModelSupportedByAccountWithContext(ctx context.Context, account *Account, requestedModel string) bool {
	return isRequestedModelSupportedByAccount(ctx, s.modelRegistryService, account, requestedModel)
}

func (s *OpenAIGatewayService) isModelSupportedByAccount(account *Account, requestedModel string) bool {
	return s.isModelSupportedByAccountWithContext(context.Background(), account, requestedModel)
}

func (s *OpenAIGatewayService) IsModelUnavailableBecauseUnsupported(ctx context.Context, groupID *int64, requestedModel string, excludedIDs map[int64]struct{}) bool {
	if s == nil {
		return false
	}
	accounts, err := s.listSchedulableAccounts(ctx, groupID)
	if err != nil {
		return false
	}
	return s.openAISelectionFailureIsModelUnsupported(ctx, accounts, requestedModel, excludedIDs)
}
