package servertiming

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCollectorHeaderValueAggregatesSafeMetricNames(t *testing.T) {
	start := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	ctx := WithCollector(context.Background(), New(start))

	Record(ctx, "HTTP upstream", 2*time.Millisecond)
	Record(ctx, "HTTP upstream", 3*time.Millisecond)
	Record(ctx, "sql;drop", 4*time.Millisecond)

	header := HeaderValue(ctx, start.Add(10*time.Millisecond))
	if !strings.Contains(header, "app;dur=10.0") {
		t.Fatalf("header should include app duration: %s", header)
	}
	if !strings.Contains(header, `httpupstream;dur=5.0;desc="2"`) {
		t.Fatalf("header should aggregate duplicate metrics: %s", header)
	}
	if strings.Contains(header, ";drop") {
		t.Fatalf("metric names must be sanitized: %s", header)
	}
}

func TestRoundTripperRecordsHTTPOnlyWhenCollectorIsActive(t *testing.T) {
	next := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
	})
	rt := WrapRoundTripper(next)

	req, _ := http.NewRequest(http.MethodGet, "https://example.test", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip without collector failed: %v", err)
	}

	ctx := WithCollector(context.Background(), New(time.Now()))
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip with collector failed: %v", err)
	}
	if header := HeaderValue(ctx, time.Now()); !strings.Contains(header, "http;dur=") {
		t.Fatalf("header should include http timing: %s", header)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
