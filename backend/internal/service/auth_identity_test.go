//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type authIdentityUserRepoStub struct {
	usersByID    map[int64]*User
	usersByEmail map[string]*User
	nextID       int64
}

type authIdentitySettingRepoStub struct {
	values map[string]string
}

func (s *authIdentitySettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	if value, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *authIdentitySettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *authIdentitySettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *authIdentitySettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *authIdentitySettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *authIdentitySettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *authIdentitySettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func (s *authIdentityUserRepoStub) Create(ctx context.Context, user *User) error {
	if s.nextID > 0 && user.ID == 0 {
		user.ID = s.nextID
		s.nextID++
	}
	clone := *user
	if s.usersByID == nil {
		s.usersByID = map[int64]*User{}
	}
	s.usersByID[user.ID] = &clone
	if s.usersByEmail == nil {
		s.usersByEmail = map[string]*User{}
	}
	s.usersByEmail[user.Email] = &clone
	return nil
}

func (s *authIdentityUserRepoStub) GetByID(ctx context.Context, id int64) (*User, error) {
	user, ok := s.usersByID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (s *authIdentityUserRepoStub) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, ok := s.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (s *authIdentityUserRepoStub) GetFirstAdmin(ctx context.Context) (*User, error) {
	return nil, ErrUserNotFound
}

func (s *authIdentityUserRepoStub) Update(ctx context.Context, user *User) error {
	if user == nil {
		return ErrUserNotFound
	}
	clone := *user
	if s.usersByID == nil {
		s.usersByID = map[int64]*User{}
	}
	s.usersByID[user.ID] = &clone
	if s.usersByEmail == nil {
		s.usersByEmail = map[string]*User{}
	}
	s.usersByEmail[user.Email] = &clone
	return nil
}

func (s *authIdentityUserRepoStub) Delete(ctx context.Context, id int64) error {
	return nil
}

func (s *authIdentityUserRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *authIdentityUserRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *authIdentityUserRepoStub) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	return nil
}

func (s *authIdentityUserRepoStub) DeductBalance(ctx context.Context, id int64, amount float64) error {
	return nil
}

func (s *authIdentityUserRepoStub) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	return nil
}

func (s *authIdentityUserRepoStub) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, ok := s.usersByEmail[email]
	return ok, nil
}

func (s *authIdentityUserRepoStub) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	return 0, nil
}

func (s *authIdentityUserRepoStub) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	return nil
}

func (s *authIdentityUserRepoStub) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	return nil
}

func (s *authIdentityUserRepoStub) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	return nil
}

func (s *authIdentityUserRepoStub) EnableTotp(ctx context.Context, userID int64) error {
	return nil
}

func (s *authIdentityUserRepoStub) DisableTotp(ctx context.Context, userID int64) error {
	return nil
}

type authIdentityRefreshTokenCacheStub struct{}

func (authIdentityRefreshTokenCacheStub) StoreRefreshToken(ctx context.Context, tokenHash string, data *RefreshTokenData, ttl time.Duration) error {
	return nil
}

func (authIdentityRefreshTokenCacheStub) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenData, error) {
	return nil, ErrRefreshTokenNotFound
}

func (authIdentityRefreshTokenCacheStub) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}

func (authIdentityRefreshTokenCacheStub) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	return nil
}

func (authIdentityRefreshTokenCacheStub) DeleteTokenFamily(ctx context.Context, familyID string) error {
	return nil
}

func (authIdentityRefreshTokenCacheStub) AddToUserTokenSet(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
	return nil
}

func (authIdentityRefreshTokenCacheStub) AddToFamilyTokenSet(ctx context.Context, familyID string, tokenHash string, ttl time.Duration) error {
	return nil
}

func (authIdentityRefreshTokenCacheStub) GetUserTokenHashes(ctx context.Context, userID int64) ([]string, error) {
	return nil, nil
}

func (authIdentityRefreshTokenCacheStub) GetFamilyTokenHashes(ctx context.Context, familyID string) ([]string, error) {
	return nil, nil
}

func (authIdentityRefreshTokenCacheStub) IsTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) {
	return false, nil
}

