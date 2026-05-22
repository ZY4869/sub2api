package service

import (
	"context"
	"encoding/json"
	"strings"

	"go.uber.org/zap"
)

func (s *PaymentService) HandleAirwallexWebhook(ctx context.Context, timestamp, signature string, body []byte) error {
	success := false
	defer func() {
		if success {
			paymentRuntimeMetrics.webhookSuccess.Add(1)
		} else {
			paymentRuntimeMetrics.webhookFailure.Add(1)
		}
	}()
	if s == nil || s.repo == nil || s.airwallex == nil {
		return ErrPaymentServiceUnavailable
	}
	settings := s.paymentSettings(ctx)
	if !settings.AirwallexWebhookSecretConfigured {
		return ErrPaymentProviderNotConfigured
	}
	if err := s.airwallex.VerifyWebhookSignature(settings.AirwallexWebhookSecret, timestamp, signature, body); err != nil {
		s.logWarn(ctx, "payment.webhook.signature_failed",
			zap.String("provider", PaymentProviderAirwallex),
			zap.String("status", "rejected"),
			zap.Error(err),
		)
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrPaymentWebhookInvalid
	}
	eventID := firstPaymentString(payload["id"], payload["event_id"])
	eventType := firstPaymentString(payload["name"], payload["type"])
	intentID := extractAirwallexIntentID(payload)
	if eventID == "" {
		eventID = hashPaymentToken(string(body))
	}
	payloadHash := hashPaymentToken(string(body))
	raw, _ := json.Marshal(redactPaymentPayload(payload))
	event := &PaymentEvent{Provider: PaymentProviderAirwallex, ProviderEventID: eventID, EventType: eventType, EventStatus: "received", PayloadHash: payloadHash, PayloadRedactedJSON: raw}
	created, err := s.repo.CreateEventIfAbsent(ctx, event)
	if err != nil || !created {
		success = true
		return err
	}
	s.logInfo(ctx, "payment.webhook.received",
		zap.String("provider", PaymentProviderAirwallex),
		zap.String("provider_event_id", eventID),
		zap.String("event_type", eventType),
		zap.String("status", "received"),
	)
	order, err := s.repo.GetOrderByProviderIntent(ctx, PaymentProviderAirwallex, intentID)
	if err != nil {
		_ = s.repo.MarkEventProcessed(ctx, PaymentProviderAirwallex, eventID, "ignored", "", "order_not_found")
		success = true
		return nil
	}
	if isAirwallexPaidEvent(eventType, payload) {
		if err := s.markOrderPaid(ctx, order); err != nil {
			_ = s.repo.MarkEventProcessed(ctx, PaymentProviderAirwallex, eventID, "failed", order.OrderNo, err.Error())
			s.logWarn(ctx, "payment.webhook.fulfillment_failed",
				zap.String("order_no", order.OrderNo),
				zap.String("provider", order.Provider),
				zap.String("provider_event_id", eventID),
				zap.String("status", "failed"),
				zap.Error(err),
			)
			return err
		}
		_ = s.repo.MarkEventProcessed(ctx, PaymentProviderAirwallex, eventID, "processed", order.OrderNo, "")
		success = true
		return nil
	}
	if isAirwallexFailedEvent(eventType, payload) {
		_ = s.repo.UpdateOrderStatus(ctx, order.OrderNo, PaymentStatusFailed, nil, nil)
		_ = s.repo.MarkEventProcessed(ctx, PaymentProviderAirwallex, eventID, "processed", order.OrderNo, "")
		success = true
		return nil
	}
	_ = s.repo.MarkEventProcessed(ctx, PaymentProviderAirwallex, eventID, "ignored", order.OrderNo, "")
	success = true
	return nil
}

func extractAirwallexIntentID(payload map[string]any) string {
	for _, key := range []string{"payment_intent_id", "intent_id", "id"} {
		if value := firstPaymentString(payload[key]); value != "" && strings.HasPrefix(value, "int_") {
			return value
		}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return extractAirwallexIntentID(data)
	}
	if obj, ok := payload["object"].(map[string]any); ok {
		return extractAirwallexIntentID(obj)
	}
	return ""
}

func isAirwallexPaidEvent(eventType string, payload map[string]any) bool {
	text := strings.ToLower(eventType + " " + firstPaymentString(payload["status"]))
	return strings.Contains(text, "succeed") || strings.Contains(text, "success") || strings.Contains(text, "paid")
}

func isAirwallexFailedEvent(eventType string, payload map[string]any) bool {
	text := strings.ToLower(eventType + " " + firstPaymentString(payload["status"]))
	return strings.Contains(text, "fail") || strings.Contains(text, "cancel") || strings.Contains(text, "expire")
}
