package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// LoginOrRegisterOAuthWithTokenPair 用于第三方 OAuth/SSO 登录，返回完整的 TokenPair。
// 与 LoginOrRegisterOAuth 功能相同，但返回 TokenPair 而非单个 token。
// invitationCode 仅在邀请码注册模式下新用户注册时使用；已有账号登录时忽略。
func (s *AuthService) LoginOrRegisterOAuthWithTokenPair(ctx context.Context, email, username, invitationCode, affCode string) (*TokenPair, *User, error) {
	// 检查 refreshTokenCache 是否可用
	if s.refreshTokenCache == nil {
		return nil, nil, errors.New("refresh token cache not configured")
	}

	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return nil, nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}

	username = strings.TrimSpace(username)
	if len([]rune(username)) > 100 {
		username = string([]rune(username)[:100])
	}

	createdUser := false

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// OAuth 首次登录视为注册
			if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
				return nil, nil, ErrRegDisabled
			}
			if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
				return nil, nil, err
			}

			// 检查是否需要邀请码
			var invitationRedeemCode *RedeemCode
			if s.settingService != nil && s.settingService.IsInvitationCodeEnabled(ctx) {
				if invitationCode == "" {
					return nil, nil, ErrOAuthInvitationRequired
				}
				redeemCode, err := s.redeemRepo.GetByCode(ctx, invitationCode)
				if err != nil {
					return nil, nil, ErrInvitationCodeInvalid
				}
				if redeemCode.Type != RedeemTypeInvitation || redeemCode.Status != StatusUnused {
					return nil, nil, ErrInvitationCodeInvalid
				}
				invitationRedeemCode = redeemCode
			}

			randomPassword, err := randomHexString(32)
			if err != nil {
				logger.LegacyPrintf("service.auth", "[Auth] Failed to generate random password for oauth signup: %v", err)
				return nil, nil, ErrServiceUnavailable
			}
			hashedPassword, err := s.HashPassword(randomPassword)
			if err != nil {
				return nil, nil, fmt.Errorf("hash password: %w", err)
			}

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

			if s.entClient != nil && invitationRedeemCode != nil {
				tx, err := s.entClient.Tx(ctx)
				if err != nil {
					logger.LegacyPrintf("service.auth", "[Auth] Failed to begin transaction for oauth registration: %v", err)
					return nil, nil, ErrServiceUnavailable
				}
				defer func() { _ = tx.Rollback() }()
				txCtx := dbent.NewTxContext(ctx, tx)

				if createErr := s.userRepo.Create(txCtx, newUser); createErr != nil {
					if errors.Is(createErr, ErrEmailExists) {
						conflictUser, lookupErr := s.userRepo.GetByEmail(ctx, email)
						if lookupErr != nil {
							logger.LegacyPrintf("service.auth", "[Auth] Database error getting user after oauth create conflict: create_err=%v lookup_err=%v", createErr, lookupErr)
							return nil, nil, oauthCreateConflictRecoveryError(createErr, lookupErr)
						}
						user = conflictUser
					} else {
						logger.LegacyPrintf("service.auth", "[Auth] Database error creating oauth user: %v", createErr)
						return nil, nil, ErrServiceUnavailable
					}
				} else {
					if err := s.redeemRepo.Use(txCtx, invitationRedeemCode.ID, newUser.ID); err != nil {
						return nil, nil, ErrInvitationCodeInvalid
					}
					if err := tx.Commit(); err != nil {
						logger.LegacyPrintf("service.auth", "[Auth] Failed to commit oauth registration transaction: %v", err)
						return nil, nil, ErrServiceUnavailable
					}
					user = newUser
					createdUser = true
					s.assignDefaultSubscriptions(ctx, user.ID)
				}
			} else {
				if createErr := s.userRepo.Create(ctx, newUser); createErr != nil {
					if errors.Is(createErr, ErrEmailExists) {
						conflictUser, lookupErr := s.userRepo.GetByEmail(ctx, email)
						if lookupErr != nil {
							logger.LegacyPrintf("service.auth", "[Auth] Database error getting user after oauth create conflict: create_err=%v lookup_err=%v", createErr, lookupErr)
							return nil, nil, oauthCreateConflictRecoveryError(createErr, lookupErr)
						}
						user = conflictUser
					} else {
						logger.LegacyPrintf("service.auth", "[Auth] Database error creating oauth user: %v", createErr)
						return nil, nil, ErrServiceUnavailable
					}
				} else {
					user = newUser
					createdUser = true
					s.assignDefaultSubscriptions(ctx, user.ID)
					if invitationRedeemCode != nil {
						if err := s.redeemRepo.Use(ctx, invitationRedeemCode.ID, user.ID); err != nil {
							return nil, nil, ErrInvitationCodeInvalid
						}
					}
				}
			}
		} else {
			logger.LegacyPrintf("service.auth", "[Auth] Database error during oauth login: %v", err)
			return nil, nil, ErrServiceUnavailable
		}
	}

	if !user.IsActive() {
		return nil, nil, ErrUserNotActive
	}

	if user.Username == "" && username != "" {
		user.Username = username
		if err := s.userRepo.Update(ctx, user); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Failed to update username after oauth login: %v", err)
		}
	}

	// Best-effort: initialize affiliate row and bind inviter (only when a new user was created).
	if createdUser && s.affiliateService != nil {
		if _, err := s.affiliateService.EnsureAffiliateRow(ctx, user.ID); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] EnsureAffiliateRow failed after oauth signup: user_id=%d err=%v", user.ID, err)
		}
		affCode = strings.TrimSpace(affCode)
		if affCode != "" && s.settingService != nil {
			if settings, err := s.settingService.GetAllSettings(ctx); err == nil && settings != nil && settings.AffiliateEnabled {
				s.affiliateService.BindInviterByCode(ctx, user.ID, affCode)
			}
		}
	}

	tokenPair, err := s.GenerateTokenPair(ctx, user, "")
	if err != nil {
		return nil, nil, fmt.Errorf("generate token pair: %w", err)
	}
	return tokenPair, user, nil
}

func oauthCreateConflictRecoveryError(createErr error, lookupErr error) error {
	return ErrServiceUnavailable.WithCause(fmt.Errorf("oauth create conflict recovery failed: create_err=%w lookup_err=%v", createErr, lookupErr))
}
