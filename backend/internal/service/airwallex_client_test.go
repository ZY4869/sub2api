package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type airwallexRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn airwallexRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestHTTPAirwallexClientCreateRefundUsesOfficialEndpoint(t *testing.T) {
	var seenPaths []string
	var refundPayload map[string]any
	client := &HTTPAirwallexClient{httpClient: &http.Client{Transport: airwallexRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		seenPaths = append(seenPaths, req.URL.Path)
		switch req.URL.Path {
		case "/api/v1/authentication/login":
			require.Equal(t, "client", req.Header.Get("x-client-id"))
			require.Equal(t, "key", req.Header.Get("x-api-key"))
			return airwallexJSONResponse(http.StatusOK, `{"token":"token"}`), nil
		case "/api/v1/pa/refunds/create":
			require.Equal(t, "Bearer token", req.Header.Get("Authorization"))
			require.NoError(t, json.NewDecoder(req.Body).Decode(&refundPayload))
			return airwallexJSONResponse(http.StatusOK, `{"id":"rf_123","status":"succeeded"}`), nil
		default:
			t.Fatalf("unexpected airwallex path %s", req.URL.Path)
			return nil, nil
		}
	})}}

	refund, err := client.CreateRefund(context.Background(), airwallexClientTestSettings(), AirwallexRefundRequest{
		RequestID:        "rf_local",
		IntentID:         "int_123",
		PaymentAttemptID: "att_123",
		AmountMinor:      1234,
		Currency:         "USD",
		Reason:           "requested",
	})
	require.NoError(t, err)
	require.Equal(t, &AirwallexRefundResponse{ID: "rf_123", Status: "succeeded"}, refund)
	require.Equal(t, []string{"/api/v1/authentication/login", "/api/v1/pa/refunds/create"}, seenPaths)
	require.Equal(t, "rf_local", refundPayload["request_id"])
	require.Equal(t, "int_123", refundPayload["payment_intent_id"])
	require.Equal(t, "att_123", refundPayload["payment_attempt_id"])
	require.Equal(t, float64(12.34), refundPayload["amount"])
	require.NotContains(t, refundPayload, "api_key")
}

func TestHTTPAirwallexClientProviderErrorIsSanitized(t *testing.T) {
	client := &HTTPAirwallexClient{httpClient: &http.Client{Transport: airwallexRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/authentication/login" {
			return airwallexJSONResponse(http.StatusOK, `{"token":"token"}`), nil
		}
		return airwallexJSONResponse(http.StatusBadGateway, `{"client_secret":"secret","message":"provider down"}`), nil
	})}}

	_, err := client.CreateRefund(context.Background(), airwallexClientTestSettings(), AirwallexRefundRequest{
		RequestID: "rf_local", IntentID: "int_123", AmountMinor: 100, Currency: "USD",
	})
	require.ErrorIs(t, err, ErrPaymentProviderFailed)
	require.NotContains(t, err.Error(), "client_secret")
	require.NotContains(t, err.Error(), "provider down")
}

func TestHTTPAirwallexClientVerifyWebhookSignature(t *testing.T) {
	client := NewHTTPAirwallexClient()
	body := []byte(`{"id":"evt_1"}`)
	timestamp := time.Now().UnixMilli()
	signature := signAirwallexWebhook("whsec", timestamp, body)

	require.NoError(t, client.VerifyWebhookSignature("whsec", int64ToString(timestamp), "sha256="+signature, body))
	require.ErrorIs(t, client.VerifyWebhookSignature("whsec", int64ToString(timestamp), "bad", body), ErrPaymentWebhookInvalid)
	require.ErrorIs(t, client.VerifyWebhookSignature("whsec", int64ToString(time.Now().Add(-10*time.Minute).Unix()), signature, body), ErrPaymentWebhookInvalid)
}

func airwallexClientTestSettings() PaymentSettings {
	return PaymentSettings{
		AirwallexEnv:      "demo",
		AirwallexClientID: "client",
		AirwallexAPIKey:   "key",
	}
}

func airwallexJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func signAirwallexWebhook(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(int64ToString(timestamp) + string(body)))
	return hex.EncodeToString(mac.Sum(nil))
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}
