package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerBackfillRepoStub struct {
	pages            [][]service.Account
	listErr          error
	updateErr        error
	updateExtraCalls []struct {
		id      int64
		updates map[string]any
	}
}

func (s *handlerBackfillRepoStub) ListWithFilters(
	_ context.Context,
	params pagination.PaginationParams,
	_, _, _, _ string,
	_ int64,
	_, _ string,
) ([]service.Account, *pagination.PaginationResult, error) {
	if s.listErr != nil {
		return nil, nil, s.listErr
	}
	page := params.Page
	if page < 1 || page > len(s.pages) {
		return nil, &pagination.PaginationResult{Page: page, PageSize: params.PageSize, Pages: len(s.pages)}, nil
	}
	return append([]service.Account(nil), s.pages[page-1]...),
		&pagination.PaginationResult{Page: page, PageSize: params.PageSize, Pages: len(s.pages)},
		nil
}

func (s *handlerBackfillRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updateExtraCalls = append(s.updateExtraCalls, struct {
		id      int64
		updates map[string]any
	}{id: id, updates: updates})
	return nil
}

func setupModelPolicyBackfillRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/model-policy/backfill", handler.BackfillModelPolicies)
	return router
}

func decodeModelPolicyBackfillResponse(t *testing.T, rec *httptest.ResponseRecorder) service.AccountModelPolicyBackfillResult {
	t.Helper()
	var payload struct {
		Data service.AccountModelPolicyBackfillResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload.Data
}

func TestAccountHandlerBackfillModelPolicies_SuccessfullyNormalizesLegacyMappingScope(t *testing.T) {
	repo := &handlerBackfillRepoStub{
		pages: [][]service.Account{
			{
				{
					ID:       501,
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"friendly-gpt": "gpt-4.1-mini",
						},
					},
					Extra: map[string]any{},
				},
			},
		},
	}
	adminSvc := newStubAdminService()
	adminSvc.backfillRepo = repo
	router := setupModelPolicyBackfillRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/model-policy/backfill", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	result := decodeModelPolicyBackfillResponse(t, rec)
	require.Equal(t, service.AccountModelPolicyBackfillResult{
		Scanned:           1,
		Updated:           1,
		ScopeNormalized:   1,
		SnapshotRefreshed: 1,
	}, result)
	require.Equal(t, accountModelPolicyBackfillDefaultPageSize, adminSvc.lastBackfillPageSize)
	require.Len(t, repo.updateExtraCalls, 1)

	scope, ok := repo.updateExtraCalls[0].updates["model_scope_v2"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, service.AccountModelPolicyModeMapping, scope["policy_mode"])

	entries, ok := scope["entries"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, entries, 1)
	require.Equal(t, "friendly-gpt", entries[0]["display_model_id"])
	require.Equal(t, "gpt-4.1-mini", entries[0]["target_model_id"])

	snapshot, ok := repo.updateExtraCalls[0].updates["model_probe_snapshot"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, snapshot["entries"])
}

func TestAccountHandlerBackfillModelPolicies_ReturnsEmptySummary(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.backfillRepo = &handlerBackfillRepoStub{}
	router := setupModelPolicyBackfillRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/model-policy/backfill", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.AccountModelPolicyBackfillResult{}, decodeModelPolicyBackfillResponse(t, rec))
}

func TestAccountHandlerBackfillModelPolicies_HandlesRepositoryFailure(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.backfillRepo = &handlerBackfillRepoStub{
		listErr: errors.New("list failed"),
	}
	router := setupModelPolicyBackfillRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/model-policy/backfill", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "\"message\":\"internal error\"")
}
