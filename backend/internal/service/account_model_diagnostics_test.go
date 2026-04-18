//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type diagnosticsAPIKeyRepoStub struct {
	apiKeyRepoStubForGroupUpdate
	keysByGroup map[int64][]APIKey
	listErr     error
}

func (s *diagnosticsAPIKeyRepoStub) ListByGroupID(_ context.Context, groupID int64, params pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	if s.listErr != nil {
		return nil, nil, s.listErr
	}
	keys := append([]APIKey(nil), s.keysByGroup[groupID]...)
	return keys, &pagination.PaginationResult{
		Total:    int64(len(keys)),
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

func newDiagnosticsServiceForTest(
	account Account,
	group *Group,
	apiKey APIKey,
	importSvc *AccountModelImportService,
) *AccountModelDiagnosticsService {
	accountCopy := account
	groupCopy := *group
	accountRepo := &mockAccountRepoForPlatform{
		accountsByID: map[int64]*Account{
			account.ID: &accountCopy,
		},
	}
	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			group.ID: &groupCopy,
		},
	}
	apiKeyRepo := &diagnosticsAPIKeyRepoStub{
		keysByGroup: map[int64][]APIKey{
			group.ID: {apiKey},
		},
	}
	return NewAccountModelDiagnosticsService(accountRepo, apiKeyRepo, groupRepo, importSvc)
}

func TestAccountModelDiagnosticsService_DiagnoseOK(t *testing.T) {
	service := newDiagnosticsServiceForTest(
		Account{
			ID:          1,
			Name:        "openai-apikey",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{10},
			Extra: map[string]any{
				"model_probe_snapshot": map[string]any{
					"models":       []string{"gpt-4.1-mini"},
					"updated_at":   "2026-04-01T00:00:00Z",
					"source":       AccountModelProbeSnapshotSourceManualProbe,
					"probe_source": accountModelProbeSourceUpstream,
				},
			},
		},
		&Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		APIKey{
			ID:               101,
			Name:             "public-key",
			ModelDisplayMode: APIKeyModelDisplayModeAliasOnly,
			GroupBindings: []APIKeyGroupBinding{
				{
					GroupID: 10,
					Group:   &Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
				},
			},
		},
		nil,
	)

	result, err := service.Diagnose(context.Background(), 1, false)

	require.NoError(t, err)
	require.Equal(t, AccountModelDiagnosticsStatusOK, result.Status)
	require.Equal(t, []string{"gpt-4.1-mini"}, result.SavedModels)
	require.Len(t, result.PublicModelsPreview, 1)
	require.Len(t, result.GroupExposures, 1)
	require.Len(t, result.GroupExposures[0].PublicModels, 1)
}

func TestAccountModelDiagnosticsService_DiagnoseDegradedWhenLiveProbeFails(t *testing.T) {
	importSvc := NewAccountModelImportService(nil, nil, &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":"forbidden"}`,
	}, nil)
	service := newDiagnosticsServiceForTest(
		Account{
			ID:          2,
			Name:        "openai-apikey",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{10},
			Credentials: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://openai.example.test",
			},
			Extra: map[string]any{
				"model_probe_snapshot": map[string]any{
					"models":       []string{"gpt-4.1-mini"},
					"updated_at":   "2026-04-01T00:00:00Z",
					"source":       AccountModelProbeSnapshotSourceManualProbe,
					"probe_source": accountModelProbeSourceUpstream,
				},
			},
		},
		&Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		APIKey{
			ID:               102,
			Name:             "public-key",
			ModelDisplayMode: APIKeyModelDisplayModeAliasOnly,
			GroupBindings: []APIKeyGroupBinding{
				{
					GroupID: 10,
					Group:   &Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
				},
			},
		},
		importSvc,
	)

	result, err := service.Diagnose(context.Background(), 2, true)

	require.NoError(t, err)
	require.Equal(t, AccountModelDiagnosticsStatusDegraded, result.Status)
	require.Len(t, result.GroupExposures[0].PublicModels, 1)
	require.Empty(t, result.DetectedModels)
}

