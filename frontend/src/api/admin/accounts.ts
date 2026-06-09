/**
 * Admin Accounts API endpoints
 * Handles AI platform account management for administrators
 */

import { apiClient } from '../client'
import type {
  Account,
  AccountRuntimeSummary,
  AccountStatusSummary,
  AccountProxyRestoreResult,
  ArchivedAccountGroupSummary,
  BatchArchiveAccountsRequest,
  BatchArchiveAccountsResult,
  CreateAccountRequest,
  UpdateAccountRequest,
  PaginatedResponse,
  AccountUsageInfo,
  AccountUsageStatsResponse,
  TempUnschedulableStatus,
  CheckMixedChannelRequest,
  CheckMixedChannelResponse,
  AccountDaily5HTriggerSettings,
  AccountDaily5HTriggerSettingsView
} from '@/types'
import {
  batchClearError,
  batchDeleteBlacklistedAccounts,
  batchRefresh,
  batchTestAccounts,
  batchUpdateCredentials,
  bindImportJobGroups,
  blacklist,
  bulkUpdate,
  cancelImportJob,
  createImportJob,
  diagnoseAccountModels,
  exportData,
  getImportJob,
  getAntigravityDefaultModelMapping,
  getAvailableModels,
  getBatchTestModels,
  getBatchTodayStats,
  getBlacklistRetestModels,
  getTodayStats,
  importData,
  importModels,
  previewFromCrs,
  probeModels,
  probeProtocolGatewayModels,
  refreshOpenAIToken,
  restoreBlacklistedAccount,
  restoreBlacklistedAccounts,
  retestBlacklistedAccounts,
  setSchedulable,
  syncFromCrs,
  unarchiveAccounts
} from './accountOperations'
export {
  batchClearError,
  batchDeleteBlacklistedAccounts,
  batchRefresh,
  batchTestAccounts,
  batchUpdateCredentials,
  bindImportJobGroups,
  blacklist,
  bulkUpdate,
  cancelImportJob,
  createImportJob,
  diagnoseAccountModels,
  exportData,
  getImportJob,
  getAntigravityDefaultModelMapping,
  getAvailableModels,
  getBatchTestModels,
  getBatchTodayStats,
  getBlacklistRetestModels,
  getTodayStats,
  importData,
  importModels,
  previewFromCrs,
  probeModels,
  probeProtocolGatewayModels,
  refreshOpenAIToken,
  restoreBlacklistedAccount,
  restoreBlacklistedAccounts,
  retestBlacklistedAccounts,
  setSchedulable,
  syncFromCrs,
  unarchiveAccounts
}
export type {
  AccountModelDiagnosticsAPIKeyExposure,
  AccountModelDiagnosticsGroupExposure,
  AccountModelDiagnosticsPreview,
  AccountModelDiagnosticsResponse,
  AccountModelImportFailure,
  AccountModelImportModelResult,
  AccountModelImportResult,
  AccountProbeModelsPayload,
  AccountProbeModelsResponse,
  BatchAccountTestModelInputMode,
  BatchAccountTestRequestPayload,
  BatchAccountTestResponse,
  BatchAccountTestResult,
  BatchOperationResult,
  BatchTodayStatsResponse,
  BlacklistAccountPayload,
  BlacklistAdviceDecision,
  BlacklistAdvicePayload,
  BlacklistFeedbackPayload,
  BlacklistRetestRequestPayload,
  BlacklistRetestResponse,
  BlacklistRetestResult,
  BlacklistedBatchDeleteFailure,
  BlacklistedBatchDeleteResult,
  BlacklistedBatchRestoreFailure,
  BlacklistedBatchRestoreResult,
  BulkUpdateAccountsFilters,
  BulkUpdateAccountsTarget,
  CRSPreviewAccount,
  PreviewFromCRSResult,
  ProtocolGatewayProbeModel,
  ProtocolGatewayProbeResponse
} from './accountOperations'

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
    privacy_mode?: string
    search?: string
    lite?: string
    lifecycle?: string
    limited_view?: string
    limited_reason?: string
    runtime_view?: string
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

