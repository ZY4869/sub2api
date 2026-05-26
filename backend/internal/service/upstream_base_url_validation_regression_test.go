package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestValidateUpstreamBaseURLPrivateHostExceptionRegression(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"127.0.0.1"},
			Ports:   []int{9000},
			Schemes: []string{"http"},
		},
	}

	normalized, err := validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:9000/v1")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:9000/v1", normalized)

	_, err = validateUpstreamBaseURLWithConfig(cfg, "http://127.0.0.1:9001/v1")
	require.Error(t, err)

	_, err = validateUpstreamBaseURLWithConfig(cfg, "https://127.0.0.1:9000/v1")
	require.Error(t, err)
}

func TestValidateUpstreamBaseURLPrivateExceptionDoesNotBypassPublicAllowlistRegression(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = true
	cfg.Security.URLAllowlist.UpstreamHosts = []string{"api.openai.com"}
	cfg.Security.URLAllowlist.PrivateHostExceptions = []config.PrivateHostExceptionConfig{
		{
			Scope:   PrivateHostExceptionScopeUpstreamBaseURL,
			Hosts:   []string{"93.184.216.34"},
			Ports:   []int{443},
			Schemes: []string{"https"},
		},
	}

	_, err := validateUpstreamBaseURLWithConfig(cfg, "https://93.184.216.34/v1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "host is not allowed")
}
