//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNormalizeExternalModelCatalogViewMode(t *testing.T) {
	require.Equal(t, ExternalModelCatalogViewModeFollowKeyBinding, NormalizeExternalModelCatalogViewMode(""))
	require.Equal(t, ExternalModelCatalogViewModeFollowKeyBinding, NormalizeExternalModelCatalogViewMode("unknown"))
	require.Equal(t, ExternalModelCatalogViewModeFollowKeyBinding, NormalizeExternalModelCatalogViewMode(ExternalModelCatalogViewModeFollowKeyBinding))
	require.Equal(t, ExternalModelCatalogViewModeGroupFirst, NormalizeExternalModelCatalogViewMode(ExternalModelCatalogViewModeGroupFirst))
	require.Equal(t, ExternalModelCatalogViewModeModelOnly, NormalizeExternalModelCatalogViewMode(ExternalModelCatalogViewModeModelOnly))
}

func TestValidateExternalModelCatalogViewMode(t *testing.T) {
	for _, mode := range []string{
		ExternalModelCatalogViewModeFollowKeyBinding,
		ExternalModelCatalogViewModeGroupFirst,
		ExternalModelCatalogViewModeModelOnly,
	} {
		require.NoError(t, ValidateExternalModelCatalogViewMode(mode))
	}

	err := ValidateExternalModelCatalogViewMode("invalid")
	require.Error(t, err)
	require.Equal(t, "EXTERNAL_MODEL_CATALOG_VIEW_MODE_INVALID", infraerrors.Reason(err))
}

func TestEffectiveExternalModelCatalogViewMode(t *testing.T) {
	cases := []struct {
		name        string
		catalogMode string
		keyMode     string
		want        string
	}{
		{
			name:        "explicit group first wins",
			catalogMode: ExternalModelCatalogViewModeGroupFirst,
			keyMode:     APIKeyModelBindingModeModelRequired,
			want:        ExternalModelCatalogViewModeGroupFirst,
		},
		{
			name:        "explicit model only wins",
			catalogMode: ExternalModelCatalogViewModeModelOnly,
			keyMode:     APIKeyModelBindingModeGroupAllowed,
			want:        ExternalModelCatalogViewModeModelOnly,
		},
		{
			name:        "follow group allowed",
			catalogMode: ExternalModelCatalogViewModeFollowKeyBinding,
			keyMode:     APIKeyModelBindingModeGroupAllowed,
			want:        ExternalModelCatalogViewModeGroupFirst,
		},
		{
			name:        "follow model required",
			catalogMode: ExternalModelCatalogViewModeFollowKeyBinding,
			keyMode:     APIKeyModelBindingModeModelRequired,
			want:        ExternalModelCatalogViewModeModelOnly,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := &User{
				ExternalModelCatalogViewMode: tc.catalogMode,
				APIKeyModelBindingMode:       tc.keyMode,
			}
			require.Equal(t, tc.want, user.EffectiveExternalModelCatalogViewMode())
			require.Equal(t, tc.want, EffectiveExternalModelCatalogViewMode(user))
		})
	}
}

