package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const airwallexWebhookTimestampTolerance = 5 * time.Minute

type HTTPAirwallexClient struct {
	httpClient *http.Client
}

func NewHTTPAirwallexClient() *HTTPAirwallexClient {
	return &HTTPAirwallexClient{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func (c *HTTPAirwallexClient) CreatePaymentIntent(ctx context.Context, settings PaymentSettings, req AirwallexPaymentIntentRequest) (*AirwallexPaymentIntentResponse, error) {
	token, err := c.login(ctx, settings)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"request_id":        req.RequestID,
		"amount":            PaymentMinorToAmount(req.AmountMinor, req.Currency),
		"currency":          req.Currency,
		"merchant_order_id": req.OrderNo,
		"descriptor":        req.Descriptor,
		"metadata":          req.Metadata,
	}
	if strings.TrimSpace(req.ReturnURL) != "" {
		payload["return_url"] = strings.TrimSpace(req.ReturnURL)
	}
	var out struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
		Status       string `json:"status"`
	}
	if err := c.postJSON(ctx, settings, token, "/api/v1/pa/payment_intents/create", payload, &out); err != nil {
		return nil, err
	}
	return &AirwallexPaymentIntentResponse{ID: out.ID, ClientSecret: out.ClientSecret, Status: out.Status}, nil
}

func (c *HTTPAirwallexClient) RetrievePaymentIntent(ctx context.Context, settings PaymentSettings, intentID string) (*AirwallexPaymentIntentResponse, error) {
	token, err := c.login(ctx, settings)
	if err != nil {
		return nil, err
	}
	var out struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
		Status       string `json:"status"`
	}
	path := fmt.Sprintf("/api/v1/pa/payment_intents/%s", strings.TrimSpace(intentID))
	if err := c.getJSON(ctx, settings, token, path, &out); err != nil {
		return nil, err
	}
	return &AirwallexPaymentIntentResponse{ID: out.ID, ClientSecret: out.ClientSecret, Status: out.Status}, nil
}

func (c *HTTPAirwallexClient) CreateRefund(ctx context.Context, settings PaymentSettings, req AirwallexRefundRequest) (*AirwallexRefundResponse, error) {
	token, err := c.login(ctx, settings)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"request_id":        req.RequestID,
		"payment_intent_id": strings.TrimSpace(req.IntentID),
		"amount":            PaymentMinorToAmount(req.AmountMinor, req.Currency),
		"currency":          req.Currency,
	}
	if strings.TrimSpace(req.PaymentAttemptID) != "" {
		payload["payment_attempt_id"] = strings.TrimSpace(req.PaymentAttemptID)
	}
	if strings.TrimSpace(req.Reason) != "" {
		payload["reason"] = strings.TrimSpace(req.Reason)
	}
	path := "/api/v1/pa/refunds/create"
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := c.postJSON(ctx, settings, token, path, payload, &out); err != nil {
		return nil, err
	}
	return &AirwallexRefundResponse{ID: out.ID, Status: out.Status}, nil
}

func (c *HTTPAirwallexClient) VerifyWebhookSignature(secret string, timestamp string, signature string, body []byte) error {
	secret = strings.TrimSpace(secret)
	timestamp = strings.TrimSpace(timestamp)
	signature = strings.TrimSpace(signature)
	if secret == "" || timestamp == "" || signature == "" {
		return ErrPaymentWebhookInvalid
	}
	if err := validateAirwallexWebhookTimestamp(timestamp, time.Now()); err != nil {
		return err
	}
	payload := []byte(timestamp + string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	signature = strings.TrimPrefix(strings.ToLower(signature), "sha256=")
	if !hmac.Equal([]byte(strings.ToLower(expected)), []byte(signature)) {
		return ErrPaymentWebhookInvalid
	}
	return nil
}

func validateAirwallexWebhookTimestamp(timestamp string, now time.Time) error {
	ts, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil || ts <= 0 {
		return ErrPaymentWebhookInvalid
	}
	var eventTime time.Time
	switch {
	case ts > 1_000_000_000_000:
		eventTime = time.UnixMilli(ts)
	default:
		eventTime = time.Unix(ts, 0)
	}
	if eventTime.Before(now.Add(-airwallexWebhookTimestampTolerance)) || eventTime.After(now.Add(airwallexWebhookTimestampTolerance)) {
		return ErrPaymentWebhookInvalid
	}
	return nil
}

func (c *HTTPAirwallexClient) login(ctx context.Context, settings PaymentSettings) (string, error) {
	if strings.TrimSpace(settings.AirwallexClientID) == "" || strings.TrimSpace(settings.AirwallexAPIKey) == "" {
		return "", ErrPaymentProviderNotConfigured
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, airwallexBaseURL(settings.AirwallexEnv)+"/api/v1/authentication/login", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-client-id", settings.AirwallexClientID)
	req.Header.Set("x-api-key", settings.AirwallexAPIKey)
	var out struct {
		Token string `json:"token"`
	}
	if err := c.do(req, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Token) == "" {
		return "", ErrPaymentProviderNotConfigured
	}
	return out.Token, nil
}

func (c *HTTPAirwallexClient) postJSON(ctx context.Context, settings PaymentSettings, token string, path string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, airwallexBaseURL(settings.AirwallexEnv)+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *HTTPAirwallexClient) getJSON(ctx context.Context, settings PaymentSettings, token string, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, airwallexBaseURL(settings.AirwallexEnv)+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return c.do(req, out)
}

func (c *HTTPAirwallexClient) do(req *http.Request, out any) error {
	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrPaymentProviderFailed.WithMetadata(map[string]string{
			"provider": "airwallex",
			"status":   fmt.Sprintf("%d", resp.StatusCode),
		})
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func airwallexBaseURL(env string) string {
	if NormalizeAirwallexEnv(env) == "prod" {
		return "https://api.airwallex.com"
	}
	return "https://api-demo.airwallex.com"
}
