package config

import "fmt"

type Config struct {
	Server                  ServerConfig                  `mapstructure:"server"`
	Log                     LogConfig                     `mapstructure:"log"`
	CORS                    CORSConfig                    `mapstructure:"cors"`
	Security                SecurityConfig                `mapstructure:"security"`
	Billing                 BillingConfig                 `mapstructure:"billing"`
	Turnstile               TurnstileConfig               `mapstructure:"turnstile"`
	Database                DatabaseConfig                `mapstructure:"database"`
	Redis                   RedisConfig                   `mapstructure:"redis"`
	Ops                     OpsConfig                     `mapstructure:"ops"`
	JWT                     JWTConfig                     `mapstructure:"jwt"`
	Totp                    TotpConfig                    `mapstructure:"totp"`
	LinuxDo                 LinuxDoConnectConfig          `mapstructure:"linuxdo_connect"`
	Default                 DefaultConfig                 `mapstructure:"default"`
	RateLimit               RateLimitConfig               `mapstructure:"rate_limit"`
	Pricing                 PricingConfig                 `mapstructure:"pricing"`
	Gateway                 GatewayConfig                 `mapstructure:"gateway"`
	APIKeyAuth              APIKeyAuthCacheConfig         `mapstructure:"api_key_auth_cache"`
	SubscriptionCache       SubscriptionCacheConfig       `mapstructure:"subscription_cache"`
	SubscriptionMaintenance SubscriptionMaintenanceConfig `mapstructure:"subscription_maintenance"`
	Dashboard               DashboardCacheConfig          `mapstructure:"dashboard_cache"`
	DashboardAgg            DashboardAggregationConfig    `mapstructure:"dashboard_aggregation"`
	UsageCleanup            UsageCleanupConfig            `mapstructure:"usage_cleanup"`
	Concurrency             ConcurrencyConfig             `mapstructure:"concurrency"`
	TokenRefresh            TokenRefreshConfig            `mapstructure:"token_refresh"`
	RunMode                 string                        `mapstructure:"run_mode" yaml:"run_mode"`
	Timezone                string                        `mapstructure:"timezone"`
	Gemini                  GeminiConfig                  `mapstructure:"gemini"`
	Update                  UpdateConfig                  `mapstructure:"update"`
	Idempotency             IdempotencyConfig             `mapstructure:"idempotency"`
}
type LogConfig struct {
	Level           string            `mapstructure:"level"`
	Format          string            `mapstructure:"format"`
	ServiceName     string            `mapstructure:"service_name"`
	Environment     string            `mapstructure:"env"`
	Caller          bool              `mapstructure:"caller"`
	StacktraceLevel string            `mapstructure:"stacktrace_level"`
	Output          LogOutputConfig   `mapstructure:"output"`
	Rotation        LogRotationConfig `mapstructure:"rotation"`
	Sampling        LogSamplingConfig `mapstructure:"sampling"`
}
type LogOutputConfig struct {
	ToStdout bool   `mapstructure:"to_stdout"`
	ToFile   bool   `mapstructure:"to_file"`
	FilePath string `mapstructure:"file_path"`
}
type LogRotationConfig struct {
	MaxSizeMB  int  `mapstructure:"max_size_mb"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAgeDays int  `mapstructure:"max_age_days"`
	Compress   bool `mapstructure:"compress"`
	LocalTime  bool `mapstructure:"local_time"`
}
type LogSamplingConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	Initial    int  `mapstructure:"initial"`
	Thereafter int  `mapstructure:"thereafter"`
}
type GeminiConfig struct {
	OAuth GeminiOAuthConfig `mapstructure:"oauth"`
	Quota GeminiQuotaConfig `mapstructure:"quota"`
}
type GeminiOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	Scopes       string `mapstructure:"scopes"`
}
type GeminiQuotaConfig struct {
	Tiers  map[string]GeminiTierQuotaConfig `mapstructure:"tiers"`
	Policy string                           `mapstructure:"policy"`
}
type GeminiTierQuotaConfig struct {
	ProRPD          *int64 `mapstructure:"pro_rpd" json:"pro_rpd"`
	FlashRPD        *int64 `mapstructure:"flash_rpd" json:"flash_rpd"`
	CooldownMinutes *int   `mapstructure:"cooldown_minutes" json:"cooldown_minutes"`
}
type UpdateConfig struct {
	ProxyURL string `mapstructure:"proxy_url"`
}
type IdempotencyConfig struct {
	ObserveOnly               bool `mapstructure:"observe_only"`
	DefaultTTLSeconds         int  `mapstructure:"default_ttl_seconds"`
	SystemOperationTTLSeconds int  `mapstructure:"system_operation_ttl_seconds"`
	ProcessingTimeoutSeconds  int  `mapstructure:"processing_timeout_seconds"`
	FailedRetryBackoffSeconds int  `mapstructure:"failed_retry_backoff_seconds"`
	MaxStoredResponseLen      int  `mapstructure:"max_stored_response_len"`
	CleanupIntervalSeconds    int  `mapstructure:"cleanup_interval_seconds"`
	CleanupBatchSize          int  `mapstructure:"cleanup_batch_size"`
}
type LinuxDoConnectConfig struct {
	Enabled              bool   `mapstructure:"enabled"`
	ClientID             string `mapstructure:"client_id"`
	ClientSecret         string `mapstructure:"client_secret"`
	AuthorizeURL         string `mapstructure:"authorize_url"`
	TokenURL             string `mapstructure:"token_url"`
	UserInfoURL          string `mapstructure:"userinfo_url"`
	Scopes               string `mapstructure:"scopes"`
	RedirectURL          string `mapstructure:"redirect_url"`
	FrontendRedirectURL  string `mapstructure:"frontend_redirect_url"`
	TokenAuthMethod      string `mapstructure:"token_auth_method"`
	UsePKCE              bool   `mapstructure:"use_pkce"`
	UserInfoEmailPath    string `mapstructure:"userinfo_email_path"`
	UserInfoIDPath       string `mapstructure:"userinfo_id_path"`
	UserInfoUsernamePath string `mapstructure:"userinfo_username_path"`
}
type TokenRefreshConfig struct {
	Enabled                  bool    `mapstructure:"enabled"`
	CheckIntervalMinutes     int     `mapstructure:"check_interval_minutes"`
	RefreshBeforeExpiryHours float64 `mapstructure:"refresh_before_expiry_hours"`
	MaxRetries               int     `mapstructure:"max_retries"`
	RetryBackoffSeconds      int     `mapstructure:"retry_backoff_seconds"`
}
type PricingConfig struct {
	RemoteURL                string `mapstructure:"remote_url"`
	HashURL                  string `mapstructure:"hash_url"`
	DataDir                  string `mapstructure:"data_dir"`
	FallbackFile             string `mapstructure:"fallback_file"`
	UpdateIntervalHours      int    `mapstructure:"update_interval_hours"`
	HashCheckIntervalMinutes int    `mapstructure:"hash_check_interval_minutes"`
}
type ServerConfig struct {
	Host               string    `mapstructure:"host"`
	Port               int       `mapstructure:"port"`
	Mode               string    `mapstructure:"mode"`
	FrontendURL        string    `mapstructure:"frontend_url"`
	ReadHeaderTimeout  int       `mapstructure:"read_header_timeout"`
	IdleTimeout        int       `mapstructure:"idle_timeout"`
	TrustedProxies     []string  `mapstructure:"trusted_proxies"`
	MaxRequestBodySize int64     `mapstructure:"max_request_body_size"`
	H2C                H2CConfig `mapstructure:"h2c"`
}
type H2CConfig struct {
	Enabled                      bool   `mapstructure:"enabled"`
	MaxConcurrentStreams         uint32 `mapstructure:"max_concurrent_streams"`
	IdleTimeout                  int    `mapstructure:"idle_timeout"`
	MaxReadFrameSize             int    `mapstructure:"max_read_frame_size"`
	MaxUploadBufferPerConnection int    `mapstructure:"max_upload_buffer_per_connection"`
	MaxUploadBufferPerStream     int    `mapstructure:"max_upload_buffer_per_stream"`
}
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}
type SecurityConfig struct {
	URLAllowlist    URLAllowlistConfig   `mapstructure:"url_allowlist"`
	ResponseHeaders ResponseHeaderConfig `mapstructure:"response_headers"`
	CSP             CSPConfig            `mapstructure:"csp"`
	ProxyFallback   ProxyFallbackConfig  `mapstructure:"proxy_fallback"`
	ProxyProbe      ProxyProbeConfig     `mapstructure:"proxy_probe"`
}
type URLAllowlistConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	UpstreamHosts     []string `mapstructure:"upstream_hosts"`
	PricingHosts      []string `mapstructure:"pricing_hosts"`
	CRSHosts          []string `mapstructure:"crs_hosts"`
	AllowPrivateHosts bool     `mapstructure:"allow_private_hosts"`
	AllowInsecureHTTP bool     `mapstructure:"allow_insecure_http"`
}
type ResponseHeaderConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	AdditionalAllowed []string `mapstructure:"additional_allowed"`
	ForceRemove       []string `mapstructure:"force_remove"`
}
type CSPConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Policy  string `mapstructure:"policy"`
}
type ProxyFallbackConfig struct {
	AllowDirectOnError bool `mapstructure:"allow_direct_on_error"`
}
type ProxyProbeConfig struct {
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}
type BillingConfig struct {
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}
type CircuitBreakerConfig struct {
	Enabled             bool `mapstructure:"enabled"`
	FailureThreshold    int  `mapstructure:"failure_threshold"`
	ResetTimeoutSeconds int  `mapstructure:"reset_timeout_seconds"`
	HalfOpenRequests    int  `mapstructure:"half_open_requests"`
}
type ConcurrencyConfig struct {
	PingInterval int `mapstructure:"ping_interval"`
}
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	User                   string `mapstructure:"user"`
	Password               string `mapstructure:"password"`
	DBName                 string `mapstructure:"dbname"`
	SSLMode                string `mapstructure:"sslmode"`
	MaxOpenConns           int    `mapstructure:"max_open_conns"`
	MaxIdleConns           int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `mapstructure:"conn_max_lifetime_minutes"`
	ConnMaxIdleTimeMinutes int    `mapstructure:"conn_max_idle_time_minutes"`
}

func (d *DatabaseConfig) DSN() string {
	if d.Password == "" {
		return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s", d.Host, d.Port, d.User, d.DBName, d.SSLMode)
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}
func (d *DatabaseConfig) DSNWithTimezone(tz string) string {
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	if d.Password == "" {
		return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s TimeZone=%s", d.Host, d.Port, d.User, d.DBName, d.SSLMode, tz)
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s", d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode, tz)
}

type RedisConfig struct {
	Host                string `mapstructure:"host"`
	Port                int    `mapstructure:"port"`
	Password            string `mapstructure:"password"`
	DB                  int    `mapstructure:"db"`
	DialTimeoutSeconds  int    `mapstructure:"dial_timeout_seconds"`
	ReadTimeoutSeconds  int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `mapstructure:"write_timeout_seconds"`
	PoolSize            int    `mapstructure:"pool_size"`
	MinIdleConns        int    `mapstructure:"min_idle_conns"`
	EnableTLS           bool   `mapstructure:"enable_tls"`
}
