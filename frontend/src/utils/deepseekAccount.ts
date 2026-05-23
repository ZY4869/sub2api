import type { AccountPlatform } from '@/types'

export const DEEPSEEK_MODEL_CONCURRENCY_LIMITS_KEY = 'deepseek_model_concurrency_limits'

export const DEEPSEEK_V4_PRO_MODEL_ID = 'deepseek-v4-pro'
export const DEEPSEEK_V4_FLASH_MODEL_ID = 'deepseek-v4-flash'

export const DEFAULT_DEEPSEEK_MODEL_CONCURRENCY_LIMITS: Record<string, number> = {
  [DEEPSEEK_V4_PRO_MODEL_ID]: 500,
  [DEEPSEEK_V4_FLASH_MODEL_ID]: 2500
}

export type DeepSeekModelConcurrencyLimitDraft = Record<string, number | ''>

export function createDefaultDeepSeekModelConcurrencyLimitDraft(): DeepSeekModelConcurrencyLimitDraft {
  return {
    ...DEFAULT_DEEPSEEK_MODEL_CONCURRENCY_LIMITS
  }
}

export function normalizeDeepSeekModelID(value: unknown): string {
  let normalized = String(value || '').trim().toLowerCase()
  if (!normalized) {
    return ''
  }
  normalized = normalized.replace(/^models\//, '')
  const slashIndex = normalized.lastIndexOf('/')
  if (slashIndex >= 0) {
    normalized = normalized.slice(slashIndex + 1)
  }
  normalized = normalized.replace(/[:\s_]+/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '')
  if (normalized.endsWith('-free')) {
    normalized = normalized.slice(0, -'-free'.length)
  }
  if (normalized === DEEPSEEK_V4_PRO_MODEL_ID || normalized === DEEPSEEK_V4_FLASH_MODEL_ID) {
    return normalized
  }
  return ''
}

export function normalizeDeepSeekModelConcurrencyLimits(raw: unknown): Record<string, number> {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return {}
  }
  const result: Record<string, number> = {}
  for (const [rawModel, rawLimit] of Object.entries(raw as Record<string, unknown>)) {
    const model = normalizeDeepSeekModelID(rawModel)
    const limit = Number(rawLimit)
    if (!model || !Number.isFinite(limit) || limit <= 0) {
      continue
    }
    result[model] = Math.floor(limit)
  }
  return result
}

export function readDeepSeekModelConcurrencyLimitDraft(extra?: Record<string, unknown> | null): DeepSeekModelConcurrencyLimitDraft {
  return {
    ...createDefaultDeepSeekModelConcurrencyLimitDraft(),
    ...normalizeDeepSeekModelConcurrencyLimits(extra?.[DEEPSEEK_MODEL_CONCURRENCY_LIMITS_KEY])
  }
}

export function applyDeepSeekModelConcurrencyLimitsExtra(
  baseExtra: Record<string, unknown> | undefined,
  platform: AccountPlatform | string,
  limits: DeepSeekModelConcurrencyLimitDraft
): Record<string, unknown> | undefined {
  const nextExtra = { ...(baseExtra || {}) }
  if (platform !== 'deepseek') {
    delete nextExtra[DEEPSEEK_MODEL_CONCURRENCY_LIMITS_KEY]
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  const normalized = normalizeDeepSeekModelConcurrencyLimits(limits)
  if (Object.keys(normalized).length === 0) {
    delete nextExtra[DEEPSEEK_MODEL_CONCURRENCY_LIMITS_KEY]
  } else {
    nextExtra[DEEPSEEK_MODEL_CONCURRENCY_LIMITS_KEY] = normalized
  }
  return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
}
