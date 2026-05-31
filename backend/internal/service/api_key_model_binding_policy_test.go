//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyService_ValidateUserAPIKeyModelBindings_EmptySelectionAllowsWholePublishedCatalog(t *testing.T) {
	svc := &APIKeyService{
		gatewayService: &GatewayService{
			accountRepo: &mockAccountRepoForPlatform{
				accounts: []Account{
					{
						ID:          1,
						Platform:    PlatformOpenAI,
						Type:        AccountTypeAPIKey,
						Status:      StatusActive,
						Schedulable: true,
						Credentials: map[string]any{
							"model_mapping": map[string]any{
								"public-gpt": "upstream-gpt",
							},
						},
					},
				},
			},
		},
	}
	group := &Group{ID: 10, Platform: PlatformOpenAI, Status: StatusActive}
	user := &User{ID: 7, APIKeyModelBindingMode: APIKeyModelBindingModeModelRequired}

	err := svc.validateUserAPIKeyModelBindings(context.Background(), user, []APIKeyGroupBinding{{GroupID: group.ID, Group: group}})
	require.NoError(t, err)

	err = svc.validateUserAPIKeyModelBindings(context.Background(), user, []APIKeyGroupBinding{{GroupID: group.ID, Group: group, ModelPatterns: []string{"public-gpt"}}})
	require.NoError(t, err)

	err = svc.validateUserAPIKeyModelBindings(context.Background(), user, []APIKeyGroupBinding{{GroupID: group.ID, Group: group, ModelPatterns: []string{"public-*"}}})
	require.ErrorIs(t, err, ErrAPIKeyModelPatternForbidden)

	err = svc.validateUserAPIKeyModelBindings(context.Background(), user, []APIKeyGroupBinding{{GroupID: group.ID, Group: group, ModelPatterns: []string{"hidden-gpt"}}})
	require.ErrorIs(t, err, ErrAPIKeyModelNotVisible)
}

func TestAPIKeyService_ValidateUserAPIKeyModelBindings_GroupAllowed(t *testing.T) {
	svc := &APIKeyService{}
	user := &User{ID: 7, APIKeyModelBindingMode: APIKeyModelBindingModeGroupAllowed}
	group := &Group{ID: 10, Platform: PlatformOpenAI, Status: StatusActive}

	err := svc.validateUserAPIKeyModelBindings(context.Background(), user, []APIKeyGroupBinding{{GroupID: group.ID, Group: group}})
	require.NoError(t, err)
}

func TestAPIKeyService_ValidateUserAPIKeyModelBindings_ImageOnlyEmptySelectionNormalizesToImages(t *testing.T) {
	svc := &APIKeyService{
		gatewayService: &GatewayService{
			accountRepo: &mockAccountRepoForPlatform{
				accounts: []Account{
					{
						ID:          1,
						Platform:    PlatformOpenAI,
						Type:        AccountTypeAPIKey,
						Status:      StatusActive,
						Schedulable: true,
						Credentials: map[string]any{
							"model_mapping": map[string]any{
								"gpt-image-2": "gpt-image-2",
							},
						},
					},
				},
			},
		},
	}
	user := &User{ID: 7, APIKeyModelBindingMode: APIKeyModelBindingModeModelRequired}
	group := &Group{ID: 10, Platform: PlatformOpenAI, Status: StatusActive}
	bindings := []APIKeyGroupBinding{{GroupID: group.ID, Group: group}}

	err := svc.validateUserAPIKeyModelBindings(context.Background(), user, bindings)
	require.NoError(t, err)

	normalized, err := svc.normalizeImageOnlyGroupBindings(context.Background(), bindings)
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-image-2"}, normalized[0].ModelPatterns)
	require.NoError(t, svc.validateUserAPIKeyModelBindings(context.Background(), user, normalized))
}
