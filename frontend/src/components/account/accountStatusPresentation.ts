import { computed, nextTick, onUnmounted, ref, type ComputedRef, type Ref } from 'vue'
import type { Account } from '@/types'
import { formatDateTime, formatTime } from '@/utils/format'
import { resolveCodexUsageWindow, type CodexUsageScope } from '@/utils/codexUsage'

export type AccountModelStatusItem = {
  kind: 'rate_limit' | 'credits_exhausted' | 'credits_active'
  model: string
  reset_at: string
}

export type AccountStatusLimitBadgeItem = {
  key: string
  tone: 'purple' | 'amber' | 'red'
  label: string
  countdown?: string
  tooltip: string
  model?: string
  modelDisplayName?: string
}

type CodexUsageWindowKind = '5h' | '7d'

type CodexScopeLimitInfo = {
  scope: CodexUsageScope
  window: CodexUsageWindowKind
  resetAt: string
  label: string
  model: string
}

const codexScopeModels: Record<CodexUsageScope, string> = {
  normal: 'gpt-5.3-codex',
  spark: 'gpt-5.3-codex-spark'
}

const codexWindowPriority: CodexUsageWindowKind[] = ['7d', '5h']

export const createAccountStatusPresentation = (
  account: Ref<Account>,
  t: (key: string, params?: Record<string, unknown>) => string,
  nowMs: Ref<number>,
  nowDate: ComputedRef<Date>,
) => {
  const formatCountdownFromNow = (targetDate: string | Date | null | undefined): string | null => {
    if (!targetDate) return null

    const target = new Date(targetDate)
    const diffMs = target.getTime() - nowMs.value
    if (diffMs <= 0 || Number.isNaN(diffMs)) return null

    const diffMins = Math.floor(diffMs / (1000 * 60))
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)
    const remainingHours = diffHours % 24
    const remainingMins = diffMins % 60

    if (diffDays > 0) {
      return t('common.time.countdown.daysHours', {
        d: diffDays,
        h: remainingHours,
      })
    }
    if (diffHours > 0) {
      return t('common.time.countdown.hoursMinutes', {
        h: diffHours,
        m: remainingMins,
      })
    }
    return t('common.time.countdown.minutes', {
      m: diffMins,
    })
  }

  const formatCountdownWithSuffixFromNow = (
    targetDate: string | Date | null | undefined,
  ): string | null => {
    const countdown = formatCountdownFromNow(targetDate)
    if (!countdown) return null
    return t('common.time.countdown.withSuffix', { time: countdown })
  }

  const isRateLimited = computed(() => {
    if (!account.value.rate_limit_reset_at) return false
    const resetAtMs = new Date(account.value.rate_limit_reset_at).getTime()
    return !Number.isNaN(resetAtMs) && resetAtMs > nowMs.value
  })

  const activeModelStatuses = computed<AccountModelStatusItem[]>(() => {
    const extra = account.value.extra as Record<string, unknown> | undefined
    const modelLimits = extra?.model_rate_limits as
      | Record<string, { rate_limited_at: string; rate_limit_reset_at: string }>
      | undefined
    const items: AccountModelStatusItem[] = []

    if (!modelLimits) return items

    const now = nowDate.value
    const aiCreditsEntry = modelLimits.AICredits
    const hasActiveAICredits = !!aiCreditsEntry && new Date(aiCreditsEntry.rate_limit_reset_at) > now
    const allowOverages = !!extra?.allow_overages

    for (const [model, info] of Object.entries(modelLimits)) {
      if (new Date(info.rate_limit_reset_at) <= now) continue

      if (model === 'AICredits') {
        items.push({ kind: 'credits_exhausted', model, reset_at: info.rate_limit_reset_at })
      } else if (allowOverages && !hasActiveAICredits) {
        items.push({ kind: 'credits_active', model, reset_at: info.rate_limit_reset_at })
      } else {
        items.push({ kind: 'rate_limit', model, reset_at: info.rate_limit_reset_at })
      }
    }

    return items
  })

  const formatScopeName = (scope: string): string => {
    const aliases: Record<string, string> = {
      'gpt-5.3-codex': 'Codex',
      'gpt-5.3-codex-spark': 'Spark',
      'claude-opus-4.1': 'COpus41',
      'claude-opus-4-1': 'COpus41',
      'claude-opus-4-1-20250805': 'COpus41',
      'claude-opus-4-6': 'COpus46',
      'claude-opus-4-6-thinking': 'COpus46T',
      'claude-opus-4-5-thinking': 'COpus45T',
      'claude-sonnet-4.5': 'CSon45',
      'claude-sonnet-4-5': 'CSon45',
      'claude-sonnet-4-5-20250929': 'CSon45',
      'claude-sonnet-4-5-thinking': 'CSon45T',
      'claude-sonnet-4-6': 'CSon46',
      'claude-haiku-4.5': 'CHai45',
      'claude-haiku-4-5': 'CHai45',
      'claude-haiku-4-5-20251001': 'CHai45',
      'gemini-2.5-flash': 'G25F',
      'gemini-2.5-flash-lite': 'G25FL',
      'gemini-2.5-flash-thinking': 'G25FT',
      'gemini-2.5-pro': 'G25P',
      'gemini-2.5-flash-image': 'G25I',
      'gemini-3-flash': 'G3F',
      'gemini-3.1-pro-high': 'G3PH',
      'gemini-3.1-pro-low': 'G3PL',
      'gemini-3-pro-image': 'G3PI',
      'gemini-3.1-flash-image': 'G31FI',
      'gpt-oss-120b-medium': 'GPT120',
      tab_flash_lite_preview: 'TabFL',
      claude: 'Claude',
      claude_sonnet: 'CSon',
      claude_opus: 'COpus',
      claude_haiku: 'CHaiku',
      gemini_text: 'Gemini',
      gemini_image: 'GImg',
      gemini_flash: 'GFlash',
      gemini_pro: 'GPro'
    }
    return aliases[scope] || scope
  }

  const codexScopeForModel = (model: string): CodexUsageScope | null => {
    const normalized = model.trim().toLowerCase()
    if (normalized.startsWith('gpt-5.3-codex-spark')) return 'spark'
    if (normalized.startsWith('gpt-5.3-codex')) return 'normal'
    return null
  }

  const codexScopeName = (scope: CodexUsageScope): string => {
    return scope === 'spark' ? 'Spark' : 'Codex'
  }

  const codexScopeWindowLabel = (scope: CodexUsageScope, window: CodexUsageWindowKind): string => {
    return `${codexScopeName(scope)} ${window}`
  }

  const normalizeResetAt = (value: unknown): string | null => {
    if (typeof value !== 'string' || value.trim() === '') return null
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return null
    return date.toISOString()
  }

  const resolvePreferredResetAt = (...values: Array<unknown>): string | null => {
    const normalized = values
      .map((value) => normalizeResetAt(value))
      .filter((value): value is string => value !== null)

    if (normalized.length === 0) return null

    return normalized.find((value) => new Date(value).getTime() > nowMs.value) ?? normalized[0]
  }

  const resolveActiveCodexScopeLimit = (scope: CodexUsageScope): CodexScopeLimitInfo | null => {
    const extra = account.value.extra
    if (!extra) return null

    for (const window of codexWindowPriority) {
      const progress = resolveCodexUsageWindow(extra, window, nowDate.value, scope)
      if (progress.usedPercent == null || progress.usedPercent < 100) continue
      const resetAt = resolvePreferredResetAt(progress.resetAt)
      if (!resetAt || new Date(resetAt).getTime() <= nowMs.value) continue

      return {
        scope,
        window,
        resetAt,
        label: codexScopeWindowLabel(scope, window),
        model: codexScopeModels[scope],
      }
    }

    return null
  }

  const formatBadgeCountdown = (resetAt: string | null | undefined): string => {
    if (!resetAt) return ''
    const date = new Date(resetAt)
    const diffMs = date.getTime() - nowMs.value
    if (diffMs <= 0) return ''
    const totalSecs = Math.floor(diffMs / 1000)
    const totalHours = Math.floor(totalSecs / 3600)
    const minutes = Math.floor((totalSecs % 3600) / 60)
    const seconds = totalSecs % 60
    if (totalHours > 0) return `${totalHours}h ${minutes}m`
    if (minutes > 0) return `${minutes}m ${seconds}s`
    return `${seconds}s`
  }

  const buildCodexScopeLimitBadge = (scope: CodexUsageScope): AccountStatusLimitBadgeItem | null => {
    const info = resolveActiveCodexScopeLimit(scope)
    if (!info) return null

    return {
      key: `account-${info.scope}-${info.window}`,
      tone: 'amber',
      label: info.label,
      countdown: formatBadgeCountdown(info.resetAt),
      tooltip: t('admin.accounts.status.modelRateLimitedUntil', {
        model: info.label,
        time: formatTime(info.resetAt),
      }),
      model: info.model,
      modelDisplayName: info.label,
    }
  }

  const resolveModelLimitBadgeDisplay = (model: string, fallbackResetAt: string) => {
    const scope = codexScopeForModel(model)
    if (scope) {
      const info = resolveActiveCodexScopeLimit(scope)
      if (info) {
        return {
          label: info.label,
          resetAt: info.resetAt,
          model: info.model,
          modelDisplayName: info.label,
        }
      }
    }

    const modelDisplayName = formatScopeName(model)
    return {
      label: modelDisplayName,
      resetAt: fallbackResetAt,
      model,
      modelDisplayName,
    }
  }

  const buildLimitBadgeIdentity = (
    item: Pick<AccountStatusLimitBadgeItem, 'label' | 'model' | 'tooltip'>,
  ): string => {
    const model = String(item.model || '').trim().toLowerCase()
    const label = String(item.label || '').trim().toLowerCase()
    const tooltip = String(item.tooltip || '').trim().toLowerCase()
    return `${model}::${label}::${tooltip}`
  }

  const rawActiveModelBadges = computed<AccountStatusLimitBadgeItem[]>(() => {
    return activeModelStatuses.value.map((item) => {
      if (item.kind === 'credits_exhausted') {
        return {
          key: `credits-${item.model}`,
          tone: 'red',
          label: t('admin.accounts.status.creditsExhausted'),
          countdown: formatBadgeCountdown(item.reset_at),
          tooltip: t('admin.accounts.status.creditsExhaustedUntil', { time: formatTime(item.reset_at) }),
        }
      }

      const display = resolveModelLimitBadgeDisplay(item.model, item.reset_at)
      return {
        key: `${item.kind}-${item.model}`,
        tone: item.kind === 'credits_active' ? 'amber' : 'purple',
        label: display.label,
        countdown: formatBadgeCountdown(display.resetAt),
        tooltip: item.kind === 'credits_active'
          ? t('admin.accounts.status.modelCreditOveragesUntil', { model: display.label, time: formatTime(display.resetAt) })
          : t('admin.accounts.status.modelRateLimitedUntil', { model: display.label, time: formatTime(display.resetAt) }),
        model: display.model,
        modelDisplayName: display.modelDisplayName,
      }
    })
  })

  const isOverloaded = computed(() => {
    if (!account.value.overload_until) return false
    const untilMs = new Date(account.value.overload_until).getTime()
    return !Number.isNaN(untilMs) && untilMs > nowMs.value
  })

  const isTempUnschedulable = computed(() => {
    if (!account.value.temp_unschedulable_until) return false
    const untilMs = new Date(account.value.temp_unschedulable_until).getTime()
    return !Number.isNaN(untilMs) && untilMs > nowMs.value
  })

  const hasError = computed(() => account.value.status === 'error')

  const rateLimitCountdown = computed(() => {
    if (!account.value.rate_limit_reset_at) return null
    return formatCountdownFromNow(account.value.rate_limit_reset_at)
  })

  const rateLimitResumeText = computed(() => {
    if (!rateLimitCountdown.value) return ''
    switch (account.value.rate_limit_reason) {
      case 'usage_5h':
        return t('admin.accounts.status.usage5hAutoResume', { time: rateLimitCountdown.value })
      case 'usage_7d':
        return t('admin.accounts.status.usage7dAutoResume', { time: rateLimitCountdown.value })
      case 'usage_7d_all':
        return t('admin.accounts.status.usage7dAllAutoResume', { time: rateLimitCountdown.value })
      default:
        return t('admin.accounts.status.rateLimitedAutoResume', { time: rateLimitCountdown.value })
    }
  })

  const rateLimitStatusLabel = computed(() => {
    switch (account.value.rate_limit_reason) {
      case 'usage_5h':
        return t('admin.accounts.status.usage5h')
      case 'usage_7d':
        return t('admin.accounts.status.usage7d')
      case 'usage_7d_all':
        return t('admin.accounts.status.usage7dAll')
      default:
        return t('admin.accounts.status.rateLimited')
    }
  })

  const rateLimitTooltipText = computed(() => {
    const time = formatDateTime(account.value.rate_limit_reset_at)
    switch (account.value.rate_limit_reason) {
      case 'usage_5h':
        return t('admin.accounts.status.usage5hUntil', { time })
      case 'usage_7d':
        return t('admin.accounts.status.usage7dUntil', { time })
      case 'usage_7d_all':
        return t('admin.accounts.status.usage7dAllUntil', { time })
      default:
        return t('admin.accounts.status.rateLimitedUntil', { time })
    }
  })

  const accountRateLimitBadges = computed<AccountStatusLimitBadgeItem[]>(() => {
    if (!isRateLimited.value) return []

    const fallbackTooltip = rateLimitTooltipText.value
    const extra = account.value.extra as Record<string, unknown> | undefined

    if (account.value.rate_limit_reason === 'usage_7d_all') {
      const codexLabel = 'Codex 7d'
      const sparkLabel = 'Spark 7d'
      const codexResetAt = resolvePreferredResetAt(extra?.codex_7d_reset_at, account.value.rate_limit_reset_at)
      const sparkResetAt = resolvePreferredResetAt(extra?.codex_spark_7d_reset_at, account.value.rate_limit_reset_at)

      return [
        {
          key: 'account-codex-7d',
          tone: 'amber',
          label: codexLabel,
          countdown: formatBadgeCountdown(codexResetAt),
          tooltip: codexResetAt
            ? t('admin.accounts.status.modelRateLimitedUntil', { model: codexLabel, time: formatTime(codexResetAt) })
            : fallbackTooltip,
          model: 'gpt-5.3-codex',
          modelDisplayName: codexLabel,
        },
        {
          key: 'account-spark-7d',
          tone: 'amber',
          label: sparkLabel,
          countdown: formatBadgeCountdown(sparkResetAt),
          tooltip: sparkResetAt
            ? t('admin.accounts.status.modelRateLimitedUntil', { model: sparkLabel, time: formatTime(sparkResetAt) })
            : fallbackTooltip,
          model: 'gpt-5.3-codex-spark',
          modelDisplayName: sparkLabel,
        },
      ]
    }

    const codexScopedBadges = [
      buildCodexScopeLimitBadge('normal'),
      buildCodexScopeLimitBadge('spark'),
    ].filter((item): item is AccountStatusLimitBadgeItem => item !== null)
    if (codexScopedBadges.length > 0) {
      return codexScopedBadges
    }

    return [{
      key: `account-${account.value.rate_limit_reason || '429'}`,
      tone: 'amber',
      label: account.value.rate_limit_reason === 'usage_5h'
        ? '5h'
        : account.value.rate_limit_reason === 'usage_7d'
          ? '7d'
          : '429',
      tooltip: fallbackTooltip,
    }]
  })

  const activeModelBadges = computed<AccountStatusLimitBadgeItem[]>(() => {
    const accountBadgeIdentities = new Set(
      accountRateLimitBadges.value.map((item) => buildLimitBadgeIdentity(item)),
    )
    return rawActiveModelBadges.value.filter(
      (item) => !accountBadgeIdentities.has(buildLimitBadgeIdentity(item)),
    )
  })

  const visibleLimitBadges = computed<AccountStatusLimitBadgeItem[]>(() => [
    ...accountRateLimitBadges.value,
    ...activeModelBadges.value,
  ])

  const limitBadgeLayoutClass = computed(() => {
    if (visibleLimitBadges.value.length <= 4) return 'flex flex-col gap-1'
    if (visibleLimitBadges.value.length <= 8) return 'grid grid-cols-2 gap-1'
    return 'grid grid-cols-3 gap-1'
  })

  const overloadCountdown = computed(() => {
    if (!account.value.overload_until) return null
    return formatCountdownWithSuffixFromNow(account.value.overload_until)
  })

  const statusClass = computed(() => {
    if (hasError.value) return 'badge-danger'
    if (isTempUnschedulable.value) return 'badge-warning'
    if (!account.value.schedulable) return 'badge-gray'

    switch (account.value.status) {
      case 'active':
        return 'badge-success'
      case 'inactive':
        return 'badge-gray'
      case 'error':
        return 'badge-danger'
      default:
        return 'badge-gray'
    }
  })

  const statusText = computed(() => {
    if (hasError.value) return t('admin.accounts.status.error')
    if (isTempUnschedulable.value) return t('admin.accounts.status.tempUnschedulable')
    if (!account.value.schedulable) return t('admin.accounts.status.paused')
    return t(`admin.accounts.status.${account.value.status}`)
  })

  return {
    isRateLimited,
    isOverloaded,
    isTempUnschedulable,
    hasError,
    statusClass,
    statusText,
    rateLimitResumeText,
    rateLimitStatusLabel,
    overloadCountdown,
    visibleLimitBadges,
    limitBadgeLayoutClass,
  }
}

