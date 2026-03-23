/**
 * Admin Accounts API endpoints
 * Handles AI platform account management for administrators
 */

import { apiClient } from '../client'
import type {
  Account,
  ArchiveGroupAccountsRequest,
  ArchiveGroupAccountsResult,
  BatchArchiveAccountsRequest,
  BatchArchiveAccountsResult,
  BatchCreateAccountsRequest,
  BatchCreateAccountsResult,
  CreateAccountRequest,
  UpdateAccountRequest,
  PaginatedResponse,
  AccountUsageInfo,
  WindowStats,
  ClaudeModel,
  AccountUsageStatsResponse,
  TempUnschedulableStatus,
  AdminDataPayload,
  AdminDataImportResult,
  CheckMixedChannelRequest,
  CheckMixedChannelResponse
} from '@/types'

/**
 * List all accounts with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters
 * @returns Paginated list of accounts
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    platform?: string
    type?: string
    status?: string
    group?: string
    search?: string
    lite?: string
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<Account>> {
  const { data } = await apiClient.get<PaginatedResponse<Account>>('/admin/accounts', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

export interface AccountListWithEtagResult {
  notModified: boolean
  etag: string | null
  data: PaginatedResponse<Account> | null
}

export async function listWithEtag(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    platform?: string
    type?: string
    status?: string
    search?: string
    lite?: string
  },
  options?: {
    signal?: AbortSignal
    etag?: string | null
  }
): Promise<AccountListWithEtagResult> {
  const headers: Record<string, string> = {}
  if (options?.etag) {
    headers['If-None-Match'] = options.etag
  }

  const response = await apiClient.get<PaginatedResponse<Account>>('/admin/accounts', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    headers,
    signal: options?.signal,
    validateStatus: (status) => (status >= 200 && status < 300) || status === 304
  })

  const etagHeader = typeof response.headers?.etag === 'string' ? response.headers.etag : null
  if (response.status === 304) {
    return {
      notModified: true,
      etag: etagHeader,
      data: null
    }
  }

  return {
    notModified: false,
    etag: etagHeader,
    data: response.data
  }
}

/**
 * Get account by ID
 * @param id - Account ID
 * @returns Account details
 */
export async function getById(id: number): Promise<Account> {
  const { data } = await apiClient.get<Account>(`/admin/accounts/${id}`)
  return data
}

/**
 * Create new account
 * @param accountData - Account data
 * @returns Created account
 */
export async function create(accountData: CreateAccountRequest): Promise<Account> {
  const { data } = await apiClient.post<Account>('/admin/accounts', accountData)
  return data
}

/**
 * Update account
 * @param id - Account ID
 * @param updates - Fields to update
 * @returns Updated account
 */
export async function update(id: number, updates: UpdateAccountRequest): Promise<Account> {
  const { data } = await apiClient.put<Account>(`/admin/accounts/${id}`, updates)
  return data
}

/**
 * Check mixed-channel risk for account-group binding.
 */
export async function checkMixedChannelRisk(
  payload: CheckMixedChannelRequest
): Promise<CheckMixedChannelResponse> {
  const { data } = await apiClient.post<CheckMixedChannelResponse>('/admin/accounts/check-mixed-channel', payload)
  return data
}

/**
 * Delete account
 * @param id - Account ID
 * @returns Success confirmation
 */
export async function deleteAccount(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/accounts/${id}`)
  return data
}

/**
 * Toggle account status
 * @param id - Account ID
 * @param status - New status
 * @returns Updated account
 */
export async function toggleStatus(id: number, status: 'active' | 'inactive'): Promise<Account> {
  return update(id, { status })
}

/**
 * Test account connectivity
 * @param id - Account ID
 * @returns Test result
 */
export async function testAccount(id: number): Promise<{
  success: boolean
  message: string
  latency_ms?: number
}> {
  const { data } = await apiClient.post<{
    success: boolean
    message: string
    latency_ms?: number
  }>(`/admin/accounts/${id}/test`)
  return data
}

/**
 * Refresh account credentials
 * @param id - Account ID
 * @returns Updated account
 */
export async function refreshCredentials(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/accounts/${id}/refresh`)
  return data
}

