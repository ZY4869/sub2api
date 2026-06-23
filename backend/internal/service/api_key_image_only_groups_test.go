package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type apiKeyImageOnlyAccountRepoStub struct {
	AccountRepository
	accountsByGroup map[int64][]Account
}

func (s *apiKeyImageOnlyAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, platforms []string) ([]Account, error) {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	var out []Account
	for _, account := range s.accountsByGroup[groupID] {
		if !account.IsSchedulable() {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[account.Platform]; !ok {
				continue
			}
		}
		out = append(out, account)
	}
	return out, nil
}

func TestAPIKeyImageOnly_NormalizeRequiresGroup(t *testing.T) {
	svc := &APIKeyService{}

	bindings, err := svc.normalizeImageOnlyGroupBindings(context.Background(), nil)
	require.ErrorIs(t, err, ErrImageOnlyGroupRequired)
	require.Nil(t, bindings)
}

func TestAPIKeyImageOnly_NormalizeEmptySelectionToGroupImageModels(t *testing.T) {
	group := &Group{ID: 101, Name: "openai-image", Platform: PlatformOpenAI, Status: StatusActive}
	svc := &APIKeyService{
		gatewayService: &GatewayService{
			accountRepo: &apiKeyImageOnlyAccountRepoStub{
				accountsByGroup: map[int64][]Account{
					group.ID: {
						newAPIKeyImageOnlyAccount(1, group.ID, PlatformOpenAI, []string{"gpt-image-2", "gpt-5.4"}),
					},
				},
			},
		},
	}

	bindings, err := svc.normalizeImageOnlyGroupBindings(context.Background(), []APIKeyGroupBinding{{
		GroupID: group.ID,
		Group:   group,
	}})
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	require.Equal(t, []string{"gpt-image-2"}, bindings[0].ModelPatterns)
}

func TestAPIKeyImageOnly_NormalizeRejectsGroupsWithoutImageModels(t *testing.T) {
	group := &Group{ID: 102, Name: "openai-chat", Platform: PlatformOpenAI, Status: StatusActive}
	svc := &APIKeyService{
		gatewayService: &GatewayService{
			accountRepo: &apiKeyImageOnlyAccountRepoStub{
				accountsByGroup: map[int64][]Account{
					group.ID: {
						newAPIKeyImageOnlyAccount(2, group.ID, PlatformOpenAI, []string{"gpt-5.4"}),
					},
				},
			},
		},
	}

	_, err := svc.normalizeImageOnlyGroupBindings(context.Background(), []APIKeyGroupBinding{{
		GroupID: group.ID,
		Group:   group,
	}})
	require.ErrorIs(t, err, ErrImageOnlyGroupHasNoImageModels)
}

func TestAPIKeyImageOnly_NormalizeRejectsExplicitNonImageSelection(t *testing.T) {
	group := &Group{ID: 103, Name: "openai-mixed", Platform: PlatformOpenAI, Status: StatusActive}
	svc := &APIKeyService{
		gatewayService: &GatewayService{
			accountRepo: &apiKeyImageOnlyAccountRepoStub{
				accountsByGroup: map[int64][]Account{
					group.ID: {
						newAPIKeyImageOnlyAccount(3, group.ID, PlatformOpenAI, []string{"gpt-image-2", "gpt-5.4"}),
					},
				},
			},
		},
	}

	_, err := svc.normalizeImageOnlyGroupBindings(context.Background(), []APIKeyGroupBinding{{
		GroupID:       group.ID,
		Group:         group,
		ModelPatterns: []string{"gpt-5.4"},
	}})
	require.ErrorIs(t, err, ErrImageOnlySelectedModelsInvalid)
}

