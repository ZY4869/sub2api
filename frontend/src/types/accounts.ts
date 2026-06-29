import type { Group, OpenAIImageProtocolMode } from './api-key-groups'
// ==================== Account & Proxy Types ====================

export type AccountPlatform =
  | "anthropic"
  | "kiro"
  | "openai"
  | "openrouter"
  | "grok"
  | "deepseek"
  | "gemini"
  | "antigravity"
  | "protocol_gateway"
  | "baidu_document_ai";
export type AccountPlatformCountSortOrder = "count_asc" | "count_desc";
export type GatewayProtocol = "openai" | "anthropic" | "gemini" | "mixed";
export type GatewayAcceptedProtocol = "openai" | "anthropic" | "gemini";
export type GatewayClientProfile = "codex" | "gemini_cli";
export type GatewayOpenAIRequestFormat =
  | "/v1/chat/completions"
  | "/v1/responses";
export interface GatewayClientRoute {
  protocol: GatewayAcceptedProtocol;
  match_type: "exact" | "prefix";
  match_value: string;
  client_profile: GatewayClientProfile;
}
export type AccountType =
  | "oauth"
  | "setup-token"
  | "apikey"
  | "sso"
  | "bedrock"
  | "upstream";
export type AccountLifecycleState = "normal" | "archived" | "blacklisted";
export type AccountLimitedView = "all" | "normal_only" | "limited_only";
export type AccountRuntimeView = "all" | "in_use_only" | "available_only";
export type AccountRateLimitReason =
  | "rate_429"
  | "usage_5h"
  | "usage_7d"
  | "usage_7d_all"
  | "quota_monthly";
export type AccountViewMode = "table" | "card";
export type AccountUsageDisplayMode = "used" | "remaining";
export type AccountAutoRenewPeriod = "month" | "quarter" | "year";
export type OAuthAddMethod = "oauth" | "setup-token";
export type ProxyProtocol = "http" | "https" | "socks5" | "socks5h";
export type OpenAIAccountTier = "pro_20x" | "pro_5x" | "plus" | "team" | "free";
export type ClaudeAccountTier = "max_20x" | "max_5x" | "pro";
export type AccountTier = OpenAIAccountTier | ClaudeAccountTier;

// Claude Model type (returned by /v1/models and account models API)
export interface ClaudeModel {
  id: string;
  type: string;
  display_name: string;
  created_at: string;
  canonical_id?: string;
  mode?: "text" | "image" | "video" | "embedding" | "other";
  provider?: string;
  provider_label?: string;
  source_protocol?: "openai" | "anthropic" | "gemini";
  status?: "stable" | "beta" | "deprecated";
  deprecated_at?: string;
  replaced_by?: string;
}

export interface AdminAccountModelOption extends ClaudeModel {
  target_model_id?: string;
  availability_state?: "verified" | "unavailable" | "unknown";
  stale_state?: "fresh" | "stale" | "unverified";
}

export interface Proxy {
  id: number;
  name: string;
  protocol: ProxyProtocol;
  host: string;
  port: number;
  username: string | null;
  password?: string | null;
  status: "active" | "inactive";
  expires_at?: string | null;
  expiry_remind_days?: number;
  fallback_proxy_id?: number | null;
  account_count?: number; // Number of accounts using this proxy
  latency_ms?: number;
  latency_status?: "success" | "failed";
  latency_message?: string;
  ip_address?: string;
  country?: string;
  country_code?: string;
  region?: string;
  city?: string;
  quality_status?: "healthy" | "warn" | "challenge" | "failed";
  quality_score?: number;
  quality_grade?: string;
  quality_summary?: string;
  quality_checked?: number;
  created_at: string;
  updated_at: string;
}

export interface ProxyAccountSummary {
  id: number;
  name: string;
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  type: AccountType;
  notes?: string | null;
}

export interface ProxyQualityCheckItem {
  target: string;
  status: "pass" | "warn" | "fail" | "challenge";
  http_status?: number;
  latency_ms?: number;
  message?: string;
  cf_ray?: string;
}

