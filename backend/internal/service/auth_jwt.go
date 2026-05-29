package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateToken 验证JWT token并返回用户声明
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// 先做长度校验，尽早拒绝异常超长 token，降低 DoS 风险。
	if len(tokenString) > maxTokenLength {
		return nil, ErrTokenTooLarge
	}

	// 使用解析器并限制可接受的签名算法，防止算法混淆。
	parser := jwt.NewParser(jwt.WithValidMethods([]string{
		jwt.SigningMethodHS256.Name,
		jwt.SigningMethodHS384.Name,
		jwt.SigningMethodHS512.Name,
	}))

	// 保留默认 claims 校验（exp/nbf），避免放行过期或未生效的 token。
	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// token 过期但仍返回 claims（用于 RefreshToken 等场景）
			// jwt-go 在解析时即使遇到过期错误，token.Claims 仍会被填充
			if token == nil || token.Claims == nil {
				return nil, ErrInvalidToken
			}
			if claims, ok := token.Claims.(*JWTClaims); ok {
				return claims, ErrTokenExpired
			}
			return nil, ErrInvalidToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func randomHexString(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 16
	}
	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// GenerateToken 生成JWT access token
// 使用新的access_token_expire_minutes配置项（如果配置了），否则回退到expire_hour
func (s *AuthService) GenerateToken(user *User) (string, error) {
	now := time.Now()
	var expiresAt time.Time
	if s.cfg.JWT.AccessTokenExpireMinutes > 0 {
		expiresAt = now.Add(time.Duration(s.cfg.JWT.AccessTokenExpireMinutes) * time.Minute)
	} else {
		// 向后兼容：使用旧的expire_hour配置
		expiresAt = now.Add(time.Duration(s.cfg.JWT.ExpireHour) * time.Hour)
	}

	claims := &JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// GetAccessTokenExpiresIn 返回Access Token的有效期（秒）
// 用于前端设置刷新定时器
func (s *AuthService) GetAccessTokenExpiresIn() int {
	if s.cfg.JWT.AccessTokenExpireMinutes > 0 {
		return s.cfg.JWT.AccessTokenExpireMinutes * 60
	}
	return s.cfg.JWT.ExpireHour * 3600
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(ctx context.Context, oldTokenString string) (string, error) {
	// 验证旧token（即使过期也允许，用于刷新）
	claims, err := s.ValidateToken(oldTokenString)
	if err != nil && !errors.Is(err, ErrTokenExpired) {
		return "", err
	}
	if claims == nil {
		return "", ErrInvalidToken
	}

	// 获取最新的用户信息
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrInvalidToken
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error refreshing token: %v", err)
		return "", ErrServiceUnavailable
	}

	// 检查用户状态
	if !user.IsActive() {
		return "", ErrUserNotActive
	}

	// Security: Check TokenVersion to prevent refreshing revoked tokens
	// This ensures tokens issued before a password change cannot be refreshed
	if claims.TokenVersion != user.TokenVersion {
		return "", ErrTokenRevoked
	}

	// 生成新token
	return s.GenerateToken(user)
}
