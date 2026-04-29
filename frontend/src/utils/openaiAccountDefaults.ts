import type { OpenAIImageProtocolMode } from '@/types'

const OPENAI_IMAGE_COMPAT_ALLOWED_PLANS = new Set(['plus', 'team', 'pro', 'business', 'enterprise', 'edu'])

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