export interface ProxyQualityCheckResult {
  proxy_id: number;
  score: number;
  grade: string;
  summary: string;
  exit_ip?: string;
  country?: string;
  country_code?: string;
  base_latency_ms?: number;
  passed_count: number;
  warn_count: number;
  failed_count: number;
  challenge_count: number;
  checked_at: number;
  items: ProxyQualityCheckItem[];
}

// Gemini credentials structure for OAuth and API Key authentication
export interface GeminiCredentials {
  // API Key authentication
  api_key?: string;
  gemini_api_variant?: "ai_studio" | "vertex_express" | string;

  // OAuth authentication
  access_token?: string;
  refresh_token?: string;
  oauth_type?:
    | "code_assist"
    | "google_one"
    | "ai_studio"
    | "vertex_ai"
    | string;
  tier_id?:
    | "google_one_free"
    | "google_ai_pro"
    | "google_ai_ultra"
    | "gcp_standard"
    | "gcp_enterprise"
    | "aistudio_free"
    | "aistudio_tier_1"
    | "aistudio_tier_2"
    | "aistudio_tier_3"
    | "aistudio_paid"
    | "LEGACY"
    | "PRO"
    | "ULTRA"
    | string;
  project_id?: string;
  vertex_project_id?: string;
  vertex_location?: string;
  vertex_service_account_json?: string;
  token_type?: string;
  scope?: string;
  expires_at?: string;
  base_url?: string;
  model_mapping?: Record<string, string>;
}

export interface TempUnschedulableRule {
  error_code: number;
  keywords: string[];
  duration_minutes: number;
  description: string;
}

export interface TempUnschedulableState {
  until_unix: number;
  triggered_at_unix: number;
  status_code: number;
  matched_keyword: string;
  rule_index: number;
  error_message: string;
}

export interface TempUnschedulableStatus {
  active: boolean;
  state?: TempUnschedulableState;
}

export interface AccountAutoRecoveryProbeSummary {
  checked_at?: string;
  status?: "success" | "retry_scheduled" | "blacklisted" | string;
  summary?: string;
  blacklisted?: boolean;
  next_retry_at?: string;
  error_code?: string;
}

export interface AccountReauthStatus {
  required_since?: string;
  deadline_at?: string;
  reason_code?: string;
  message?: string;
}

export interface Account {
  id: number;
  name: string;
  notes?: string | null;
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  gateway_batch_enabled?: boolean | null;
  active_usage_available?: boolean;
  batch_archive_enabled?: boolean | null;
  batch_archive_auto_prefetch_enabled?: boolean | null;
  batch_archive_retention_days?: number | null;
  batch_archive_billing_mode?: "log_only" | "archive_charge" | null;
  batch_archive_download_price_usd?: number | null;
  allow_vertex_batch_overflow?: boolean | null;
  accept_aistudio_batch_overflow?: boolean | null;
  type: AccountType;
  lifecycle_state?: AccountLifecycleState;
  lifecycle_reason_code?: string | null;
  lifecycle_reason_message?: string | null;
  blacklisted_at?: string | null;
  blacklist_purge_at?: string | null;
  credentials?: Record<string, unknown>;
  // Extra fields including Codex usage and model-level rate limits (Antigravity smart retry)
  extra?: CodexUsageSnapshot & {
    model_rate_limits?: Record<
      string,
      { rate_limited_at: string; rate_limit_reset_at: string }
    >;
    image_protocol_mode?: OpenAIImageProtocolMode;
    image_compat_allowed?: boolean;
    account_tier?: AccountTier | string;
    reauth_status?: AccountReauthStatus;
    gateway_protocol?: GatewayProtocol;
    gateway_accepted_protocols?: GatewayAcceptedProtocol[];
    gateway_openai_request_format?: GatewayOpenAIRequestFormat;
    gateway_openai_image_protocol_mode?: OpenAIImageProtocolMode;
    deepseek_model_concurrency_limits?: Record<string, number>;
  } & Record<string, unknown>;
  proxy_id: number | null;
  original_proxy_id?: number | null;
  original_proxy_name?: string | null;
  concurrency: number;
  load_factor?: number | null;
  current_concurrency?: number; // Real-time concurrency count from Redis
  priority: number;
  rate_multiplier?: number; // Account billing multiplier (>=0, 0 means free)
  status: "active" | "inactive" | "error";
  error_message: string | null;
  last_used_at: string | null;
  expires_at: number | null;
  auto_pause_on_expired: boolean;
  auto_renew_enabled: boolean;
  auto_renew_period: AccountAutoRenewPeriod;
  created_at: string;
  updated_at: string;
  proxy?: Proxy;
  group_ids?: number[]; // Groups this account belongs to
  groups?: Group[]; // Preloaded group objects

