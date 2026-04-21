package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type metaSettingRepoStub struct {
	values map[string]string
}

func (s *metaSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (s *metaSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *metaSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *metaSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}

func (s *metaSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *metaSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *metaSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestMetaHandler_ModelCatalogHonorsETag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &metaSettingRepoStub{values: map[string]string{}}
	repo.values[service.SettingKeyPublicModelCatalogPublishedSnapshot] = mustMetaJSON(t, buildMetaPublishedSnapshot("W/\"published-etag\""))

	modelCatalogService := service.NewModelCatalogService(
		repo,
		nil,
		service.NewBillingService(&config.Config{}, nil),
		nil,
		&config.Config{},
	)

	metaHandler := NewMetaHandler(modelCatalogService)
	router := gin.New()
	router.GET("/api/v1/meta/model-catalog", metaHandler.ModelCatalog)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	require.Equal(t, http.StatusOK, firstRec.Code)
	require.Equal(t, "If-None-Match", firstRec.Header().Get("Vary"))
	etag := firstRec.Header().Get("ETag")
	require.NotEmpty(t, etag)
	require.Contains(t, firstRec.Body.String(), "gpt-5.4")

	secondReq := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
	secondReq.Header.Set("If-None-Match", etag)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	require.Equal(t, http.StatusNotModified, secondRec.Code)
	require.Equal(t, etag, secondRec.Header().Get("ETag"))
	require.Empty(t, strings.TrimSpace(secondRec.Body.String()))
}

func TestMetaHandler_ModelCatalogRequiresAuthWhenPublicCatalogDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &metaSettingRepoStub{values: map[string]string{
		service.SettingKeyPublicModelCatalogEnabled: "false",
	}}
	settingService := service.NewSettingService(repo, &config.Config{})
	metaHandler := NewMetaHandler(nil)
	metaHandler.SetSettingService(settingService)
	metaHandler.SetAuthResolverForTest(func(*gin.Context) bool { return false })

	router := gin.New()
	router.GET("/api/v1/meta/model-catalog", metaHandler.ModelCatalog)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMetaHandler_ModelCatalogAuthMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		publicEnabled bool
		authenticated bool
		wantStatus    int
	}{
		{name: "guest allowed when public catalog enabled", publicEnabled: true, authenticated: false, wantStatus: http.StatusOK},
		{name: "authenticated allowed when public catalog enabled", publicEnabled: true, authenticated: true, wantStatus: http.StatusOK},
		{name: "guest rejected when public catalog disabled", publicEnabled: false, authenticated: false, wantStatus: http.StatusUnauthorized},
		{name: "authenticated allowed when public catalog disabled", publicEnabled: false, authenticated: true, wantStatus: http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &metaSettingRepoStub{values: map[string]string{
				service.SettingKeyPublicModelCatalogEnabled:           strconv.FormatBool(tc.publicEnabled),
				service.SettingKeyPublicModelCatalogPublishedSnapshot: mustMetaJSON(t, buildMetaPublishedSnapshot("W/\"matrix-etag\"")),
			}}

			settingService := service.NewSettingService(repo, &config.Config{})
			modelCatalogService := service.NewModelCatalogService(
				repo,
				nil,
				service.NewBillingService(&config.Config{}, nil),
				nil,
				&config.Config{},
			)
			metaHandler := NewMetaHandler(modelCatalogService)
			metaHandler.SetSettingService(settingService)
			metaHandler.SetAuthResolverForTest(func(*gin.Context) bool { return tc.authenticated })

			router := gin.New()
			router.GET("/api/v1/meta/model-catalog", metaHandler.ModelCatalog)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantStatus == http.StatusOK {
				require.Contains(t, rec.Body.String(), "gpt-5.4")
			}
		})
	}
}

func TestMetaHandler_ModelCatalogAllowsAuthenticatedRequestWhenPublicCatalogDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &metaSettingRepoStub{values: map[string]string{
		service.SettingKeyPublicModelCatalogEnabled:           "false",
		service.SettingKeyPublicModelCatalogPublishedSnapshot: mustMetaJSON(t, buildMetaPublishedSnapshot("W/\"auth-etag\"")),
	}}
	settingService := service.NewSettingService(repo, &config.Config{})
	modelCatalogService := service.NewModelCatalogService(
		repo,
		nil,
		service.NewBillingService(&config.Config{}, nil),
		nil,
		&config.Config{},
	)
	metaHandler := NewMetaHandler(modelCatalogService)
	metaHandler.SetSettingService(settingService)
	metaHandler.SetAuthResolverForTest(func(*gin.Context) bool { return true })

	router := gin.New()
	router.GET("/api/v1/meta/model-catalog", metaHandler.ModelCatalog)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "gpt-5.4")
}

