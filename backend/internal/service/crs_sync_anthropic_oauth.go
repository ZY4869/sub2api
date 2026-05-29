package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *CRSSyncService) syncCRSClaudeAccounts(ctx context.Context, input SyncFromCRSInput, exported *crsExportResponse, result *SyncFromCRSResult, selectedSet map[string]struct{}, proxies *[]Proxy, now string) {
	// Claude OAuth / Setup Token -> sub2api anthropic oauth/setup-token
	for _, src := range exported.Data.ClaudeAccounts {
		item := SyncFromCRSItemResult{
			CRSAccountID: src.ID,
			Kind:         src.Kind,
			Name:         src.Name,
		}

		targetType := strings.TrimSpace(src.AuthType)
		if targetType == "" {
			targetType = "oauth"
		}
		if targetType != AccountTypeOAuth && targetType != AccountTypeSetupToken {
			item.Action = "skipped"
			item.Error = "unsupported authType: " + targetType
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}

		accessToken, _ := src.Credentials["access_token"].(string)
		if strings.TrimSpace(accessToken) == "" {
			item.Action = "failed"
			item.Error = "missing access_token"
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		proxyID, err := s.mapOrCreateProxy(ctx, input.SyncProxies, proxies, src.Proxy, fmt.Sprintf("crs-%s", src.Name))
		if err != nil {
			item.Action = "failed"
			item.Error = "proxy sync failed: " + err.Error()
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		credentials := sanitizeCredentialsMap(src.Credentials)
		// Remove /v1 suffix from base_url for Claude accounts
		cleanBaseURL(credentials, "/v1")
		// Convert expires_at from ISO string to Unix timestamp
		if expiresAtStr, ok := credentials["expires_at"].(string); ok && expiresAtStr != "" {
			if t, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
				credentials["expires_at"] = t.Unix()
			}
		}
		// Add intercept_warmup_requests if not present (defaults to false)
		if _, exists := credentials["intercept_warmup_requests"]; !exists {
			credentials["intercept_warmup_requests"] = false
		}
		priority := clampPriority(src.Priority)
		concurrency := 3
		status := mapCRSStatus(src.IsActive, src.Status)

		// Preserve all CRS extra fields and add sync metadata
		extra := make(map[string]any)
		if src.Extra != nil {
			for k, v := range src.Extra {
				extra[k] = v
			}
		}
		extra["crs_account_id"] = src.ID
		extra["crs_kind"] = src.Kind
		extra["crs_synced_at"] = now
		// Extract org_uuid and account_uuid from CRS credentials to extra
		if orgUUID, ok := src.Credentials["org_uuid"]; ok {
			extra["org_uuid"] = orgUUID
		}
		if accountUUID, ok := src.Credentials["account_uuid"]; ok {
			extra["account_uuid"] = accountUUID
		}

		existing, err := s.accountRepo.GetByCRSAccountID(ctx, src.ID)
		if err != nil {
			item.Action = "failed"
			item.Error = "db lookup failed: " + err.Error()
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		if existing == nil {
			if !shouldCreateAccount(src.ID, selectedSet) {
				item.Action = "skipped"
				item.Error = "not selected"
				result.Skipped++
				result.Items = append(result.Items, item)
				continue
			}
			account := &Account{
				Name:        defaultName(src.Name, src.ID),
				Platform:    PlatformAnthropic,
				Type:        targetType,
				Credentials: credentials,
				Extra:       extra,
				ProxyID:     proxyID,
				Concurrency: concurrency,
				Priority:    priority,
				Status:      status,
				Schedulable: src.Schedulable,
			}
			if err := s.accountRepo.Create(ctx, account); err != nil {
				item.Action = "failed"
				item.Error = "create failed: " + err.Error()
				result.Failed++
				result.Items = append(result.Items, item)
				continue
			}
			// Refresh OAuth token after creation
			if targetType == AccountTypeOAuth {
				if refreshedCreds := s.refreshOAuthToken(ctx, account); refreshedCreds != nil {
					_ = persistAccountCredentials(ctx, s.accountRepo, account, refreshedCreds)
				}
			}
			item.Action = "created"
			result.Created++
			result.Items = append(result.Items, item)
			continue
		}

		// Update existing
		existing.Extra = mergeMap(existing.Extra, extra)
		existing.Name = defaultName(src.Name, src.ID)
		existing.Platform = PlatformAnthropic
		existing.Type = targetType
		existing.Credentials = mergeMap(existing.Credentials, credentials)
		if proxyID != nil {
			existing.ProxyID = proxyID
		}
		existing.Concurrency = concurrency
		existing.Priority = priority
		existing.Status = status
		existing.Schedulable = src.Schedulable

		if err := s.accountRepo.Update(ctx, existing); err != nil {
			item.Action = "failed"
			item.Error = "update failed: " + err.Error()
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		// Refresh OAuth token after update
		if targetType == AccountTypeOAuth {
			if refreshedCreds := s.refreshOAuthToken(ctx, existing); refreshedCreds != nil {
				_ = persistAccountCredentials(ctx, s.accountRepo, existing, refreshedCreds)
			}
		}

		item.Action = "updated"
		result.Updated++
		result.Items = append(result.Items, item)
	}
}
