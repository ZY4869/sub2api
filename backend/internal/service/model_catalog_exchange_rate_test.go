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
	callCount := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		switch callCount {
		case 1:
			body := `{"base":"USD","date":"2026-03-10","rates":{"CNY":7.2}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		case 2:
			body := `{"base":"USD","date":"2026-03-11","rates":{"CNY":7.3}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		default:
			return nil, context.DeadlineExceeded
		}
	})}

	svc := newModelCatalogExchangeRateService(client)
	rate, err := svc.GetUSDCNY(context.Background())
	require.NoError(t, err)
	require.Equal(t, 7.2, rate.Rate)
	require.False(t, rate.Cached)
	require.Equal(t, 1, callCount)

	cached, err := svc.GetUSDCNY(context.Background())
	require.NoError(t, err)
	require.Equal(t, 7.2, cached.Rate)
	require.True(t, cached.Cached)
	require.Equal(t, 1, callCount)

	refreshed, err := svc.RefreshUSDCNY(context.Background())
	require.NoError(t, err)
	require.Equal(t, 7.3, refreshed.Rate)
	require.False(t, refreshed.Cached)
	require.Equal(t, 2, callCount)

	svc.ttl = time.Hour
	stale, err := svc.RefreshUSDCNY(context.Background())
	require.NoError(t, err)
	require.Equal(t, 7.3, stale.Rate)
	require.True(t, stale.Cached)
	require.Equal(t, 3, callCount)
}
