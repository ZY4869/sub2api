package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *CRSSyncService) syncCRSGeminiAPIKeyAccounts(ctx context.Context, input SyncFromCRSInput, exported *crsExportResponse, result *SyncFromCRSResult, selectedSet map[string]struct{}, proxies *[]Proxy, now string) {
	// Gemini API Key -> sub2api gemini apikey
	for _, src := range exported.Data.GeminiAPIKeyAccounts {
		item := SyncFromCRSItemResult{
			CRSAccountID: src.ID,
			Kind:         src.Kind,
			Name:         src.Name,
		}

		apiKey, _ := src.Credentials["api_key"].(string)
		if strings.TrimSpace(apiKey) == "" {
			item.Action = "failed"
			item.Error = "missing api_key"
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
		if baseURL, ok := credentials["base_url"].(string); !ok || strings.TrimSpace(baseURL) == "" {
			credentials["base_url"] = "https://generativelanguage.googleapis.com"
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
				Type:        AccountTypeAPIKey,
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
			item.Action = "created"
			result.Created++
			result.Items = append(result.Items, item)
			continue
		}

		existing.Extra = mergeMap(existing.Extra, extra)
		existing.Name = defaultName(src.Name, src.ID)
		existing.Platform = PlatformGemini
		existing.Type = AccountTypeAPIKey
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

		item.Action = "updated"
		result.Updated++
		result.Items = append(result.Items, item)
	}
}
