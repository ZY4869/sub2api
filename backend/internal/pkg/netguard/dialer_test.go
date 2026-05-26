package netguard

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type testResolver struct {
	ips []net.IP
	err error
}

func (r testResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]net.IP(nil), r.ips...), nil
}

func TestDialerRejectsPrivateResolvedIPBeforeConnect(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	u.Host = net.JoinHostPort("rebind.test", u.Port())

	dialer := NewDialer(Options{
		Resolver:           testResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}},
		ValidateResolvedIP: true,
	})
	transport := &http.Transport{DialContext: dialer.DialContext}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(u.String())
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "loopback")
	require.Equal(t, int32(0), atomic.LoadInt32(&hits))
}

func TestDialerPreservesOriginalHTTPHostWhileDialingVerifiedIP(t *testing.T) {
	hostCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostCh <- r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	u.Host = net.JoinHostPort("api.example.com", u.Port())

	dialer := NewDialer(Options{
		Resolver:           testResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}},
		ValidateResolvedIP: true,
		AllowPrivateHosts:  true,
	})
	transport := &http.Transport{DialContext: dialer.DialContext}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(u.String())
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, u.Host, <-hostCh)
}

func TestDialerAllowsPrivateResolvedIPByCallback(t *testing.T) {
	hostCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostCh <- r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	u.Host = net.JoinHostPort("local-upstream.test", u.Port())

	dialer := NewDialer(Options{
		Resolver:           testResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}},
		ValidateResolvedIP: true,
		AllowResolvedIP: func(host string, port int, ip net.IP) bool {
			return host == "local-upstream.test" && port == serverPort(t, server.URL) && ip.IsLoopback()
		},
	})
	transport := &http.Transport{DialContext: dialer.DialContext}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(u.String())
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, u.Host, <-hostCh)
}

func TestDialerAllowsPrivateResolvedIPByContextCallback(t *testing.T) {
	hostCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostCh <- r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	u.Host = net.JoinHostPort("local-upstream.test", u.Port())

	type schemeContextKey struct{}
	dialer := NewDialer(Options{
		Resolver:           testResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}},
		ValidateResolvedIP: true,
		AllowResolvedIPWithContext: func(ctx context.Context, host string, port int, ip net.IP) bool {
			scheme, _ := ctx.Value(schemeContextKey{}).(string)
			return scheme == "http" && host == "local-upstream.test" && port == serverPort(t, server.URL) && ip.IsLoopback()
		},
	})
	transport := &http.Transport{DialContext: dialer.DialContext}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	require.NoError(t, err)
	req = req.WithContext(context.WithValue(req.Context(), schemeContextKey{}, "http"))

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, u.Host, <-hostCh)
}

func TestDialerRejectsPrivateResolvedIPWhenCallbackDoesNotMatch(t *testing.T) {
	dialer := NewDialer(Options{
		Resolver:           testResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}},
		ValidateResolvedIP: true,
		AllowResolvedIP: func(host string, port int, ip net.IP) bool {
			return false
		},
	})

	_, err := dialer.DialContext(context.Background(), "tcp", "local-upstream.test:9000")
	require.Error(t, err)
	require.Contains(t, err.Error(), "loopback")
}

func TestBlockedIPReasonCoversUnsafeRanges(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1":       "loopback",
		"10.0.0.1":        "private",
		"172.16.0.1":      "private",
		"192.168.1.1":     "private",
		"169.254.169.254": "link-local",
		"::1":             "loopback",
		"fc00::1":         "private",
		"::":              "unspecified",
		"ff02::1":         "link-local-multicast",
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			require.Equal(t, want, BlockedIPReason(net.ParseIP(raw)))
		})
	}
}

func serverPort(t *testing.T, raw string) int {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	port, err := net.LookupPort("tcp", u.Port())
	require.NoError(t, err)
	return port
}