func TestAccountModelDiagnosticsService_DiagnoseFilteredEmptyWhenBindingsFilterModels(t *testing.T) {
	service := newDiagnosticsServiceForTest(
		Account{
			ID:          3,
			Name:        "openai-apikey",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{10},
			Extra: map[string]any{
				"model_probe_snapshot": map[string]any{
					"models":       []string{"gpt-4.1-mini"},
					"updated_at":   "2026-04-01T00:00:00Z",
					"source":       AccountModelProbeSnapshotSourceManualProbe,
					"probe_source": accountModelProbeSourceUpstream,
				},
			},
		},
		&Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		APIKey{
			ID:               103,
			Name:             "filtered-key",
			ModelDisplayMode: APIKeyModelDisplayModeAliasOnly,
			GroupBindings: []APIKeyGroupBinding{
				{
					GroupID:       10,
					Group:         &Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
					ModelPatterns: []string{"grok-*"},
				},
			},
		},
		nil,
	)

	result, err := service.Diagnose(context.Background(), 3, false)

	require.NoError(t, err)
	require.Equal(t, AccountModelDiagnosticsStatusFilteredEmpty, result.Status)
	require.Len(t, result.PublicModelsPreview, 1)
	require.Empty(t, result.GroupExposures[0].PublicModels)
}

func TestAccountModelDiagnosticsService_DiagnoseProbeFailedEmptyWithoutSavedModels(t *testing.T) {
	importSvc := NewAccountModelImportService(nil, nil, &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":"forbidden"}`,
	}, nil)
	service := newDiagnosticsServiceForTest(
		Account{
			ID:          4,
			Name:        "openai-apikey",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{10},
			Credentials: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://openai.example.test",
			},
		},
		&Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		APIKey{
			ID:               104,
			Name:             "public-key",
			ModelDisplayMode: APIKeyModelDisplayModeAliasOnly,
			GroupBindings: []APIKeyGroupBinding{
				{
					GroupID: 10,
					Group:   &Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
				},
			},
		},
		importSvc,
	)

	result, err := service.Diagnose(context.Background(), 4, false)

	require.NoError(t, err)
	require.Equal(t, AccountModelDiagnosticsStatusProbeFailed, result.Status)
	require.Empty(t, result.SavedModels)
	require.Empty(t, result.PublicModelsPreview)
}

func TestAccountModelDiagnosticsService_DiagnoseUsesAliasOnlyPreviewForExplicitMappings(t *testing.T) {
	service := newDiagnosticsServiceForTest(
		Account{
			ID:          5,
			Name:        "openai-apikey",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{10},
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"friendly-gpt": "gpt-4.1-mini",
				},
			},
			Extra: map[string]any{
				"model_probe_snapshot": map[string]any{
					"models":       []string{"gpt-4.1-mini"},
					"updated_at":   "2026-04-01T00:00:00Z",
					"source":       AccountModelProbeSnapshotSourceManualProbe,
					"probe_source": accountModelProbeSourceUpstream,
				},
			},
		},
		&Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		APIKey{
			ID:               105,
			Name:             "public-key",
			ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
			GroupBindings: []APIKeyGroupBinding{
				{
					GroupID: 10,
					Group:   &Group{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
				},
			},
		},
		nil,
	)

	result, err := service.Diagnose(context.Background(), 5, false)

	require.NoError(t, err)
	require.Len(t, result.PublicModelsPreview, 1)
	require.Equal(t, "friendly-gpt", result.PublicModelsPreview[0].PublicID)
	require.Equal(t, "friendly-gpt", result.PublicModelsPreview[0].DisplayName)
	require.Equal(t, "gpt-4.1-mini", result.PublicModelsPreview[0].SourceID)
	require.Len(t, result.GroupExposures, 1)
	require.Equal(t, "friendly-gpt", result.GroupExposures[0].PublicModels[0].PublicID)
}
