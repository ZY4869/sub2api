import type { PaginatedResponse } from '@/types'
export type OpsRequestTraceSort = 'created_at_desc' | 'duration_desc'

export interface OpsRequestTraceFilter {
  page?: number
  page_size?: number
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h' | '7d' | '30d'
  start_time?: string
  end_time?: string
  status?: string
  platform?: string
  protocol_in?: string
  protocol_out?: string
  channel?: string
  route_path?: string
  request_type?: string
  finish_reason?: string
  capture_reason?: string
  requested_model?: string
  upstream_model?: string
  request_id?: string
  client_request_id?: string
  upstream_request_id?: string
  gemini_surface?: string
  billing_rule_id?: string
  probe_action?: string
  q?: string
  user_id?: number | null
  api_key_id?: number | null
  account_id?: number | null
  group_id?: number | null
  status_code?: number | null
  stream?: boolean
  has_tools?: boolean
  has_thinking?: boolean
  raw_available?: boolean
  sampled?: boolean
  sort?: OpsRequestTraceSort
}

export type OpsRequestTraceQuery = OpsRequestTraceFilter

export interface OpsRequestTraceListItem {
  id: number
  created_at: string
  request_id: string
  client_request_id: string
  upstream_request_id: string
  platform: string
  protocol_in: string
  protocol_out: string
  channel: string
  route_path: string
  request_type: string
  user_id?: number | null
  api_key_id?: number | null
  account_id?: number | null
  group_id?: number | null
  account_name?: string
  group_name?: string
  requested_model: string
  upstream_model: string
  actual_upstream_model: string
  gemini_surface?: string
  billing_rule_id?: string
  probe_action?: string
  status: string
  status_code: number
  upstream_status_code?: number | null
  duration_ms: number
  ttft_ms?: number | null
  input_tokens: number
  output_tokens: number
  total_tokens: number
  finish_reason: string
  prompt_block_reason: string
  stream: boolean
  has_tools: boolean
  tool_kinds?: string[]
  has_thinking: boolean
  thinking_source: string
  thinking_level: string
  thinking_budget?: number | null
  media_resolution: string
  count_tokens_source: string
  capture_reason: string
  sampled: boolean
  raw_available: boolean
  raw_access_allowed: boolean
}

export interface OpsRequestTraceAuditLog {
  id: number
  trace_id?: number | null
  operator_id: number
  action: 'view_raw' | 'export_csv' | 'cleanup_filter' | 'cleanup_expired' | string
  meta_json: string
  created_at: string
}

export type OpsRequestTraceCleanupMode = 'filter' | 'expired'

export interface OpsRequestTraceCleanupRequest {
  mode: OpsRequestTraceCleanupMode
  filter?: OpsRequestTraceFilter
}

export interface OpsRequestTraceCleanupResult {
  mode: OpsRequestTraceCleanupMode
  deleted_traces: number
  deleted_audits: number
  cutoff?: string
}

export interface OpsRequestTraceDetail extends OpsRequestTraceListItem {
  inbound_request_json: string
  normalized_request_json: string
  upstream_request_json: string
  upstream_response_json: string
  gateway_response_json: string
  tool_trace_json: string
  request_headers_json: string
  response_headers_json: string
  audits: OpsRequestTraceAuditLog[]
}

export interface OpsRequestTraceRawDetail {
  id: number
  request_id: string
  raw_request: string
  raw_response: string
}

export interface OpsRequestTraceSummaryTotals {
  request_count: number
  success_count: number
  error_count: number
  stream_count: number
  tool_count: number
  thinking_count: number
  raw_available_count: number
  avg_duration_ms: number
  p50_duration_ms: number
  p95_duration_ms: number
  p99_duration_ms: number
}

export interface OpsRequestTraceSummaryPoint {
  bucket_start: string
  request_count: number
  error_count: number
  p50_duration_ms: number
  p95_duration_ms: number
  p99_duration_ms: number
}

export interface OpsRequestTraceSummaryBreakdownItem {
  key: string
  label: string
  count: number
}

export interface OpsRequestTraceSummary {
  start_time: string
  end_time: string
  totals: OpsRequestTraceSummaryTotals
  trend: OpsRequestTraceSummaryPoint[]
  status_distribution: OpsRequestTraceSummaryBreakdownItem[]
  finish_reason_distribution: OpsRequestTraceSummaryBreakdownItem[]
  protocol_pair_distribution: OpsRequestTraceSummaryBreakdownItem[]
  model_distribution: OpsRequestTraceSummaryBreakdownItem[]
  capability_distribution: OpsRequestTraceSummaryBreakdownItem[]
  raw_access_allowed: boolean
}

export type OpsRequestTraceListResponse = PaginatedResponse<OpsRequestTraceListItem>

export type OpsRequestSubjectType = 'account' | 'group' | 'api_key'

export interface OpsRequestSubjectInsightsParams {
  subject_type: OpsRequestSubjectType
  subject_id: number
  time_range?: OpsRequestTraceFilter['time_range']
  start_time?: string
  end_time?: string
}

export interface OpsRequestSubjectReference {
  type: OpsRequestSubjectType
  id: number
  name: string
  user_id?: number | null
  user_email?: string
  group_id?: number | null
  group_name?: string
}

export interface OpsRequestSubjectSummaryDay {
  date: string
  account_cost: number
  user_cost: number
  standard_cost: number
  requests: number
  tokens: number
}

export interface OpsRequestSubjectSummaryCostDay {
  date: string
  label: string
  account_cost: number
  user_cost: number
  standard_cost: number
  requests: number
}

export interface OpsRequestSubjectSummaryRequestDay {
  date: string
  label: string
  requests: number
  account_cost: number
  user_cost: number
  standard_cost: number
}

export interface OpsRequestSubjectSummary {
  total_account_cost: number
  total_user_cost: number
  total_standard_cost: number
  total_requests: number
  total_tokens: number
  avg_daily_account_cost: number
  avg_daily_user_cost: number
  avg_daily_standard_cost: number
  avg_daily_requests: number
  avg_daily_tokens: number
  avg_duration_ms: number
  active_days: number
  window_days: number
  today?: OpsRequestSubjectSummaryDay | null
  highest_cost_day?: OpsRequestSubjectSummaryCostDay | null
  highest_request_day?: OpsRequestSubjectSummaryRequestDay | null
}

export interface OpsRequestPreviewCoverage {
  total_requests: number
  preview_available_count: number
  preview_available_rate: number
  normalized_count: number
  upstream_request_count: number
  upstream_response_count: number
  gateway_response_count: number
  tool_trace_count: number
}

export interface OpsRequestSubjectHistoryPoint {
  date: string
  label: string
  requests: number
  tokens: number
  cost: number
  actual_cost: number
  user_cost: number
}

export interface OpsRequestSubjectModelStat {
  model: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  total_tokens: number
  cost: number
  actual_cost: number
}

export interface OpsRequestSubjectEndpointStat {
  endpoint: string
  requests: number
  total_tokens: number
  cost: number
  actual_cost: number
}

export interface OpsRequestSubjectInsights {
  subject: OpsRequestSubjectReference
  summary: OpsRequestSubjectSummary
  history: OpsRequestSubjectHistoryPoint[]
  models: OpsRequestSubjectModelStat[]
  endpoints: OpsRequestSubjectEndpointStat[]
  upstream_endpoints: OpsRequestSubjectEndpointStat[]
  request_preview_coverage: OpsRequestPreviewCoverage
}
