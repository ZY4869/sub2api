import type {
  AccountModelImportFailure,
  AccountModelImportModelResult,
  AccountModelImportResult
} from '@/api/admin/accounts'
import type { ToastOptions, ToastType } from '@/types'

export type AccountModelImportTranslate = (
  key: string,
  named?: Record<string, unknown>
) => string

interface AccountModelImportErrorLike {
  message?: unknown
  response?: {
    data?: {
      detail?: unknown
      message?: unknown
    }
  }
}

interface AccountModelImportSummary {
  imported: number
  merged: number
  skipped: number
  failed: number
}

export interface AccountModelImportToastPayload {
  type: ToastType
  message: string
  options: ToastOptions
  summary: AccountModelImportSummary
}

const DETAIL_LIMIT = 8

export const MODEL_IMPORT_SYNC_EXPOSURES = ['whitelist', 'use_key', 'test', 'runtime'] as const

function extractAccountModelImportErrorMessage(error: unknown): string {
  const err = (error || {}) as AccountModelImportErrorLike
  const detail = typeof err.response?.data?.detail === 'string'
    ? err.response.data.detail.trim()
    : ''
  if (detail) {
    return detail
  }

  const responseMessage = typeof err.response?.data?.message === 'string'
    ? err.response.data.message.trim()
    : ''
  if (responseMessage) {
    return responseMessage
  }

  return typeof err.message === 'string' ? err.message.trim() : ''
}

function normalizeLegacyFailedModels(failedModels?: AccountModelImportFailure[]): AccountModelImportModelResult[] {
  if (!Array.isArray(failedModels)) {
    return []
  }
  return failedModels.map((item) => ({
    source_model: item.model,
    canonical_model: item.model,
    status: 'failed',
    reason_code: 'persist_failed',
    detail: item.error
  }))
}

function normalizeModelResults(
  result: AccountModelImportResult | null | undefined
): AccountModelImportModelResult[] {
  if (Array.isArray(result?.model_results) && result.model_results.length > 0) {
    return result.model_results
  }

  const fallbackResults: AccountModelImportModelResult[] = []
  for (const model of result?.imported_models || []) {
    fallbackResults.push({
      source_model: model,
      canonical_model: model,
      status: 'imported',
      reason_code: 'imported_new'
    })
  }
  fallbackResults.push(...normalizeLegacyFailedModels(result?.failed_models))
  return fallbackResults
}

function summarizeModelResults(
  result: AccountModelImportResult | null | undefined
): AccountModelImportSummary {
  const summary: AccountModelImportSummary = {
    imported: 0,
    merged: 0,
    skipped: 0,
    failed: 0
  }

  const modelResults = normalizeModelResults(result)
  if (modelResults.length > 0) {
    for (const item of modelResults) {
      if (item.status === 'imported') {
        summary.imported += 1
      } else if (item.status === 'merged') {
        summary.merged += 1
      } else if (item.status === 'skipped') {
        summary.skipped += 1
      } else if (item.status === 'failed') {
        summary.failed += 1
      }
    }
    return summary
  }

  summary.imported = Math.max(0, Number(result?.imported_count) || 0)
  summary.skipped = Math.max(0, Number(result?.skipped_count) || 0)
  summary.failed = Array.isArray(result?.failed_models) ? result.failed_models.length : 0
  return summary
}

