package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestModelCatalogExchangeRateService_UsesFreshAndStaleCache(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"base":"USD","date":"2026-03-10","rates":{"CNY":7.2}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	svc := newModelCatalogExchangeRateService(client)
	rate, err := svc.GetUSDCNY(context.Background())
	require.NoError(t, err)
	require.Equal(t, 7.2, rate.Rate)
	require.False(t, rate.Cached)

	cached, err := svc.GetUSDCNY(context.Background())
	require.NoError(t, err)
	require.Equal(t, 7.2, cached.Rate)
	require.True(t, cached.Cached)

	svc.ttl = 0
	svc.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}
	svc.cached.UpdatedAt = time.Now().Add(-2 * time.Hour)

	stale, err := svc.GetUSDCNY(context.Background())
	require.NoError(t, err)
	require.Equal(t, 7.2, stale.Rate)
	require.True(t, stale.Cached)
}
