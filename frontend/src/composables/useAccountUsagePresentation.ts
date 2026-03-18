import { computed, reactive, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useUiNow } from '@/composables/useUiNow'
import type {
  Account,
  AccountUsageInfo,
  AccountUsagePresentation,
  AccountUsagePresentationMeta,
  AccountUsagePresentationRow,
  AccountUsageRowColor,
  GeminiCredentials,
  UsageProgress,
  WindowStats,
} from '@/types'
import { buildOpenAIUsageRefreshKey } from '@/utils/accountUsageRefresh'
import { resolveCodexUsageWindow } from '@/utils/codexUsage'
import { formatLocalAbsoluteTime, formatLocalTimestamp, parseEffectiveResetAt } from '@/utils/usageResetTime'

interface UsageCacheEntry {
  loading: boolean
  error: string | null
  usageInfo: AccountUsageInfo | null
  request: Promise<void> | null
}

interface UsageRowOptions {
  windowStats?: WindowStats | null
  remainingSeconds?: number | null
  inlineRemaining?: boolean
}

const usageCache = new Map<number, UsageCacheEntry>()
const EXPIRED_OPENAI_USAGE_REFRESH_COOLDOWN_MS = 60 * 1000

function createUsageCacheEntry(): UsageCacheEntry {
  return reactive({
    loading: false,
    error: null,
    usageInfo: null,
    request: null,
  }) as UsageCacheEntry
}

function getUsageCacheEntry(accountID: number): UsageCacheEntry {
  const cached = usageCache.get(accountID)
  if (cached) return cached

  const created = createUsageCacheEntry()
  usageCache.set(accountID, created)
  return created
}

function buildUsageRow(
  key: string,
  label: string,
  utilization: number,
  resetsAt: string | null,
  color: AccountUsageRowColor,
  options: UsageRowOptions = {},
): AccountUsagePresentationRow {
  const normalizedResetAt = typeof resetsAt === 'string' && resetsAt.trim() !== '' ? resetsAt : null
  const effectiveResetAt =
    normalizedResetAt === null && (options.remainingSeconds ?? 0) > 0
      ? parseEffectiveResetAt(null, options.remainingSeconds ?? null)
      : null

  return {
    key,
    label,
    utilization,
    resetsAt: normalizedResetAt ?? (effectiveResetAt ? effectiveResetAt.toISOString() : null),
    remainingSeconds: options.remainingSeconds ?? null,
    windowStats: options.windowStats ?? null,
    color,
    inlineRemaining: options.inlineRemaining ?? false,
  }
}

function buildProgressRow(
  key: string,
  label: string,
  progress: UsageProgress | null | undefined,
  color: AccountUsageRowColor,
  options: UsageRowOptions = {},
): AccountUsagePresentationRow | null {
  if (!progress) return null

  return buildUsageRow(key, label, progress.utilization, progress.resets_at, color, {
    windowStats: progress.window_stats,
    remainingSeconds: progress.remaining_seconds,
    inlineRemaining: options.inlineRemaining,
  })
}

function buildRows(...rows: Array<AccountUsagePresentationRow | null>): AccountUsagePresentationRow[] {
  return rows.filter((row): row is AccountUsagePresentationRow => row !== null)
}

function createEmptyMeta(loadingRows: number): AccountUsagePresentationMeta {
  return { loadingRows }
}

export function resetAccountUsagePresentationCache(): void {
  usageCache.clear()
}

