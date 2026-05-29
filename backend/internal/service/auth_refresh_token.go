package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// TokenPair 包含Access Token和Refresh Token
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access Token有效期（秒）
}

// TokenPairWithUser extends TokenPair with user role for backend mode checks
type TokenPairWithUser struct {
	TokenPair
	UserRole string
}

// GenerateTokenPair 生成Access Token和Refresh Token对
// familyID: 可选的Token家族ID，用于Token轮转时保持家族关系
func (s *AuthService) GenerateTokenPair(ctx context.Context, user *User, familyID string) (*TokenPair, error) {
	// 检查 refreshTokenCache 是否可用
	if s.refreshTokenCache == nil {
		return nil, errors.New("refresh token cache not configured")
	}

	// 生成Access Token
	accessToken, err := s.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 生成Refresh Token
	refreshToken, err := s.generateRefreshToken(ctx, user, familyID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.GetAccessTokenExpiresIn(),
	}, nil
}

// generateRefreshToken 生成并存储Refresh Token
func (s *AuthService) generateRefreshToken(ctx context.Context, user *User, familyID string) (string, error) {
	// 生成随机Token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawToken := refreshTokenPrefix + hex.EncodeToString(tokenBytes)

	// 计算Token哈希（存储哈希而非原始Token）
	tokenHash := hashToken(rawToken)

	// 如果没有提供familyID，生成新的
	if familyID == "" {
		familyBytes := make([]byte, 16)
		if _, err := rand.Read(familyBytes); err != nil {
			return "", fmt.Errorf("generate family id: %w", err)
		}
		familyID = hex.EncodeToString(familyBytes)
	}

	now := time.Now()
	ttl := time.Duration(s.cfg.JWT.RefreshTokenExpireDays) * 24 * time.Hour

	data := &RefreshTokenData{
		UserID:       user.ID,
		TokenVersion: user.TokenVersion,
		FamilyID:     familyID,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
	}

	// 存储Token数据
	if err := s.refreshTokenCache.StoreRefreshToken(ctx, tokenHash, data, ttl); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	// 添加到用户Token集合
	if err := s.refreshTokenCache.AddToUserTokenSet(ctx, user.ID, tokenHash, ttl); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to add token to user set: %v", err)
		// 不影响主流程
	}

	// 添加到家族Token集合
	if err := s.refreshTokenCache.AddToFamilyTokenSet(ctx, familyID, tokenHash, ttl); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to add token to family set: %v", err)
		// 不影响主流程
	}

	return rawToken, nil
}

// hashToken 计算Token的SHA256哈希
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
