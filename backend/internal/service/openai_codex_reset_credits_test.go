package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type openAICodexResetCreditsClientStub struct {
	readSnapshot  *OpenAICodexAppServerRateLimitsSnapshot
	readErr       error
	consumeResult *OpenAICodexAppServerConsumeResult
	consumeErr    error
	consumeKeys   []string
}

func (s *openAICodexResetCreditsClientStub) ReadRateLimits(context.Context, OpenAICodexAppServerAuthTokens) (*OpenAICodexAppServerRateLimitsSnapshot, error) {
	return s.readSnapshot, s.readErr
}

func (s *openAICodexResetCreditsClientStub) ConsumeResetCredit(_ context.Context, _ OpenAICodexAppServerAuthTokens, idempotencyKey string) (*OpenAICodexAppServerConsumeResult, error) {
	s.consumeKeys = append(s.consumeKeys, idempotencyKey)
	return s.consumeResult, s.consumeErr
}

func TestOpenAICodexResetCreditServiceReadUnsupportedReturnsSnapshot(t *testing.T) {
	repo := &accountUsageCodexProbeRepo{updateExtraCh: make(chan map[string]any, 1)}
	svc := NewOpenAICodexResetCreditService(repo, &openAICodexResetCreditsClientStub{
		readErr: infraerrors.New(
			http.StatusNotImplemented,
			"OPENAI_CODEX_RESET_CREDITS_UNSUPPORTED",
			"当前 Codex app-server 不支持 OpenAI 官方真实重置",
		),
	}, nil)

	snapshot, err := svc.ReadResetCredits(context.Background(), &Account{
		ID:       7101,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "token",
			"chatgpt_account_id": "acct",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, openAIResetCreditsStatusUnsupported, snapshot.Status)
	require.Nil(t, snapshot.AvailableCount)

	updates := <-repo.updateExtraCh
	require.Equal(t, openAIResetCreditsStatusUnsupported, updates[openAIResetCreditsStatusExtraKey])
	require.Contains(t, updates[openAIResetCreditsUnsupportedReasonExtraKey], "不支持")
	require.Contains(t, updates, openAIResetCreditsAvailableCountExtraKey)
	require.Nil(t, updates[openAIResetCreditsAvailableCountExtraKey])
	require.Contains(t, updates, openAIResetCreditsUpdatedAtExtraKey)
	require.Nil(t, updates[openAIResetCreditsUpdatedAtExtraKey])
}

func TestOpenAICodexResetCreditServiceConsumeUnsupportedPersistsAndReturnsError(t *testing.T) {
	repo := &accountUsageCodexProbeRepo{updateExtraCh: make(chan map[string]any, 1)}
	svc := NewOpenAICodexResetCreditService(repo, &openAICodexResetCreditsClientStub{
		consumeErr: infraerrors.New(
			http.StatusNotImplemented,
			"OPENAI_CODEX_RESET_CREDITS_UNSUPPORTED",
			"当前 Codex app-server 不支持 OpenAI 官方真实重置",
		),
	}, nil)

	result, err := svc.ConsumeResetCredit(context.Background(), &Account{
		ID:       7102,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "token",
			"chatgpt_account_id": "acct",
		},
	}, "")

	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, isOpenAIResetCreditsUnsupportedError(err))

	updates := <-repo.updateExtraCh
	require.Equal(t, openAIResetCreditsStatusUnsupported, updates[openAIResetCreditsStatusExtraKey])
	require.Contains(t, updates, openAIResetCreditsAvailableCountExtraKey)
	require.Nil(t, updates[openAIResetCreditsAvailableCountExtraKey])
}

