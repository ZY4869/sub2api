package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type batchConcurrencyAdminServiceStub struct {
	service.AdminService
	users       []service.User
	updateCalls int
	failUserIDs map[int64]error
}

func (s *batchConcurrencyAdminServiceStub) ListUsers(ctx context.Context, page, pageSize int, filters service.UserListFilters) ([]service.User, int64, error) {
	return append([]service.User(nil), s.users...), int64(len(s.users)), nil
}

func (s *batchConcurrencyAdminServiceStub) UpdateUser(ctx context.Context, id int64, input *service.UpdateUserInput) (*service.User, error) {
	s.updateCalls++
	if err := s.failUserIDs[id]; err != nil {
		return nil, err
	}
	for i := range s.users {
		if s.users[i].ID != id {
			continue
		}
		if input != nil && input.Concurrency != nil {
			s.users[i].Concurrency = *input.Concurrency
		}
		s.users[i].UpdatedAt = time.Now().UTC()
		return &s.users[i], nil
	}
	return &service.User{ID: id, Email: "updated@example.com", Status: service.StatusActive}, nil
}

func newBatchConcurrencyRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewUserHandler(adminSvc, nil)
	router.POST("/api/v1/admin/users/batch-concurrency", func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 99})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
		handler.BatchUpdateConcurrency(c)
	})
	return router
}

func TestBatchUpdateConcurrency_ZeroMatchReturnsEmptyResult(t *testing.T) {
	adminSvc := &batchConcurrencyAdminServiceStub{users: []service.User{}}
	router := newBatchConcurrencyRouter(adminSvc)

	body := bytes.NewBufferString(`{"concurrency":9,"search":"nobody@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/batch-concurrency", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "batch-zero-match")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{
		"code": 0,
		"message": "success",
		"data": {
			"matched": 0,
			"success_count": 0,
			"failed_count": 0,
			"concurrency": 9,
			"results": []
		}
	}`, rec.Body.String())
	require.Equal(t, 0, adminSvc.updateCalls)
}

func TestBatchUpdateConcurrency_ReplaysDuplicateIdempotencyKey(t *testing.T) {
	repo := newMemoryIdempotencyRepoStub()
	cfg := service.DefaultIdempotencyConfig()
	cfg.ProcessingTimeout = 2 * time.Second
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repo, cfg))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	adminSvc := &batchConcurrencyAdminServiceStub{
		users: []service.User{
			{ID: 1, Email: "alice@example.com", Status: service.StatusActive},
		},
	}
	router := newBatchConcurrencyRouter(adminSvc)

	requestBody := []byte(`{"concurrency":7,"search":"alice"}`)
	doRequest := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/batch-concurrency", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "batch-replay-1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := doRequest()
	second := doRequest()

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "true", second.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, 1, adminSvc.updateCalls)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &payload))
	require.Equal(t, float64(0), payload["code"])
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), data["matched"])
	require.Equal(t, float64(1), data["success_count"])
	require.Equal(t, float64(0), data["failed_count"])
}
