import { apiClient } from '../client'
import type {
  Account,
  AccountTodayStats,
  AdminAccountImportGroupBindingRequest,
  AdminAccountImportGroupBindingResult,
  AdminAccountImportJob,
  AdminAccountImportJobCreateResult,
  AdminAccountModelOption,
  AdminDataImportResult,
  AdminDataPayload,
  UnarchiveAccountsResult
} from '@/types'
import type {
  AccountManualModel,
  AccountTestMode,
  AccountTestModelInputMode
} from './accounts'

export async function unarchiveAccounts(accountIds: number[]): Promise<UnarchiveAccountsResult> {
  const { data } = await apiClient.post<UnarchiveAccountsResult>('/admin/accounts/unarchive', {
    account_ids: accountIds
  })
  return data
}

export interface BlacklistRetestResult {
  account_id: number
  success: boolean
  restored: boolean
  error_message?: string
  response_text?: string
  latency_ms?: number
}

export interface BlacklistRetestResponse {
  results: BlacklistRetestResult[]
}

export interface BlacklistRetestRequestPayload {
  account_ids: number[]
  model_id?: string
  model_input_mode?: AccountTestModelInputMode
  manual_model_id?: string
  source_protocol?: 'openai' | 'anthropic' | 'gemini'
  target_provider?: string
  target_model_id?: string
}

export type BatchAccountTestModelInputMode = 'auto' | AccountTestModelInputMode

export interface BatchAccountTestRequestPayload {
  account_ids: number[]
  model_id?: string
  model?: string
  model_input_mode?: BatchAccountTestModelInputMode
  manual_model_id?: string
  source_protocol?: 'openai' | 'anthropic' | 'gemini'
  target_provider?: string
  target_model_id?: string
  prompt?: string
  test_mode?: AccountTestMode
}

export interface BatchAccountTestResult {
  account_id: number
  account_name?: string
  platform?: string
  status: 'success' | 'failed'
  error_message?: string
  response_text?: string
  latency_ms?: number
  resolved_model_id?: string
  resolved_platform?: string
  resolved_source_protocol?: 'openai' | 'anthropic' | 'gemini'
  blacklist_advice_decision?: BlacklistAdviceDecision
  current_lifecycle_state?: string
  lifecycle_reason_code?: string
}

export interface BatchAccountTestResponse {
  results: BatchAccountTestResult[]
}

export async function retestBlacklistedAccounts(
  payload: BlacklistRetestRequestPayload
): Promise<BlacklistRetestResponse> {
  const { data } = await apiClient.post<BlacklistRetestResponse>('/admin/accounts/blacklist/retest', {
    ...payload
  })
  return data
}

export async function getBlacklistRetestModels(accountIds: number[]): Promise<AdminAccountModelOption[]> {
  const { data } = await apiClient.post<AdminAccountModelOption[]>('/admin/accounts/blacklist/retest-models', {
    account_ids: accountIds
  })
  return Array.isArray(data) ? data : []
}

export async function getBatchTestModels(accountIds: number[]): Promise<AdminAccountModelOption[]> {
  const { data } = await apiClient.post<AdminAccountModelOption[]>('/admin/accounts/batch-test-models', {
    account_ids: accountIds
  })
  return Array.isArray(data) ? data : []
}

export async function batchTestAccounts(
  payload: BatchAccountTestRequestPayload
): Promise<BatchAccountTestResponse> {
  const { data } = await apiClient.post<BatchAccountTestResponse>('/admin/accounts/batch-test', payload)
  return data
}

export interface BlacklistedBatchDeleteFailure {
  id: number
  reason: string
}

export interface BlacklistedBatchDeleteResult {
  deleted_ids: number[]
  failed: BlacklistedBatchDeleteFailure[]
  deleted_count: number
  failed_count: number
}

export interface BlacklistedBatchRestoreFailure {
  id: number
  reason: string
}

export interface BlacklistedBatchRestoreResult {
  restored_ids: number[]
  failed: BlacklistedBatchRestoreFailure[]
  restored_count: number
  failed_count: number
}

export async function batchDeleteBlacklistedAccounts(payload: {
  ids?: number[]
  delete_all?: boolean
}): Promise<BlacklistedBatchDeleteResult> {
  const { data } = await apiClient.post<BlacklistedBatchDeleteResult>(
    '/admin/accounts/blacklist/batch-delete',
    payload
  )
  return data
}

