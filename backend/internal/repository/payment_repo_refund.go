package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *paymentRepository) CreateRefund(ctx context.Context, refund *service.PaymentRefund) error {
	if refund == nil {
		return nil
	}
	err := scanSingleRow(ctx, paymentExec(ctx, r.db), `
		INSERT INTO payment_refunds (
			refund_no, order_no, amount_minor, currency, reason, status, requested_by, idempotency_key_hash
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at, updated_at
	`, []any{refund.RefundNo, refund.OrderNo, refund.AmountMinor, refund.Currency, refund.Reason, refund.Status, refund.RequestedBy, nullEmpty(refund.IdempotencyKeyHash)}, &refund.ID, &refund.CreatedAt, &refund.UpdatedAt)
	if isUniqueConstraintViolation(err) {
		existing, getErr := r.GetRefundByOrderIdempotencyHash(ctx, refund.OrderNo, refund.IdempotencyKeyHash)
		if getErr == nil && existing != nil {
			*refund = *existing
			return nil
		}
	}
	return err
}

func (r *paymentRepository) GetRefundByOrderIdempotencyHash(ctx context.Context, orderNo string, idempotencyKeyHash string) (*service.PaymentRefund, error) {
	orderNo = strings.TrimSpace(orderNo)
	idempotencyKeyHash = strings.TrimSpace(idempotencyKeyHash)
	if orderNo == "" || idempotencyKeyHash == "" {
		return nil, service.ErrPaymentOrderNotFound
	}
	refund := &service.PaymentRefund{}
	var providerRefundID sql.NullString
	var requestedBy sql.NullInt64
	err := scanSingleRow(ctx, paymentExec(ctx, r.db), `
		SELECT id, refund_no, order_no, provider_refund_id, amount_minor, currency,
			COALESCE(reason,''), status, requested_by, COALESCE(idempotency_key_hash,''),
			created_at, updated_at
		FROM payment_refunds
		WHERE order_no = $1 AND idempotency_key_hash = $2
	`, []any{orderNo, idempotencyKeyHash},
		&refund.ID, &refund.RefundNo, &refund.OrderNo, &providerRefundID, &refund.AmountMinor, &refund.Currency,
		&refund.Reason, &refund.Status, &requestedBy, &refund.IdempotencyKeyHash,
		&refund.CreatedAt, &refund.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPaymentOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	if providerRefundID.Valid {
		refund.ProviderRefundID = providerRefundID.String
	}
	if requestedBy.Valid {
		v := requestedBy.Int64
		refund.RequestedBy = &v
	}
	return refund, nil
}

func (r *paymentRepository) UpdateRefundProvider(ctx context.Context, refundNo, providerRefundID, status string) error {
	_, err := paymentExec(ctx, r.db).ExecContext(ctx, `
		UPDATE payment_refunds
		SET provider_refund_id = NULLIF($2, ''), status = $3, updated_at = NOW()
		WHERE refund_no = $1
	`, refundNo, providerRefundID, status)
	return err
}

func (r *paymentRepository) SumSuccessfulRefundAmount(ctx context.Context, orderNo string) (int64, error) {
	var total int64
	err := scanSingleRow(ctx, paymentExec(ctx, r.db), `
		SELECT COALESCE(SUM(amount_minor), 0)
		FROM payment_refunds
		WHERE order_no = $1 AND status IN ($2, $3)
	`, []any{strings.TrimSpace(orderNo), service.PaymentRefundStatusAccepted, service.PaymentRefundStatusSettled}, &total)
	return total, err
}
