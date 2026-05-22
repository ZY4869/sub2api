package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) service.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	if fn == nil {
		return nil
	}
	if r == nil || r.db == nil {
		return fn(ctx)
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, paymentSQLTxContextKey{}, tx)
	defer func() { _ = tx.Rollback() }()
	if err := fn(txCtx); err != nil {
		return err
	}
	return tx.Commit()
}