export async function restoreBlacklistedAccount(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/accounts/${id}/blacklist/restore`)
  return data
}

export async function restoreBlacklistedAccounts(payload: {
  ids: number[]
}): Promise<BlacklistedBatchRestoreResult> {
  const { data } = await apiClient.post<BlacklistedBatchRestoreResult>(
    '/admin/accounts/blacklist/restore',
    payload
  )
  return data
}

export type BlacklistAdviceDecision =
  | 'auto_blacklisted'
  | 'recommend_blacklist'
  | 'not_recommended'

export interface BlacklistAdvicePayload {
  decision: BlacklistAdviceDecision
  reason_code?: string
  reason_message?: string
  already_blacklisted?: boolean
  feedback_fingerprint?: string
  collect_feedback?: boolean
  platform?: string
  status_code?: number
  error_code?: string
  message_keywords?: string[]
}

export interface BlacklistFeedbackPayload {
  fingerprint?: string
  advice_decision?: BlacklistAdviceDecision | string
  action: 'blacklist'
  platform?: string
  status_code?: number
  error_code?: string
  message_keywords?: string[]
}

export interface BlacklistAccountPayload {
  source?: 'manual_menu' | 'test_modal' | string
  feedback?: BlacklistFeedbackPayload
}

export async function blacklist(id: number, payload?: BlacklistAccountPayload): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/accounts/${id}/blacklist`, payload)
  return data
}

/**
 * Batch update credentials fields for multiple accounts
 * @param request - Batch update request containing account IDs, field name, and value
 * @returns Results of batch update
 */
export async function batchUpdateCredentials(request: {
  account_ids: number[]
  field: string
  value: any
}): Promise<{
  success: number
  failed: number
  results: Array<{ account_id: number; success: boolean; error?: string }>
}> {
  const { data } = await apiClient.post<{
    success: number
    failed: number
    results: Array<{ account_id: number; success: boolean; error?: string }>
  }>('/admin/accounts/batch-update-credentials', request)
  return data
}

/**
 * Bulk update multiple accounts
 * @param accountIds - Array of account IDs
 * @param updates - Fields to update
 * @returns Success confirmation
 */
export interface BulkUpdateAccountsFilters {
  platform?: string
  type?: string
  status?: string
  group?: string
  search?: string
  lifecycle?: string
  privacy_mode?: string
  limited_view?: string
  limited_reason?: string
  runtime_view?: string
}

export type BulkUpdateAccountsTarget =
  | { account_ids: number[] }
  | { filters: BulkUpdateAccountsFilters }

type BulkUpdateAccountsResponse = {
  success: number
  failed: number
  success_ids?: number[]
  failed_ids?: number[]
  results: Array<{ account_id: number; success: boolean; error?: string }>
}

export async function bulkUpdate(
  accountIds: number[],
  updates: Record<string, unknown>
): Promise<BulkUpdateAccountsResponse>
export async function bulkUpdate(
  target: BulkUpdateAccountsTarget,
  updates: Record<string, unknown>
): Promise<BulkUpdateAccountsResponse>
export async function bulkUpdate(
  target: number[] | BulkUpdateAccountsTarget,
  updates: Record<string, unknown>
): Promise<BulkUpdateAccountsResponse> {
  const body = Array.isArray(target)
    ? { account_ids: target, ...updates }
    : { ...target, ...updates }
  const { data } = await apiClient.post<BulkUpdateAccountsResponse>('/admin/accounts/bulk-update', body)
  return data
}

/**
 * Get account today statistics
 * @param id - Account ID
 * @returns Today's stats (requests, tokens, cost)
 */
export async function getTodayStats(id: number): Promise<AccountTodayStats> {
  const { data } = await apiClient.get<AccountTodayStats>(`/admin/accounts/${id}/today-stats`)
  return data
}

export interface BatchTodayStatsResponse {
  stats: Record<string, AccountTodayStats>
}

/**
 * 批量获取多个账号的今日统计
 * @param accountIds - 账号 ID 列表
 * @returns 以账号 ID（字符串）为键的统计映射
 */
export async function getBatchTodayStats(accountIds: number[]): Promise<BatchTodayStatsResponse> {
  const { data } = await apiClient.post<BatchTodayStatsResponse>('/admin/accounts/today-stats/batch', {
    account_ids: accountIds
  })
  return data
}

/**
 * Set account schedulable status
 * @param id - Account ID
 * @param schedulable - Whether the account should participate in scheduling
 * @returns Updated account
 */
