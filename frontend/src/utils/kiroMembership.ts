export type KiroMemberLevel =
  | 'kiro_free'
  | 'kiro_pro'
  | 'kiro_pro_plus'
  | 'kiro_power'

export const KIRO_MEMBER_LEVEL_DEFAULT_CREDITS: Record<KiroMemberLevel, number> = {
  kiro_free: 50,
  kiro_pro: 1000,
  kiro_pro_plus: 2000,
  kiro_power: 10000
}

export function normalizeKiroMemberLevel(
  value: unknown,
  fallback: KiroMemberLevel = 'kiro_free'
): KiroMemberLevel {
  const normalized = typeof value === 'string' ? value.trim().toLowerCase() : ''
  switch (normalized) {
    case 'kiro_pro':
    case 'kiro_pro_plus':
    case 'kiro_power':
      return normalized
    case 'kiro_free':
      return 'kiro_free'
    default:
      return fallback
  }
}

export function defaultKiroMemberCredits(level: KiroMemberLevel): number {
  return KIRO_MEMBER_LEVEL_DEFAULT_CREDITS[level]
}

export function parseKiroMemberCredits(value: unknown): number | null {
  if (typeof value === 'number' && Number.isInteger(value) && value >= 0) {
    return value
  }
  if (typeof value !== 'string') {
    return null
  }
  const trimmed = value.trim()
  if (!/^\d+$/.test(trimmed)) {
    return null
  }
  const parsed = Number.parseInt(trimmed, 10)
  return Number.isFinite(parsed) ? parsed : null
}

export function buildKiroMembershipExtra(
  level: KiroMemberLevel,
  credits: number,
  base?: Record<string, unknown>
): Record<string, unknown> {
  return {
    ...(base || {}),
    kiro_member_level: normalizeKiroMemberLevel(level),
    kiro_member_credits: credits
  }
}

export function readKiroMembershipFromExtra(extra?: Record<string, unknown> | null): {
  level: KiroMemberLevel
  credits: number
} {
  const level = normalizeKiroMemberLevel(extra?.kiro_member_level)
  const parsedCredits = parseKiroMemberCredits(extra?.kiro_member_credits)
  return {
    level,
    credits: parsedCredits ?? defaultKiroMemberCredits(level)
  }
}
