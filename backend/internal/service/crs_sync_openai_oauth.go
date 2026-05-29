package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *CRSSyncService) syncCRSOpenAIOAuthAccounts(ctx context.Context, input SyncFromCRSInput, exported *crsExportResponse, result *SyncFromCRSResult, selectedSet map[string]struct{}, proxies *[]Proxy, now string) {
	// OpenAI OAuth -> sub2api openai oauth
	for _, src := range exported.Data.OpenAIOAuthAccounts {
		item := SyncFromCRSItemResult{
			CRSAccountID: src.ID,
			Kind:         src.Kind,
			Name:         src.Name,
		}

		accessToken, _ := src.Credentials["access_token"].(string)
		if strings.TrimSpace(accessToken) == "" {
			item.Action = "failed"
			item.Error = "missing access_token"
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		proxyID, err := s.mapOrCreateProxy(
			ctx,
			input.SyncProxies,
			proxies,
			src.Proxy,
			fmt.Sprintf("crs-%s", src.Name),
		)
		if err != nil {
			item.Action = "failed"
			item.Error = "proxy sync failed: " + err.Error()
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		credentials := sanitizeCredentialsMap(src.Credentials)
		// Normalize token_type
		if v, ok := credentials["token_type"].(string); !ok || strings.TrimSpace(v) == "" {
			credentials["token_type"] = "Bearer"
		}
		// Convert expires_at from ISO string to Unix timestamp
		if expiresAtStr, ok := credentials["expires_at"].(string); ok && expiresAtStr != "" {
			if t, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
				credentials["expires_at"] = t.Unix()
			}
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
		// Extract email from CRS extra (crs_email -> email)
		if crsEmail, ok := src.Extra["crs_email"]; ok {
			extra["email"] = crsEmail
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
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
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
			if refreshedCreds := s.refreshOAuthToken(ctx, account); refreshedCreds != nil {
				_ = persistAccountCredentials(ctx, s.accountRepo, account, refreshedCreds)
			}
			item.Action = "created"
			result.Created++
			result.Items = append(result.Items, item)
			continue
		}

		existing.Extra = mergeMap(existing.Extra, extra)
		existing.Name = defaultName(src.Name, src.ID)
		existing.Platform = PlatformOpenAI
		existing.Type = AccountTypeOAuth
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
		if refreshedCreds := s.refreshOAuthToken(ctx, existing); refreshedCreds != nil {
			_ = persistAccountCredentials(ctx, s.accountRepo, existing, refreshedCreds)
		}

		item.Action = "updated"
		result.Updated++
		result.Items = append(result.Items, item)
	}
}
