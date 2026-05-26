package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
	"github.com/stretchr/testify/require"
)

type staticResolver struct {
	ips []net.IP
	err error
}

func (r staticResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]net.IP(nil), r.ips...), nil
}

func TestClientDialValidationRejectsPrivateResolvedIPBeforeConnect(t *testing.T) {
	originalResolver := resolver
	resolver = staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}}
	t.Cleanup(func() {
		resolver = originalResolver
		sharedClients = sync.Map{}
	})
	sharedClients = sync.Map{}

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	u.Host = net.JoinHostPort("rebind.test", u.Port())

	client, err := GetClient(Options{
		Timeout:            0,
		ValidateResolvedIP: true,
		AllowPrivateHosts:  false,
	})
	require.NoError(t, err)

	resp, err := client.Get(u.String())
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
	require.Equal(t, int32(0), atomic.LoadInt32(&hits))
}

func TestClientDialValidationAllowsPrivateWhenConfigured(t *testing.T) {
	originalResolver := resolver
	resolver = staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}}
	t.Cleanup(func() {
		resolver = originalResolver
		sharedClients = sync.Map{}
	})
	sharedClients = sync.Map{}

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := GetClient(Options{
		Timeout:            0,
		ValidateResolvedIP: true,
		AllowPrivateHosts:  true,
	})
	require.NoError(t, err)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Equal(t, int32(1), atomic.LoadInt32(&hits))
}

func TestHTTPProxyHostDialValidationRejectsPrivateResolvedIP(t *testing.T) {
	originalResolver := resolver
	resolver = staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}}
	t.Cleanup(func() {
		resolver = originalResolver
		sharedClients = sync.Map{}
	})
	sharedClients = sync.Map{}

	client, err := GetClient(Options{
		ProxyURL:           "http://proxy.rebind.test:8080",
		ValidateResolvedIP: true,
		AllowPrivateHosts:  false,
	})
	require.NoError(t, err)

	resp, err := client.Get("http://example.com/")
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestSOCKSProxyHostDialValidationRejectsPrivateResolvedIP(t *testing.T) {
	for _, scheme := range []string{"socks5", "socks5h"} {
		t.Run(scheme, func(t *testing.T) {
			originalResolver := resolver
			resolver = staticResolver{ips: []net.IP{net.ParseIP("169.254.169.254")}}
			t.Cleanup(func() {
				resolver = originalResolver
				sharedClients = sync.Map{}
			})
			sharedClients = sync.Map{}

			client, err := GetClient(Options{
				ProxyURL:           fmt.Sprintf("%s://proxy.rebind.test:1080", scheme),
				ValidateResolvedIP: true,
				AllowPrivateHosts:  false,
			})
			require.NoError(t, err)

			resp, err := client.Get("http://example.com/")
			if resp != nil {
				_ = resp.Body.Close()
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), "not allowed")
		})
	}
}

func TestProxyHostDialValidationUsesNormalizedSOCKS5HSchemeForException(t *testing.T) {
	originalResolver := resolver
	resolver = staticResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}}
	t.Cleanup(func() {
		resolver = originalResolver
		sharedClients = sync.Map{}
	})
	sharedClients = sync.Map{}

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   hostexceptions.ScopeProxyHost,
			Hosts:   []string{"proxy.local.test"},
			Ports:   []int{1080},
			Schemes: []string{"socks5h"},
		},
	}

	client, err := GetClient(Options{
		ProxyURL:           "socks5://proxy.local.test:1080",
		ValidateResolvedIP: true,
		AllowPrivateHosts:  false,
		PrivateHostConfig:  cfg,
		PrivateHostScope:   hostexceptions.ScopeProxyHost,
	})
	require.NoError(t, err)

	resp, err := client.Get("http://example.com/")
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err)
	require.NotContains(t, err.Error(), "not allowed")
}
