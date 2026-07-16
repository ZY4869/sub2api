package service

import (
	"context"
	"net/http"
	"strconv"
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

type grokModelProbeAccess struct {
	Token   string
	BaseURL string
	Headers map[string]string
}

func (s *AccountModelImportService) detectGrokModels(ctx context.Context, account *Account) (*accountModelProbeResult, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if account.IsGrokAPIKey() || account.IsGrokOAuth() {
		access, err := s.grokModelProbeAccess(ctx, account)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(access.Token) == "" {
			return nil, infraerrors.BadRequest("ACCOUNT_CREDENTIAL_REQUIRED", "missing Grok API token for model import")
		}
		modelsURL, err := grokJoinVersionedEndpoint(access.BaseURL, "/v1/models")
		if err != nil {
			return nil, err
		}
		body, err := s.doImportGET(ctx, account, modelsURL, access.Headers, false)
		if err != nil {
			if isGrokModelListingUnsupportedError(err) {
				return grokBuildBuiltinProbeResult(ctx, account, err), nil
			}
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

func (s *AccountModelImportService) grokModelProbeAccess(ctx context.Context, account *Account) (grokModelProbeAccess, error) {
	if account == nil {
		return grokModelProbeAccess{}, nil
	}
	if account.IsGrokAPIKey() {
		token := strings.TrimSpace(account.GetGrokAPIKey())
		baseURL := strings.TrimSpace(account.GetBaseURL())
		if baseURL == "" {
			baseURL = defaultGrokAPIBaseURL
		}
		normalizedBaseURL, err := s.validateProbeBaseURL(baseURL)
		if err != nil {
			return grokModelProbeAccess{}, err
		}
		return grokModelProbeAccess{
			Token:   token,
			BaseURL: normalizedBaseURL,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Accept":        "application/json",
			},
		}, nil
	}
	if account.IsGrokOAuth() {
		token, err := s.grokProbeAccessToken(ctx, account)
		if err != nil {
			return grokModelProbeAccess{}, err
		}
		return grokModelProbeAccess{
			Token:   strings.TrimSpace(token),
			BaseURL: defaultGrokCLIBaseURL,
			Headers: map[string]string{
				"Authorization":         "Bearer " + strings.TrimSpace(token),
				"Accept":                "application/json",
				"User-Agent":            grokUpstreamUserAgent,
				"X-Grok-Client-Version": grokCLIVersion,
			},
		}, nil
	}
	return grokModelProbeAccess{}, nil
}

func (s *AccountModelImportService) grokProbeAccessToken(ctx context.Context, account *Account) (string, error) {
	if account == nil {
		return "", nil
	}
	if account.IsGrokAPIKey() {
		return strings.TrimSpace(account.GetGrokAPIKey()), nil
	}
	if !account.IsGrokOAuth() {
		return "", nil
	}
	if s != nil && s.grokTokenProvider != nil {
		token, err := s.grokTokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			logger.FromContext(ctx).Warn("account model import: grok oauth token provider failed",
				zap.Int64("account_id", account.ID),
				zap.String("platform", RoutingPlatformForAccount(account)),
				zap.String("type", account.Type),
				zap.String("error", sanitizeUpstreamErrorMessage(err.Error())),
			)
			return "", infraerrors.New(http.StatusBadGateway, "GROK_OAUTH_TOKEN_UNAVAILABLE", "Grok OAuth access token is unavailable for model import")
		}
		return strings.TrimSpace(token), nil
	}
	return strings.TrimSpace(account.GetGrokOAuthAccessToken()), nil
}

func isGrokModelListingUnsupportedError(err error) bool {
	switch grokModelListingUnsupportedStatus(err) {
	case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusGone:
		return true
	default:
		return false
	}
}

func grokModelListingUnsupportedStatus(err error) int {
	appErr := infraerrors.FromError(err)
	if appErr == nil || appErr.Metadata == nil {
		return 0
	}
	status, parseErr := strconv.Atoi(strings.TrimSpace(appErr.Metadata["upstream_status"]))
	if parseErr != nil {
		return 0
	}
	return status
}

func grokBuildBuiltinProbeResult(ctx context.Context, account *Account, cause error) *accountModelProbeResult {
	logger.FromContext(ctx).Warn("account model import: grok upstream model listing invalid; using builtin catalog", grokBuildBuiltinProbeLogFields(account, cause)...)

	result := newAccountModelProbeResult(GrokBuildTextModelIDs())
	result.Source = accountModelProbeSourceGrokBuildBuiltin
	result.Notice = grokBuildBuiltinProbeNotice
	return result
}

func grokBuildBuiltinProbeLogFields(account *Account, cause error) []zap.Field {
	fields := []zap.Field{
		zap.Int64("account_id", account.ID),
		zap.String("platform", RoutingPlatformForAccount(account)),
		zap.String("type", account.Type),
		zap.String("base_host", extractImportBaseHost(account.GetBaseURL())),
		zap.String("fallback_source", accountModelProbeSourceGrokBuildBuiltin),
	}
	if status := grokModelListingUnsupportedStatus(cause); status > 0 {
		fields = append(fields, zap.Int("upstream_status", status))
	}
	return fields
}
