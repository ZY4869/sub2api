package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
)

const (
	PrivateHostExceptionScopeUpstreamBaseURL = hostexceptions.ScopeUpstreamBaseURL
	PrivateHostExceptionScopeProxyHost       = hostexceptions.ScopeProxyHost
)

type PrivateHostExceptionMatch = hostexceptions.Match

func IsPrivateHostExceptionAllowed(cfg *config.Config, scope, scheme, host string, port int) (PrivateHostExceptionMatch, bool) {
	return hostexceptions.IsAllowed(cfg, scope, scheme, host, port)
}

func LogPrivateHostExceptionMatch(ctx context.Context, match PrivateHostExceptionMatch) {
	hostexceptions.LogMatch(ctx, match)
}

func PrivateHostExceptionKey(cfg *config.Config) string {
	return hostexceptions.Key(cfg)
}
