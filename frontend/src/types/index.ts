/**
 * Core Type Definitions for Sub2API Frontend
 */

// ==================== Common Types ====================

export interface SelectOption {
  value: string | number | boolean | null;
  label: string;
  [key: string]: any; // Support extra properties for custom templates
}

export interface BasePaginationResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

export interface FetchOptions {
  signal?: AbortSignal;
}

// ==================== User & Auth Types ====================

export interface User {
  id: number;
  username: string;
  email: string;
  role: "admin" | "user"; // User role for authorization
  request_details_review?: boolean;
  admin_free_billing?: boolean;
  balance: number; // User balance for API usage
  concurrency: number; // Allowed concurrent requests
  status: "active" | "disabled"; // Account status
  allowed_groups: number[] | null; // Allowed group IDs (null = all non-exclusive groups)
  subscriptions?: UserSubscription[]; // User's active subscriptions
  created_at: string;
  updated_at: string;
}

export interface AdminUser extends User {
  notes: string;
  admin_free_billing: boolean;
  // Per-group rate multipliers keyed by group_id.
  group_rates?: Record<number, number>;
  // Current concurrency snapshot for admin views.
  current_concurrency?: number;
}

export interface LoginRequest {
  email: string;
  password: string;
  turnstile_token?: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  verify_code?: string;
  turnstile_token?: string;
  promo_code?: string;
  invitation_code?: string;
}

export interface SendVerifyCodeRequest {
  email: string;
  turnstile_token?: string;
}

export interface SendVerifyCodeResponse {
  message: string;
  countdown: number;
}

export interface CustomMenuItem {
  id: string;
  label: string;
  icon_svg: string;
  url: string;
  visibility: "user" | "admin";
  sort_order: number;
}

export interface PublicSettings {
  registration_enabled: boolean;
  email_verify_enabled: boolean;
  registration_email_suffix_whitelist: string[];
  promo_code_enabled: boolean;
  password_reset_enabled: boolean;
  invitation_code_enabled: boolean;
  turnstile_enabled: boolean;
  turnstile_site_key: string;
  site_name: string;
  site_logo: string;
  site_subtitle: string;
  api_base_url: string;
  contact_info: string;
  doc_url: string;
  home_content: string;
  hide_ccs_import_button: boolean;
  available_channels_enabled: boolean;
  channel_monitor_enabled: boolean;
  public_model_catalog_enabled: boolean;
  purchase_subscription_enabled: boolean;
  purchase_subscription_url: string;
  custom_menu_items: CustomMenuItem[];
  linuxdo_oauth_enabled: boolean;
  backend_mode_enabled: boolean;
  maintenance_mode_enabled: boolean;
  version: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token?: string; // New: Refresh Token for token renewal
  expires_in?: number; // New: Access Token expiry time in seconds
  token_type: string;
  user: User & { run_mode?: "standard" | "simple" };
}

export interface CurrentUserResponse extends User {
  run_mode?: "standard" | "simple";
}

// ==================== Subscription Types ====================

