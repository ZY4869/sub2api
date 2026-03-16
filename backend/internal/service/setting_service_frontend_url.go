package service

import (
	"context"
	"strings"
)

// GetFrontendURL returns the configured frontend base URL for building links
// such as password reset URLs.
//
// Priority:
//  1. `settings.frontend_url` (SettingKeyFrontendURL) when present and non-empty
//  2. `server.frontend_url` from config
func (s *SettingService) GetFrontendURL(ctx context.Context) string {
	if s == nil {
		return ""
	}
	if s.settingRepo != nil {
		if v, err := s.settingRepo.GetValue(ctx, SettingKeyFrontendURL); err == nil {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	if s.cfg != nil {
		return strings.TrimSpace(s.cfg.Server.FrontendURL)
	}
	return ""
}

