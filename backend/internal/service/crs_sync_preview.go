package service

import (
	"context"
	"fmt"
	"strings"
)

// PreviewFromCRS connects to CRS, fetches all accounts, and classifies them
// as new or existing by batch-querying local crs_account_id mappings.
func (s *CRSSyncService) PreviewFromCRS(ctx context.Context, input SyncFromCRSInput) (*PreviewFromCRSResult, error) {
	exported, err := s.fetchCRSExport(ctx, input.BaseURL, input.Username, input.Password)
	if err != nil {
		return nil, err
	}

	// Batch query all existing CRS account IDs
	existingCRSIDs, err := s.accountRepo.ListCRSAccountIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing CRS accounts: %w", err)
	}

	result := &PreviewFromCRSResult{
		NewAccounts:      make([]CRSPreviewAccount, 0),
		ExistingAccounts: make([]CRSPreviewAccount, 0),
	}

	classify := func(crsID, kind, name, platform, accountType string) {
		preview := CRSPreviewAccount{
			CRSAccountID: crsID,
			Kind:         kind,
			Name:         defaultName(name, crsID),
			Platform:     platform,
			Type:         accountType,
		}
		if _, exists := existingCRSIDs[crsID]; exists {
			result.ExistingAccounts = append(result.ExistingAccounts, preview)
		} else {
			result.NewAccounts = append(result.NewAccounts, preview)
		}
	}

	for _, src := range exported.Data.ClaudeAccounts {
		authType := strings.TrimSpace(src.AuthType)
		if authType == "" {
			authType = AccountTypeOAuth
		}
		classify(src.ID, src.Kind, src.Name, PlatformAnthropic, authType)
	}
	for _, src := range exported.Data.ClaudeConsoleAccounts {
		classify(src.ID, src.Kind, src.Name, PlatformAnthropic, AccountTypeAPIKey)
	}
	for _, src := range exported.Data.OpenAIOAuthAccounts {
		classify(src.ID, src.Kind, src.Name, PlatformOpenAI, AccountTypeOAuth)
	}
	for _, src := range exported.Data.OpenAIResponsesAccounts {
		classify(src.ID, src.Kind, src.Name, PlatformOpenAI, AccountTypeAPIKey)
	}
	for _, src := range exported.Data.GeminiOAuthAccounts {
		classify(src.ID, src.Kind, src.Name, PlatformGemini, AccountTypeOAuth)
	}
	for _, src := range exported.Data.GeminiAPIKeyAccounts {
		classify(src.ID, src.Kind, src.Name, PlatformGemini, AccountTypeAPIKey)
	}

	return result, nil
}
