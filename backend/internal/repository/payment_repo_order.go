package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *paymentRepository) CreateOrder(ctx context.Context, order *service.PaymentOrder) error {
	if r == nil || r.db == nil || order == nil {
		return nil
	}
	exec := paymentExec(ctx, r.db)
	err := scanSingleRow(ctx, exec, `
		INSERT INTO payment_orders (
			order_no, user_id, product_type, status, provider, provider_env,
			amount_minor, currency, country_code, provider_intent_id,
			resume_token_hash, idempotency_key_hash, snapshot_json, expires_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULL,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at
	`,
		[]any{
			order.OrderNo, order.UserID, order.ProductType, order.Status, order.Provider, order.ProviderEnv,
			order.AmountMinor, order.Currency, nullEmpty(order.CountryCode), order.ResumeTokenHash,
			nullEmpty(order.IdempotencyKeyHash), jsonOrEmpty(order.SnapshotJSON), order.ExpiresAt,
		},
		&order.ID, &order.CreatedAt, &order.UpdatedAt,
	)
	if isUniqueConstraintViolation(err) {
		existing, getErr := r.GetOrderByUserIdempotencyHash(ctx, order.UserID, order.IdempotencyKeyHash)
		if getErr == nil && existing != nil {
			*order = *existing
			return nil
		}
	}
	return err
}

func (r *paymentRepository) UpdateOrderProviderIntent(ctx context.Context, orderNo string, providerIntentID string, status string) error {
	_, err := paymentExec(ctx, r.db).ExecContext(ctx, `
		UPDATE payment_orders
		SET provider_intent_id = $2, status = $3, updated_at = NOW()
		WHERE order_no = $1
	`, orderNo, providerIntentID, status)
	return err
}

func (r *paymentRepository) GetOrderByOrderNo(ctx context.Context, orderNo string) (*service.PaymentOrder, error) {
	return r.getOrder(ctx, `WHERE order_no = $1`, strings.TrimSpace(orderNo))
}

func (r *paymentRepository) GetOrderByOrderNoForUpdate(ctx context.Context, orderNo string) (*service.PaymentOrder, error) {
	return r.getOrder(ctx, `WHERE order_no = $1 FOR UPDATE`, strings.TrimSpace(orderNo))
}

func (r *paymentRepository) GetOrderByUserIdempotencyHash(ctx context.Context, userID int64, idempotencyKeyHash string) (*service.PaymentOrder, error) {
	idempotencyKeyHash = strings.TrimSpace(idempotencyKeyHash)
	if userID <= 0 || idempotencyKeyHash == "" {
		return nil, service.ErrPaymentOrderNotFound
	}
	return r.getOrder(ctx, `WHERE user_id = $1 AND idempotency_key_hash = $2`, userID, idempotencyKeyHash)
}

func (r *paymentRepository) GetOrderByResumeTokenHash(ctx context.Context, tokenHash string) (*service.PaymentOrder, error) {
	return r.getOrder(ctx, `WHERE resume_token_hash = $1`, strings.TrimSpace(tokenHash))
}

func (r *paymentRepository) GetOrderByProviderIntent(ctx context.Context, provider, providerIntentID string) (*service.PaymentOrder, error) {
	return r.getOrder(ctx, `WHERE provider = $1 AND provider_intent_id = $2`, strings.TrimSpace(provider), strings.TrimSpace(providerIntentID))
}

