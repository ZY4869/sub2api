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

var errGeminiDriveScopeUnavailable = errors.New("gemini_drive_scope_unavailable")

// inferGoogleOneTier infers Google One tier from Drive storage limit
func inferGoogleOneTier(storageBytes int64) string {
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] inferGoogleOneTier - input: %d bytes (%.2f TB)", storageBytes, float64(storageBytes)/float64(TB))

	if storageBytes <= 0 {
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] inferGoogleOneTier - storageBytes <= 0, returning UNKNOWN")
		return GeminiTierGoogleOneUnknown
	}

	if storageBytes > StorageTierUnlimited {
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] inferGoogleOneTier - > %d bytes (100TB), returning UNLIMITED", StorageTierUnlimited)
		return GeminiTierGoogleAIUltra
	}
	if storageBytes >= StorageTierAIPremium {
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] inferGoogleOneTier - >= %d bytes (2TB), returning google_ai_pro", StorageTierAIPremium)
		return GeminiTierGoogleAIPro
	}
	if storageBytes >= StorageTierFree {
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] inferGoogleOneTier - >= %d bytes (15GB), returning FREE", StorageTierFree)
		return GeminiTierGoogleOneFree
	}

	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] inferGoogleOneTier - < %d bytes (15GB), returning UNKNOWN", StorageTierFree)
	return GeminiTierGoogleOneUnknown
}

// FetchGoogleOneTier fetches Google One tier from Drive API.
// Note: LoadCodeAssist API is NOT called for Google One accounts because:
// 1. It's designed for GCP IAM (enterprise), not personal Google accounts
// 2. Personal accounts will get 403/404 from cloudaicompanion.googleapis.com
// 3. Google consumer (Google One) and enterprise (GCP) systems are physically isolated
func (s *GeminiOAuthService) FetchGoogleOneTier(ctx context.Context, accessToken, proxyURL string) (string, *geminicli.DriveStorageInfo, error) {
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Starting FetchGoogleOneTier (Google One personal account)")

	// Use Drive API to infer tier from storage quota (requires drive.readonly scope)
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Calling Drive API for storage quota...")

	storageInfo, err := s.driveClient.GetStorageQuota(ctx, accessToken, proxyURL)
	if err != nil {
		// Check if it's a 403 (scope not granted)
		if strings.Contains(err.Error(), "status 403") {
			logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Drive API returned 403; treating Google One tier as unavailable for this token")
			return GeminiTierGoogleOneUnknown, nil, fmt.Errorf("%w: status 403", errGeminiDriveScopeUnavailable)
		}
		// Other errors
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Failed to fetch Drive storage: %v", err)
		return GeminiTierGoogleOneUnknown, nil, err
	}

	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Drive API response - Limit: %d bytes (%.2f TB), Usage: %d bytes (%.2f GB)",
		storageInfo.Limit, float64(storageInfo.Limit)/float64(TB),
		storageInfo.Usage, float64(storageInfo.Usage)/float64(GB))

	tierID := inferGoogleOneTier(storageInfo.Limit)
	logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Inferred tier from storage: %s", tierID)

	return tierID, storageInfo, nil
}

// RefreshAccountGoogleOneTier 刷新单个账号的 Google One Tier
func (s *GeminiOAuthService) RefreshAccountGoogleOneTier(
	ctx context.Context,
	account *Account,
) (tierID string, extra map[string]any, credentials map[string]any, err error) {
	if account == nil {
		return "", nil, nil, fmt.Errorf("account is nil")
	}

	// 验证账号类型
	oauthType, ok := account.Credentials["oauth_type"].(string)
	if !ok || oauthType != "google_one" {
		return "", nil, nil, fmt.Errorf("not a google_one OAuth account")
	}

	// 获取 access_token
	accessToken, ok := account.Credentials["access_token"].(string)
	if !ok || accessToken == "" {
		return "", nil, nil, fmt.Errorf("missing access_token")
	}

	existingTierID := canonicalGeminiTierIDForOAuthType(oauthType, account.GetCredential("tier_id"))

	// 获取 proxy URL
	var proxyURL string
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	scope := strings.TrimSpace(account.GetCredential("scope"))
	if !canProbeGoogleOneDriveTier(scope) {
		tierID = existingTierID
		if tierID == "" {
			tierID = GeminiTierGoogleOneFree
		}
		extra = make(map[string]any)
		for k, v := range account.Extra {
			extra[k] = v
		}
		credentials = make(map[string]any)
		for k, v := range account.Credentials {
			credentials[k] = v
		}
		credentials["tier_id"] = tierID
		logger.LegacyPrintf("service.gemini_oauth", "[GeminiOAuth] Skipping manual Google One Drive tier refresh because drive scope is not present on this account")
		return tierID, extra, credentials, nil
	}

	// 调用 Drive API
	tierID, storageInfo, err := s.FetchGoogleOneTier(ctx, accessToken, proxyURL)
	if err != nil {
		return "", nil, nil, err
	}

	// 构建 extra 数据（保留原有 extra 字段）
	extra = make(map[string]any)
	for k, v := range account.Extra {
		extra[k] = v
	}
	if storageInfo != nil {
		extra["drive_storage_limit"] = storageInfo.Limit
		extra["drive_storage_usage"] = storageInfo.Usage
		extra["drive_tier_updated_at"] = time.Now().Format(time.RFC3339)
	}

	// 构建 credentials 数据
	credentials = make(map[string]any)
	for k, v := range account.Credentials {
		credentials[k] = v
	}
	credentials["tier_id"] = tierID

	return tierID, extra, credentials, nil
}
