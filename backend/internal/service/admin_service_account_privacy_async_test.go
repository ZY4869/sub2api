//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

type createAccountPrivacyRepoStub struct {
	AccountRepository
	created       *Account
	updateExtraCh chan map[string]any
}

func (s *createAccountPrivacyRepoStub) Create(_ context.Context, account *Account) error {
	if account.ID == 0 {
		account.ID = 1
	}
	s.created = account
	return nil
}

func (s *createAccountPrivacyRepoStub) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	copied := make(map[string]any, len(updates))
	for k, v := range updates {
		copied[k] = v
	}
	s.updateExtraCh <- copied
	return nil
}

func TestAdminServiceCreateAccount_OpenAIOAuthEnsuresPrivacyAsync(t *testing.T) {
	reqHeadersCh := make(chan http.Header, 1)
	reqURLCh := make(chan string, 1)
	repo := &createAccountPrivacyRepoStub{updateExtraCh: make(chan map[string]any, 1)}
	svc := &adminServiceImpl{
		accountRepo: repo,
		privacyClientFactory: func(_ string) (*req.Client, error) {
			client := req.C()
			client.GetClient().Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				reqHeadersCh <- req.Header.Clone()
				reqURLCh <- req.URL.String()
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			})
			return client, nil
		},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		Credentials:          map[string]any{"access_token": "test-token"},
		Concurrency:          1,
		Priority:             1,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)

	select {
	case updates := <-repo.updateExtraCh:
		require.Equal(t, PrivacyModeTrainingOff, updates["privacy_mode"])
	case <-time.After(2 * time.Second):
		t.Fatal("expected CreateAccount to trigger OpenAI privacy ensure")
	}

	select {
	case reqURL := <-reqURLCh:
		require.Contains(t, reqURL, openAISettingsURL)
	case <-time.After(2 * time.Second):
		t.Fatal("expected OpenAI privacy request URL to be captured")
	}

	select {
	case headers := <-reqHeadersCh:
		require.Equal(t, "application/json", headers.Get("Accept"))
		require.Equal(t, "cors", headers.Get("sec-fetch-mode"))
		require.Equal(t, "same-origin", headers.Get("sec-fetch-site"))
		require.Equal(t, "empty", headers.Get("sec-fetch-dest"))
	case <-time.After(2 * time.Second):
		t.Fatal("expected OpenAI privacy request headers to be captured")
	}

	require.Equal(t, PrivacyModeTrainingOff, account.Extra["privacy_mode"])
}
