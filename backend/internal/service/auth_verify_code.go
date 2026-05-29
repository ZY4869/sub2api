package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// SendVerifyCodeResult 发送验证码返回结果
type SendVerifyCodeResult struct {
	Countdown int `json:"countdown"` // 倒计时秒数
}

// SendVerifyCode 发送邮箱验证码（同步方式）
func (s *AuthService) SendVerifyCode(ctx context.Context, email string) error {
	// 检查是否开放注册（默认关闭）
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return ErrRegDisabled
	}

	if isReservedEmail(email) {
		return ErrEmailReserved
	}
	if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
		return err
	}

	// 检查邮箱是否已存在
	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email exists: %v", err)
		return ErrServiceUnavailable
	}
	if existsEmail {
		return ErrEmailExists
	}

	// 发送验证码
	if s.emailService == nil {
		return errors.New("email service not configured")
	}

	// 获取网站名称
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}

	return s.emailService.SendVerifyCode(ctx, email, siteName)
}

// SendVerifyCodeAsync 异步发送邮箱验证码并返回倒计时
func (s *AuthService) SendVerifyCodeAsync(ctx context.Context, email string) (*SendVerifyCodeResult, error) {
	return s.SendVerifyCodeAsyncWithLocale(ctx, email, "")
}

// SendVerifyCodeAsyncWithLocale 异步发送指定语言的邮箱验证码并返回倒计时。
func (s *AuthService) SendVerifyCodeAsyncWithLocale(ctx context.Context, email, locale string) (*SendVerifyCodeResult, error) {
	logger.LegacyPrintf("service.auth", "[Auth] SendVerifyCodeAsync called for email: %s", email)

	// 检查是否开放注册（默认关闭）
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Registration is disabled")
		return nil, ErrRegDisabled
	}

	if isReservedEmail(email) {
		return nil, ErrEmailReserved
	}
	if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
		return nil, err
	}

	// 检查邮箱是否已存在
	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email exists: %v", err)
		return nil, ErrServiceUnavailable
	}
	if existsEmail {
		logger.LegacyPrintf("service.auth", "[Auth] Email already exists: %s", email)
		return nil, ErrEmailExists
	}

	// 检查邮件队列服务是否配置
	if s.emailQueueService == nil {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Email queue service not configured")
		return nil, errors.New("email queue service not configured")
	}

	// 获取网站名称
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}

	// 异步发送
	logger.LegacyPrintf("service.auth", "[Auth] Enqueueing verify code for: %s", email)
	if err := s.emailQueueService.EnqueueVerifyCodeWithLocale(email, siteName, locale); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to enqueue: %v", err)
		return nil, fmt.Errorf("enqueue verify code: %w", err)
	}

	logger.LegacyPrintf("service.auth", "[Auth] Verify code enqueued successfully for: %s", email)
	return &SendVerifyCodeResult{
		Countdown: 60, // 60秒倒计时
	}, nil
}
