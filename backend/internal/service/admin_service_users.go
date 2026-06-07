package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"

	"go.uber.org/zap"
)

func (s *adminServiceImpl) ListUsers(ctx context.Context, page, pageSize int, filters UserListFilters) ([]User, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	users, result, err := s.userRepo.ListWithFilters(ctx, params, filters)
	if err != nil {
		return nil, 0, err
	}
	if s.userGroupRateRepo != nil && len(users) > 0 {
		if batchRepo, ok := s.userGroupRateRepo.(userGroupRateBatchReader); ok {
			userIDs := make([]int64, 0, len(users))
			for i := range users {
				userIDs = append(userIDs, users[i].ID)
			}
			ratesByUser, err := batchRepo.GetByUserIDs(ctx, userIDs)
			if err != nil {
				logger.LegacyPrintf("service.admin", "failed to load user group rates in batch: err=%v", err)
				s.loadUserGroupRatesOneByOne(ctx, users)
			} else {
				for i := range users {
					if rates, ok := ratesByUser[users[i].ID]; ok {
						users[i].GroupRates = rates
					}
				}
			}
		} else {
			s.loadUserGroupRatesOneByOne(ctx, users)
		}
	}
	return users, result.Total, nil
}
func (s *adminServiceImpl) loadUserGroupRatesOneByOne(ctx context.Context, users []User) {
	if s.userGroupRateRepo == nil {
		return
	}
	for i := range users {
		rates, err := s.userGroupRateRepo.GetByUserID(ctx, users[i].ID)
		if err != nil {
			logger.LegacyPrintf("service.admin", "failed to load user group rates: user_id=%d err=%v", users[i].ID, err)
			continue
		}
		users[i].GroupRates = rates
	}
}
func (s *adminServiceImpl) GetUser(ctx context.Context, id int64) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.userGroupRateRepo != nil {
		rates, err := s.userGroupRateRepo.GetByUserID(ctx, id)
		if err != nil {
			logger.LegacyPrintf("service.admin", "failed to load user group rates: user_id=%d err=%v", id, err)
		} else {
			user.GroupRates = rates
		}
	}
	return user, nil
}
func (s *adminServiceImpl) CreateUser(ctx context.Context, input *CreateUserInput) (*User, error) {
	policy, err := NormalizeTimeAccessPolicy(input.APIKeyAccessTimePolicy)
	if err != nil {
		return nil, timeAccessPolicyInputError(err)
	}
	user := &User{Email: input.Email, Username: input.Username, Notes: input.Notes, Role: RoleUser, Balance: input.Balance, Concurrency: input.Concurrency, Status: StatusActive, AllowedGroups: input.AllowedGroups, APIKeyModelBindingMode: NormalizeAPIKeyModelBindingMode(input.APIKeyModelBindingMode), APIKeyAccessTimePolicy: policy}
	if err := user.SetPassword(input.Password); err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	s.assignDefaultSubscriptions(ctx, user.ID)
	return user, nil
}
func (s *adminServiceImpl) assignDefaultSubscriptions(ctx context.Context, userID int64) {
	if s.settingService == nil || s.defaultSubAssigner == nil || userID <= 0 {
		return
	}
	items := s.settingService.GetDefaultSubscriptions(ctx)
	for _, item := range items {
		if _, _, err := s.defaultSubAssigner.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{UserID: userID, GroupID: item.GroupID, ValidityDays: item.ValidityDays, Notes: "auto assigned by default user subscriptions setting"}); err != nil {
			logger.LegacyPrintf("service.admin", "failed to assign default subscription: user_id=%d group_id=%d err=%v", userID, item.GroupID, err)
		}
	}
}
func (s *adminServiceImpl) UpdateUser(ctx context.Context, id int64, input *UpdateUserInput) (*User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.Role == "admin" && input.Status == "disabled" {
		return nil, errors.New("cannot disable admin user")
	}
	oldConcurrency := user.Concurrency
	oldStatus := user.Status
	oldRole := user.Role
	oldAdminFreeBilling := user.AdminFreeBilling
	oldRequestDetailsReview := user.RequestDetailsReview
	oldAPIKeyModelBindingMode := user.EffectiveAPIKeyModelBindingMode()
	oldAPIKeyAccessTimePolicy := user.APIKeyAccessTimePolicy
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Password != "" {
		if err := user.SetPassword(input.Password); err != nil {
			return nil, err
		}
	}
	if input.Username != nil {
		user.Username = *input.Username
	}
	if input.Notes != nil {
		user.Notes = *input.Notes
	}
	if input.Status != "" {
		user.Status = input.Status
	}
	if input.Concurrency != nil {
		user.Concurrency = *input.Concurrency
	}
	if input.AllowedGroups != nil {
		user.AllowedGroups = *input.AllowedGroups
	}
	if input.AdminFreeBilling != nil {
		user.AdminFreeBilling = user.Role == RoleAdmin && *input.AdminFreeBilling
	}
	if input.RequestDetailsReview != nil {
		if user.Role == RoleAdmin {
			user.RequestDetailsReview = false
		} else {
			user.RequestDetailsReview = *input.RequestDetailsReview
		}
	}
	if input.APIKeyModelBindingMode != nil {
		user.APIKeyModelBindingMode = NormalizeAPIKeyModelBindingMode(*input.APIKeyModelBindingMode)
	}
	if input.ClearAPIKeyAccessTimePolicy {
		user.APIKeyAccessTimePolicy = nil
	} else if input.APIKeyAccessTimePolicy != nil {
		policy, err := NormalizeTimeAccessPolicy(input.APIKeyAccessTimePolicy)
		if err != nil {
			return nil, timeAccessPolicyInputError(err)
		}
		user.APIKeyAccessTimePolicy = policy
	}
	if user.Role != RoleAdmin {
		user.AdminFreeBilling = false
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	if input.GroupRates != nil && s.userGroupRateRepo != nil {
		if err := s.userGroupRateRepo.SyncUserGroupRates(ctx, user.ID, input.GroupRates); err != nil {
			logger.LegacyPrintf("service.admin", "failed to sync user group rates: user_id=%d err=%v", user.ID, err)
		}
	}
	if s.authCacheInvalidator != nil {
		if user.Concurrency != oldConcurrency || user.Status != oldStatus || user.Role != oldRole || user.AdminFreeBilling != oldAdminFreeBilling || user.RequestDetailsReview != oldRequestDetailsReview || user.EffectiveAPIKeyModelBindingMode() != oldAPIKeyModelBindingMode || !timeAccessPoliciesEqual(user.APIKeyAccessTimePolicy, oldAPIKeyAccessTimePolicy) {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, user.ID)
		}
	}
	if user.AdminFreeBilling != oldAdminFreeBilling {
		logger.With(
			zap.String("component", "audit.admin_free_billing"),
			zap.Int64("user_id", user.ID),
			zap.String("role", user.Role),
			zap.Bool("before", oldAdminFreeBilling),
			zap.Bool("after", user.AdminFreeBilling),
		).Info("admin free billing updated")
	}
	if user.RequestDetailsReview != oldRequestDetailsReview {
		logger.With(
			zap.String("component", "audit.request_details_review"),
			zap.Int64("user_id", user.ID),
			zap.String("role", user.Role),
			zap.Bool("before", oldRequestDetailsReview),
			zap.Bool("after", user.RequestDetailsReview),
		).Info("request details review updated")
	}
	if user.EffectiveAPIKeyModelBindingMode() != oldAPIKeyModelBindingMode {
		logger.With(
			zap.String("component", "audit.api_key_model_binding_mode"),
			zap.Int64("user_id", user.ID),
			zap.String("before", oldAPIKeyModelBindingMode),
			zap.String("after", user.EffectiveAPIKeyModelBindingMode()),
		).Info("api key model binding mode updated")
	}
	if !timeAccessPoliciesEqual(user.APIKeyAccessTimePolicy, oldAPIKeyAccessTimePolicy) {
		logger.With(
			zap.String("component", "audit.api_key_access_time_policy"),
			zap.Int64("user_id", user.ID),
		).Info("api key access time policy updated")
	}
	concurrencyDiff := user.Concurrency - oldConcurrency
	if concurrencyDiff != 0 {
		code, err := GenerateRedeemCode()
		if err != nil {
			logger.LegacyPrintf("service.admin", "failed to generate adjustment redeem code: %v", err)
			return user, nil
		}
		adjustmentRecord := &RedeemCode{Code: code, Type: AdjustmentTypeAdminConcurrency, Value: float64(concurrencyDiff), Status: StatusUsed, UsedBy: &user.ID}
		now := time.Now()
		adjustmentRecord.UsedAt = &now
		if err := s.redeemCodeRepo.Create(ctx, adjustmentRecord); err != nil {
			logger.LegacyPrintf("service.admin", "failed to create concurrency adjustment redeem code: %v", err)
		}
	}
	return user, nil
}
func (s *adminServiceImpl) DeleteUser(ctx context.Context, id int64) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user.Role == "admin" {
		return errors.New("cannot delete admin user")
	}
	deletedKeys, err := s.deleteUserAndAPIKeys(ctx, id)
	if err != nil {
		logger.LegacyPrintf("service.admin", "delete user failed: user_id=%d err=%v", id, err)
		return err
	}
	if s.authCacheInvalidator != nil {
		for _, key := range deletedKeys {
			s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, key)
		}
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, id)
	}
	return nil
}

