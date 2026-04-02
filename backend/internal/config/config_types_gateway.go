package config

import "time"

type GatewayConfig struct {
	ResponseHeaderTimeout                int                      `mapstructure:"response_header_timeout"`
	MaxBodySize                          int64                    `mapstructure:"max_body_size"`
	UpstreamResponseReadMaxBytes         int64                    `mapstructure:"upstream_response_read_max_bytes"`
	ProxyProbeResponseReadMaxBytes       int64                    `mapstructure:"proxy_probe_response_read_max_bytes"`
	GeminiDebugResponseHeaders           bool                     `mapstructure:"gemini_debug_response_headers"`
	ConnectionPoolIsolation              string                   `mapstructure:"connection_pool_isolation"`
	ForceCodexCLI                        bool                     `mapstructure:"force_codex_cli"`
	OpenAIPassthroughAllowTimeoutHeaders bool                     `mapstructure:"openai_passthrough_allow_timeout_headers"`
	OpenAIWS                             GatewayOpenAIWSConfig    `mapstructure:"openai_ws"`
	MaxIdleConns                         int                      `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost                  int                      `mapstructure:"max_idle_conns_per_host"`
	MaxConnsPerHost                      int                      `mapstructure:"max_conns_per_host"`
	IdleConnTimeoutSeconds               int                      `mapstructure:"idle_conn_timeout_seconds"`
	MaxUpstreamClients                   int                      `mapstructure:"max_upstream_clients"`
	ClientIdleTTLSeconds                 int                      `mapstructure:"client_idle_ttl_seconds"`
	ConcurrencySlotTTLMinutes            int                      `mapstructure:"concurrency_slot_ttl_minutes"`
	SessionIdleTimeoutMinutes            int                      `mapstructure:"session_idle_timeout_minutes"`
	StreamDataIntervalTimeout            int                      `mapstructure:"stream_data_interval_timeout"`
	StreamKeepaliveInterval              int                      `mapstructure:"stream_keepalive_interval"`
	MaxLineSize                          int                      `mapstructure:"max_line_size"`
	LogUpstreamErrorBody                 bool                     `mapstructure:"log_upstream_error_body"`
	LogUpstreamErrorBodyMaxBytes         int                      `mapstructure:"log_upstream_error_body_max_bytes"`
	InjectBetaForAPIKey                  bool                     `mapstructure:"inject_beta_for_apikey"`
	FailoverOn400                        bool                     `mapstructure:"failover_on_400"`
	SoraMaxBodySize                      int64                    `mapstructure:"sora_max_body_size"`
	GrokVideoPollIntervalSeconds         int                      `mapstructure:"grok_video_poll_interval_seconds"`
	GrokVideoWaitTimeoutSeconds          int                      `mapstructure:"grok_video_wait_timeout_seconds"`
	SoraStreamTimeoutSeconds             int                      `mapstructure:"sora_stream_timeout_seconds"`
	SoraRequestTimeoutSeconds            int                      `mapstructure:"sora_request_timeout_seconds"`
	SoraStreamMode                       string                   `mapstructure:"sora_stream_mode"`
	SoraModelFilters                     SoraModelFiltersConfig   `mapstructure:"sora_model_filters"`
	SoraMediaRequireAPIKey               bool                     `mapstructure:"sora_media_require_api_key"`
	SoraMediaSigningKey                  string                   `mapstructure:"sora_media_signing_key"`
	SoraMediaSignedURLTTLSeconds         int                      `mapstructure:"sora_media_signed_url_ttl_seconds"`
	MaxAccountSwitches                   int                      `mapstructure:"max_account_switches"`
	MaxAccountSwitchesGemini             int                      `mapstructure:"max_account_switches_gemini"`
	AntigravityFallbackCooldownMinutes   int                      `mapstructure:"antigravity_fallback_cooldown_minutes"`
	Scheduling                           GatewaySchedulingConfig  `mapstructure:"scheduling"`
	TLSFingerprint                       TLSFingerprintConfig     `mapstructure:"tls_fingerprint"`
	UsageRecord                          GatewayUsageRecordConfig `mapstructure:"usage_record"`
	UserGroupRateCacheTTLSeconds         int                      `mapstructure:"user_group_rate_cache_ttl_seconds"`
	ModelsListCacheTTLSeconds            int                      `mapstructure:"models_list_cache_ttl_seconds"`
	UserMessageQueue                     UserMessageQueueConfig   `mapstructure:"user_message_queue"`
}
type UserMessageQueueConfig struct {
	Mode                   string `mapstructure:"mode"`
	Enabled                bool   `mapstructure:"enabled"`
	LockTTLMs              int    `mapstructure:"lock_ttl_ms"`
	WaitTimeoutMs          int    `mapstructure:"wait_timeout_ms"`
	MinDelayMs             int    `mapstructure:"min_delay_ms"`
	MaxDelayMs             int    `mapstructure:"max_delay_ms"`
	CleanupIntervalSeconds int    `mapstructure:"cleanup_interval_seconds"`
}

func (c *UserMessageQueueConfig) WaitTimeout() time.Duration {
	if c.WaitTimeoutMs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.WaitTimeoutMs) * time.Millisecond
}
func (c *UserMessageQueueConfig) GetEffectiveMode() string {
	if c.Mode == UMQModeSerialize || c.Mode == UMQModeThrottle {
		return c.Mode
	}
	if c.Enabled {
		return UMQModeSerialize
	}
	return ""
}

type GatewayOpenAIWSConfig struct {
	ModeRouterV2Enabled                        bool                                 `mapstructure:"mode_router_v2_enabled"`
	IngressModeDefault                         string                               `mapstructure:"ingress_mode_default"`
	Enabled                                    bool                                 `mapstructure:"enabled"`
	OAuthEnabled                               bool                                 `mapstructure:"oauth_enabled"`
	APIKeyEnabled                              bool                                 `mapstructure:"apikey_enabled"`
	ForceHTTP                                  bool                                 `mapstructure:"force_http"`
	AllowStoreRecovery                         bool                                 `mapstructure:"allow_store_recovery"`
	IngressPreviousResponseRecoveryEnabled     bool                                 `mapstructure:"ingress_previous_response_recovery_enabled"`
	StoreDisabledConnMode                      string                               `mapstructure:"store_disabled_conn_mode"`
	StoreDisabledForceNewConn                  bool                                 `mapstructure:"store_disabled_force_new_conn"`
	PrewarmGenerateEnabled                     bool                                 `mapstructure:"prewarm_generate_enabled"`
	ResponsesWebsockets                        bool                                 `mapstructure:"responses_websockets"`
	ResponsesWebsocketsV2                      bool                                 `mapstructure:"responses_websockets_v2"`
	MaxConnsPerAccount                         int                                  `mapstructure:"max_conns_per_account"`
	MinIdlePerAccount                          int                                  `mapstructure:"min_idle_per_account"`
	MaxIdlePerAccount                          int                                  `mapstructure:"max_idle_per_account"`
	DynamicMaxConnsByAccountConcurrencyEnabled bool                                 `mapstructure:"dynamic_max_conns_by_account_concurrency_enabled"`
	OAuthMaxConnsFactor                        float64                              `mapstructure:"oauth_max_conns_factor"`
	APIKeyMaxConnsFactor                       float64                              `mapstructure:"apikey_max_conns_factor"`
	DialTimeoutSeconds                         int                                  `mapstructure:"dial_timeout_seconds"`
	ReadTimeoutSeconds                         int                                  `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds                        int                                  `mapstructure:"write_timeout_seconds"`
	PoolTargetUtilization                      float64                              `mapstructure:"pool_target_utilization"`
	QueueLimitPerConn                          int                                  `mapstructure:"queue_limit_per_conn"`
	EventFlushBatchSize                        int                                  `mapstructure:"event_flush_batch_size"`
	EventFlushIntervalMS                       int                                  `mapstructure:"event_flush_interval_ms"`
	PrewarmCooldownMS                          int                                  `mapstructure:"prewarm_cooldown_ms"`
	FallbackCooldownSeconds                    int                                  `mapstructure:"fallback_cooldown_seconds"`
	RetryBackoffInitialMS                      int                                  `mapstructure:"retry_backoff_initial_ms"`
	RetryBackoffMaxMS                          int                                  `mapstructure:"retry_backoff_max_ms"`
	RetryJitterRatio                           float64                              `mapstructure:"retry_jitter_ratio"`
	RetryTotalBudgetMS                         int                                  `mapstructure:"retry_total_budget_ms"`
	PayloadLogSampleRate                       float64                              `mapstructure:"payload_log_sample_rate"`
	LBTopK                                     int                                  `mapstructure:"lb_top_k"`
	StickySessionTTLSeconds                    int                                  `mapstructure:"sticky_session_ttl_seconds"`
	SessionHashReadOldFallback                 bool                                 `mapstructure:"session_hash_read_old_fallback"`
	SessionHashDualWriteOld                    bool                                 `mapstructure:"session_hash_dual_write_old"`
	MetadataBridgeEnabled                      bool                                 `mapstructure:"metadata_bridge_enabled"`
	StickyResponseIDTTLSeconds                 int                                  `mapstructure:"sticky_response_id_ttl_seconds"`
	StickyPreviousResponseTTLSeconds           int                                  `mapstructure:"sticky_previous_response_ttl_seconds"`
	SchedulerScoreWeights                      GatewayOpenAIWSSchedulerScoreWeights `mapstructure:"scheduler_score_weights"`
}
type GatewayOpenAIWSSchedulerScoreWeights struct {
	Priority  float64 `mapstructure:"priority"`
	Load      float64 `mapstructure:"load"`
	Queue     float64 `mapstructure:"queue"`
	ErrorRate float64 `mapstructure:"error_rate"`
	TTFT      float64 `mapstructure:"ttft"`
}
type GatewayUsageRecordConfig struct {
	WorkerCount                   int    `mapstructure:"worker_count"`
	QueueSize                     int    `mapstructure:"queue_size"`
	TaskTimeoutSeconds            int    `mapstructure:"task_timeout_seconds"`
	OverflowPolicy                string `mapstructure:"overflow_policy"`
	OverflowSamplePercent         int    `mapstructure:"overflow_sample_percent"`
	AutoScaleEnabled              bool   `mapstructure:"auto_scale_enabled"`
	AutoScaleMinWorkers           int    `mapstructure:"auto_scale_min_workers"`
	AutoScaleMaxWorkers           int    `mapstructure:"auto_scale_max_workers"`
	AutoScaleUpQueuePercent       int    `mapstructure:"auto_scale_up_queue_percent"`
	AutoScaleDownQueuePercent     int    `mapstructure:"auto_scale_down_queue_percent"`
	AutoScaleUpStep               int    `mapstructure:"auto_scale_up_step"`
	AutoScaleDownStep             int    `mapstructure:"auto_scale_down_step"`
	AutoScaleCheckIntervalSeconds int    `mapstructure:"auto_scale_check_interval_seconds"`
	AutoScaleCooldownSeconds      int    `mapstructure:"auto_scale_cooldown_seconds"`
}
type SoraModelFiltersConfig struct {
	HidePromptEnhance bool `mapstructure:"hide_prompt_enhance"`
}
type TLSFingerprintConfig struct {
	Enabled  bool                        `mapstructure:"enabled"`
	Profiles map[string]TLSProfileConfig `mapstructure:"profiles"`
}
type TLSProfileConfig struct {
	Name                string   `mapstructure:"name"`
	EnableGREASE        bool     `mapstructure:"enable_grease"`
	CipherSuites        []uint16 `mapstructure:"cipher_suites"`
	Curves              []uint16 `mapstructure:"curves"`
	PointFormats        []uint16 `mapstructure:"point_formats"`
	SignatureAlgorithms []uint16 `mapstructure:"signature_algorithms"`
	ALPNProtocols       []string `mapstructure:"alpn_protocols"`
	SupportedVersions   []uint16 `mapstructure:"supported_versions"`
	KeyShareGroups      []uint16 `mapstructure:"key_share_groups"`
	PSKModes            []uint16 `mapstructure:"psk_modes"`
	Extensions          []uint16 `mapstructure:"extensions"`
}
type GatewaySchedulingConfig struct {
	StickySessionMaxWaiting    int           `mapstructure:"sticky_session_max_waiting"`
	StickySessionWaitTimeout   time.Duration `mapstructure:"sticky_session_wait_timeout"`
	FallbackWaitTimeout        time.Duration `mapstructure:"fallback_wait_timeout"`
	FallbackMaxWaiting         int           `mapstructure:"fallback_max_waiting"`
	FallbackSelectionMode      string        `mapstructure:"fallback_selection_mode"`
	LoadBatchEnabled           bool          `mapstructure:"load_batch_enabled"`
	SlotCleanupInterval        time.Duration `mapstructure:"slot_cleanup_interval"`
	DbFallbackEnabled          bool          `mapstructure:"db_fallback_enabled"`
	DbFallbackTimeoutSeconds   int           `mapstructure:"db_fallback_timeout_seconds"`
	DbFallbackMaxQPS           int           `mapstructure:"db_fallback_max_qps"`
	OutboxPollIntervalSeconds  int           `mapstructure:"outbox_poll_interval_seconds"`
	OutboxLagWarnSeconds       int           `mapstructure:"outbox_lag_warn_seconds"`
	OutboxLagRebuildSeconds    int           `mapstructure:"outbox_lag_rebuild_seconds"`
	OutboxLagRebuildFailures   int           `mapstructure:"outbox_lag_rebuild_failures"`
	OutboxBacklogRebuildRows   int           `mapstructure:"outbox_backlog_rebuild_rows"`
	FullRebuildIntervalSeconds int           `mapstructure:"full_rebuild_interval_seconds"`
}