function translateStatus(
  t: AccountModelImportTranslate,
  status: AccountModelImportModelResult['status']
): string {
  const key = `admin.accounts.modelImportStatus.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}

function translateReason(t: AccountModelImportTranslate, reasonCode?: string): string {
  if (!reasonCode) {
    return ''
  }
  const key = `admin.accounts.modelImportReason.${reasonCode}`
  const translated = t(key)
  return translated === key ? reasonCode : translated
}

function buildModelResultLine(
  t: AccountModelImportTranslate,
  item: AccountModelImportModelResult
): string {
  const sourceModel = item.source_model || '-'
  const canonicalSuffix = item.canonical_model && item.canonical_model !== item.source_model
    ? ` -> ${item.canonical_model}`
    : ''
  const statusLabel = translateStatus(t, item.status)
  const reasonLabel = translateReason(t, item.reason_code)
  const meta = reasonLabel && reasonLabel !== statusLabel
    ? `${statusLabel}: ${reasonLabel}`
    : statusLabel
  const detail = typeof item.detail === 'string' ? item.detail.trim() : ''
  return `${sourceModel}${canonicalSuffix} (${meta})${detail ? ` - ${detail}` : ''}`
}

function resolveToastType(
  summary: AccountModelImportSummary,
  probeNotice: string
): ToastType {
  if (summary.failed > 0 && summary.imported === 0 && summary.merged === 0) {
    return 'error'
  }
  if (summary.failed > 0 || summary.skipped > 0 || summary.merged > 0 || probeNotice) {
    return 'warning'
  }
  return 'success'
}

function buildDisplayedDetails(
  t: AccountModelImportTranslate,
  results: AccountModelImportModelResult[]
): { details: string[]; allDetails: string[] } {
  const interestingResults = results.filter((item) => item.status !== 'imported')
  const sourceResults = interestingResults.length > 0 ? interestingResults : results
  const allDetails = sourceResults.map((item) => buildModelResultLine(t, item))
  const details = allDetails.slice(0, DETAIL_LIMIT)
  if (allDetails.length > DETAIL_LIMIT) {
    details.push(t('admin.accounts.modelImportMore', { count: allDetails.length - DETAIL_LIMIT }))
  }
  return {
    details,
    allDetails
  }
}

export function resolveAccountModelImportProbeNoticeMessage(
  t: AccountModelImportTranslate,
  result: Pick<AccountModelImportResult, 'imported_count' | 'probe_source' | 'probe_notice'> | null | undefined
): string {
  const probeNotice = typeof result?.probe_notice === 'string'
    ? result.probe_notice.trim()
    : ''
  if (probeNotice) {
    return probeNotice
  }

  if (result?.probe_source === 'gemini_cli_default_fallback') {
    return t('admin.accounts.modelImportGeminiFallback', {
      count: Math.max(0, Number(result.imported_count) || 0)
    })
  }

  return ''
}

export function resolveAccountModelImportErrorMessage(
  t: AccountModelImportTranslate,
  error: unknown
): string {
  const message = extractAccountModelImportErrorMessage(error)
  if (message.includes('does not support real model probing') || message.includes('does not support model import')) {
    return t('admin.accounts.modelImportUnsupported')
  }
  return message || t('admin.accounts.modelImportFailed')
}

export function extractSyncableRegistryModels(
  result: AccountModelImportResult | null | undefined
): string[] {
  const models = new Set<string>()
  for (const item of normalizeModelResults(result)) {
    const registryModel = typeof item.registry_model === 'string' ? item.registry_model.trim() : ''
    if (!registryModel) {
      continue
    }
    if (item.status === 'failed' || item.reason_code === 'blocked_tombstone') {
      continue
    }
    models.add(registryModel)
  }
  return Array.from(models)
}

export function mergeAccountModelImportResults(
  results: AccountModelImportResult[]
): AccountModelImportResult | null {
  if (!results.length) {
    return null
  }

  const merged: AccountModelImportResult = {
    account_id: results[0].account_id,
    detected_models: [],
    imported_models: [],
    imported_count: 0,
    skipped_count: 0,
    failed_models: [],
    model_results: [],
    probe_source: 'upstream',
    probe_notice: '',
    trigger: results[0].trigger || 'manual'
  }

  const detectedSet = new Set<string>()
  const importedSet = new Set<string>()
  const notices: string[] = []

  for (const result of results) {
    for (const model of result.detected_models || []) {
      if (!detectedSet.has(model)) {
        detectedSet.add(model)
        merged.detected_models.push(model)
      }
    }
    for (const model of result.imported_models || []) {
      if (!importedSet.has(model)) {
        importedSet.add(model)
        merged.imported_models.push(model)
      }
    }
    merged.imported_count += Math.max(0, Number(result.imported_count) || 0)
    merged.skipped_count += Math.max(0, Number(result.skipped_count) || 0)
    merged.failed_models?.push(...(result.failed_models || []))
    merged.model_results?.push(...(result.model_results || []))
    if (result.probe_source === 'gemini_cli_default_fallback') {
      merged.probe_source = result.probe_source
    }
    if (typeof result.probe_notice === 'string' && result.probe_notice.trim()) {
      const notice = result.probe_notice.trim()
      if (!notices.includes(notice)) {
        notices.push(notice)
      }
    }
  }

  merged.probe_notice = notices.join(' | ')
  return merged
}

export function shouldInvalidateModelInventory(
  result: AccountModelImportResult | null | undefined
): boolean {
  const summary = summarizeModelResults(result)
  return summary.imported > 0 || summary.merged > 0
}

export function buildAccountModelImportToastPayload(
  t: AccountModelImportTranslate,
  result: AccountModelImportResult
): AccountModelImportToastPayload {
  const summary = summarizeModelResults(result)
  const probeNotice = resolveAccountModelImportProbeNoticeMessage(t, result)
  const modelResults = normalizeModelResults(result)
  const { details, allDetails } = buildDisplayedDetails(t, modelResults)
  const summaryNamed = { ...summary }
  const message = probeNotice
    ? `${t('admin.accounts.modelImportSummary', summaryNamed)} - ${probeNotice}`
    : t('admin.accounts.modelImportSummary', summaryNamed)
  const copyLines = [
    t('admin.accounts.modelImportResultTitle'),
    t('admin.accounts.modelImportSummary', summaryNamed)
  ]
  if (probeNotice) {
    copyLines.push(probeNotice)
  }
  if (allDetails.length > 0) {
    copyLines.push('', ...allDetails)
  }

  return {
    type: resolveToastType(summary, probeNotice),
    message,
    options: {
      title: t('admin.accounts.modelImportResultTitle'),
      details: details.length > 0 ? details : undefined,
      copyText: allDetails.length > 0 ? copyLines.join(String.fromCharCode(10)) : undefined,
      ...(details.length > 0 ? { persistent: true } : {})
    },
    summary
  }
}
