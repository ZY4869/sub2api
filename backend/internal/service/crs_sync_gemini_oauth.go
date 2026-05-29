package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *CRSSyncService) syncCRSGeminiOAuthAccounts(ctx context.Context, input SyncFromCRSInput, exported *crsExportResponse, result *SyncFromCRSResult, selectedSet map[string]struct{}, proxies *[]Proxy, now string) {
	// Gemini OAuth -> sub2api gemini oauth
	for _, src := range exported.Data.GeminiOAuthAccounts {
		item := SyncFromCRSItemResult{
			CRSAccountID: src.ID,
			Kind:         src.Kind,
			Name:         src.Name,
		}

		refreshToken, _ := src.Credentials["refresh_token"].(string)
		if strings.TrimSpace(refreshToken) == "" {
			item.Action = "failed"
			item.Error = "missing refresh_token"
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
		if v, ok := credentials["token_type"].(string); !ok || strings.TrimSpace(v) == "" {
			credentials["token_type"] = "Bearer"
		}
		// Convert expires_at from RFC3339 to Unix seconds string (recommended to keep consistent with GetCredential())
		if expiresAtStr, ok := credentials["expires_at"].(string); ok && strings.TrimSpace(expiresAtStr) != "" {
			if t, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
				credentials["expires_at"] = strconv.FormatInt(t.Unix(), 10)
			}
		}

		extra := make(map[string]any)
		if src.Extra != nil {
			for k, v := range src.Extra {
				extra[k] = v
			}
		}
		extra["crs_account_id"] = src.ID
		extra["crs_kind"] = src.Kind
		extra["crs_synced_at"] = now

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
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Credentials: credentials,
				Extra:       extra,
				ProxyID:     proxyID,
				Concurrency: 3,
				Priority:    clampPriority(src.Priority),
				Status:      mapCRSStatus(src.IsActive, src.Status),
				Schedulable: src.Schedulable,
			}
			if err := s.accountRepo.Create(ctx, account); err != nil {
				item.Action = "failed"
				item.Error = "create failed: " + err.Error()
				result.Failed++
				result.Items = append(result.Items, item)
				continue
			}
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
		existing.Platform = PlatformGemini
		existing.Type = AccountTypeOAuth
		existing.Credentials = mergeMap(existing.Credentials, credentials)
		if proxyID != nil {
			existing.ProxyID = proxyID
		}
		existing.Concurrency = 3
		existing.Priority = clampPriority(src.Priority)
		existing.Status = mapCRSStatus(src.IsActive, src.Status)
		existing.Schedulable = src.Schedulable

		if err := s.accountRepo.Update(ctx, existing); err != nil {
			item.Action = "failed"
			item.Error = "update failed: " + err.Error()
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		if refreshedCreds := s.refreshOAuthToken(ctx, existing); refreshedCreds != nil {
			_ = persistAccountCredentials(ctx, s.accountRepo, existing, refreshedCreds)
		}

		item.Action = "updated"
		result.Updated++
		result.Items = append(result.Items, item)
	}
}
