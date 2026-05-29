package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *GeminiOAuthService) RefreshToken(ctx context.Context, oauthType, refreshToken, proxyURL string) (*GeminiTokenInfo, error) {
	var lastErr error

	for attempt := 0; attempt <= 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			time.Sleep(backoff)
		}

		tokenResp, err := s.oauthClient.RefreshToken(ctx, oauthType, refreshToken, proxyURL)
		if err == nil {
			// 计算过期时间：减去 5 分钟安全时间窗口（考虑网络延迟和时钟偏差）
			// 同时设置下界保护，防止 expires_in 过小导致过去时间（引发刷新风暴）
			const safetyWindow = 300 // 5 minutes
			const minTTL = 30        // minimum 30 seconds
			expiresAt := time.Now().Unix() + tokenResp.ExpiresIn - safetyWindow
			minExpiresAt := time.Now().Unix() + minTTL
			if expiresAt < minExpiresAt {
				expiresAt = minExpiresAt
			}
			return &GeminiTokenInfo{
				AccessToken:  tokenResp.AccessToken,
				RefreshToken: tokenResp.RefreshToken,
				TokenType:    tokenResp.TokenType,
				ExpiresIn:    tokenResp.ExpiresIn,
				ExpiresAt:    expiresAt,
				Scope:        tokenResp.Scope,
			}, nil
		}

		if isNonRetryableGeminiOAuthError(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("token refresh failed after retries: %w", lastErr)
}