export interface Subscription {
  id: number;
  user_id: number;
  name: string;
  url: string;
  type: "clash" | "v2ray" | "surge" | "quantumult" | "shadowrocket";
  update_interval: number; // in hours
  last_updated: string | null;
  node_count: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateSubscriptionRequest {
  name: string;
  url: string;
  type: Subscription["type"];
  update_interval?: number;
}

export interface UpdateSubscriptionRequest {
  name?: string;
  url?: string;
  type?: Subscription["type"];
  update_interval?: number;
  is_active?: boolean;
}

// ==================== Announcement Types ====================

export type AnnouncementStatus = "draft" | "active" | "archived";
export type AnnouncementNotifyMode = "silent" | "popup";

export type AnnouncementConditionType = "subscription" | "balance";

export type AnnouncementOperator = "in" | "gt" | "gte" | "lt" | "lte" | "eq";

export interface AnnouncementCondition {
  type: AnnouncementConditionType;
  operator: AnnouncementOperator;
  group_ids?: number[];
  value?: number;
}

export interface AnnouncementConditionGroup {
  all_of?: AnnouncementCondition[];
}

export interface AnnouncementTargeting {
  any_of?: AnnouncementConditionGroup[];
}

export interface Announcement {
  id: number;
  title: string;
  content: string;
  status: AnnouncementStatus;
  notify_mode: AnnouncementNotifyMode;
  targeting: AnnouncementTargeting;
  starts_at?: string;
  ends_at?: string;
  created_by?: number;
  updated_by?: number;
  created_at: string;
  updated_at: string;
}

export interface UserAnnouncement {
  id: number;
  title: string;
  content: string;
  notify_mode: AnnouncementNotifyMode;
  starts_at?: string;
  ends_at?: string;
  read_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAnnouncementRequest {
  title: string;
  content: string;
  status?: AnnouncementStatus;
  notify_mode?: AnnouncementNotifyMode;
  targeting: AnnouncementTargeting;
  starts_at?: number;
  ends_at?: number;
}

export interface UpdateAnnouncementRequest {
  title?: string;
  content?: string;
  status?: AnnouncementStatus;
  notify_mode?: AnnouncementNotifyMode;
  targeting?: AnnouncementTargeting;
  starts_at?: number;
  ends_at?: number;
}

export interface AnnouncementUserReadStatus {
  user_id: number;
  email: string;
  username: string;
  balance: number;
  eligible: boolean;
  read_at?: string;
}

// ==================== Proxy Node Types ====================

export interface ProxyNode {
  id: number;
  subscription_id: number;
  name: string;
  type: "ss" | "ssr" | "vmess" | "vless" | "trojan" | "hysteria" | "hysteria2";
  server: string;
  port: number;
  config: Record<string, unknown>; // JSON configuration specific to proxy type
  latency: number | null; // in milliseconds
  last_checked: string | null;
  is_available: boolean;
  created_at: string;
  updated_at: string;
}

// ==================== Conversion Types ====================

export interface ConversionRequest {
  subscription_ids: number[];
  target_type: "clash" | "v2ray" | "surge" | "quantumult" | "shadowrocket";
  filter?: {
    name_pattern?: string;
    types?: ProxyNode["type"][];
    min_latency?: number;
    max_latency?: number;
    available_only?: boolean;
  };
  sort?: {
    by: "name" | "latency" | "type";
    order: "asc" | "desc";
  };
}

export interface ConversionResult {
  url: string; // URL to download the converted subscription
  expires_at: string;
  node_count: number;
}

// ==================== Statistics Types ====================

export interface SubscriptionStats {
  subscription_id: number;
  total_nodes: number;
  available_nodes: number;
  avg_latency: number | null;
  by_type: Record<ProxyNode["type"], number>;
  last_update: string;
}

export interface UserStats {
  total_subscriptions: number;
  total_nodes: number;
  active_subscriptions: number;
  total_conversions: number;
  last_conversion: string | null;
}

// ==================== API Response Types ====================

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface ApiError {
  detail: string;
  code?: string;
  field?: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

// ==================== UI State Types ====================

export type ToastType = "success" | "error" | "info" | "warning";
export type ToastDetailTone = "success" | "error" | "info" | "warning";

export interface ToastDetailItem {
  text: string;
  tone?: ToastDetailTone;
}

export type ToastDetailInput = string | ToastDetailItem;

export interface ToastOptions {
  title?: string;
  details?: ToastDetailInput[];
  copyText?: string;
  persistent?: boolean;
  duration?: number;
}

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  title?: string;
  details?: ToastDetailItem[];
  copyText?: string;
  persistent?: boolean;
  duration?: number;
  startTime?: number; // timestamp when toast was created, for progress bar
}

export interface AppState {
  sidebarCollapsed: boolean;
  loading: boolean;
  toasts: Toast[];
}

// ==================== Validation Types ====================

export interface ValidationError {
  field: string;
  message: string;
}

// ==================== Table/List Types ====================

export interface SortConfig {
  key: string;
  order: "asc" | "desc";
}

export interface FilterConfig {
  [key: string]: string | number | boolean | null | undefined;
}

export interface PaginationConfig {
  page: number;
  page_size: number;
}

// ==================== API Key & Group Types ====================

export type GroupPlatform =
  | "anthropic"
  | "kiro"
  | "openai"
  | "copilot"
  | "grok"
  | "gemini"
  | "antigravity"
  | "baidu_document_ai";

export type SubscriptionType = "standard" | "subscription";
export type OpenAIImageProtocolMode = "native" | "compat";
export type OpenAIGroupImageProtocolMode =
  | "inherit"
  | OpenAIImageProtocolMode;

export interface Group {
  id: number;
  name: string;
  description: string | null;
  platform: GroupPlatform;
  priority: number;
  rate_multiplier: number;
  is_exclusive: boolean;
  status: "active" | "inactive";
  subscription_type: SubscriptionType;
  daily_limit_usd: number | null;
  weekly_limit_usd: number | null;
  monthly_limit_usd: number | null;
  // Image pricing is only used by antigravity groups.
  image_price_1k: number | null;
  image_price_2k: number | null;
  image_price_4k: number | null;
  // Restrict the group to Claude Code clients only.
  claude_code_only: boolean;
  image_protocol_mode: OpenAIGroupImageProtocolMode;
  fallback_group_id: number | null;
  fallback_group_id_on_invalid_request: number | null;
  // Toggle OpenAI Messages dispatch support for this group.
  allow_messages_dispatch?: boolean;
  gemini_mixed_protocol_enabled?: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminGroup extends Group {
  // Optional routing map from requested model ids to account ids.
  model_routing: Record<string, number[]> | null;
  model_routing_enabled: boolean;

  // MCP XML injection toggle for antigravity groups.
  mcp_xml_inject: boolean;
  // Optional model scope allowlist for antigravity groups.
  supported_model_scopes?: string[];
  // Aggregated account counters for admin list views.
  account_count?: number;
  active_account_count?: number;
  rate_limited_account_count?: number;
  // Default mapped model for OpenAI Messages-compatible groups.
  default_mapped_model?: string;
  // UI sort weight for admin lists.
  sort_order: number;
}

export interface ApiKey {
  id: number;
  user_id: number;
  key: string;
  name: string;
  deleted?: boolean;
  model_display_mode?: "alias_only" | "source_only" | "alias_and_source";
  group_id: number | null;
  group_ids?: number[];
  api_key_groups?: ApiKeyGroup[];
  status: "active" | "inactive" | "quota_exhausted" | "expired";
  ip_whitelist: string[];
  ip_blacklist: string[];
  last_used_at: string | null;
  quota: number; // Quota limit in USD (0 = unlimited)
  quota_used: number; // Used quota amount in USD
  // Image-only key settings
  image_only_enabled: boolean;
  image_count_billing_enabled: boolean;
  image_max_count: number; // 0 = not configured (falls back to token billing)
  image_count_used: number;
  expires_at: string | null; // Expiration time (null = never expires)
  created_at: string;
  updated_at: string;
  group?: Group;
  rate_limit_5h: number;
  rate_limit_1d: number;
  rate_limit_7d: number;
  usage_5h: number;
  usage_1d: number;
  usage_7d: number;
  window_5h_start: string | null;
  window_1d_start: string | null;
  window_7d_start: string | null;
  reset_5h_at: string | null;
  reset_1d_at: string | null;
  reset_7d_at: string | null;
}

export interface ApiKeyGroup {
  group_id: number;
  group_name: string;
  platform: GroupPlatform;
  priority: number;
  quota: number;
  quota_used: number;
  model_patterns: string[];
}

export interface ApiKeyGroupBindingInput {
  group_id: number;
  quota?: number;
  model_patterns?: string[];
}

export interface UserGroupModelOption {
  public_id: string;
  display_name: string;
  request_protocols?: string[];
  source_ids?: string[];
}

export interface UserGroupModelOptionGroup {
  group_id: number;
  name: string;
  platform: GroupPlatform;
  priority: number;
  models: UserGroupModelOption[];
  model_count: number;
}

export interface CreateApiKeyRequest {
  name: string;
  group_id?: number | null;
  groups?: ApiKeyGroupBindingInput[];
  custom_key?: string; // Optional custom API Key
  ip_whitelist?: string[];
  ip_blacklist?: string[];
  quota?: number; // Quota limit in USD (0 = unlimited)
  expires_in_days?: number; // Days until expiry (null = never expires)
  rate_limit_5h?: number;
  rate_limit_1d?: number;
  rate_limit_7d?: number;
  image_only_enabled?: boolean;
  image_count_billing_enabled?: boolean;
  image_max_count?: number;
}

export interface UpdateApiKeyRequest {
  name?: string;
  group_id?: number | null;
  groups?: ApiKeyGroupBindingInput[];
  status?: "active" | "inactive";
  ip_whitelist?: string[];
  ip_blacklist?: string[];
  quota?: number; // Quota limit in USD (null = no change, 0 = unlimited)
  expires_at?: string | null; // Expiration time (null = no change)
  reset_quota?: boolean; // Reset quota_used to 0
  rate_limit_5h?: number;
  rate_limit_1d?: number;
  rate_limit_7d?: number;
  reset_rate_limit_usage?: boolean;
  image_only_enabled?: boolean;
  image_count_billing_enabled?: boolean;
  image_max_count?: number;
}

export interface CreateGroupRequest {
  name: string;
  description?: string | null;
  platform?: GroupPlatform;
  priority?: number;
  rate_multiplier?: number;
  is_exclusive?: boolean;
  gemini_mixed_protocol_enabled?: boolean;
  subscription_type?: SubscriptionType;
  daily_limit_usd?: number | null;
  weekly_limit_usd?: number | null;
  monthly_limit_usd?: number | null;
  image_price_1k?: number | null;
  image_price_2k?: number | null;
  image_price_4k?: number | null;
  image_protocol_mode?: OpenAIGroupImageProtocolMode;
  claude_code_only?: boolean;
  fallback_group_id?: number | null;
  fallback_group_id_on_invalid_request?: number | null;
  mcp_xml_inject?: boolean;
  supported_model_scopes?: string[];
  // Optional source groups to clone accounts from during group creation.
  copy_accounts_from_group_ids?: number[];
}

export interface UpdateGroupRequest {
  name?: string;
  description?: string | null;
  platform?: GroupPlatform;
  priority?: number;
  rate_multiplier?: number;
  is_exclusive?: boolean;
  gemini_mixed_protocol_enabled?: boolean;
  status?: "active" | "inactive";
  subscription_type?: SubscriptionType;
  daily_limit_usd?: number | null;
  weekly_limit_usd?: number | null;
  monthly_limit_usd?: number | null;
  image_price_1k?: number | null;
  image_price_2k?: number | null;
  image_price_4k?: number | null;
  image_protocol_mode?: OpenAIGroupImageProtocolMode;
  claude_code_only?: boolean;
  fallback_group_id?: number | null;
  fallback_group_id_on_invalid_request?: number | null;
  mcp_xml_inject?: boolean;
  supported_model_scopes?: string[];
  copy_accounts_from_group_ids?: number[];
}

// ==================== Account & Proxy Types ====================

export type AccountPlatform =
  | "anthropic"
  | "kiro"
  | "openai"
  | "copilot"
  | "grok"
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
  | "usage_7d_all";
export type AccountViewMode = "table" | "card";
export type AccountUsageDisplayMode = "used" | "remaining";
export type OAuthAddMethod = "oauth" | "setup-token";
export type ProxyProtocol = "http" | "https" | "socks5" | "socks5h";

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

export interface Account {
  id: number;
  name: string;
  notes?: string | null;
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  gateway_batch_enabled?: boolean | null;
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
    gateway_protocol?: GatewayProtocol;
    gateway_accepted_protocols?: GatewayAcceptedProtocol[];
    gateway_openai_request_format?: GatewayOpenAIRequestFormat;
    gateway_openai_image_protocol_mode?: OpenAIImageProtocolMode;
  } & Record<string, unknown>;
  proxy_id: number | null;
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
  quota_daily_limit?: number | null;
  quota_daily_used?: number | null;
  quota_weekly_limit?: number | null;
  quota_weekly_used?: number | null;

