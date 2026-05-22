import type { CSSProperties } from 'vue'
import type { Account, AccountRateLimitReason } from '@/types'

export type AccountVisualTone =
  | 'available'
  | 'usage_7d_0_25'
  | 'usage_7d_25_50'
  | 'usage_7d_50_75'
  | 'usage_7d_75_100'
  | 'usage_5h_0_25'
  | 'usage_5h_25_50'
  | 'usage_5h_50_75'
  | 'usage_5h_75_100'
  | 'error'
  | 'paused'

export type AccountRowVisualState = {
  tone: AccountVisualTone
  className: string
  style: CSSProperties
}

type VisualDefinition = {
  className: string
  style: CSSProperties
}

const SHARED_ROW_CLASS =
  'account-visual-row transition-colors duration-200 ease-out'

const buildStaticRowStyle = (style: CSSProperties): CSSProperties => ({
  ...style,
  backgroundImage: 'none',
  backgroundSize: 'auto',
  animation: 'none'
})

const VISUALS: Record<AccountVisualTone, VisualDefinition> = {
  available: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-available`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#F6FEF9',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#F6FEF9',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(167, 243, 208, 0.22)',
      '--account-row-text': '#047857',
      backgroundColor: '#F7FCF9'
    } as CSSProperties)
  },
  usage_7d_0_25: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-7d-0-25`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#FCFDFF',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#FCFDFF',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(165, 180, 252, 0.18)',
      '--account-row-text': '#3730A3',
      backgroundColor: '#FCFDFF'
    } as CSSProperties)
  },
  usage_7d_25_50: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-7d-25-50`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#FAFCFF',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#FAFCFF',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(147, 197, 253, 0.22)',
      '--account-row-text': '#1D4ED8',
      backgroundColor: '#FAFCFF'
    } as CSSProperties)
  },
  usage_7d_50_75: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-7d-50-75`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#F6FAFF',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#F6FAFF',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(125, 211, 252, 0.22)',
      '--account-row-text': '#0369A1',
      backgroundColor: '#F6FAFF'
    } as CSSProperties)
  },
  usage_7d_75_100: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-7d-75-100`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#F2FAFA',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#F2FAFA',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(45, 212, 191, 0.2)',
      '--account-row-text': '#0F766E',
      backgroundColor: '#F2FAFA'
    } as CSSProperties)
  },
  usage_5h_0_25: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-5h-0-25`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#FFF9F2',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#FFF9F2',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(251, 146, 60, 0.18)',
      '--account-row-text': '#C2410C',
      backgroundColor: '#FFF9F2'
    } as CSSProperties)
  },
  usage_5h_25_50: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-5h-25-50`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#FEFCF4',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#FEFCF4',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(251, 191, 36, 0.18)',
      '--account-row-text': '#B45309',
      backgroundColor: '#FEFCF4'
    } as CSSProperties)
  },
  usage_5h_50_75: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-5h-50-75`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#F4FAFF',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#F4FAFF',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(56, 189, 248, 0.18)',
      '--account-row-text': '#0369A1',
      backgroundColor: '#F4FAFF'
    } as CSSProperties)
  },
  usage_5h_75_100: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-5h-75-100`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#F0FDF4',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#F0FDF4',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(74, 222, 128, 0.18)',
      '--account-row-text': '#15803D',
      backgroundColor: '#F0FDF4'
    } as CSSProperties)
  },
  error: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-error`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#FFECEC',
      '--account-row-bg-hover': '#FFF5F5',
      '--account-row-sticky-bg': '#FFECEC',
      '--account-row-sticky-bg-hover': '#FFF5F5',
      '--account-row-border': 'rgba(220, 38, 38, 0.18)',
      '--account-row-text': '#9F1239',
      backgroundColor: '#FFECEC'
    } as CSSProperties)
  },
  paused: {
    className: `${SHARED_ROW_CLASS} account-visual-tone-paused`,
    style: buildStaticRowStyle({
      '--account-row-bg': '#F4F5F7',
      '--account-row-bg-hover': '#FFFFFF',
      '--account-row-sticky-bg': '#F4F5F7',
      '--account-row-sticky-bg-hover': '#FFFFFF',
      '--account-row-border': 'rgba(71, 85, 105, 0.16)',
      '--account-row-text': '#1E293B',
      backgroundColor: '#F4F5F7'
    } as CSSProperties)
  }
}

const resolveDefinition = (tone: AccountVisualTone): AccountRowVisualState => {
  const visual = VISUALS[tone]
  return {
    tone,
    className: visual.className,
    style: visual.style
  }
}

const getRateLimitDurationMs = (account: Account): number | null => {
  if (!account.rate_limited_at || !account.rate_limit_reset_at) return null
  const startedAt = new Date(account.rate_limited_at).getTime()
  const resetAt = new Date(account.rate_limit_reset_at).getTime()
  if (!Number.isFinite(startedAt) || !Number.isFinite(resetAt) || resetAt <= startedAt) {
    return null
  }
  return resetAt - startedAt
}

const getRateLimitProgress = (account: Account, nowMs: number): number | null => {
  if (!account.rate_limit_reset_at) return null
  const resetAt = new Date(account.rate_limit_reset_at).getTime()
  if (!Number.isFinite(resetAt)) return null

  const durationMs = getRateLimitDurationMs(account)
  if (!durationMs) return null

  const remainingMs = resetAt - nowMs
  if (remainingMs <= 0) return 1
  const elapsedRatio = 1 - remainingMs / durationMs
  if (!Number.isFinite(elapsedRatio)) return null
  return Math.min(1, Math.max(0, elapsedRatio))
}

const resolveToneByProgress = (
  reason: AccountRateLimitReason,
  progress: number | null
): AccountVisualTone => {
  const safeProgress = progress ?? 0
  const bucket =
    safeProgress >= 0.75
      ? '75_100'
      : safeProgress >= 0.5
        ? '50_75'
        : safeProgress >= 0.25
          ? '25_50'
          : '0_25'

  if (reason === 'usage_5h' || reason === 'rate_429') {
    return `usage_5h_${bucket}` as AccountVisualTone
  }
  return `usage_7d_${bucket}` as AccountVisualTone
}

export const resolveAccountRowVisualState = (
  account: Account,
  nowMs: number = Date.now(),
): AccountRowVisualState => {
  if (account.rate_limit_reset_at) {
    const resetAt = new Date(account.rate_limit_reset_at).getTime()
    if (Number.isFinite(resetAt) && resetAt > nowMs) {
      return resolveDefinition(
        resolveToneByProgress(account.rate_limit_reason || 'rate_429', getRateLimitProgress(account, nowMs))
      )
    }
  }

  if (account.overload_until) {
    const overloadUntil = new Date(account.overload_until).getTime()
    if (Number.isFinite(overloadUntil) && overloadUntil > nowMs) {
      return resolveDefinition(resolveToneByProgress('usage_5h', 0.1))
    }
  }

  if (account.temp_unschedulable_until) {
    const tempUnschedUntil = new Date(account.temp_unschedulable_until).getTime()
    if (Number.isFinite(tempUnschedUntil) && tempUnschedUntil > nowMs) {
      return resolveDefinition(resolveToneByProgress('usage_5h', 0.45))
    }
  }

  if (account.status === 'error' || account.lifecycle_state === 'blacklisted') {
    return resolveDefinition('error')
  }

  if (!account.schedulable || account.status === 'inactive' || account.lifecycle_state === 'archived') {
    return resolveDefinition('paused')
  }

  return resolveDefinition('available')
}
