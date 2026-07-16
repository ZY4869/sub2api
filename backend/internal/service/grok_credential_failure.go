package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

const (
	GrokCredentialUnavailableClientMessage = "No healthy Grok OAuth account is currently available"

	grokCredentialScopeAccount  = "account"
	grokCredentialScopeProvider = "provider"

	GrokCredentialReasonRevoked          = "grok_oauth_credential_revoked"
	GrokCredentialReasonMissing          = "grok_oauth_credentials_missing"
	GrokCredentialReasonEntitlement      = "grok_oauth_entitlement_action_required"
	GrokCredentialReasonProxyInvalid     = "grok_oauth_proxy_invalid"
	GrokCredentialReasonRefreshTransient = "grok_oauth_refresh_transient"
	GrokCredentialReasonProviderConfig   = "grok_oauth_provider_config"
	GrokCredentialReasonProviderDown     = "grok_oauth_provider_unavailable"
)

type grokCredentialFailureClass struct {
	scope     string
	reason    string
	permanent bool
	transient bool
	message   string
}

func classifyGrokCredentialFailure(account *Account, err error) grokCredentialFailureClass {
	stableReason := strings.ToLower(strings.TrimSpace(infraerrors.Reason(err)))
	message := ""
	if err != nil {
		message = strings.ToLower(err.Error())
	}
	contains := func(values ...string) bool {
		for _, value := range values {
			if strings.Contains(stableReason, value) || strings.Contains(message, value) {
				return true
			}
		}
		return false
	}

	switch {
	case errors.Is(err, errGrokOAuthRefreshTokenMissing), errors.Is(err, errGrokOAuthAccessTokenMissing), errors.Is(err, errGrokOAuthAccessTokenExpired):
		return grokCredentialFailureClass{scope: grokCredentialScopeAccount, reason: GrokCredentialReasonMissing, permanent: true, message: "Grok OAuth credentials are missing or expired"}
	case contains("invalid_grant", "invalid_refresh_token", "token_expired", "refresh_token_reused", "refresh_token_invalidated", "app_session_terminated"):
		return grokCredentialFailureClass{scope: grokCredentialScopeAccount, reason: GrokCredentialReasonRevoked, permanent: true, message: "Grok OAuth credentials require account action"}
	case contains("entitlement_denied", "access_denied", "subscription required", "no active grok subscription"):
		return grokCredentialFailureClass{scope: grokCredentialScopeAccount, reason: GrokCredentialReasonEntitlement, permanent: true, message: "Grok OAuth entitlement requires account action"}
	case contains("grok_oauth_proxy_not_found", "proxy configuration is invalid"):
		return grokCredentialFailureClass{scope: grokCredentialScopeAccount, reason: GrokCredentialReasonProxyInvalid, permanent: true, message: "Grok OAuth account proxy configuration is invalid"}
	case errors.Is(err, errGrokOAuthRefreshNotConfigured), contains("invalid_client", "unauthorized_client", "invalid_scope", "unknown scope", "not configured"):
		return grokCredentialFailureClass{scope: grokCredentialScopeProvider, reason: GrokCredentialReasonProviderConfig, message: "Grok OAuth provider configuration is unavailable"}
	case contains("status 429", "status 500", "status 502", "status 503", "status 504", "provider unavailable", "temporarily unavailable") && (account == nil || account.ProxyID == nil):
		return grokCredentialFailureClass{scope: grokCredentialScopeProvider, reason: GrokCredentialReasonProviderDown, message: "Grok OAuth provider is temporarily unavailable"}
	default:
		return grokCredentialFailureClass{scope: grokCredentialScopeAccount, reason: GrokCredentialReasonRefreshTransient, transient: true, message: "Grok OAuth credential refresh is temporarily unavailable"}
	}
}

func (s *GrokGatewayService) handleGrokCredentialFailure(ctx context.Context, c *gin.Context, account *Account, err error) error {
	class := classifyGrokCredentialFailure(account, err)
	if account != nil && s.accountRepo != nil {
		if class.permanent {
			_ = s.accountRepo.SetError(ctx, account.ID, string(class.reason))
			_ = s.accountRepo.SetSchedulable(ctx, account.ID, false)
		} else if class.transient {
			_ = s.accountRepo.SetTempUnschedulable(ctx, account.ID, time.Now().Add(tokenRefreshTempUnschedDuration), string(class.reason))
		}
	}
	return s.newGrokCredentialFailover(c, account, class)
}

func (s *GrokGatewayService) newGrokCredentialFailover(c *gin.Context, account *Account, class grokCredentialFailureClass) error {
	if strings.TrimSpace(class.message) == "" {
		class.message = "Grok OAuth credentials are unavailable"
	}
	accountID := int64(0)
	accountName := ""
	if account != nil {
		accountID = account.ID
		accountName = account.Name
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:    PlatformGrok,
		AccountID:   accountID,
		AccountName: accountName,
		Kind:        "credential_failover",
		Message:     class.message,
		Detail:      fmt.Sprintf("stage=account_auth scope=%s reason=%s", class.scope, class.reason),
	})
	return &UpstreamFailoverError{
		StatusCode:             http.StatusServiceUnavailable,
		ResponseBody:           []byte(fmt.Sprintf(`{"error":{"type":"grok_oauth_unavailable","message":%q}}`, GrokCredentialUnavailableClientMessage)),
		RetryableOnSameAccount: false,
		TempUnscheduleAccount:  class.transient,
	}
}
