package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *paymentRepository) CreateEventIfAbsent(ctx context.Context, event *service.PaymentEvent) (bool, error) {
	if event == nil {
		return false, nil
	}
	var id int64
	err := scanSingleRow(ctx, paymentExec(ctx, r.db), `
		INSERT INTO payment_events (
			provider, provider_event_id, order_no, event_type, event_status,
			payload_hash, payload_redacted_json
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (provider, provider_event_id) DO NOTHING
		RETURNING id
	`, []any{event.Provider, event.ProviderEventID, nullEmpty(event.OrderNo), event.EventType, event.EventStatus, event.PayloadHash, jsonOrEmpty(event.PayloadRedactedJSON)}, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *paymentRepository) MarkEventProcessed(ctx context.Context, provider, providerEventID, status, orderNo, errorReason string) error {
	_, err := paymentExec(ctx, r.db).ExecContext(ctx, `
		UPDATE payment_events
		SET event_status = $3, order_no = COALESCE($4, order_no),
			error_reason = NULLIF($5, ''), processed_at = NOW(), updated_at = NOW()
		WHERE provider = $1 AND provider_event_id = $2
	`, provider, providerEventID, status, nullEmpty(orderNo), errorReason)
	return err
}
