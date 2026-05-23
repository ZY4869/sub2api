package admin

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCreateAndRedeemHandler creates a RedeemHandler with a non-nil (but minimal)
// RedeemService so that CreateAndRedeem's nil guard passes and we can test the
// parameter-validation layer that runs before any service call.
func newCreateAndRedeemHandler() *RedeemHandler {
	return &RedeemHandler{
		adminService:  newStubAdminService(),
		redeemService: &service.RedeemService{}, // non-nil to pass nil guard
	}
}

// postCreateAndRedeemValidation calls CreateAndRedeem and returns the response
// status code. For cases that pass validation and proceed into the service layer,
// a panic may occur (because RedeemService internals are nil); this is expected
// and treated as "validation passed" (returns 0 to indicate panic).
func postCreateAndRedeemValidation(t *testing.T, handler *RedeemHandler, body any) (code int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBytes, err := json.Marshal(body)
	require.NoError(t, err)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes/create-and-redeem", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	defer func() {
		if r := recover(); r != nil {
			// Panic means we passed validation and entered service layer (expected for minimal stub).
			code = 0
		}
	}()
	handler.CreateAndRedeem(c)
	return w.Code
}

func TestCreateAndRedeem_TypeDefaultsToBalance(t *testing.T) {
	// 不传 type 字段时应默认 balance，不触发 subscription 校验。
	// 验证通过后进入 service 层会 panic（返回 0），说明默认值生效。
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-default",
		"value":   10.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"omitting type should default to balance and pass validation")
}

func TestCreateAndRedeem_SubscriptionRequiresGroupID(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-no-group",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"validity_days": 30,
		// group_id 缺失
	})

	assert.Equal(t, http.StatusBadRequest, code)
}

func TestCreateAndRedeem_SubscriptionRejectsZeroValidityDays(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-bad-days-zero",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"group_id":      groupID,
		"validity_days": 0,
	})

	assert.Equal(t, http.StatusBadRequest, code)
}

func TestCreateAndRedeem_SubscriptionAllowsNegativeValidityDays(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-negative-days",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"group_id":      groupID,
		"validity_days": -7,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"negative subscription validity_days should pass validation for shortening flows")
}

func TestCreateAndRedeem_SubscriptionValidParamsPassValidation(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-valid",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"group_id":      groupID,
		"validity_days": 31,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"valid subscription params should pass validation")
}

func TestCreateAndRedeem_BalanceIgnoresSubscriptionFields(t *testing.T) {
	h := newCreateAndRedeemHandler()
	// balance 类型不传 group_id 和 validity_days，不应报 400
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-no-extras",
		"type":    "balance",
		"value":   50.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"balance type should not require group_id or validity_days")
}

func TestRedeemExportIncludesExpiresAtColumn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	expiresAt := time.Date(2026, 6, 1, 8, 30, 0, 0, time.UTC)
	adminSvc.redeems = []service.RedeemCode{
		{
			ID:        1,
			Code:      "EXP-1",
			Type:      service.RedeemTypeBalance,
			Value:     10,
			Status:    service.StatusUnused,
			CreatedAt: time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC),
			ExpiresAt: &expiresAt,
		},
		{
			ID:        2,
			Code:      "LEGACY-NO-EXP",
			Type:      service.RedeemTypeBalance,
			Value:     20,
			Status:    service.StatusUnused,
			CreatedAt: time.Date(2026, 5, 22, 2, 3, 4, 0, time.UTC),
		},
	}
	handler := NewRedeemHandler(adminSvc, nil)
	router := gin.New()
	router.GET("/api/v1/admin/redeem-codes/export", handler.Export)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/export", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := strings.TrimPrefix(rec.Body.String(), "\uFEFF")
	rows, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, "expires_at", rows[0][9])
	require.Equal(t, "2026-06-01 08:30:00", rows[1][9])
	require.Equal(t, "", rows[2][9], "legacy codes without expires_at must export as blank")
}