export async function setSchedulable(id: number, schedulable: boolean): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/accounts/${id}/schedulable`, {
    schedulable
  })
  return data
}

export interface AccountModelImportFailure {
  model: string
  error: string
}

export interface AccountModelImportModelResult {
  source_model: string
  canonical_model?: string
  registry_model?: string
  status: 'imported' | 'merged' | 'skipped' | 'failed'
  reason_code:
    | 'imported_new'
    | 'merged_canonical'
    | 'duplicate_canonical'
    | 'already_exists'
    | 'blocked_tombstone'
    | 'invalid_model_id'
    | 'unsupported_runtime_platform'
    | 'persist_failed'
    | string
  detail?: string
}

export interface AccountModelImportResult {
  account_id: number
  detected_models: string[]
  imported_models: string[]
  imported_count: number
  skipped_count: number
  failed_models?: AccountModelImportFailure[]
  model_results?: AccountModelImportModelResult[]
  probe_source?: string
  probe_notice?: string
  trigger: string
}

export interface AccountModelDiagnosticsPreview {
  public_id: string
  alias_id?: string
  source_id: string
  display_name: string
  platform: string
}

export interface AccountModelDiagnosticsAPIKeyExposure {
  api_key_id: number
  api_key_name: string
  model_display_mode: string
  model_patterns?: string[]
  public_models: AccountModelDiagnosticsPreview[]
}

export interface AccountModelDiagnosticsGroupExposure {
  group_id: number
  group_name: string
  group_platform: string
  public_models: AccountModelDiagnosticsPreview[]
  api_keys: AccountModelDiagnosticsAPIKeyExposure[]
  warnings?: string[]
}

export interface AccountModelDiagnosticsResponse {
  account_id: number
  routing_platform: string
  status: string
  probe_source?: string
  probe_notice?: string
  resolved_upstream_url?: string
  resolved_upstream_host?: string
  resolved_upstream_service?: string
  saved_models: string[]
  detected_models: string[]
  public_models_preview: AccountModelDiagnosticsPreview[]
  group_exposures: AccountModelDiagnosticsGroupExposure[]
  warnings: string[]
}

export async function importModels(
  id: number,
  payload: { trigger?: string; models?: string[] } = {}
): Promise<AccountModelImportResult> {
  const { data } = await apiClient.post<AccountModelImportResult>(`/admin/accounts/${id}/import-models`, payload)
  return data
}

export async function diagnoseAccountModels(
  id: number,
  payload: { refresh?: boolean } = {}
): Promise<AccountModelDiagnosticsResponse> {
  const { data } = await apiClient.post<AccountModelDiagnosticsResponse>(
    `/admin/accounts/${id}/model-diagnostics`,
    payload
  )
  return data
}

export interface ProtocolGatewayProbeModel {
  id: string
  display_name: string
  provider?: string
  provider_label?: string
  registry_state: 'existing' | 'missing'
  registry_model_id?: string
  source_protocol?: 'openai' | 'anthropic' | 'gemini'
  upstream_source?: 'official' | 'verified_extra'
  availability?: 'callable' | 'uncallable'
  availability_reason?: string
}

export interface ProtocolGatewayProbeResponse {
  probe_source: string
  probe_notice?: string
  resolved_upstream_url?: string
  resolved_upstream_host?: string
  resolved_upstream_service?: string
  models: ProtocolGatewayProbeModel[]
}

export interface AccountProbeModelsPayload {
  platform: string
  type: string
  credentials?: Record<string, unknown>
  extra?: Record<string, unknown>
  manual_models?: AccountManualModel[]
  proxy_id?: number | null
}

export type AccountProbeModelsResponse = ProtocolGatewayProbeResponse

export async function probeModels(payload: AccountProbeModelsPayload): Promise<AccountProbeModelsResponse> {
  const { data } = await apiClient.post<AccountProbeModelsResponse>('/admin/accounts/probe-models', payload)
  return data
}

export async function probeProtocolGatewayModels(payload: {
  gateway_protocol: string
  accepted_protocols?: string[]
  base_url?: string
  api_key: string
  target_provider?: string
  target_model_id?: string
  manual_models?: AccountManualModel[]
  proxy_id?: number | null
}): Promise<ProtocolGatewayProbeResponse> {
  const { data } = await apiClient.post<ProtocolGatewayProbeResponse>(
    '/admin/accounts/protocol-gateway/probe-models',
    payload
  )
  return data
}

/**
 * Get available models for an account
 * @param id - Account ID
 * @returns List of available models for this account
 */
export async function getAvailableModels(
  id: number,
  options?: { refresh?: boolean }
): Promise<AdminAccountModelOption[]> {
  const { data } = await apiClient.get<AdminAccountModelOption[]>(`/admin/accounts/${id}/models`, {
    params: options?.refresh ? { refresh: true } : undefined
  })
  return data
}

export interface CRSPreviewAccount {
  crs_account_id: string
  kind: string
  name: string
  platform: string
  type: string
}

export interface PreviewFromCRSResult {
  new_accounts: CRSPreviewAccount[]
  existing_accounts: CRSPreviewAccount[]
}

export async function previewFromCrs(params: {
  base_url: string
  username: string
  password: string
}): Promise<PreviewFromCRSResult> {
  const { data } = await apiClient.post<PreviewFromCRSResult>('/admin/accounts/sync/crs/preview', params)
  return data
}

export async function syncFromCrs(params: {
  base_url: string
  username: string
  password: string
  sync_proxies?: boolean
  selected_account_ids?: string[]
}): Promise<{
  created: number
  updated: number
  skipped: number
  failed: number
  items: Array<{
    crs_account_id: string
    kind: string
    name: string
    action: string
    error?: string
  }>
}> {
  const { data } = await apiClient.post<{
    created: number
    updated: number
    skipped: number
    failed: number
    items: Array<{
      crs_account_id: string
      kind: string
      name: string
      action: string
      error?: string
    }>
  }>('/admin/accounts/sync/crs', params)
  return data
}

export async function exportData(options?: {
  ids?: number[]
  filters?: {
    platform?: string
    type?: string
    status?: string
    search?: string
  }
  includeProxies?: boolean
}): Promise<AdminDataPayload> {
  const params: Record<string, string> = {}
  if (options?.ids && options.ids.length > 0) {
    params.ids = options.ids.join(',')
  } else if (options?.filters) {
    const { platform, type, status, search } = options.filters
    if (platform) params.platform = platform
    if (type) params.type = type
    if (status) params.status = status
    if (search) params.search = search
  }
  if (options?.includeProxies === false) {
    params.include_proxies = 'false'
  }
  const { data } = await apiClient.get<AdminDataPayload>('/admin/accounts/data', { params })
  return data
}

export async function importData(payload: {
  data: AdminDataPayload
  skip_default_group_bind?: boolean
}): Promise<AdminDataImportResult> {
  const { data } = await apiClient.post<AdminDataImportResult>('/admin/accounts/data', {
    data: payload.data,
    skip_default_group_bind: payload.skip_default_group_bind
  })
  return data
}

export async function createImportJob(payload: {
  data: AdminDataPayload
  skip_default_group_bind?: boolean
}): Promise<AdminAccountImportJobCreateResult> {
  const { data } = await apiClient.post<AdminAccountImportJobCreateResult>('/admin/accounts/data/import-jobs', {
    data: payload.data,
    skip_default_group_bind: payload.skip_default_group_bind
  })
  return data
}

export async function getImportJob(jobId: string): Promise<AdminAccountImportJob> {
  const { data } = await apiClient.get<AdminAccountImportJob>(`/admin/accounts/data/import-jobs/${jobId}`)
  return data
}

export async function cancelImportJob(jobId: string): Promise<AdminAccountImportJob> {
  const { data } = await apiClient.post<AdminAccountImportJob>(`/admin/accounts/data/import-jobs/${jobId}/cancel`)
  return data
}

export async function bindImportJobGroups(
  jobId: string,
  payload: AdminAccountImportGroupBindingRequest
): Promise<AdminAccountImportGroupBindingResult> {
  const { data } = await apiClient.post<AdminAccountImportGroupBindingResult>(
    `/admin/accounts/data/import-jobs/${jobId}/group-bindings`,
    payload
  )
  return data
}

/**
 * Get Antigravity default model mapping from backend
 * @returns Default model mapping (from -> to)
 */
export async function getAntigravityDefaultModelMapping(): Promise<Record<string, string>> {
  const { data } = await apiClient.get<Record<string, string>>(
    '/admin/accounts/antigravity/default-model-mapping'
  )
  return data
}

/**
 * Refresh OpenAI token using refresh token
 * @param refreshToken - The refresh token
 * @param proxyId - Optional proxy ID
 * @returns Token information including access_token, email, etc.
 */
export async function refreshOpenAIToken(
  refreshToken: string,
  proxyId?: number | null,
  endpoint: string = '/admin/openai/refresh-token'
): Promise<Record<string, unknown>> {
  const payload: { refresh_token: string; proxy_id?: number } = {
    refresh_token: refreshToken
  }
  if (proxyId) {
    payload.proxy_id = proxyId
  }
  const { data } = await apiClient.post<Record<string, unknown>>(endpoint, payload)
  return data
}

/**
 * Batch operation result type
 */
export interface BatchOperationResult {
  total: number
  success: number
  failed: number
  errors?: Array<{ account_id: number; error: string }>
  warnings?: Array<{ account_id: number; warning: string }>
}

/**
 * Batch clear account errors
 * @param accountIds - Array of account IDs
 * @returns Batch operation result
 */
export async function batchClearError(accountIds: number[]): Promise<BatchOperationResult> {
  const { data } = await apiClient.post<BatchOperationResult>('/admin/accounts/batch-clear-error', {
    account_ids: accountIds
  })
  return data
}

/**
 * Batch refresh account credentials
 * @param accountIds - Array of account IDs
 * @returns Batch operation result
 */
export async function batchRefresh(accountIds: number[]): Promise<BatchOperationResult> {
  const { data } = await apiClient.post<BatchOperationResult>('/admin/accounts/batch-refresh', {
    account_ids: accountIds,
  }, {
    timeout: 120000  // 120s timeout for large batch refreshes
  })
  return data
}
