package service

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"time"
)

func (s *APIKeyService) Create(ctx context.Context, userID int64, req CreateAPIKeyRequest) (*APIKey, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	requestedGroups, err := resolveRequestedAPIKeyGroupUpdates(req.GroupID, req.Groups)
	if err != nil {
		return nil, err
	}
	if req.ImageOnlyEnabled && (requestedGroups == nil || len(*requestedGroups) == 0) {
		return nil, ErrImageOnlyGroupRequired
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
	accessPolicy, err := NormalizeTimeAccessPolicy(req.AccessTimePolicy)
	if err != nil {
		return nil, timeAccessPolicyInputError(err)
	}
	if err := ValidateTimeAccessSubset(accessPolicy, user.APIKeyAccessTimePolicy); err != nil {
		return nil, timeAccessPolicyInputError(err)
	}

	var key string

	if req.CustomKey != nil && *req.CustomKey != "" {
		if err := s.checkAPIKeyRateLimit(ctx, userID); err != nil {
			return nil, err
		}

		if err := s.ValidateCustomKey(*req.CustomKey); err != nil {
			return nil, err
		}

		exists, err := s.apiKeyRepo.ExistsByKey(ctx, *req.CustomKey)
		if err != nil {
			return nil, fmt.Errorf("check key exists: %w", err)
		}
		if exists {
			s.incrementAPIKeyErrorCount(ctx, userID)
			return nil, ErrAPIKeyExists
		}

		key = *req.CustomKey
	} else {
		key, err = s.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
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

	imageOnlyEnabled := req.ImageOnlyEnabled
	imageCountBillingEnabled := false
	imageMaxCount := 0
	if imageOnlyEnabled {
		maxCandidate := req.ImageMaxCount
		if maxCandidate < 0 {
			maxCandidate = 0
		}
		if req.ImageCountBillingEnabled && maxCandidate > 0 {
			imageCountBillingEnabled = true
			imageMaxCount = maxCandidate
		}
	}

	apiKey := &APIKey{
		UserID: userID,
		Key:    key,
		Name:   req.Name,
		ModelDisplayMode: NormalizeAPIKeyModelDisplayMode(func() string {
			if req.ModelDisplayMode == nil {
				return ""
			}
			return *req.ModelDisplayMode
		}()),
		Status:                   StatusActive,
		IPWhitelist:              req.IPWhitelist,
		IPBlacklist:              req.IPBlacklist,
		ImageOnlyEnabled:         imageOnlyEnabled,
		ImageCountBillingEnabled: imageCountBillingEnabled,
		ImageMaxCount:            imageMaxCount,
		ImageCountUsed:           0,
		ImageCountWeights:        NormalizeAPIKeyImageCountWeights(req.ImageCountWeights),
		Quota:                    req.Quota,
		QuotaUsed:                0,
		StartsAt:                 req.StartsAt,
		AccessTimePolicy:         accessPolicy,
		RateLimit5h:              req.RateLimit5h,
		RateLimit1d:              req.RateLimit1d,
		RateLimit7d:              req.RateLimit7d,
	}
	sanitizeAPIKeyImageCountBillingForActor(apiKey, user)

	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		apiKey.ExpiresAt = &expiresAt
	}

	var pendingGroupBindings []APIKeyGroupBinding
	if requestedGroups != nil {
		bindings, _, err := buildAPIKeyGroupBindings(
			opCtx,
			apiKeyGroupBindingMutationDeps{
				groupRepo:   s.groupRepo,
				userRepo:    s.userRepo,
				userSubRepo: s.userSubRepo,
			},
			user,
			apiKey.ID,
			nil,
			*requestedGroups,
			user.IsAdmin(),
		)
		if err != nil {
			return nil, err
		}
		if !user.IsAdmin() {
			if err := s.validateUserAPIKeyModelBindings(opCtx, user, bindings); err != nil {
				return nil, err
			}
		}
		if imageOnlyEnabled {
			bindings, err = s.normalizeImageOnlyGroupBindings(opCtx, bindings)
			if err != nil {
				return nil, err
			}
			if !user.IsAdmin() {
				if err := s.validateUserAPIKeyModelBindings(opCtx, user, bindings); err != nil {
					return nil, err
				}
			}
		}
		pendingGroupBindings = bindings
	}

	if err := s.apiKeyRepo.Create(opCtx, apiKey); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	if requestedGroups != nil {
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

	createdKey, err := s.apiKeyRepo.GetByID(ctx, apiKey.ID)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	s.InvalidateAuthCacheByKey(ctx, createdKey.Key)
	s.compileAPIKeyIPRules(createdKey)

	return createdKey, nil
}

// List 获取用户的API Key列表
