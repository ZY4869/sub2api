//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type socialOAuthUserRepoStub struct {
	usersByID    map[int64]*service.User
	usersByEmail map[string]*service.User
	nextID       int64
}

func (s *socialOAuthUserRepoStub) Create(ctx context.Context, user *service.User) error {
	if s.nextID > 0 && user.ID == 0 {
		user.ID = s.nextID
		s.nextID++
	}
	clone := *user
	if s.usersByID == nil {
		s.usersByID = map[int64]*service.User{}
	}
	if s.usersByEmail == nil {
		s.usersByEmail = map[string]*service.User{}
	}
	s.usersByID[clone.ID] = &clone
	s.usersByEmail[clone.Email] = &clone
	return nil
}

func (s *socialOAuthUserRepoStub) GetByID(ctx context.Context, id int64) (*service.User, error) {
	user, ok := s.usersByID[id]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (s *socialOAuthUserRepoStub) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	user, ok := s.usersByEmail[email]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (s *socialOAuthUserRepoStub) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	return nil, service.ErrUserNotFound
}

func (s *socialOAuthUserRepoStub) Update(ctx context.Context, user *service.User) error {
	if user == nil {
		return service.ErrUserNotFound
	}
	clone := *user
	if s.usersByID == nil {
		s.usersByID = map[int64]*service.User{}
	}
	if s.usersByEmail == nil {
		s.usersByEmail = map[string]*service.User{}
	}
	s.usersByID[clone.ID] = &clone
	s.usersByEmail[clone.Email] = &clone
	return nil
}

func (s *socialOAuthUserRepoStub) Delete(ctx context.Context, id int64) error { return nil }
func (s *socialOAuthUserRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *socialOAuthUserRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *socialOAuthUserRepoStub) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	return nil
}
func (s *socialOAuthUserRepoStub) DeductBalance(ctx context.Context, id int64, amount float64) error {
	return nil
}
func (s *socialOAuthUserRepoStub) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	return nil
}
func (s *socialOAuthUserRepoStub) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, ok := s.usersByEmail[email]
	return ok, nil
}
func (s *socialOAuthUserRepoStub) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	return 0, nil
}
func (s *socialOAuthUserRepoStub) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	return nil
}
func (s *socialOAuthUserRepoStub) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	return nil
}
func (s *socialOAuthUserRepoStub) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	return nil
}
func (s *socialOAuthUserRepoStub) EnableTotp(ctx context.Context, userID int64) error { return nil }
func (s *socialOAuthUserRepoStub) DisableTotp(ctx context.Context, userID int64) error {
	return nil
}

type socialOAuthSettingRepoStub struct {
	values map[string]string
}

func (s *socialOAuthSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	if value, ok := s.values[key]; ok {
		return &service.Setting{Key: key, Value: value}, nil
	}
	return nil, service.ErrSettingNotFound
}

func (s *socialOAuthSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *socialOAuthSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *socialOAuthSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *socialOAuthSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *socialOAuthSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *socialOAuthSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type socialOAuthRefreshTokenCacheStub struct{}

func (socialOAuthRefreshTokenCacheStub) StoreRefreshToken(ctx context.Context, tokenHash string, data *service.RefreshTokenData, ttl time.Duration) error {
	return nil
}
func (socialOAuthRefreshTokenCacheStub) GetRefreshToken(ctx context.Context, tokenHash string) (*service.RefreshTokenData, error) {
	return nil, service.ErrRefreshTokenNotFound
}
func (socialOAuthRefreshTokenCacheStub) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}
func (socialOAuthRefreshTokenCacheStub) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	return nil
}
func (socialOAuthRefreshTokenCacheStub) DeleteTokenFamily(ctx context.Context, familyID string) error {
	return nil
}
func (socialOAuthRefreshTokenCacheStub) AddToUserTokenSet(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
	return nil
}
func (socialOAuthRefreshTokenCacheStub) AddToFamilyTokenSet(ctx context.Context, familyID string, tokenHash string, ttl time.Duration) error {
	return nil
}
func (socialOAuthRefreshTokenCacheStub) GetUserTokenHashes(ctx context.Context, userID int64) ([]string, error) {
	return nil, nil
}
func (socialOAuthRefreshTokenCacheStub) GetFamilyTokenHashes(ctx context.Context, familyID string) ([]string, error) {
	return nil, nil
}
func (socialOAuthRefreshTokenCacheStub) IsTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) {
	return false, nil
}

