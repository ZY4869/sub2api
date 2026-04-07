//go:build unit

package service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

func newGatewayTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
	return c, rec
}

func newJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

type queuedHTTPUpstream struct {
	responses []*http.Response
	errors    []error
	requests  []*http.Request
	callCount int
}

func (s *queuedHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	var bodyBytes []byte
	if req != nil && req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	if req != nil {
		cloned := req.Clone(req.Context())
		cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		s.requests = append(s.requests, cloned)
	}

	idx := s.callCount
	s.callCount++

	var resp *http.Response
	if idx < len(s.responses) {
		resp = s.responses[idx]
	}
	var err error
	if idx < len(s.errors) {
		err = s.errors[idx]
	}
	return resp, err
}

func (s *queuedHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *TLSFingerprintProfile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}
