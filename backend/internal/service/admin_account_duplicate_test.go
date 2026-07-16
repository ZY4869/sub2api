//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type duplicateAccountRepoStub struct {
	nextID  int64
	records map[int64]*Account
}

func newDuplicateAccountRepoStub() *duplicateAccountRepoStub {
	return &duplicateAccountRepoStub{nextID: 1, records: map[int64]*Account{}}
}

func (s *duplicateAccountRepoStub) Create(_ context.Context, account *Account) error {
	if account.ID == 0 {
		account.ID = s.nextID
		s.nextID++
	} else if account.ID >= s.nextID {
		s.nextID = account.ID + 1
	}
	s.records[account.ID] = cloneDuplicateAccount(account)
	return nil
}

func (s *duplicateAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	account, ok := s.records[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return cloneDuplicateAccount(account), nil
}

func (s *duplicateAccountRepoStub) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	out := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := s.records[id]; ok {
			out = append(out, cloneDuplicateAccount(account))
		}
	}
	return out, nil
}

func (s *duplicateAccountRepoStub) ExistsByID(_ context.Context, id int64) (bool, error) {
	_, ok := s.records[id]
	return ok, nil
}

func (s *duplicateAccountRepoStub) FindByExtraField(_ context.Context, key string, value any) ([]Account, error) {
	matches := make([]Account, 0)
	for _, account := range s.records {
		if account == nil || account.Extra == nil {
			continue
		}
		if account.Extra[key] == value {
			cloned := cloneDuplicateAccount(account)
			matches = append(matches, *cloned)
		}
	}
	return matches, nil
}

func (s *duplicateAccountRepoStub) Update(_ context.Context, account *Account) error {
	if _, ok := s.records[account.ID]; !ok {
		return ErrAccountNotFound
	}
	s.records[account.ID] = cloneDuplicateAccount(account)
	return nil
}

func (s *duplicateAccountRepoStub) Delete(_ context.Context, id int64) error {
	delete(s.records, id)
	return nil
}

func (s *duplicateAccountRepoStub) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	account, ok := s.records[accountID]
	if !ok {
		return ErrAccountNotFound
	}
	account.GroupIDs = append([]int64(nil), groupIDs...)
	return nil
}

