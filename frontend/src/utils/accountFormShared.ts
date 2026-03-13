import type { AccountPlatform, CheckMixedChannelResponse } from '@/types'

export interface ModelMapping {
  from: string
  to: string
}

export interface TempUnschedRuleForm {
  error_code: number | null
  keywords: string
  duration_minutes: number | null
  description: string
}

export interface TempUnschedRulePayload {
  error_code: number
  keywords: string[]
  duration_minutes: number
  description: string
}

export interface TempUnschedPreset {
  label: string
  rule: TempUnschedRuleForm
}

export interface MixedChannelWarningDetails {
  groupName: string
  currentPlatform: string
  otherPlatform: string
}

export const DEFAULT_POOL_MODE_RETRY_COUNT = 3
export const MAX_POOL_MODE_RETRY_COUNT = 10
export const DEFAULT_TEMP_UNSCHED_DURATION_MINUTES = 30

export function normalizePoolModeRetryCount(value: number): number {
  if (!Number.isFinite(value)) {
    return DEFAULT_POOL_MODE_RETRY_COUNT
  }

  const normalized = Math.trunc(value)
  if (normalized < 0) {
    return 0
  }
  if (normalized > MAX_POOL_MODE_RETRY_COUNT) {
    return MAX_POOL_MODE_RETRY_COUNT
  }
  return normalized
}

export function createEmptyTempUnschedRule(): TempUnschedRuleForm {
  return {
    error_code: null,
    keywords: '',
    duration_minutes: DEFAULT_TEMP_UNSCHED_DURATION_MINUTES,
    description: ''
  }
}

export function splitTempUnschedKeywords(value: string): string[] {
  return value
    .split(/[,;]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

export function buildTempUnschedRules(
  rules: TempUnschedRuleForm[]
): TempUnschedRulePayload[] {
  const out: TempUnschedRulePayload[] = []

  for (const rule of rules) {
    const errorCode = Number(rule.error_code)
    const duration = Number(rule.duration_minutes)
    const keywords = splitTempUnschedKeywords(rule.keywords)
    if (!Number.isFinite(errorCode) || errorCode < 100 || errorCode > 599) {
      continue
    }
    if (!Number.isFinite(duration) || duration <= 0) {
      continue
    }
    if (keywords.length === 0) {
      continue
    }

    out.push({
      error_code: Math.trunc(errorCode),
      keywords,
      duration_minutes: Math.trunc(duration),
      description: rule.description.trim()
    })
  }

  return out
}

export function formatTempUnschedKeywords(value: unknown): string {
  if (!Array.isArray(value)) {
    return ''
  }

  return value
    .map((item) => String(item).trim())
    .filter((item) => item.length > 0)
    .join(', ')
}

export function loadTempUnschedRulesFromCredentials(credentials?: Record<string, unknown>): {
  enabled: boolean
  rules: TempUnschedRuleForm[]
} {
  const rawRules = credentials?.temp_unschedulable_rules
  if (!Array.isArray(rawRules)) {
    return {
      enabled: credentials?.temp_unschedulable_enabled === true,
      rules: []
    }
  }

  return {
    enabled: credentials?.temp_unschedulable_enabled === true,
    rules: rawRules.map((rule) => {
      const entry = rule as Record<string, unknown>
      return {
        error_code: toPositiveNumber(entry.error_code),
        keywords: formatTempUnschedKeywords(entry.keywords),
        duration_minutes: toPositiveNumber(entry.duration_minutes),
        description: typeof entry.description === 'string' ? entry.description : ''
      }
    })
  }
}

export function createTempUnschedPresets(
  t: (key: string) => string
): TempUnschedPreset[] {
  return [
    {
      label: t('admin.accounts.tempUnschedulable.presets.overloadLabel'),
      rule: {
        error_code: 529,
        keywords: 'overloaded, too many',
        duration_minutes: 60,
        description: t('admin.accounts.tempUnschedulable.presets.overloadDesc')
      }
    },
    {
      label: t('admin.accounts.tempUnschedulable.presets.rateLimitLabel'),
      rule: {
        error_code: 429,
        keywords: 'rate limit, too many requests',
        duration_minutes: 10,
        description: t('admin.accounts.tempUnschedulable.presets.rateLimitDesc')
      }
    },
    {
      label: t('admin.accounts.tempUnschedulable.presets.unavailableLabel'),
      rule: {
        error_code: 503,
        keywords: 'unavailable, maintenance',
        duration_minutes: 30,
        description: t('admin.accounts.tempUnschedulable.presets.unavailableDesc')
      }
    }
  ]
}

export function supportsMixedChannelCheck(platform?: AccountPlatform | null): boolean {
  return platform === 'antigravity' || platform === 'anthropic'
}

export function buildMixedChannelWarningDetails(
  response?: CheckMixedChannelResponse
): MixedChannelWarningDetails | null {
  const details = response?.details
  if (!details) {
    return null
  }

  return {
    groupName: details.group_name || 'Unknown',
    currentPlatform: details.current_platform || 'Unknown',
    otherPlatform: details.other_platform || 'Unknown'
  }
}

function toPositiveNumber(value: unknown): number | null {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return null
  }
  return Math.trunc(parsed)
}
