export type PromptAuditMode = 'off' | 'async_audit' | 'blocking'
export type PromptAuditDecision = 'pass' | 'flag' | 'critical'
export type PromptAuditRiskLevel = 'low' | 'medium' | 'high' | 'critical'

export interface PromptAuditEndpoint {
  id: string
  name: string
  protocol: string
  base_url: string
  model: string
  timeout_ms: number
  input_limit: number
  enabled: boolean
  has_token?: boolean
  token_status?: string
}

export interface PromptAuditUpdateEndpoint extends PromptAuditEndpoint {
  token?: string
  clear_token: boolean
}

export interface PromptAuditConfig {
  enabled: boolean
  blocking_enabled: boolean
  store_pass_events: boolean
  effective_mode: PromptAuditMode
  strategy: string
  worker_count: number
  queue_capacity: number
  scanners: string[]
  all_groups: boolean
  group_ids: number[]
  endpoints: PromptAuditEndpoint[]
  config_version: number
  updated_at: string
  updated_by: number
  change_summary: string
}

export interface PromptAuditUpdateConfig {
  expected_config_version: number
  enabled: boolean
  blocking_enabled: boolean
  store_pass_events: boolean
  strategy: string
  worker_count: number
  queue_capacity: number
  scanners: string[]
  all_groups: boolean
  group_ids: number[]
  endpoints: PromptAuditUpdateEndpoint[]
}

export interface PromptAuditQueueStats {
  queued: number
  processing: number
  retry: number
  done: number
  failed: number
}

export interface PromptAuditProbeResult {
  ok: boolean
  status: string
  message: string
  error_code?: string
  http_status?: number
  retryable: boolean
  token_applied: boolean
  latency_ms: number
  checked_at: string
}

export interface PromptAuditMetrics {
  allow: number
  flag: number
  block: number
  unavailable: number
  timeout: number
  failover: number
  bulkhead: number
  enqueued: number
  dropped: number
  processed: number
  failed: number
}

export interface PromptAuditRuntime {
  process_status: string
  effective_mode: PromptAuditMode
  expected_config_version: number
  active_config_version: number
  config_loaded_at?: string
  config_load_error?: string
  worker_total: number
  worker_active: number
  queue_capacity: number
  queue: PromptAuditQueueStats
  database_status: string
  redis_status: string
  endpoints: Record<string, PromptAuditProbeResult>
  guard_metrics: PromptAuditMetrics
  enqueued_total: number
  dropped_total: number
  processed_total: number
  failed_total: number
  worker_heartbeat_at?: string
  last_processed_at?: string
  last_error_code?: string
  last_error_message?: string
}
