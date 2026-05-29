package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *GeminiOAuthService) ExchangeCode(ctx context.Context, input *GeminiExchangeCodeInput) (*GeminiTokenInfo, error) {
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ========== ExchangeCode START ==========")
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] SessionID: %s", input.SessionID)

	session, ok := s.sessionStore.Get(input.SessionID)
	if !ok {
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ERROR: Session not found or expired")
		return nil, fmt.Errorf("session not found or expired")
	}
	if strings.TrimSpace(input.State) == "" || input.State != session.State {
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ERROR: Invalid state")
		return nil, fmt.Errorf("invalid state")
	}

	proxyURL := session.ProxyURL
	if input.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *input.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ProxyURL: %s", proxyURL)

	redirectURI := session.RedirectURI

	// Resolve oauth_type early (defaults to code_assist for backward compatibility).
	oauthType := session.OAuthType
	if oauthType == "" {
		oauthType = "code_assist"
	}
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] OAuth Type: %s", oauthType)
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Project ID from session: %s", session.ProjectID)

	// If the session was created for AI Studio OAuth, ensure a custom OAuth client is configured.
	if oauthType == "ai_studio" {
		effectiveCfg, err := geminicli.EffectiveOAuthConfig(buildGeminiOAuthConfigInput(s.cfg, "ai_studio"), "ai_studio")
		if err != nil {
			return nil, err
		}
		isBuiltinClient := effectiveCfg.ClientID == geminicli.GeminiCLIOAuthClientID
		if isBuiltinClient {
			return nil, fmt.Errorf("AI Studio OAuth requires a custom OAuth Client. Please use an AI Studio API Key account, or configure GEMINI_OAUTH_CLIENT_ID / GEMINI_OAUTH_CLIENT_SECRET and re-authorize")
		}
	}

	effectiveCfg, err := geminicli.EffectiveOAuthConfig(buildGeminiOAuthConfigInput(s.cfg, oauthType), oauthType)
	if err != nil {
		return nil, err
	}
	isBuiltinClient := effectiveCfg.ClientID == geminicli.GeminiCLIOAuthClientID

	// code_assist always uses the built-in client and its fixed redirect URI.
	if oauthType == "code_assist" || isBuiltinClient {
		redirectURI = geminicli.GeminiCLIRedirectURI
	}

	tokenResp, err := s.oauthClient.ExchangeCode(ctx, oauthType, input.Code, session.CodeVerifier, redirectURI, proxyURL)
	if err != nil {
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ERROR: Failed to exchange code: %v", err)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Token exchange successful")
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Token scope: %s", tokenResp.Scope)
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Token expires_in: %d seconds", tokenResp.ExpiresIn)

	sessionProjectID := strings.TrimSpace(session.ProjectID)
	s.sessionStore.Delete(input.SessionID)

	// 计算过期时间：减去 5 分钟安全时间窗口（考虑网络延迟和时钟偏差）
	// 同时设置下界保护，防止 expires_in 过小导致过去时间（引发刷新风暴）
	const safetyWindow = 300 // 5 minutes
	const minTTL = 30        // minimum 30 seconds
	expiresAt := time.Now().Unix() + tokenResp.ExpiresIn - safetyWindow
	minExpiresAt := time.Now().Unix() + minTTL
	if expiresAt < minExpiresAt {
		expiresAt = minExpiresAt
	}

	projectID := sessionProjectID
	var tierID string
	fallbackTierID := canonicalGeminiTierIDForOAuthType(oauthType, input.TierID)
	if fallbackTierID == "" {
		fallbackTierID = canonicalGeminiTierIDForOAuthType(oauthType, session.TierID)
	}

	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ========== Account Type Detection START ==========")
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] OAuth Type: %s", oauthType)

	// 对于 code_assist 模式，project_id 是必需的，需要调用 Code Assist API
	// 对于 google_one 模式，使用个人 Google 账号，不需要 project_id，配额由 Google 网关自动识别
	// 对于 ai_studio 模式，project_id 是可选的（不影响使用 AI Studio API）
	switch oauthType {
	case "code_assist":
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Processing code_assist OAuth type")
		if projectID == "" {
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] No project_id provided, attempting to fetch from LoadCodeAssist API...")
			var err error
			projectID, tierID, err = s.fetchProjectID(ctx, tokenResp.AccessToken, proxyURL)
			if err != nil {
				// 记录警告但不阻断流程，允许后续补充 project_id
				fmt.Printf("[GeminiOAuth] Warning: Failed to fetch project_id during token exchange: %v\n", err)
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] WARNING: Failed to fetch project_id: %v", err)
			} else {
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Successfully fetched project_id: %s, tier_id: %s", projectID, tierID)
			}
		} else {
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] User provided project_id: %s, fetching tier_id...", projectID)
			// 用户手动填了 project_id，仍需调用 LoadCodeAssist 获取 tierID
			_, fetchedTierID, err := s.fetchProjectID(ctx, tokenResp.AccessToken, proxyURL)
			if err != nil {
				fmt.Printf("[GeminiOAuth] Warning: Failed to fetch tierID: %v\n", err)
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] WARNING: Failed to fetch tier_id: %v", err)
			} else {
				tierID = fetchedTierID
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Successfully fetched tier_id: %s", tierID)
			}
		}
		if strings.TrimSpace(projectID) == "" {
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ERROR: Missing project_id for Code Assist OAuth")
			return nil, fmt.Errorf("missing project_id for Code Assist OAuth: please fill Project ID (optional field) and regenerate the auth URL, or ensure your Google account has an ACTIVE GCP project")
		}
		// Prefer auto-detected tier; fall back to user-selected tier.
		tierID = canonicalGeminiTierIDForOAuthType(oauthType, tierID)
		if tierID == "" {
			if fallbackTierID != "" {
				tierID = fallbackTierID
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Using fallback tier_id from user/session: %s", tierID)
			} else {
				tierID = GeminiTierGCPStandard
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Using default tier_id: %s", tierID)
			}
		}
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Final code_assist result - project_id: %s, tier_id: %s", projectID, tierID)

	case "google_one":
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Processing google_one OAuth type")

		// Google One accounts use cloudaicompanion API, which requires a project_id.
		// For personal accounts, Google auto-assigns a project_id via the LoadCodeAssist API.
		if projectID == "" {
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] No project_id provided, attempting to fetch from LoadCodeAssist API...")
			var err error
			projectID, _, err = s.fetchProjectID(ctx, tokenResp.AccessToken, proxyURL)
			if err != nil {
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ERROR: Failed to fetch project_id: %v", err)
				return nil, fmt.Errorf("google One accounts require a project_id, failed to auto-detect: %w", err)
			}
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Successfully fetched project_id: %s", projectID)
		}

		probeScope := googleOneDriveProbeScope(tokenResp.Scope, effectiveCfg.Scopes)
		var storageInfo *geminicli.DriveStorageInfo
		if !canProbeGoogleOneDriveTier(probeScope) {
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Skipping Drive tier probe because drive scope is not granted for this Google One token")
		} else {
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Attempting to fetch Google One tier from Drive API...")
			// Attempt to fetch Drive storage tier
			var err error
			tierID, storageInfo, err = s.FetchGoogleOneTier(ctx, tokenResp.AccessToken, proxyURL)
			if err != nil {
				if errors.Is(err, errGeminiDriveScopeUnavailable) {
					logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Drive scope unavailable during tier probe; falling back to cached or default Google One tier")
				} else {
					// Log warning but don't block - use fallback
					fmt.Printf("[GeminiOAuth] Warning: Failed to fetch Drive tier: %v\n", err)
					logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] WARNING: Failed to fetch Drive tier: %v", err)
				}
				tierID = ""
			} else {
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Successfully fetched Drive tier: %s", tierID)
				if storageInfo != nil {
					logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Drive storage - Limit: %d bytes (%.2f TB), Usage: %d bytes (%.2f GB)",
						storageInfo.Limit, float64(storageInfo.Limit)/float64(TB),
						storageInfo.Usage, float64(storageInfo.Usage)/float64(GB))
				}
			}
		}
		tierID = canonicalGeminiTierIDForOAuthType(oauthType, tierID)
		if tierID == "" || tierID == GeminiTierGoogleOneUnknown {
			if fallbackTierID != "" {
				tierID = fallbackTierID
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Using fallback tier_id from user/session: %s", tierID)
			} else {
				tierID = GeminiTierGoogleOneFree
				logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Using default tier_id: %s", tierID)
			}
		}
		fmt.Printf("[GeminiOAuth] Google One tierID after normalization: %s\n", tierID)

		// Store Drive info in extra field for caching
		if storageInfo != nil {
			tokenInfo := &GeminiTokenInfo{
				AccessToken:  tokenResp.AccessToken,
				RefreshToken: tokenResp.RefreshToken,
				TokenType:    tokenResp.TokenType,
				ExpiresIn:    tokenResp.ExpiresIn,
				ExpiresAt:    expiresAt,
				Scope:        tokenResp.Scope,
				ProjectID:    projectID,
				TierID:       tierID,
				OAuthType:    oauthType,
				Extra: map[string]any{
					"drive_storage_limit":   storageInfo.Limit,
					"drive_storage_usage":   storageInfo.Usage,
					"drive_tier_updated_at": time.Now().Format(time.RFC3339),
				},
			}
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ========== ExchangeCode END (google_one with storage info) ==========")
			return tokenInfo, nil
		}

	case "ai_studio":
		// No automatic tier detection for AI Studio OAuth; rely on user selection.
		if fallbackTierID != "" {
			tierID = fallbackTierID
		} else {
			tierID = GeminiTierAIStudioFree
		}

	default:
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Processing %s OAuth type (no tier detection)", oauthType)
	}

	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ========== Account Type Detection END ==========")

	result := &GeminiTokenInfo{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    expiresAt,
		Scope:        tokenResp.Scope,
		ProjectID:    projectID,
		TierID:       tierID,
		OAuthType:    oauthType,
	}
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Final result - OAuth Type: %s, Project ID: %s, Tier ID: %s", result.OAuthType, result.ProjectID, result.TierID)
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] ========== ExchangeCode END ==========")
	return result, nil
}
