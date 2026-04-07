package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log/slog"
	"net/url"
	"strings"
)

const (
	RunModeStandard = "standard"
	RunModeSimple   = "simple"
)
const (
	UsageRecordOverflowPolicyDrop   = "drop"
	UsageRecordOverflowPolicySample = "sample"
	UsageRecordOverflowPolicySync   = "sync"
)
const DefaultCSPPolicy = "default-src 'self'; script-src 'self' __CSP_NONCE__ https://challenges.cloudflare.com https://static.cloudflareinsights.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; font-src 'self' data: https://fonts.gstatic.com; connect-src 'self' https:; frame-src https://challenges.cloudflare.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
const (
	UMQModeSerialize = "serialize"
	UMQModeThrottle  = "throttle"
)
const (
	ConnectionPoolIsolationProxy        = "proxy"
	ConnectionPoolIsolationAccount      = "account"
	ConnectionPoolIsolationAccountProxy = "account_proxy"
)

func (r *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}
func NormalizeRunMode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case RunModeStandard, RunModeSimple:
		return normalized
	default:
		return RunModeStandard
	}
}
func GetServerAddress() string {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/sub2api")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	_ = v.ReadInConfig()
	host := v.GetString("server.host")
	port := v.GetInt("server.port")
	return fmt.Sprintf("%s:%d", host, port)
}
func warnIfInsecureURL(field, raw string) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return
	}
	if strings.EqualFold(u.Scheme, "http") {
		slog.Warn("url uses http scheme; use https in production to avoid token leakage", "field", field)
	}
}
