import type { AccountPlatform, AccountType } from '@/types'

export type AccountKeyTierDisplay = {
  primaryLabel: string
  tierLabel: string
  title: string
  className: string
}

type KeyTierInput = {
  platform: AccountPlatform
  type: AccountType
  credentials?: Record<string, unknown> | null
  extra?: Record<string, unknown> | null
}

const asText = (value: unknown): string => (typeof value === 'string' ? value.trim() : '')

const normalize = (value: unknown): string => asText(value).toLowerCase().replace(/[\s-]+/g, '_')

const resolveProMultiplier = (value: unknown): number | null => {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) return value
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed) && parsed > 0) return parsed
  }
  return null
}

const formatProLabel = (multiplier: number | null) => {
  if (!multiplier) return 'Pro'
  return `Pro${Number.isInteger(multiplier) ? multiplier : Number(multiplier.toFixed(2))}x`
}

export function resolveAccountKeyTierLabel(input: KeyTierInput): string {
  const credentials = input.credentials ?? {}
  const extra = input.extra ?? {}
  const explicitLabel = asText(credentials.plan_type_label)
  const planType = normalize(credentials.plan_type)
  const tierId = normalize(credentials.tier_id)
  const accountTier = normalize(extra.account_tier)
  const combined = [accountTier, tierId, planType].filter(Boolean).join(' ')

  if (explicitLabel && explicitLabel.toLowerCase() !== 'pro') return explicitLabel
  if (combined.includes('pro_20x') || combined.includes('max_20x')) return 'Pro20x'
  if (combined.includes('pro_5x') || combined.includes('max_5x')) return 'Pro5x'
  if (planType === 'pro' || planType === 'chatgptpro' || explicitLabel.toLowerCase() === 'pro') {
    return formatProLabel(resolveProMultiplier(credentials.pro_multiplier))
  }
  if (combined.includes('google_ai_ultra') || combined.includes('ultra')) return 'Ultra'
  if (combined.includes('google_ai_pro')) return 'Gemini Pro'
  if (combined.includes('aistudio_paid')) return 'AI Studio Paid'
  if (combined.includes('aistudio_tier_3')) return 'AI Studio T3'
  if (combined.includes('aistudio_tier_2')) return 'AI Studio T2'
  if (combined.includes('aistudio_tier_1')) return 'AI Studio T1'
  if (combined.includes('gcp_enterprise')) return 'GCP Enterprise'
  if (combined.includes('gcp_standard')) return 'GCP Standard'
  if (combined.includes('team')) return 'Team'
  if (combined.includes('plus')) return 'Plus'
  if (combined.includes('free') || combined.includes('legacy')) return 'Free'
  if (asText(credentials.plan_type)) return asText(credentials.plan_type)
  if (asText(extra.account_tier)) return asText(extra.account_tier)
  if (asText(credentials.tier_id)) return asText(credentials.tier_id)
  return ''
}

export function resolveAccountKeyTierClass(tierLabel: string): string {
  const normalized = tierLabel.toLowerCase()
  if (!tierLabel) {
    return 'border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200'
  }
  if (normalized.includes('free') || normalized.includes('legacy')) {
    return 'border-slate-300/80 bg-slate-100 text-slate-600 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200'
  }
  if (normalized.includes('plus')) {
    return 'border-emerald-300/80 bg-emerald-50 text-emerald-700 dark:border-emerald-400/25 dark:bg-emerald-400/10 dark:text-emerald-100'
  }
  if (normalized.includes('team')) {
    return 'border-blue-300/80 bg-blue-50 text-blue-700 dark:border-blue-400/25 dark:bg-blue-400/10 dark:text-blue-100'
  }
  if (normalized.includes('20x') || normalized.includes('ultra')) {
    return 'border-slate-700 bg-slate-800 text-amber-400 ring-1 ring-slate-900 dark:border-amber-400/30 dark:bg-slate-950 dark:text-amber-200'
  }
  if (normalized.includes('5x') || normalized === 'pro') {
    return 'border-cyan-200 bg-cyan-50 text-cyan-700 dark:border-cyan-400/25 dark:bg-cyan-400/10 dark:text-cyan-100'
  }
  if (normalized.includes('gemini') || normalized.includes('studio') || normalized.includes('gcp')) {
    return 'border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-400/25 dark:bg-violet-400/10 dark:text-violet-100'
  }
  return 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-400/25 dark:bg-sky-400/10 dark:text-sky-100'
}

export function resolveAccountKeyTierDisplay(input: KeyTierInput): AccountKeyTierDisplay {
  const tierLabel = resolveAccountKeyTierLabel(input)
  const titleParts = ['Key']
  if (tierLabel) titleParts.push(tierLabel)
  titleParts.push(input.platform)
  return {
    primaryLabel: 'Key',
    tierLabel,
    title: titleParts.join(' / '),
    className: resolveAccountKeyTierClass(tierLabel),
  }
}
