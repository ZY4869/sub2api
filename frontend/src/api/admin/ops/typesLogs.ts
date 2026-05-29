import type { PaginatedResponse } from '@/types'
import type { OpsPhase, OpsSeverity } from './typesShared'
export interface OpsRuntimeLogConfig {
  level: 'debug' | 'info' | 'warn' | 'error'
  enable_sampling: boolean
  sampling_initial: number
  sampling_thereafter: number
  caller: boolean
  stacktrace_level: 'none' | 'error' | 'fatal'
  retention_days: number
  source?: string
  updated_at?: string
  updated_by_user_id?: number
}

export interface OpsSystemLog {
  id: number
  created_at: string
  level: string
  component: string
  message: string
  request_id?: string
  client_request_id?: string
  user_id?: number | null
  account_id?: number | null
  platform?: string
  model?: string
  extra?: Record<string, any>
}

export type OpsSystemLogListResponse = PaginatedResponse<OpsSystemLog>

export interface OpsSystemLogQuery {
  page?: number
  page_size?: number
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h' | '7d' | '30d'
  start_time?: string
  end_time?: string
  level?: string
  component?: string
  request_id?: string
  client_request_id?: string
  user_id?: number | null
  account_id?: number | null
  platform?: string
  model?: string
  q?: string
}

export interface OpsSystemLogCleanupRequest {
  start_time?: string
  end_time?: string
  level?: string
  component?: string
  request_id?: string
  client_request_id?: string
  user_id?: number | null
  account_id?: number | null
  platform?: string
  model?: string
  q?: string
}

export interface OpsSystemLogSinkHealth {
  queue_depth: number
  queue_capacity: number
  dropped_count: number
  write_failed_count: number
  written_count: number
  avg_write_delay_ms: number
  last_error?: string
}

export interface OpsErrorLog {
  id: number
  created_at: string

  // Standardized classification
  phase: OpsPhase
  type: string
  error_owner: 'client' | 'provider' | 'platform' | string
  error_source: 'client_request' | 'upstream_http' | 'gateway' | string

  severity: OpsSeverity
  status_code: number
  platform: string
  model: string

  is_retryable: boolean
  retry_count: number

  resolved: boolean
  resolved_at?: string | null
  resolved_by_user_id?: number | null
  resolved_retry_id?: number | null

  client_request_id: string
  request_id: string
  message: string

  user_id?: number | null
  user_email: string
  api_key_id?: number | null
  account_id?: number | null
  account_name: string
  group_id?: number | null
  group_name: string

  client_ip?: string | null
  request_path?: string
  stream?: boolean
  inbound_endpoint?: string
  upstream_endpoint?: string
  requested_model?: string
  upstream_model?: string
  request_type?: number | null
  upstream_url?: string
  gemini_surface?: string
  billing_rule_id?: string
  probe_action?: string
}

export interface OpsErrorDetail extends OpsErrorLog {
  error_body: string
  user_agent: string

  // Upstream context (optional; enriched by gateway services)
  upstream_status_code?: number | null
  upstream_error_message?: string
  upstream_error_detail?: string
  upstream_errors?: string

  auth_latency_ms?: number | null
  routing_latency_ms?: number | null
  upstream_latency_ms?: number | null
  response_latency_ms?: number | null
  time_to_first_token_ms?: number | null

  request_body: string
  request_body_truncated: boolean
  request_body_bytes?: number | null

  is_business_limited: boolean
}

export type OpsErrorLogsResponse = PaginatedResponse<OpsErrorLog>

export type OpsErrorListView = 'errors' | 'excluded' | 'all'

export type OpsErrorListQueryParams = {
  page?: number
  page_size?: number
  time_range?: string
  start_time?: string
  end_time?: string
  platform?: string
  group_id?: number | null
  account_id?: number | null

  phase?: string
  error_owner?: string
  error_source?: string
  resolved?: string
  view?: OpsErrorListView

  q?: string
  status_codes?: string
  status_codes_other?: string
  gemini_surface?: string
  billing_rule_id?: string
  probe_action?: string
}
