package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *OpsAlertEvaluatorService) maybeSendAlertEmail(ctx context.Context, runtimeCfg *OpsAlertRuntimeSettings, rule *OpsAlertRule, event *OpsAlertEvent) bool {
	if s == nil || s.emailService == nil || s.opsService == nil || event == nil || rule == nil {
		return false
	}
	if event.EmailSent {
		return false
	}
	if !rule.NotifyEmail {
		return false
	}

	emailCfg, err := s.opsService.GetEmailNotificationConfig(ctx)
	if err != nil || emailCfg == nil || !emailCfg.Alert.Enabled {
		return false
	}

	if len(emailCfg.Alert.Recipients) == 0 {
		return false
	}
	if !shouldSendOpsAlertEmailByMinSeverity(strings.TrimSpace(emailCfg.Alert.MinSeverity), strings.TrimSpace(rule.Severity)) {
		return false
	}

	if runtimeCfg != nil && runtimeCfg.Silencing.Enabled {
		if isOpsAlertSilenced(time.Now().UTC(), rule, event, runtimeCfg.Silencing) {
			return false
		}
	}

	s.emailLimiter.SetLimit(emailCfg.Alert.RateLimitPerHour)

	subject := fmt.Sprintf("[Ops Alert][%s] %s", strings.TrimSpace(rule.Severity), strings.TrimSpace(rule.Name))
	body := buildOpsAlertEmailBody(rule, event)

	anySent := false
	for _, to := range emailCfg.Alert.Recipients {
		addr := strings.TrimSpace(to)
		if addr == "" {
			continue
		}
		if !s.emailLimiter.Allow(time.Now().UTC()) {
			continue
		}
		if err := s.emailService.SendEmail(ctx, addr, subject, body); err != nil {
			continue
		}
		anySent = true
	}

	if anySent {
		_ = s.opsRepo.UpdateAlertEventEmailSent(context.Background(), event.ID, true)
	}
	return anySent
}

func buildOpsAlertEmailBody(rule *OpsAlertRule, event *OpsAlertEvent) string {
	if rule == nil || event == nil {
		return ""
	}
	metric := strings.TrimSpace(rule.MetricType)
	value := "-"
	threshold := fmt.Sprintf("%.2f", rule.Threshold)
	if event.MetricValue != nil {
		value = fmt.Sprintf("%.2f", *event.MetricValue)
	}
	if event.ThresholdValue != nil {
		threshold = fmt.Sprintf("%.2f", *event.ThresholdValue)
	}
	return fmt.Sprintf(`
<h2>Ops Alert</h2>
<p><b>Rule</b>: %s</p>
<p><b>Severity</b>: %s</p>
<p><b>Status</b>: %s</p>
<p><b>Metric</b>: %s %s %s</p>
<p><b>Fired at</b>: %s</p>
<p><b>Description</b>: %s</p>
`,
		htmlEscape(rule.Name),
		htmlEscape(rule.Severity),
		htmlEscape(event.Status),
		htmlEscape(metric),
		htmlEscape(rule.Operator),
		htmlEscape(fmt.Sprintf("%s (threshold %s)", value, threshold)),
		event.FiredAt.Format(time.RFC3339),
		htmlEscape(event.Description),
	)
}

func shouldSendOpsAlertEmailByMinSeverity(minSeverity string, ruleSeverity string) bool {
	minSeverity = strings.ToLower(strings.TrimSpace(minSeverity))
	if minSeverity == "" {
		return true
	}

	eventLevel := opsEmailSeverityForOps(ruleSeverity)
	minLevel := strings.ToLower(minSeverity)

	rank := func(level string) int {
		switch level {
		case "critical":
			return 3
		case "warning":
			return 2
		case "info":
			return 1
		default:
			return 0
		}
	}
	return rank(eventLevel) >= rank(minLevel)
}

func opsEmailSeverityForOps(severity string) string {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "P0":
		return "critical"
	case "P1":
		return "warning"
	default:
		return "info"
	}
}

func isOpsAlertSilenced(now time.Time, rule *OpsAlertRule, event *OpsAlertEvent, silencing OpsAlertSilencingSettings) bool {
	if !silencing.Enabled {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if strings.TrimSpace(silencing.GlobalUntilRFC3339) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(silencing.GlobalUntilRFC3339)); err == nil {
			if now.Before(t) {
				return true
			}
		}
	}

	for _, entry := range silencing.Entries {
		untilRaw := strings.TrimSpace(entry.UntilRFC3339)
		if untilRaw == "" {
			continue
		}
		until, err := time.Parse(time.RFC3339, untilRaw)
		if err != nil {
			continue
		}
		if now.After(until) {
			continue
		}
		if entry.RuleID != nil && rule != nil && rule.ID > 0 && *entry.RuleID != rule.ID {
			continue
		}
		if len(entry.Severities) > 0 {
			match := false
			for _, s := range entry.Severities {
				if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(event.Severity)) || strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(rule.Severity)) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		return true
	}

	return false
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}
