import type { GatewayAcceptedProtocol } from './accounts'
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
