package repository

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestOpsRepositoryBatchInsertSystemLogs_DriverErrSkipFallsBack(t *testing.T) {
	db, state := openSystemLogBatchTestDB(t, "skip")
	repoIface := NewOpsRepository(db)
	repo, ok := repoIface.(*opsRepository)
	if !ok {
		t.Fatalf("NewOpsRepository() type = %T, want *opsRepository", repoIface)
	}

	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	inserted, err := repo.BatchInsertSystemLogs(context.Background(), []*service.OpsInsertSystemLogInput{
		{
			CreatedAt:       now,
			Level:           "warn",
			Component:       "http.access",
			Message:         "boom",
			RequestID:       "req-1",
			ClientRequestID: "creq-1",
			Platform:        "openai",
			Model:           "gpt-5",
			ExtraJSON:       `{"host":"api.example.com"}`,
		},
	})
	if err != nil {
		t.Fatalf("BatchInsertSystemLogs() error = %v", err)
	}
	if inserted != 1 {
		t.Fatalf("BatchInsertSystemLogs() inserted = %d, want 1", inserted)
	}
	if got := atomic.LoadInt32(&state.copyExecCount); got != 1 {
		t.Fatalf("copy exec count = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&state.insertExecCount); got != 1 {
		t.Fatalf("fallback insert count = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&state.beginCount); got != 2 {
		t.Fatalf("begin count = %d, want 2", got)
	}
	if got := atomic.LoadInt32(&state.rollbackCount); got != 1 {
		t.Fatalf("rollback count = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&state.commitCount); got != 1 {
		t.Fatalf("commit count = %d, want 1", got)
	}

	state.mu.Lock()
	lastInsertQuery := state.lastInsertQuery
	lastInsertArgCount := state.lastInsertArgCount
	state.mu.Unlock()
	if !strings.Contains(strings.ToUpper(lastInsertQuery), "INSERT INTO OPS_SYSTEM_LOGS") {
		t.Fatalf("unexpected fallback query: %s", lastInsertQuery)
	}
	if lastInsertArgCount != 12 {
		t.Fatalf("fallback arg count = %d, want 12", lastInsertArgCount)
	}
}

func TestOpsRepositoryBatchInsertSystemLogs_RealErrorDoesNotFallback(t *testing.T) {
	db, state := openSystemLogBatchTestDB(t, "error")
	repoIface := NewOpsRepository(db)
	repo, ok := repoIface.(*opsRepository)
	if !ok {
		t.Fatalf("NewOpsRepository() type = %T, want *opsRepository", repoIface)
	}

	_, err := repo.BatchInsertSystemLogs(context.Background(), []*service.OpsInsertSystemLogInput{
		{
			CreatedAt: time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
			Level:     "warn",
			Component: "app",
			Message:   "boom",
		},
	})
	if err == nil {
		t.Fatalf("BatchInsertSystemLogs() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "copy fast-path failed") {
		t.Fatalf("BatchInsertSystemLogs() error = %v, want copy fast-path failure", err)
	}
	if got := atomic.LoadInt32(&state.copyExecCount); got != 1 {
		t.Fatalf("copy exec count = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&state.insertExecCount); got != 0 {
		t.Fatalf("fallback insert count = %d, want 0", got)
	}
	if got := atomic.LoadInt32(&state.beginCount); got != 1 {
		t.Fatalf("begin count = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&state.rollbackCount); got != 1 {
		t.Fatalf("rollback count = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&state.commitCount); got != 0 {
		t.Fatalf("commit count = %d, want 0", got)
	}
}
