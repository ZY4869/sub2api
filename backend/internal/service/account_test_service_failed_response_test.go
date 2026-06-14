package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type accountTestFailureRepoStub struct {
	AccountRepository
	markBlacklistedCalls []accountTestBlacklistCall
	setErrorCalls        []accountTestErrorCall
	updateCalls          []*Account
}

type accountTestBlacklistCall struct {
	id            int64
	reasonCode    string
	reasonMessage string
	blacklistedAt time.Time
	purgeAt       time.Time
}

type accountTestErrorCall struct {
	id      int64
	message string
}

func (s *accountTestFailureRepoStub) MarkBlacklisted(ctx context.Context, id int64, reasonCode, reasonMessage string, blacklistedAt, purgeAt time.Time) error {
	s.markBlacklistedCalls = append(s.markBlacklistedCalls, accountTestBlacklistCall{
		id:            id,
		reasonCode:    reasonCode,
		reasonMessage: reasonMessage,
		blacklistedAt: blacklistedAt,
		purgeAt:       purgeAt,
	})
	return nil
}

func (s *accountTestFailureRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	s.setErrorCalls = append(s.setErrorCalls, accountTestErrorCall{
		id:      id,
		message: errorMsg,
	})
	return nil
}

func (s *accountTestFailureRepoStub) Update(_ context.Context, account *Account) error {
	s.updateCalls = append(s.updateCalls, account)
	return nil
}

func TestAccountTestServiceFormatFailedTestResponseMarksRecommendedUnauthorizedForReauth(t *testing.T) {
	t.Parallel()

	repo := &accountTestFailureRepoStub{}
	svc := &AccountTestService{accountRepo: repo}
	account := &Account{
		ID:             654,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		LifecycleState: AccountLifecycleNormal,
	}

	message, advice := svc.formatFailedTestResponse(
		context.Background(),
		account,
		401,
		[]byte(`{"detail":"Unauthorized"}`),
		"API returned",
	)

	require.Contains(t, message, `API returned 401: {"detail":"Unauthorized"}`)
	require.NotNil(t, advice)
	require.Equal(t, BlacklistAdviceRecommendBlacklist, advice.Decision)
	require.Equal(t, "credentials_need_reauth", advice.ReasonCode)
	require.Equal(t, "Unauthorized", advice.ReasonMessage)
	require.False(t, advice.AlreadyBlacklisted)
	require.True(t, advice.CollectFeedback)
	require.Empty(t, repo.markBlacklistedCalls)
	require.Len(t, repo.updateCalls, 1)
	require.Equal(t, StatusError, repo.updateCalls[0].Status)
	require.False(t, repo.updateCalls[0].Schedulable)
	require.Equal(t, AccountReauthReasonCode, repo.updateCalls[0].LifecycleReasonCode)
	require.NotNil(t, AccountReauthStatusFromExtra(repo.updateCalls[0].Extra))
	require.Empty(t, repo.setErrorCalls)
}

func TestAccountTestServiceFormatFailedTestResponseMarksNestedUnauthorizedMessageForReauth(t *testing.T) {
	t.Parallel()

	repo := &accountTestFailureRepoStub{}
	svc := &AccountTestService{accountRepo: repo}
	account := &Account{
		ID:             655,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		LifecycleState: AccountLifecycleNormal,
	}

	_, advice := svc.formatFailedTestResponse(
		context.Background(),
		account,
		401,
		[]byte(`{"error":{"message":"Unauthorized"}}`),
		"",
	)

	require.NotNil(t, advice)
	require.Equal(t, BlacklistAdviceRecommendBlacklist, advice.Decision)
	require.Equal(t, "Unauthorized", advice.ReasonMessage)
	require.Empty(t, repo.markBlacklistedCalls)
	require.Len(t, repo.updateCalls, 1)
	require.Equal(t, AccountReauthReasonCode, repo.updateCalls[0].LifecycleReasonCode)
}

func TestAccountTestServiceFormatFailedTestResponseMarksPlainUnauthorizedTextForReauth(t *testing.T) {
	t.Parallel()

	repo := &accountTestFailureRepoStub{}
	svc := &AccountTestService{accountRepo: repo}
	account := &Account{
		ID:             656,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		LifecycleState: AccountLifecycleNormal,
	}

	message, advice := svc.formatFailedTestResponse(
		context.Background(),
		account,
		401,
		[]byte(`unauthorized`),
		"API returned",
	)

	require.Contains(t, message, `API returned 401: unauthorized`)
	require.NotNil(t, advice)
	require.Equal(t, BlacklistAdviceRecommendBlacklist, advice.Decision)
	require.Empty(t, repo.markBlacklistedCalls)
	require.Len(t, repo.updateCalls, 1)
	require.Equal(t, AccountReauthReasonCode, repo.updateCalls[0].LifecycleReasonCode)
}

func TestAccountTestServiceFormatFailedTestResponseMarksFailoverWrappedUnauthorizedTextForReauth(t *testing.T) {
	t.Parallel()

	repo := &accountTestFailureRepoStub{}
	svc := &AccountTestService{accountRepo: repo}
	account := &Account{
		ID:             657,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		LifecycleState: AccountLifecycleNormal,
	}

	message, advice := svc.formatFailedTestResponse(
		context.Background(),
		account,
		401,
		[]byte(`upstream error: 401 (failover) unauthorized`),
		"API returned",
	)

	require.Contains(t, message, `API returned 401: upstream error: 401 (failover) unauthorized`)
	require.NotNil(t, advice)
	require.Equal(t, BlacklistAdviceRecommendBlacklist, advice.Decision)
	require.Equal(t, "credentials_need_reauth", advice.ReasonCode)
	require.Empty(t, repo.markBlacklistedCalls)
	require.Len(t, repo.updateCalls, 1)
	require.Equal(t, int64(657), repo.updateCalls[0].ID)
}