type authIdentityRepoStub struct {
	items []*AuthIdentity
}

func newAuthIdentityRepoStub() *authIdentityRepoStub {
	return &authIdentityRepoStub{items: make([]*AuthIdentity, 0)}
}

func (r *authIdentityRepoStub) Create(ctx context.Context, identity *AuthIdentity) error {
	if identity == nil {
		return ErrAuthIdentityNotFound
	}
	if identity.ID == 0 {
		identity.ID = int64(len(r.items) + 1)
	}
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = time.Now().UTC()
	}
	if identity.UpdatedAt.IsZero() {
		identity.UpdatedAt = identity.CreatedAt
	}
	clone := *identity
	r.items = append(r.items, &clone)
	return nil
}

func (r *authIdentityRepoStub) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*AuthIdentity, error) {
	for _, item := range r.items {
		if item.Provider == provider && item.ProviderUserID == providerUserID {
			clone := *item
			return &clone, nil
		}
	}
	return nil, ErrAuthIdentityNotFound
}

func (r *authIdentityRepoStub) GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*AuthIdentity, error) {
	for _, item := range r.items {
		if item.UserID == userID && item.Provider == provider {
			clone := *item
			return &clone, nil
		}
	}
	return nil, ErrAuthIdentityNotFound
}

func (r *authIdentityRepoStub) ListByUserID(ctx context.Context, userID int64) ([]*AuthIdentity, error) {
	out := make([]*AuthIdentity, 0)
	for _, item := range r.items {
		if item.UserID != userID {
			continue
		}
		clone := *item
		out = append(out, &clone)
	}
	return out, nil
}

func (r *authIdentityRepoStub) DeleteByUserIDAndProvider(ctx context.Context, userID int64, provider string) error {
	next := make([]*AuthIdentity, 0, len(r.items))
	deleted := false
	for _, item := range r.items {
		if item.UserID == userID && item.Provider == provider {
			deleted = true
			continue
		}
		next = append(next, item)
	}
	if !deleted {
		return ErrAuthIdentityNotFound
	}
	r.items = next
	return nil
}

type authIdentityRedeemRepoStub struct{}

