package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type proxyExpiryRepoStub struct {
	listNow time.Time
	proxies []Proxy
	err     error
}

func (r *proxyExpiryRepoStub) ListExpiredProxies(_ context.Context, now time.Time) ([]Proxy, error) {
	r.listNow = now
	if r.err != nil {
		return nil, r.err
	}
	return append([]Proxy(nil), r.proxies...), nil
}

type proxyExpiryReaderStub struct {
	proxies map[int64]*Proxy
	err     error
}

func (r *proxyExpiryReaderStub) Create(context.Context, *Proxy) error { return nil }
func (r *proxyExpiryReaderStub) GetByID(_ context.Context, id int64) (*Proxy, error) {
	if r.err != nil {
		return nil, r.err
	}
	if proxy, ok := r.proxies[id]; ok {
		cp := *proxy
		return &cp, nil
	}
	return nil, ErrProxyNotFound
}
func (r *proxyExpiryReaderStub) ListByIDs(context.Context, []int64) ([]Proxy, error) {
	return nil, nil
}
func (r *proxyExpiryReaderStub) Update(context.Context, *Proxy) error { return nil }
func (r *proxyExpiryReaderStub) Delete(context.Context, int64) error  { return nil }
func (r *proxyExpiryReaderStub) List(context.Context, pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *proxyExpiryReaderStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]Proxy, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *proxyExpiryReaderStub) ListWithFiltersAndAccountCount(context.Context, pagination.PaginationParams, string, string, string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *proxyExpiryReaderStub) ListActive(context.Context) ([]Proxy, error) {
	return nil, nil
}
func (r *proxyExpiryReaderStub) ListActiveWithAccountCount(context.Context) ([]ProxyWithAccountCount, error) {
	return nil, nil
}
func (r *proxyExpiryReaderStub) ExistsByHostPortAuth(context.Context, string, int, string, string) (bool, error) {
	return false, nil
}
func (r *proxyExpiryReaderStub) CountAccountsByProxyID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (r *proxyExpiryReaderStub) ListAccountSummariesByProxyID(context.Context, int64) ([]ProxyAccountSummary, error) {
	return nil, nil
}

type proxyExpirySwitchCall struct {
	expired    Proxy
	fallback   Proxy
	switchedAt time.Time
}

type proxyExpiryAccountRepoStub struct {
	calls []proxyExpirySwitchCall
	err   error
}

func (r *proxyExpiryAccountRepoStub) SwitchExpiredProxyAccounts(_ context.Context, expired Proxy, fallback Proxy, switchedAt time.Time) ([]int64, error) {
	r.calls = append(r.calls, proxyExpirySwitchCall{expired: expired, fallback: fallback, switchedAt: switchedAt})
	if r.err != nil {
		return nil, r.err
	}
	return []int64{101, 102}, nil
}

func (r *proxyExpiryAccountRepoStub) RestoreAccountOriginalProxy(context.Context, int64) (*AccountProxyRestoreResult, error) {
	return nil, nil
}

func TestProxyExpiryServiceRunOnceSwitchesAccountsToActiveFallback(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-time.Hour)
	fallbackID := int64(20)
	expired := Proxy{ID: 10, Name: "expired", Status: StatusActive, ExpiresAt: &expiresAt, FallbackProxyID: &fallbackID}
	fallback := Proxy{ID: fallbackID, Name: "fallback", Status: StatusActive}

	proxyRepo := &proxyExpiryRepoStub{proxies: []Proxy{expired}}
	accountRepo := &proxyExpiryAccountRepoStub{}
	reader := &proxyExpiryReaderStub{proxies: map[int64]*Proxy{fallbackID: &fallback}}
	svc := NewProxyExpiryService(proxyRepo, accountRepo, reader, time.Minute)
	svc.SetNow(func() time.Time { return now.In(time.FixedZone("test", 8*60*60)) })

	svc.runOnce(context.Background())

	require.Equal(t, now, proxyRepo.listNow)
	require.Len(t, accountRepo.calls, 1)
	require.Equal(t, expired.ID, accountRepo.calls[0].expired.ID)
	require.Equal(t, fallbackID, accountRepo.calls[0].fallback.ID)
	require.Equal(t, now, accountRepo.calls[0].switchedAt)
	require.Equal(t, time.UTC, accountRepo.calls[0].switchedAt.Location())
}

func TestProxyExpiryServiceRunOnceSkipsUnavailableFallbacks(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-time.Hour)
	fallbackID := int64(20)

	tests := []struct {
		name    string
		expired Proxy
		reader  *proxyExpiryReaderStub
	}{
		{
			name:    "without fallback",
			expired: Proxy{ID: 10, Name: "expired", Status: StatusActive, ExpiresAt: &expiresAt},
			reader:  &proxyExpiryReaderStub{},
		},
		{
			name:    "fallback missing",
			expired: Proxy{ID: 11, Name: "expired", Status: StatusActive, ExpiresAt: &expiresAt, FallbackProxyID: &fallbackID},
			reader:  &proxyExpiryReaderStub{proxies: map[int64]*Proxy{}},
		},
		{
			name:    "fallback inactive",
			expired: Proxy{ID: 12, Name: "expired", Status: StatusActive, ExpiresAt: &expiresAt, FallbackProxyID: &fallbackID},
			reader: &proxyExpiryReaderStub{proxies: map[int64]*Proxy{
				fallbackID: &Proxy{ID: fallbackID, Name: "inactive", Status: StatusDisabled},
			}},
		},
		{
			name:    "fallback read error",
			expired: Proxy{ID: 13, Name: "expired", Status: StatusActive, ExpiresAt: &expiresAt, FallbackProxyID: &fallbackID},
			reader:  &proxyExpiryReaderStub{err: errors.New("read failed")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountRepo := &proxyExpiryAccountRepoStub{}
			svc := NewProxyExpiryService(
				&proxyExpiryRepoStub{proxies: []Proxy{tt.expired}},
				accountRepo,
				tt.reader,
				time.Minute,
			)
			svc.SetNow(func() time.Time { return now })

			svc.runOnce(context.Background())

			require.Empty(t, accountRepo.calls)
		})
	}
}
