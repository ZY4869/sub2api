package service

import (
	"context"
	"net/http"
	"testing"
	"time"

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

type adminResetCreditConsumerStub struct {
	calls  int
	status string
	err    error
}

func (s *adminResetCreditConsumerStub) ConsumeResetCredit(_ context.Context, _ *Account, _ string) (*OpenAICodexResetCreditConsumeResult, error) {
	s.calls++
	count := 1
	return &OpenAICodexResetCreditConsumeResult{
		Status: s.status,
		Snapshot: &OpenAICodexResetCreditsSnapshot{
			AvailableCount: &count,
			UpdatedAt:      time.Now().UTC(),
			Source:         openAIResetCreditsSourceCodexAppServer,
		},
	}, s.err
}

func TestAdminService_ResetAccountQuota_OpenAIOAuthConsumesRealCredit(t *testing.T) {
	repo := &adminResetAccountRepoStub{
		account: &Account{
			ID:       7001,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
		},
	}
	consumer := &adminResetCreditConsumerStub{status: openAIResetCreditConsumeStatusReset}
	svc := &adminServiceImpl{accountRepo: repo, openAIResetCreditService: consumer}

	err := svc.ResetAccountQuota(context.Background(), 7001)

	require.NoError(t, err)
	require.Equal(t, 1, consumer.calls)
	require.Zero(t, repo.resetQuotaCalls)
}

func TestAdminService_ResetAccountQuota_NonOpenAIOAuthUsesLocalReset(t *testing.T) {
	repo := &adminResetAccountRepoStub{
		account: &Account{
			ID:       7002,
			Platform: PlatformAnthropic,
			Type:     AccountTypeOAuth,
		},
	}
	consumer := &adminResetCreditConsumerStub{status: openAIResetCreditConsumeStatusReset}
	svc := &adminServiceImpl{accountRepo: repo, openAIResetCreditService: consumer}

	err := svc.ResetAccountQuota(context.Background(), 7002)

	require.NoError(t, err)
	require.Zero(t, consumer.calls)
	require.Equal(t, 1, repo.resetQuotaCalls)
}

func TestAdminService_ResetAccountQuota_OpenAIOAuthRequiresAppServer(t *testing.T) {
	repo := &adminResetAccountRepoStub{
		account: &Account{
			ID:       7003,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	err := svc.ResetAccountQuota(context.Background(), 7003)

	require.Error(t, err)
	require.Zero(t, repo.resetQuotaCalls)
}

func TestAdminService_ResetAccountQuota_OpenAIOAuthUnsupportedDoesNotLocalReset(t *testing.T) {
	repo := &adminResetAccountRepoStub{
		account: &Account{
			ID:       7004,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
		},
	}
	consumer := &adminResetCreditConsumerStub{
		err: infraerrors.New(
			http.StatusNotImplemented,
			"OPENAI_CODEX_RESET_CREDITS_UNSUPPORTED",
			"当前 Codex app-server 不支持 OpenAI 官方真实重置",
		),
	}
	svc := &adminServiceImpl{accountRepo: repo, openAIResetCreditService: consumer}

	err := svc.ResetAccountQuota(context.Background(), 7004)

	require.Error(t, err)
	require.Equal(t, 1, consumer.calls)
	require.Zero(t, repo.resetQuotaCalls)
}
