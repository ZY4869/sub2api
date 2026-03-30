package service

import (
	"net/url"
	"strings"
	"time"
)

type ResolvedUpstreamInfo struct {
	URL         string
	Host        string
	Service     string
	ProbeSource string
	Region      string
	ProbedAt    time.Time
}

func ResolveUpstreamInfo(rawURL string, defaultService string, probeSource string) ResolvedUpstreamInfo {
	info := ResolvedUpstreamInfo{
		Service:     strings.TrimSpace(defaultService),
		ProbeSource: strings.TrimSpace(probeSource),
	}
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return info
	}
	parsed, err := url.Parse(trimmedURL)
	if err != nil {
		return info
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	info.URL = strings.TrimRight(parsed.String(), "/")
	info.Host = strings.TrimSpace(parsed.Hostname())
	if info.Service == "" {
		info.Service = inferUpstreamServiceFromHost(info.Host)
	}
	return info
}

func MergeUpstreamExtra(base map[string]any, info ResolvedUpstreamInfo) map[string]any {
	if len(base) == 0 && info.URL == "" && info.Host == "" && info.Service == "" && info.ProbeSource == "" && info.Region == "" && info.ProbedAt.IsZero() {
		return nil
	}
	out := MergeStringAnyMap(base, nil)
	if out == nil {
		out = map[string]any{}
	}
	if info.URL != "" {
		out["upstream_url"] = info.URL
	}
	if info.Host != "" {
		out["upstream_host"] = info.Host
	}
	if info.Service != "" {
		out["upstream_service"] = info.Service
	}
	if info.ProbeSource != "" {
		out["upstream_probe_source"] = info.ProbeSource
	}
	if !info.ProbedAt.IsZero() {
		out["upstream_probed_at"] = info.ProbedAt.UTC().Format(time.RFC3339)
	}
	if info.Region != "" {
		out["upstream_region"] = info.Region
	}
	return out
}

func inferUpstreamServiceFromHost(host string) string {
	normalized := strings.ToLower(strings.TrimSpace(host))
	switch {
	case strings.Contains(normalized, "githubcopilot.com"), strings.Contains(normalized, "githubusercontent.com"):
		return PlatformCopilot
	case strings.Contains(normalized, "amazonaws.com"):
		return PlatformKiro
	case strings.Contains(normalized, "openai.com"):
		return PlatformOpenAI
	case strings.Contains(normalized, "anthropic.com"):
		return PlatformAnthropic
	case strings.Contains(normalized, "googleapis.com"), strings.Contains(normalized, "google.com"):
		return PlatformGemini
	case strings.Contains(normalized, "x.ai"):
		return PlatformGrok
	default:
		return ""
	}
}
