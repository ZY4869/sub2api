package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *CRSSyncService) syncCRSClaudeConsoleAccounts(ctx context.Context, input SyncFromCRSInput, exported *crsExportResponse, result *SyncFromCRSResult, selectedSet map[string]struct{}, proxies *[]Proxy, now string) {
	// Claude Console API Key -> sub2api anthropic apikey
	for _, src := range exported.Data.ClaudeConsoleAccounts {
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
		priority := clampPriority(src.Priority)
		concurrency := 3
		if src.MaxConcurrentTasks > 0 {
			concurrency = src.MaxConcurrentTasks
		}
		status := mapCRSStatus(src.IsActive, src.Status)

		extra := map[string]any{
			"crs_account_id": src.ID,
			"crs_kind":       src.Kind,
			"crs_synced_at":  now,
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
				Type:        AccountTypeAPIKey,
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
			item.Action = "created"
			result.Created++
			result.Items = append(result.Items, item)
			continue
		}

		existing.Extra = mergeMap(existing.Extra, extra)
		existing.Name = defaultName(src.Name, src.ID)
		existing.Platform = PlatformAnthropic
		existing.Type = AccountTypeAPIKey
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

		item.Action = "updated"
		result.Updated++
		result.Items = append(result.Items, item)
	}
}