  // Rate limit & scheduling fields
  schedulable: boolean;
  rate_limited_at: string | null;
  rate_limit_reset_at: string | null;
  rate_limit_reason?: AccountRateLimitReason | null;
  overload_until: string | null;
  temp_unschedulable_until: string | null;
  temp_unschedulable_reason: string | null;
  auto_recovery_probe?: AccountAutoRecoveryProbeSummary | null;

  // Session window fields (5-hour window)
  session_window_start: string | null;
  session_window_end: string | null;
  session_window_status: "allowed" | "allowed_warning" | "rejected" | null;
  // 5-hour window cost guardrails for Anthropic OAuth/SetupToken accounts.
  window_cost_limit?: number | null;
  window_cost_sticky_reserve?: number | null;
  // Session cap settings for Anthropic OAuth/SetupToken accounts.
  max_sessions?: number | null;
  session_idle_timeout_minutes?: number | null;
  // RPM scheduling settings for Anthropic OAuth/SetupToken accounts.
  base_rpm?: number | null;
  rpm_strategy?: string | null;
  rpm_sticky_buffer?: number | null;
  user_msg_queue_mode?: string | null; // "serialize" | "throttle" | null
  // TLS fingerprinting settings for Anthropic OAuth/SetupToken accounts.
  enable_tls_fingerprint?: boolean | null;
  tls_fingerprint_profile_id?: number | null;
  claude_code_mimic_enabled?: boolean | null;
  // Mask session ids inside metadata.user_id for Anthropic OAuth/SetupToken requests.
  session_id_masking_enabled?: boolean | null;
  // Force cache creation billing into a specific TTL bucket for Anthropic OAuth/SetupToken accounts.
  cache_ttl_override_enabled?: boolean | null;
  cache_ttl_override_target?: string | null;
  custom_base_url_enabled?: boolean | null;
  custom_base_url?: string | null;
  // API Key quota limits and usage snapshots.
  quota_limit?: number | null;
  quota_used?: number | null;
  quota_limit_by_currency?: Record<string, number>;
  quota_used_by_currency?: Record<string, number>;
  quota_daily_limit?: number | null;
  quota_daily_used?: number | null;
  quota_daily_limit_by_currency?: Record<string, number>;
  quota_daily_used_by_currency?: Record<string, number>;
  quota_weekly_limit?: number | null;
  quota_weekly_used?: number | null;
  quota_weekly_limit_by_currency?: Record<string, number>;
  quota_weekly_used_by_currency?: Record<string, number>;
  quota_daily_reset_at?: string | null;
  quota_weekly_reset_at?: string | null;
  quota_monthly_limit?: number | null;
  quota_monthly_used?: number | null;
  quota_monthly_limit_by_currency?: Record<string, number>;
  quota_monthly_used_by_currency?: Record<string, number>;
  quota_monthly_reset_at?: string | null;

  // Runtime snapshots captured for account-level usage widgets.
  current_window_cost?: number | null; // Runtime snapshot of the current 5-hour window cost.
  active_sessions?: number | null; // Runtime snapshot of active sessions.
  current_rpm?: number | null; // Runtime snapshot of current requests per minute.
}

