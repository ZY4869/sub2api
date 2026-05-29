import type { OpsSeverity } from './typesShared'
export type AlertSeverity = 'critical' | 'warning' | 'info'
export type ThresholdMode = 'count' | 'percentage' | 'both'
export type MetricType =
  | 'success_rate'
  | 'error_rate'
  | 'upstream_error_rate'
  | 'cpu_usage_percent'
  | 'memory_usage_percent'
  | 'concurrency_queue_depth'
  | 'group_available_accounts'
  | 'group_available_ratio'
  | 'group_rate_limit_ratio'
  | 'account_rate_limited_count'
  | 'account_error_count'
  | 'account_error_ratio'
  | 'overload_account_count'
  | 'recovery_probe_started_count'
  | 'recovery_probe_success_count'
  | 'recovery_probe_retry_count'
  | 'recovery_probe_blacklisted_count'
  | 'gemini_billing_fallback_applied_count'
  | 'gemini_billing_fallback_miss_count'
export type Operator = '>' | '>=' | '<' | '<=' | '==' | '!='

export interface AlertRule {
  id?: number
  name: string
  description?: string
  enabled: boolean
  metric_type: MetricType
  operator: Operator
  threshold: number
  window_minutes: number
  sustained_minutes: number
  severity: OpsSeverity
  cooldown_minutes: number
  notify_email: boolean
  filters?: Record<string, any>
  created_at?: string
  updated_at?: string
  last_triggered_at?: string | null
}

export interface AlertEvent {
  id: number
  rule_id: number
  severity: OpsSeverity | string
  status: 'firing' | 'resolved' | 'manual_resolved' | string
  title?: string
  description?: string
  metric_value?: number
  threshold_value?: number
  dimensions?: Record<string, any>
  fired_at: string
  resolved_at?: string | null
  email_sent: boolean
  created_at: string
}

export interface EmailNotificationConfig {
  alert: {
    enabled: boolean
    recipients: string[]
    min_severity: AlertSeverity | ''
    rate_limit_per_hour: number
    batching_window_seconds: number
    include_resolved_alerts: boolean
  }
  report: {
    enabled: boolean
    recipients: string[]
    daily_summary_enabled: boolean
    daily_summary_schedule: string
    weekly_summary_enabled: boolean
    weekly_summary_schedule: string
    error_digest_enabled: boolean
    error_digest_schedule: string
    error_digest_min_count: number
    account_health_enabled: boolean
    account_health_schedule: string
    account_health_error_rate_threshold: number
  }
}

export interface OpsMetricThresholds {
  sla_percent_min?: number | null                 // SLA低于此值变红
  ttft_p99_ms_max?: number | null                 // TTFT P99高于此值变红
  request_error_rate_percent_max?: number | null  // 请求错误率高于此值变红
  upstream_error_rate_percent_max?: number | null // 上游错误率高于此值变红
}

export interface OpsDistributedLockSettings {
  enabled: boolean
  key: string
  ttl_seconds: number
}

export interface OpsAlertRuntimeSettings {
  evaluation_interval_seconds: number
  distributed_lock: OpsDistributedLockSettings
  silencing: {
    enabled: boolean
    global_until_rfc3339: string
    global_reason: string
    entries?: Array<{
      rule_id?: number
      severities?: Array<OpsSeverity | string>
      until_rfc3339: string
      reason: string
    }>
  }
  thresholds: OpsMetricThresholds // 指标阈值配置
}

export interface OpsAdvancedSettings {
  data_retention: OpsDataRetentionSettings
  aggregation: OpsAggregationSettings
  request_details_enabled: boolean
  request_detail_cleanup_enabled: boolean
  request_detail_cleanup_schedule: string
  request_detail_retention_days: number
  request_detail_payload_preview_limit_bytes: number
  success_sample_rate: number
  force_capture_slow_ms: number
  raw_export_max_rows: number
  ignore_count_tokens_errors: boolean
  ignore_context_canceled: boolean
  ignore_no_available_accounts: boolean
  ignore_invalid_api_key_errors: boolean
  ignore_insufficient_balance_errors: boolean
  display_openai_token_stats: boolean
  display_alert_events: boolean
  auto_refresh_enabled: boolean
  auto_refresh_interval_seconds: number
}

export interface OpsDataRetentionSettings {
  cleanup_enabled: boolean
  cleanup_schedule: string
  error_log_retention_days: number
  minute_metrics_retention_days: number
  hourly_metrics_retention_days: number
}

export interface OpsAggregationSettings {
  aggregation_enabled: boolean
}

export interface AlertEventsQuery {
  limit?: number
  status?: string
  severity?: string
  email_sent?: boolean
  time_range?: string
  start_time?: string
  end_time?: string
  before_fired_at?: string
  before_id?: number
  platform?: string
  group_id?: number
}