export interface CopilotDeviceFlowStartResult {
  session_id: string
  device_code?: string
  user_code: string
  verification_uri: string
  verification_uri_complete?: string
  expires_in: number
  interval: number
}

export interface CopilotDeviceFlowUser {
  login?: string
  email?: string
  name?: string
}

export interface CopilotDeviceFlowPollResult {
  session_id: string
  status: 'pending' | 'completed'
  interval: number
  expires_in?: number
  user?: CopilotDeviceFlowUser
}

export async function startCopilotDeviceFlow(payload: {
  proxy_id?: number | null
} = {}): Promise<CopilotDeviceFlowStartResult> {
  const { data } = await apiClient.post<CopilotDeviceFlowStartResult>('/admin/copilot/device/start', payload)
  return data
}

export async function pollCopilotDeviceFlow(sessionId: string): Promise<CopilotDeviceFlowPollResult> {
  const { data } = await apiClient.post<CopilotDeviceFlowPollResult>('/admin/copilot/device/poll', {
    session_id: sessionId
  })
  return data
}

export async function createCopilotAccountFromDevice(payload: {
  session_id: string
  proxy_id?: number | null
  name: string
  notes?: string | null
  concurrency?: number
  load_factor?: number | null
  priority?: number
  rate_multiplier?: number
  group_ids?: number[]
  expires_at?: number | null
  auto_pause_on_expired?: boolean
  confirm_mixed_channel_risk?: boolean
}): Promise<Account> {
  const { data } = await apiClient.post<Account>('/admin/copilot/create-from-device', payload)
  return data
}

export async function reauthorizeCopilotAccountFromDevice(
  id: number,
  payload: { session_id: string }
): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/copilot/accounts/${id}/reauthorize-from-device`, payload)
  return data
}

export async function refreshCopilotAccount(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/copilot/accounts/${id}/refresh`)
  return data
}

export interface KiroAuthUrlResult {
  auth_url: string
  session_id: string
  redirect_uri: string
  state: string
}

export interface KiroExchangeCodeResult {
  access_token: string
  refresh_token?: string
  expires_at?: string
  auth_method?: string
  provider?: string
  client_id?: string
  client_secret?: string
  client_id_hash?: string
  start_url?: string
  region?: string
  profile_arn?: string
  email?: string
  username?: string
  display_name?: string
}

export async function generateKiroAuthUrl(payload: {
  proxy_id?: number | null
  redirect_uri?: string
  method?: string
  start_url?: string
  region?: string
} = {}): Promise<KiroAuthUrlResult> {
  const { data } = await apiClient.post<KiroAuthUrlResult>('/admin/kiro/oauth/auth-url', payload)
  return data
}

export async function exchangeKiroAuthCode(payload: {
  session_id: string
  code: string
  state: string
  proxy_id?: number | null
}): Promise<KiroExchangeCodeResult> {
  const { data } = await apiClient.post<KiroExchangeCodeResult>('/admin/kiro/oauth/exchange-code', payload)
  return data
}

export async function createKiroAccountFromOAuth(payload: {
  session_id: string
  code: string
  state: string
  proxy_id?: number | null
  name: string
  notes?: string | null
  concurrency?: number
  load_factor?: number | null
  priority?: number
  rate_multiplier?: number
  group_ids?: number[]
  expires_at?: number | null
  auto_pause_on_expired?: boolean
  confirm_mixed_channel_risk?: boolean
}): Promise<Account> {
  const { data } = await apiClient.post<Account>('/admin/kiro/create-from-oauth', payload)
  return data
}

export async function reauthorizeKiroAccountFromOAuth(
  id: number,
  payload: {
    session_id: string
    code: string
    state: string
    proxy_id?: number | null
  }
): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/kiro/accounts/${id}/reauthorize-from-oauth`, payload)
  return data
}

export async function refreshKiroAccount(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/kiro/accounts/${id}/refresh`)
  return data
}