export interface AccountRuntimeSummaryWithEtagResult {
  notModified: boolean
  etag: string | null
  data: AccountRuntimeSummary | null
}

export async function listWithEtag(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    platform?: string
    type?: string
    group?: string
    status?: string
    privacy_mode?: string
    search?: string
    lite?: string
    lifecycle?: string
    limited_view?: string
    limited_reason?: string
    runtime_view?: string
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
    validateStatus: (status: number) => (status >= 200 && status < 300) || status === 304
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
export async function getById(
  id: number,
  options?: {
    signal?: AbortSignal
  }
): Promise<Account> {
  const { data } = await apiClient.get<Account>(`/admin/accounts/${id}`, {
    signal: options?.signal
  })
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

export async function restoreOriginalProxy(id: number): Promise<AccountProxyRestoreResult> {
  const { data } = await apiClient.post<AccountProxyRestoreResult>(
    `/admin/accounts/${id}/restore-original-proxy`
  )
  return data
}

export async function getDaily5HTriggerSettings(): Promise<AccountDaily5HTriggerSettingsView> {
  const { data } = await apiClient.get<AccountDaily5HTriggerSettingsView>(
    '/admin/accounts/daily-5h-trigger-settings'
  )
  return data
}

export async function updateDaily5HTriggerSettings(
  payload: AccountDaily5HTriggerSettings
): Promise<AccountDaily5HTriggerSettingsView> {
  const { data } = await apiClient.put<AccountDaily5HTriggerSettingsView>(
    '/admin/accounts/daily-5h-trigger-settings',
    payload
  )
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
export type AccountTestMode = 'real_forward' | 'health_check'
export type AccountTestModelInputMode = 'catalog' | 'manual'

export interface AccountManualModel {
  model_id: string
  request_alias?: string
  provider?: string
  source_protocol?: 'openai' | 'anthropic' | 'gemini'
}

export interface AccountTestRequestPayload {
  model_id?: string
  model?: string
  model_input_mode?: AccountTestModelInputMode
  manual_model_id?: string
  request_alias?: string
  prompt?: string
  source_protocol?: 'openai' | 'anthropic' | 'gemini'
  target_provider?: string
  target_model_id?: string
  test_mode?: AccountTestMode
}

export interface GrokImportParseError {
  index: number
  message: string
}

export interface GrokImportPreviewItem {
  index: number
  name: string
  type: string
  detected_kind: string
  credential_masked: string
  source_pool?: string
  grok_tier: string
  priority: number
  concurrency: number
  status: 'ready' | 'skipped' | 'failed'
  reason?: string
}

export interface GrokImportPreviewResponse {
  detected_kind?: string
  total: number
  items: GrokImportPreviewItem[]
  errors?: GrokImportParseError[]
}

export interface GrokImportResultItem {
  index: number
  name: string
  type: string
  status: 'created' | 'skipped' | 'failed'
  reason?: string
  account_id?: number
  source_pool?: string
}

export interface GrokImportResult {
  detected_kind?: string
  created: number
  skipped: number
  failed: number
  errors?: GrokImportParseError[]
  results: GrokImportResultItem[]
}

export async function testAccount(id: number, payload: AccountTestRequestPayload = {}): Promise<{
  success: boolean
  message: string
  latency_ms?: number
}> {
  const { data } = await apiClient.post<{
    success: boolean
    message: string
    latency_ms?: number
  }>(`/admin/accounts/${id}/test`, payload)
  return data
}

export async function previewGrokImport(payload: {
  content: string
  skip_default_group_bind?: boolean
}): Promise<GrokImportPreviewResponse> {
  const { data } = await apiClient.post<GrokImportPreviewResponse>('/admin/grok/import/preview', payload)
  return data
}

export async function importGrok(payload: {
  content: string
  skip_default_group_bind?: boolean
}): Promise<GrokImportResult> {
  const { data } = await apiClient.post<GrokImportResult>('/admin/grok/import', payload)
  return data
}

function buildAdminStreamUrl(path: string): string {
  const base = String(apiClient.defaults.baseURL || '').replace(/\/+$/, '')
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${base}${normalizedPath}`
}

export async function testGrokAccount(
  id: number,
  payload: AccountTestRequestPayload = {},
  options?: {
    signal?: AbortSignal
  }
): Promise<Response> {
  return fetch(buildAdminStreamUrl(`/admin/grok/accounts/${id}/test`), {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload),
    signal: options?.signal
  })
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

export async function setPrivacy(id: number): Promise<Account> {
  const { data } = await apiClient.post<Account>(`/admin/accounts/${id}/set-privacy`)
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
  auto_renew_enabled?: boolean
  auto_renew_period?: 'month' | 'quarter' | 'year'
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

export async function batchArchiveAccounts(
  payload: BatchArchiveAccountsRequest
): Promise<BatchArchiveAccountsResult> {
  const { data } = await apiClient.post<BatchArchiveAccountsResult>('/admin/accounts/batch-archive', payload)
  return data
}

export async function listArchivedGroups(filters?: {
  platform?: string
  type?: string
  status?: string
  group?: string
  search?: string
}): Promise<ArchivedAccountGroupSummary[]> {
  const { data } = await apiClient.get<ArchivedAccountGroupSummary[]>('/admin/accounts/archived-groups', {
    params: filters
  })
  return Array.isArray(data) ? data.map(normalizeArchivedGroupSummary) : []
}

export async function getStatusSummary(filters?: {
  platform?: string
  type?: string
  group?: string
  privacy_mode?: string
  search?: string
  lifecycle?: string
  limited_view?: string
  limited_reason?: string
  runtime_view?: string
}): Promise<AccountStatusSummary> {
  const { data } = await apiClient.get<AccountStatusSummary>('/admin/accounts/summary', {
    params: filters
  })
  return normalizeAccountStatusSummary(data)
}

export async function getRuntimeSummary(filters?: {
  platform?: string
  type?: string
  group?: string
  privacy_mode?: string
  search?: string
  lifecycle?: string
  limited_view?: string
  limited_reason?: string
  runtime_view?: string
}): Promise<AccountRuntimeSummary> {
  const { data } = await apiClient.get<AccountRuntimeSummary>('/admin/accounts/runtime-summary', {
    params: filters
  })
  return normalizeAccountRuntimeSummary(data)
}

export async function getRuntimeSummaryWithEtag(
  filters?: {
    platform?: string
    type?: string
    group?: string
    privacy_mode?: string
    search?: string
    lifecycle?: string
    limited_view?: string
    limited_reason?: string
    runtime_view?: string
  },
  options?: {
    signal?: AbortSignal
    etag?: string | null
  }
): Promise<AccountRuntimeSummaryWithEtagResult> {
  const headers: Record<string, string> = {}
  if (options?.etag) {
    headers['If-None-Match'] = options.etag
  }

  const response = await apiClient.get<AccountRuntimeSummary>('/admin/accounts/runtime-summary', {
    params: filters,
    headers,
    signal: options?.signal,
    validateStatus: (status: number) => (status >= 200 && status < 300) || status === 304
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
    data: normalizeAccountRuntimeSummary(response.data)
  }
}

function normalizeArchivedGroupSummary(raw: any): ArchivedAccountGroupSummary {
  return {
    group_id: Number(raw?.group_id ?? raw?.GroupID ?? 0),
    group_name: String(raw?.group_name ?? raw?.GroupName ?? ''),
    total_count: Number(raw?.total_count ?? raw?.TotalCount ?? 0),
    available_count: Number(raw?.available_count ?? raw?.AvailableCount ?? 0),
    invalid_count: Number(raw?.invalid_count ?? raw?.InvalidCount ?? 0),
    latest_updated_at: String(raw?.latest_updated_at ?? raw?.LatestUpdatedAt ?? '')
  }
}

function normalizeAccountStatusSummary(raw: any): AccountStatusSummary {
  return {
    total: Number(raw?.total ?? raw?.Total ?? 0),
    by_status: {
      active: Number(raw?.by_status?.active ?? raw?.ByStatus?.active ?? 0),
      inactive: Number(raw?.by_status?.inactive ?? raw?.ByStatus?.inactive ?? 0),
      error: Number(raw?.by_status?.error ?? raw?.ByStatus?.error ?? 0)
    },
    rate_limited: Number(raw?.rate_limited ?? raw?.RateLimited ?? 0),
    temp_unschedulable: Number(raw?.temp_unschedulable ?? raw?.TempUnschedulable ?? 0),
    overloaded: Number(raw?.overloaded ?? raw?.Overloaded ?? 0),
    paused: Number(raw?.paused ?? raw?.Paused ?? 0),
    in_use: Number(raw?.in_use ?? raw?.InUse ?? 0),
    remaining_available: Number(raw?.remaining_available ?? raw?.RemainingAvailable ?? 0),
    by_platform: normalizePlatformCounts(raw?.by_platform ?? raw?.ByPlatform),
    limited_breakdown: {
      total: Number(raw?.limited_breakdown?.total ?? raw?.LimitedBreakdown?.total ?? 0),
      rate_429: Number(raw?.limited_breakdown?.rate_429 ?? raw?.LimitedBreakdown?.rate_429 ?? 0),
      usage_5h: Number(raw?.limited_breakdown?.usage_5h ?? raw?.LimitedBreakdown?.usage_5h ?? 0),
      usage_7d: Number(raw?.limited_breakdown?.usage_7d ?? raw?.LimitedBreakdown?.usage_7d ?? 0),
      usage_7d_all: Number(raw?.limited_breakdown?.usage_7d_all ?? raw?.LimitedBreakdown?.usage_7d_all ?? 0)
    }
  }
}

function normalizeAccountRuntimeSummary(raw: any): AccountRuntimeSummary {
  return {
    in_use: Number(raw?.in_use ?? raw?.InUse ?? 0)
  }
}

function normalizePlatformCounts(raw: any): Partial<Record<Account['platform'], number>> {
  if (!raw || typeof raw !== 'object') {
    return {}
  }
  return Object.entries(raw).reduce<Partial<Record<Account['platform'], number>>>((acc, [key, value]) => {
    acc[key as Account['platform']] = Number(value ?? 0)
    return acc
  }, {})
}

export const accountsAPI = {
  list,
  listWithEtag,
  getById,
  create,
  update,
  restoreOriginalProxy,
  getDaily5HTriggerSettings,
  updateDaily5HTriggerSettings,
  checkMixedChannelRisk,
  delete: deleteAccount,
  toggleStatus,
  testAccount,
  previewGrokImport,
  importGrok,
  testGrokAccount,
  refreshCredentials,
  setPrivacy,
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
  diagnoseAccountModels,
  probeModels,
  probeProtocolGatewayModels,
  generateAuthUrl,
  exchangeCode,
  refreshOpenAIToken,
  batchArchiveAccounts,
  getStatusSummary,
  getRuntimeSummary,
  getRuntimeSummaryWithEtag,
  listArchivedGroups,
  unarchiveAccounts,
  retestBlacklistedAccounts,
  getBlacklistRetestModels,
  restoreBlacklistedAccount,
  restoreBlacklistedAccounts,
  getBatchTestModels,
  batchTestAccounts,
  batchDeleteBlacklistedAccounts,
  blacklist,
  batchUpdateCredentials,
  bulkUpdate,
  bindImportJobGroups,
  cancelImportJob,
  createImportJob,
  previewFromCrs,
  syncFromCrs,
  getImportJob,
  exportData,
  importData,
  getAntigravityDefaultModelMapping,
  batchClearError,
  batchRefresh
}

export default accountsAPI
