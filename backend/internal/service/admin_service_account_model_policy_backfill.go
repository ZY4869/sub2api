package service

import (
	"context"
	"time"
)

func (s *adminServiceImpl) BackfillAccountModelPolicies(ctx context.Context, registry *ModelRegistryService, pageSize int) (*AccountModelPolicyBackfillResult, error) {
	return BackfillAccountModelPolicies(ctx, s.accountRepo, registry, pageSize)
}

func (s *adminServiceImpl) EnsureGrokBuildModelScope(ctx context.Context, account *Account, registry *ModelRegistryService) (*GrokBuildModelScopeEnsureResult, error) {
	return EnsureGrokBuildModelScopePersisted(ctx, s.accountRepo, account, registry, time.Now().UTC())
}
