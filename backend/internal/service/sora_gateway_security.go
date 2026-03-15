package service

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var soraBlockedHostnames = map[string]struct{}{
	"localhost":                 {},
	"localhost.localdomain":     {},
	"metadata.google.internal":  {},
	"metadata.google.internal.": {},
}

var soraBlockedCIDRs = mustParseCIDRs([]string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
})

func validateSoraRemoteURL(raw string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("empty remote url")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid remote url: %w", err)
	}
	if err := validateSoraRemoteURLValue(parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func validateSoraRemoteURLValue(parsed *url.URL) error {
	if parsed == nil {
		return errors.New("invalid remote url")
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return errors.New("only http/https remote url is allowed")
	}
	if parsed.User != nil {
		return errors.New("remote url cannot contain userinfo")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return errors.New("remote url missing host")
	}
	if _, blocked := soraBlockedHostnames[host]; blocked {
		return errors.New("remote url is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isSoraBlockedIP(ip) {
			return errors.New("remote url is not allowed")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve remote url failed: %w", err)
	}
	for _, ip := range ips {
		if isSoraBlockedIP(ip) {
			return errors.New("remote url is not allowed")
		}
	}
	return nil
}

func isSoraBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	for _, cidr := range soraBlockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseCIDRs(values []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(values))
	for _, val := range values {
		_, cidr, err := net.ParseCIDR(val)
		if err != nil {
			continue
		}
		out = append(out, cidr)
	}
	return out
}
