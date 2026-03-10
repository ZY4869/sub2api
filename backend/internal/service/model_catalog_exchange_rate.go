package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const modelCatalogExchangeRateURL = "https://api.frankfurter.dev/v1/latest?base=USD"

type modelCatalogExchangeRateHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ModelCatalogExchangeRateService struct {
	client modelCatalogExchangeRateHTTPClient
	ttl    time.Duration

	mu     sync.RWMutex
	cached *ModelCatalogExchangeRate
}

type modelCatalogExchangeRateResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

func newModelCatalogExchangeRateService(client modelCatalogExchangeRateHTTPClient) *ModelCatalogExchangeRateService {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &ModelCatalogExchangeRateService{client: client, ttl: time.Hour}
}

func (s *ModelCatalogExchangeRateService) GetUSDCNY(ctx context.Context) (*ModelCatalogExchangeRate, error) {
	if cached := s.getFreshCache(); cached != nil {
		copy := *cached
		copy.Cached = true
		return &copy, nil
	}
	rate, err := s.fetch(ctx)
	if err == nil {
		s.setCache(rate)
		copy := *rate
		copy.Cached = false
		return &copy, nil
	}
	if cached := s.getAnyCache(); cached != nil {
		logger.FromContext(ctx).Warn("model catalog: exchange rate fetch failed, using stale cache", zap.Error(err))
		copy := *cached
		copy.Cached = true
		return &copy, nil
	}
	return nil, err
}

func (s *ModelCatalogExchangeRateService) fetch(ctx context.Context) (*ModelCatalogExchangeRate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelCatalogExchangeRateURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("exchange rate request failed: status %d", resp.StatusCode)
	}
	var payload modelCatalogExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	rate, ok := payload.Rates["CNY"]
	if !ok || rate <= 0 {
		return nil, fmt.Errorf("exchange rate response missing CNY rate")
	}
	return &ModelCatalogExchangeRate{
		Base:      payload.Base,
		Quote:     "CNY",
		Rate:      rate,
		Date:      payload.Date,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *ModelCatalogExchangeRateService) getFreshCache() *ModelCatalogExchangeRate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cached == nil {
		return nil
	}
	if time.Since(s.cached.UpdatedAt) > s.ttl {
		return nil
	}
	copy := *s.cached
	return &copy
}

func (s *ModelCatalogExchangeRateService) getAnyCache() *ModelCatalogExchangeRate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cached == nil {
		return nil
	}
	copy := *s.cached
	return &copy
}

func (s *ModelCatalogExchangeRateService) setCache(rate *ModelCatalogExchangeRate) {
	if rate == nil {
		return
	}
	copy := *rate
	s.mu.Lock()
	s.cached = &copy
	s.mu.Unlock()
}
