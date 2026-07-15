package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	accountModelProbeSourceGrokSSOCapability = "grok_sso_capability"
	accountModelProbeSourceGrokBuildBuiltin  = "grok_build_builtin_catalog"
	grokBuildBuiltinProbeNotice              = "Grok upstream model listing returned an unexpected structure; using built-in Grok Build text model catalog"
)

func (s *AccountModelImportService) detectGrokModels(ctx context.Context, account *Account) (*accountModelProbeResult, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if account.IsGrokAPIKey() || account.IsGrokOAuth() {
		apiKey := strings.TrimSpace(account.GetGrokAPIKey())
		if apiKey == "" && account.IsGrokOAuth() {
			apiKey = strings.TrimSpace(account.GetGrokOAuthAccessToken())
		}
		if apiKey == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Grok API token for model import")
		}
		baseURL := strings.TrimSpace(account.GetBaseURL())
		if baseURL == "" {
			baseURL = "https://api.x.ai"
		}
		normalizedBaseURL, err := s.validateProbeBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		baseURL = normalizedBaseURL
		body, err := s.doImportGET(ctx, account, strings.TrimRight(baseURL, "/")+"/v1/models", map[string]string{
			"Authorization": "Bearer " + apiKey,
			"Accept":        "application/json",
		}, false)
		if err != nil {
			return nil, err
		}
		models, err := parseOpenAIModelListForAccount(account, body)
		if err == nil {
			models = canonicalizeGrokDetectedModels(models)
			if len(models) > 0 {
				return newAccountModelProbeResult(models), nil
			}
		}
		return grokBuildBuiltinProbeResult(ctx, account, err), nil
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

func grokBuildBuiltinProbeResult(ctx context.Context, account *Account, cause error) *accountModelProbeResult {
	fields := []zap.Field{
		zap.Int64("account_id", account.ID),
		zap.String("platform", RoutingPlatformForAccount(account)),
		zap.String("type", account.Type),
		zap.String("base_host", extractImportBaseHost(account.GetBaseURL())),
		zap.String("fallback_source", accountModelProbeSourceGrokBuildBuiltin),
	}
	if cause != nil {
		fields = append(fields, zap.Error(cause))
	}
	logger.FromContext(ctx).Warn("account model import: grok upstream model listing invalid; using builtin catalog", fields...)

	result := newAccountModelProbeResult(GrokBuildTextModelIDs())
	result.Source = accountModelProbeSourceGrokBuildBuiltin
	result.Notice = grokBuildBuiltinProbeNotice
	return result
}
