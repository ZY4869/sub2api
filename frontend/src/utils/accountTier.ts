import type {
  AccountPlatform,
  AccountTier,
  ClaudeAccountTier,
  OpenAIAccountTier,
} from '@/types'

export const OPENAI_ACCOUNT_TIERS: OpenAIAccountTier[] = [
  'pro_20x',
  'pro_5x',
  'plus',
  'team',
  'free',
]

export const CLAUDE_ACCOUNT_TIERS: ClaudeAccountTier[] = [
  'max_20x',
  'max_5x',
  'pro',
]

export const DEFAULT_OPENAI_ACCOUNT_TIER: OpenAIAccountTier = 'pro_5x'
export const DEFAULT_CLAUDE_ACCOUNT_TIER: ClaudeAccountTier = 'pro'

const OPENAI_TIER_CAPACITY: Record<OpenAIAccountTier, number> = {
  pro_20x: 10,
  pro_5x: 5,
  plus: 2,
  team: 2,
  free: 1,
}

const CLAUDE_TIER_CAPACITY: Record<ClaudeAccountTier, number> = {
  max_20x: 10,
  max_5x: 5,
  pro: 2,
}

export function isOpenAIAccountTier(value: unknown): value is OpenAIAccountTier {
  return OPENAI_ACCOUNT_TIERS.includes(value as OpenAIAccountTier)
}

export function isClaudeAccountTier(value: unknown): value is ClaudeAccountTier {
  return CLAUDE_ACCOUNT_TIERS.includes(value as ClaudeAccountTier)
}

export function isAccountTierPlatform(platform?: AccountPlatform | string): boolean {
  return platform === 'openai' || platform === 'anthropic'
}

export function resolveAccountTierOptions(platform?: AccountPlatform | string): AccountTier[] {
  if (platform === 'openai') return [...OPENAI_ACCOUNT_TIERS]
  if (platform === 'anthropic') return [...CLAUDE_ACCOUNT_TIERS]
  return []
}

export function normalizeAccountTier(
  platform?: AccountPlatform | string,
  value?: unknown,
): AccountTier | '' {
  if (platform === 'openai' && isOpenAIAccountTier(value)) return value
  if (platform === 'anthropic' && isClaudeAccountTier(value)) return value
  return ''
}

export function defaultAccountTierForPlatform(
  platform?: AccountPlatform | string,
): AccountTier | '' {
  if (platform === 'openai') return DEFAULT_OPENAI_ACCOUNT_TIER
  if (platform === 'anthropic') return DEFAULT_CLAUDE_ACCOUNT_TIER
  return ''
}

export function resolveAccountTierCapacity(
  platform?: AccountPlatform | string,
  tier?: AccountTier | string,
): number {
  const normalized = normalizeAccountTier(platform, tier)
  if (platform === 'openai' && isOpenAIAccountTier(normalized)) {
    return OPENAI_TIER_CAPACITY[normalized]
  }
  if (platform === 'anthropic' && isClaudeAccountTier(normalized)) {
    return CLAUDE_TIER_CAPACITY[normalized]
  }
  return 0
}

export function applyAccountTierToExtra(
  extra: Record<string, unknown> | undefined,
  platform: AccountPlatform | string,
  tier: AccountTier | string,
): Record<string, unknown> | undefined {
  const normalized = normalizeAccountTier(platform, tier)
  const next: Record<string, unknown> = { ...(extra || {}) }
  if (!normalized) {
    delete next.account_tier
    return Object.keys(next).length > 0 ? next : undefined
  }
  next.account_tier = normalized
  if (platform === 'openai' && normalized === 'free' && !('image_compat_allowed' in next)) {
    next.image_protocol_mode = 'native'
    next.image_compat_allowed = false
  }
  return next
}

export function accountTierI18nKey(tier: AccountTier | string): string {
  return `admin.accounts.accountTier.options.${tier}`
}
