package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type kiroAdminAccountRepoStub struct {
	AccountRepository
	account  *Account
	created  *Account
	updated  *Account
	createID int64
}

func (s *kiroAdminAccountRepoStub) Create(_ context.Context, account *Account) error {
	s.createID++
	if account.ID == 0 {
		account.ID = s.createID
	}
	s.account = account
	s.created = account
	return nil
}

func (s *kiroAdminAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.account != nil && s.account.ID == id {
		return s.account, nil
	}
	return nil, ErrAccountNotFound
}

func (s *kiroAdminAccountRepoStub) Update(_ context.Context, account *Account) error {
	s.account = account
	s.updated = account
	return nil
}

func TestAdminServiceCreateAccount_NormalizesKiroLegacyRegionForStorage(t *testing.T) {
	repo := &kiroAdminAccountRepoStub{}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "kiro-import",
		Platform:             PlatformKiro,
		Type:                 AccountTypeOAuth,
		Credentials:          map[string]any{"access_token": "kiro-access", "region": "ap-southeast-1"},
		Concurrency:          1,
		Priority:             1,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, "ap-southeast-1", account.GetCredential("api_region"))
	_, hasLegacy := account.Credentials["region"]
	require.False(t, hasLegacy)
	require.Equal(t, "ap-southeast-1", repo.created.GetCredential("api_region"))
}

func TestAdminServiceUpdateAccount_NormalizesStoredKiroLegacyRegionOnSave(t *testing.T) {
	repo := &kiroAdminAccountRepoStub{
		account: &Account{
			ID:       42,
			Name:     "kiro-old",
			Platform: PlatformKiro,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"access_token": "kiro-access",
				"region":       "eu-west-1",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.UpdateAccount(context.Background(), 42, &UpdateAccountInput{
		Name: "kiro-renamed",
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, "kiro-renamed", account.Name)
	require.Equal(t, "eu-west-1", account.GetCredential("api_region"))
	_, hasLegacy := account.Credentials["region"]
	require.False(t, hasLegacy)
	require.NotNil(t, repo.updated)
	require.Equal(t, "eu-west-1", repo.updated.GetCredential("api_region"))
}
