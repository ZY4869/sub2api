package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/grokoauth"
)

type GrokOAuthClient interface {
	ExchangeCode(ctx context.Context, tokenURL string, code string, codeVerifier string, redirectURI string, clientID string, proxyURL string) (*grokoauth.TokenResponse, error)
	RefreshToken(ctx context.Context, tokenURL string, refreshToken string, clientID string, scope string, proxyURL string) (*grokoauth.TokenResponse, error)
	FetchUserInfo(ctx context.Context, userInfoURL string, accessToken string, proxyURL string) (*grokoauth.UserInfo, error)
}
