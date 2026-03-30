package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func refreshCopilotOAuthAccount(
	ctx context.Context,
	adminService service.AdminService,
	copilotOAuthService *service.CopilotOAuthService,
	account *service.Account,
) (*service.Account, error) {
	if copilotOAuthService == nil {
		return nil, errors.InternalServer("COPILOT_OAUTH_UNAVAILABLE", "copilot oauth service is unavailable")
	}
	if account == nil || account.Platform != service.PlatformCopilot || account.Type != service.AccountTypeOAuth {
		return nil, errors.BadRequest("COPILOT_INVALID_ACCOUNT", "account is not a Copilot OAuth account")
	}

	result, err := copilotOAuthService.RefreshAccountState(ctx, account)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return account, nil
	}
	return adminService.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{
		Credentials: result.Credentials,
		Extra:       result.Extra,
	})
}
