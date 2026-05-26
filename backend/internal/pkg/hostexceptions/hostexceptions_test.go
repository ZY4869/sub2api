package hostexceptions

import (
	"net"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestIsAllowedMatchesScopeHostPortAndScheme(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   ScopeUpstreamBaseURL,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	_, ok := IsAllowed(cfg, ScopeUpstreamBaseURL, "http", "127.0.0.1", 9000)
	require.True(t, ok)

	_, ok = IsAllowed(cfg, ScopeProxyHost, "http", "127.0.0.1", 9000)
	require.False(t, ok)

	_, ok = IsAllowed(cfg, ScopeUpstreamBaseURL, "https", "127.0.0.1", 9000)
	require.False(t, ok)

	_, ok = IsAllowed(cfg, ScopeUpstreamBaseURL, "http", "127.0.0.1", 9001)
	require.False(t, ok)
}

func TestIsResolvedIPAllowedMatchesCIDR(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   ScopeUpstreamBaseURL,
			CIDRs:   []string{"172.17.0.0/16"},
			Ports:   []int{8080},
			Schemes: []string{"http"},
		},
	}

	_, ok := IsResolvedIPAllowed(cfg, ScopeUpstreamBaseURL, "http", "docker-api.internal", 8080, net.ParseIP("172.17.0.10"))
	require.True(t, ok)

	_, ok = IsResolvedIPAllowed(cfg, ScopeUpstreamBaseURL, "http", "docker-api.internal", 8081, net.ParseIP("172.17.0.10"))
	require.False(t, ok)
}

func TestIsAllowedRequiresExplicitPortAndScheme(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope: ScopeUpstreamBaseURL,
			Hosts: []string{"127.0.0.1"},
		},
	}

	_, ok := IsAllowed(cfg, ScopeUpstreamBaseURL, "http", "127.0.0.1", 9000)
	require.False(t, ok)

	cfg.Security.URLAllowlist.PrivateHostExceptions[0].Ports = []int{9000}
	_, ok = IsAllowed(cfg, ScopeUpstreamBaseURL, "http", "127.0.0.1", 9000)
	require.False(t, ok)
}

func TestResolvedMatchOnlyAppliesToBlockedResolvedIP(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   ScopeUpstreamBaseURL,
			Hosts:   []string{"93.184.216.34"},
			Ports:   []int{443},
			Schemes: []string{"https"},
		},
	}

	_, ok, err := ResolvedMatch(t.Context(), cfg, ScopeUpstreamBaseURL, "https", "93.184.216.34", 443)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestValidateResolvedHostExceptionRequiresEveryBlockedIPToMatch(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   ScopeUpstreamBaseURL,
			CIDRs:   []string{"127.0.0.0/8"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	_, ok := validateResolvedHostExceptionIPs(
		cfg,
		ScopeUpstreamBaseURL,
		"http",
		"local-upstream.test",
		9000,
		[]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.17.0.1")},
	)
	require.False(t, ok)

	cfg.Security.URLAllowlist.PrivateHostExceptions[0].CIDRs = []string{"127.0.0.0/8", "172.17.0.0/16"}
	match, ok := validateResolvedHostExceptionIPs(
		cfg,
		ScopeUpstreamBaseURL,
		"http",
		"local-upstream.test",
		9000,
		[]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.17.0.1")},
	)
	require.True(t, ok)
	require.Equal(t, ScopeUpstreamBaseURL, match.Scope)
}