func isNonRetryableGeminiOAuthError(err error) bool {
	msg := err.Error()
	nonRetryable := []string{
		"invalid_grant",
		"invalid_client",
		"unauthorized_client",
		"access_denied",
	}
	for _, needle := range nonRetryable {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func (s *GeminiOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*GeminiTokenInfo, error) {
	if account.Platform != PlatformGemini || account.Type != AccountTypeOAuth {
		return nil, fmt.Errorf("account is not a Gemini OAuth account")
	}

	refreshToken := account.GetCredential("refresh_token")
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// Preserve oauth_type from the account (defaults to code_assist for backward compatibility).
	oauthType := strings.TrimSpace(account.GetCredential("oauth_type"))
	if oauthType == "" {
		oauthType = "code_assist"
	}

	var proxyURL string
	if account.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	tokenInfo, err := s.RefreshToken(ctx, oauthType, refreshToken, proxyURL)
	// Backward compatibility:
	// Older versions could refresh Code Assist tokens using a user-provided OAuth client when configured.
	// If the refresh token was originally issued to that custom client, forcing the built-in client will
	// fail with "unauthorized_client". In that case, retry with the custom client (ai_studio path) when available.
	if err != nil && oauthType == "code_assist" && strings.Contains(err.Error(), "unauthorized_client") && s.GetOAuthConfig().AIStudioOAuthEnabled {
		if alt, altErr := s.RefreshToken(ctx, "ai_studio", refreshToken, proxyURL); altErr == nil {
			tokenInfo = alt
			err = nil
		}
	}
	// Backward compatibility for google_one:
	// - New behavior: when a custom OAuth client is configured, google_one will use it.
	// - Old behavior: google_one always used the built-in Gemini CLI OAuth client.
	// If an existing account was authorized with the built-in client, refreshing with the custom client
	// will fail with "unauthorized_client". Retry with the built-in client (code_assist path forces it).
	if err != nil && oauthType == "google_one" && strings.Contains(err.Error(), "unauthorized_client") && s.GetOAuthConfig().AIStudioOAuthEnabled {
		if alt, altErr := s.RefreshToken(ctx, "code_assist", refreshToken, proxyURL); altErr == nil {
			tokenInfo = alt
			err = nil
		}
	}
	if err != nil {
		// Provide a more actionable error for common OAuth client mismatch issues.
		if strings.Contains(err.Error(), "unauthorized_client") {
			return nil, fmt.Errorf("%w (OAuth client mismatch: the refresh_token is bound to the OAuth client used during authorization; please re-authorize this account or restore the original GEMINI_OAUTH_CLIENT_ID/SECRET)", err)
		}
		return nil, err
	}

	tokenInfo.OAuthType = oauthType

	// Preserve account's project_id when present.
	existingProjectID := strings.TrimSpace(account.GetCredential("project_id"))
	if existingProjectID != "" {
		tokenInfo.ProjectID = existingProjectID
	}

	// 尝试从账号凭证获取 tierID（向后兼容）
	existingTierID := strings.TrimSpace(account.GetCredential("tier_id"))

	// For Code Assist, project_id is required. Auto-detect if missing.
	// For AI Studio OAuth, project_id is optional and should not block refresh.
	switch oauthType {
	case "code_assist":
		// 先设置默认值或保留旧值，确保 tier_id 始终有值
		if existingTierID != "" {
			tokenInfo.TierID = canonicalGeminiTierIDForOAuthType(oauthType, existingTierID)
		}
		if tokenInfo.TierID == "" {
			tokenInfo.TierID = GeminiTierGCPStandard
		}

		// 尝试自动探测 project_id 和 tier_id
		needDetect := strings.TrimSpace(tokenInfo.ProjectID) == "" || tokenInfo.TierID == ""
		if needDetect {
			projectID, tierID, err := s.fetchProjectID(ctx, tokenInfo.AccessToken, proxyURL)
			if err != nil {
				fmt.Printf("[GeminiOAuth] Warning: failed to auto-detect project/tier: %v\n", err)
			} else {
				if strings.TrimSpace(tokenInfo.ProjectID) == "" && projectID != "" {
					tokenInfo.ProjectID = projectID
				}
				if tierID != "" {
					if canonical := canonicalGeminiTierIDForOAuthType(oauthType, tierID); canonical != "" {
						tokenInfo.TierID = canonical
					}
				}
			}
		}

		if strings.TrimSpace(tokenInfo.ProjectID) == "" {
			return nil, fmt.Errorf("failed to auto-detect project_id: empty result")
		}
	case "google_one":
		canonicalExistingTier := canonicalGeminiTierIDForOAuthType(oauthType, existingTierID)
		effectiveScope := strings.TrimSpace(tokenInfo.Scope)
		if effectiveScope == "" {
			effectiveScope = strings.TrimSpace(account.GetCredential("scope"))
		}
		// Check if tier cache is stale (> 24 hours)
		needsRefresh := true
		if account.Extra != nil {
			if updatedAtStr, ok := account.Extra["drive_tier_updated_at"].(string); ok {
				if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
					if time.Since(updatedAt) <= 24*time.Hour {
						needsRefresh = false
						// Use cached tier
						tokenInfo.TierID = canonicalExistingTier
					}
				}
			}
		}

		if tokenInfo.TierID == "" {
			tokenInfo.TierID = canonicalExistingTier
		}

		if needsRefresh {
			if !canProbeGoogleOneDriveTier(effectiveScope) {
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Skipping Drive tier refresh because drive scope is not present on the current Google One token")
			} else {
				tierID, storageInfo, err := s.FetchGoogleOneTier(ctx, tokenInfo.AccessToken, proxyURL)
				if err == nil {
					if canonical := canonicalGeminiTierIDForOAuthType(oauthType, tierID); canonical != "" && canonical != GeminiTierGoogleOneUnknown {
						tokenInfo.TierID = canonical
					}
					if storageInfo != nil {
						tokenInfo.Extra = map[string]any{
							"drive_storage_limit":   storageInfo.Limit,
							"drive_storage_usage":   storageInfo.Usage,
							"drive_tier_updated_at": time.Now().Format(time.RFC3339),
						}
					}
				} else if errors.Is(err, errGeminiDriveScopeUnavailable) {
					logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Drive scope unavailable during refresh; reusing cached or default Google One tier")
				}
			}
		}

		if tokenInfo.TierID == "" || tokenInfo.TierID == GeminiTierGoogleOneUnknown {
			if canonicalExistingTier != "" {
				tokenInfo.TierID = canonicalExistingTier
			} else {
				tokenInfo.TierID = GeminiTierGoogleOneFree
			}
		}
	}

	return tokenInfo, nil
}

func (s *GeminiOAuthService) BuildAccountCredentials(tokenInfo *GeminiTokenInfo) map[string]any {
	creds := map[string]any{
		"access_token": tokenInfo.AccessToken,
		"expires_at":   strconv.FormatInt(tokenInfo.ExpiresAt, 10),
	}
	if tokenInfo.RefreshToken != "" {
		creds["refresh_token"] = tokenInfo.RefreshToken
	}
	if tokenInfo.TokenType != "" {
		creds["token_type"] = tokenInfo.TokenType
	}
	if tokenInfo.Scope != "" {
		creds["scope"] = tokenInfo.Scope
	}
	if tokenInfo.ProjectID != "" {
		creds["project_id"] = tokenInfo.ProjectID
	}
	if tokenInfo.TierID != "" {
		// Validate tier_id before storing
		if err := validateTierID(tokenInfo.TierID); err == nil {
			creds["tier_id"] = tokenInfo.TierID
			fmt.Printf("[GeminiOAuth] Storing tier_id: %s\n", tokenInfo.TierID)
		} else {
			fmt.Printf("[GeminiOAuth] Invalid tier_id %s: %v\n", tokenInfo.TierID, err)
		}
		// Silently skip invalid tier_id (don't block account creation)
	}
	if tokenInfo.OAuthType != "" {
		creds["oauth_type"] = tokenInfo.OAuthType
	}
	// Store extra metadata (Drive info) if present
	if len(tokenInfo.Extra) > 0 {
		for k, v := range tokenInfo.Extra {
			creds[k] = v
		}
	}
	return creds
}
