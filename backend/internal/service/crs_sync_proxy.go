package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *CRSSyncService) mapOrCreateProxy(ctx context.Context, enabled bool, cached *[]Proxy, src *crsProxy, defaultName string) (*int64, error) {
	if !enabled || src == nil {
		return nil, nil
	}
	protocol := strings.ToLower(strings.TrimSpace(src.Protocol))
	switch protocol {
	case "socks":
		protocol = "socks5"
	case "socks5h":
		protocol = "socks5"
	}
	host := strings.TrimSpace(src.Host)
	port := src.Port
	username := strings.TrimSpace(src.Username)
	password := strings.TrimSpace(src.Password)

	if protocol == "" || host == "" || port <= 0 {
		return nil, nil
	}
	if protocol != "http" && protocol != "https" && protocol != "socks5" {
		return nil, nil
	}
	if err := ValidateProxyEndpointWithConfig(s.cfg, protocol, host, port); err != nil {
		return nil, err
	}

	// Find existing proxy (active only).
	for _, p := range *cached {
		if strings.EqualFold(p.Protocol, protocol) &&
			p.Host == host &&
			p.Port == port &&
			p.Username == username &&
			p.Password == password {
			id := p.ID
			return &id, nil
		}
	}

	// Create new proxy
	proxy := &Proxy{
		Name:     defaultProxyName(defaultName, protocol, host, port),
		Protocol: protocol,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Status:   StatusActive,
	}
	if err := s.proxyRepo.Create(ctx, proxy); err != nil {
		return nil, err
	}

	*cached = append(*cached, *proxy)
	id := proxy.ID
	return &id, nil
}

func defaultProxyName(base, protocol, host string, port int) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "crs"
	}
	return fmt.Sprintf("%s (%s://%s:%d)", base, protocol, host, port)
}