func (authIdentityRedeemRepoStub) Create(ctx context.Context, code *RedeemCode) error { return nil }
func (authIdentityRedeemRepoStub) CreateBatch(ctx context.Context, codes []RedeemCode) error {
	return nil
}
func (authIdentityRedeemRepoStub) GetByID(ctx context.Context, id int64) (*RedeemCode, error) {
	return nil, ErrRedeemCodeNotFound
}
func (authIdentityRedeemRepoStub) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
	return nil, ErrRedeemCodeNotFound
}
func (authIdentityRedeemRepoStub) Update(ctx context.Context, code *RedeemCode) error { return nil }
func (authIdentityRedeemRepoStub) Delete(ctx context.Context, id int64) error         { return nil }
func (authIdentityRedeemRepoStub) Use(ctx context.Context, id, userID int64) error    { return nil }
func (authIdentityRedeemRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (authIdentityRedeemRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (authIdentityRedeemRepoStub) ListByUser(ctx context.Context, userID int64, limit int) ([]RedeemCode, error) {
	return nil, nil
}
func (authIdentityRedeemRepoStub) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (authIdentityRedeemRepoStub) SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error) {
	return 0, nil
}

type authIdentityAffiliateRepoStub struct{}

func (authIdentityAffiliateRepoStub) GetUserAffiliate(ctx context.Context, userID int64) (*UserAffiliate, error) {
	return nil, nil
}
func (authIdentityAffiliateRepoStub) EnsureAffiliateRow(ctx context.Context, userID int64, affCode string) (bool, error) {
	return true, nil
}
func (authIdentityAffiliateRepoStub) BindInviterByCode(ctx context.Context, inviteeUserID int64, affCode string) (int64, bool, error) {
	return 0, false, nil
}
func (authIdentityAffiliateRepoStub) AccrueTopupRebate(ctx context.Context, redeemCodeID int64, inviteeUserID int64, creditedAmount float64, policy AffiliateRebatePolicy) (float64, error) {
	return 0, nil
}
func (authIdentityAffiliateRepoStub) ThawFrozenIfNeeded(ctx context.Context, inviterUserID int64) (float64, error) {
	return 0, nil
}
func (authIdentityAffiliateRepoStub) TransferToBalance(ctx context.Context, userID int64) (*AffiliateTransferResult, error) {
	return nil, nil
}
func (authIdentityAffiliateRepoStub) ListAffiliateUsers(ctx context.Context, params pagination.PaginationParams, filters AffiliateAdminUserListFilters) ([]AffiliateAdminUser, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (authIdentityAffiliateRepoStub) LookupAffiliateUsers(ctx context.Context, q string, limit int) ([]AffiliateAdminUser, error) {
	return nil, nil
}
func (authIdentityAffiliateRepoStub) UpdateAffiliateUserCustom(ctx context.Context, userID int64, update AffiliateAdminUserCustomUpdate, newAffCodeForClear string) (*UserAffiliate, error) {
	return nil, nil
}
func (authIdentityAffiliateRepoStub) ResetAffiliateUserCustom(ctx context.Context, userID int64, newAffCode string) (*UserAffiliate, error) {
	return nil, nil
}
func (authIdentityAffiliateRepoStub) BatchUpdateAffiliateUserCustomRates(ctx context.Context, userIDs []int64, customRatePercent float64) (int, error) {
	return 0, nil
}

func newAuthIdentityTestService(t *testing.T, userRepo *authIdentityUserRepoStub, settingValues map[string]string) (*AuthIdentityService, *authIdentityRepoStub, *AuthService) {
	t.Helper()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:     "auth-identity-test-secret",
			ExpireHour: 1,
		},
		Default: config.DefaultConfig{
			UserBalance:     1,
			UserConcurrency: 2,
		},
	}
	settingService := NewSettingService(&authIdentitySettingRepoStub{values: settingValues}, cfg)
	affiliateService := NewAffiliateService(settingService, authIdentityAffiliateRepoStub{})
	authService := NewAuthService(
		nil,
		userRepo,
		authIdentityRedeemRepoStub{},
		authIdentityRefreshTokenCacheStub{},
		cfg,
		settingService,
		nil,
		nil,
		nil,
		nil,
		affiliateService,
		nil,
	)
	repo := newAuthIdentityRepoStub()
	service := NewAuthIdentityService(repo, userRepo, authService)
	authService.SetAuthIdentityRepository(repo)
	return service, repo, authService
}

func TestAuthIdentityService_ResolveLoginOrBind_RejectsVerifiedExistingEmailTakeover(t *testing.T) {
	userRepo := &authIdentityUserRepoStub{
		usersByID: map[int64]*User{
			7: {ID: 7, Email: "alice@example.com", Username: "alice", Role: RoleUser, Status: StatusActive},
		},
		usersByEmail: map[string]*User{
			"alice@example.com": {ID: 7, Email: "alice@example.com", Username: "alice", Role: RoleUser, Status: StatusActive},
		},
		nextID: 10,
	}
	svc, repo, _ := newAuthIdentityTestService(t, userRepo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	})

	result, err := svc.ResolveLoginOrBind(context.Background(), "login", 0, &AuthIdentity{
		Provider:       AuthProviderGitHub,
		ProviderUserID: "gh-1",
		Email:          "alice@example.com",
		EmailVerified:  true,
		DisplayName:    "Alice GH",
	}, "", "")

	require.ErrorIs(t, err, ErrAuthIdentityEmailConflict)
	require.Nil(t, result)
	require.Empty(t, repo.items)
}

func TestAuthIdentityService_ResolveLoginOrBind_RejectsUnsafeUnverifiedEmailTakeover(t *testing.T) {
	userRepo := &authIdentityUserRepoStub{
		usersByID: map[int64]*User{
			7: {ID: 7, Email: "alice@example.com", Username: "alice", Role: RoleUser, Status: StatusActive},
		},
		usersByEmail: map[string]*User{
			"alice@example.com": {ID: 7, Email: "alice@example.com", Username: "alice", Role: RoleUser, Status: StatusActive},
		},
	}
	svc, _, _ := newAuthIdentityTestService(t, userRepo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	})

	_, err := svc.ResolveLoginOrBind(context.Background(), "login", 0, &AuthIdentity{
		Provider:       AuthProviderGoogle,
		ProviderUserID: "google-1",
		Email:          "alice@example.com",
		EmailVerified:  false,
		DisplayName:    "Alice Google",
	}, "", "")

	require.ErrorIs(t, err, ErrAuthIdentityUnsafeEmail)
}

