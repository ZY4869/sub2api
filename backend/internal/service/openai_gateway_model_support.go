package service

import "context"

func (s *OpenAIGatewayService) isModelSupportedByAccountWithContext(ctx context.Context, account *Account, requestedModel string) bool {
	return isRequestedModelSupportedByAccount(ctx, s.modelRegistryService, account, requestedModel)
}

func (s *OpenAIGatewayService) isModelSupportedByAccount(account *Account, requestedModel string) bool {
	return s.isModelSupportedByAccountWithContext(context.Background(), account, requestedModel)
}
