package servertiming

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"
	"time"
)

func TestSQLStmtExecAndQueryContextFallbackToLegacy(t *testing.T) {
	driverName, err := RegisterSQLDriver("legacy-fallback", legacyFallbackDriver{})
	if err != nil {
		t.Fatalf("RegisterSQLDriver() error = %v", err)
	}
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	}()

	ctx := WithCollector(context.Background(), New(time.Now()))
	stmt, err := db.PrepareContext(ctx, "SELECT legacy")
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			t.Fatalf("Stmt.Close() error = %v", err)
		}
	}()

	result, err := stmt.ExecContext(ctx, 1, "x")
	if err != nil {
		t.Fatalf("Stmt.ExecContext() legacy fallback error = %v", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("RowsAffected() = %d, %v; want 1, nil", affected, err)
	}

	rows, err := stmt.QueryContext(ctx, "y")
	if err != nil {
		t.Fatalf("Stmt.QueryContext() legacy fallback error = %v", err)
	}
	if !rows.Next() {
		t.Fatalf("expected one legacy row")
	}
	var got string
	if err := rows.Scan(&got); err != nil {
		t.Fatalf("Rows.Scan() error = %v", err)
	}
	if got != "legacy-row" {
		t.Fatalf("legacy row = %q, want legacy-row", got)
	}
	if rows.Next() {
		t.Fatalf("unexpected extra legacy row")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows.Err() = %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("Rows.Close() error = %v", err)
	}

	header := HeaderValue(ctx, time.Now())
	if !strings.Contains(header, `db;dur=`) {
		t.Fatalf("Server-Timing header missing db metric: %s", header)
	}
	if !strings.Contains(header, `desc="2"`) {
		t.Fatalf("Server-Timing header missing legacy stmt count: %s", header)
	}
}

type legacyFallbackDriver struct{}

func (legacyFallbackDriver) Open(string) (driver.Conn, error) {
	return legacyFallbackConn{}, nil
}

type legacyFallbackConn struct{}

func (legacyFallbackConn) Prepare(string) (driver.Stmt, error) {
	return legacyFallbackStmt{}, nil
}

func (legacyFallbackConn) Close() error {
	return nil
}

func (legacyFallbackConn) Begin() (driver.Tx, error) {
	return legacyFallbackTx{}, nil
}

type legacyFallbackStmt struct{}

func (legacyFallbackStmt) Close() error {
	return nil
}

func (legacyFallbackStmt) NumInput() int {
	return -1
}

func (legacyFallbackStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (legacyFallbackStmt) Query([]driver.Value) (driver.Rows, error) {
	return &legacyFallbackRows{}, nil
}

type legacyFallbackRows struct {
	returned bool
}

func (legacyFallbackRows) Columns() []string {
	return []string{"value"}
}

func (legacyFallbackRows) Close() error {
	return nil
}

func (r *legacyFallbackRows) Next(dest []driver.Value) error {
	if r.returned {
		return io.EOF
	}
	r.returned = true
	dest[0] = "legacy-row"
	return nil
}

type legacyFallbackTx struct{}

func (legacyFallbackTx) Commit() error {
	return nil
}

func (legacyFallbackTx) Rollback() error {
	return nil
}
