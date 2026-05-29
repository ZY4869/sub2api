import type { PaginatedResponse } from '@/types'
import type { OpsWSStatus } from './typesShared'
export interface OpsDashboardOverview {
  start_time: string
  end_time: string
  platform: string
  group_id?: number | null
  channel_id?: number | null

  health_score?: number

  system_metrics?: OpsSystemMetricsSnapshot | null
  job_heartbeats?: OpsJobHeartbeat[] | null

  success_count: number
  error_count_total: number
  business_limited_count: number
  error_count_sla: number
  request_count_total: number
  request_count_sla: number

  token_consumed: number

  sla: number
  error_rate: number
  upstream_error_rate: number
  upstream_error_count_excl_429_529: number
  upstream_429_count: number
  upstream_529_count: number

  qps: {
    current: number
    peak: number
    avg: number
  }
  tps: {
    current: number
    peak: number
    avg: number
  }

  duration: OpsPercentiles
  ttft: OpsPercentiles
}

export interface OpsPercentiles {
  p50_ms?: number | null
  p90_ms?: number | null
  p95_ms?: number | null
  p99_ms?: number | null
  avg_ms?: number | null
  max_ms?: number | null
}

export interface OpsThroughputTrendPoint {
  bucket_start: string
  request_count: number
  token_consumed: number
  switch_count?: number
  qps: number
  tps: number
}

export interface OpsThroughputPlatformBreakdownItem {
  platform: string
  request_count: number
  token_consumed: number
}

export interface OpsThroughputGroupBreakdownItem {
  group_id: number
  group_name: string
  request_count: number
  token_consumed: number
}

export interface OpsThroughputTrendResponse {
  bucket: string
  points: OpsThroughputTrendPoint[]
  by_platform?: OpsThroughputPlatformBreakdownItem[]
  top_groups?: OpsThroughputGroupBreakdownItem[]
}

export type OpsRequestKind = 'success' | 'error'
export type OpsRequestDetailsKind = OpsRequestKind | 'all'
export type OpsRequestDetailsSort = 'created_at_desc' | 'duration_desc'

export interface OpsRequestDetail {
  kind: OpsRequestKind
  created_at: string
  request_id: string

  platform?: string
  model?: string
  channel_id?: number | null
  channel_name?: string
  model_mapping_chain?: string
  billing_tier?: string
  billing_mode?: string
  image_output_tokens?: number | null
  image_output_cost?: number | null
  duration_ms?: number | null
  status_code?: number | null

  error_id?: number | null
  phase?: string
  severity?: string
  message?: string

  user_id?: number | null
  api_key_id?: number | null
  account_id?: number | null
  group_id?: number | null

  stream?: boolean
}

export interface OpsRequestDetailsParams {
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h'
  start_time?: string
  end_time?: string

  kind?: OpsRequestDetailsKind

  platform?: string
  group_id?: number | null
  channel_id?: number | null

  user_id?: number
  api_key_id?: number
  account_id?: number

  model?: string
  request_id?: string
  q?: string

  min_duration_ms?: number
  max_duration_ms?: number

  sort?: OpsRequestDetailsSort

  page?: number
  page_size?: number
}

export type OpsRequestDetailsResponse = PaginatedResponse<OpsRequestDetail>

export interface OpsLatencyHistogramBucket {
  range: string
  count: number
}

export interface OpsLatencyHistogramResponse {
  start_time: string
  end_time: string
  platform: string
  group_id?: number | null
  channel_id?: number | null

  total_requests: number
  buckets: OpsLatencyHistogramBucket[]
}

export interface OpsErrorTrendPoint {
  bucket_start: string
  error_count_total: number
  business_limited_count: number
  error_count_sla: number
  upstream_error_count_excl_429_529: number
  upstream_429_count: number
  upstream_529_count: number
}

export interface OpsErrorTrendResponse {
  bucket: string
  points: OpsErrorTrendPoint[]
}

export interface OpsErrorDistributionItem {
  status_code: number
  total: number
  sla: number
  business_limited: number
}

export interface OpsErrorDistributionResponse {
  total: number
  items: OpsErrorDistributionItem[]
}

export interface OpsDashboardSnapshotV2Response {
  generated_at: string
  overview: OpsDashboardOverview
  throughput_trend: OpsThroughputTrendResponse
  error_trend: OpsErrorTrendResponse
}

export type OpsOpenAITokenStatsTimeRange = '30m' | '1h' | '1d' | '15d' | '30d'

export interface OpsOpenAITokenStatsItem {
  model: string
  request_count: number
  avg_tokens_per_sec?: number | null
  avg_first_token_ms?: number | null
  total_output_tokens: number
  avg_duration_ms: number
  requests_with_first_token: number
}

export interface OpsOpenAITokenStatsResponse {
  time_range: OpsOpenAITokenStatsTimeRange
  start_time: string
  end_time: string
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  items: OpsOpenAITokenStatsItem[]
  total: number
  page?: number
  page_size?: number
  top_n?: number | null
}

