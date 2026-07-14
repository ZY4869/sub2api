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

func TestSQLDriverRecordsDBTiming(t *testing.T) {
	driverName, err := RegisterSQLDriver("fake-servertiming", fakeSQLDriver{})
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
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("PingContext() error = %v", err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE t SET v = 1"); err != nil {
		t.Fatalf("ExecContext() error = %v", err)
	}
	rows, err := db.QueryContext(ctx, "SELECT 1")
	if err != nil {
		t.Fatalf("QueryContext() error = %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("Rows.Close() error = %v", err)
	}
	stmt, err := db.PrepareContext(ctx, "SELECT 1")
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}
	if _, err := stmt.QueryContext(ctx); err != nil {
		t.Fatalf("Stmt.QueryContext() error = %v", err)
	}
	if err := stmt.Close(); err != nil {
		t.Fatalf("Stmt.Close() error = %v", err)
	}

	header := HeaderValue(ctx, time.Now())
	if !strings.Contains(header, `db;dur=`) {
		t.Fatalf("Server-Timing header missing db metric: %s", header)
	}
	if !strings.Contains(header, `desc="4"`) {
		t.Fatalf("Server-Timing header missing db count: %s", header)
	}
}

type fakeSQLDriver struct{}

func (fakeSQLDriver) Open(string) (driver.Conn, error) {
	return fakeSQLConn{}, nil
}

type fakeSQLConn struct{}

func (fakeSQLConn) Prepare(string) (driver.Stmt, error) {
	return fakeSQLStmt{}, nil
}

func (fakeSQLConn) Close() error {
	return nil
}

func (fakeSQLConn) Begin() (driver.Tx, error) {
	return fakeSQLTx{}, nil
}

func (fakeSQLConn) Ping(context.Context) error {
	return nil
}

func (fakeSQLConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (fakeSQLConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return fakeSQLRows{}, nil
}

func (fakeSQLConn) PrepareContext(context.Context, string) (driver.Stmt, error) {
	return fakeSQLStmt{}, nil
}

type fakeSQLStmt struct{}

func (fakeSQLStmt) Close() error {
	return nil
}

func (fakeSQLStmt) NumInput() int {
	return -1
}

func (fakeSQLStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (fakeSQLStmt) Query([]driver.Value) (driver.Rows, error) {
	return fakeSQLRows{}, nil
}

func (fakeSQLStmt) QueryContext(context.Context, []driver.NamedValue) (driver.Rows, error) {
	return fakeSQLRows{}, nil
}

type fakeSQLRows struct{}

func (fakeSQLRows) Columns() []string {
	return []string{"value"}
}

func (fakeSQLRows) Close() error {
	return nil
}

func (fakeSQLRows) Next([]driver.Value) error {
	return io.EOF
}

type fakeSQLTx struct{}

func (fakeSQLTx) Commit() error {
	return nil
}

func (fakeSQLTx) Rollback() error {
	return nil
}
