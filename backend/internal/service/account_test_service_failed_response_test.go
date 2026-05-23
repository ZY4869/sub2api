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

func TestAccountTestServiceFormatFailedTestResponseAutoBlacklistsRecommendedUnauthorized(t *testing.T) {
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
	require.Equal(t, BlacklistAdviceAutoBlacklisted, advice.Decision)
	require.Equal(t, "credentials_need_reauth", advice.ReasonCode)
	require.Equal(t, "Unauthorized", advice.ReasonMessage)
	require.True(t, advice.AlreadyBlacklisted)
	require.False(t, advice.CollectFeedback)
	require.Len(t, repo.markBlacklistedCalls, 1)
	require.Equal(t, int64(654), repo.markBlacklistedCalls[0].id)
	require.Equal(t, "credentials_need_reauth", repo.markBlacklistedCalls[0].reasonCode)
	require.Equal(t, "Unauthorized", repo.markBlacklistedCalls[0].reasonMessage)
	require.Empty(t, repo.setErrorCalls)
}

func TestAccountTestServiceFormatFailedTestResponseAutoBlacklistsNestedUnauthorizedMessage(t *testing.T) {
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
	require.Equal(t, BlacklistAdviceAutoBlacklisted, advice.Decision)
	require.Equal(t, "Unauthorized", advice.ReasonMessage)
	require.Len(t, repo.markBlacklistedCalls, 1)
	require.Equal(t, "credentials_need_reauth", repo.markBlacklistedCalls[0].reasonCode)
}

func TestAccountTestServiceFormatFailedTestResponseAutoBlacklistsPlainUnauthorizedText(t *testing.T) {
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
	require.Equal(t, BlacklistAdviceAutoBlacklisted, advice.Decision)
	require.Len(t, repo.markBlacklistedCalls, 1)
	require.Equal(t, "credentials_need_reauth", repo.markBlacklistedCalls[0].reasonCode)
}

func TestAccountTestServiceFormatFailedTestResponseAutoBlacklistsFailoverWrappedUnauthorizedText(t *testing.T) {
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
	require.Equal(t, BlacklistAdviceAutoBlacklisted, advice.Decision)
	require.Equal(t, "credentials_need_reauth", advice.ReasonCode)
	require.Len(t, repo.markBlacklistedCalls, 1)
	require.Equal(t, int64(657), repo.markBlacklistedCalls[0].id)
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