func TestAPIKeyImageOnly_ConfiguredModelScopeHelper(t *testing.T) {
	group := &Group{ID: 104, Name: "openai-image", Platform: PlatformOpenAI, Status: StatusActive}
	apiKey := &APIKey{
		ImageOnlyEnabled: true,
		GroupBindings: []APIKeyGroupBinding{{
			GroupID:       group.ID,
			Group:         group,
			ModelPatterns: []string{"gpt-image-2"},
		}},
	}

	require.True(t, APIKeyAllowsConfiguredModel(apiKey, "gpt-image-2"))
	require.False(t, APIKeyAllowsConfiguredModel(apiKey, "gpt-image-1.5"))
	require.False(t, APIKeyAllowsConfiguredModel(apiKey, "gpt-5.4"))
	require.False(t, APIKeyAllowsConfiguredModel(&APIKey{ImageOnlyEnabled: true}, "gpt-image-2"))
}

func TestAPIKeyImageOnly_CreateRequiresGroup(t *testing.T) {
	repo := &apiKeyImageOnlyCreateRepoStub{}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo: &apiKeyImageOnlyUserRepoStub{
			user: &User{ID: 7, Role: RoleUser},
		},
	}

	_, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:             "image-only",
		ImageOnlyEnabled: true,
	})
	require.ErrorIs(t, err, ErrImageOnlyGroupRequired)
	require.False(t, repo.createCalled)
}

func TestAPIKeyImageOnly_UpdateRequiresExistingOrRequestedGroup(t *testing.T) {
	enabled := true
	repo := &apiKeyImageOnlyUpdateRepoStub{
		key: &APIKey{ID: 9, UserID: 7, Key: "sk-test", Status: StatusActive},
	}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo: &apiKeyImageOnlyUserRepoStub{
			user: &User{ID: 7, Role: RoleUser},
		},
	}

	_, err := svc.Update(context.Background(), 9, 7, UpdateAPIKeyRequest{
		ImageOnlyEnabled: &enabled,
	})
	require.ErrorIs(t, err, ErrImageOnlyGroupRequired)
	require.False(t, repo.updateCalled)
}

func newAPIKeyImageOnlyAccount(id, groupID int64, platform string, models []string) Account {
	supported := make([]any, 0, len(models))
	for _, modelID := range models {
		supported = append(supported, modelID)
	}
	return Account{
		ID:          id,
		Name:        "image-only-account",
		Platform:    platform,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://example.test",
		},
		Extra: map[string]any{
			"model_scope_v2": map[string]any{
				"supported_models_by_provider": map[string]any{
					normalizePublicImageProvider(platform): supported,
				},
			},
		},
		AccountGroups: []AccountGroup{{AccountID: id, GroupID: groupID}},
	}
}

type apiKeyImageOnlyUserRepoStub struct {
	UserRepository
	user *User
}

func (s *apiKeyImageOnlyUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	clone := *s.user
	clone.ID = id
	return &clone, nil
}

type apiKeyImageOnlyCreateRepoStub struct {
	APIKeyRepository
	createCalled bool
}

func (s *apiKeyImageOnlyCreateRepoStub) Create(context.Context, *APIKey) error {
	s.createCalled = true
	return errors.New("unexpected create")
}

type apiKeyImageOnlyUpdateRepoStub struct {
	APIKeyRepository
	key          *APIKey
	updateCalled bool
}

func (s *apiKeyImageOnlyUpdateRepoStub) GetByID(context.Context, int64) (*APIKey, error) {
	if s.key == nil {
		return nil, ErrAPIKeyNotFound
	}
	clone := *s.key
	return &clone, nil
}

func (s *apiKeyImageOnlyUpdateRepoStub) Update(context.Context, *APIKey) error {
	s.updateCalled = true
	return errors.New("unexpected update")
}