type socialOAuthIdentityRepoStub struct {
	items []*service.AuthIdentity
}

func newSocialOAuthIdentityRepoStub() *socialOAuthIdentityRepoStub {
	return &socialOAuthIdentityRepoStub{items: make([]*service.AuthIdentity, 0)}
}

func (r *socialOAuthIdentityRepoStub) Create(ctx context.Context, identity *service.AuthIdentity) error {
	if identity == nil {
		return service.ErrAuthIdentityNotFound
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

func (r *socialOAuthIdentityRepoStub) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*service.AuthIdentity, error) {
	for _, item := range r.items {
		if item.Provider == provider && item.ProviderUserID == providerUserID {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrAuthIdentityNotFound
}

func (r *socialOAuthIdentityRepoStub) GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*service.AuthIdentity, error) {
	for _, item := range r.items {
		if item.UserID == userID && item.Provider == provider {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrAuthIdentityNotFound
}

func (r *socialOAuthIdentityRepoStub) ListByUserID(ctx context.Context, userID int64) ([]*service.AuthIdentity, error) {
	out := make([]*service.AuthIdentity, 0)
	for _, item := range r.items {
		if item.UserID != userID {
			continue
		}
		clone := *item
		out = append(out, &clone)
	}
	return out, nil
}

func (r *socialOAuthIdentityRepoStub) DeleteByUserIDAndProvider(ctx context.Context, userID int64, provider string) error {
	return service.ErrAuthIdentityNotFound
}

type socialOAuthRedeemRepoStub struct{}

func (socialOAuthRedeemRepoStub) Create(ctx context.Context, code *service.RedeemCode) error {
	return nil
}
func (socialOAuthRedeemRepoStub) CreateBatch(ctx context.Context, codes []service.RedeemCode) error {
	return nil
}
func (socialOAuthRedeemRepoStub) GetByID(ctx context.Context, id int64) (*service.RedeemCode, error) {
	return nil, service.ErrRedeemCodeNotFound
}
func (socialOAuthRedeemRepoStub) GetByCode(ctx context.Context, code string) (*service.RedeemCode, error) {
	return &service.RedeemCode{
		ID:     1,
		Code:   code,
		Type:   service.RedeemTypeInvitation,
		Status: service.StatusUnused,
	}, nil
}
func (socialOAuthRedeemRepoStub) Update(ctx context.Context, code *service.RedeemCode) error {
	return nil
}
func (socialOAuthRedeemRepoStub) Delete(ctx context.Context, id int64) error      { return nil }
func (socialOAuthRedeemRepoStub) Use(ctx context.Context, id, userID int64) error { return nil }
func (socialOAuthRedeemRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (socialOAuthRedeemRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (socialOAuthRedeemRepoStub) ListByUser(ctx context.Context, userID int64, limit int) ([]service.RedeemCode, error) {
	return nil, nil
}
func (socialOAuthRedeemRepoStub) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (socialOAuthRedeemRepoStub) SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error) {
	return 0, nil
}

type socialOAuthAffiliateRepoStub struct{}

func (socialOAuthAffiliateRepoStub) GetUserAffiliate(ctx context.Context, userID int64) (*service.UserAffiliate, error) {
	return nil, nil
}
func (socialOAuthAffiliateRepoStub) EnsureAffiliateRow(ctx context.Context, userID int64, affCode string) (bool, error) {
	return true, nil
}
func (socialOAuthAffiliateRepoStub) BindInviterByCode(ctx context.Context, inviteeUserID int64, affCode string) (int64, bool, error) {
	return 0, false, nil
}
func (socialOAuthAffiliateRepoStub) AccrueTopupRebate(ctx context.Context, redeemCodeID int64, inviteeUserID int64, creditedAmount float64, policy service.AffiliateRebatePolicy) (float64, error) {
	return 0, nil
}
func (socialOAuthAffiliateRepoStub) ThawFrozenIfNeeded(ctx context.Context, inviterUserID int64) (float64, error) {
	return 0, nil
}
func (socialOAuthAffiliateRepoStub) TransferToBalance(ctx context.Context, userID int64) (*service.AffiliateTransferResult, error) {
	return nil, nil
}
func (socialOAuthAffiliateRepoStub) ListAffiliateUsers(ctx context.Context, params pagination.PaginationParams, filters service.AffiliateAdminUserListFilters) ([]service.AffiliateAdminUser, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (socialOAuthAffiliateRepoStub) LookupAffiliateUsers(ctx context.Context, q string, limit int) ([]service.AffiliateAdminUser, error) {
	return nil, nil
}
func (socialOAuthAffiliateRepoStub) UpdateAffiliateUserCustom(ctx context.Context, userID int64, update service.AffiliateAdminUserCustomUpdate, newAffCodeForClear string) (*service.UserAffiliate, error) {
	return nil, nil
}
func (socialOAuthAffiliateRepoStub) ResetAffiliateUserCustom(ctx context.Context, userID int64, newAffCode string) (*service.UserAffiliate, error) {
	return nil, nil
}
func (socialOAuthAffiliateRepoStub) BatchUpdateAffiliateUserCustomRates(ctx context.Context, userIDs []int64, customRatePercent float64) (int, error) {
	return 0, nil
}

func newSocialOAuthTestHandler(t *testing.T, settings map[string]string) (*AuthHandler, *service.AuthIdentityService, *service.AuthService, *socialOAuthIdentityRepoStub) {
	t.Helper()

	userRepo := &socialOAuthUserRepoStub{
		usersByID:    map[int64]*service.User{},
		usersByEmail: map[string]*service.User{},
		nextID:       100,
	}
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:     "social-oauth-handler-test-secret",
			ExpireHour: 1,
		},
		Default: config.DefaultConfig{
			UserBalance:     1,
			UserConcurrency: 2,
		},
	}
	settingService := service.NewSettingService(&socialOAuthSettingRepoStub{values: settings}, cfg)
	affiliateService := service.NewAffiliateService(settingService, socialOAuthAffiliateRepoStub{})
	authService := service.NewAuthService(
		nil,
		userRepo,
		socialOAuthRedeemRepoStub{},
		socialOAuthRefreshTokenCacheStub{},
		cfg,
		settingService,
		nil,
		nil,
		nil,
		nil,
		affiliateService,
		nil,
	)
	repo := newSocialOAuthIdentityRepoStub()
	identities := service.NewAuthIdentityService(repo, userRepo, authService)
	authService.SetAuthIdentityRepository(repo)
	userService := service.NewUserService(userRepo, nil, nil)
	handler := NewAuthHandler(cfg, authService, userService, settingService, nil, nil, nil)
	handler.SetAuthIdentityService(identities)
	return handler, identities, authService, repo
}

func createPendingOAuthTokenForTest(t *testing.T, secret string, claims map[string]any) string {
	t.Helper()
	now := time.Now()
	claims["purpose"] = "pending_oauth_registration"
	claims["exp"] = now.Add(10 * time.Minute).Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func TestCompleteSocialOAuthRegistration_ProviderMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, _, _ := newSocialOAuthTestHandler(t, map[string]string{
		service.SettingKeyRegistrationEnabled: "true",
	})
	pendingToken := createPendingOAuthTokenForTest(t, "social-oauth-handler-test-secret", map[string]any{
		"email":            "pending@example.com",
		"username":         "pending-user",
		"provider":         service.AuthProviderGitHub,
		"provider_user_id": "gh-pending-1",
		"email_verified":   true,
	})

	body := bytes.NewBufferString(`{"pending_oauth_token":"` + pendingToken + `","invitation_code":"invite-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/google/complete", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "provider", Value: "google"}}

	handler.CompleteSocialOAuthRegistration(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "OAUTH_PROVIDER_MISMATCH", payload["reason"])
}

func TestCompleteSocialOAuthRegistration_PendingInvitationSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, _, repo := newSocialOAuthTestHandler(t, map[string]string{
		service.SettingKeyRegistrationEnabled:   "true",
		service.SettingKeyInvitationCodeEnabled: "true",
	})
	pendingToken := createPendingOAuthTokenForTest(t, "social-oauth-handler-test-secret", map[string]any{
		"email":            "new-social@example.com",
		"username":         "new-social-user",
		"provider":         service.AuthProviderGitHub,
		"provider_user_id": "gh-complete-1",
		"email_verified":   true,
		"display_name":     "New Social User",
		"avatar_url":       "https://avatars.example.com/social.png",
	})

	body := bytes.NewBufferString(`{"pending_oauth_token":"` + pendingToken + `","invitation_code":"INVITE-CODE"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/github/complete", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "provider", Value: "github"}}

	handler.CompleteSocialOAuthRegistration(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, float64(0), payload["code"])
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, strings.TrimSpace(anyString(data["access_token"])))
	require.NotEmpty(t, strings.TrimSpace(anyString(data["refresh_token"])))
	require.Equal(t, "Bearer", anyString(data["token_type"]))

	require.Len(t, repo.items, 1)
	require.Equal(t, service.AuthProviderGitHub, repo.items[0].Provider)
	require.Equal(t, "gh-complete-1", repo.items[0].ProviderUserID)
	require.Equal(t, "new-social@example.com", repo.items[0].Email)
	require.True(t, repo.items[0].EmailVerified)
}

func TestSocialOAuthCallback_RedirectsPendingInvitationFragment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, _, repo := newSocialOAuthTestHandler(t, map[string]string{
		service.SettingKeyRegistrationEnabled:     "true",
		service.SettingKeyInvitationCodeEnabled:   "true",
		service.SettingKeyGitHubOAuthEnabled:      "true",
		service.SettingKeyGitHubOAuthClientID:     "github-client-id",
		service.SettingKeyGitHubOAuthClientSecret: "github-client-secret",
		service.SettingKeyGitHubOAuthRedirectURL:  "https://api.example.com/api/v1/auth/oauth/github/callback",
	})

	originalExchange := socialOAuthExchangeCodeFn
	originalFetch := socialOAuthFetchUserInfoFn
	t.Cleanup(func() {
		socialOAuthExchangeCodeFn = originalExchange
		socialOAuthFetchUserInfoFn = originalFetch
	})

	socialOAuthExchangeCodeFn = func(_ctx context.Context, _cfg service.SocialOAuthConfig, code string, verifier string) (*socialOAuthTokenResponse, error) {
		require.Equal(t, "oauth-code-1", code)
		require.Equal(t, "", verifier)
		return &socialOAuthTokenResponse{
			AccessToken: "upstream-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}, nil
	}
	socialOAuthFetchUserInfoFn = func(_ctx context.Context, _cfg service.SocialOAuthConfig, tokenResp *socialOAuthTokenResponse) (*socialOAuthUserInfo, error) {
		require.NotNil(t, tokenResp)
		return &socialOAuthUserInfo{
			ProviderUserID: "gh-callback-1",
			Email:          "callback@example.com",
			EmailVerified:  true,
			DisplayName:    "Callback User",
			AvatarURL:      "https://avatars.example.com/callback.png",
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github/callback?code=oauth-code-1&state=state-1", nil)
	req.AddCookie(&http.Cookie{
		Name:  socialOAuthStateCookieName(service.AuthProviderGitHub),
		Value: encodeCookieValue("state-1"),
		Path:  socialOAuthCookiePath(service.AuthProviderGitHub),
	})
	req.AddCookie(&http.Cookie{
		Name:  socialOAuthRedirectCookieName(service.AuthProviderGitHub),
		Value: encodeCookieValue("/workspace"),
		Path:  socialOAuthCookiePath(service.AuthProviderGitHub),
	})
	req.AddCookie(&http.Cookie{
		Name:  socialOAuthModeCookieName(service.AuthProviderGitHub),
		Value: encodeCookieValue("login"),
		Path:  socialOAuthCookiePath(service.AuthProviderGitHub),
	})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "provider", Value: "github"}}

	handler.SocialOAuthCallback(c)

	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	require.NotEmpty(t, location)
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	fragment, err := url.ParseQuery(parsed.Fragment)
	require.NoError(t, err)
	require.Equal(t, "invitation_required", fragment.Get("error"))
	require.Equal(t, "github", fragment.Get("provider"))
	require.Equal(t, "login", fragment.Get("mode"))
	require.Equal(t, "/workspace", fragment.Get("redirect"))
	require.NotEmpty(t, fragment.Get("pending_oauth_token"))
	require.Empty(t, repo.items)
}

func anyString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
