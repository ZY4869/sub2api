package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type authBindingUserRepoStub struct {
	user *User
	err  error
}

func (s *authBindingUserRepoStub) Create(context.Context, *User) error { panic("unexpected Create") }
func (s *authBindingUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	return s.user, nil
}
func (s *authBindingUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail")
}
func (s *authBindingUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin")
}
func (s *authBindingUserRepoStub) Update(context.Context, *User) error { panic("unexpected Update") }
func (s *authBindingUserRepoStub) Delete(context.Context, int64) error { panic("unexpected Delete") }
func (s *authBindingUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List")
}
func (s *authBindingUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters")
}
func (s *authBindingUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance")
}
func (s *authBindingUserRepoStub) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance")
}
func (s *authBindingUserRepoStub) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency")
}
func (s *authBindingUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail")
}
func (s *authBindingUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups")
}
func (s *authBindingUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups")
}
func (s *authBindingUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups")
}
func (s *authBindingUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret")
}
func (s *authBindingUserRepoStub) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp")
}
func (s *authBindingUserRepoStub) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp")
}

type authBindingRefreshTokenCacheStub struct {
	tokenData     *RefreshTokenData
	stored        []*RefreshTokenData
	deletedHashes []string
	deletedFamily []string
}

func (s *authBindingRefreshTokenCacheStub) StoreRefreshToken(_ context.Context, tokenHash string, data *RefreshTokenData, _ time.Duration) error {
	clone := *data
	s.tokenData = &clone
	s.stored = append(s.stored, &clone)
	return nil
}
func (s *authBindingRefreshTokenCacheStub) GetRefreshToken(context.Context, string) (*RefreshTokenData, error) {
	if s.tokenData == nil {
		return nil, ErrRefreshTokenNotFound
	}
	clone := *s.tokenData
	return &clone, nil
}
func (s *authBindingRefreshTokenCacheStub) DeleteRefreshToken(_ context.Context, tokenHash string) error {
	s.deletedHashes = append(s.deletedHashes, tokenHash)
	s.tokenData = nil
	return nil
}
func (s *authBindingRefreshTokenCacheStub) DeleteUserRefreshTokens(context.Context, int64) error {
	return nil
}
func (s *authBindingRefreshTokenCacheStub) DeleteTokenFamily(_ context.Context, familyID string) error {
	s.deletedFamily = append(s.deletedFamily, familyID)
	s.tokenData = nil
	return nil
}
func (s *authBindingRefreshTokenCacheStub) AddToUserTokenSet(context.Context, int64, string, time.Duration) error {
	return nil
}
func (s *authBindingRefreshTokenCacheStub) AddToFamilyTokenSet(context.Context, string, string, time.Duration) error {
	return nil
}
func (s *authBindingRefreshTokenCacheStub) GetUserTokenHashes(context.Context, int64) ([]string, error) {
	return nil, nil
}
func (s *authBindingRefreshTokenCacheStub) GetFamilyTokenHashes(context.Context, string) ([]string, error) {
	return nil, nil
}
func (s *authBindingRefreshTokenCacheStub) IsTokenInFamily(context.Context, string, string) (bool, error) {
	return false, nil
}

func newAuthBindingService(cache *authBindingRefreshTokenCacheStub) *AuthService {
	user := &User{ID: 42, Email: "bound@example.com", Role: RoleUser, Status: StatusActive, TokenVersion: 7}
	return NewAuthService(nil, &authBindingUserRepoStub{user: user}, nil, cache, &config.Config{
		JWT: config.JWTConfig{
			Secret:                 "test-secret",
			ExpireHour:             1,
			RefreshTokenExpireDays: 7,
		},
	}, nil, nil, nil, nil, nil, nil, nil)
}

