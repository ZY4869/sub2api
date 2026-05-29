package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

func hasGeminiCustomOAuthClient(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	clientID := strings.TrimSpace(cfg.Gemini.OAuth.ClientID)
	clientSecret := strings.TrimSpace(cfg.Gemini.OAuth.ClientSecret)
	return clientID != "" && clientSecret != "" && clientID != geminicli.GeminiCLIOAuthClientID
}

func buildGeminiOAuthConfigInput(cfg *config.Config, oauthType string) geminicli.OAuthConfig {
	var oauthCfg geminicli.OAuthConfig
	if cfg != nil {
		oauthCfg = geminicli.OAuthConfig{
			ClientID:     cfg.Gemini.OAuth.ClientID,
			ClientSecret: cfg.Gemini.OAuth.ClientSecret,
			Scopes:       cfg.Gemini.OAuth.Scopes,
		}
	}
	if oauthType == "code_assist" {
		oauthCfg.ClientID = ""
		oauthCfg.ClientSecret = ""
	}
	return oauthCfg
}

func geminiScopeContains(scope string, prefixes ...string) bool {
	for _, item := range strings.Fields(scope) {
		for _, prefix := range prefixes {
			if prefix != "" && strings.HasPrefix(item, prefix) {
				return true
			}
		}
	}
	return false
}

func googleOneDriveProbeScope(scope string, fallback string) string {
	scope = strings.TrimSpace(scope)
	if scope != "" {
		return scope
	}
	return strings.TrimSpace(fallback)
}

func canProbeGoogleOneDriveTier(scope string) bool {
	return geminiScopeContains(scope,
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/drive",
	)
}