func TestAccountTestServiceFormatFailedTestResponseMarksSetupTokenUnauthorizedForReauth(t *testing.T) {
	t.Parallel()

	repo := &accountTestFailureRepoStub{}
	svc := &AccountTestService{accountRepo: repo}
	account := &Account{
		ID:             659,
		Platform:       PlatformAnthropic,
		Type:           AccountTypeSetupToken,
		LifecycleState: AccountLifecycleNormal,
	}

	_, advice := svc.formatFailedTestResponse(
		context.Background(),
		account,
		401,
		[]byte(`upstream error: 401 (failover) unauthorized`),
		"API returned",
	)

	require.NotNil(t, advice)
	require.Equal(t, BlacklistAdviceRecommendBlacklist, advice.Decision)
	require.Equal(t, AccountReauthReasonCode, advice.ReasonCode)
	require.Empty(t, repo.markBlacklistedCalls)
	require.Len(t, repo.updateCalls, 1)
	require.Equal(t, int64(659), repo.updateCalls[0].ID)
	require.Equal(t, StatusError, repo.updateCalls[0].Status)
	require.False(t, repo.updateCalls[0].Schedulable)
	require.Equal(t, AccountReauthReasonCode, repo.updateCalls[0].LifecycleReasonCode)
	require.NotNil(t, AccountReauthStatusFromExtra(repo.updateCalls[0].Extra))
}

func TestAccountTestServiceFormatFailedTestResponseBlacklistsExpiredReauthDeadline(t *testing.T) {
	t.Parallel()

	repo := &accountTestFailureRepoStub{}
	svc := &AccountTestService{accountRepo: repo}
	deadline := time.Now().Add(-time.Hour)
	required := deadline.Add(-AccountReauthGracePeriod)
	account := &Account{
		ID:             658,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		LifecycleState: AccountLifecycleNormal,
		Extra: map[string]any{
			AccountReauthStatusExtraKey: map[string]any{
				"required_since": required.Format(time.RFC3339),
				"deadline_at":    deadline.Format(time.RFC3339),
				"reason_code":    AccountReauthReasonCode,
				"message":        "Unauthorized",
			},
		},
	}

	_, advice := svc.formatFailedTestResponse(
		context.Background(),
		account,
		401,
		[]byte(`upstream error: 401 (failover) unauthorized`),
		"API returned",
	)

	require.NotNil(t, advice)
	require.Equal(t, BlacklistAdviceAutoBlacklisted, advice.Decision)
	require.Equal(t, AccountReauthDeadlineExpiredCode, advice.ReasonCode)
	require.True(t, advice.AlreadyBlacklisted)
	require.Empty(t, repo.updateCalls)
	require.Len(t, repo.markBlacklistedCalls, 1)
	require.Equal(t, int64(658), repo.markBlacklistedCalls[0].id)
	require.Equal(t, AccountReauthDeadlineExpiredCode, repo.markBlacklistedCalls[0].reasonCode)
}

func TestPersistAccountCredentialsClearsReauthState(t *testing.T) {
	t.Parallel()

	deadline := time.Now().Add(time.Hour)
	account := &Account{
		ID:          660,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusError,
		Schedulable: false,
		Credentials: map[string]any{
			"access_token": "old-token",
		},
		Extra: map[string]any{
			AccountReauthStatusExtraKey: map[string]any{
				"required_since": deadline.Add(-time.Hour).UTC().Format(time.RFC3339),
				"deadline_at":    deadline.UTC().Format(time.RFC3339),
				"reason_code":    AccountReauthReasonCode,
				"message":        "Unauthorized",
			},
		},
		LifecycleReasonCode:    AccountReauthReasonCode,
		LifecycleReasonMessage: "Unauthorized",
	}
	repo := &accountTestFailureRepoStub{}

	err := persistAccountCredentials(context.Background(), repo, account, map[string]any{
		"access_token": "new-token",
	})

	require.NoError(t, err)
	require.Len(t, repo.updateCalls, 1)
	require.Equal(t, "new-token", repo.updateCalls[0].Credentials["access_token"])
	require.Nil(t, AccountReauthStatusFromExtra(repo.updateCalls[0].Extra))
	require.Equal(t, StatusActive, repo.updateCalls[0].Status)
	require.True(t, repo.updateCalls[0].Schedulable)
	require.Empty(t, repo.updateCalls[0].LifecycleReasonCode)
	require.Empty(t, repo.updateCalls[0].LifecycleReasonMessage)
}

func TestAccountTestServiceFormatFailedTestResponseRedirectBlockedUsesControlledMessage(t *testing.T) {
	t.Parallel()

	svc := &AccountTestService{}
	message, advice := svc.formatFailedTestResponse(
		context.Background(),
		nil,
		http.StatusBadGateway,
		UpstreamRedirectBlockedBody(),
		"API returned",
	)

	require.Equal(t, UpstreamRedirectBlockedMessage, message)
	require.Nil(t, advice)
}

func TestAccountTestServiceFormatFailedTestResponseRedirectBlockedWithOriginalStatusStillUsesControlledMessage(t *testing.T) {
	t.Parallel()

	svc := &AccountTestService{}
	message, advice := svc.formatFailedTestResponse(
		context.Background(),
		nil,
		http.StatusFound,
		[]byte(`{"error":{"code":"UPSTREAM_REDIRECT_NOT_ALLOWED","message":"Upstream redirect is not allowed"}}`),
		"API returned",
	)

	require.Equal(t, UpstreamRedirectBlockedMessage, message)
	require.Nil(t, advice)
}
