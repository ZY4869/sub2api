package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func newOpsGoogleBatchRuntimeRouter(handler *OpsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/runtime/google-batch", handler.GetGoogleBatchRuntimeMetrics)
	return r
}

func TestOpsGoogleBatchRuntimeHandler_GetSnapshot(t *testing.T) {
	h := NewOpsHandler(newRuntimeOpsService(t))
	r := newOpsGoogleBatchRuntimeRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runtime/google-batch", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("code=%d, want 0", resp.Code)
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal response data: %v", err)
	}
	var snapshot service.GoogleBatchRuntimeMetricsSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if snapshot.BatchCreateTotal != 0 {
		t.Fatalf("batch_create_total=%d, want 0", snapshot.BatchCreateTotal)
	}
	if snapshot.ListFanoutSamples != 0 {
		t.Fatalf("list_fanout_samples=%d, want 0", snapshot.ListFanoutSamples)
	}
}