  // Runtime snapshots captured for account-level usage widgets.
  current_window_cost?: number | null; // Runtime snapshot of the current 5-hour window cost.
  active_sessions?: number | null; // Runtime snapshot of active sessions.
  current_rpm?: number | null; // Runtime snapshot of current requests per minute.
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
  };
}

export interface AccountRuntimeSummary {
  in_use: number;
}

// Account Usage types
export interface WindowStats {
  requests: number;
  tokens: number;
  cost: number; // Standard cost before final billing adjustments.
  standard_cost?: number;
  user_cost?: number;
}

export interface UsageProgress {
  utilization: number; // Utilization percentage in the range 0-100.
  resets_at: string | null;
  remaining_seconds: number;
  window_stats?: WindowStats | null; // Optional stats snapshot for the active quota window.
  used_requests?: number;
  limit_requests?: number;
}

// Color palette keys used for account usage rows.
export type AccountUsageRowColor = "indigo" | "emerald" | "purple" | "amber";

export interface AccountUsageResetRow {
  key: string;
  label: string;
  resetsAt: string | null;
  remainingSeconds?: number | null;
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
}

export interface UpdateProxyRequest {
  name?: string;
  protocol?: ProxyProtocol;
  host?: string;
  port?: number;
  username?: string | null;
  password?: string | null;
  status?: "active" | "inactive";
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
  errors?: AdminDataImportError[];
}

// ==================== Usage & Redeem Types ====================

export type RedeemCodeType =
  | "balance"
  | "concurrency"
  | "subscription"
  | "invitation";
export type UsageRequestType = "unknown" | "sync" | "stream" | "ws_v2";
export type UsageLogStatus = "succeeded" | "failed";
export type UsageLogSimulatedClient = "codex" | "gemini_cli";
export type TokenDisplayMode = "full" | "compact";

export interface UsageLog {
  id: number;
  user_id: number;
  api_key_id: number;
  account_id: number | null;
  request_id: string;
  model: string;
  upstream_model?: string | null;
  service_tier?: string | null;
  reasoning_effort?: string | null;
  thinking_enabled?: boolean | null;
  inbound_endpoint?: string | null;
  upstream_endpoint?: string | null;
  channel_id?: number | null;
  model_mapping_chain?: string | null;
  billing_tier?: string | null;
  billing_mode?: string | null;