export type AccountDaily5HTriggerAccountType =
  | "chatgpt_oauth"
  | "claude_code_oauth_setup_token"
  | "google_oauth";

export type AccountDaily5HTriggerModelMode = "auto" | "fixed";

export interface AccountDaily5HTriggerModelSettings {
  mode: AccountDaily5HTriggerModelMode;
  fixed_model_id?: string;
}

export interface AccountDaily5HTriggerModelOption {
  model_id: string;
  display_name: string;
  provider?: string;
  provider_label?: string;
  account_count: number;
}

export interface AccountDaily5HTriggerAccountTypeSummary {
  account_type: AccountDaily5HTriggerAccountType;
  count: number;
  models: AccountDaily5HTriggerModelOption[];
}

export interface AccountDaily5HTriggerSettings {
  enabled: boolean;
  selected_account_types: AccountDaily5HTriggerAccountType[];
  include_paused_accounts: boolean;
  ignore_free_accounts: boolean;
  skip_cn_holidays_and_weekends: boolean;
  openai_model_mode: AccountDaily5HTriggerModelSettings;
  anthropic_model_mode: AccountDaily5HTriggerModelSettings;
  gemini_model_mode: AccountDaily5HTriggerModelSettings;
}

export interface AccountDaily5HTriggerSettingsView {
  settings: AccountDaily5HTriggerSettings;
  candidates: AccountDaily5HTriggerAccountTypeSummary[];
}

export interface AccountStatusSummary {
  total: number;
  by_status: {
    active: number;
    inactive: number;
    error: number;
  };
  rate_limited: number;
  temp_unschedulable: number;
  overloaded: number;
  paused: number;
  in_use: number;
  remaining_available: number;
  by_platform: Partial<Record<AccountPlatform, number>>;
  limited_breakdown: {
    total: number;
    rate_429: number;
    usage_5h: number;
    usage_7d: number;
    usage_7d_all: number;
    quota_monthly: number;
  };
}

export interface AccountRuntimeSummary {
  in_use: number;
}

// Account Usage types
export interface WindowStats {
  requests: number;
  tokens: number;
  input_tokens?: number;
  output_tokens?: number;
  cache_creation_tokens?: number;
  cache_read_tokens?: number;
  cache_tokens?: number;
  cache_hit_rate?: number;
  cost: number; // Standard cost before final billing adjustments.
  standard_cost?: number;
  user_cost?: number;
  success_rate?: number;
  average_duration_ms?: number;
  weekly?: WindowStats | null;
  monthly?: WindowStats | null;
  total?: WindowStats | null;
}

export type AccountTodayStats = WindowStats;

export interface UsageProgress {
  utilization: number; // Utilization percentage in the range 0-100.
  resets_at: string | null;
  remaining_seconds: number;
  window_stats?: WindowStats | null; // Optional stats snapshot for the active quota window.
  used_requests?: number;
  limit_requests?: number;
}

// Color palette keys used for account usage rows.
export type AccountUsageRowColor =
  | "indigo"
  | "emerald"
  | "purple"
  | "amber"
  | "orange"
  | "green";

export interface AccountUsageResetRow {
  key: string;
  label: string;
  resetsAt: string | null;
  remainingSeconds?: number | null;
  remainingAnchorMs?: number | null;
}

export interface AccountUsagePresentationRow extends AccountUsageResetRow {
  utilization: number;
  windowStats?: WindowStats | null;
  color: AccountUsageRowColor;
  inlineRemaining?: boolean;
  detailedReset?: boolean;
}

