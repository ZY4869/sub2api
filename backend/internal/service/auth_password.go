package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用bcrypt加密密码
func (s *AuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPassword 验证密码是否匹配
func (s *AuthService) CheckPassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// IsPasswordResetEnabled 检查是否启用密码重置功能
// 要求：必须同时开启邮件验证且 SMTP 配置正确
func (s *AuthService) IsPasswordResetEnabled(ctx context.Context) bool {
	if s.settingService == nil {
		return false
	}
	// Must have email verification enabled and SMTP configured
	if !s.settingService.IsEmailVerifyEnabled(ctx) {
		return false
	}
	return s.settingService.IsPasswordResetEnabled(ctx)
}

// preparePasswordReset validates the password reset request and returns necessary data
// Returns (siteName, resetURL, shouldProceed)
// shouldProceed is false when we should silently return success (to prevent enumeration)
func (s *AuthService) preparePasswordReset(ctx context.Context, email, frontendBaseURL string) (string, string, bool) {
	// Check if user exists (but don't reveal this to the caller)
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Security: Log but don't reveal that user doesn't exist
			logger.LegacyPrintf("service.auth", "[Auth] Password reset requested for non-existent email: %s", email)
			return "", "", false
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email for password reset: %v", err)
		return "", "", false
	}

	// Check if user is active
	if !user.IsActive() {
		logger.LegacyPrintf("service.auth", "[Auth] Password reset requested for inactive user: %s", email)
		return "", "", false
	}

	// Get site name
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}

	// Build reset URL base
	resetURL := fmt.Sprintf("%s/reset-password", strings.TrimSuffix(frontendBaseURL, "/"))

	return siteName, resetURL, true
}

// RequestPasswordReset 请求密码重置（同步发送）
// Security: Returns the same response regardless of whether the email exists (prevent user enumeration)
func (s *AuthService) RequestPasswordReset(ctx context.Context, email, frontendBaseURL string) error {
	if !s.IsPasswordResetEnabled(ctx) {
		return infraerrors.Forbidden("PASSWORD_RESET_DISABLED", "password reset is not enabled")
	}
	if s.emailService == nil {
		return ErrServiceUnavailable
	}

	siteName, resetURL, shouldProceed := s.preparePasswordReset(ctx, email, frontendBaseURL)
	if !shouldProceed {
		return nil // Silent success to prevent enumeration
	}

	if err := s.emailService.SendPasswordResetEmail(ctx, email, siteName, resetURL); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to send password reset email to %s: %v", email, err)
		return nil // Silent success to prevent enumeration
	}

	logger.LegacyPrintf("service.auth", "[Auth] Password reset email sent to: %s", email)
	return nil
}

// RequestPasswordResetAsync 异步请求密码重置（队列发送）
// Security: Returns the same response regardless of whether the email exists (prevent user enumeration)
func (s *AuthService) RequestPasswordResetAsync(ctx context.Context, email, frontendBaseURL string) error {
	if !s.IsPasswordResetEnabled(ctx) {
		return infraerrors.Forbidden("PASSWORD_RESET_DISABLED", "password reset is not enabled")
	}
	if s.emailQueueService == nil {
		return ErrServiceUnavailable
	}

	siteName, resetURL, shouldProceed := s.preparePasswordReset(ctx, email, frontendBaseURL)
	if !shouldProceed {
		return nil // Silent success to prevent enumeration
	}

	if err := s.emailQueueService.EnqueuePasswordReset(email, siteName, resetURL); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to enqueue password reset email for %s: %v", email, err)
		return nil // Silent success to prevent enumeration
	}

	logger.LegacyPrintf("service.auth", "[Auth] Password reset email enqueued for: %s", email)
	return nil
}

// ResetPassword 重置密码
// Security: Increments TokenVersion to invalidate all existing JWT tokens
func (s *AuthService) ResetPassword(ctx context.Context, email, token, newPassword string) error {
	// Check if password reset is enabled
	if !s.IsPasswordResetEnabled(ctx) {
		return infraerrors.Forbidden("PASSWORD_RESET_DISABLED", "password reset is not enabled")
	}

	if s.emailService == nil {
		return ErrServiceUnavailable
	}

	// Verify and consume the reset token (one-time use)
	if err := s.emailService.ConsumePasswordResetToken(ctx, email, token); err != nil {
		return err
	}

	// Get user
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrInvalidResetToken // Token was valid but user was deleted
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error getting user for password reset: %v", err)
		return ErrServiceUnavailable
	}

	// Check if user is active
	if !user.IsActive() {
		return ErrUserNotActive
	}

	// Hash new password
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update password and increment TokenVersion
	user.PasswordHash = hashedPassword
	user.TokenVersion++ // Invalidate all existing tokens

	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error updating password for user %d: %v", user.ID, err)
		return ErrServiceUnavailable
	}

	// Also revoke all refresh tokens for this user
	if err := s.RevokeAllUserSessions(ctx, user.ID); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to revoke refresh tokens for user %d: %v", user.ID, err)
		// Don't return error - password was already changed successfully
	}

	logger.LegacyPrintf("service.auth", "[Auth] Password reset successful for user: %s", email)
	return nil
}