func TestOpenAICodexResetCreditServiceConsumeResetPersistsLatestSnapshot(t *testing.T) {
	repo := &accountUsageCodexProbeRepo{updateExtraCh: make(chan map[string]any, 1)}
	count := 2
	svc := NewOpenAICodexResetCreditService(repo, &openAICodexResetCreditsClientStub{
		consumeResult: &OpenAICodexAppServerConsumeResult{
			Status: openAIResetCreditConsumeStatusReset,
			Snapshot: &OpenAICodexAppServerRateLimitsSnapshot{
				AvailableCount: &count,
				Status:         openAIResetCreditsStatusAvailable,
				UpdatedAt:      testOpenAIResetCreditsUpdatedAt(),
				ExtraUpdates: map[string]any{
					openAIResetCreditsStatusExtraKey:           openAIResetCreditsStatusAvailable,
					openAIResetCreditsAvailableCountExtraKey:   count,
					openAIResetCreditsUpdatedAtExtraKey:        testOpenAIResetCreditsUpdatedAt().Format(time.RFC3339),
					openAIRateLimitsAppServerUpdatedAtExtraKey: testOpenAIResetCreditsUpdatedAt().Format(time.RFC3339),
				},
			},
		},
	}, nil)

	result, err := svc.ConsumeResetCredit(context.Background(), testOpenAIOAuthAccount(7103), "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, openAIResetCreditConsumeStatusReset, result.Status)
	require.NotNil(t, result.Snapshot)
	require.Equal(t, count, *result.Snapshot.AvailableCount)

	updates := <-repo.updateExtraCh
	require.Equal(t, openAIResetCreditsStatusAvailable, updates[openAIResetCreditsStatusExtraKey])
	require.Equal(t, count, updates[openAIResetCreditsAvailableCountExtraKey])
	require.Equal(t, openAIResetCreditConsumeStatusReset, updates[openAIResetCreditLastConsumeStatusExtraKey])
}

func TestOpenAICodexResetCreditServiceConsumeConflictPersistsLatestSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     string
		wantReason string
	}{
		{name: "no credit", status: openAIResetCreditConsumeStatusNoCredit, wantReason: "OPENAI_RESET_CREDITS_NO_CREDIT"},
		{name: "nothing to reset", status: openAIResetCreditConsumeStatusNothingToReset, wantReason: "OPENAI_RESET_CREDITS_NOTHING_TO_RESET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &accountUsageCodexProbeRepo{updateExtraCh: make(chan map[string]any, 1)}
			count := 0
			svc := NewOpenAICodexResetCreditService(repo, &openAICodexResetCreditsClientStub{
				consumeResult: &OpenAICodexAppServerConsumeResult{
					Status: tc.status,
					Snapshot: &OpenAICodexAppServerRateLimitsSnapshot{
						AvailableCount: &count,
						Status:         openAIResetCreditsStatusAvailable,
						UpdatedAt:      testOpenAIResetCreditsUpdatedAt(),
						ExtraUpdates: map[string]any{
							openAIResetCreditsStatusExtraKey:           openAIResetCreditsStatusAvailable,
							openAIResetCreditsAvailableCountExtraKey:   count,
							openAIResetCreditsUpdatedAtExtraKey:        testOpenAIResetCreditsUpdatedAt().Format(time.RFC3339),
							openAIRateLimitsAppServerUpdatedAtExtraKey: testOpenAIResetCreditsUpdatedAt().Format(time.RFC3339),
						},
					},
				},
			}, nil)

			result, err := svc.ConsumeResetCredit(context.Background(), testOpenAIOAuthAccount(7104), "")

			require.Error(t, err)
			require.NotNil(t, result)
			require.Contains(t, err.Error(), tc.wantReason)

			updates := <-repo.updateExtraCh
			require.Equal(t, tc.status, updates[openAIResetCreditLastConsumeStatusExtraKey])
			require.Equal(t, count, updates[openAIResetCreditsAvailableCountExtraKey])
		})
	}
}

func testOpenAIResetCreditsUpdatedAt() time.Time {
	return time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
}

func testOpenAIOAuthAccount(id int64) *Account {
	return &Account{
		ID:       id,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "token",
			"chatgpt_account_id": "acct",
		},
	}
}
