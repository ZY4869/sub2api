package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
	"github.com/gin-gonic/gin"
)

func TestServerTimingDisabledDoesNotAttachCollectorOrHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ServerTiming(false))
	r.GET("/", func(c *gin.Context) {
		if servertiming.Active(c.Request.Context()) {
			t.Fatalf("collector should not be active when disabled")
		}
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := w.Header().Get(servertiming.HeaderName); got != "" {
		t.Fatalf("unexpected Server-Timing header: %s", got)
	}
}

func TestServerTimingEnabledAddsHeaderBeforeBodyWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ServerTiming(true))
	r.GET("/", func(c *gin.Context) {
		if !servertiming.Active(c.Request.Context()) {
			t.Fatalf("collector should be active")
		}
		servertiming.Record(c.Request.Context(), "db", time.Millisecond)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	got := w.Header().Get(servertiming.HeaderName)
	if !strings.Contains(got, "app;dur=") || !strings.Contains(got, "db;dur=1.0") {
		t.Fatalf("unexpected Server-Timing header: %s", got)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("body should be preserved, got %q", w.Body.String())
	}
}
