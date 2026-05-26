package hostexceptions

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/netguard"
)

const (
	ScopeUpstreamBaseURL = "upstream_base_url"
	ScopeProxyHost       = "proxy_host"
)

type Match struct {
	Scope       string
	Host        string
	Port        int
	Scheme      string
	Description string
	RuleIndex   int
}

func IsAllowed(cfg *config.Config, scope, scheme, host string, port int) (Match, bool) {
	return match(cfg, scope, scheme, host, port, nil)
}

func IsResolvedIPAllowed(cfg *config.Config, scope, scheme, host string, port int, ip net.IP) (Match, bool) {
	return match(cfg, scope, scheme, host, port, ip)
}

func LogMatch(ctx context.Context, match Match) {
	if match.Scope == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	slog.WarnContext(ctx, "private_host_exception_matched",
		"scope", match.Scope,
		"host", match.Host,
		"port", match.Port,
		"scheme", match.Scheme,
		"rule_index", match.RuleIndex,
		"description", match.Description,
	)
}

func ValidateResolvedHost(ctx context.Context, cfg *config.Config, scope, scheme, host string, port int) error {
	if cfg != nil && cfg.Security.URLAllowlist.AllowPrivateHosts {
		return nil
	}
	ips, err := resolveHost(ctx, host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if reason := netguard.BlockedIPReason(ip); reason != "" {
			match, ok := IsResolvedIPAllowed(cfg, scope, scheme, host, port, ip)
			if ok {
				LogMatch(ctx, match)
				continue
			}
			return fmt.Errorf("resolved ip for host %q is not allowed: %s", host, reason)
		}
	}
	return nil
}

func ResolvedMatch(ctx context.Context, cfg *config.Config, scope, scheme, host string, port int) (Match, bool, error) {
	ips, err := resolveHost(ctx, host)
	if err != nil {
		return Match{}, false, err
	}
	for _, ip := range ips {
		if netguard.BlockedIPReason(ip) == "" {
			continue
		}
		if match, ok := IsResolvedIPAllowed(cfg, scope, scheme, host, port, ip); ok {
			return match, true, nil
		}
	}
	return Match{}, false, nil
}

func ValidateResolvedHostException(ctx context.Context, cfg *config.Config, scope, scheme, host string, port int) (Match, bool, error) {
	ips, err := resolveHost(ctx, host)
	if err != nil {
		return Match{}, false, err
	}
	match, ok := validateResolvedHostExceptionIPs(cfg, scope, scheme, host, port, ips)
	return match, ok, nil
}

func validateResolvedHostExceptionIPs(cfg *config.Config, scope, scheme, host string, port int, ips []net.IP) (Match, bool) {
	var first Match
	matched := false
	for _, ip := range ips {
		if netguard.BlockedIPReason(ip) == "" {
			continue
		}
		match, ok := IsResolvedIPAllowed(cfg, scope, scheme, host, port, ip)
		if !ok {
			return Match{}, false
		}
		if !matched {
			first = match
			matched = true
		}
	}
	return first, matched
}

func PortForURL(u *url.URL) int {
	if u == nil {
		return 0
	}
	if raw := u.Port(); raw != "" {
		port, _ := strconv.Atoi(raw)
		return port
	}
	return DefaultPortForScheme(u.Scheme)
}

func DefaultPortForScheme(scheme string) int {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "http":
		return 80
	case "https":
		return 443
	case "socks5", "socks5h":
		return 1080
	default:
		return 0
	}
}

func Key(cfg *config.Config) string {
	if cfg == nil || len(cfg.Security.URLAllowlist.PrivateHostExceptions) == 0 {
		return "exceptions:none"
	}
	parts := make([]string, 0, len(cfg.Security.URLAllowlist.PrivateHostExceptions))
	for _, rule := range cfg.Security.URLAllowlist.PrivateHostExceptions {
		parts = append(parts, strings.Join([]string{
			strings.ToLower(strings.TrimSpace(rule.Scope)),
			normalizeListKey(rule.Hosts),
			normalizeListKey(rule.CIDRs),
			intListKey(rule.Ports),
			normalizeListKey(rule.Schemes),
		}, "|"))
	}
	return "exceptions:" + strings.Join(parts, ";")
}

func match(cfg *config.Config, scope, scheme, host string, port int, resolvedIP net.IP) (Match, bool) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	host = normalizeHost(host)
	if cfg == nil || scope == "" || host == "" || port <= 0 {
		return Match{}, false
	}

	hostIP := net.ParseIP(host)
	if hostIP == nil {
		hostIP = resolvedIP
	}

	for i, rule := range cfg.Security.URLAllowlist.PrivateHostExceptions {
		if strings.ToLower(strings.TrimSpace(rule.Scope)) != scope {
			continue
		}
		if !matchesPort(rule.Ports, port) || !matchesScheme(rule.Schemes, scheme) {
			continue
		}
		if matchesHost(rule.Hosts, host, port) || matchesCIDR(rule.CIDRs, hostIP) {
			return Match{
				Scope:       scope,
				Host:        host,
				Port:        port,
				Scheme:      scheme,
				Description: strings.TrimSpace(rule.Description),
				RuleIndex:   i,
			}, true
		}
	}
	return Match{}, false
}

func matchesPort(ports []int, port int) bool {
	if len(ports) == 0 {
		return false
	}
	for _, candidate := range ports {
		if candidate == port {
			return true
		}
	}
	return false
}

func matchesScheme(schemes []string, scheme string) bool {
	if len(schemes) == 0 {
		return false
	}
	if scheme == "" {
		return false
	}
	for _, candidate := range schemes {
		if strings.EqualFold(strings.TrimSpace(candidate), scheme) {
			return true
		}
	}
	return false
}

func matchesHost(hosts []string, host string, port int) bool {
	for _, candidate := range hosts {
		normalized, candidatePort, ok := normalizeHostRule(candidate)
		if !ok || normalized == "" {
			continue
		}
		if candidatePort > 0 && candidatePort != port {
			continue
		}
		if normalized == host {
			return true
		}
	}
	return false
}

func matchesCIDR(cidrs []string, ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, raw := range cidrs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func normalizeHostRule(raw string) (string, int, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", 0, false
	}
	if strings.Contains(value, "://") || strings.ContainsAny(value, "/?#@") {
		return "", 0, false
	}
	if host, port, err := net.SplitHostPort(value); err == nil {
		parsedPort, err := strconv.Atoi(port)
		if err != nil || parsedPort <= 0 || parsedPort > 65535 {
			return "", 0, false
		}
		return normalizeHost(host), parsedPort, true
	}
	return normalizeHost(value), 0, true
}

func normalizeHost(host string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(host), "[]"))
}

func normalizeListKey(values []string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, ",")
}

func intListKey(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}

func resolveHost(ctx context.Context, host string) ([]net.IP, error) {
	host = normalizeHost(host)
	if host == "" {
		return nil, fmt.Errorf("host is empty")
	}
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	lookupCtx := ctx
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		lookupCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	ips, err := net.DefaultResolver.LookupIP(lookupCtx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("dns resolution failed for host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("dns resolution returned no records for host %q", host)
	}
	return ips, nil
}