/**
 * Get account usage statistics
 * @param id - Account ID
 * @param days - Number of days (default: 30)
 * @returns Account usage statistics with history, summary, and models
 */
export async function getStats(id: number, days: number = 30): Promise<AccountUsageStatsResponse> {
  const { data } = await apiClient.get<AccountUsageStatsResponse>(`/admin/accounts/${id}/stats`, {
    params: { days }
  })
  return data
}

/**
 * Clear account error
 * @param id - Account ID
 * @returns Updated account
 */
export async function clearError(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/accounts/${id}/clear-error`)
  return data
}

/**
 * Get account usage information (5h/7d window)
 * @param id - Account ID
 * @param options.force - Whether to bypass server-side usage cache
 * @param options.source - Usage source to query
 * @returns Account usage info
 */
export async function getUsage(
  id: number,
  options?: { force?: boolean; source?: "passive" | "active" }
): Promise<AccountUsageInfo> {
  const params = {
    ...(options?.force ? { force: "1" } : {}),
    ...(options?.source ? { source: options.source } : {}),
  }
  const { data } = await apiClient.get<AccountUsageInfo>(`/admin/accounts/${id}/usage`, { params })
  return data
}

/**
 * Clear account rate limit status
 * @param id - Account ID
 * @returns Updated account
 */
export async function clearRateLimit(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(
    `/admin/accounts/${id}/clear-rate-limit`
  )
  return data
}

/**
 * Recover account runtime state in one call
 * @param id - Account ID
 * @returns Updated account
 */
export async function recoverState(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/accounts/${id}/recover-state`)
  return data
}

/**
 * Reset account quota usage
 * @param id - Account ID
 * @returns Updated account
 */
export async function resetAccountQuota(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(
    `/admin/accounts/${id}/reset-quota`
  )
  return data
}

/**
 * Get temporary unschedulable status
 * @param id - Account ID
 * @returns Status with detail state if active
 */
export async function getTempUnschedulableStatus(id: number): Promise<TempUnschedulableStatus> {
  const { data } = await apiClient.get<TempUnschedulableStatus>(
    `/admin/accounts/${id}/temp-unschedulable`
  )
  return data
}

/**
 * Reset temporary unschedulable status
 * @param id - Account ID
 * @returns Success confirmation
 */