func TestAuthServiceGenerateTokenPairStoresSessionBindingHashes(t *testing.T) {
	cache := &authBindingRefreshTokenCacheStub{}
	svc := newAuthBindingService(cache)
	user := &User{ID: 42, Email: "bound@example.com", Role: RoleUser, Status: StatusActive, TokenVersion: 7}
	ctx := WithAuthSessionBinding(context.Background(), "203.0.113.9", "sub2api-test/1.0")

	pair, err := svc.GenerateTokenPair(ctx, user, "")

	require.NoError(t, err)
	require.NotEmpty(t, pair.RefreshToken)
	require.NotNil(t, cache.tokenData)
	require.Equal(t, hashAuthSessionBindingValue("203.0.113.9"), cache.tokenData.ClientIPHash)
	require.Equal(t, hashAuthSessionBindingValue("sub2api-test/1.0"), cache.tokenData.UserAgentHash)
	require.NotContains(t, cache.tokenData.ClientIPHash, "203.0.113.9")
	require.NotContains(t, cache.tokenData.UserAgentHash, "sub2api-test")
}

func TestAuthServiceRefreshTokenPairRejectsSessionBindingMismatch(t *testing.T) {
	cache := &authBindingRefreshTokenCacheStub{tokenData: &RefreshTokenData{
		UserID:        42,
		TokenVersion:  7,
		FamilyID:      "family-1",
		ClientIPHash:  hashAuthSessionBindingValue("203.0.113.9"),
		UserAgentHash: hashAuthSessionBindingValue("sub2api-test/1.0"),
		CreatedAt:     time.Now().Add(-time.Minute),
		ExpiresAt:     time.Now().Add(time.Hour),
	}}
	svc := newAuthBindingService(cache)
	ctx := WithAuthSessionBinding(context.Background(), "203.0.113.10", "sub2api-test/1.0")

	pair, err := svc.RefreshTokenPair(ctx, refreshTokenPrefix+"0123456789abcdef")

	require.ErrorIs(t, err, ErrRefreshSessionMismatch)
	require.Nil(t, pair)
	require.Equal(t, []string{"family-1"}, cache.deletedFamily)
}

func TestAuthServiceRefreshTokenPairAcceptsMatchingSessionBinding(t *testing.T) {
	cache := &authBindingRefreshTokenCacheStub{tokenData: &RefreshTokenData{
		UserID:        42,
		TokenVersion:  7,
		FamilyID:      "family-bound",
		ClientIPHash:  hashAuthSessionBindingValue("203.0.113.9"),
		UserAgentHash: hashAuthSessionBindingValue("sub2api-test/1.0"),
		CreatedAt:     time.Now().Add(-time.Minute),
		ExpiresAt:     time.Now().Add(time.Hour),
	}}
	svc := newAuthBindingService(cache)
	ctx := WithAuthSessionBinding(context.Background(), "203.0.113.9", "sub2api-test/1.0")

	pair, err := svc.RefreshTokenPair(ctx, refreshTokenPrefix+"fedcba9876543210")

	require.NoError(t, err)
	require.NotNil(t, pair)
	require.Empty(t, cache.deletedFamily)
	require.Len(t, cache.stored, 1)
	require.Equal(t, "family-bound", cache.stored[0].FamilyID)
	require.Equal(t, hashAuthSessionBindingValue("203.0.113.9"), cache.stored[0].ClientIPHash)
	require.Equal(t, hashAuthSessionBindingValue("sub2api-test/1.0"), cache.stored[0].UserAgentHash)
}

func TestAuthServiceRefreshTokenPairAllowsLegacyUnboundTokenAndRotatesWithBinding(t *testing.T) {
	cache := &authBindingRefreshTokenCacheStub{tokenData: &RefreshTokenData{
		UserID:       42,
		TokenVersion: 7,
		FamilyID:     "family-legacy",
		CreatedAt:    time.Now().Add(-time.Minute),
		ExpiresAt:    time.Now().Add(time.Hour),
	}}
	svc := newAuthBindingService(cache)
	ctx := WithAuthSessionBinding(context.Background(), "198.51.100.7", "sub2api-test/2.0")

	pair, err := svc.RefreshTokenPair(ctx, refreshTokenPrefix+"abcdef0123456789")

	require.NoError(t, err)
	require.NotNil(t, pair)
	require.NotEmpty(t, pair.RefreshToken)
	require.Len(t, cache.stored, 1)
	require.Equal(t, "family-legacy", cache.stored[0].FamilyID)
	require.Equal(t, hashAuthSessionBindingValue("198.51.100.7"), cache.stored[0].ClientIPHash)
	require.Equal(t, hashAuthSessionBindingValue("sub2api-test/2.0"), cache.stored[0].UserAgentHash)
}
