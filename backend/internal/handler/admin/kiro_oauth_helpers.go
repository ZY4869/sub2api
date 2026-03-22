package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func refreshKiroOAuthAccount(
	ctx context.Context,
	adminService service.AdminService,
	kiroOAuthService *service.KiroOAuthService,
	account *service.Account,
) (*service.Account, error) {
	if kiroOAuthService == nil {
		return nil, errors.InternalServer("KIRO_OAUTH_UNAVAILABLE", "kiro oauth service is unavailable")
	}
	if account == nil || account.Platform != service.PlatformKiro || account.Type != service.AccountTypeOAuth {
		return nil, errors.BadRequest("KIRO_INVALID_ACCOUNT", "account is not a Kiro OAuth account")
	}

	tokenInfo, err := kiroOAuthService.RefreshAccountToken(ctx, account)
	if err != nil {
		return nil, err
	}
	newCredentials := kiroOAuthService.BuildAccountCredentials(tokenInfo)
	newCredentials = service.MergeCredentials(account.Credentials, newCredentials)

	return adminService.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{
		Type:        service.AccountTypeOAuth,
		Credentials: newCredentials,
	})
}
