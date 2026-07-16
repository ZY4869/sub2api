package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGrokCredentialFailurePermanentBlocksAccount(t *testing.T) {
	repo := &grokCredentialFailureRepo{}
	account := &Account{ID: 12, Platform: PlatformGrok, Type: AccountTypeOAuth}
	svc := &GrokGatewayService{accountRepo: repo}

	err := svc.handleGrokCredentialFailure(context.Background(), nil, account, errGrokOAuthRefreshTokenMissing)

	require.Error(t, err)
	var failover *UpstreamFailoverError
	require.ErrorAs(t, err, &failover)
	require.Equal(t, 503, failover.StatusCode)
	require.Contains(t, string(failover.ResponseBody), "grok_oauth_unavailable")
	require.Equal(t, int64(12), repo.errorAccountID)
	require.Equal(t, GrokCredentialReasonMissing, repo.errorMessage)
	require.Equal(t, int64(12), repo.schedulableAccountID)
	require.False(t, repo.schedulable)
}

func TestGrokCredentialFailureTransientTempUnschedulesAccount(t *testing.T) {
	repo := &grokCredentialFailureRepo{}
	account := &Account{ID: 13, Platform: PlatformGrok, Type: AccountTypeOAuth}
	svc := &GrokGatewayService{accountRepo: repo}

	err := svc.handleGrokCredentialFailure(context.Background(), nil, account, errors.New("network timeout"))

	require.Error(t, err)
	var failover *UpstreamFailoverError
	require.ErrorAs(t, err, &failover)
	require.Equal(t, 503, failover.StatusCode)
	require.True(t, failover.TempUnscheduleAccount)
	require.Equal(t, int64(13), repo.tempAccountID)
	require.Equal(t, GrokCredentialReasonRefreshTransient, repo.tempReason)
	require.True(t, repo.tempUntil.After(time.Now()))
}

type grokCredentialFailureRepo struct {
	AccountRepository
	errorAccountID       int64
	errorMessage         string
	schedulableAccountID int64
	schedulable          bool
	tempAccountID        int64
	tempUntil            time.Time
	tempReason           string
}

func (r *grokCredentialFailureRepo) SetError(_ context.Context, id int64, errorMsg string) error {
	r.errorAccountID = id
	r.errorMessage = errorMsg
	return nil
}

func (r *grokCredentialFailureRepo) SetSchedulable(_ context.Context, id int64, schedulable bool) error {
	r.schedulableAccountID = id
	r.schedulable = schedulable
	return nil
}

func (r *grokCredentialFailureRepo) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.tempAccountID = id
	r.tempUntil = until
	r.tempReason = reason
	return nil
}
