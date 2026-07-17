package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountBillingProbeEnvelope struct {
	Code int                       `json:"code"`
	Data AccountBillingProbeResult `json:"data"`
}

type batchAccountBillingProbeEnvelope struct {
	Code int                              `json:"code"`
	Data BatchAccountBillingProbeResponse `json:"data"`
}

func setupAccountBillingProbeRouter(adminSvc *stubAdminService, settingService *service.SettingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	h.SetSettingService(settingService)
	router.POST("/api/v1/admin/accounts/billing-probe", h.BatchProbeBilling)
	router.POST("/api/v1/admin/accounts/:id/billing-probe", h.ProbeBilling)
	return router
}

func TestAccountBillingProbeUnsupportedPlatform(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:       42,
		Name:     "OpenAI",
		Platform: service.PlatformOpenAI,
		Status:   service.StatusActive,
	}}
	router := setupAccountBillingProbeRouter(adminSvc, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/42/billing-probe", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var envelope accountBillingProbeEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, int64(42), envelope.Data.AccountID)
	require.False(t, envelope.Data.Supported)
	require.Equal(t, "unsupported", envelope.Data.Status)
}

func TestAccountBillingProbeDisabledBySettings(t *testing.T) {
	adminSvc := newStubAdminService()
	settingSvc := service.NewSettingService(&adminSettingRepoStub{values: map[string]string{
		service.SettingKeyUpstreamBillingProbeSettings: `{"enabled":false,"batch_concurrency":2,"timeout_seconds":20}`,
	}}, nil)
	router := setupAccountBillingProbeRouter(adminSvc, settingSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/42/billing-probe", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "UPSTREAM_BILLING_PROBE_DISABLED")
}

func TestBatchAccountBillingProbeDeduplicatesAndSummarizes(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "OpenAI", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 2, Name: "Claude", Platform: service.PlatformAnthropic, Status: service.StatusActive},
	}
	router := setupAccountBillingProbeRouter(adminSvc, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/billing-probe", bytes.NewReader([]byte(`{"account_ids":[1,1,2]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var envelope batchAccountBillingProbeEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, 2, envelope.Data.Total)
	require.Equal(t, 0, envelope.Data.Succeeded)
	require.Equal(t, 2, envelope.Data.Unsupported)
	require.Len(t, envelope.Data.Items, 2)
}
