import type { User } from './auth'
import type { ApiKey, Group, GroupPlatform } from './api-key-groups'
import type { AccountPlatform } from './accounts'
import type { UserSubscription } from './user-subscriptions'
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
  reasoning_effort_raw?: string | null;
  reasoning_effort_effective?: string | null;
  request_context_length_tokens?: number | null;
  million_context_requested?: boolean | null;
  million_context_effective?: boolean | null;
  million_context_source?: string | null;
  million_context_beta_token?: string | null;
  thinking_enabled?: boolean | null;
  inbound_endpoint?: string | null;
  upstream_endpoint?: string | null;
  upstream_service?: string | null;
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
  billing_currency?: string;
  total_cost_usd_equivalent?: number | null;
  actual_cost_usd_equivalent?: number | null;
  cost_by_currency?: Record<string, number>;
  actual_cost_by_currency?: Record<string, number>;
  billing_exempt_reason?: "admin_free" | null;
  rate_multiplier: number | null;
  billing_type: number;

  request_type?: UsageRequestType;
  status: UsageLogStatus;
  stream: boolean;
  openai_ws_mode?: boolean;
  duration_ms: number | null;
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
  status: "active" | "used" | "expired" | "unused" | "disabled";
  used_by: number | null;
  used_at: string | null;
  created_at: string;
  expires_at?: string | null;
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
  expires_at?: string | null; // Redeem code expiration time, separate from subscription validity.
  expires_in_days?: number; // Relative redeem code expiration days; mutually exclusive with expires_at.
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
  cost_by_currency?: Record<string, number>;
  actual_cost_by_currency?: Record<string, number>;
  // Today request, token, and cost totals.
  today_requests: number;
  today_input_tokens: number;
  today_output_tokens: number;
  today_cache_creation_tokens: number;
  today_cache_read_tokens: number;
  today_tokens: number;
  today_cost: number; // Standard cost before final billing adjustments.
  today_actual_cost: number; // Actual billed cost.
  today_cost_by_currency?: Record<string, number>;
  today_actual_cost_by_currency?: Record<string, number>;
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
  cost_by_currency?: Record<string, number>;
  actual_cost_by_currency?: Record<string, number>;
  admin_free_requests: number;
  admin_free_standard_cost: number;
  average_duration_ms: number;
  today_requests: number;
  today_input_tokens: number;
  today_output_tokens: number;
  today_cache_tokens: number;
  today_tokens: number;
  today_cost: number;
  today_actual_cost: number;
  today_cost_by_currency?: Record<string, number>;
  today_actual_cost_by_currency?: Record<string, number>;
  today_average_duration_ms: number;
  models?: Record<string, number>;
  platform_breakdown?: PlatformUsageStat[];
}

export interface PlatformUsageStat {
  platform: AccountPlatform | GroupPlatform | "unknown" | string;
  requests: number;
  input_tokens: number;
  output_tokens: number;
  cache_tokens: number;
  total_tokens: number;
  cost: number;
  actual_cost: number;
  cost_by_currency?: Record<string, number>;
  actual_cost_by_currency?: Record<string, number>;
  average_duration_ms: number;
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


// ==================== Query Parameters ====================

export interface UsageQueryParams {
  page?: number;
  page_size?: number;
  api_key_id?: number;
  user_id?: number;
  account_id?: number;
  group_id?: number;
  channel_id?: number;
  platform?: AccountPlatform | GroupPlatform | string | null;
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
