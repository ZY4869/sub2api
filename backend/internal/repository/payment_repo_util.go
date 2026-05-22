package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func paymentExec(ctx context.Context, db *sql.DB) sqlExecutor {
	if tx, ok := ctx.Value(paymentSQLTxContextKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return db
}

type paymentSQLTxContextKey struct{}

func nullEmpty(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func jsonOrEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func populatePaymentRefundableFields(order *service.PaymentOrder) {
	if order == nil {
		return
	}
	if order.RefundedAmountMinor < 0 {
		order.RefundedAmountMinor = 0
	}
	remaining := order.AmountMinor - order.RefundedAmountMinor
	if remaining < 0 {
		remaining = 0
	}
	order.RefundableAmountMinor = remaining
}
