//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type adminAccountBaseURLRepoStub struct {
	AccountRepository
	account *Account
	created *Account
	updated *Account
}

func (s *adminAccountBaseURLRepoStub) Create(_ context.Context, account *Account) error {
	s.created = account
	return nil
}

func (s *adminAccountBaseURLRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.account != nil && s.account.ID == id {
		return s.account, nil
	}
	return nil, ErrAccountNotFound
}

func (s *adminAccountBaseURLRepoStub) Update(_ context.Context, account *Account) error {
	s.account = account
	s.updated = account
	return nil
}

func TestAdminServiceCreateAccount_InvalidBaseURLReturnsStructuredError(t *testing.T) {
	repo := &adminAccountBaseURLRepoStub{}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "invalid-openai",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeAPIKey,
		Credentials:          map[string]any{"api_key": "sk-test", "base_url": "http://127.0.0.1"},
		SkipDefaultGroupBind: true,
	})

	require.Nil(t, account)
	require.Error(t, err)
	require.Equal(t, accountInvalidBaseURLCode, infraerrors.Reason(err))
	require.Nil(t, repo.created)
}

func TestAdminServiceUpdateAccount_NormalizesAllowedBaseURL(t *testing.T) {
	repo := &adminAccountBaseURLRepoStub{
		account: &Account{
			ID:       42,
			Name:     "openai",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://api.openai.com",
			},
		},
	}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	account, err := svc.UpdateAccount(context.Background(), 42, &UpdateAccountInput{
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": " https://api.openai.com/ ",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotNil(t, repo.updated)
	require.Equal(t, "https://api.openai.com", repo.updated.Credentials["base_url"])
}

func TestAdminServiceCreateBaiduDocumentAIRejectsDisallowedAsyncBaseURL(t *testing.T) {
	repo := &adminAccountBaseURLRepoStub{}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.DocumentAIHosts = []string{"paddleocr.aistudio-app.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "doc-ai",
		Platform: PlatformBaiduDocumentAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"async_bearer_token": "token",
			"async_base_url":     "https://example.com/api",
		},
		SkipDefaultGroupBind: true,
	})

	require.Nil(t, account)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, baiduDocumentAIInvalidCredentialsCode, infraerrors.Reason(err))
	require.Contains(t, err.Error(), "async_base_url")
	require.Nil(t, repo.created)
}

func TestAdminServiceCreateBaiduDocumentAIRejectsDisallowedDirectAPIURL(t *testing.T) {
	repo := &adminAccountBaseURLRepoStub{}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.DocumentAIHosts = []string{"paddleocr.aistudio-app.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "doc-ai",
		Platform: PlatformBaiduDocumentAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"direct_token": "direct-token",
			"direct_api_urls": map[string]any{
				DocumentAIModelPPOCRV5Server: "https://example.com/api/v2/ocr/direct",
			},
		},
		SkipDefaultGroupBind: true,
	})

	require.Nil(t, account)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, baiduDocumentAIInvalidCredentialsCode, infraerrors.Reason(err))
	require.Contains(t, err.Error(), "direct_api_urls")
	require.Nil(t, repo.created)
}

func TestAdminServiceCreateBaiduDocumentAINormalizesAllowedURLs(t *testing.T) {
	repo := &adminAccountBaseURLRepoStub{}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.DocumentAIHosts = []string{"paddleocr.aistudio-app.com"}
	svc := &adminServiceImpl{accountRepo: repo, cfg: cfg}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "doc-ai",
		Platform: PlatformBaiduDocumentAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"async_bearer_token": "token",
			"async_base_url":     " https://paddleocr.aistudio-app.com/api/v2/ocr/ ",
			"direct_token":       "direct-token",
			"direct_api_urls": map[string]any{
				DocumentAIModelPPOCRV5Server: " https://paddleocr.aistudio-app.com/api/v2/ocr/direct/ ",
			},
		},
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotNil(t, repo.created)
	require.Equal(t, "https://paddleocr.aistudio-app.com/api/v2/ocr", repo.created.Credentials["async_base_url"])
	directURLs, ok := repo.created.Credentials["direct_api_urls"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://paddleocr.aistudio-app.com/api/v2/ocr/direct", directURLs[DocumentAIModelPPOCRV5Server])
}
