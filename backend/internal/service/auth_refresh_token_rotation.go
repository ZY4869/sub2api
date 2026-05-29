package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// RefreshTokenPair 使用Refresh Token刷新Token对
// 实现Token轮转：每次刷新都会生成新的Refresh Token，旧Token立即失效
func (s *AuthService) RefreshTokenPair(ctx context.Context, refreshToken string) (*TokenPairWithUser, error) {
	// 检查 refreshTokenCache 是否可用
	if s.refreshTokenCache == nil {
		return nil, ErrRefreshTokenInvalid
	}

	// 验证Token格式
	if !strings.HasPrefix(refreshToken, refreshTokenPrefix) {
		return nil, ErrRefreshTokenInvalid
	}

	tokenHash := hashToken(refreshToken)

	// 获取Token数据
	data, err := s.refreshTokenCache.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			// Token不存在，可能是已被使用（Token轮转）或已过期
			logger.LegacyPrintf("service.auth", "[Auth] Refresh token not found, possible reuse attack")
			return nil, ErrRefreshTokenInvalid
		}
		logger.LegacyPrintf("service.auth", "[Auth] Error getting refresh token: %v", err)
		return nil, ErrServiceUnavailable
	}

	// 检查Token是否过期
	if time.Now().After(data.ExpiresAt) {
		// 删除过期Token
		_ = s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash)
		return nil, ErrRefreshTokenExpired
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, data.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// 用户已删除，撤销整个Token家族
			_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
			return nil, ErrRefreshTokenInvalid
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error getting user for token refresh: %v", err)
		return nil, ErrServiceUnavailable
	}

	// 检查用户状态
	if !user.IsActive() {
		// 用户被禁用，撤销整个Token家族
		_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
		return nil, ErrUserNotActive
	}

	// 检查TokenVersion（密码更改后所有Token失效）
	if data.TokenVersion != user.TokenVersion {
		// TokenVersion不匹配，撤销整个Token家族
		_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
		return nil, ErrTokenRevoked
	}

	// Token轮转：立即使旧Token失效
	if err := s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to delete old refresh token: %v", err)
		// 继续处理，不影响主流程
	}

	// 生成新的Token对，保持同一个家族ID
	pair, err := s.GenerateTokenPair(ctx, user, data.FamilyID)
	if err != nil {
		return nil, err
	}
	return &TokenPairWithUser{
		TokenPair: *pair,
		UserRole:  user.Role,
	}, nil
}

// RevokeRefreshToken 撤销单个Refresh Token
func (s *AuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if s.refreshTokenCache == nil {
		return nil // No-op if cache not configured
	}
	if !strings.HasPrefix(refreshToken, refreshTokenPrefix) {
		return ErrRefreshTokenInvalid
	}

	tokenHash := hashToken(refreshToken)
	return s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash)
}

// RevokeAllUserSessions 撤销用户的所有会话（所有Refresh Token）
// 用于密码更改或用户主动登出所有设备
func (s *AuthService) RevokeAllUserSessions(ctx context.Context, userID int64) error {
	if s.refreshTokenCache == nil {
		return nil // No-op if cache not configured
	}
	return s.refreshTokenCache.DeleteUserRefreshTokens(ctx, userID)
}
