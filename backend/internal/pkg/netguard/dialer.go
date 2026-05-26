package netguard

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// Resolver is the subset of net.Resolver used by guarded outbound dialing.
type Resolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

type defaultResolver struct{}

func (defaultResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

// Options configures a Dialer.
type Options struct {
	Base                       *net.Dialer
	Resolver                   Resolver
	ValidateResolvedIP         bool
	AllowPrivateHosts          bool
	AllowResolvedIP            func(host string, port int, ip net.IP) bool
	AllowResolvedIPWithContext func(ctx context.Context, host string, port int, ip net.IP) bool
	ResolutionTimeout          time.Duration
}

// Dialer resolves outbound hosts, rejects unsafe resolved IPs, and dials the
// verified IP address to avoid DNS rebinding between preflight and connect.
type Dialer struct {
	base               *net.Dialer
	resolver           Resolver
	validateResolved   bool
	allowPrivateHosts  bool
	allowResolvedIP    func(host string, port int, ip net.IP) bool
	allowResolvedIPCtx func(ctx context.Context, host string, port int, ip net.IP) bool
	resolutionTimeout  time.Duration
}

// NewDialer returns a guarded dialer. When validation is enabled it resolves
// and dials the selected IP address; AllowPrivateHosts only controls whether
// unsafe ranges are rejected.
func NewDialer(opts Options) *Dialer {
	base := opts.Base
	if base == nil {
		base = &net.Dialer{}
	}
	resolver := opts.Resolver
	if resolver == nil {
		resolver = defaultResolver{}
	}
	timeout := opts.ResolutionTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Dialer{
		base:               base,
		resolver:           resolver,
		validateResolved:   opts.ValidateResolvedIP,
		allowPrivateHosts:  opts.AllowPrivateHosts,
		allowResolvedIP:    opts.AllowResolvedIP,
		allowResolvedIPCtx: opts.AllowResolvedIPWithContext,
		resolutionTimeout:  timeout,
	}
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d == nil {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	if !d.validateResolved {
		return d.base.DialContext(ctx, network, address)
	}
	dialAddresses, err := d.resolveDialAddresses(ctx, network, address)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, dialAddress := range dialAddresses {
		conn, err := d.base.DialContext(ctx, network, dialAddress)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("dns resolution produced no dialable address for %q", address)
}

func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *Dialer) resolveDialAddresses(ctx context.Context, network, address string) ([]string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("outbound host is empty")
	}
	portNumber, err := parsePort(port)
	if err != nil {
		return nil, err
	}

	lookupCtx := ctx
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok && d.resolutionTimeout > 0 {
		lookupCtx, cancel = context.WithTimeout(ctx, d.resolutionTimeout)
		defer cancel()
	}

	ips, err := d.lookupIP(lookupCtx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("dns resolution returned no records for host %q", host)
	}

	dialAddresses := make([]string, 0, len(ips))
	for _, ip := range ips {
		if reason := BlockedIPReason(ip); reason != "" && !d.allowPrivateHosts {
			if !d.isResolvedIPAllowed(ctx, host, portNumber, normalizeIP(ip)) {
				return nil, fmt.Errorf("resolved ip for host %q is not allowed: %s", host, reason)
			}
		}
		if ipMatchesNetwork(network, ip) {
			dialAddresses = append(dialAddresses, net.JoinHostPort(normalizeIP(ip).String(), port))
		}
	}
	if len(dialAddresses) == 0 {
		return nil, fmt.Errorf("dns resolution for host %q returned no %s-compatible address", host, network)
	}
	return dialAddresses, nil
}

func (d *Dialer) isResolvedIPAllowed(ctx context.Context, host string, port int, ip net.IP) bool {
	if d.allowResolvedIPCtx != nil && d.allowResolvedIPCtx(ctx, host, port, ip) {
		return true
	}
	return d.allowResolvedIP != nil && d.allowResolvedIP(host, port, ip)
}

func (d *Dialer) lookupIP(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	ips, err := d.resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("dns resolution failed for host %q: %w", host, err)
	}
	return ips, nil
}

func parsePort(port string) (int, error) {
	value, err := net.LookupPort("tcp", port)
	if err != nil || value <= 0 || value > 65535 {
		return 0, fmt.Errorf("invalid outbound port: %s", port)
	}
	return value, nil
}

// BlockedIPReason returns a non-empty reason when the IP is unsafe for default
// outbound access.
func BlockedIPReason(ip net.IP) string {
	ip = normalizeIP(ip)
	if ip == nil {
		return "invalid"
	}
	switch {
	case ip.IsLoopback():
		return "loopback"
	case ip.IsPrivate():
		return "private"
	case ip.IsLinkLocalUnicast():
		return "link-local"
	case ip.IsLinkLocalMulticast():
		return "link-local-multicast"
	case ip.IsMulticast():
		return "multicast"
	case ip.IsUnspecified():
		return "unspecified"
	default:
		return ""
	}
}

func normalizeIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}

func ipMatchesNetwork(network string, ip net.IP) bool {
	ip = normalizeIP(ip)
	switch strings.ToLower(network) {
	case "tcp4":
		return ip.To4() != nil
	case "tcp6":
		return ip.To4() == nil
	default:
		return true
	}
}