export interface OpsOpenAITokenStatsParams {
  time_range?: OpsOpenAITokenStatsTimeRange
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  page?: number
  page_size?: number
  top_n?: number
}

export interface OpsSystemMetricsSnapshot {
  id: number
  created_at: string
  window_minutes: number

  cpu_usage_percent?: number | null
  memory_used_mb?: number | null
  memory_total_mb?: number | null
  memory_usage_percent?: number | null

  db_ok?: boolean | null
  redis_ok?: boolean | null

  // Config-derived limits (best-effort) for rendering "current vs max".
  db_max_open_conns?: number | null
  redis_pool_size?: number | null

  redis_conn_total?: number | null
  redis_conn_idle?: number | null

  db_conn_active?: number | null
  db_conn_idle?: number | null
  db_conn_waiting?: number | null

  goroutine_count?: number | null
  concurrency_queue_depth?: number | null
  account_switch_count?: number | null
}

export interface OpsJobHeartbeat {
  job_name: string
  last_run_at?: string | null
  last_success_at?: string | null
  last_error_at?: string | null
  last_error?: string | null
  last_duration_ms?: number | null
  last_result?: string | null
  updated_at: string
}

export interface PlatformConcurrencyInfo {
  platform: string
  current_in_use: number
  max_capacity: number
  load_percentage: number
  waiting_in_queue: number
}

export interface GroupConcurrencyInfo {
  group_id: number
  group_name: string
  platform: string
  current_in_use: number
  max_capacity: number
  load_percentage: number
  waiting_in_queue: number
}

export interface AccountConcurrencyInfo {
  account_id: number
  account_name?: string
  platform: string
  group_id: number
  group_name: string
  current_in_use: number
  max_capacity: number
  load_percentage: number
  waiting_in_queue: number
}

export interface OpsConcurrencyStatsResponse {
  enabled: boolean
  platform: Record<string, PlatformConcurrencyInfo>
  group: Record<string, GroupConcurrencyInfo>
  account: Record<string, AccountConcurrencyInfo>
  timestamp?: string
}

export interface UserConcurrencyInfo {
  user_id: number
  user_email: string
  username: string
  current_in_use: number
  max_capacity: number
  load_percentage: number
  waiting_in_queue: number
}

export interface OpsUserConcurrencyStatsResponse {
  enabled: boolean
  user: Record<string, UserConcurrencyInfo>
  timestamp?: string
}

export interface PlatformAvailability {
  platform: string
  total_accounts: number
  available_count: number
  rate_limit_count: number
  error_count: number
}

export interface GroupAvailability {
  group_id: number
  group_name: string
  platform: string
  total_accounts: number
  available_count: number
  rate_limit_count: number
  error_count: number
}

export interface AccountAvailability {
  account_id: number
  account_name: string
  platform: string
  group_id: number
  group_name: string
  status: string
  is_available: boolean
  is_rate_limited: boolean
  rate_limit_reset_at?: string
  rate_limit_remaining_sec?: number
  is_overloaded: boolean
  overload_until?: string
  overload_remaining_sec?: number
  has_error: boolean
  error_message?: string
}

export interface OpsAccountAvailabilityStatsResponse {
  enabled: boolean
  platform: Record<string, PlatformAvailability>
  group: Record<string, GroupAvailability>
  account: Record<string, AccountAvailability>
  timestamp?: string
}

export interface OpsRateSummary {
  current: number
  peak: number
  avg: number
}

export interface OpsRealtimeTrafficSummary {
  window: string
  start_time: string
  end_time: string
  platform: string
  group_id?: number | null
  channel_id?: number | null
  qps: OpsRateSummary
  tps: OpsRateSummary
}

export interface OpsRealtimeTrafficSummaryResponse {
  enabled: boolean
  summary: OpsRealtimeTrafficSummary | null
  timestamp?: string
}

export interface SubscribeQPSOptions {
  token?: string | null
  onOpen?: () => void
  onClose?: (event: CloseEvent) => void
  onError?: (event: Event) => void
  /**
   * Called when the server closes with an application close code that indicates
   * reconnecting is not useful (e.g. feature flag disabled).
   */
  onFatalClose?: (event: CloseEvent) => void
  /**
   * More granular status updates for UI (connecting/reconnecting/offline/etc).
   */
  onStatusChange?: (status: OpsWSStatus) => void
  /**
   * Called when a reconnect is scheduled (helps display "retry in Xs").
   */
  onReconnectScheduled?: (info: { attempt: number, delayMs: number }) => void
  wsBaseUrl?: string
  /**
   * Maximum reconnect attempts. Defaults to Infinity to keep the dashboard live.
   * Set to 0 to disable reconnect.
   */
  maxReconnectAttempts?: number
  reconnectBaseDelayMs?: number
  reconnectMaxDelayMs?: number
  /**
   * Stale connection detection (heartbeat-by-observation).
   * If no messages are received within this window, the socket is closed to trigger a reconnect.
   * Set to 0 to disable.
   */
  staleTimeoutMs?: number
  /**
   * How often to check staleness. Only used when `staleTimeoutMs > 0`.
   */
  staleCheckIntervalMs?: number
}
