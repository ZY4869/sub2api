package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestValidateProxyEndpointPrivateHostExceptions(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeProxyHost,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{8080},
			Schemes: []string{"socks5h"},
		},
	}

	require.NoError(t, ValidateProxyEndpointWithConfig(cfg, "socks5h", "127.0.0.1", 8080))
	require.Error(t, ValidateProxyEndpointWithConfig(cfg, "socks5h", "127.0.0.1", 8081))
	require.Error(t, ValidateProxyEndpointWithConfig(cfg, "http", "127.0.0.1", 8080))
}

func TestValidateProxyEndpointRejectsUpstreamOnlyPrivateHostException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{8080},
			Schemes: []string{"socks5h"},
		},
	}

	err := ValidateProxyEndpointWithConfig(cfg, "socks5h", "127.0.0.1", 8080)
	require.Error(t, err)
	require.Contains(t, err.Error(), proxyInvalidHostCode)
}

func TestValidateProxyEndpointAllowsCIDRPrivateHostException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeProxyHost,
			CIDRs:   []string{"172.17.0.0/16"},
			Ports:   []int{7890},
			Schemes: []string{"http"},
		},
	}

	require.NoError(t, ValidateProxyEndpointWithConfig(cfg, "http", "172.17.0.1", 7890))
}

func TestValidateProxyURLForOutboundUsesDefaultPortForPrivateException(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeProxyHost,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{80},
			Schemes: []string{"http"},
		},
	}

	require.NoError(t, ValidateProxyURLForOutboundWithConfig("http://127.0.0.1", cfg, false))
}
