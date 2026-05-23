package service

import (
	"context"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AuthProviderGitHub   = "github"
	AuthProviderGoogle   = "google"
	AuthProviderDingTalk = "dingtalk"
)

var (
	ErrAuthIdentityNotFound       = infraerrors.NotFound("AUTH_IDENTITY_NOT_FOUND", "auth identity not found")
	ErrAuthIdentityAlreadyBound   = infraerrors.Conflict("AUTH_IDENTITY_ALREADY_BOUND", "auth identity already bound")
	ErrAuthIdentityProviderExists = infraerrors.Conflict("AUTH_IDENTITY_PROVIDER_EXISTS", "provider already bound to user")
	ErrAuthIdentityEmailConflict  = infraerrors.Conflict("AUTH_IDENTITY_EMAIL_CONFLICT", "oauth email cannot safely take over existing account")
	ErrAuthIdentityBindRequired   = infraerrors.BadRequest("AUTH_IDENTITY_BIND_REQUIRED", "bind mode requires authenticated user")
	ErrAuthIdentityUnsafeEmail    = infraerrors.Forbidden("AUTH_IDENTITY_UNVERIFIED_EMAIL", "unverified oauth email cannot take over existing account")
	ErrOAuthProviderUnsupported   = infraerrors.BadRequest("OAUTH_PROVIDER_UNSUPPORTED", "oauth provider is unsupported")
)

type AuthIdentity struct {
	ID             int64
	Provider       string
	ProviderUserID string
	UserID         int64
	Email          string
	EmailVerified  bool
	DisplayName    string
	AvatarURL      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AuthIdentityRepository interface {
	Create(ctx context.Context, identity *AuthIdentity) error
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*AuthIdentity, error)
	GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*AuthIdentity, error)
	ListByUserID(ctx context.Context, userID int64) ([]*AuthIdentity, error)
	DeleteByUserIDAndProvider(ctx context.Context, userID int64, provider string) error
}

type SocialOAuthConfig struct {
	Provider                  string
	Enabled                   bool
	ClientID                  string
	ClientSecret              string
	AuthorizeURL              string
	TokenURL                  string
	UserInfoURL               string
	Scopes                    string
	RedirectURL               string
	FrontendRedirectURL       string
	TokenAuthMethod           string
	UsePKCE                   bool
	UserInfoEmailPath         string
	UserInfoIDPath            string
	UserInfoUsernamePath      string
	UserInfoAvatarPath        string
	UserInfoEmailVerifiedPath string
}

type SocialIdentityResult struct {
	Identity   *AuthIdentity
	User       *User
	TokenPair  *TokenPair
	Pending    string
	Outcome    string
	RedirectTo string
}

type AuthIdentityService struct {
	repo        AuthIdentityRepository
	userRepo    UserRepository
	authService *AuthService
}

func NewAuthIdentityService(repo AuthIdentityRepository, userRepo UserRepository, authService *AuthService) *AuthIdentityService {
	return &AuthIdentityService{
		repo:        repo,
		userRepo:    userRepo,
		authService: authService,
	}
}

func NormalizeOAuthProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case AuthProviderGitHub:
		return AuthProviderGitHub
	case AuthProviderGoogle:
		return AuthProviderGoogle
	case AuthProviderDingTalk:
		return AuthProviderDingTalk
	default:
		return ""
	}
}

func (s *AuthIdentityService) ListByUserID(ctx context.Context, userID int64) ([]*AuthIdentity, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListByUserID(ctx, userID)
}

func (s *AuthIdentityService) DeleteByUserIDAndProvider(ctx context.Context, userID int64, provider string) error {
	if s == nil || s.repo == nil {
		return ErrAuthIdentityNotFound
	}
	provider = NormalizeOAuthProvider(provider)
	if provider == "" {
		return ErrOAuthProviderUnsupported
	}
	return s.repo.DeleteByUserIDAndProvider(ctx, userID, provider)
}

