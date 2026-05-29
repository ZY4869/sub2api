package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// VerifyTurnstileForRegister 在注册场景下验证 Turnstile。
// 当邮箱验证开启且已提交验证码时，说明验证码发送阶段已完成 Turnstile 校验，
// 此处跳过二次校验，避免一次性 token 在注册提交时重复使用导致误报失败。
func (s *AuthService) VerifyTurnstileForRegister(ctx context.Context, token, remoteIP, verifyCode string) error {
	if s.IsEmailVerifyEnabled(ctx) && strings.TrimSpace(verifyCode) != "" {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Email verify flow detected, skip duplicate Turnstile check on register")
		return nil
	}
	return s.VerifyTurnstile(ctx, token, remoteIP)
}

// VerifyTurnstile 验证Turnstile token
func (s *AuthService) VerifyTurnstile(ctx context.Context, token string, remoteIP string) error {
	required := s.cfg != nil && s.cfg.Server.Mode == "release" && s.cfg.Turnstile.Required

	if required {
		if s.settingService == nil {
			logger.LegacyPrintf("service.auth", "%s", "[Auth] Turnstile required but settings service is not configured")
			return ErrTurnstileNotConfigured
		}
		enabled := s.settingService.IsTurnstileEnabled(ctx)
		secretConfigured := s.settingService.GetTurnstileSecretKey(ctx) != ""
		if !enabled || !secretConfigured {
			logger.LegacyPrintf("service.auth", "[Auth] Turnstile required but not configured (enabled=%v, secret_configured=%v)", enabled, secretConfigured)
			return ErrTurnstileNotConfigured
		}
	}

	if s.turnstileService == nil {
		if required {
			logger.LegacyPrintf("service.auth", "%s", "[Auth] Turnstile required but service not configured")
			return ErrTurnstileNotConfigured
		}
		return nil // 服务未配置则跳过验证
	}

	if !required && s.settingService != nil && s.settingService.IsTurnstileEnabled(ctx) && s.settingService.GetTurnstileSecretKey(ctx) == "" {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Turnstile enabled but secret key not configured")
	}

	return s.turnstileService.VerifyToken(ctx, token, remoteIP)
}

// IsTurnstileEnabled 检查是否启用Turnstile验证
func (s *AuthService) IsTurnstileEnabled(ctx context.Context) bool {
	if s.turnstileService == nil {
		return false
	}
	return s.turnstileService.IsEnabled(ctx)
}

// IsRegistrationEnabled 检查是否开放注册
func (s *AuthService) IsRegistrationEnabled(ctx context.Context) bool {
	if s.settingService == nil {
		return false // 安全默认：settingService 未配置时关闭注册
	}
	return s.settingService.IsRegistrationEnabled(ctx)
}

// IsEmailVerifyEnabled 检查是否开启邮件验证
func (s *AuthService) IsEmailVerifyEnabled(ctx context.Context) bool {
	if s.settingService == nil {
		return false
	}
	return s.settingService.IsEmailVerifyEnabled(ctx)
}
