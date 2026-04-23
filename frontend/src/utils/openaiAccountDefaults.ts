import type { OpenAIImageProtocolMode } from '@/types'

const OPENAI_BASE_DEFAULT_WHITELIST = ['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini'] as const
const OPENAI_PRO_DEFAULT_WHITELIST = [...OPENAI_BASE_DEFAULT_WHITELIST, 'gpt-5.3-codex-spark'] as const
const OPENAI_IMAGE_COMPAT_ALLOWED_PLANS = new Set(['plus', 'team', 'pro', 'business', 'enterprise', 'edu'])

function normalizeOpenAIWhitelist(models: readonly string[] | null | undefined): string[] {
  if (!Array.isArray(models)) {
    return []
  }

  const seen = new Set<string>()
  const normalized: string[] = []
  for (const model of models) {
    const trimmed = String(model || '').trim()
    if (!trimmed || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    normalized.push(trimmed)
  }
  return normalized
}

function normalizeOpenAIWhitelistForComparison(models: readonly string[] | null | undefined): string[] {
  return [...normalizeOpenAIWhitelist(models)].sort()
}

function isSameOpenAIWhitelist(
  left: readonly string[] | null | undefined,
  right: readonly string[] | null | undefined,
): boolean {
  const leftNormalized = normalizeOpenAIWhitelistForComparison(left)
  const rightNormalized = normalizeOpenAIWhitelistForComparison(right)
  if (leftNormalized.length !== rightNormalized.length) {
    return false
  }
  return leftNormalized.every((value, index) => value === rightNormalized[index])
}

export function normalizeOpenAIPlanType(raw: string | null | undefined): string {
  const trimmed = String(raw || '').trim()
  if (!trimmed) {
    return ''
  }

  const normalized = trimmed.toLowerCase().replace(/[-_\s]/g, '')
  switch (normalized) {
    case 'chatgptplus':
    case 'plus':
      return 'plus'
    case 'chatgptteam':
    case 'team':
      return 'team'
    case 'chatgptpro':
    case 'pro':
      return 'pro'
    case 'chatgptfree':
    case 'free':
      return 'free'
    default:
      return trimmed
  }
}

export function getOpenAIDefaultWhitelist(planType?: string | null): string[] {
  const normalizedPlanType = normalizeOpenAIPlanType(planType)
  return normalizedPlanType === 'pro'
    ? [...OPENAI_PRO_DEFAULT_WHITELIST]
    : [...OPENAI_BASE_DEFAULT_WHITELIST]
}

export function shouldAutoReplaceOpenAIWhitelist(currentModels: string[] | null | undefined): boolean {
  const current = normalizeOpenAIWhitelist(currentModels)
  return (
    current.length === 0 ||
    isSameOpenAIWhitelist(current, OPENAI_BASE_DEFAULT_WHITELIST) ||
    isSameOpenAIWhitelist(current, OPENAI_PRO_DEFAULT_WHITELIST)
  )
}

export function normalizeOpenAIImageProtocolMode(
  raw: string | null | undefined
): OpenAIImageProtocolMode | '' {
  const normalized = String(raw || '').trim().toLowerCase()
  if (normalized === 'compat') {
    return 'compat'
  }
  if (normalized === 'native') {
    return 'native'
  }
  return ''
}

export function isOpenAIImageCompatAllowedPlan(planType?: string | null): boolean {
  const normalizedPlanType = normalizeOpenAIPlanType(planType)
  if (normalizedPlanType === 'free') {
    return false
  }
  if (!normalizedPlanType) {
    return true
  }
  if (OPENAI_IMAGE_COMPAT_ALLOWED_PLANS.has(normalizedPlanType)) {
    return true
  }
  return true
}

export function getDefaultOpenAIImageProtocolMode(planType?: string | null): OpenAIImageProtocolMode {
  return isOpenAIImageCompatAllowedPlan(planType) ? 'compat' : 'native'
}

export function resolveOpenAIImageProtocolState(options: {
  accountCategory: 'oauth-based' | 'apikey' | 'vertex_ai'
  planType?: string | null
  storedMode?: string | null
  storedCompatAllowed?: unknown
}): {
  compatAllowed: boolean
  mode: OpenAIImageProtocolMode
} {
  const compatAllowed = typeof options.storedCompatAllowed === 'boolean'
    ? options.storedCompatAllowed
    : options.accountCategory !== 'oauth-based'
      ? true
      : isOpenAIImageCompatAllowedPlan(options.planType)

  const defaultMode = options.accountCategory === 'oauth-based'
    ? getDefaultOpenAIImageProtocolMode(options.planType)
    : 'native'
  const normalizedStoredMode = normalizeOpenAIImageProtocolMode(options.storedMode)
  const mode = normalizedStoredMode || defaultMode

  return {
    compatAllowed,
    mode: compatAllowed || mode !== 'compat' ? mode : 'native'
  }
}