func (s *AuthIdentityService) BindIdentity(ctx context.Context, userID int64, identity *AuthIdentity) (*AuthIdentity, error) {
	if s == nil || s.repo == nil || identity == nil {
		return nil, ErrAuthIdentityNotFound
	}
	identity.Provider = NormalizeOAuthProvider(identity.Provider)
	if identity.Provider == "" {
		return nil, ErrOAuthProviderUnsupported
	}
	if userID <= 0 {
		return nil, ErrAuthIdentityBindRequired
	}

	if existing, err := s.repo.GetByProviderUserID(ctx, identity.Provider, identity.ProviderUserID); err == nil && existing != nil {
		if existing.UserID == userID {
			return existing, nil
		}
		return nil, ErrAuthIdentityAlreadyBound
	}

	if existing, err := s.repo.GetByUserIDAndProvider(ctx, userID, identity.Provider); err == nil && existing != nil {
		if existing.ProviderUserID == identity.ProviderUserID {
			return existing, nil
		}
		return nil, ErrAuthIdentityProviderExists
	}

	identity.UserID = userID
	if err := s.repo.Create(ctx, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

func (s *AuthIdentityService) ResolveLoginOrBind(ctx context.Context, mode string, bindUserID int64, identity *AuthIdentity, invitationCode string, affCode string) (*SocialIdentityResult, error) {
	if s == nil || s.repo == nil || s.userRepo == nil || s.authService == nil || identity == nil {
		return nil, ErrServiceUnavailable
	}
	identity.Provider = NormalizeOAuthProvider(identity.Provider)
	if identity.Provider == "" {
		return nil, ErrOAuthProviderUnsupported
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "login"
	}

	if existing, err := s.repo.GetByProviderUserID(ctx, identity.Provider, identity.ProviderUserID); err == nil && existing != nil {
		if mode == "bind" {
			if bindUserID <= 0 {
				return nil, ErrAuthIdentityBindRequired
			}
			if existing.UserID != bindUserID {
				return nil, ErrAuthIdentityAlreadyBound
			}
			user, getErr := s.userRepo.GetByID(ctx, bindUserID)
			if getErr != nil {
				return nil, getErr
			}
			return &SocialIdentityResult{
				Identity: existing,
				User:     user,
				Outcome:  "bind_success",
			}, nil
		}
		user, getErr := s.userRepo.GetByID(ctx, existing.UserID)
		if getErr != nil {
			return nil, getErr
		}
		tokenPair, tokenErr := s.authService.GenerateTokenPair(ctx, user, "")
		if tokenErr != nil {
			return nil, tokenErr
		}
		return &SocialIdentityResult{
			Identity:  existing,
			User:      user,
			TokenPair: tokenPair,
			Outcome:   "login",
		}, nil
	}

	if mode == "bind" {
		if bindUserID <= 0 {
			return nil, ErrAuthIdentityBindRequired
		}
		bound, err := s.BindIdentity(ctx, bindUserID, identity)
		if err != nil {
			return nil, err
		}
		user, getErr := s.userRepo.GetByID(ctx, bindUserID)
		if getErr != nil {
			return nil, getErr
		}
		return &SocialIdentityResult{
			Identity: bound,
			User:     user,
			Outcome:  "bind_success",
		}, nil
	}

	if identity.EmailVerified && strings.TrimSpace(identity.Email) != "" {
		user, err := s.userRepo.GetByEmail(ctx, strings.TrimSpace(identity.Email))
		if err == nil && user != nil {
			bound, bindErr := s.BindIdentity(ctx, user.ID, identity)
			if bindErr != nil {
				return nil, bindErr
			}
			tokenPair, tokenErr := s.authService.GenerateTokenPair(ctx, user, "")
			if tokenErr != nil {
				return nil, tokenErr
			}
			return &SocialIdentityResult{
				Identity:  bound,
				User:      user,
				TokenPair: tokenPair,
				Outcome:   "login",
			}, nil
		}
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}

	if strings.TrimSpace(identity.Email) != "" && !identity.EmailVerified {
		_, err := s.userRepo.GetByEmail(ctx, strings.TrimSpace(identity.Email))
		if err == nil {
			return nil, ErrAuthIdentityUnsafeEmail
		}
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}

	if strings.TrimSpace(identity.Email) != "" && identity.EmailVerified {
		_, err := s.userRepo.GetByEmail(ctx, strings.TrimSpace(identity.Email))
		if err == nil {
			return nil, ErrAuthIdentityEmailConflict
		}
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
	}

	tokenPair, user, err := s.authService.LoginOrRegisterOAuthWithTokenPair(ctx, strings.TrimSpace(identity.Email), strings.TrimSpace(identity.DisplayName), invitationCode, affCode)
	if err != nil {
		if errors.Is(err, ErrOAuthInvitationRequired) {
			pending, pendingErr := s.authService.CreatePendingOAuthTokenWithIdentity(&pendingOAuthClaims{
				Email:          strings.TrimSpace(identity.Email),
				Username:       strings.TrimSpace(identity.DisplayName),
				AffCode:        affCode,
				Provider:       identity.Provider,
				ProviderUserID: identity.ProviderUserID,
				EmailVerified:  identity.EmailVerified,
				DisplayName:    identity.DisplayName,
				AvatarURL:      identity.AvatarURL,
			})
			if pendingErr != nil {
				return nil, pendingErr
			}
			return &SocialIdentityResult{
				Identity: identity,
				Pending:  pending,
				Outcome:  "pending",
			}, nil
		}
		return nil, err
	}

	identity.UserID = user.ID
	if _, bindErr := s.BindIdentity(ctx, user.ID, identity); bindErr != nil {
		return nil, bindErr
	}
	return &SocialIdentityResult{
		Identity:  identity,
		User:      user,
		TokenPair: tokenPair,
		Outcome:   "login",
	}, nil
}
