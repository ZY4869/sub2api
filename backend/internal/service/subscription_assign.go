package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// AssignSubscriptionInput 分配订阅输入
type AssignSubscriptionInput struct {
	UserID       int64
	GroupID      int64
	ValidityDays int
	AssignedBy   int64
	Notes        string
}

type userSubscriptionForUpdateGetter interface {
	GetByUserIDAndGroupIDForUpdate(ctx context.Context, userID, groupID int64) (*UserSubscription, error)
}

// AssignSubscription 分配订阅给用户（不允许重复分配）
func (s *SubscriptionService) AssignSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	sub, _, err := s.assignSubscriptionWithReuse(ctx, input)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// AssignOrExtendSubscription 分配或续期订阅（用于兑换码等场景）
// 如果用户已有同分组的订阅：
//   - 未过期：从当前过期时间累加天数
//   - 已过期：从当前时间开始计算新的过期时间，并激活订阅
//
// 如果没有订阅：创建新订阅
func (s *SubscriptionService) AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	if s.entClient != nil {
		return s.assignOrExtendSubscriptionWithRowLock(ctx, input, validityDays)
	}

	// 查询是否已有订阅
	existingSub, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		// 不存在记录是正常情况，其他错误需要返回
		existingSub = nil
	}

	// 已有订阅，兼容路径执行续期；生产路径在上方使用事务内行锁。
	if existingSub != nil {
		now := time.Now()
		var newExpiresAt time.Time

		if existingSub.ExpiresAt.After(now) {
			// 未过期：从当前过期时间累加
			newExpiresAt = existingSub.ExpiresAt.AddDate(0, 0, validityDays)
		} else {
			// 已过期：从当前时间开始计算
			newExpiresAt = now.AddDate(0, 0, validityDays)
		}

		// 确保不超过最大过期时间
		if newExpiresAt.After(MaxExpiresAt) {
			newExpiresAt = MaxExpiresAt
		}

		// 更新过期时间
		if err := s.userSubRepo.ExtendExpiry(ctx, existingSub.ID, newExpiresAt); err != nil {
			return nil, false, fmt.Errorf("extend subscription: %w", err)
		}

		// 如果订阅已过期或被暂停，恢复为active状态
		if existingSub.Status != SubscriptionStatusActive {
			if err := s.userSubRepo.UpdateStatus(ctx, existingSub.ID, SubscriptionStatusActive); err != nil {
				return nil, false, fmt.Errorf("update subscription status: %w", err)
			}
		}

		// 追加备注
		if input.Notes != "" {
			newNotes := existingSub.Notes
			if newNotes != "" {
				newNotes += "\n"
			}
			newNotes += input.Notes
			if err := s.userSubRepo.UpdateNotes(ctx, existingSub.ID, newNotes); err != nil {
				return nil, false, fmt.Errorf("update subscription notes: %w", err)
			}
		}

		// 失效订阅缓存
		s.InvalidateSubCache(input.UserID, input.GroupID)
		if s.billingCacheService != nil {
			userID, groupID := input.UserID, input.GroupID
			go func() {
				cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
			}()
		}

		// 返回更新后的订阅
		sub, err := s.userSubRepo.GetByID(ctx, existingSub.ID)
		return sub, true, err // true 表示是续期
	}

	// 没有订阅，创建新订阅
	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.InvalidateSubCache(input.UserID, input.GroupID)
	if s.billingCacheService != nil {
		userID, groupID := input.UserID, input.GroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return sub, false, nil // false 表示是新建
}