func TestAPIKeyImageOnly_UserCreateClearsImageCountBilling(t *testing.T) {
	fixture := newAPIKeyImageCountControlsFixture()
	customKey := "sk-image-user-create"
	svc := newAPIKeyImageCountControlsService(RoleUser, fixture)

	key, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:                     "image-only",
		CustomKey:                &customKey,
		Groups:                   &[]APIKeyGroupUpdateInput{{GroupID: 101}},
		ImageOnlyEnabled:         true,
		ImageCountBillingEnabled: true,
		ImageMaxCount:            100,
		ImageCountWeights:        map[string]int{"1K": 2, "2K": 3, "4K": 4},
	})

	require.NoError(t, err)
	require.True(t, key.ImageOnlyEnabled)
	require.False(t, key.ImageCountBillingEnabled)
	require.Equal(t, 0, key.ImageMaxCount)
	require.Equal(t, 0, key.ImageCountUsed)
	require.Equal(t, DefaultAPIKeyImageCountWeights(), key.ImageCountWeights)
}

func TestAPIKeyImageOnly_UserUpdateClearsExistingImageCountBilling(t *testing.T) {
	fixture := newAPIKeyImageCountControlsFixture()
	fixture.keyRepo.key = &APIKey{
		ID:                       9,
		UserID:                   7,
		Key:                      "sk-image-user-update",
		Name:                     "old",
		Status:                   StatusActive,
		ImageOnlyEnabled:         true,
		ImageCountBillingEnabled: true,
		ImageMaxCount:            100,
		ImageCountUsed:           25,
		ImageCountWeights:        map[string]int{"1K": 2, "2K": 3, "4K": 4},
		GroupBindings: []APIKeyGroupBinding{{
			APIKeyID:            9,
			GroupID:             101,
			Group:               fixture.group,
			ModelPatterns:       []string{"gpt-image-2"},
			QuotaUsed:           0,
			QuotaUsedByCurrency: nil,
		}},
	}
	fixture.keyRepo.bindings = fixture.keyRepo.key.GroupBindings
	svc := newAPIKeyImageCountControlsService(RoleUser, fixture)
	name := "renamed"

	key, err := svc.Update(context.Background(), 9, 7, UpdateAPIKeyRequest{
		Name: &name,
	})

	require.NoError(t, err)
	require.Equal(t, "renamed", key.Name)
	require.True(t, key.ImageOnlyEnabled)
	require.False(t, key.ImageCountBillingEnabled)
	require.Equal(t, 0, key.ImageMaxCount)
	require.Equal(t, 0, key.ImageCountUsed)
	require.Equal(t, DefaultAPIKeyImageCountWeights(), key.ImageCountWeights)
}

func TestAPIKeyImageOnly_AdminCreateAndUpdatePreserveImageCountBilling(t *testing.T) {
	fixture := newAPIKeyImageCountControlsFixture()
	customKey := "sk-image-admin-create"
	svc := newAPIKeyImageCountControlsService(RoleAdmin, fixture)

	created, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:                     "image-admin",
		CustomKey:                &customKey,
		Groups:                   &[]APIKeyGroupUpdateInput{{GroupID: 101}},
		ImageOnlyEnabled:         true,
		ImageCountBillingEnabled: true,
		ImageMaxCount:            100,
		ImageCountWeights:        map[string]int{"1K": 2, "2K": 3, "4K": 4},
	})

	require.NoError(t, err)
	require.True(t, created.ImageCountBillingEnabled)
	require.Equal(t, 100, created.ImageMaxCount)
	require.Equal(t, map[string]int{"1K": 2, "2K": 3, "4K": 4}, created.ImageCountWeights)

	updatedMax := 200
	updatedWeights := map[string]int{"1K": 1, "2K": 2, "4K": 6}
	updated, err := svc.Update(context.Background(), created.ID, 7, UpdateAPIKeyRequest{
		ImageMaxCount:     &updatedMax,
		ImageCountWeights: updatedWeights,
	})

	require.NoError(t, err)
	require.True(t, updated.ImageCountBillingEnabled)
	require.Equal(t, 200, updated.ImageMaxCount)
	require.Equal(t, updatedWeights, updated.ImageCountWeights)
}