export interface AccountUsagePresentationMeta {
  loadingRows: number;
  snapshotUpdatedAtText?: string;
  snapshotUpdatedAtTooltip?: string;
  sampledBadgeLabel?: string;
  sampledBadgeTooltip?: string;
  noteText?: string;
  antigravityTierLabel?: string | null;
  antigravityTierClass?: string;
  hasIneligibleTiers?: boolean;
  protocolGatewayBadgeLabel?: string | null;
  protocolGatewayBadgeClass?: string;
  geminiAuthTypeLabel?: string | null;
  geminiTierClass?: string;
  geminiQuotaPolicyChannel?: string;
  geminiQuotaPolicyLimits?: string;
  geminiQuotaPolicyDocsUrl?: string;
  openAIResetCreditsAvailableCount?: number | null;
  openAIResetCreditsKnown?: boolean;
  openAIResetCreditsStatus?: OpenAIResetCreditsStatus;
  openAIResetCreditsUnsupportedReason?: string;
}

export interface AccountUsagePresentation {
  loading: boolean;
  error: string | null;
  state: "bars" | "loading" | "error" | "empty" | "unlimited";
  windowRows: AccountUsagePresentationRow[];
  resetRows: AccountUsageResetRow[];
  meta: AccountUsagePresentationMeta;
}

export interface AntigravityModelQuota {
  utilization: number; // Utilization percentage in the range 0-100.
  reset_time: string; // Next reset timestamp in ISO8601 format.
}

export interface AccountUsageInfo {
  source?: "passive" | "active";
  updated_at: string | null;
  five_hour: UsageProgress | null;
  seven_day: UsageProgress | null;
  spark_five_hour?: UsageProgress | null;
  spark_seven_day?: UsageProgress | null;
  seven_day_sonnet: UsageProgress | null;
  gemini_shared_daily?: UsageProgress | null;
  gemini_pro_daily?: UsageProgress | null;
  gemini_flash_daily?: UsageProgress | null;
  gemini_shared_minute?: UsageProgress | null;
  gemini_pro_minute?: UsageProgress | null;
  gemini_flash_minute?: UsageProgress | null;
  openai_reset_credits?: OpenAIResetCreditsInfo | null;
  antigravity_quota?: Record<string, AntigravityModelQuota> | null;
  ai_credits?: Array<{
    credit_type?: string;
    amount?: number;
    minimum_balance?: number;
  }> | null;
  is_forbidden?: boolean;
  forbidden_reason?: string;
  forbidden_type?: string;
  validation_url?: string;
  needs_verify?: boolean;
  is_banned?: boolean;
  needs_reauth?: boolean;
  error_code?: string;
  error?: string;
}

export interface OpenAIResetCreditsInfo {
  available_count?: number | null;
  updated_at?: string | null;
  source?: string;
  status?: OpenAIResetCreditsStatus;
  unsupported_reason?: string;
}

export type OpenAIResetCreditsStatus =
  | "available"
  | "unknown_or_unsupported"
  | "unsupported"
  | string;

// OpenAI Codex usage snapshot (from response headers)
export interface CodexUsageSnapshot {
  // Legacy fields (kept for backwards compatibility)
  // NOTE: The naming is ambiguous - actual window type is determined by window_minutes value
  codex_primary_used_percent?: number; // Usage percentage (check window_minutes for actual window type)
  codex_primary_reset_after_seconds?: number; // Seconds until reset
  codex_primary_window_minutes?: number; // Window in minutes
  codex_secondary_used_percent?: number; // Usage percentage (check window_minutes for actual window type)
  codex_secondary_reset_after_seconds?: number; // Seconds until reset
  codex_secondary_window_minutes?: number; // Window in minutes
  codex_primary_over_secondary_percent?: number; // Overflow ratio