type apiKeyUserDeleter interface {
	DeleteByUserID(ctx context.Context, userID int64) ([]string, error)
}

func (s *adminServiceImpl) deleteUserAndAPIKeys(ctx context.Context, userID int64) ([]string, error) {
	if s.entClient == nil {
		return s.deleteUserAndAPIKeysWithoutTx(ctx, userID)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	opCtx := dbent.NewTxContext(ctx, tx)
	deletedKeys, err := s.deleteUserAPIKeys(opCtx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.userRepo.Delete(opCtx, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return deletedKeys, nil
}

func (s *adminServiceImpl) deleteUserAndAPIKeysWithoutTx(ctx context.Context, userID int64) ([]string, error) {
	deletedKeys, err := s.deleteUserAPIKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return nil, err
	}
	return deletedKeys, nil
}

func (s *adminServiceImpl) deleteUserAPIKeys(ctx context.Context, userID int64) ([]string, error) {
	if deleter, ok := s.apiKeyRepo.(apiKeyUserDeleter); ok {
		keys, err := deleter.DeleteByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("delete user api keys: %w", err)
		}
		return keys, nil
	}
	if s.apiKeyRepo == nil {
		return nil, nil
	}

	deletedKeys := make([]string, 0)
	for {
		keys, _, err := s.apiKeyRepo.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, APIKeyListFilters{})
		if err != nil {
			return nil, fmt.Errorf("list user api keys: %w", err)
		}
		if len(keys) == 0 {
			return deletedKeys, nil
		}
		for _, key := range keys {
			if err := s.apiKeyRepo.Delete(ctx, key.ID); err != nil {
				return nil, fmt.Errorf("delete user api key: %w", err)
			}
			deletedKeys = append(deletedKeys, key.Key)
		}
	}
}
func (s *adminServiceImpl) UpdateUserBalance(ctx context.Context, userID int64, balance float64, operation string, notes string) (*User, error) {
	balance, err := NormalizeAndValidateBillingAmount(balance)
	if err != nil {
		return nil, err
	}
	balanceMoney, err := NewBillingMoneyFromFloat(balance)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	oldBalance := user.Balance
	oldBalanceMoney, err := NewBillingMoneyFromFloat(oldBalance)
	if err != nil {
		return nil, err
	}
	newBalanceMoney := oldBalanceMoney
	switch operation {
	case "set":
		newBalanceMoney = balanceMoney
	case "add":
		newBalanceMoney, err = oldBalanceMoney.Add(balanceMoney)
	case "subtract":
		newBalanceMoney, err = oldBalanceMoney.Sub(balanceMoney)
	}
	if err != nil {
		return nil, err
	}
	if newBalanceMoney.IsNegative() {
		return nil, fmt.Errorf("balance cannot be negative, current balance: %.2f, requested operation would result in: %.2f", oldBalance, newBalanceMoney.Float64())
	}
	user.Balance, err = NormalizeAndValidateNonNegativeBillingAmount(newBalanceMoney.Float64())
	if err != nil {
		return nil, err
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	newBalanceMoney, err = NewNonNegativeBillingMoneyFromFloat(user.Balance)
	if err != nil {
		return nil, err
	}
	balanceDiffMoney, err := newBalanceMoney.Sub(oldBalanceMoney)
	if err != nil {
		return nil, err
	}
	balanceDiff := balanceDiffMoney.Float64()
	if s.authCacheInvalidator != nil && !balanceDiffMoney.IsZero() {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, userID); err != nil {
				logger.LegacyPrintf("service.admin", "invalidate user balance cache failed: user_id=%d err=%v", userID, err)
			}
		}()
	}
	if !balanceDiffMoney.IsZero() {
		code, err := GenerateRedeemCode()
		if err != nil {
			logger.LegacyPrintf("service.admin", "failed to generate adjustment redeem code: %v", err)
			return user, nil
		}
		adjustmentRecord := &RedeemCode{Code: code, Type: AdjustmentTypeAdminBalance, Value: balanceDiff, Status: StatusUsed, UsedBy: &user.ID, Notes: notes}
		now := time.Now()
		adjustmentRecord.UsedAt = &now
		createErr := s.redeemCodeRepo.Create(ctx, adjustmentRecord)
		if createErr != nil {
			logger.LegacyPrintf("service.admin", "failed to create balance adjustment redeem code: %v", createErr)
		}
		if createErr == nil && balanceDiffMoney.IsPositive() && s.affiliateService != nil && adjustmentRecord.ID > 0 {
			s.affiliateService.AccrueTopupRebateBestEffort(ctx, adjustmentRecord.ID, userID, balanceDiff)
		}
	}
	return user, nil
}
func (s *adminServiceImpl) GetUserAPIKeys(ctx context.Context, userID int64, page, pageSize int) ([]APIKey, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	keys, result, err := s.apiKeyRepo.ListByUserID(ctx, userID, params, APIKeyListFilters{})
	if err != nil {
		return nil, 0, err
	}
	return keys, result.Total, nil
}
func (s *adminServiceImpl) GetUserUsageStats(ctx context.Context, userID int64, period string) (any, error) {
	return map[string]any{"period": period, "total_requests": 0, "total_cost": 0.0, "total_tokens": 0, "avg_duration_ms": 0}, nil
}
func (s *adminServiceImpl) GetUserBalanceHistory(ctx context.Context, userID int64, page, pageSize int, codeType string) ([]RedeemCode, int64, float64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	codes, result, err := s.redeemCodeRepo.ListByUserPaginated(ctx, userID, params, codeType)
	if err != nil {
		return nil, 0, 0, err
	}
	totalRecharged, err := s.redeemCodeRepo.SumPositiveBalanceByUser(ctx, userID)
	if err != nil {
		return nil, 0, 0, err
	}
	return codes, result.Total, totalRecharged, nil
}