func (r *paymentRepository) getOrder(ctx context.Context, where string, args ...any) (*service.PaymentOrder, error) {
	query := `
		SELECT id, order_no, user_id, product_type, status, provider, provider_env,
			amount_minor,
			COALESCE((
				SELECT SUM(amount_minor)
				FROM payment_refunds
				WHERE order_no = payment_orders.order_no AND status IN ('accepted', 'settled')
			), 0) AS refunded_amount_minor,
			currency, COALESCE(country_code,''), COALESCE(provider_intent_id,''),
			resume_token_hash, COALESCE(idempotency_key_hash,''), snapshot_json,
			paid_at, refunded_at, expires_at, created_at, updated_at
		FROM payment_orders ` + where
	order := &service.PaymentOrder{}
	var snapshot []byte
	err := scanSingleRow(ctx, paymentExec(ctx, r.db), query, args,
		&order.ID, &order.OrderNo, &order.UserID, &order.ProductType, &order.Status, &order.Provider, &order.ProviderEnv,
		&order.AmountMinor, &order.RefundedAmountMinor, &order.Currency, &order.CountryCode, &order.ProviderIntentID,
		&order.ResumeTokenHash, &order.IdempotencyKeyHash, &snapshot,
		&order.PaidAt, &order.RefundedAt, &order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPaymentOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	order.SnapshotJSON = snapshot
	populatePaymentRefundableFields(order)
	return order, nil
}

func (r *paymentRepository) UpdateOrderStatus(ctx context.Context, orderNo, status string, paidAt, refundedAt *time.Time) error {
	_, err := paymentExec(ctx, r.db).ExecContext(ctx, `
		UPDATE payment_orders
		SET status = $2,
			paid_at = COALESCE($3, paid_at),
			refunded_at = COALESCE($4, refunded_at),
			updated_at = NOW()
		WHERE order_no = $1
	`, orderNo, status, paidAt, refundedAt)
	return err
}

func (r *paymentRepository) CancelOrder(ctx context.Context, orderNo string) error {
	result, err := paymentExec(ctx, r.db).ExecContext(ctx, `
		UPDATE payment_orders
		SET status = $2, updated_at = NOW()
		WHERE order_no = $1 AND status IN ($3, $4)
	`, orderNo, service.PaymentStatusCancelled, service.PaymentStatusCreated, service.PaymentStatusPending)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrPaymentOrderNotCancelable
	}
	return nil
}

func (r *paymentRepository) ListOrders(ctx context.Context, params pagination.PaginationParams, status, provider, productType string, userID *int64) ([]service.PaymentOrder, *pagination.PaginationResult, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(cond string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if strings.TrimSpace(status) != "" {
		add("status = $%d", strings.TrimSpace(status))
	}
	if strings.TrimSpace(provider) != "" {
		add("provider = $%d", strings.TrimSpace(provider))
	}
	if strings.TrimSpace(productType) != "" {
		add("product_type = $%d", strings.TrimSpace(productType))
	}
	if userID != nil && *userID > 0 {
		add("user_id = $%d", *userID)
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := scanSingleRow(ctx, paymentExec(ctx, r.db), `SELECT COUNT(*) FROM payment_orders WHERE `+whereSQL, args, &total); err != nil {
		return nil, nil, err
	}
	limit := params.Limit()
	offset := params.Offset()
	args = append(args, limit, offset)
	rows, err := paymentExec(ctx, r.db).QueryContext(ctx, `
		SELECT id, order_no, user_id, product_type, status, provider, provider_env,
			amount_minor,
			COALESCE((
				SELECT SUM(amount_minor)
				FROM payment_refunds
				WHERE order_no = payment_orders.order_no AND status IN ('accepted', 'settled')
			), 0) AS refunded_amount_minor,
			currency, COALESCE(country_code,''), COALESCE(provider_intent_id,''),
			resume_token_hash, COALESCE(idempotency_key_hash,''), snapshot_json,
			paid_at, refunded_at, expires_at, created_at, updated_at
		FROM payment_orders
		WHERE `+whereSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.PaymentOrder, 0)
	for rows.Next() {
		var item service.PaymentOrder
		var snapshot []byte
		if err := rows.Scan(
			&item.ID, &item.OrderNo, &item.UserID, &item.ProductType, &item.Status, &item.Provider, &item.ProviderEnv,
			&item.AmountMinor, &item.RefundedAmountMinor, &item.Currency, &item.CountryCode, &item.ProviderIntentID,
			&item.ResumeTokenHash, &item.IdempotencyKeyHash, &snapshot,
			&item.PaidAt, &item.RefundedAt, &item.ExpiresAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, nil, err
		}
		item.SnapshotJSON = snapshot
		populatePaymentRefundableFields(&item)
		items = append(items, item)
	}
	pages := int(math.Ceil(float64(total) / float64(limit)))
	if pages < 1 {
		pages = 1
	}
	return items, &pagination.PaginationResult{Total: total, Page: params.Page, PageSize: limit, Pages: pages}, rows.Err()
}