export function useAccountUsagePresentation(accountSource: MaybeRefOrGetter<Account>) {
  const { t } = useI18n()
  const { nowMs, nowDate } = useUiNow()
  const account = computed(() => toValue(accountSource))
  const cacheEntry = computed(() => getUsageCacheEntry(account.value.id))
  const usageInfo = computed(() => cacheEntry.value.usageInfo)
  const lastExpiredOpenAIUsageRefresh = ref<{ key: string; requestedAt: number } | null>(null)

  const loadingRows = computed(() => {
    if (account.value.platform === 'anthropic') {
      return account.value.type === 'oauth' ? 3 : 1
    }
    if (account.value.platform === 'openai') return 2
    return 1
  })

  const shouldFetchUsage = computed(() => {
    if (account.value.platform === 'anthropic') {
      return account.value.type === 'oauth' || account.value.type === 'setup-token'
    }
    if (account.value.platform === 'gemini') {
      return true
    }
    if (account.value.platform === 'antigravity') {
      return account.value.type === 'oauth'
    }
    if (account.value.platform === 'openai') {
      return account.value.type === 'oauth'
    }
    return false
  })

  const codex5hWindow = computed(() => resolveCodexUsageWindow(account.value.extra, '5h', nowDate.value))
  const codex7dWindow = computed(() => resolveCodexUsageWindow(account.value.extra, '7d', nowDate.value))

  const codexRows = computed(() =>
    buildRows(
      codex5hWindow.value.usedPercent === null
        ? null
        : buildUsageRow('openai-5h', '5h', codex5hWindow.value.usedPercent, codex5hWindow.value.resetAt, 'indigo', {
            inlineRemaining: true,
          }),
      codex7dWindow.value.usedPercent === null
        ? null
        : buildUsageRow('openai-7d', '7d', codex7dWindow.value.usedPercent, codex7dWindow.value.resetAt, 'emerald', {
            inlineRemaining: true,
          }),
    ),
  )

  const hasCodexUsage = computed(() => codexRows.value.length > 0)

  const openAIFetchedRows = computed(() =>
    buildRows(
      buildProgressRow('openai-5h', '5h', usageInfo.value?.five_hour, 'indigo', { inlineRemaining: true }),
      buildProgressRow('openai-7d', '7d', usageInfo.value?.seven_day, 'emerald', { inlineRemaining: true }),
    ),
  )

  const hasOpenAIUsageFallback = computed(() => openAIFetchedRows.value.length > 0)

  const isActiveOpenAIRateLimited = computed(() => {
    if (account.value.platform !== 'openai' || account.value.type !== 'oauth') return false
    if (!account.value.rate_limit_reset_at) return false

    const resetAt = Date.parse(account.value.rate_limit_reset_at)
    return !Number.isNaN(resetAt) && resetAt > nowMs.value
  })

  const isOpenAICodexSnapshotStale = computed(() => {
    if (account.value.platform !== 'openai' || account.value.type !== 'oauth') return false

    const updatedAtRaw = account.value.extra?.codex_usage_updated_at
    if (!updatedAtRaw) return true

    const updatedAt = Date.parse(String(updatedAtRaw))
    if (Number.isNaN(updatedAt)) return true

    return nowMs.value - updatedAt >= 10 * 60 * 1000
  })

  const openAICodexEarliestResetAt = computed(() => {
    if (account.value.platform !== 'openai' || account.value.type !== 'oauth') return null

    let earliestResetAt: string | null = null
    let earliestResetMs = Number.POSITIVE_INFINITY

    for (const row of codexRows.value) {
      const effectiveResetAt = parseEffectiveResetAt(row.resetsAt, row.remainingSeconds)
      if (!effectiveResetAt) continue

      const resetMs = effectiveResetAt.getTime()
      if (resetMs < earliestResetMs) {
        earliestResetMs = resetMs
        earliestResetAt = effectiveResetAt.toISOString()
      }
    }

    return earliestResetAt
  })

  const hasExpiredOpenAICodexWindow = computed(() => {
    if (!openAICodexEarliestResetAt.value) return false

    const resetAt = Date.parse(openAICodexEarliestResetAt.value)
    return !Number.isNaN(resetAt) && resetAt <= nowMs.value
  })

  const preferFetchedOpenAIUsage = computed(() => {
    return (
      isActiveOpenAIRateLimited.value ||
      isOpenAICodexSnapshotStale.value ||
      hasExpiredOpenAICodexWindow.value
    ) && hasOpenAIUsageFallback.value
  })

  const shouldAutoLoadUsageOnMount = computed(() => {
    if (account.value.platform === 'openai' && account.value.type === 'oauth') {
      return isActiveOpenAIRateLimited.value || !hasCodexUsage.value || isOpenAICodexSnapshotStale.value
    }
    return shouldFetchUsage.value
  })

  const openAIUsageRefreshKey = computed(() => buildOpenAIUsageRefreshKey(account.value))

  const snapshotUpdatedAt = computed(() => {
    if (!hasCodexUsage.value || preferFetchedOpenAIUsage.value) return null

    const updatedAtRaw = account.value.extra?.codex_usage_updated_at
    if (typeof updatedAtRaw !== 'string' || updatedAtRaw.trim() === '') return null

    const parsed = new Date(updatedAtRaw)
    if (Number.isNaN(parsed.getTime())) return null

    return parsed
  })

  const openAISnapshotUpdatedAtText = computed(() => {
    if (!snapshotUpdatedAt.value) return ''
    return formatLocalAbsoluteTime(snapshotUpdatedAt.value, nowDate.value, {
      today: t('dates.today'),
      tomorrow: t('dates.tomorrow'),
    })
  })

  const openAISnapshotUpdatedAtTooltip = computed(() => {
    if (!snapshotUpdatedAt.value) return ''
    return formatLocalTimestamp(snapshotUpdatedAt.value)
  })

  const getAntigravityUsageFromAPI = (modelNames: string[]) => {
    const quota = usageInfo.value?.antigravity_quota
    if (!quota) return null

    let maxUtilization = 0
    let earliestReset: string | null = null

    for (const model of modelNames) {
      const modelQuota = quota[model]
      if (!modelQuota) continue

      if (modelQuota.utilization > maxUtilization) {
        maxUtilization = modelQuota.utilization
      }

      if (modelQuota.reset_time && (!earliestReset || modelQuota.reset_time < earliestReset)) {
        earliestReset = modelQuota.reset_time
      }
    }

    if (maxUtilization === 0 && earliestReset === null) {
      const hasAnyData = modelNames.some((model) => quota[model])
      if (!hasAnyData) return null
    }

    return { utilization: maxUtilization, resetTime: earliestReset }
  }

  const antigravityRows = computed(() =>
    buildRows(
      (() => {
        const usage = getAntigravityUsageFromAPI(['gemini-3-pro-low', 'gemini-3-pro-high', 'gemini-3-pro-preview'])
        return usage
          ? buildUsageRow('antigravity-g3p', t('admin.accounts.usageWindow.gemini3Pro'), usage.utilization, usage.resetTime, 'indigo')
          : null
      })(),
      (() => {
        const usage = getAntigravityUsageFromAPI(['gemini-3-flash'])
        return usage
          ? buildUsageRow('antigravity-g3f', t('admin.accounts.usageWindow.gemini3Flash'), usage.utilization, usage.resetTime, 'emerald')
          : null
      })(),
      (() => {
        const usage = getAntigravityUsageFromAPI(['gemini-2.5-flash-image', 'gemini-3.1-flash-image', 'gemini-3-pro-image'])
        return usage
          ? buildUsageRow('antigravity-image', t('admin.accounts.usageWindow.gemini3Image'), usage.utilization, usage.resetTime, 'purple')
          : null
      })(),
      (() => {
        const usage = getAntigravityUsageFromAPI([
          'claude-opus-4.1',
          'claude-sonnet-4.5',
          'claude-haiku-4.5',
          'claude-opus-4-1-20250805',
          'claude-sonnet-4-5-20250929',
          'claude-haiku-4-5-20251001',
          'claude-sonnet-4-5',
          'claude-opus-4-5-thinking',
          'claude-sonnet-4-6',
          'claude-opus-4-6',
          'claude-opus-4-6-thinking',
        ])
        return usage
          ? buildUsageRow('antigravity-claude', t('admin.accounts.usageWindow.claude'), usage.utilization, usage.resetTime, 'amber')
          : null
      })(),
    ),
  )

  const antigravityTier = computed(() => {
    const loadCodeAssist = account.value.extra?.load_code_assist as Record<string, unknown> | undefined
    if (!loadCodeAssist) return null

    const paidTier = loadCodeAssist.paidTier as Record<string, unknown> | undefined
    if (paidTier && typeof paidTier.id === 'string') {
      return paidTier.id
    }

    const currentTier = loadCodeAssist.currentTier as Record<string, unknown> | undefined
    if (currentTier && typeof currentTier.id === 'string') {
      return currentTier.id
    }

    return null
  })

  const antigravityTierLabel = computed(() => {
    switch (antigravityTier.value) {
      case 'free-tier':
        return t('admin.accounts.tier.free')
      case 'g1-pro-tier':
        return t('admin.accounts.tier.pro')
      case 'g1-ultra-tier':
        return t('admin.accounts.tier.ultra')
      default:
        return null
    }
  })

  const antigravityTierClass = computed(() => {
    switch (antigravityTier.value) {
      case 'free-tier':
        return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
      case 'g1-pro-tier':
        return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
      case 'g1-ultra-tier':
        return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
      default:
        return ''
    }
  })

  const hasIneligibleTiers = computed(() => {
    const loadCodeAssist = account.value.extra?.load_code_assist as Record<string, unknown> | undefined
    const ineligibleTiers = loadCodeAssist?.ineligibleTiers as unknown[] | undefined
    return Array.isArray(ineligibleTiers) && ineligibleTiers.length > 0
  })

  const geminiTier = computed(() => {
    if (account.value.platform !== 'gemini') return null
    const credentials = account.value.credentials as GeminiCredentials | undefined
    return credentials?.tier_id || null
  })

  const geminiOAuthType = computed(() => {
    if (account.value.platform !== 'gemini') return null
    const credentials = account.value.credentials as GeminiCredentials | undefined
    return (credentials?.oauth_type || '').trim() || null
  })

  const isGeminiCodeAssist = computed(() => {
    if (account.value.platform !== 'gemini') return false
    const credentials = account.value.credentials as GeminiCredentials | undefined
    return credentials?.oauth_type === 'code_assist' || (!credentials?.oauth_type && !!credentials?.project_id)
  })

  const geminiChannelShort = computed((): 'ai studio' | 'gcp' | 'google one' | 'client' | null => {
    if (account.value.platform !== 'gemini') return null
    if (account.value.type === 'apikey') return 'ai studio'
    if (geminiOAuthType.value === 'google_one') return 'google one'
    if (isGeminiCodeAssist.value) return 'gcp'
    if (geminiOAuthType.value === 'ai_studio') return 'client'
    return 'ai studio'
  })

  const geminiUserLevel = computed((): string | null => {
    if (account.value.platform !== 'gemini') return null

    const tier = (geminiTier.value || '').toString().trim()
    const tierLower = tier.toLowerCase()
    const tierUpper = tier.toUpperCase()

    if (geminiOAuthType.value === 'google_one') {
      if (tierLower === 'google_one_free') return 'free'
      if (tierLower === 'google_ai_pro') return 'pro'
      if (tierLower === 'google_ai_ultra') return 'ultra'
      if (tierUpper === 'AI_PREMIUM' || tierUpper === 'GOOGLE_ONE_STANDARD') return 'pro'
      if (tierUpper === 'GOOGLE_ONE_UNLIMITED') return 'ultra'
      if (tierUpper === 'FREE' || tierUpper === 'GOOGLE_ONE_BASIC' || tierUpper === 'GOOGLE_ONE_UNKNOWN' || tierUpper === '') {
        return 'free'
      }
      return null
    }

    if (isGeminiCodeAssist.value) {
      if (tierLower === 'gcp_enterprise') return 'enterprise'
      if (tierLower === 'gcp_standard') return 'standard'
      if (tierUpper.includes('ULTRA') || tierUpper.includes('ENTERPRISE')) return 'enterprise'
      return 'standard'
    }

    if (account.value.type === 'apikey' || geminiOAuthType.value === 'ai_studio') {
      if (tierLower === 'aistudio_paid') return 'paid'
      if (tierLower === 'aistudio_free') return 'free'
      if (tierUpper.includes('PAID') || tierUpper.includes('PAYG') || tierUpper.includes('PAY')) return 'paid'
      if (tierUpper.includes('FREE')) return 'free'
      if (account.value.type === 'apikey') return 'free'
      return null
    }

    return null
  })

  const geminiAuthTypeLabel = computed(() => {
    if (account.value.platform !== 'gemini' || !geminiChannelShort.value) return null
    return geminiUserLevel.value ? `${geminiChannelShort.value} ${geminiUserLevel.value}` : geminiChannelShort.value
  })

  const geminiTierClass = computed(() => {
    const channel = geminiChannelShort.value
    const level = geminiUserLevel.value

    if (channel === 'client' || channel === 'ai studio') {
      return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    }
    if (channel === 'google one') {
      if (level === 'ultra') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
      if (level === 'pro') return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    }
    if (channel === 'gcp') {
      if (level === 'enterprise') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
      return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    }
    return ''
  })

  const geminiQuotaPolicyChannel = computed(() => {
    if (geminiOAuthType.value === 'google_one') {
      return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.channel')
    }
    if (isGeminiCodeAssist.value) {
      return t('admin.accounts.gemini.quotaPolicy.rows.gcp.channel')
    }
    return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.channel')
  })

  const geminiQuotaPolicyLimits = computed(() => {
    const tierLower = (geminiTier.value || '').toString().trim().toLowerCase()

    if (geminiOAuthType.value === 'google_one') {
      if (tierLower === 'google_ai_ultra' || geminiUserLevel.value === 'ultra') {
        return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsUltra')
      }
      if (tierLower === 'google_ai_pro' || geminiUserLevel.value === 'pro') {
        return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsPro')
      }
      return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsFree')
    }

    if (isGeminiCodeAssist.value) {
      if (tierLower === 'gcp_enterprise' || geminiUserLevel.value === 'enterprise') {
        return t('admin.accounts.gemini.quotaPolicy.rows.gcp.limitsEnterprise')
      }
      return t('admin.accounts.gemini.quotaPolicy.rows.gcp.limitsStandard')
    }

    if (tierLower === 'aistudio_paid' || geminiUserLevel.value === 'paid') {
      return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsPaid')
    }
    return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsFree')
  })

  const geminiQuotaPolicyDocsUrl = computed(() => {
    if (geminiOAuthType.value === 'google_one' || isGeminiCodeAssist.value) {
      return 'https://developers.google.com/gemini-code-assist/resources/quotas'
    }
    return 'https://ai.google.dev/pricing'
  })

  const geminiUsesSharedDaily = computed(() => {
    if (account.value.platform !== 'gemini') return false
    return (
      !!usageInfo.value?.gemini_shared_daily ||
      !!usageInfo.value?.gemini_shared_minute ||
      geminiOAuthType.value === 'google_one' ||
      isGeminiCodeAssist.value
    )
  })

  const geminiRows = computed(() => {
    if (account.value.platform !== 'gemini' || !usageInfo.value) return []

    if (geminiUsesSharedDaily.value) {
      return buildRows(buildProgressRow('gemini-shared-daily', '1d', usageInfo.value.gemini_shared_daily, 'indigo'))
    }

    return buildRows(
      buildProgressRow('gemini-pro-daily', 'pro', usageInfo.value.gemini_pro_daily, 'indigo'),
      buildProgressRow('gemini-flash-daily', 'flash', usageInfo.value.gemini_flash_daily, 'emerald'),
    )
  })

  const hasApiKeyQuota = computed(() => {
    if (account.value.type !== 'apikey') return false
    return (
      (account.value.quota_daily_limit ?? 0) > 0 ||
      (account.value.quota_weekly_limit ?? 0) > 0 ||
      (account.value.quota_limit ?? 0) > 0
    )
  })

  const makeQuotaRow = (
    key: string,
    label: string,
    used: number,
    limit: number,
    color: AccountUsageRowColor,
    startKey?: string,
  ): AccountUsagePresentationRow | null => {
    if (limit <= 0) return null

    let resetsAt: string | null = null
    if (startKey) {
      const startStr = account.value.extra?.[startKey] as string | undefined
      if (startStr) {
        const startDate = new Date(startStr)
        const periodMs = startKey.includes('daily') ? 24 * 60 * 60 * 1000 : 7 * 24 * 60 * 60 * 1000
        resetsAt = new Date(startDate.getTime() + periodMs).toISOString()
      }
    }

    const utilization = limit > 0 ? (used / limit) * 100 : 0
    return buildUsageRow(key, label, utilization, resetsAt, color)
  }

  const apiKeyQuotaRows = computed(() =>
    buildRows(
      makeQuotaRow('apikey-daily', '1d', account.value.quota_daily_used ?? 0, account.value.quota_daily_limit ?? 0, 'indigo', 'quota_daily_start'),
      makeQuotaRow('apikey-weekly', '7d', account.value.quota_weekly_used ?? 0, account.value.quota_weekly_limit ?? 0, 'emerald', 'quota_weekly_start'),
      makeQuotaRow('apikey-total', 'total', account.value.quota_used ?? 0, account.value.quota_limit ?? 0, 'purple'),
    ),
  )

  const anthropicRows = computed(() =>
    buildRows(
      buildProgressRow('anthropic-5h', '5h', usageInfo.value?.five_hour, 'indigo'),
      buildProgressRow('anthropic-7d', '7d', usageInfo.value?.seven_day, 'emerald'),
      buildProgressRow('anthropic-7d-sonnet', '7d S', usageInfo.value?.seven_day_sonnet, 'purple'),
    ),
  )

  const openAIRefreshRows = computed(() => {
    if (account.value.platform !== 'openai' || account.value.type !== 'oauth') return []
    if (preferFetchedOpenAIUsage.value && openAIFetchedRows.value.length > 0) return openAIFetchedRows.value
    if (codexRows.value.length > 0) return codexRows.value
    return openAIFetchedRows.value
  })

  const openAIEarliestResetAt = computed(() => {
    if (account.value.platform !== 'openai' || account.value.type !== 'oauth') return null

    let earliestResetAt: string | null = null
    let earliestResetMs = Number.POSITIVE_INFINITY

    for (const row of openAIRefreshRows.value) {
      const effectiveResetAt = parseEffectiveResetAt(row.resetsAt, row.remainingSeconds)
      if (!effectiveResetAt) continue

      const resetMs = effectiveResetAt.getTime()
      if (resetMs < earliestResetMs) {
        earliestResetMs = resetMs
        earliestResetAt = effectiveResetAt.toISOString()
      }
    }

    return earliestResetAt
  })

  const loadUsage = async () => {
    const currentAccount = account.value
    if (!shouldFetchUsage.value) return

    const entry = getUsageCacheEntry(currentAccount.id)
    if (entry.request) {
      await entry.request
      return
    }

    entry.loading = true
    entry.error = null
    entry.request = adminAPI.accounts
      .getUsage(currentAccount.id)
      .then((data) => {
        entry.usageInfo = data
      })
      .catch((error) => {
        entry.error = t('common.error')
        console.error('Failed to load usage:', error)
      })
      .finally(() => {
        entry.loading = false
        entry.request = null
      })

    await entry.request
  }

  watch(
    () => [account.value.id, shouldAutoLoadUsageOnMount.value] as const,
    ([, shouldLoad]) => {
      if (!shouldLoad) return
      if (cacheEntry.value.usageInfo || cacheEntry.value.loading) return
      loadUsage().catch((error) => {
        console.error('Failed to initialize account usage:', error)
      })
    },
    { immediate: true },
  )

  watch(openAIUsageRefreshKey, (nextKey, prevKey) => {
    if (!prevKey || nextKey === prevKey) return
    if (account.value.platform !== 'openai' || account.value.type !== 'oauth') return
    if (!isActiveOpenAIRateLimited.value && hasCodexUsage.value && !isOpenAICodexSnapshotStale.value) return

    loadUsage().catch((error) => {
      console.error('Failed to refresh OpenAI usage:', error)
    })
  })

  watch(
    () => [account.value.id, openAIEarliestResetAt.value, nowMs.value] as const,
    ([accountID, earliestResetAt, currentNow]) => {
      if (account.value.platform !== 'openai' || account.value.type !== 'oauth') return
      if (!earliestResetAt) return

      const resetAtMs = Date.parse(earliestResetAt)
      if (Number.isNaN(resetAtMs) || resetAtMs > currentNow) return

      const refreshKey = `${accountID}:${earliestResetAt}`
      const lastRefresh = lastExpiredOpenAIUsageRefresh.value
      if (
        lastRefresh &&
        lastRefresh.key === refreshKey &&
        currentNow - lastRefresh.requestedAt < EXPIRED_OPENAI_USAGE_REFRESH_COOLDOWN_MS
      ) {
        return
      }

      lastExpiredOpenAIUsageRefresh.value = {
        key: refreshKey,
        requestedAt: currentNow,
      }

      loadUsage().catch((error) => {
        console.error('Failed to refresh expired OpenAI usage window:', error)
      })
    },
    { immediate: true },
  )

  const presentation = computed<AccountUsagePresentation>(() => {
    const meta = createEmptyMeta(loadingRows.value)
    const currentState = cacheEntry.value
    let state: AccountUsagePresentation['state'] = 'empty'
    let windowRows: AccountUsagePresentationRow[] = []

    if (account.value.platform === 'openai' && account.value.type === 'oauth') {
      meta.snapshotUpdatedAtText = openAISnapshotUpdatedAtText.value || undefined
      meta.snapshotUpdatedAtTooltip = openAISnapshotUpdatedAtTooltip.value || undefined

      if (preferFetchedOpenAIUsage.value && openAIFetchedRows.value.length > 0) {
        state = 'bars'
        windowRows = openAIFetchedRows.value
      } else if (isActiveOpenAIRateLimited.value && currentState.loading && openAIFetchedRows.value.length === 0) {
        state = 'loading'
      } else if (codexRows.value.length > 0) {
        state = 'bars'
        windowRows = codexRows.value
      } else if (currentState.loading) {
        state = 'loading'
      } else if (openAIFetchedRows.value.length > 0) {
        state = 'bars'
        windowRows = openAIFetchedRows.value
      }
    } else if (account.value.platform === 'anthropic' && (account.value.type === 'oauth' || account.value.type === 'setup-token')) {
      if (currentState.loading) {
        state = 'loading'
      } else if (currentState.error) {
        state = 'error'
      } else if (anthropicRows.value.length > 0) {
        state = 'bars'
        windowRows = anthropicRows.value
      }
    } else if (account.value.platform === 'antigravity' && account.value.type === 'oauth') {
      meta.antigravityTierLabel = antigravityTierLabel.value
      meta.antigravityTierClass = antigravityTierClass.value
      meta.hasIneligibleTiers = hasIneligibleTiers.value

      if (currentState.loading) {
        state = 'loading'
      } else if (currentState.error) {
        state = 'error'
      } else if (antigravityRows.value.length > 0) {
        state = 'bars'
        windowRows = antigravityRows.value
      }
    } else if (account.value.platform === 'gemini') {
      meta.geminiAuthTypeLabel = geminiAuthTypeLabel.value
      meta.geminiTierClass = geminiTierClass.value
      meta.geminiQuotaPolicyChannel = geminiQuotaPolicyChannel.value
      meta.geminiQuotaPolicyLimits = geminiQuotaPolicyLimits.value
      meta.geminiQuotaPolicyDocsUrl = geminiQuotaPolicyDocsUrl.value

      if (currentState.loading) {
        state = 'loading'
      } else if (currentState.error) {
        state = 'error'
      } else if (geminiRows.value.length > 0) {
        state = 'bars'
        windowRows = geminiRows.value
        meta.noteText = t('admin.accounts.gemini.quotaPolicy.simulatedNote') || 'Simulated quota'
      } else {
        state = 'unlimited'
      }
    } else if (hasApiKeyQuota.value) {
      if (apiKeyQuotaRows.value.length > 0) {
        state = 'bars'
        windowRows = apiKeyQuotaRows.value
      }
    }

    const resetRows = state === 'bars'
      ? windowRows
          .filter((row) => parseEffectiveResetAt(row.resetsAt, row.remainingSeconds) !== null)
          .map((row) => ({
            key: row.key,
            label: row.label,
            resetsAt: row.resetsAt,
            remainingSeconds: row.remainingSeconds,
          }))
      : []

    return {
      loading: currentState.loading,
      error: currentState.error,
      state,
      windowRows,
      resetRows,
      meta,
    }
  })

  return {
    presentation,
    loadUsage,
    shouldFetchUsage,
  }
}