func (s *duplicateAccountRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *duplicateAccountRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string, string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *duplicateAccountRepoStub) GetStatusSummary(context.Context, AccountStatusSummaryFilters) (*AccountStatusSummary, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListActive(context.Context) ([]Account, error) { return nil, nil }
func (s *duplicateAccountRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) UpdateLastUsed(context.Context, int64) error { return nil }
func (s *duplicateAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (s *duplicateAccountRepoStub) SetError(context.Context, int64, string) error { return nil }
func (s *duplicateAccountRepoStub) ClearError(context.Context, int64) error       { return nil }
func (s *duplicateAccountRepoStub) SetSchedulable(context.Context, int64, bool) error {
	return nil
}
func (s *duplicateAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (s *duplicateAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}
func (s *duplicateAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (s *duplicateAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (s *duplicateAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}
func (s *duplicateAccountRepoStub) ClearRateLimit(context.Context, int64) error { return nil }
func (s *duplicateAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (s *duplicateAccountRepoStub) ClearModelRateLimits(context.Context, int64) error {
	return nil
}
func (s *duplicateAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (s *duplicateAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	account, ok := s.records[id]
	if !ok {
		return ErrAccountNotFound
	}
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	for key, value := range updates {
		account.Extra[key] = value
	}
	return nil
}
func (s *duplicateAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (s *duplicateAccountRepoStub) MarkBlacklisted(context.Context, int64, string, string, time.Time, time.Time) error {
	return nil
}
func (s *duplicateAccountRepoStub) RestoreBlacklisted(context.Context, int64) error { return nil }
func (s *duplicateAccountRepoStub) ListBlacklistedIDs(context.Context) ([]int64, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) ListBlacklistedForPurge(context.Context, time.Time, int) ([]Account, error) {
	return nil, nil
}
func (s *duplicateAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (s *duplicateAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func cloneDuplicateAccount(account *Account) *Account {
	if account == nil {
		return nil
	}
	cloned := *account
	cloned.Notes = cloneStringPtr(account.Notes)
	cloned.Credentials = cloneJSONMap(account.Credentials)
	cloned.Extra = cloneJSONMap(account.Extra)
	cloned.ProxyID = cloneInt64Ptr(account.ProxyID)
	cloned.RateMultiplier = cloneAccountFloat64Ptr(account.RateMultiplier)
	cloned.LoadFactor = cloneIntPtr(account.LoadFactor)
	cloned.GroupIDs = append([]int64(nil), account.GroupIDs...)
	cloned.RateLimitedAt = cloneTimePtr(account.RateLimitedAt)
	cloned.ExpiresAt = cloneTimePtr(account.ExpiresAt)
	return &cloned
}

func TestDuplicateAccountCopiesConfigAndClearsRuntimeState(t *testing.T) {
	ctx := context.Background()
	repo := newDuplicateAccountRepoStub()
	svc := &adminServiceImpl{accountRepo: repo}
	notes := "keep"
	proxyID := int64(12)
	rateMultiplier := 1.25
	loadFactor := 8
	resetAt := time.Now().Add(time.Hour)
	source := &Account{
		Name:           "primary",
		Notes:          &notes,
		Platform:       PlatformAnthropic,
		Type:           AccountTypeAPIKey,
		Credentials:    map[string]any{"api_key": "secret", "nested": map[string]any{"token": "source"}},
		Extra:          map[string]any{"quota_limit": 1000, "quota_used": 50, "model_rate_limits": map[string]any{"x": "y"}, "config": map[string]any{"region": "us"}},
		ProxyID:        &proxyID,
		Concurrency:    6,
		Priority:       40,
		RateMultiplier: &rateMultiplier,
		LoadFactor:     &loadFactor,
		Status:         StatusError,
		Schedulable:    true,
		ErrorMessage:   "upstream unavailable",
		GroupIDs:       []int64{7, 3},
		RateLimitedAt:  &resetAt,
	}
	require.NoError(t, repo.Create(ctx, source))

	duplicate, err := svc.DuplicateAccount(ctx, source.ID, "admin:1", "copy-key")

	require.NoError(t, err)
	require.NotEqual(t, source.ID, duplicate.ID)
	require.Equal(t, "primary (Copy)", duplicate.Name)
	require.Equal(t, source.Credentials, duplicate.Credentials)
	require.Equal(t, []int64{7, 3}, duplicate.GroupIDs)
	require.Equal(t, float64(1000), duplicate.Extra["quota_limit"])
	require.Equal(t, map[string]any{"region": "us"}, duplicate.Extra["config"])
	require.NotContains(t, duplicate.Extra, "quota_used")
	require.NotContains(t, duplicate.Extra, "model_rate_limits")
	require.Equal(t, StatusActive, duplicate.Status)
	require.False(t, duplicate.Schedulable)
	require.Empty(t, duplicate.ErrorMessage)
	require.Nil(t, duplicate.RateLimitedAt)
	require.NotEmpty(t, duplicate.Extra[duplicateAccountOperationIDExtraKey])

	duplicate.Credentials["nested"].(map[string]any)["token"] = "changed"
	duplicate.Extra["config"].(map[string]any)["region"] = "eu"
	storedSource, err := repo.GetByID(ctx, source.ID)
	require.NoError(t, err)
	require.Equal(t, "source", storedSource.Credentials["nested"].(map[string]any)["token"])
	require.Equal(t, "us", storedSource.Extra["config"].(map[string]any)["region"])
}

func TestDuplicateAccountRejectsRotatingCredentials(t *testing.T) {
	ctx := context.Background()
	repo := newDuplicateAccountRepoStub()
	svc := &adminServiceImpl{accountRepo: repo}
	source := &Account{
		Name:        "oauth",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"refresh_token": "secret"},
	}
	require.NoError(t, repo.Create(ctx, source))

	_, err := svc.DuplicateAccount(ctx, source.ID, "admin:1", "copy-key")

	require.Error(t, err)
	require.Contains(t, err.Error(), "ACCOUNT_DUPLICATE_CREDENTIAL_TYPE_UNSUPPORTED")
}

func TestDuplicateAccountReusesOperationKey(t *testing.T) {
	ctx := context.Background()
	repo := newDuplicateAccountRepoStub()
	svc := &adminServiceImpl{accountRepo: repo}
	source := &Account{Name: "source", Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	require.NoError(t, repo.Create(ctx, source))

	first, err := svc.DuplicateAccount(ctx, source.ID, "admin:7", "stable")
	require.NoError(t, err)
	second, err := svc.DuplicateAccount(ctx, source.ID, "admin:7", "stable")
	require.NoError(t, err)
	other, err := svc.DuplicateAccount(ctx, source.ID, "admin:8", "stable")
	require.NoError(t, err)

	require.Equal(t, first.ID, second.ID)
	require.NotEqual(t, first.ID, other.ID)
}