export const useAccountErrorTooltip = () => {
  const errorTooltipVisible = ref(false)
  const errorTooltipTriggerRef = ref<HTMLElement | null>(null)
  const errorTooltipRef = ref<HTMLElement | null>(null)
  const errorTooltipStyle = ref<Record<string, string>>({})
  const errorTooltipArrowStyle = ref<Record<string, string>>({})

  const ERROR_TOOLTIP_MARGIN = 12
  const ERROR_TOOLTIP_OFFSET = 10
  const ERROR_TOOLTIP_ARROW_SIZE = 12

  const syncErrorTooltipPosition = () => {
    const trigger = errorTooltipTriggerRef.value
    const tooltip = errorTooltipRef.value
    if (!trigger || !tooltip || typeof window === 'undefined') {
      return
    }

    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight
    const maxWidth = Math.max(180, Math.min(360, viewportWidth - ERROR_TOOLTIP_MARGIN * 2))
    tooltip.style.maxWidth = `${maxWidth}px`

    const triggerRect = trigger.getBoundingClientRect()
    const tooltipRect = tooltip.getBoundingClientRect()
    const spaceAbove = triggerRect.top - ERROR_TOOLTIP_MARGIN
    const spaceBelow = viewportHeight - triggerRect.bottom - ERROR_TOOLTIP_MARGIN

    let top = triggerRect.bottom + ERROR_TOOLTIP_OFFSET
    let placement: 'top' | 'bottom' = 'bottom'
    if (tooltipRect.height > spaceBelow && spaceAbove >= spaceBelow) {
      placement = 'top'
      top = triggerRect.top - tooltipRect.height - ERROR_TOOLTIP_OFFSET
    }
    top = Math.max(
      ERROR_TOOLTIP_MARGIN,
      Math.min(top, viewportHeight - tooltipRect.height - ERROR_TOOLTIP_MARGIN)
    )

    let left = triggerRect.left + triggerRect.width / 2 - tooltipRect.width / 2
    left = Math.max(
      ERROR_TOOLTIP_MARGIN,
      Math.min(left, viewportWidth - tooltipRect.width - ERROR_TOOLTIP_MARGIN)
    )

    const arrowLeft = Math.max(
      ERROR_TOOLTIP_ARROW_SIZE,
      Math.min(
        triggerRect.left + triggerRect.width / 2 - left - ERROR_TOOLTIP_ARROW_SIZE / 2,
        tooltipRect.width - ERROR_TOOLTIP_ARROW_SIZE * 1.5
      )
    )

    errorTooltipStyle.value = {
      top: `${top}px`,
      left: `${left}px`,
      maxWidth: `${maxWidth}px`
    }
    errorTooltipArrowStyle.value = placement === 'top'
      ? { left: `${arrowLeft}px`, bottom: `-${ERROR_TOOLTIP_ARROW_SIZE / 2}px` }
      : { left: `${arrowLeft}px`, top: `-${ERROR_TOOLTIP_ARROW_SIZE / 2}px` }
  }

  const detachErrorTooltipListeners = () => {
    if (typeof window === 'undefined') return
    window.removeEventListener('resize', syncErrorTooltipPosition)
    window.removeEventListener('scroll', syncErrorTooltipPosition, true)
  }

  const showErrorTooltip = async () => {
    errorTooltipVisible.value = true
    await nextTick()
    syncErrorTooltipPosition()
    if (typeof window !== 'undefined') {
      window.addEventListener('resize', syncErrorTooltipPosition)
      window.addEventListener('scroll', syncErrorTooltipPosition, true)
    }
  }

  const hideErrorTooltip = () => {
    errorTooltipVisible.value = false
    detachErrorTooltipListeners()
  }

  onUnmounted(() => {
    detachErrorTooltipListeners()
  })

  return {
    errorTooltipVisible,
    errorTooltipTriggerRef,
    errorTooltipRef,
    errorTooltipStyle,
    errorTooltipArrowStyle,
    showErrorTooltip,
    hideErrorTooltip,
  }
}
