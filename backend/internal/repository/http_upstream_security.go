package repository

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
	"net/http"
	"strings"
)

func (s *httpUpstreamService) shouldValidateResolvedIP() bool {
	if s.cfg == nil {
		return true
	}
	return !s.cfg.Security.URLAllowlist.AllowPrivateHosts
}

func (s *httpUpstreamService) validateRequestHost(req *http.Request) error {
	if !s.shouldValidateResolvedIP() {
		return nil
	}
	if req == nil || req.URL == nil {
		return errors.New("request url is nil")
	}
	host := strings.TrimSpace(req.URL.Hostname())
	if host == "" {
		return errors.New("request host is empty")
	}
	port := hostexceptions.PortForURL(req.URL)
	return hostexceptions.ValidateResolvedHost(req.Context(), s.cfg, hostexceptions.ScopeUpstreamBaseURL, req.URL.Scheme, host, port)
}

func withUpstreamRequestScheme(req *http.Request) *http.Request {
	if req == nil || req.URL == nil || strings.TrimSpace(req.URL.Scheme) == "" {
		return req
	}
	ctx := context.WithValue(req.Context(), upstreamRequestSchemeContextKey{}, strings.ToLower(strings.TrimSpace(req.URL.Scheme)))
	return req.WithContext(ctx)
}

func (s *httpUpstreamService) redirectChecker(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

// acquireClient 获取或创建客户端，并标记为进行中请求
// 用于请求路径，避免在获取后被淘汰