func TestAuthIdentityService_ResolveLoginOrBind_ReturnsPendingWhenInvitationRequired(t *testing.T) {
	userRepo := &authIdentityUserRepoStub{
		usersByID:    map[int64]*User{},
		usersByEmail: map[string]*User{},
		nextID:       21,
	}
	svc, repo, authService := newAuthIdentityTestService(t, userRepo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
	})

	result, err := svc.ResolveLoginOrBind(context.Background(), "login", 0, &AuthIdentity{
		Provider:       AuthProviderGitHub,
		ProviderUserID: "gh-pending",
		Email:          "pending@example.com",
		EmailVerified:  true,
		DisplayName:    "Pending User",
		AvatarURL:      "https://avatars.example.com/pending.png",
	}, "", "")

	require.NoError(t, err)
	require.Equal(t, "pending", result.Outcome)
	require.NotEmpty(t, result.Pending)
	require.Empty(t, repo.items)

	claims, verifyErr := authService.VerifyPendingOAuthClaims(result.Pending)
	require.NoError(t, verifyErr)
	require.Equal(t, AuthProviderGitHub, claims.Provider)
	require.Equal(t, "gh-pending", claims.ProviderUserID)
	require.Equal(t, "pending@example.com", claims.Email)
}

func TestAuthIdentityService_ResolveLoginOrBind_BindsNewVerifiedIdentityAfterRegistration(t *testing.T) {
	userRepo := &authIdentityUserRepoStub{
		usersByID:    map[int64]*User{},
		usersByEmail: map[string]*User{},
		nextID:       30,
	}
	svc, repo, _ := newAuthIdentityTestService(t, userRepo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	})

	result, err := svc.ResolveLoginOrBind(context.Background(), "login", 0, &AuthIdentity{
		Provider:       AuthProviderGitHub,
		ProviderUserID: "gh-new",
		Email:          "new-user@example.com",
		EmailVerified:  true,
		DisplayName:    "New User",
	}, "", "")

	require.NoError(t, err)
	require.Equal(t, "login", result.Outcome)
	require.NotNil(t, result.TokenPair)
	require.Len(t, repo.items, 1)
	require.Equal(t, "new-user@example.com", repo.items[0].Email)
	require.Equal(t, int64(30), repo.items[0].UserID)
}

func TestAuthIdentityService_ResolveLoginOrBind_RespectsRegistrationEmailWildcard(t *testing.T) {
	userRepo := &authIdentityUserRepoStub{
		usersByID:    map[int64]*User{},
		usersByEmail: map[string]*User{},
		nextID:       40,
	}
	svc, repo, _ := newAuthIdentityTestService(t, userRepo, map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyRegistrationEmailSuffixWhitelist: `["@*.example.com"]`,
	})

	result, err := svc.ResolveLoginOrBind(context.Background(), "login", 0, &AuthIdentity{
		Provider:       AuthProviderGoogle,
		ProviderUserID: "google-subdomain",
		Email:          "new-user@team.example.com",
		EmailVerified:  true,
		DisplayName:    "New User",
	}, "", "")

	require.NoError(t, err)
	require.Equal(t, "login", result.Outcome)
	require.Len(t, repo.items, 1)

	blockedResult, blockedErr := svc.ResolveLoginOrBind(context.Background(), "login", 0, &AuthIdentity{
		Provider:       AuthProviderGoogle,
		ProviderUserID: "google-root",
		Email:          "root@example.com",
		EmailVerified:  true,
		DisplayName:    "Root User",
	}, "", "")

	require.Error(t, blockedErr)
	require.Nil(t, blockedResult)
	require.Equal(t, "EMAIL_SUFFIX_NOT_ALLOWED", infraerrors.Reason(blockedErr))
}
