package repository

import (
	"context"
	"database/sql"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

type affiliateSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type affiliateSQLTxStarter interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func affiliateSQLExecutorFromContext(ctx context.Context, db *sql.DB) affiliateSQLExecutor {
	if db == nil {
		return nil
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		if exec, ok := any(tx.Client()).(affiliateSQLExecutor); ok {
			return exec
		}
		if exec, ok := any(tx).(affiliateSQLExecutor); ok {
			return exec
		}
	}
	return db
}

func beginAffiliateSQLTx(ctx context.Context, exec affiliateSQLExecutor) (affiliateSQLExecutor, func() error, func(), error) {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		if txExec, ok := any(tx.Client()).(affiliateSQLExecutor); ok {
			return txExec, func() error { return nil }, func() {}, nil
		}
	}
	if tx, ok := exec.(*sql.Tx); ok {
		return tx, func() error { return nil }, func() {}, nil
	}
	starter, ok := exec.(affiliateSQLTxStarter)
	if !ok {
		return nil, nil, nil, fmt.Errorf("sql executor does not support transactions")
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	return tx, tx.Commit, func() { _ = tx.Rollback() }, nil
}
