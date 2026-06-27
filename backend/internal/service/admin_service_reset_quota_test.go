package service

import (
	"context"
	"net/http"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type adminResetAccountRepoStub struct {
	stubOpenAIAccountRepo
	account         *Account
	resetQuotaCalls int
}

func (r *adminResetAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func (r *adminResetAccountRepoStub) ResetQuotaUsed(context.Context, int64) error {
	r.resetQuotaCalls++
	return nil
}

func TestAdminService_ResetAccountQuota_UsesLocalResetOnly(t *testing.T) {
	repo := &adminResetAccountRepoStub{
		account: &Account{
			ID:       7001,
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	err := svc.ResetAccountQuota(context.Background(), 7001)

	require.NoError(t, err)
	require.Equal(t, 1, repo.resetQuotaCalls)
}

func TestAdminService_ResetAccountQuotaRejectsOpenAIAccounts(t *testing.T) {
	repo := &adminResetAccountRepoStub{
		account: &Account{
			ID:       7002,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	err := svc.ResetAccountQuota(context.Background(), 7002)

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
	require.Equal(t, "ACCOUNT_QUOTA_RESET_UNSUPPORTED_PLATFORM", infraerrors.Reason(err))
	require.Zero(t, repo.resetQuotaCalls)
}
