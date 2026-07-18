import type { PromptAuditDecision, PromptAuditRiskLevel } from './promptAuditTypes'

export interface PromptAuditEvent {
  id: number
  job_id: number
  request_id: string
  user_id?: number
  username_snapshot: string
  user_email_snapshot: string
  api_key_id?: number
  api_key_name_snapshot: string
  group_id?: number
  group_name: string
  provider: string
  endpoint: string
  protocol: string
  model: string
  prompt_hash: string
  redacted_preview: string
  full_prompt?: string
  stage: string
  decision: PromptAuditDecision
  risk_level: PromptAuditRiskLevel
  action: string
  categories: string[]
  matched_scanners: string[]
  scanner_scores: Record<string, number>
  scanner_evidence: Record<string, string>
  scanner_backend: string
  scanner_version: string
  guard_endpoint_id: string
  policy_id: string
  policy_version: number
  config_version: number
  chunk_total: number
  latency_ms: number
  issue_summaries?: Array<{ code: string; title: string; description: string; evidence_hash: string }>
  created_at: string
}

export interface PromptAuditEventFilter {
  page?: number
  page_size?: number
  decision?: string
  risk_level?: string
  endpoint?: string
  protocol?: string
  user_id?: number
  api_key_id?: number
  group_id?: number
  prompt_hash?: string
  request_id?: string
  keyword?: string
}

export interface PromptAuditEventList {
  items: PromptAuditEvent[]
  total: number
  page: number
  page_size: number
}

export interface PromptAuditDeletePreview {
  matched: number
  snapshot_max_id: number
  filter_hash: string
  confirmation_token?: string
  expires_at?: string
}

export interface PromptAuditDeleteResult {
  deleted: number
}