export async function resetTempUnschedulable(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/accounts/${id}/temp-unschedulable`
  )
  return data
}

/**
 * Generate OAuth authorization URL
 * @param endpoint - API endpoint path
 * @param config - Proxy configuration
 * @returns Auth URL and session ID
 */
export async function generateAuthUrl(
  endpoint: string,
  config: { proxy_id?: number }
): Promise<{ auth_url: string; session_id: string }> {
  const { data } = await apiClient.post<{ auth_url: string; session_id: string }>(endpoint, config)
  return data
}

/**
 * Exchange authorization code for tokens
 * @param endpoint - API endpoint path
 * @param exchangeData - Session ID, code, and optional proxy config
 * @returns Token information
 */
export async function exchangeCode(
  endpoint: string,
  exchangeData: { session_id: string; code: string; state?: string; proxy_id?: number }
): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post<Record<string, unknown>>(endpoint, exchangeData)
  return data
}

export async function batchCreateAccounts(
  payload: BatchCreateAccountsRequest
): Promise<BatchCreateAccountsResult> {
  const { data } = await apiClient.post<BatchCreateAccountsResult>('/admin/accounts/batch-create', payload)
  return data
}

export async function batchArchiveAccounts(
  payload: BatchArchiveAccountsRequest
): Promise<BatchArchiveAccountsResult> {
  const { data } = await apiClient.post<BatchArchiveAccountsResult>('/admin/accounts/batch-archive', payload)
  return data
}

export async function archiveGroupAccounts(
  payload: ArchiveGroupAccountsRequest
): Promise<ArchiveGroupAccountsResult> {
  const { data } = await apiClient.post<ArchiveGroupAccountsResult>('/admin/accounts/archive-group', payload)
  return data
}

/**
 * Batch create accounts
 * @param accounts - Array of account data
 * @returns Results of batch creation
 */
export async function batchCreate(accounts: CreateAccountRequest[]): Promise<{
  success: number
  failed: number
  results: Array<{ success: boolean; account?: Account; error?: string }>
}> {
  const { data } = await apiClient.post<{
    success: number
    failed: number
    results: Array<{ success: boolean; account?: Account; error?: string }>
  }>('/admin/accounts/batch', { accounts })
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
export async function bulkUpdate(
  accountIds: number[],
  updates: Record<string, unknown>
): Promise<{
  success: number
  failed: number
  success_ids?: number[]
  failed_ids?: number[]
  results: Array<{ account_id: number; success: boolean; error?: string }>
  }> {
  const { data } = await apiClient.post<{
    success: number
    failed: number
    success_ids?: number[]
    failed_ids?: number[]
    results: Array<{ account_id: number; success: boolean; error?: string }>
  }>('/admin/accounts/bulk-update', {
    account_ids: accountIds,
    ...updates
  })
  return data
}

/**
 * Get account today statistics
 * @param id - Account ID
 * @returns Today's stats (requests, tokens, cost)
 */
export async function getTodayStats(id: number): Promise<WindowStats> {
  const { data } = await apiClient.get<WindowStats>(`/admin/accounts/${id}/today-stats`)
  return data
}

export interface BatchTodayStatsResponse {
  stats: Record<string, WindowStats>
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
  probe_source?: 'upstream' | 'gemini_cli_default_fallback' | 'kiro_builtin_catalog'
  probe_notice?: string
  trigger: string
}

export async function importModels(
  id: number,
  payload: { trigger?: string } = {}
): Promise<AccountModelImportResult> {
  const { data } = await apiClient.post<AccountModelImportResult>(`/admin/accounts/${id}/import-models`, payload)
  return data
}

/**
 * Get available models for an account
 * @param id - Account ID
 * @returns List of available models for this account
 */
export async function getAvailableModels(id: number): Promise<ClaudeModel[]> {
  const { data } = await apiClient.get<ClaudeModel[]>(`/admin/accounts/${id}/models`)
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
 * Validate Sora session token and exchange to access token
 * @param sessionToken - Sora session token
 * @param proxyId - Optional proxy ID
 * @param endpoint - API endpoint path
 * @returns Token information including access_token
 */
export async function validateSoraSessionToken(
  sessionToken: string,
  proxyId?: number | null,
  endpoint: string = '/admin/sora/st2at'
): Promise<Record<string, unknown>> {
  const payload: { session_token: string; proxy_id?: number } = {
    session_token: sessionToken
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

export const accountsAPI = {
  list,
  listWithEtag,
  getById,
  create,
  update,
  checkMixedChannelRisk,
  delete: deleteAccount,
  toggleStatus,
  testAccount,
  refreshCredentials,
  startCopilotDeviceFlow,
  pollCopilotDeviceFlow,
  createCopilotAccountFromDevice,
  reauthorizeCopilotAccountFromDevice,
  refreshCopilotAccount,
  generateKiroAuthUrl,
  exchangeKiroAuthCode,
  createKiroAccountFromOAuth,
  reauthorizeKiroAccountFromOAuth,
  refreshKiroAccount,
  getStats,
  clearError,
  getUsage,
  getTodayStats,
  getBatchTodayStats,
  clearRateLimit,
  recoverState,
  resetAccountQuota,
  getTempUnschedulableStatus,
  resetTempUnschedulable,
  setSchedulable,
  getAvailableModels,
  importModels,
  generateAuthUrl,
  exchangeCode,
  refreshOpenAIToken,
  validateSoraSessionToken,
  batchCreateAccounts,
  batchArchiveAccounts,
  archiveGroupAccounts,
  batchCreate,
  batchUpdateCredentials,
  bulkUpdate,
  previewFromCrs,
  syncFromCrs,
  exportData,
  importData,
  getAntigravityDefaultModelMapping,
  batchClearError,
  batchRefresh
}

export default accountsAPI
