package service

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
)

func (s *APIKeyService) Update(ctx context.Context, id int64, userID int64, req UpdateAPIKeyRequest) (*APIKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}

	// 验证所有权
	if apiKey.UserID != userID {
		return nil, ErrInsufficientPerms
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	requestedGroups, err := resolveRequestedAPIKeyGroupUpdates(req.GroupID, req.Groups)
	if err != nil {
		return nil, err
	}

	if len(req.IPWhitelist) > 0 {
		if invalid := ip.ValidateIPPatterns(req.IPWhitelist); len(invalid) > 0 {
			return nil, fmt.Errorf("%w: %v", ErrInvalidIPPattern, invalid)
		}
	}

	if len(req.IPBlacklist) > 0 {
		if invalid := ip.ValidateIPPatterns(req.IPBlacklist); len(invalid) > 0 {
			return nil, fmt.Errorf("%w: %v", ErrInvalidIPPattern, invalid)
		}
	}

	var opCtx context.Context
	var txStarter apiKeyGroupMutationTxStarter
	if starter, ok := s.apiKeyRepo.(apiKeyGroupMutationTxStarter); ok {
		txStarter = starter
	}
	opCtx, tx, rollback, err := beginAPIKeyMutationTx(ctx, txStarter)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer rollback()

	resetRateLimit := s.applyAPIKeyUpdateFields(opCtx, apiKey, req)
	pendingGroupBindings, shouldSetGroupBindings, err := s.prepareAPIKeyUpdateGroupBindings(opCtx, apiKey, user, requestedGroups)
	if err != nil {
		return nil, err
	}

	if err := s.apiKeyRepo.Update(opCtx, apiKey); err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}

	if shouldSetGroupBindings {
		for i := range pendingGroupBindings {
			pendingGroupBindings[i].APIKeyID = apiKey.ID
		}
		if err := s.apiKeyRepo.SetAPIKeyGroups(opCtx, apiKey.ID, pendingGroupBindings); err != nil {
			return nil, fmt.Errorf("set api key groups: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}
	}

	updatedKey, err := s.apiKeyRepo.GetByID(ctx, apiKey.ID)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	s.InvalidateAuthCacheByKey(ctx, updatedKey.Key)
	s.compileAPIKeyIPRules(updatedKey)

	if resetRateLimit && s.rateLimitCacheInvalid != nil {
		_ = s.rateLimitCacheInvalid.InvalidateAPIKeyRateLimit(ctx, updatedKey.ID)
	}

	return updatedKey, nil
}

// Delete 删除API Key
