package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

const systemLogBatchTestDriverName = "ops-system-logs-batch-test"

var systemLogBatchTestDriverOnce sync.Once
var systemLogBatchTestStates sync.Map

func openSystemLogBatchTestDB(t *testing.T, mode string) (*sql.DB, *systemLogBatchState) {
	t.Helper()

	systemLogBatchTestDriverOnce.Do(func() {
		sql.Register(systemLogBatchTestDriverName, systemLogBatchDriver{})
	})

	dsn := t.Name() + ":" + mode
	state := &systemLogBatchState{mode: mode}
	systemLogBatchTestStates.Store(dsn, state)
	t.Cleanup(func() {
		systemLogBatchTestStates.Delete(dsn)
	})

	db, err := sql.Open(systemLogBatchTestDriverName, dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	})
	return db, state
}

type systemLogBatchDriver struct{}

func (systemLogBatchDriver) Open(name string) (driver.Conn, error) {
	v, ok := systemLogBatchTestStates.Load(name)
	if !ok {
		return nil, fmt.Errorf("missing test state for dsn %q", name)
	}
	state, ok := v.(*systemLogBatchState)
	if !ok {
		return nil, fmt.Errorf("invalid test state type %T for dsn %q", v, name)
	}
	return &systemLogBatchConn{state: state}, nil
}

type systemLogBatchState struct {
	mode string

	beginCount      int32
	rollbackCount   int32
	commitCount     int32
	copyExecCount   int32
	insertExecCount int32

	mu                 sync.Mutex
	lastCopyQuery      string
	lastCopyArgCount   int
	lastInsertQuery    string
	lastInsertArgCount int
}

type systemLogBatchConn struct {
	state *systemLogBatchState

	mu   sync.Mutex
	inTx bool
}

func (c *systemLogBatchConn) Prepare(query string) (driver.Stmt, error) {
	return c.prepare(query)
}

func (c *systemLogBatchConn) Close() error {
	return nil
}

func (c *systemLogBatchConn) Begin() (driver.Tx, error) {
	c.mu.Lock()
	c.inTx = true
	c.mu.Unlock()
	atomic.AddInt32(&c.state.beginCount, 1)
	return &systemLogBatchTx{conn: c}, nil
}

func (c *systemLogBatchConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	trimmed := strings.TrimSpace(query)
	if !strings.HasPrefix(strings.ToUpper(trimmed), "INSERT INTO OPS_SYSTEM_LOGS") {
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}
	if !c.isInTx() {
		return nil, errors.New("fallback insert outside transaction")
	}
	atomic.AddInt32(&c.state.insertExecCount, 1)
	c.state.mu.Lock()
	c.state.lastInsertQuery = trimmed
	c.state.lastInsertArgCount = len(args)
	c.state.mu.Unlock()
	return driver.RowsAffected(1), nil
}

func (c *systemLogBatchConn) prepare(query string) (driver.Stmt, error) {
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "COPY ") || !strings.Contains(upper, "OPS_SYSTEM_LOGS") {
		return nil, fmt.Errorf("unexpected prepare query: %s", query)
	}
	c.state.mu.Lock()
	c.state.lastCopyQuery = trimmed
	c.state.lastCopyArgCount = 0
	c.state.mu.Unlock()
	return &systemLogBatchCopyStmt{conn: c, query: trimmed}, nil
}

func (c *systemLogBatchConn) isInTx() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inTx
}

func (c *systemLogBatchConn) setInTx(v bool) {
	c.mu.Lock()
	c.inTx = v
	c.mu.Unlock()
}

type systemLogBatchTx struct {
	conn *systemLogBatchConn
}

func (tx *systemLogBatchTx) Commit() error {
	tx.conn.setInTx(false)
	atomic.AddInt32(&tx.conn.state.commitCount, 1)
	return nil
}

func (tx *systemLogBatchTx) Rollback() error {
	tx.conn.setInTx(false)
	atomic.AddInt32(&tx.conn.state.rollbackCount, 1)
	return nil
}

type systemLogBatchCopyStmt struct {
	conn  *systemLogBatchConn
	query string
}

func (s *systemLogBatchCopyStmt) Close() error {
	return nil
}

func (s *systemLogBatchCopyStmt) NumInput() int {
	return -1
}

func (s *systemLogBatchCopyStmt) ExecContext(_ context.Context, args []driver.NamedValue) (driver.Result, error) {
	if !s.conn.isInTx() {
		return nil, errors.New("copy fast-path outside transaction")
	}
	s.conn.state.mu.Lock()
	s.conn.state.lastCopyQuery = s.query
	s.conn.state.lastCopyArgCount = len(args)
	s.conn.state.mu.Unlock()
	atomic.AddInt32(&s.conn.state.copyExecCount, 1)
	switch s.conn.state.mode {
	case "skip":
		return nil, driver.ErrSkip
	case "error":
		return nil, errors.New("copy fast-path failed")
	default:
		return driver.RowsAffected(1), nil
	}
}

func (s *systemLogBatchCopyStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, driver.ErrSkip
}

func (s *systemLogBatchCopyStmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, driver.ErrSkip
}