  group_id: number | null;
  subscription_id: number | null;

  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  cache_creation_5m_tokens: number;
  cache_creation_1h_tokens: number;

  input_cost: number | null;
  output_cost: number | null;
  cache_creation_cost: number | null;
  cache_read_cost: number | null;
  total_cost: number | null;
  actual_cost: number | null;
  billing_exempt_reason?: "admin_free" | null;
  rate_multiplier: number | null;
  billing_type: number;

  request_type?: UsageRequestType;
  status: UsageLogStatus;
  stream: boolean;
  openai_ws_mode?: boolean;
  duration_ms: number;
  first_token_ms: number | null;
  http_status?: number | null;
  error_code?: string | null;
  error_message?: string | null;
  simulated_client?: UsageLogSimulatedClient | null;
  operation_type?: string | null;
  charge_source?: string | null;
  // Image generation usage metrics.
  image_count: number;
  image_size: string | null;
  image_output_tokens?: number | null;
  image_output_cost?: number | null;

  // User-Agent
  user_agent: string | null;

  // Cache TTL Override
  cache_ttl_overridden: boolean;

  created_at: string;

  user?: User;
  api_key?: ApiKey;
  group?: Group;
  subscription?: UserSubscription;
}

export interface UsageRequestPreviewResponse {
  available: boolean;
  request_id: string;
  captured_at: string | null;
  inbound_request_json: string;
  normalized_request_json: string;
  upstream_request_json: string;
  upstream_response_json: string;
  gateway_response_json: string;
  tool_trace_json: string;
}

export interface UsageLogAccountSummary {
  id: number;
  name: string;
}

export interface AdminUsageLog extends UsageLog {
  // Account billing multiplier joined into admin usage rows.
  account_rate_multiplier?: number | null;
  // Best-effort client IP captured for admin review.
  ip_address?: string | null;
  // Preloaded account summary for admin usage tables.
  account?: UsageLogAccountSummary;
  preview_available?: boolean | null;
}

export interface UsageCleanupFilters {
  start_time: string;
  end_time: string;
  user_id?: number;
  api_key_id?: number;
  account_id?: number;
  group_id?: number;
  model?: string | null;
  request_type?: UsageRequestType | null;
  stream?: boolean | null;
  billing_type?: number | null;
}

export interface UsageCleanupTask {
  id: number;
  status: string;
  filters: UsageCleanupFilters;
  created_by: number;
  deleted_rows: number;
  error_message?: string | null;
  canceled_by?: number | null;
  canceled_at?: string | null;
  started_at?: string | null;
  finished_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface RedeemCode {
  id: number;
  code: string;
  type: RedeemCodeType;
  value: number;
  status: "active" | "used" | "expired" | "unused";
  used_by: number | null;
  used_at: string | null;
  created_at: string;
  updated_at?: string;
  group_id?: number | null; // Subscription group bound to this redeem code.
  validity_days?: number; // Subscription validity in days for subscription codes.
  user?: User;
  group?: Group; // Preloaded group object when available.
}

export interface GenerateRedeemCodesRequest {
  count: number;
  type: RedeemCodeType;
  value: number;
  group_id?: number | null; // Subscription group bound to the generated codes.
  validity_days?: number; // Subscription validity in days for subscription codes.
}

export interface RedeemCodeRequest {
  code: string;
}

// ==================== Dashboard & Statistics ====================
export interface DashboardStats {
  // User counters.
  total_users: number;
  today_new_users: number; // Users created today.
  active_users: number; // Daily active users.
  hourly_active_users: number; // Hourly active users.
  stats_updated_at: string; // Last dashboard snapshot update time in RFC3339.
  stats_stale: boolean; // Whether the dashboard snapshot is stale.
  // API key counters.
  total_api_keys: number;
  active_api_keys: number; // Active API keys.
  // Account counters.
  total_accounts: number;
  normal_accounts: number; // Accounts in normal state.
  error_accounts: number; // Accounts in error state.
  ratelimit_accounts: number; // Accounts currently rate limited.
  overload_accounts: number; // Accounts currently overloaded.
  // Lifetime request, token, and cost totals.
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_creation_tokens: number;
  total_cache_read_tokens: number;
  total_tokens: number;
  total_cost: number; // Standard cost before final billing adjustments.
  total_actual_cost: number; // Actual billed cost.
  // Today request, token, and cost totals.
  today_requests: number;
  today_input_tokens: number;
  today_output_tokens: number;
  today_cache_creation_tokens: number;
  today_cache_read_tokens: number;
  today_tokens: number;
  today_cost: number; // Standard cost before final billing adjustments.
  today_actual_cost: number; // Actual billed cost.
  average_duration_ms: number; // Average request duration in milliseconds.
  uptime: number; // Service uptime in seconds.
  rpm: number; // Requests per minute.
  tpm: number; // Tokens per minute.
}

export interface UsageStatsResponse {
  period?: string;
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_tokens: number;
  total_tokens: number;
  total_cost: number;
  total_actual_cost: number;
  admin_free_requests: number;
  admin_free_standard_cost: number;
  average_duration_ms: number;
  models?: Record<string, number>;
}

// ==================== Trend & Chart Types ====================

export interface TrendDataPoint {
  date: string;
  requests: number;
  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  total_tokens: number;
  cost: number; // Standard cost before final billing adjustments.
  actual_cost: number; // Actual billed cost.
}

export interface ModelStat {
  model: string;
  requests: number;
  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  total_tokens: number;
  cost: number; // Standard cost before final billing adjustments.
  actual_cost: number; // Actual billed cost.
}

export interface EndpointStat {
  endpoint: string;
  requests: number;
  total_tokens: number;
  cost: number;
  actual_cost: number;
}

export interface GroupStat {
  group_id: number;
  group_name: string;
  requests: number;
  total_tokens: number;
  cost: number; // Standard cost before final billing adjustments.
  actual_cost: number; // Actual billed cost.
}

export interface UserBreakdownItem {
  user_id: number;
  email: string;
  requests: number;
  total_tokens: number;
  cost: number;
  actual_cost: number;
}

export interface UserUsageTrendPoint {
  date: string;
  user_id: number;
  email: string;
  username?: string;
  requests: number;
  tokens: number;
  cost: number; // Standard cost before final billing adjustments.
  actual_cost: number; // Actual billed cost.
}

export interface UserSpendingRankingItem {
  user_id: number;
  email: string;
  username?: string;
  actual_cost: number;
  requests: number;
  tokens: number;
}

export interface UserSpendingRankingResponse {
  ranking: UserSpendingRankingItem[];
  total_actual_cost: number;
  total_requests: number;
  total_tokens: number;
  start_date: string;
  end_date: string;
}

export interface ApiKeyUsageTrendPoint {
  date: string;
  api_key_id: number;
  key_name: string;
  requests: number;
  tokens: number;
}

// ==================== Admin User Management ====================

export interface UpdateUserRequest {
  email?: string;
  password?: string;
  username?: string;
  notes?: string;
  role?: "admin" | "user";
  request_details_review?: boolean;
  admin_free_billing?: boolean;
  balance?: number;
  concurrency?: number;
  status?: "active" | "disabled";
  allowed_groups?: number[] | null;
  // Per-group billing overrides keyed by group_id. Use null to clear an override.
  group_rates?: Record<number, number | null>;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// ==================== User Subscription Types ====================

export interface UserSubscription {
  id: number;
  user_id: number;
  group_id: number;
  status: "active" | "expired" | "revoked";
  daily_usage_usd: number;
  weekly_usage_usd: number;
  monthly_usage_usd: number;
  daily_window_start: string | null;
  weekly_window_start: string | null;
  monthly_window_start: string | null;
  created_at: string;
  updated_at: string;
  expires_at: string | null;
  user?: User;
  group?: Group;
}

export interface SubscriptionProgress {
  subscription_id: number;
  daily: {
    used: number;
    limit: number | null;
    percentage: number;
    reset_in_seconds: number | null;
  } | null;
  weekly: {
    used: number;
    limit: number | null;
    percentage: number;
    reset_in_seconds: number | null;
  } | null;
  monthly: {
    used: number;
    limit: number | null;
    percentage: number;
    reset_in_seconds: number | null;
  } | null;
  expires_at: string | null;
  days_remaining: number | null;
}

export interface AssignSubscriptionRequest {
  user_id: number;
  group_id: number;
  validity_days?: number;
}

export interface BulkAssignSubscriptionRequest {
  user_ids: number[];
  group_id: number;
  validity_days?: number;
}

export interface ExtendSubscriptionRequest {
  days: number;
}

// ==================== Query Parameters ====================

export interface UsageQueryParams {
  page?: number;
  page_size?: number;
  api_key_id?: number;
  user_id?: number;
  account_id?: number;
  group_id?: number;
  channel_id?: number;
  model?: string;
  request_type?: UsageRequestType;
  stream?: boolean;
  billing_type?: number | null;
  start_date?: string;
  end_date?: string;
}

// ==================== Account Usage Statistics ====================

export interface AccountUsageHistory {
  date: string;
  label: string;
  requests: number;
  tokens: number;
  cost: number;
  actual_cost: number; // Actual billed cost.
  user_cost: number; // Standard cost before final billing adjustments.
}

export interface AccountUsageSummary {
  days: number;
  actual_days_used: number;
  total_cost: number; // Standard cost before final billing adjustments.
  total_user_cost: number;
  total_standard_cost: number;
  total_requests: number;
  total_tokens: number;
  avg_daily_cost: number; // Standard cost before final billing adjustments.
  avg_daily_user_cost: number;
  avg_daily_requests: number;
  avg_daily_tokens: number;
  avg_duration_ms: number;
  today: {
    date: string;
    cost: number;
    user_cost: number;
    requests: number;
    tokens: number;
  } | null;
  highest_cost_day: {
    date: string;
    label: string;
    cost: number;
    user_cost: number;
    requests: number;
  } | null;
  highest_request_day: {
    date: string;
    label: string;
    requests: number;
    cost: number;
    user_cost: number;
  } | null;
}

export interface AccountUsageStatsResponse {
  history: AccountUsageHistory[];
  summary: AccountUsageSummary;
  models: ModelStat[];
  endpoints: EndpointStat[];
  upstream_endpoints: EndpointStat[];
}

// ==================== User Attribute Types ====================

export type UserAttributeType =
  | "text"
  | "textarea"
  | "number"
  | "email"
  | "url"
  | "date"
  | "select"
  | "multi_select";

export interface UserAttributeOption {
  value: string;
  label: string;
  [key: string]: unknown;
}

export interface UserAttributeValidation {
  min_length?: number;
  max_length?: number;
  min?: number;
  max?: number;
  pattern?: string;
  message?: string;
}

export interface UserAttributeDefinition {
  id: number;
  key: string;
  name: string;
  description: string;
  type: UserAttributeType;
  options: UserAttributeOption[];
  required: boolean;
  validation: UserAttributeValidation;
  placeholder: string;
  display_order: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserAttributeValue {
  id: number;
  user_id: number;
  attribute_id: number;
  value: string;
  created_at: string;
  updated_at: string;
}

export interface CreateUserAttributeRequest {
  key: string;
  name: string;
  description?: string;
  type: UserAttributeType;
  options?: UserAttributeOption[];
  required?: boolean;
  validation?: UserAttributeValidation;
  placeholder?: string;
  display_order?: number;
  enabled?: boolean;
}

export interface UpdateUserAttributeRequest {
  key?: string;
  name?: string;
  description?: string;
  type?: UserAttributeType;
  options?: UserAttributeOption[];
  required?: boolean;
  validation?: UserAttributeValidation;
  placeholder?: string;
  display_order?: number;
  enabled?: boolean;
}

export interface UserAttributeValuesMap {
  [attributeId: number]: string;
}

// ==================== Promo Code Types ====================

export interface PromoCode {
  id: number;
  code: string;
  bonus_amount: number;
  max_uses: number;
  used_count: number;
  status: "active" | "disabled";
  expires_at: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface PromoCodeUsage {
  id: number;
  promo_code_id: number;
  user_id: number;
  bonus_amount: number;
  used_at: string;
  user?: User;
}

export interface CreatePromoCodeRequest {
  code?: string;
  bonus_amount: number;
  max_uses?: number;
  expires_at?: number | null;
  notes?: string;
}

export interface UpdatePromoCodeRequest {
  code?: string;
  bonus_amount?: number;
  max_uses?: number;
  status?: "active" | "disabled";
  expires_at?: number | null;
  notes?: string;
}

// ==================== TOTP (2FA) Types ====================

export interface TotpStatus {
  enabled: boolean;
  enabled_at: number | null; // Unix timestamp in seconds
  feature_enabled: boolean;
}

export interface TotpSetupRequest {
  email_code?: string;
  password?: string;
}

export interface TotpSetupResponse {
  secret: string;
  qr_code_url: string;
  setup_token: string;
  countdown: number;
}

export interface TotpEnableRequest {
  totp_code: string;
  setup_token: string;
}

export interface TotpEnableResponse {
  success: boolean;
}

export interface TotpDisableRequest {
  email_code?: string;
  password?: string;
}

export interface TotpVerificationMethod {
  method: "email" | "password";
}

export interface TotpLoginResponse {
  requires_2fa: boolean;
  temp_token?: string;
  user_email_masked?: string;
}

export interface TotpLogin2FARequest {
  temp_token: string;
  totp_code: string;
}

// ==================== Scheduled Test Types ====================

export interface ScheduledTestPlan {
  id: number;
  account_id: number;
  model_id: string;
  model_input_mode?: "catalog" | "manual";
  manual_model_id?: string;
  request_alias?: string;
  source_protocol?: GatewayAcceptedProtocol;
  cron_expression: string;
  enabled: boolean;
  max_results: number;
  auto_recover: boolean;
  notify_policy: "none" | "always" | "failure_only";
  notify_failure_threshold: number;
  retry_interval_minutes: number;
  max_retries: number;
  consecutive_failures: number;
  current_retry_count: number;
  last_run_at: string | null;
  next_run_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ScheduledTestResult {
  id: number;
  plan_id: number;
  status: string;
  response_text: string;
  error_message: string;
  latency_ms: number;
  started_at: string;
  finished_at: string;
  created_at: string;
}

export interface CreateScheduledTestPlanRequest {
  account_id: number;
  model_id: string;
  model_input_mode?: "catalog" | "manual";
  manual_model_id?: string;
  request_alias?: string;
  source_protocol?: GatewayAcceptedProtocol | "";
  cron_expression: string;
  enabled?: boolean;
  max_results?: number;
  auto_recover?: boolean;
  notify_policy?: "none" | "always" | "failure_only";
  notify_failure_threshold?: number;
  retry_interval_minutes?: number;
  max_retries?: number;
}

export interface UpdateScheduledTestPlanRequest {
  model_id?: string;
  model_input_mode?: "catalog" | "manual";
  manual_model_id?: string;
  request_alias?: string;
  source_protocol?: GatewayAcceptedProtocol | "";
  cron_expression?: string;
  enabled?: boolean;
  max_results?: number;
  auto_recover?: boolean;
  notify_policy?: "none" | "always" | "failure_only";
  notify_failure_threshold?: number;
  retry_interval_minutes?: number;
  max_retries?: number;
}
