package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Login 用户登录，返回JWT token
func (s *AuthService) Login(ctx context.Context, email, password string) (string, *User, error) {
	// 查找用户
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		// 记录数据库错误但不暴露给用户
		logger.LegacyPrintf("service.auth", "[Auth] Database error during login: %v", err)
		return "", nil, ErrServiceUnavailable
	}

	// 验证密码
	if !s.CheckPassword(password, user.PasswordHash) {
		return "", nil, ErrInvalidCredentials
	}

	// 检查用户状态
	if !user.IsActive() {
		return "", nil, ErrUserNotActive
	}

	// 生成JWT token
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	return token, user, nil
}

// LoginOrRegisterOAuth 用于第三方 OAuth/SSO 登录：
// - 如果邮箱已存在：直接登录（不需要本地密码）
// - 如果邮箱不存在：创建新用户并登录
//
// 注意：该函数用于 LinuxDo OAuth 登录场景（不同于上游账号的 OAuth，例如 Claude/OpenAI/Gemini）。
// 为了满足现有数据库约束（需要密码哈希），新用户会生成随机密码并进行哈希保存。
func (s *AuthService) LoginOrRegisterOAuth(ctx context.Context, email, username string) (string, *User, error) {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return "", nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}

	username = strings.TrimSpace(username)
	if len([]rune(username)) > 100 {
		username = string([]rune(username)[:100])
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// OAuth 首次登录视为注册（fail-close：settingService 未配置时不允许注册）
			if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
				return "", nil, ErrRegDisabled
			}

			randomPassword, err := randomHexString(32)
			if err != nil {
				logger.LegacyPrintf("service.auth", "[Auth] Failed to generate random password for oauth signup: %v", err)
				return "", nil, ErrServiceUnavailable
			}
			hashedPassword, err := s.HashPassword(randomPassword)
			if err != nil {
				return "", nil, fmt.Errorf("hash password: %w", err)
			}

			// 新用户默认值。
			defaultBalance := s.cfg.Default.UserBalance
			defaultConcurrency := s.cfg.Default.UserConcurrency
			defaultAPIKeyModelBindingMode := APIKeyModelBindingModeGroupAllowed
			if s.settingService != nil {
				defaultBalance = s.settingService.GetDefaultBalance(ctx)
				defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
				defaultAPIKeyModelBindingMode = s.settingService.GetDefaultAPIKeyModelBindingMode(ctx)
			}

			newUser := &User{
				Email:                  email,
				Username:               username,
				PasswordHash:           hashedPassword,
				Role:                   RoleUser,
				Balance:                defaultBalance,
				Concurrency:            defaultConcurrency,
				Status:                 StatusActive,
				APIKeyModelBindingMode: defaultAPIKeyModelBindingMode,
			}

			if createErr := s.userRepo.Create(ctx, newUser); createErr != nil {
				if errors.Is(createErr, ErrEmailExists) {
					// 并发场景：GetByEmail 与 Create 之间用户被创建。
					conflictUser, lookupErr := s.userRepo.GetByEmail(ctx, email)
					if lookupErr != nil {
						logger.LegacyPrintf("service.auth", "[Auth] Database error getting user after oauth create conflict: create_err=%v lookup_err=%v", createErr, lookupErr)
						return "", nil, oauthCreateConflictRecoveryError(createErr, lookupErr)
					}
					user = conflictUser
				} else {
					logger.LegacyPrintf("service.auth", "[Auth] Database error creating oauth user: %v", createErr)
					return "", nil, ErrServiceUnavailable
				}
			} else {
				user = newUser
				s.assignDefaultSubscriptions(ctx, user.ID)
			}
		} else {
			logger.LegacyPrintf("service.auth", "[Auth] Database error during oauth login: %v", err)
			return "", nil, ErrServiceUnavailable
		}
	}

	if !user.IsActive() {
		return "", nil, ErrUserNotActive
	}

	// 尽力补全：当用户名为空时，使用第三方返回的用户名回填。
	if user.Username == "" && username != "" {
		user.Username = username
		if err := s.userRepo.Update(ctx, user); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Failed to update username after oauth login: %v", err)
		}
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	return token, user, nil
}
