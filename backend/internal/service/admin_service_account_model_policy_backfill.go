package service

import "context"

func (s *adminServiceImpl) BackfillAccountModelPolicies(ctx context.Context, registry *ModelRegistryService, pageSize int) (*AccountModelPolicyBackfillResult, error) {
	return BackfillAccountModelPolicies(ctx, s.accountRepo, registry, pageSize)
}
