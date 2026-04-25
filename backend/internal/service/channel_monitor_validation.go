package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

func validateChannelMonitorEndpointForSave(cfg *config.Config, endpoint string, requireAllowlist bool) (string, error) {
	if cfg == nil {
		return "", ErrChannelMonitorInvalidEndpoint
	}

	allowInsecure := cfg.Security.URLAllowlist.AllowInsecureHTTP
	opts := urlvalidator.ValidationOptions{
		AllowedHosts:     cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: requireAllowlist,
		AllowPrivate:     cfg.Security.URLAllowlist.AllowPrivateHosts,
	}

	if cfg.Security.URLAllowlist.Enabled {
		out, err := urlvalidator.ValidateHTTPURL(endpoint, allowInsecure, opts)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not allowed") {
				return "", ErrChannelMonitorEndpointNotAllowed
			}
			return "", ErrChannelMonitorInvalidEndpoint
		}
		return out, nil
	}

	out, err := urlvalidator.ValidateURLFormat(endpoint, allowInsecure)
	if err != nil {
		return "", ErrChannelMonitorInvalidEndpoint
	}
	return out, nil
}

func ensureNextRunAtOnEnable(m *ChannelMonitor, now time.Time) {
	if m == nil || !m.Enabled {
		return
	}
	if m.NextRunAt == nil {
		v := now
		m.NextRunAt = &v
		return
	}
	if m.NextRunAt.Before(now) {
		v := now
		m.NextRunAt = &v
	}
}

func normalizeChannelMonitor(m *ChannelMonitor) (*ChannelMonitor, error) {
	if m == nil {
		return nil, errors.New("nil monitor")
	}

	out := *m
	out.Name = strings.TrimSpace(out.Name)
	out.Provider = strings.TrimSpace(strings.ToLower(out.Provider))
	out.Endpoint = strings.TrimSpace(out.Endpoint)
	out.PrimaryModelID = strings.TrimSpace(out.PrimaryModelID)
	out.BodyOverrideMode = strings.TrimSpace(strings.ToLower(out.BodyOverrideMode))

	if out.Name == "" || len(out.Name) > 100 {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_NAME_INVALID", "invalid name")
	}
	if !isValidChannelMonitorProvider(out.Provider) {
		return nil, ErrChannelMonitorInvalidProvider
	}

	if out.IntervalSeconds < channelMonitorMinIntervalSeconds || out.IntervalSeconds > channelMonitorMaxIntervalSeconds {
		return nil, ErrChannelMonitorInvalidInterval
	}

	if out.PrimaryModelID == "" {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_PRIMARY_MODEL_REQUIRED", "primary_model_id is required")
	}

	if out.BodyOverrideMode == "" {
		out.BodyOverrideMode = ChannelMonitorBodyOverrideModeOff
	}
	if !isValidChannelMonitorBodyOverrideMode(out.BodyOverrideMode) {
		return nil, ErrChannelMonitorInvalidOverrideMode
	}

	out.ExtraHeaders = normalizeChannelMonitorHeaders(out.ExtraHeaders)
	out.BodyOverride = ensureAnyMap(out.BodyOverride)
	if out.BodyOverrideMode == ChannelMonitorBodyOverrideModeReplace && len(out.BodyOverride) == 0 {
		return nil, ErrChannelMonitorInvalidBodyOverride
	}

	out.AdditionalModelIDs = normalizeModelIDList(out.AdditionalModelIDs, out.PrimaryModelID)
	return &out, nil
}

func normalizeModelIDList(values []string, primary string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" || v == primary {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func normalizeChannelMonitorHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for k, v := range headers {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		lk := strings.ToLower(key)
		// 强制由系统控制鉴权头，避免错误配置导致泄露或意外行为。
		if lk == "authorization" || lk == "x-api-key" || lk == "x-goog-api-key" {
			continue
		}
		out[key] = val
	}
	return out
}

func isValidChannelMonitorProvider(provider string) bool {
	switch provider {
	case ChannelMonitorProviderOpenAI,
		ChannelMonitorProviderAnthropic,
		ChannelMonitorProviderGemini,
		ChannelMonitorProviderGrok,
		ChannelMonitorProviderAntigravity:
		return true
	default:
		return false
	}
}

func isValidChannelMonitorBodyOverrideMode(mode string) bool {
	switch mode {
	case ChannelMonitorBodyOverrideModeOff,
		ChannelMonitorBodyOverrideModeMerge,
		ChannelMonitorBodyOverrideModeReplace:
		return true
	default:
		return false
	}
}

func ensureAnyMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func utcDayStart(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	u := now.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func channelMonitorStartDay(now time.Time, days int) time.Time {
	if days <= 1 {
		return utcDayStart(now)
	}
	return utcDayStart(now).AddDate(0, 0, -(days - 1))
}

func channelMonitorRequireEnabled(ctx context.Context, settingSvc *SettingService) bool {
	if settingSvc == nil {
		return false
	}
	runtime, err := settingSvc.GetChannelMonitorRuntime(ctx)
	if err != nil || runtime == nil {
		return false
	}
	return runtime.Enabled
}