  // Canonical fields (normalized by backend, use these preferentially)
  codex_5h_used_percent?: number; // 5-hour window usage percentage
  codex_5h_reset_after_seconds?: number; // Seconds until 5h window reset
  codex_5h_reset_at?: string; // 5-hour window absolute reset time (RFC3339)
  codex_5h_window_minutes?: number; // 5h window in minutes (should be ~300)
  codex_7d_used_percent?: number; // 7-day window usage percentage
  codex_7d_reset_after_seconds?: number; // Seconds until 7d window reset
  codex_7d_reset_at?: string; // 7-day window absolute reset time (RFC3339)
  codex_7d_window_minutes?: number; // 7d window in minutes (should be ~10080)
  codex_spark_5h_used_percent?: number; // Spark 5-hour window usage percentage
  codex_spark_5h_reset_after_seconds?: number; // Seconds until Spark 5h window reset
  codex_spark_5h_reset_at?: string; // Spark 5-hour window absolute reset time (RFC3339)
  codex_spark_5h_window_minutes?: number; // Spark 5h window in minutes (should be ~300)
  codex_spark_7d_used_percent?: number; // Spark 7-day window usage percentage
  codex_spark_7d_reset_after_seconds?: number; // Seconds until Spark 7d window reset
  codex_spark_7d_reset_at?: string; // Spark 7-day window absolute reset time (RFC3339)
  codex_spark_7d_window_minutes?: number; // Spark 7d window in minutes (should be ~10080)
  codex_account_7d_all_exhausted?: boolean; // Whether both Codex 7d windows are exhausted
  openai_rate_limit_reset_credits_available_count?: number;
  openai_rate_limit_reset_credits_updated_at?: string;
  openai_quota_usage_updated_at?: string;
  openai_rate_limit_reset_credits_status?: OpenAIResetCreditsStatus;
  openai_rate_limit_reset_credits_unsupported_reason?: string;

  codex_usage_updated_at?: string; // Last update timestamp
  openai_known_models?: string[];
  openai_known_models_updated_at?: string;
  openai_known_models_source?:
    | "import_models"
    | "test_probe"
    | "model_mapping"
    | string;
  model_probe_snapshot?: {
    models: string[];
    updated_at?: string;
    source?: string;
    probe_source?: string;
  };
}

export interface CreateAccountRequest {
  name: string;
  notes?: string | null;
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  type: AccountType;
  lifecycle_state?: AccountLifecycleState;
  lifecycle_reason_code?: string | null;
  lifecycle_reason_message?: string | null;
  credentials: Record<string, unknown>;
  extra?: Record<string, unknown>;
  proxy_id?: number | null;
  concurrency?: number;
  load_factor?: number | null;
  priority?: number;
  rate_multiplier?: number; // Account billing multiplier (>=0, 0 means free)
  group_ids?: number[];
  expires_at?: number | null;
  auto_pause_on_expired?: boolean;
  auto_renew_enabled?: boolean;
  auto_renew_period?: AccountAutoRenewPeriod;
  confirm_mixed_channel_risk?: boolean;
}

export interface BatchArchiveAccountsRequest {
  account_ids: number[];
  group_name: string;
}

export interface BatchArchiveAccountsResult {
  archived_count: number;
  failed_count: number;
  archive_group_id: number;
  archive_group_name: string;
  success_ids?: number[];
  failed_ids?: number[];
}

export interface ArchivedAccountGroupSummary {
  group_id: number;
  group_name: string;
  total_count: number;
  available_count: number;
  invalid_count: number;
  latest_updated_at: string;
}

export interface UnarchiveAccountResult {
  account_id: number;
  success: boolean;
  restored_group_ids?: number[];
  used_fallback_current_group: boolean;
  error_message?: string;
}

export interface UnarchiveAccountsResult {
  restored_count: number;
  failed_count: number;
  restored_to_original_group_count: number;
  restored_in_place_count: number;
  results: UnarchiveAccountResult[];
}

export interface UpdateAccountRequest {
  name?: string;
  notes?: string | null;
  gateway_protocol?: GatewayProtocol;
  type?: AccountType;
  lifecycle_state?: AccountLifecycleState;
  lifecycle_reason_code?: string | null;
  lifecycle_reason_message?: string | null;
  credentials?: Record<string, unknown>;
  extra?: Record<string, unknown>;
  proxy_id?: number | null;
  concurrency?: number;
  load_factor?: number | null;
  priority?: number;
  rate_multiplier?: number; // Account billing multiplier (>=0, 0 means free)
  schedulable?: boolean;
  status?: "active" | "inactive" | "error";
  group_ids?: number[];
  expires_at?: number | null;
  auto_pause_on_expired?: boolean;
  auto_renew_enabled?: boolean;
  auto_renew_period?: AccountAutoRenewPeriod;
  confirm_mixed_channel_risk?: boolean;
}

