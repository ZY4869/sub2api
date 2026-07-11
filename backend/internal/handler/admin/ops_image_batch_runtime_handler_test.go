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

func newOpsImageBatchRuntimeRouter(handler *OpsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/runtime/image-batch", handler.GetImageBatchRuntimeMetrics)
	return r
}

func TestOpsImageBatchRuntimeHandler_GetSnapshot(t *testing.T) {
	h := NewOpsHandler(newRuntimeOpsService(t))
	r := newOpsImageBatchRuntimeRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runtime/image-batch", nil)
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
	var snapshot service.ImageBatchRuntimeMetricsSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if snapshot.SubmittedJobs != 0 {
		t.Fatalf("submitted_jobs=%d, want 0", snapshot.SubmittedJobs)
	}
}