func TestMetaHandler_ModelCatalogDetailReturnsModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &metaSettingRepoStub{values: map[string]string{}}
	repo.values[service.SettingKeyPublicModelCatalogPublishedSnapshot] = mustMetaJSON(t, buildMetaPublishedSnapshot("W/\"detail-etag\""))

	modelCatalogService := service.NewModelCatalogService(
		repo,
		nil,
		service.NewBillingService(&config.Config{}, nil),
		nil,
		&config.Config{},
	)

	metaHandler := NewMetaHandler(modelCatalogService)
	router := gin.New()
	router.GET("/api/v1/meta/model-catalog/:model", metaHandler.ModelCatalogDetail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog/gpt-5.4", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"model\":\"gpt-5.4\"")
	require.Contains(t, rec.Body.String(), "\"item\"")
	require.Contains(t, rec.Body.String(), "\"example_source\":\"docs_section\"")
}

func TestMetaHandler_ModelCatalogReturnsEmptySnapshotWhenNotPublished(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &metaSettingRepoStub{values: map[string]string{}}
	modelCatalogService := service.NewModelCatalogService(
		repo,
		nil,
		service.NewBillingService(&config.Config{}, nil),
		nil,
		&config.Config{},
	)

	metaHandler := NewMetaHandler(modelCatalogService)
	router := gin.New()
	router.GET("/api/v1/meta/model-catalog", metaHandler.ModelCatalog)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"items\":[]")
	require.Contains(t, rec.Body.String(), "\"page_size\":10")
}

func TestMetaHandler_ModelCatalogDetailReturnsNotFoundWhenNotPublished(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &metaSettingRepoStub{values: map[string]string{}}
	modelCatalogService := service.NewModelCatalogService(
		repo,
		nil,
		service.NewBillingService(&config.Config{}, nil),
		nil,
		&config.Config{},
	)

	metaHandler := NewMetaHandler(modelCatalogService)
	router := gin.New()
	router.GET("/api/v1/meta/model-catalog/:model", metaHandler.ModelCatalogDetail)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog/gpt-5.4", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func mustMetaJSON(t *testing.T, value any) string {
	t.Helper()

	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return string(payload)
}

func buildMetaPublishedSnapshot(etag string) service.PublicModelCatalogPublishedSnapshot {
	return service.PublicModelCatalogPublishedSnapshot{
		Snapshot: service.PublicModelCatalogSnapshot{
			ETag:      etag,
			UpdatedAt: "2026-04-20T10:00:00Z",
			PageSize:  10,
			Items: []service.PublicModelCatalogItem{
				{
					Model:            "gpt-5.4",
					DisplayName:      "GPT-5.4",
					Provider:         service.PlatformOpenAI,
					ProviderIconKey:  service.PlatformOpenAI,
					RequestProtocols: []string{service.PlatformOpenAI},
					Mode:             "chat",
					Currency:         "USD",
					PriceDisplay: service.PublicModelCatalogPriceDisplay{
						Primary: []service.PublicModelCatalogPriceEntry{
							{ID: "input_price", Unit: service.BillingUnitInputToken, Value: 1e-6},
							{ID: "output_price", Unit: service.BillingUnitOutputToken, Value: 2e-6},
						},
					},
					MultiplierSummary: service.PublicModelCatalogMultiplierSummary{
						Enabled: false,
						Kind:    "disabled",
					},
				},
			},
		},
		Details: map[string]service.PublicModelCatalogDetail{
			"gpt-5.4": {
				Item: service.PublicModelCatalogItem{
					Model:            "gpt-5.4",
					DisplayName:      "GPT-5.4",
					Provider:         service.PlatformOpenAI,
					ProviderIconKey:  service.PlatformOpenAI,
					RequestProtocols: []string{service.PlatformOpenAI},
					Mode:             "chat",
					Currency:         "USD",
					PriceDisplay: service.PublicModelCatalogPriceDisplay{
						Primary: []service.PublicModelCatalogPriceEntry{
							{ID: "input_price", Unit: service.BillingUnitInputToken, Value: 1e-6},
							{ID: "output_price", Unit: service.BillingUnitOutputToken, Value: 2e-6},
						},
					},
					MultiplierSummary: service.PublicModelCatalogMultiplierSummary{
						Enabled: false,
						Kind:    "disabled",
					},
				},
				ExampleSource:   "docs_section",
				ExampleProtocol: service.PlatformOpenAI,
				ExamplePageID:   "common",
				ExampleMarkdown: "```bash\ncurl https://example.com/v1/responses\n```",
			},
		},
	}
}