export interface CheckMixedChannelRequest {
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  group_ids: number[];
  account_id?: number;
}

export interface MixedChannelWarningDetails {
  group_id: number;
  group_name: string;
  current_platform: string;
  other_platform: string;
}

export interface CheckMixedChannelResponse {
  has_risk: boolean;
  error?: string;
  message?: string;
  details?: MixedChannelWarningDetails;
}

export interface CreateProxyRequest {
  name: string;
  protocol: ProxyProtocol;
  host: string;
  port: number;
  username?: string | null;
  password?: string | null;
  expires_at?: string | null;
  expiry_remind_days?: number;
  fallback_proxy_id?: number | null;
}

export interface UpdateProxyRequest {
  name?: string;
  protocol?: ProxyProtocol;
  host?: string;
  port?: number;
  username?: string | null;
  password?: string | null;
  status?: "active" | "inactive";
  expires_at?: string | null;
  expiry_remind_days?: number;
  fallback_proxy_id?: number | null;
}

export interface AccountProxyRestoreResult {
  account_id: number;
  restored_proxy_id: number;
  restored_proxy_name: string;
  previous_fallback_id?: number | null;
  previous_fallback_name?: string;
}

export interface AdminDataPayload {
  type?: string;
  version?: number;
  exported_at: string;
  proxies: AdminDataProxy[];
  accounts: AdminDataAccount[];
}

export interface AdminDataProxy {
  proxy_key: string;
  name: string;
  protocol: ProxyProtocol;
  host: string;
  port: number;
  username?: string | null;
  password?: string | null;
  status: "active" | "inactive";
  expires_at?: string | null;
  expiry_remind_days?: number;
  fallback_proxy_key?: string | null;
}

export interface AdminDataAccount {
  name: string;
  notes?: string | null;
  platform: AccountPlatform;
  type: AccountType;
  credentials: Record<string, unknown>;
  extra?: Record<string, unknown>;
  proxy_key?: string | null;
  concurrency: number;
  priority: number;
  rate_multiplier?: number | null;
  expires_at?: number | null;
  auto_pause_on_expired?: boolean;
  auto_renew_enabled?: boolean;
  auto_renew_period?: AccountAutoRenewPeriod;
}

export interface AdminDataImportError {
  kind: "proxy" | "account";
  name?: string;
  proxy_key?: string;
  message: string;
}

export interface AdminDataImportResult {
  proxy_created: number;
  proxy_reused: number;
  proxy_failed: number;
  account_created: number;
  account_failed: number;
  created_accounts?: AdminDataImportCreatedAccount[];
  errors?: AdminDataImportError[];
}

export interface AdminDataImportCreatedAccount {
  account_id: number;
  name: string;
  platform: AccountPlatform;
  type: AccountType;
}

export type AdminAccountImportJobStatus =
  | "queued"
  | "running"
  | "succeeded"
  | "partial_failed"
  | "failed"
  | "cancelled";

export interface AdminAccountImportJobProgress {
  total: number;
  processed: number;
}

export interface AdminAccountImportJob {
  job_id: string;
  status: AdminAccountImportJobStatus;
  progress: AdminAccountImportJobProgress;
  result: AdminDataImportResult;
  created_accounts_summary: AdminDataImportCreatedAccount[];
  error?: string;
  cancel_requested: boolean;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AdminAccountImportJobCreateResult {
  job_id: string;
}

export interface AdminAccountImportGroupBindingSection {
  platform: AccountPlatform;
  type: AccountType;
  group_ids: number[];
}

export interface AdminAccountImportGroupBindingRequest {
  sections: AdminAccountImportGroupBindingSection[];
}

export interface AdminAccountImportGroupBindingResult {
  success: number;
  failed: number;
  bound_count: number;
  skipped: number;
  errors?: string[];
}
