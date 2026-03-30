package service

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type testVertexCatalogProvider struct {
	result            *VertexCatalogResult
	err               error
	calls             int
	forceRefreshCalls []bool
	lastAccount       *Account
}

func (s *testVertexCatalogProvider) GetCatalog(ctx context.Context, account *Account, forceRefresh bool) (*VertexCatalogResult, error) {
	_ = ctx
	s.calls++
	s.lastAccount = account
	s.forceRefreshCalls = append(s.forceRefreshCalls, forceRefresh)
	if s.err != nil {
		return nil, s.err
	}
	return cloneVertexCatalogResult(s.result), nil
}

func newTestVertexCatalogProvider(result *VertexCatalogResult) *testVertexCatalogProvider {
	return &testVertexCatalogProvider{result: cloneVertexCatalogResult(result)}
}

type testVertexCatalogHTTPUpstream struct {
	requests []string
	doFunc   func(req *http.Request) (*http.Response, error)
}

func (s *testVertexCatalogHTTPUpstream) Do(
	req *http.Request,
	proxyURL string,
	accountID int64,
	accountConcurrency int,
) (*http.Response, error) {
	_ = proxyURL
	_ = accountID
	_ = accountConcurrency
	if req != nil && req.URL != nil {
		s.requests = append(s.requests, req.URL.String())
	}
	if s.doFunc != nil {
		return s.doFunc(req)
	}
	return newTestVertexCatalogHTTPResponse(http.StatusOK, `{}`), nil
}

func (s *testVertexCatalogHTTPUpstream) DoWithTLS(
	req *http.Request,
	proxyURL string,
	accountID int64,
	accountConcurrency int,
	tlsProfile *TLSFingerprintProfile,
) (*http.Response, error) {
	_ = tlsProfile
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

func newTestVertexCatalogHTTPResponse(statusCode int, body string) *http.Response {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newTestVertexUpstreamCatalogService(upstream HTTPUpstream, accessToken string) *VertexUpstreamCatalogService {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	var tokenProvider *GeminiTokenProvider
	if strings.TrimSpace(accessToken) != "" {
		tokenProvider = &GeminiTokenProvider{
			tokenCache: &accountModelImportGeminiTokenCacheStub{token: accessToken},
		}
	}
	return NewVertexUpstreamCatalogService(upstream, tokenProvider, nil, cfg)
}

func findProbeModelByID(models []AccountModelProbeModel, modelID string) (AccountModelProbeModel, bool) {
	for _, model := range models {
		if strings.TrimSpace(model.ID) == strings.TrimSpace(modelID) {
			return model, true
		}
	}
	return AccountModelProbeModel{}, false
}

func findVertexCatalogModelByID(models []VertexCatalogModel, modelID string) (VertexCatalogModel, bool) {
	for _, model := range models {
		if strings.TrimSpace(model.ID) == strings.TrimSpace(modelID) {
			return model, true
		}
	}
	return VertexCatalogModel{}, false
}
