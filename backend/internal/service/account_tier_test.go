package service

import (
	"context"
	"testing"
)

type accountTierAdminRepoStub struct {
	AccountRepository
	account *Account
	created *Account
	updated *Account
	nextID  int64
}

func (s *accountTierAdminRepoStub) Create(_ context.Context, account *Account) error {
	s.nextID++
	if account.ID == 0 {
		account.ID = s.nextID
	}
	s.account = account
	s.created = account
	return nil
}

func (s *accountTierAdminRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.account != nil && s.account.ID == id {
		return s.account, nil
	}
	return nil, ErrAccountNotFound
}

func (s *accountTierAdminRepoStub) Update(_ context.Context, account *Account) error {
	s.account = account
	s.updated = account
	return nil
}

func TestDefaultAccountTierConcurrency(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		tier     string
		want     int
	}{
		{name: "openai pro 20x", platform: PlatformOpenAI, tier: OpenAIAccountTierPro20x, want: 10},
		{name: "openai pro 5x", platform: PlatformOpenAI, tier: OpenAIAccountTierPro5x, want: 5},
		{name: "openai plus", platform: PlatformOpenAI, tier: OpenAIAccountTierPlus, want: 2},
		{name: "openai team", platform: PlatformOpenAI, tier: OpenAIAccountTierTeam, want: 2},
		{name: "openai free", platform: PlatformOpenAI, tier: OpenAIAccountTierFree, want: 1},
		{name: "claude max 20x", platform: PlatformAnthropic, tier: ClaudeAccountTierMax20x, want: 10},
		{name: "claude max 5x", platform: PlatformAnthropic, tier: ClaudeAccountTierMax5x, want: 5},
		{name: "claude pro", platform: PlatformAnthropic, tier: ClaudeAccountTierPro, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultAccountTierConcurrency(tt.platform, tt.tier); got != tt.want {
				t.Fatalf("DefaultAccountTierConcurrency(%q, %q) = %d, want %d", tt.platform, tt.tier, got, tt.want)
			}
		})
	}
}

func TestApplyAccountTierDefaults_OpenAIFreeSetsImageCompatDefault(t *testing.T) {
	extra, concurrency := ApplyAccountTierDefaults(PlatformOpenAI, AccountTypeOAuth, map[string]any{
		AccountExtraKeyAccountTier: "Free",
	}, 0)

	if concurrency != 1 {
		t.Fatalf("concurrency = %d, want 1", concurrency)
	}
	if got := extra[AccountExtraKeyAccountTier]; got != OpenAIAccountTierFree {
		t.Fatalf("account_tier = %v, want %q", got, OpenAIAccountTierFree)
	}
	if got := extra[openAIImageProtocolModeExtraKey]; got != OpenAIImageProtocolModeNative {
		t.Fatalf("image_protocol_mode = %v, want %q", got, OpenAIImageProtocolModeNative)
	}
	if got := extra[openAIImageCompatAllowedExtraKey]; got != false {
		t.Fatalf("image_compat_allowed = %v, want false", got)
	}
}

func TestApplyAccountTierDefaults_PreservesPositiveConcurrencyAndManualImageCompat(t *testing.T) {
	extra, concurrency := ApplyAccountTierDefaults(PlatformOpenAI, AccountTypeOAuth, map[string]any{
		AccountExtraKeyAccountTier:       OpenAIAccountTierFree,
		openAIImageCompatAllowedExtraKey: true,
	}, 8)

	if concurrency != 8 {
		t.Fatalf("concurrency = %d, want 8", concurrency)
	}
	if got := extra[openAIImageCompatAllowedExtraKey]; got != true {
		t.Fatalf("image_compat_allowed = %v, want true", got)
	}
}

func TestAdminServiceCreateAccountAppliesTierDefaults(t *testing.T) {
	repo := &accountTierAdminRepoStub{}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "openai-free",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "refresh-token",
		},
		Extra: map[string]any{
			AccountExtraKeyAccountTier: OpenAIAccountTierFree,
		},
		Concurrency:          0,
		SkipDefaultGroupBind: true,
	})

	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	if account == nil || repo.created == nil {
		t.Fatal("expected account to be created")
	}
	if account.Concurrency != 1 {
		t.Fatalf("concurrency = %d, want 1", account.Concurrency)
	}
	if got := account.Extra[AccountExtraKeyAccountTier]; got != OpenAIAccountTierFree {
		t.Fatalf("account_tier = %v, want %q", got, OpenAIAccountTierFree)
	}
	if got := account.Extra[openAIImageProtocolModeExtraKey]; got != OpenAIImageProtocolModeNative {
		t.Fatalf("image_protocol_mode = %v, want %q", got, OpenAIImageProtocolModeNative)
	}
	if got := account.Extra[openAIImageCompatAllowedExtraKey]; got != false {
		t.Fatalf("image_compat_allowed = %v, want false", got)
	}
}

func TestAdminServiceUpdateAccountAppliesTierCapacityWhenRequestedConcurrencyIsZero(t *testing.T) {
	repo := &accountTierAdminRepoStub{
		account: &Account{
			ID:          42,
			Name:        "claude",
			Platform:    PlatformAnthropic,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Concurrency: 1,
			Extra: map[string]any{
				AccountExtraKeyAccountTier: ClaudeAccountTierMax20x,
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}
	zero := 0

	account, err := svc.UpdateAccount(context.Background(), 42, &UpdateAccountInput{
		Concurrency: &zero,
	})

	if err != nil {
		t.Fatalf("UpdateAccount returned error: %v", err)
	}
	if account == nil || repo.updated == nil {
		t.Fatal("expected account to be updated")
	}
	if account.Concurrency != 10 {
		t.Fatalf("concurrency = %d, want 10", account.Concurrency)
	}
	if got := account.Extra[AccountExtraKeyAccountTier]; got != ClaudeAccountTierMax20x {
		t.Fatalf("account_tier = %v, want %q", got, ClaudeAccountTierMax20x)
	}
}
