package servertiming

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
)

var registeredDrivers sync.Map

func RegisterSQLDriver(name string, wrapped driver.Driver) (string, error) {
	if wrapped == nil {
		return "", errors.New("wrapped sql driver is nil")
	}
	driverName := "servertiming-" + name
	if _, loaded := registeredDrivers.LoadOrStore(driverName, struct{}{}); !loaded {
		sql.Register(driverName, sqlDriver{wrapped: wrapped})
	}
	return driverName, nil
}

type sqlDriver struct {
	wrapped driver.Driver
}

func (d sqlDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.wrapped.Open(name)
	if err != nil {
		return nil, err
	}
	return sqlConn{Conn: conn}, nil
}

type sqlConn struct {
	driver.Conn
}

func (c sqlConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if pc, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := pc.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		return sqlStmt{Stmt: stmt}, nil
	}
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return sqlStmt{Stmt: stmt}, nil
}

func (c sqlConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if bt, ok := c.Conn.(driver.ConnBeginTx); ok {
		done := Observe(ctx, "db")
		tx, err := bt.BeginTx(ctx, opts)
		done()
		return tx, err
	}
	if opts.Isolation != driver.IsolationLevel(0) || opts.ReadOnly {
		return nil, driver.ErrSkip
	}
	done := Observe(ctx, "db")
	tx, err := c.Conn.Begin()
	done()
	return tx, err
}

func (c sqlConn) Ping(ctx context.Context) error {
	if p, ok := c.Conn.(driver.Pinger); ok {
		done := Observe(ctx, "db")
		err := p.Ping(ctx)
		done()
		return err
	}
	return driver.ErrSkip
}

func (c sqlConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if execer, ok := c.Conn.(driver.ExecerContext); ok {
		done := Observe(ctx, "db")
		result, err := execer.ExecContext(ctx, query, args)
		done()
		return result, err
	}
	return nil, driver.ErrSkip
}

func (c sqlConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if queryer, ok := c.Conn.(driver.QueryerContext); ok {
		done := Observe(ctx, "db")
		rows, err := queryer.QueryContext(ctx, query, args)
		done()
		return rows, err
	}
	return nil, driver.ErrSkip
}

func (c sqlConn) CheckNamedValue(value *driver.NamedValue) error {
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return driver.ErrSkip
}

func (c sqlConn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.Conn.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}
	return nil
}

func (c sqlConn) IsValid() bool {
	if validator, ok := c.Conn.(driver.Validator); ok {
		return validator.IsValid()
	}
	return true
}

func (c sqlConn) Raw(fn func(driverConn any) error) error {
	if raw, ok := c.Conn.(interface {
		Raw(func(driverConn any) error) error
	}); ok {
		return raw.Raw(fn)
	}
	return fn(c.Conn)
}

type sqlStmt struct {
	driver.Stmt
}

func (s sqlStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if execer, ok := s.Stmt.(driver.StmtExecContext); ok {
		done := Observe(ctx, "db")
		result, err := execer.ExecContext(ctx, args)
		done()
		return result, err
	}
	return nil, driver.ErrSkip
}

func (s sqlStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if queryer, ok := s.Stmt.(driver.StmtQueryContext); ok {
		done := Observe(ctx, "db")
		rows, err := queryer.QueryContext(ctx, args)
		done()
		return rows, err
	}
	return nil, driver.ErrSkip
}

func (s sqlStmt) CheckNamedValue(value *driver.NamedValue) error {
	if checker, ok := s.Stmt.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return driver.ErrSkip
}

func (d sqlDriver) String() string {
	return fmt.Sprintf("%T", d.wrapped)
}
