package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Register 用户注册，返回token和用户
func (s *AuthService) Register(ctx context.Context, email, password string) (string, *User, error) {
	return s.RegisterWithVerification(ctx, email, password, "", "", "", "")
}

// RegisterWithVerification 用户注册（支持邮件验证、优惠码和邀请码），返回token和用户
func (s *AuthService) RegisterWithVerification(ctx context.Context, email, password, verifyCode, promoCode, invitationCode, affCode string) (string, *User, error) {
	// 检查是否开放注册（默认关闭：settingService 未配置时不允许注册）
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return "", nil, ErrRegDisabled
	}

	// 防止用户注册 LinuxDo OAuth 合成邮箱，避免第三方登录与本地账号发生碰撞。
	if isReservedEmail(email) {
		return "", nil, ErrEmailReserved
	}
	if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
		return "", nil, err
	}

	// 检查是否需要邀请码
	var invitationRedeemCode *RedeemCode
	if s.settingService != nil && s.settingService.IsInvitationCodeEnabled(ctx) {
		if invitationCode == "" {
			return "", nil, ErrInvitationCodeRequired
		}
		// 验证邀请码
		redeemCode, err := s.redeemRepo.GetByCode(ctx, invitationCode)
		if err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Invalid invitation code: %s, error: %v", invitationCode, err)
			return "", nil, ErrInvitationCodeInvalid
		}
		// 检查类型和状态
		if redeemCode.Type != RedeemTypeInvitation || redeemCode.Status != StatusUnused {
			logger.LegacyPrintf("service.auth", "[Auth] Invitation code invalid: type=%s, status=%s", redeemCode.Type, redeemCode.Status)
			return "", nil, ErrInvitationCodeInvalid
		}
		invitationRedeemCode = redeemCode
	}

	// 检查是否需要邮件验证
	if s.settingService != nil && s.settingService.IsEmailVerifyEnabled(ctx) {
		// 如果邮件验证已开启但邮件服务未配置，拒绝注册
		// 这是一个配置错误，不应该允许绕过验证
		if s.emailService == nil {
			logger.LegacyPrintf("service.auth", "%s", "[Auth] Email verification enabled but email service not configured, rejecting registration")
			return "", nil, ErrServiceUnavailable
		}
		if verifyCode == "" {
			return "", nil, ErrEmailVerifyRequired
		}
		// 验证邮箱验证码
		if err := s.emailService.VerifyCode(ctx, email, verifyCode); err != nil {
			return "", nil, fmt.Errorf("verify code: %w", err)
		}
	}

	// 检查邮箱是否已存在
	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email exists: %v", err)
		return "", nil, ErrServiceUnavailable
	}
	if existsEmail {
		return "", nil, ErrEmailExists
	}

	// 密码哈希
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return "", nil, fmt.Errorf("hash password: %w", err)
	}

	// 获取默认配置
	defaultBalance := s.cfg.Default.UserBalance
	defaultConcurrency := s.cfg.Default.UserConcurrency
	if s.settingService != nil {
		defaultBalance = s.settingService.GetDefaultBalance(ctx)
		defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
	}

	// 创建用户
	user := &User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         RoleUser,
		Balance:      defaultBalance,
		Concurrency:  defaultConcurrency,
		Status:       StatusActive,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		// 优先检查邮箱冲突错误（竞态条件下可能发生）
		if errors.Is(err, ErrEmailExists) {
			return "", nil, ErrEmailExists
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error creating user: %v", err)
		return "", nil, ErrServiceUnavailable
	}
	s.assignDefaultSubscriptions(ctx, user.ID)

	// 标记邀请码为已使用（如果使用了邀请码）
	if invitationRedeemCode != nil {
		if err := s.redeemRepo.Use(ctx, invitationRedeemCode.ID, user.ID); err != nil {
			// 邀请码标记失败不影响注册，只记录日志
			logger.LegacyPrintf("service.auth", "[Auth] Failed to mark invitation code as used for user %d: %v", user.ID, err)
		}
	}
	// 应用优惠码（如果提供且功能已启用）
	if promoCode != "" && s.promoService != nil && s.settingService != nil && s.settingService.IsPromoCodeEnabled(ctx) {
		if err := s.promoService.ApplyPromoCode(ctx, user.ID, promoCode); err != nil {
			// 优惠码应用失败不影响注册，只记录日志
			logger.LegacyPrintf("service.auth", "[Auth] Failed to apply promo code for user %d: %v", user.ID, err)
		} else {
			// 重新获取用户信息以获取更新后的余额
			if updatedUser, err := s.userRepo.GetByID(ctx, user.ID); err == nil {
				user = updatedUser
			}
		}
	}

	// Best-effort: initialize affiliate row and bind inviter (does not block registration).
	if s.affiliateService != nil {
		if _, err := s.affiliateService.EnsureAffiliateRow(ctx, user.ID); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] EnsureAffiliateRow failed: user_id=%d err=%v", user.ID, err)
		}
		affCode = strings.TrimSpace(affCode)
		if affCode != "" && s.settingService != nil {
			if settings, err := s.settingService.GetAllSettings(ctx); err == nil && settings != nil && settings.AffiliateEnabled {
				s.affiliateService.BindInviterByCode(ctx, user.ID, affCode)
			}
		}
	}

	// 生成token
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	return token, user, nil
}
