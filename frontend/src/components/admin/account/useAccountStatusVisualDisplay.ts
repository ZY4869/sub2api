import { computed, type ComputedRef, type Ref } from 'vue'
import type { Account } from '@/types'
import { resolveCodexUsageWindow } from '@/utils/codexUsage'
import type { AiryStatusKind, AiryStatusVisual } from './accountAiryStatusTypes'
import {
  resolveAccountGlassToneStyles,
  type AccountGlassTone,
} from './accountVisualGlass'

type Translate = (key: string, params?: Record<string, unknown>) => string

const DETAIL_STATUS_KINDS = new Set<AiryStatusKind>([
  'banned',
  'locked',
  'maintenance',
  'offline',
  'overdue',
  'degraded',
  'captcha',
  'syncing',
  'error',
  'tempUnschedulable',
])

const firstText = (...values: Array<unknown>) => {
  for (const value of values) {
    const text = String(value || '').trim()
    if (text) return text
  }
  return ''
}

export function useAccountStatusVisualDisplay(options: {
  account: Ref<Account>
  airyStatus: ComputedRef<AiryStatusVisual>
  visualTone: ComputedRef<AccountGlassTone>
  nowDate: ComputedRef<Date>
  isRateLimited: ComputedRef<boolean>
  isOverloaded: ComputedRef<boolean>
  rateLimitResumeText: ComputedRef<string>
  overloadCountdown: ComputedRef<string | null>
  t: Translate
}) {
  const toneStyles = computed(() =>
    resolveAccountGlassToneStyles(options.visualTone.value)
  )
  const whiteSurfaceClass = computed(() => {
    switch (options.visualTone.value) {
      case 'red':
        return 'border-rose-200/80 bg-white dark:border-rose-400/20 dark:bg-slate-900'
      case 'orange':
      case 'amber':
        return 'border-amber-200/80 bg-white dark:border-amber-400/20 dark:bg-slate-900'
      case 'indigo':
      case 'sky':
      case 'purple':
      case 'teal':
        return 'border-sky-200/80 bg-white dark:border-sky-400/20 dark:bg-slate-900'
      case 'emerald':
        return 'border-emerald-200/80 bg-white dark:border-emerald-400/20 dark:bg-slate-900'
      default:
        return 'border-slate-200/85 bg-white dark:border-slate-700/80 dark:bg-slate-900'
    }
  })
  const statusTitle = computed(() => {
    const title = options.t(options.airyStatus.value.titleKey)
    if (options.airyStatus.value.kind !== 'usage7d') return title
    const label = resolveCodexUsageWindow(
      options.account.value.extra,
      '7d',
      options.nowDate.value,
    ).label
    return title.replace(/7D/gi, label)
  })
  const helperDetailText = computed(() => {
    if (!DETAIL_STATUS_KINDS.has(options.airyStatus.value.kind)) return ''
    if (options.airyStatus.value.helperKey) {
      return options.t(options.airyStatus.value.helperKey)
    }
    return firstText(options.airyStatus.value.helper)
  })
  const statusDetailText = computed(() => {
    if (helperDetailText.value) return helperDetailText.value
    if (options.isRateLimited.value) return options.rateLimitResumeText.value
    if (options.isOverloaded.value) return options.overloadCountdown.value || ''
    return ''
  })
  const countdownResetAt = computed(() => {
    if (options.isRateLimited.value) {
      return options.account.value.rate_limit_reset_at || null
    }
    if (options.isOverloaded.value) {
      return options.account.value.overload_until || null
    }
    return null
  })
  const countdownPrefix = computed(() => {
    if (options.airyStatus.value.kind === 'usage7d') {
      return resolveCodexUsageWindow(
        options.account.value.extra,
        '7d',
        options.nowDate.value,
      ).label
    }
    if (options.airyStatus.value.kind === 'usage5h') {
      if (options.account.value.rate_limit_reason !== 'usage_5h') return ''
      return resolveCodexUsageWindow(
        options.account.value.extra,
        '5h',
        options.nowDate.value,
      ).label
    }
    return ''
  })
  return {
    toneStyles,
    whiteSurfaceClass,
    statusTitle,
    statusDetailText,
    countdownResetAt,
    countdownPrefix,
  }
}