func (s *SubscriptionService) assignOrExtendSubscriptionWithRowLock(ctx context.Context, input *AssignSubscriptionInput, validityDays int) (*UserSubscription, bool, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	txCtx := dbent.NewTxContext(ctx, tx)

	existingSub, err := getUserSubscriptionForRenewal(txCtx, s.userSubRepo, input.UserID, input.GroupID)
	if err != nil && !errors.Is(err, ErrSubscriptionNotFound) {
		return nil, false, err
	}

	if existingSub == nil {
		sub, err := s.createSubscription(txCtx, input)
		if err != nil {
			return nil, false, err
		}
		if err := tx.Commit(); err != nil {
			return nil, false, fmt.Errorf("commit transaction: %w", err)
		}
		tx = nil
		s.invalidateSubscriptionRuntimeCaches(input.UserID, input.GroupID)
		return sub, false, nil
	}

	now := time.Now()
	var newExpiresAt time.Time
	if existingSub.ExpiresAt.After(now) {
		newExpiresAt = existingSub.ExpiresAt.AddDate(0, 0, validityDays)
	} else {
		newExpiresAt = now.AddDate(0, 0, validityDays)
	}
	if newExpiresAt.After(MaxExpiresAt) {
		newExpiresAt = MaxExpiresAt
	}

	if err := s.userSubRepo.ExtendExpiry(txCtx, existingSub.ID, newExpiresAt); err != nil {
		return nil, false, fmt.Errorf("extend subscription: %w", err)
	}
	if existingSub.Status != SubscriptionStatusActive {
		if err := s.userSubRepo.UpdateStatus(txCtx, existingSub.ID, SubscriptionStatusActive); err != nil {
			return nil, false, fmt.Errorf("update subscription status: %w", err)
		}
	}
	if input.Notes != "" {
		newNotes := existingSub.Notes
		if newNotes != "" {
			newNotes += "\n"
		}
		newNotes += input.Notes
		if err := s.userSubRepo.UpdateNotes(txCtx, existingSub.ID, newNotes); err != nil {
			return nil, false, fmt.Errorf("update subscription notes: %w", err)
		}
	}
	sub, err := s.userSubRepo.GetByID(txCtx, existingSub.ID)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit transaction: %w", err)
	}
	tx = nil
	s.invalidateSubscriptionRuntimeCaches(input.UserID, input.GroupID)
	return sub, true, nil
}

func getUserSubscriptionForRenewal(ctx context.Context, repo UserSubscriptionRepository, userID, groupID int64) (*UserSubscription, error) {
	if repo == nil {
		return nil, ErrSubscriptionNotFound
	}
	if locker, ok := repo.(userSubscriptionForUpdateGetter); ok {
		return locker.GetByUserIDAndGroupIDForUpdate(ctx, userID, groupID)
	}
	return repo.GetByUserIDAndGroupID(ctx, userID, groupID)
}

// createSubscription 创建新订阅（内部方法）
func (s *SubscriptionService) createSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, validityDays)
	if expiresAt.After(MaxExpiresAt) {
		expiresAt = MaxExpiresAt
	}

	sub := &UserSubscription{
		UserID:     input.UserID,
		GroupID:    input.GroupID,
		StartsAt:   now,
		ExpiresAt:  expiresAt,
		Status:     SubscriptionStatusActive,
		AssignedAt: now,
		Notes:      input.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	// 只有当 AssignedBy > 0 时才设置（0 表示系统分配，如兑换码）
	if input.AssignedBy > 0 {
		sub.AssignedBy = &input.AssignedBy
	}

	if err := s.userSubRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	// 重新获取完整订阅信息（包含关联）
	return s.userSubRepo.GetByID(ctx, sub.ID)
}

func (s *SubscriptionService) assignSubscriptionWithReuse(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	// 检查是否已存在订阅；若已存在，则按幂等成功返回现有订阅
	exists, err := s.userSubRepo.ExistsByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		return nil, false, err
	}
	if exists {
		sub, getErr := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
		if getErr != nil {
			return nil, false, getErr
		}
		if conflictReason, conflict := detectAssignSemanticConflict(sub, input); conflict {
			return nil, false, ErrSubscriptionAssignConflict.WithMetadata(map[string]string{
				"conflict_reason": conflictReason,
			})
		}
		return sub, true, nil
	}

	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.InvalidateSubCache(input.UserID, input.GroupID)
	if s.billingCacheService != nil {
		userID, groupID := input.UserID, input.GroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return sub, false, nil
}

func detectAssignSemanticConflict(existing *UserSubscription, input *AssignSubscriptionInput) (string, bool) {
	if existing == nil || input == nil {
		return "", false
	}

	normalizedDays := normalizeAssignValidityDays(input.ValidityDays)
	if !existing.StartsAt.IsZero() {
		expectedExpiresAt := existing.StartsAt.AddDate(0, 0, normalizedDays)
		if expectedExpiresAt.After(MaxExpiresAt) {
			expectedExpiresAt = MaxExpiresAt
		}
		if !existing.ExpiresAt.Equal(expectedExpiresAt) {
			return "validity_days_mismatch", true
		}
	}

	existingNotes := strings.TrimSpace(existing.Notes)
	inputNotes := strings.TrimSpace(input.Notes)
	if existingNotes != inputNotes {
		return "notes_mismatch", true
	}

	return "", false
}

func normalizeAssignValidityDays(days int) int {
	if days <= 0 {
		days = 30
	}
	if days > MaxValidityDays {
		days = MaxValidityDays
	}
	return days
}