func TestAPIKeyService_GetExternalModelCatalogView_FiltersByGroupProjection(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:              "etag-external-catalog",
			UpdatedAt:         "2026-05-01T00:00:00Z",
			PublishedAt:       "2026-05-01T00:00:00Z",
			LastRevalidatedAt: "2026-05-01T00:00:00Z",
			PageSize:          10,
			CatalogSource:     PublicModelCatalogSourcePublished,
			Items: []PublicModelCatalogItem{
				testPublishedCatalogRouteItem("gpt-5.4", PlatformOpenAI, "chat", 42),
				testPublishedCatalogRouteItem("gpt-5.4-mini", PlatformOpenAI, "chat", 42),
				testPublishedCatalogRouteItem("claude-sonnet-4", PlatformAnthropic, "chat", 43),
			},
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))

	modelCatalogSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	groupRepo := &publicCatalogGroupRepoStub{
		groups: []Group{
			{ID: 10, Name: "OpenAI", Description: "openai group", Platform: PlatformOpenAI, Status: StatusActive, Priority: 2},
			{ID: 20, Name: "Anthropic", Platform: PlatformAnthropic, Status: StatusActive, Priority: 1},
		},
	}
	gatewaySvc := &GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{
			10: {
				testPublishedCatalogAccount(42, 10, PlatformOpenAI, "gpt-5.4"),
			},
			20: {
				testPublishedCatalogAccount(43, 20, PlatformAnthropic, "claude-sonnet-4"),
			},
		}),
		groupRepo: groupRepo,
		cfg:       &config.Config{},
	}
	modelCatalogSvc.SetGatewayService(gatewaySvc)

	apiKeySvc := NewAPIKeyService(
		nil,
		&publicCatalogUserRepoStub{user: &User{
			ID:                           7,
			Role:                         RoleUser,
			APIKeyModelBindingMode:       APIKeyModelBindingModeGroupAllowed,
			ExternalModelCatalogViewMode: ExternalModelCatalogViewModeFollowKeyBinding,
		}},
		groupRepo,
		publicCatalogUserSubRepoStub{},
		nil,
		nil,
		&config.Config{},
	)
	apiKeySvc.SetGatewayService(gatewaySvc)
	apiKeySvc.SetModelCatalogService(modelCatalogSvc)

	view, err := apiKeySvc.GetExternalModelCatalogView(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ExternalModelCatalogViewModeFollowKeyBinding, view.ExternalModelCatalogViewMode)
	require.Equal(t, ExternalModelCatalogViewModeGroupFirst, view.EffectiveExternalModelCatalogViewMode)
	require.Equal(t, PublicModelCatalogSourcePublished, view.CatalogSource)
	require.Len(t, view.Groups, 2)
	require.Equal(t, int64(20), view.Groups[0].ID)
	require.Equal(t, 1, view.Groups[0].ModelCount)
	require.Equal(t, int64(10), view.Groups[1].ID)
	require.Equal(t, 1, view.Groups[1].ModelCount)

	require.Len(t, view.Items, 2)
	require.ElementsMatch(t, []string{"claude-sonnet-4", "gpt-5.4"}, externalCatalogModelIDs(view.Items))
	require.ElementsMatch(t, []string{"gpt-5.4"}, externalCatalogModelIDs(view.GroupCatalogs["10"]))
	require.ElementsMatch(t, []string{"claude-sonnet-4"}, externalCatalogModelIDs(view.GroupCatalogs["20"]))

	encoded, err := json.Marshal(view)
	require.NoError(t, err)
	body := string(encoded)
	require.NotContains(t, body, "source_account_id")
	require.NotContains(t, body, "source_model_id")
	require.NotContains(t, body, "source_alias")
	require.NotContains(t, body, "target_model_id")
}

func TestAPIKeyService_GetExternalModelCatalogView_ExplicitModelOnlyMode(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			UpdatedAt: "2026-05-01T00:00:00Z",
			PageSize:  10,
			Items: []PublicModelCatalogItem{
				testPublishedCatalogRouteItem("gpt-5.4", PlatformOpenAI, "chat", 42),
			},
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))

	modelCatalogSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	groupRepo := &publicCatalogGroupRepoStub{
		groups: []Group{
			{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		},
	}
	gatewaySvc := &GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{
			10: {
				testPublishedCatalogAccount(42, 10, PlatformOpenAI, "gpt-5.4"),
			},
		}),
		groupRepo: groupRepo,
		cfg:       &config.Config{},
	}
	modelCatalogSvc.SetGatewayService(gatewaySvc)

	apiKeySvc := NewAPIKeyService(
		nil,
		&publicCatalogUserRepoStub{user: &User{
			ID:                           7,
			Role:                         RoleUser,
			APIKeyModelBindingMode:       APIKeyModelBindingModeGroupAllowed,
			ExternalModelCatalogViewMode: ExternalModelCatalogViewModeModelOnly,
		}},
		groupRepo,
		publicCatalogUserSubRepoStub{},
		nil,
		nil,
		&config.Config{},
	)
	apiKeySvc.SetGatewayService(gatewaySvc)
	apiKeySvc.SetModelCatalogService(modelCatalogSvc)

	view, err := apiKeySvc.GetExternalModelCatalogView(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ExternalModelCatalogViewModeModelOnly, view.ExternalModelCatalogViewMode)
	require.Equal(t, ExternalModelCatalogViewModeModelOnly, view.EffectiveExternalModelCatalogViewMode)
	require.Len(t, view.Items, 1)
	require.Equal(t, "gpt-5.4", view.Items[0].Model)
}

func externalCatalogModelIDs(items []PublicModelCatalogItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Model)
	}
	return out
}
