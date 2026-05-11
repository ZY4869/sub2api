package repository

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *httpUpstreamService) finalizeResponse(resp *http.Response, entry *upstreamClientEntry) *http.Response {
	if resp == nil {
		return nil
	}
	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		resp = service.RewriteUpstreamRedirectBlockedResponse(resp)
	}
	resp.Body = wrapTrackedBody(resp.Body, func() {
		atomic.AddInt64(&entry.inFlight, -1)
		atomic.StoreInt64(&entry.lastUsed, time.Now().UnixNano())
	})
	return resp
}
