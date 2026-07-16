//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateGrokAccountInput_APIKeyIgnoresLegacyTierFields(t *testing.T) {
	err := validateGrokAccountInput(
		PlatformGrok,
		AccountTypeAPIKey,
		map[string]any{"api_key": "xai-key"},
		map[string]any{
			"grok_tier":         "not-a-real-tier",
			"grok_capabilities": "legacy-string",
		},
	)

	require.NoError(t, err)
}

func TestNormalizeGrokExtraForStorageByType_APIKeyDropsTierFields(t *testing.T) {
	normalized := normalizeGrokExtraForStorageByType(AccountTypeAPIKey, map[string]any{
		"grok_tier":         GrokTierHeavy,
		"grok_capabilities": map[string]any{"vision": true},
		"manual_models":     []any{"grok-4"},
	})

	require.NotNil(t, normalized)
	require.NotContains(t, normalized, "grok_tier")
	require.NotContains(t, normalized, "grok_capabilities")
	require.Equal(t, []any{"grok-4"}, normalized["manual_models"])
}

func TestNormalizeGrokExtraForStorageByType_SSOKeepsTierAndCapabilities(t *testing.T) {
	normalized := normalizeGrokExtraForStorageByType(AccountTypeSSO, map[string]any{
		"grok_tier": GrokTierSuper,
	})

	require.Equal(t, GrokTierSuper, normalized["grok_tier"])
	capabilities, ok := normalized["grok_capabilities"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, capabilities)
}

type grokSSOBoundaryAccountRepoStub struct {
	AccountRepository
	account *Account
	created *Account
	updated *Account
}

func (s *grokSSOBoundaryAccountRepoStub) Create(_ context.Context, account *Account) error {
	s.created = account
	s.account = account
	return nil
}

func (s *grokSSOBoundaryAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.account != nil && s.account.ID == id {
		return s.account, nil
	}
	return nil, ErrAccountNotFound
}

func (s *grokSSOBoundaryAccountRepoStub) Update(_ context.Context, account *Account) error {
	s.updated = account
	s.account = account
	return nil
}

func TestAdminServiceCreateAccountRejectsNewGrokSSO(t *testing.T) {
	repo := &grokSSOBoundaryAccountRepoStub{}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "legacy-grok-sso",
		Platform: PlatformGrok,
		Type:     AccountTypeSSO,
		Credentials: map[string]any{
			"sso_token": "legacy-token",
		},
		Extra: map[string]any{
			"grok_tier": GrokTierSuper,
		},
		SkipDefaultGroupBind: true,
	})

	require.Nil(t, account)
	require.Error(t, err)
	require.Equal(t, grokSSOCreateDisabledCode, infraerrors.Reason(err))
	require.Nil(t, repo.created)
}

func TestAdminServiceUpdateAccountRejectsGrokSSOConversion(t *testing.T) {
	repo := &grokSSOBoundaryAccountRepoStub{
		account: &Account{
			ID:       42,
			Name:     "grok-apikey",
			Platform: PlatformGrok,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"api_key": "xai-key",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.UpdateAccount(context.Background(), 42, &UpdateAccountInput{
		Type: AccountTypeSSO,
		Credentials: map[string]any{
			"sso_token": "legacy-token",
		},
		Extra: map[string]any{
			"grok_tier": GrokTierBasic,
		},
	})

	require.Nil(t, account)
	require.Error(t, err)
	require.Equal(t, grokSSOConversionDisabledCode, infraerrors.Reason(err))
	require.Nil(t, repo.updated)
}

func TestAdminServiceUpdateAccountAllowsExistingGrokSSOMaintenance(t *testing.T) {
	repo := &grokSSOBoundaryAccountRepoStub{
		account: &Account{
			ID:       43,
			Name:     "legacy-grok-sso",
			Platform: PlatformGrok,
			Type:     AccountTypeSSO,
			Status:   StatusActive,
			Credentials: map[string]any{
				"sso_token": "old-token",
			},
			Extra: map[string]any{
				"grok_tier": GrokTierBasic,
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.UpdateAccount(context.Background(), 43, &UpdateAccountInput{
		Credentials: map[string]any{
			"sso_token": "new-token",
		},
		Extra: map[string]any{
			"grok_tier": GrokTierHeavy,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.NotNil(t, repo.updated)
	require.Equal(t, AccountTypeSSO, repo.updated.Type)
	require.Equal(t, "new-token", repo.updated.Credentials["sso_token"])
	require.Equal(t, GrokTierHeavy, repo.updated.Extra["grok_tier"])
}

func TestApplyAccountAutoRenewConfig_RequiresExpiration(t *testing.T) {
	enabled := true
	account := &Account{}

	err := applyAccountAutoRenewConfig(account, &enabled, nil, false)

	require.Error(t, err)
	require.False(t, account.AutoRenewEnabled)
	require.Equal(t, AccountAutoRenewPeriodMonth, account.AutoRenewPeriod)
}

func TestApplyAccountAutoRenewConfig_ClearingExpirationDisablesAutoRenew(t *testing.T) {
	enabled := true
	period := AccountAutoRenewPeriodQuarter
	expiresAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	account := &Account{
		ExpiresAt:          &expiresAt,
		AutoRenewEnabled:   true,
		AutoRenewPeriod:    AccountAutoRenewPeriodYear,
		AutoPauseOnExpired: true,
	}
	account.ExpiresAt = nil

	err := applyAccountAutoRenewConfig(account, &enabled, &period, true)

	require.NoError(t, err)
	require.False(t, account.AutoRenewEnabled)
	require.Equal(t, AccountAutoRenewPeriodQuarter, account.AutoRenewPeriod)
}

func TestNormalizeAccountAutoRenewPeriod(t *testing.T) {
	period, err := NormalizeAccountAutoRenewPeriod(" QUARTER ")
	require.NoError(t, err)
	require.Equal(t, AccountAutoRenewPeriodQuarter, period)

	_, err = NormalizeAccountAutoRenewPeriod("week")
	require.Error(t, err)
}
