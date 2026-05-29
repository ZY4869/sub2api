import type { OpsRetryMode } from './typesShared'
export interface OpsRetryRequest {
  mode: OpsRetryMode
  pinned_account_id?: number
  force?: boolean
}

export interface OpsRetryAttempt {
  id: number
  created_at: string
  requested_by_user_id: number
  source_error_id: number
  mode: string
  pinned_account_id?: number | null
  pinned_account_name?: string

  status: string
  started_at?: string | null
  finished_at?: string | null
  duration_ms?: number | null

  success?: boolean | null
  http_status_code?: number | null
  upstream_request_id?: string | null
  used_account_id?: number | null
  used_account_name?: string
  response_preview?: string | null
  response_truncated?: boolean | null

  result_request_id?: string | null
  result_error_id?: number | null
  error_message?: string | null
}

export type OpsUpstreamErrorEvent = {
  at_unix_ms?: number
  platform?: string
  account_id?: number
  account_name?: string
  upstream_status_code?: number
  upstream_request_id?: string
  upstream_request_body?: string
  kind?: string
  message?: string
  detail?: string
}

export interface OpsRetryResult {
  attempt_id: number
  mode: OpsRetryMode
  status: 'running' | 'succeeded' | 'failed' | string

  pinned_account_id?: number | null
  used_account_id?: number | null

  http_status_code: number
  upstream_request_id: string

  response_preview: string
  response_truncated: boolean

  error_message: string

  started_at: string
  finished_at: string
  duration_ms: number
}