func newAPIKeyImageCountControlsService(role string, fixture *apiKeyImageCountControlsFixture) *APIKeyService {
	return &APIKeyService{
		apiKeyRepo: fixture.keyRepo,
		userRepo:   &apiKeyImageOnlyUserRepoStub{user: &User{ID: 7, Role: role}},
		groupRepo:  fixture.groupRepo,
		gatewayService: &GatewayService{
			accountRepo: &apiKeyImageOnlyAccountRepoStub{
				accountsByGroup: map[int64][]Account{
					fixture.group.ID: {
						newAPIKeyImageOnlyAccount(1, fixture.group.ID, PlatformOpenAI, []string{"gpt-image-2"}),
					},
				},
			},
		},
	}
}

func newAPIKeyImageCountControlsFixture() *apiKeyImageCountControlsFixture {
	group := &Group{ID: 101, Name: "openai-image", Platform: PlatformOpenAI, Status: StatusActive}
	return &apiKeyImageCountControlsFixture{
		group:     group,
		keyRepo:   &apiKeyImageCountControlsAPIKeyRepoStub{},
		groupRepo: &apiKeyImageCountControlsGroupRepoStub{group: group},
	}
}

type apiKeyImageCountControlsFixture struct {
	group     *Group
	keyRepo   *apiKeyImageCountControlsAPIKeyRepoStub
	groupRepo *apiKeyImageCountControlsGroupRepoStub
}

type apiKeyImageCountControlsAPIKeyRepoStub struct {
	APIKeyRepository
	key      *APIKey
	bindings []APIKeyGroupBinding
	nextID   int64
}

func (s *apiKeyImageCountControlsAPIKeyRepoStub) GetByID(_ context.Context, id int64) (*APIKey, error) {
	if s.key == nil || s.key.ID != id {
		return nil, ErrAPIKeyNotFound
	}
	clone := *s.key
	clone.ImageCountWeights = CloneAPIKeyImageCountWeights(s.key.ImageCountWeights)
	clone.GroupBindings = append([]APIKeyGroupBinding(nil), s.bindings...)
	return &clone, nil
}

func (s *apiKeyImageCountControlsAPIKeyRepoStub) Create(_ context.Context, key *APIKey) error {
	s.nextID++
	clone := *key
	clone.ID = s.nextID
	clone.ImageCountWeights = CloneAPIKeyImageCountWeights(key.ImageCountWeights)
	s.key = &clone
	key.ID = clone.ID
	return nil
}

func (s *apiKeyImageCountControlsAPIKeyRepoStub) Update(_ context.Context, key *APIKey) error {
	clone := *key
	clone.ImageCountWeights = CloneAPIKeyImageCountWeights(key.ImageCountWeights)
	s.key = &clone
	return nil
}

func (s *apiKeyImageCountControlsAPIKeyRepoStub) ExistsByKey(context.Context, string) (bool, error) {
	return false, nil
}

func (s *apiKeyImageCountControlsAPIKeyRepoStub) GetAPIKeyGroups(context.Context, int64) ([]APIKeyGroupBinding, error) {
	return append([]APIKeyGroupBinding(nil), s.bindings...), nil
}

func (s *apiKeyImageCountControlsAPIKeyRepoStub) SetAPIKeyGroups(_ context.Context, keyID int64, bindings []APIKeyGroupBinding) error {
	s.bindings = append([]APIKeyGroupBinding(nil), bindings...)
	if s.key != nil && s.key.ID == keyID {
		s.key.GroupBindings = append([]APIKeyGroupBinding(nil), bindings...)
	}
	return nil
}

type apiKeyImageCountControlsGroupRepoStub struct {
	GroupRepository
	group *Group
}

func (s *apiKeyImageCountControlsGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if s.group == nil || s.group.ID != id {
		return nil, ErrGroupNotFound
	}
	clone := *s.group
	return &clone, nil
}

func (s *apiKeyImageCountControlsGroupRepoStub) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return s.GetByID(ctx, id)
}
