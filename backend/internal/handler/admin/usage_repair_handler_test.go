package admin

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type repairRepoStub struct {
	created    []*service.UsageRepairTask
	listTasks  []service.UsageRepairTask
	listResult *pagination.PaginationResult
	statusByID map[int64]string
}

func (s *repairRepoStub) CreateTask(ctx context.Context, task *service.UsageRepairTask) error {
	if task == nil {
		return nil
	}
	if task.ID == 0 {
		task.ID = int64(len(s.created) + 1)
	}
	now := time.Now().UTC()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	clone := *task
	s.created = append(s.created, &clone)
	if s.statusByID == nil {
		s.statusByID = map[int64]string{}
	}
	s.statusByID[task.ID] = task.Status
	return nil
}

func (s *repairRepoStub) ListTasks(ctx context.Context, params pagination.PaginationParams) ([]service.UsageRepairTask, *pagination.PaginationResult, error) {
	if s.listResult == nil {
		s.listResult = &pagination.PaginationResult{Total: int64(len(s.listTasks)), Page: params.Page, PageSize: params.PageSize}
	}
	return s.listTasks, s.listResult, nil
}

func (s *repairRepoStub) ClaimNextPendingTask(ctx context.Context, staleRunningAfterSeconds int64) (*service.UsageRepairTask, error) {
	return nil, nil
}

func (s *repairRepoStub) GetTaskStatus(ctx context.Context, taskID int64) (string, error) {
	if s.statusByID == nil {
		return "", sql.ErrNoRows
	}
	status, ok := s.statusByID[taskID]
	if !ok {
		return "", sql.ErrNoRows
	}
	return status, nil
}

func (s *repairRepoStub) UpdateTaskProgress(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error {
	return nil
}

func (s *repairRepoStub) CancelTask(ctx context.Context, taskID int64, canceledBy int64) (bool, error) {
	if s.statusByID == nil {
		s.statusByID = map[int64]string{}
	}
	status := s.statusByID[taskID]
	if status != service.UsageRepairStatusPending && status != service.UsageRepairStatusRunning {
		return false, nil
	}
	s.statusByID[taskID] = service.UsageRepairStatusCanceled
	return true, nil
}

func (s *repairRepoStub) MarkTaskSucceeded(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error {
	return nil
}

func (s *repairRepoStub) MarkTaskFailed(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64, errorMsg string) error {
	return nil
}

func (s *repairRepoStub) ListClaudeRequestMetadataCandidates(ctx context.Context, since time.Time, afterID int64, limit int) ([]service.ClaudeUsageRepairCandidate, error) {
	return []service.ClaudeUsageRepairCandidate{}, nil
}

func (s *repairRepoStub) ApplyClaudeRequestMetadataPatch(ctx context.Context, usageID int64, patch service.UsageRepairTaskPatch) (bool, error) {
	return false, nil
}

var _ service.UsageRepairRepository = (*repairRepoStub)(nil)

func setupRepairRouter(repairService *service.UsageRepairService, userID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if userID > 0 {
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID})
			c.Next()
		})
	}
	handler := NewUsageHandler(nil, nil, nil, nil, repairService)
	router.POST("/api/v1/admin/usage/repair-tasks", handler.CreateRepairTask)
	router.GET("/api/v1/admin/usage/repair-tasks", handler.ListRepairTasks)
	router.POST("/api/v1/admin/usage/repair-tasks/:id/cancel", handler.CancelRepairTask)
	return router
}

func TestUsageHandlerCreateRepairTaskSuccess(t *testing.T) {
	repo := &repairRepoStub{}
	svc := service.NewUsageRepairService(repo, nil)
	router := setupRepairRouter(svc, 88)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/usage/repair-tasks", bytes.NewBufferString(`{"kind":"claude_request_metadata","days":7}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.created, 1)
	require.Equal(t, int64(88), repo.created[0].CreatedBy)
	require.Equal(t, service.UsageRepairKindClaudeRequestMetadata, repo.created[0].Kind)
	require.Equal(t, 7, repo.created[0].Days)
}

func TestUsageHandlerCreateRepairTaskRejectsRangeOver30(t *testing.T) {
	repo := &repairRepoStub{}
	svc := service.NewUsageRepairService(repo, nil)
	router := setupRepairRouter(svc, 88)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/usage/repair-tasks", bytes.NewBufferString(`{"kind":"claude_request_metadata","days":31}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, repo.created)
}

func TestUsageHandlerListRepairTasksSuccess(t *testing.T) {
	repo := &repairRepoStub{
		listTasks: []service.UsageRepairTask{
			{ID: 5, Kind: service.UsageRepairKindClaudeRequestMetadata, Days: 30, Status: service.UsageRepairStatusSucceeded, CreatedBy: 1},
		},
		listResult: &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 20, Pages: 1},
	}
	svc := service.NewUsageRepairService(repo, nil)
	router := setupRepairRouter(svc, 88)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/usage/repair-tasks", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []dto.UsageRepairTask `json:"items"`
			Total int64                 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, int64(5), resp.Data.Items[0].ID)
	require.Equal(t, int64(1), resp.Data.Total)
}

func TestUsageHandlerCancelRepairTaskSuccess(t *testing.T) {
	repo := &repairRepoStub{statusByID: map[int64]string{3: service.UsageRepairStatusPending}}
	svc := service.NewUsageRepairService(repo, nil)
	router := setupRepairRouter(svc, 88)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/usage/repair-tasks/3/cancel", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.UsageRepairStatusCanceled, repo.statusByID[3])
}
