package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// pendingOAuthTokenTTL is the validity period for pending OAuth tokens.
const pendingOAuthTokenTTL = 10 * time.Minute

// pendingOAuthPurpose is the purpose claim value for pending OAuth registration tokens.
const pendingOAuthPurpose = "pending_oauth_registration"

type pendingOAuthClaims struct {
	Email          string `json:"email"`
	Username       string `json:"username"`
	AffCode        string `json:"aff_code,omitempty"`
	Provider       string `json:"provider,omitempty"`
	ProviderUserID string `json:"provider_user_id,omitempty"`
	EmailVerified  bool   `json:"email_verified,omitempty"`
	DisplayName    string `json:"display_name,omitempty"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	BindUserID     int64  `json:"bind_user_id,omitempty"`
	Purpose        string `json:"purpose"`
	jwt.RegisteredClaims
}

// CreatePendingOAuthToken generates a short-lived JWT that carries the OAuth identity
// while waiting for the user to supply an invitation code.
func (s *AuthService) CreatePendingOAuthToken(email, username, affCode string) (string, error) {
	return s.CreatePendingOAuthTokenWithIdentity(&pendingOAuthClaims{
		Email:    email,
		Username: username,
		AffCode:  affCode,
	})
}

func (s *AuthService) CreatePendingOAuthTokenWithIdentity(claims *pendingOAuthClaims) (string, error) {
	now := time.Now()
	if claims == nil {
		claims = &pendingOAuthClaims{}
	}
	affCode := strings.TrimSpace(claims.AffCode)
	if len([]rune(affCode)) > 64 {
		affCode = string([]rune(affCode)[:64])
	}
	claims.AffCode = affCode
	claims.Provider = NormalizeOAuthProvider(claims.Provider)
	claims.Purpose = pendingOAuthPurpose
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(pendingOAuthTokenTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// VerifyPendingOAuthToken validates a pending OAuth token and returns the embedded identity.
// Returns ErrInvalidToken when the token is invalid or expired.
func (s *AuthService) VerifyPendingOAuthToken(tokenStr string) (email, username, affCode string, err error) {
	claims, err := s.VerifyPendingOAuthClaims(tokenStr)
	if err != nil {
		return "", "", "", err
	}
	return claims.Email, claims.Username, claims.AffCode, nil
}

func (s *AuthService) VerifyPendingOAuthClaims(tokenStr string) (*pendingOAuthClaims, error) {
	if len(tokenStr) > maxTokenLength {
		return nil, ErrInvalidToken
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	token, parseErr := parser.ParseWithClaims(tokenStr, &pendingOAuthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if parseErr != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*pendingOAuthClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Purpose != pendingOAuthPurpose {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
