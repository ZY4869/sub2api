package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strings"
)

func Load() (*Config, error) {
	return load(false)
}
func LoadForBootstrap() (*Config, error) {
	return load(true)
}
func load(allowMissingJWTSecret bool) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	if dataDir := os.Getenv("DATA_DIR"); dataDir != "" {
		viper.AddConfigPath(dataDir)
	}
	viper.AddConfigPath("/app/data")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/sub2api")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setDefaults()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config error: %w", err)
		}
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config error: %w", err)
	}
	cfg.RunMode = NormalizeRunMode(cfg.RunMode)
	cfg.Server.Mode = strings.ToLower(strings.TrimSpace(cfg.Server.Mode))
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	cfg.Server.FrontendURL = strings.TrimSpace(cfg.Server.FrontendURL)
	cfg.JWT.Secret = strings.TrimSpace(cfg.JWT.Secret)
	cfg.LinuxDo.ClientID = strings.TrimSpace(cfg.LinuxDo.ClientID)
	cfg.LinuxDo.ClientSecret = strings.TrimSpace(cfg.LinuxDo.ClientSecret)
	cfg.LinuxDo.AuthorizeURL = strings.TrimSpace(cfg.LinuxDo.AuthorizeURL)
	cfg.LinuxDo.TokenURL = strings.TrimSpace(cfg.LinuxDo.TokenURL)
	cfg.LinuxDo.UserInfoURL = strings.TrimSpace(cfg.LinuxDo.UserInfoURL)
	cfg.LinuxDo.Scopes = strings.TrimSpace(cfg.LinuxDo.Scopes)
	cfg.LinuxDo.RedirectURL = strings.TrimSpace(cfg.LinuxDo.RedirectURL)
	cfg.LinuxDo.FrontendRedirectURL = strings.TrimSpace(cfg.LinuxDo.FrontendRedirectURL)
	cfg.LinuxDo.TokenAuthMethod = strings.ToLower(strings.TrimSpace(cfg.LinuxDo.TokenAuthMethod))
	cfg.LinuxDo.UserInfoEmailPath = strings.TrimSpace(cfg.LinuxDo.UserInfoEmailPath)
	cfg.LinuxDo.UserInfoIDPath = strings.TrimSpace(cfg.LinuxDo.UserInfoIDPath)
	cfg.LinuxDo.UserInfoUsernamePath = strings.TrimSpace(cfg.LinuxDo.UserInfoUsernamePath)
	cfg.Dashboard.KeyPrefix = strings.TrimSpace(cfg.Dashboard.KeyPrefix)
	cfg.CORS.AllowedOrigins = normalizeStringSlice(cfg.CORS.AllowedOrigins)
	cfg.Security.ResponseHeaders.AdditionalAllowed = normalizeStringSlice(cfg.Security.ResponseHeaders.AdditionalAllowed)
	cfg.Security.ResponseHeaders.ForceRemove = normalizeStringSlice(cfg.Security.ResponseHeaders.ForceRemove)
	cfg.Security.CSP.Policy = strings.TrimSpace(cfg.Security.CSP.Policy)
	cfg.Log.Level = strings.ToLower(strings.TrimSpace(cfg.Log.Level))
	cfg.Log.Format = strings.ToLower(strings.TrimSpace(cfg.Log.Format))
	cfg.Log.ServiceName = strings.TrimSpace(cfg.Log.ServiceName)
	cfg.Log.Environment = strings.TrimSpace(cfg.Log.Environment)
	cfg.Log.StacktraceLevel = strings.ToLower(strings.TrimSpace(cfg.Log.StacktraceLevel))
	cfg.Log.Output.FilePath = strings.TrimSpace(cfg.Log.Output.FilePath)
	if cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds <= 0 && cfg.Gateway.OpenAIWS.StickyPreviousResponseTTLSeconds > 0 {
		cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = cfg.Gateway.OpenAIWS.StickyPreviousResponseTTLSeconds
	}
	if m := cfg.Gateway.UserMessageQueue.Mode; m != "" && m != UMQModeSerialize && m != UMQModeThrottle {
		slog.Warn("invalid user_message_queue mode, disabling", "mode", m, "valid_modes", []string{UMQModeSerialize, UMQModeThrottle})
		cfg.Gateway.UserMessageQueue.Mode = ""
	}
	cfg.Totp.EncryptionKey = strings.TrimSpace(cfg.Totp.EncryptionKey)
	if cfg.Totp.EncryptionKey == "" {
		key, err := generateJWTSecret(32)
		if err != nil {
			return nil, fmt.Errorf("generate totp encryption key error: %w", err)
		}
		cfg.Totp.EncryptionKey = key
		cfg.Totp.EncryptionKeyConfigured = false
		slog.Warn("TOTP encryption key auto-generated. Consider setting a fixed key for production.")
	} else {
		cfg.Totp.EncryptionKeyConfigured = true
	}
	originalJWTSecret := cfg.JWT.Secret
	if allowMissingJWTSecret && originalJWTSecret == "" {
		cfg.JWT.Secret = strings.Repeat("0", 32)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config error: %w", err)
	}
	if allowMissingJWTSecret && originalJWTSecret == "" {
		cfg.JWT.Secret = ""
	}
	if !cfg.Security.URLAllowlist.Enabled {
		slog.Warn("security.url_allowlist.enabled=false; allowlist/SSRF checks disabled (minimal format validation only).")
	}
	if !cfg.Security.ResponseHeaders.Enabled {
		slog.Warn("security.response_headers.enabled=false; configurable header filtering disabled (default allowlist only).")
	}
	if cfg.JWT.Secret != "" && isWeakJWTSecret(cfg.JWT.Secret) {
		slog.Warn("JWT secret appears weak; use a 32+ character random secret in production.")
	}
	if len(cfg.Security.ResponseHeaders.AdditionalAllowed) > 0 || len(cfg.Security.ResponseHeaders.ForceRemove) > 0 {
		slog.Info("response header policy configured", "additional_allowed", cfg.Security.ResponseHeaders.AdditionalAllowed, "force_remove", cfg.Security.ResponseHeaders.ForceRemove)
	}
	return &cfg, nil
}
