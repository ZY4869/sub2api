import type { AccountPlatform, AccountType } from '@/types'
import {
  normalizeGeminiAPIKeyVariant,
  normalizeGeminiOAuthType,
} from '@/utils/geminiAccount'

export type GoogleBatchArchiveBillingMode = 'log_only' | 'archive_charge'
export type GoogleBatchArchiveTargetKind = 'ai_studio' | 'vertex' | 'none'

export interface GoogleBatchArchiveFormState {
  enabled: boolean
  autoPrefetchEnabled: boolean
  retentionDays: number
  billingMode: GoogleBatchArchiveBillingMode
  downloadPriceUSD: number
  allowVertexBatchOverflow: boolean
  acceptAIStudioBatchOverflow: boolean
}

export const DEFAULT_GOOGLE_BATCH_ARCHIVE_RETENTION_DAYS = 7
export const DEFAULT_GOOGLE_BATCH_ARCHIVE_DOWNLOAD_PRICE_USD = 0

const archiveExtraKeys = [
  'batch_archive_enabled',
  'batch_archive_auto_prefetch_enabled',
  'batch_archive_retention_days',
  'batch_archive_billing_mode',
  'batch_archive_download_price_usd',
  'allow_vertex_batch_overflow',
  'accept_aistudio_batch_overflow',
] as const

export function normalizeGoogleBatchArchiveBillingMode(
  value: unknown,
): GoogleBatchArchiveBillingMode {
  return value === 'archive_charge' ? 'archive_charge' : 'log_only'
}

export function createDefaultGoogleBatchArchiveFormState(): GoogleBatchArchiveFormState {
  return {
    enabled: false,
    autoPrefetchEnabled: false,
    retentionDays: DEFAULT_GOOGLE_BATCH_ARCHIVE_RETENTION_DAYS,
    billingMode: 'log_only',
    downloadPriceUSD: DEFAULT_GOOGLE_BATCH_ARCHIVE_DOWNLOAD_PRICE_USD,
    allowVertexBatchOverflow: false,
    acceptAIStudioBatchOverflow: false,
  }
}

export function readGoogleBatchArchiveFormState(
  value?: Record<string, unknown> | null,
): GoogleBatchArchiveFormState {
  const fallback = createDefaultGoogleBatchArchiveFormState()
  if (!value) {
    return fallback
  }

  const retentionDays = Number(value.batch_archive_retention_days)
  const downloadPriceUSD = Number(value.batch_archive_download_price_usd)

  return {
    enabled: value.batch_archive_enabled === true,
    autoPrefetchEnabled: value.batch_archive_auto_prefetch_enabled === true,
    retentionDays:
      Number.isFinite(retentionDays) && retentionDays > 0
        ? retentionDays
        : fallback.retentionDays,
    billingMode: normalizeGoogleBatchArchiveBillingMode(
      value.batch_archive_billing_mode,
    ),
    downloadPriceUSD:
      Number.isFinite(downloadPriceUSD) && downloadPriceUSD >= 0
        ? downloadPriceUSD
        : fallback.downloadPriceUSD,
    allowVertexBatchOverflow: value.allow_vertex_batch_overflow === true,
    acceptAIStudioBatchOverflow: value.accept_aistudio_batch_overflow === true,
  }
}

export function isGeminiAIStudioBatchArchiveAccount(
  platform: AccountPlatform | string | undefined,
  type: AccountType | string | undefined,
  credentials?: Record<string, unknown> | null,
): boolean {
  if (platform !== 'gemini' || type !== 'apikey') {
    return false
  }
  return (
    normalizeGeminiAPIKeyVariant(credentials?.gemini_api_variant) === 'ai_studio'
  )
}

export function isGeminiVertexBatchArchiveAccount(
  platform: AccountPlatform | string | undefined,
  type: AccountType | string | undefined,
  credentials?: Record<string, unknown> | null,
): boolean {
  if (platform !== 'gemini' || type !== 'oauth') {
    return false
  }
  return normalizeGeminiOAuthType(credentials?.oauth_type) === 'vertex_ai'
}

export function resolveGoogleBatchArchiveTargetKind(
  platform: AccountPlatform | string | undefined,
  type: AccountType | string | undefined,
  credentials?: Record<string, unknown> | null,
): GoogleBatchArchiveTargetKind {
  if (isGeminiAIStudioBatchArchiveAccount(platform, type, credentials)) {
    return 'ai_studio'
  }
  if (isGeminiVertexBatchArchiveAccount(platform, type, credentials)) {
    return 'vertex'
  }
  return 'none'
}

export function applyGoogleBatchArchiveExtra(
  base: Record<string, unknown> | undefined,
  target: GoogleBatchArchiveTargetKind,
  state: GoogleBatchArchiveFormState,
): Record<string, unknown> | undefined {
  const nextExtra: Record<string, unknown> = { ...(base || {}) }

  for (const key of archiveExtraKeys) {
    delete nextExtra[key]
  }

  if (target === 'none') {
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  nextExtra.batch_archive_enabled = state.enabled
  nextExtra.batch_archive_retention_days =
    state.retentionDays > 0
      ? state.retentionDays
      : DEFAULT_GOOGLE_BATCH_ARCHIVE_RETENTION_DAYS
  nextExtra.batch_archive_billing_mode =
    normalizeGoogleBatchArchiveBillingMode(state.billingMode)
  nextExtra.batch_archive_download_price_usd =
    state.downloadPriceUSD >= 0
      ? state.downloadPriceUSD
      : DEFAULT_GOOGLE_BATCH_ARCHIVE_DOWNLOAD_PRICE_USD

  if (target === 'ai_studio') {
    nextExtra.batch_archive_auto_prefetch_enabled = state.autoPrefetchEnabled
    nextExtra.allow_vertex_batch_overflow = state.allowVertexBatchOverflow
  }

  if (target === 'vertex') {
    nextExtra.accept_aistudio_batch_overflow =
      state.acceptAIStudioBatchOverflow
  }

  return nextExtra
}
