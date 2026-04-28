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
