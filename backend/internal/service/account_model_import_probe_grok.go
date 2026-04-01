package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const accountModelProbeSourceGrokSSOCapability = "grok_sso_capability"

func (s *AccountModelImportService) detectGrokModels(ctx context.Context, account *Account) (*accountModelProbeResult, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if account.IsGrokAPIKey() {
		apiKey := strings.TrimSpace(account.GetGrokAPIKey())
		if apiKey == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Grok API key for model import")
		}
		baseURL := strings.TrimSpace(account.GetBaseURL())
		if baseURL == "" {
			baseURL = "https://api.x.ai"
		}
		body, err := s.doImportGET(ctx, account, strings.TrimRight(baseURL, "/")+"/v1/models", map[string]string{
			"Authorization": "Bearer " + apiKey,
			"Accept":        "application/json",
		}, false)
		if err != nil {
			return nil, err
		}
		models, err := parseOpenAIModelListForAccount(account, body)
		if err != nil {
			return nil, err
		}
		return newAccountModelProbeResult(canonicalizeGrokDetectedModels(models)), nil
	}
	if account.IsGrokSSO() {
		visibleModels := GrokVisibleModelIDsForAccount(account)
		result := newAccountModelProbeResult(visibleModels)
		result.Source = accountModelProbeSourceGrokSSOCapability
		if len(account.GetModelMapping()) > 0 {
			result.Notice = "SSO accounts expose capability-derived models filtered by account model_mapping"
		} else {
			result.Notice = "SSO accounts expose capability-derived models before upstream probing"
		}
		return result, nil
	}
	return nil, infraerrors.BadRequest("ACCOUNT_TYPE_UNSUPPORTED", "current Grok account type does not support model import")
}
