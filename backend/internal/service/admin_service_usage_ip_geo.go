package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const DefaultUsageIPGeoProviderURL = "https://get.geojs.io/v1/ip/geo/{ip}.json"

type UsageIPGeoLookupItem struct {
	IP      string `json:"ip"`
	Status  string `json:"status"`
	Country string `json:"country,omitempty"`
	Region  string `json:"region,omitempty"`
	City    string `json:"city,omitempty"`
	ASN     string `json:"asn,omitempty"`
	Cached  bool   `json:"cached"`
}

type usageIPGeoCache struct {
	ttl   time.Duration
	mu    sync.Mutex
	items map[string]usageIPGeoCacheEntry
}

type usageIPGeoCacheEntry struct {
	item      UsageIPGeoLookupItem
	expiresAt time.Time
}

func newUsageIPGeoCache(ttl time.Duration) *usageIPGeoCache {
	return &usageIPGeoCache{ttl: ttl, items: map[string]usageIPGeoCacheEntry{}}
}

func (c *usageIPGeoCache) get(ip string) (UsageIPGeoLookupItem, bool) {
	if c == nil {
		return UsageIPGeoLookupItem{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[ip]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(c.items, ip)
		}
		return UsageIPGeoLookupItem{}, false
	}
	entry.item.Cached = true
	return entry.item, true
}

func (c *usageIPGeoCache) set(item UsageIPGeoLookupItem) {
	if c == nil || item.IP == "" || item.Status != "ok" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[item.IP] = usageIPGeoCacheEntry{item: item, expiresAt: time.Now().Add(c.ttl)}
}

func (s *adminServiceImpl) LookupUsageIPGeo(ctx context.Context, ips []string) ([]UsageIPGeoLookupItem, error) {
	if len(ips) > 50 {
		return nil, infraerrors.BadRequest("USAGE_IP_GEO_TOO_MANY_IPS", "at most 50 IPs can be looked up at once")
	}
	settings := &SystemSettings{UsageIPGeoEnabled: false, UsageIPGeoProviderURL: DefaultUsageIPGeoProviderURL, UsageIPGeoTimeoutMs: 1500}
	if s != nil && s.settingService != nil {
		if loaded, err := s.settingService.GetAllSettings(ctx); err == nil && loaded != nil {
			settings = loaded
		}
	}
	out := make([]UsageIPGeoLookupItem, 0, len(ips))
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		item := UsageIPGeoLookupItem{IP: ip}
		parsed := net.ParseIP(ip)
		if parsed == nil {
			item.Status = "error"
			out = append(out, item)
			continue
		}
		if isPrivateUsageIP(parsed) {
			item.Status = "private"
			out = append(out, item)
			continue
		}
		if !settings.UsageIPGeoEnabled {
			item.Status = "disabled"
			out = append(out, item)
			continue
		}
		if cached, ok := s.usageIPGeoCache.get(ip); ok {
			out = append(out, cached)
			continue
		}
		startedAt := time.Now()
		lookup, err := lookupUsageIPGeoProvider(ctx, settings, ip)
		if err != nil {
			logUsageIPGeoLookupFailed(ctx, ip, time.Since(startedAt), err)
			lookup = UsageIPGeoLookupItem{IP: ip, Status: "error"}
		}
		if lookup.Status == "" {
			lookup.Status = "not_found"
		}
		if lookup.IP == "" {
			lookup.IP = ip
		}
		s.usageIPGeoCache.set(lookup)
		out = append(out, lookup)
	}
	return out, nil
}

func lookupUsageIPGeoProvider(ctx context.Context, settings *SystemSettings, ip string) (UsageIPGeoLookupItem, error) {
	endpoint := strings.TrimSpace(settings.UsageIPGeoProviderURL)
	if endpoint == "" {
		endpoint = DefaultUsageIPGeoProviderURL
	}
	endpoint = strings.ReplaceAll(endpoint, "{ip}", url.PathEscape(ip))
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return UsageIPGeoLookupItem{}, fmt.Errorf("invalid usage ip geo provider url")
	}
	timeout := time.Duration(settings.UsageIPGeoTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return UsageIPGeoLookupItem{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return UsageIPGeoLookupItem{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return UsageIPGeoLookupItem{IP: ip, Status: "not_found"}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UsageIPGeoLookupItem{}, fmt.Errorf("usage ip geo provider status %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32*1024)).Decode(&payload); err != nil {
		return UsageIPGeoLookupItem{}, err
	}
	item := UsageIPGeoLookupItem{
		IP:      ip,
		Status:  "ok",
		Country: firstGeoString(payload, "country", "country_name"),
		Region:  firstGeoString(payload, "region", "region_name"),
		City:    firstGeoString(payload, "city"),
		ASN:     firstGeoString(payload, "asn", "organization_name", "organization"),
	}
	if item.Country == "" && item.Region == "" && item.City == "" && item.ASN == "" {
		item.Status = "not_found"
	}
	return item, nil
}

func firstGeoString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func isPrivateUsageIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func logUsageIPGeoLookupFailed(ctx context.Context, ip string, duration time.Duration, err error) {
	if err == nil {
		return
	}
	requestID := ""
	if ctx != nil {
		if value, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(value) != "" {
			requestID = strings.TrimSpace(value)
		} else if value, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(value) != "" {
			requestID = strings.TrimSpace(value)
		}
	}
	logger.FromContext(ctx).Warn(
		"usage_ip_geo_lookup_failed",
		zap.String("component", "service.admin_usage_ip_geo"),
		zap.String("request_id", requestID),
		zap.String("ip_hash", usageIPGeoIPHash(ip)),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.String("status", "error"),
		zap.Error(err),
	)
}

func usageIPGeoIPHash(ip string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(ip)))
	return hex.EncodeToString(sum[:])[:16]
}
