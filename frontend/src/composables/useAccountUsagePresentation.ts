import { computed, reactive, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { i18n } from '@/i18n'
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
import { resolveEffectiveAccountPlatformFromAccount } from '@/utils/accountProtocolGateway'
import { buildOpenAIUsageRefreshKey } from '@/utils/accountUsageRefresh'
import { resolveCodexUsageWindow } from '@/utils/codexUsage'
import { resolveGeminiChannel, resolveGeminiChannelDisplayName } from '@/utils/geminiAccount'
import { formatLocalAbsoluteTime, formatLocalTimestamp, parseEffectiveResetAt } from '@/utils/usageResetTime'

interface UsageCacheEntry {
  loading: boolean
  error: string | null
  usageInfo: AccountUsageInfo | null
  preferOpenAIFetchedUsage: boolean
  request: Promise<void> | null
}

interface UsageRowOptions {
  windowStats?: WindowStats | null
  remainingSeconds?: number | null
  inlineRemaining?: boolean
}

interface LoadUsageOptions {
  force?: boolean
  source?: 'passive' | 'active'
}

interface RefreshUsageOptions extends LoadUsageOptions {
  concurrency?: number
  resolveLoadOptions?: (account: Account) => LoadUsageOptions | undefined
}

interface RefreshUsageResult {
  total: number
  success: number
  failed: number
}

const usageCache = new Map<number, UsageCacheEntry>()
const EXPIRED_OPENAI_USAGE_REFRESH_COOLDOWN_MS = 60 * 1000

function createUsageCacheEntry(): UsageCacheEntry {
  return reactive({
    loading: false,
    error: null,
    usageInfo: null,
    preferOpenAIFetchedUsage: false,
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

function resetUsageCacheEntry(entry: UsageCacheEntry): void {
  entry.loading = false
  entry.error = null
  entry.usageInfo = null
  entry.preferOpenAIFetchedUsage = false
  entry.request = null
}

function getUsageLoadErrorMessage(): string {
  const translated = i18n.global.t('common.error')
  return typeof translated === 'string' && translated.trim() !== '' ? translated : 'Error'
}

function getRuntimePlatform(account: Account): string {
  return resolveEffectiveAccountPlatformFromAccount(account)
}

function shouldFallbackToActiveAnthropicUsage(
  account: Account,
  source: LoadUsageOptions['source'] | undefined,
  usageInfo: AccountUsageInfo,
): boolean {
  return (
    (getRuntimePlatform(account) === 'anthropic' || getRuntimePlatform(account) === 'kiro') &&
    account.type === 'oauth' &&
    source === 'passive' &&
    !usageInfo.seven_day
  )
}

export function canAccountFetchUsage(account: Account): boolean {
  const runtimePlatform = getRuntimePlatform(account)
  if (runtimePlatform === 'anthropic' || runtimePlatform === 'kiro') {
    return account.type === 'oauth' || account.type === 'setup-token'
  }
  if (runtimePlatform === 'gemini') {
    return true
  }
  if (runtimePlatform === 'antigravity') {
    return account.type === 'oauth'
  }
  if (runtimePlatform === 'openai' || runtimePlatform === 'copilot') {
    return account.type === 'oauth'
  }
  return false
}

async function performUsageLoad(account: Account, options: LoadUsageOptions = {}): Promise<void> {
  const entry = getUsageCacheEntry(account.id)

  if (entry.request) {
    await entry.request
    if (!options.force) {
      return
    }
  }

  entry.loading = true
  entry.error = null

  const source =
    options.source ??
    ((getRuntimePlatform(account) === 'anthropic' || getRuntimePlatform(account) === 'kiro') &&
    (account.type === 'oauth' || account.type === 'setup-token')
      ? 'passive'
      : undefined)

  const request = adminAPI.accounts
    .getUsage(account.id, {
      force: options.force,
      source,
    })
    .then(async (data) => {
      let resolvedUsageInfo = data

      if (shouldFallbackToActiveAnthropicUsage(account, source, data)) {
        try {
          resolvedUsageInfo = await adminAPI.accounts.getUsage(account.id, {
            force: options.force,
            source: 'active',
          })
        } catch (fallbackError) {
          console.error('Failed to supplement anthropic passive usage with active usage:', fallbackError)
        }
      }

      entry.usageInfo = resolvedUsageInfo
      entry.preferOpenAIFetchedUsage = Boolean(
        options.force &&
          (getRuntimePlatform(account) === 'openai' || getRuntimePlatform(account) === 'copilot') &&
          account.type === 'oauth',
      )
    })
    .catch((error) => {
      entry.error = getUsageLoadErrorMessage()
      entry.preferOpenAIFetchedUsage = false
      console.error('Failed to load usage:', error)
      throw error
    })
    .finally(() => {
      entry.loading = false
      entry.request = null
    })

  entry.request = request
  await request
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

function findRowByKey(
  rows: AccountUsagePresentationRow[],
  key: AccountUsagePresentationRow['key'],
): AccountUsagePresentationRow | null {
  return rows.find((row) => row.key === key) ?? null
}

function mergeOpenAIUsageRows(
  primaryRows: AccountUsagePresentationRow[],
  fallbackRows: AccountUsagePresentationRow[],
): AccountUsagePresentationRow[] {
  return buildRows(
    findRowByKey(primaryRows, 'openai-5h') ?? findRowByKey(fallbackRows, 'openai-5h'),
    findRowByKey(primaryRows, 'openai-7d') ?? findRowByKey(fallbackRows, 'openai-7d'),
  )
}

function createEmptyMeta(loadingRows: number): AccountUsagePresentationMeta {
  return { loadingRows }
}

export function resetAccountUsagePresentationCache(): void {
  usageCache.clear()
}

export function invalidateAccountUsagePresentationCache(accountIDs: number[]): void {
  const uniqueIDs = [...new Set(accountIDs.filter((accountID) => Number.isFinite(accountID) && accountID > 0))]
  uniqueIDs.forEach((accountID) => {
    const entry = usageCache.get(accountID)
    if (entry) {
      resetUsageCacheEntry(entry)
    }
  })
}

export function resolveActualUsageRefreshLoadOptions(account: Account): LoadUsageOptions {
  if ((getRuntimePlatform(account) === 'anthropic' || getRuntimePlatform(account) === 'kiro') && account.type === 'oauth') {
    return { source: 'active' }
  }

  return {}
}

export async function refreshAccountUsagePresentation(
  accounts: Account[],
  options: RefreshUsageOptions = {},
): Promise<RefreshUsageResult> {
  const refreshableAccounts = accounts.filter(canAccountFetchUsage)
  if (refreshableAccounts.length === 0) {
    return { total: 0, success: 0, failed: 0 }
  }

  const concurrency = Math.max(1, Math.min(options.concurrency ?? 4, refreshableAccounts.length))
  let index = 0
  let success = 0
  let failed = 0

  const worker = async () => {
    while (index < refreshableAccounts.length) {
      const current = refreshableAccounts[index]
      index += 1

      try {
        const accountOptions = options.resolveLoadOptions?.(current)

        await performUsageLoad(current, {
          force: accountOptions?.force ?? options.force,
          source: accountOptions?.source ?? options.source,
        })
        success += 1
      } catch {
        failed += 1
      }
    }
  }

  await Promise.all(Array.from({ length: concurrency }, () => worker()))

  return {
    total: refreshableAccounts.length,
    success,
    failed,
  }
}

export function useAccountUsagePresentation(accountSource: MaybeRefOrGetter<Account>) {
  const { t } = useI18n()
  const { nowMs, nowDate } = useUiNow()
  const account = computed(() => toValue(accountSource))
  const cacheEntry = computed(() => getUsageCacheEntry(account.value.id))
  const usageInfo = computed(() => cacheEntry.value.usageInfo)
  const lastExpiredOpenAIUsageRefresh = ref<{ key: string; requestedAt: number } | null>(null)

  const loadingRows = computed(() => {
    const runtimePlatform = getRuntimePlatform(account.value)
    if (runtimePlatform === 'anthropic') {
      return account.value.type === 'oauth' ? 3 : 1
    }
    if (runtimePlatform === 'openai') return 2
    return 1
  })

  const shouldFetchUsage = computed(() => {
    return canAccountFetchUsage(account.value)
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
  const hasCompleteCodexUsage = computed(() => codexRows.value.length === 2)

  const openAIFetchedRows = computed(() =>
    buildRows(
      buildProgressRow('openai-5h', '5h', usageInfo.value?.five_hour, 'indigo', { inlineRemaining: true }),
      buildProgressRow('openai-7d', '7d', usageInfo.value?.seven_day, 'emerald', { inlineRemaining: true }),
    ),
  )

  const hasOpenAIUsageFallback = computed(() => openAIFetchedRows.value.length > 0)

  const isActiveOpenAIRateLimited = computed(() => {
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return false
    if (!account.value.rate_limit_reset_at) return false

    const resetAt = Date.parse(account.value.rate_limit_reset_at)
    return !Number.isNaN(resetAt) && resetAt > nowMs.value
  })

  const isOpenAICodexSnapshotStale = computed(() => {
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return false

    const updatedAtRaw = account.value.extra?.codex_usage_updated_at
    if (!updatedAtRaw) return true

    const updatedAt = Date.parse(String(updatedAtRaw))
    if (Number.isNaN(updatedAt)) return true

    return nowMs.value - updatedAt >= 10 * 60 * 1000
  })

  const openAICodexEarliestResetAt = computed(() => {
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return null

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

  const shouldUseFetchedOpenAIUsage = computed(() => {
    if (!hasOpenAIUsageFallback.value) return false
    return preferFetchedOpenAIUsage.value || cacheEntry.value.preferOpenAIFetchedUsage
  })

  const openAIResolvedRows = computed(() => {
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return []

    if (shouldUseFetchedOpenAIUsage.value) {
      return mergeOpenAIUsageRows(openAIFetchedRows.value, codexRows.value)
    }

    return mergeOpenAIUsageRows(codexRows.value, openAIFetchedRows.value)
  })

  const shouldPreferFetchedOpenAIMeta = computed(() => {
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return false
    return shouldUseFetchedOpenAIUsage.value || (!hasCompleteCodexUsage.value && hasOpenAIUsageFallback.value)
  })

  const shouldAutoLoadUsageOnMount = computed(() => {
    if (getRuntimePlatform(account.value) === 'openai' && account.value.type === 'oauth') {
      return isActiveOpenAIRateLimited.value || !hasCompleteCodexUsage.value || isOpenAICodexSnapshotStale.value
    }
    return shouldFetchUsage.value
  })

  const openAIUsageRefreshKey = computed(() => buildOpenAIUsageRefreshKey(account.value))

  const snapshotUpdatedAt = computed(() => {
    if (!hasCodexUsage.value || shouldPreferFetchedOpenAIMeta.value) return null

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
  const fetchedSnapshotUpdatedAt = computed(() => {
    const updatedAtRaw = usageInfo.value?.updated_at
    if (typeof updatedAtRaw !== 'string' || updatedAtRaw.trim() === '') return null

    const parsed = new Date(updatedAtRaw)
    if (Number.isNaN(parsed.getTime())) return null

    return parsed
  })
  const fetchedSnapshotUpdatedAtText = computed(() => {
    if (!fetchedSnapshotUpdatedAt.value) return ''
    return formatLocalAbsoluteTime(fetchedSnapshotUpdatedAt.value, nowDate.value, {
      today: t('dates.today'),
      tomorrow: t('dates.tomorrow'),
    })
  })
  const fetchedSnapshotUpdatedAtTooltip = computed(() => {
    if (!fetchedSnapshotUpdatedAt.value) return ''
    return formatLocalTimestamp(fetchedSnapshotUpdatedAt.value)
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
    if (getRuntimePlatform(account.value) !== 'gemini') return null
    const credentials = account.value.credentials as GeminiCredentials | undefined
    return credentials?.tier_id || null
  })

  const geminiOAuthType = computed(() => {
    if (getRuntimePlatform(account.value) !== 'gemini') return null
    const credentials = account.value.credentials as GeminiCredentials | undefined
    return (credentials?.oauth_type || '').trim() || null
  })

  const geminiChannel = computed(() => {
    if (getRuntimePlatform(account.value) !== 'gemini') return null
    return resolveGeminiChannel({
      type: account.value.type,
      credentials: account.value.credentials as GeminiCredentials | undefined
    })
  })

  const isGeminiCodeAssist = computed(() => {
    return geminiChannel.value === 'gcp'
  })

  const geminiChannelShort = computed((): string | null => {
    if (getRuntimePlatform(account.value) !== 'gemini') return null
    return resolveGeminiChannelDisplayName(geminiChannel.value)
  })

  const geminiUserLevel = computed((): string | null => {
    if (getRuntimePlatform(account.value) !== 'gemini') return null

    const tier = (geminiTier.value || '').toString().trim()
    const tierLower = tier.toLowerCase()
    const tierUpper = tier.toUpperCase()

    if (geminiChannel.value === 'google_one') {
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

    if (geminiChannel.value === 'gcp') {
      if (tierLower === 'gcp_enterprise') return 'enterprise'
      if (tierLower === 'gcp_standard') return 'standard'
      if (tierUpper.includes('ULTRA') || tierUpper.includes('ENTERPRISE')) return 'enterprise'
      return 'standard'
    }

    if (geminiChannel.value === 'ai_studio' || geminiChannel.value === 'ai_studio_client') {
      if (tierLower === 'aistudio_tier_3') return 'tier_3'
      if (tierLower === 'aistudio_tier_2') return 'tier_2'
      if (tierLower === 'aistudio_tier_1' || tierLower === 'aistudio_paid') return 'tier_1'
      if (tierLower === 'aistudio_free') return 'free'
      if (tierUpper.includes('TIER_3')) return 'tier_3'
      if (tierUpper.includes('TIER_2')) return 'tier_2'
      if (tierUpper.includes('PAID') || tierUpper.includes('PAYG') || tierUpper.includes('PAY') || tierUpper.includes('TIER_1')) return 'tier_1'
      if (tierUpper.includes('FREE')) return 'free'
      if (geminiChannel.value === 'ai_studio') return 'free'
      return null
    }

    return null
  })

  const geminiAuthTypeLabel = computed(() => {
    if (getRuntimePlatform(account.value) !== 'gemini' || !geminiChannelShort.value) return null
    let levelLabel = geminiUserLevel.value
    if (geminiChannel.value === 'ai_studio' || geminiChannel.value === 'ai_studio_client') {
      switch (geminiUserLevel.value) {
        case 'tier_3':
          levelLabel = 'Tier 3'
          break
        case 'tier_2':
          levelLabel = 'Tier 2'
          break
        case 'tier_1':
          levelLabel = 'Tier 1'
          break
        case 'free':
          levelLabel = 'Free'
          break
      }
    }
    return levelLabel ? `${geminiChannelShort.value} ${levelLabel}` : geminiChannelShort.value
  })

  const geminiTierClass = computed(() => {
    const channel = geminiChannelShort.value
    const level = geminiUserLevel.value

    if (channel === 'AI Studio Client' || channel === 'AI Studio') {
      if (level === 'tier_3') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
      if (level === 'tier_2') return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
      if (level === 'tier_1') return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    }
    if (channel === 'Vertex AI') {
      return 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300'
    }
    if (channel === 'Google One') {
      if (level === 'ultra') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
      if (level === 'pro') return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    }
    if (channel === 'GCP') {
      if (level === 'enterprise') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
      return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    }
    return ''
  })

  const geminiQuotaPolicyChannel = computed(() => {
    if (geminiChannel.value === 'google_one') {
      return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.channel')
    }
    if (geminiChannel.value === 'gcp') {
      return t('admin.accounts.gemini.quotaPolicy.rows.gcp.channel')
    }
    if (geminiChannel.value === 'vertex_ai') {
      return t('admin.accounts.gemini.quotaPolicy.rows.vertex.channel')
    }
    if (geminiChannel.value === 'ai_studio_client') {
      return t('admin.accounts.gemini.quotaPolicy.rows.customOAuth.channel')
    }
    return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.channel')
  })

  const geminiQuotaPolicyLimits = computed(() => {
    const tierLower = (geminiTier.value || '').toString().trim().toLowerCase()

    if (geminiChannel.value === 'google_one') {
      if (tierLower === 'google_ai_ultra' || geminiUserLevel.value === 'ultra') {
        return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsUltra')
      }
      if (tierLower === 'google_ai_pro' || geminiUserLevel.value === 'pro') {
        return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsPro')
      }
      return t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsFree')
    }

    if (geminiChannel.value === 'gcp') {
      if (tierLower === 'gcp_enterprise' || geminiUserLevel.value === 'enterprise') {
        return t('admin.accounts.gemini.quotaPolicy.rows.gcp.limitsEnterprise')
      }
      return t('admin.accounts.gemini.quotaPolicy.rows.gcp.limitsStandard')
    }

    if (geminiChannel.value === 'vertex_ai') {
      return t('admin.accounts.gemini.quotaPolicy.rows.vertex.limits')
    }

    if (geminiChannel.value === 'ai_studio_client') {
      if (tierLower === 'aistudio_tier_3' || geminiUserLevel.value === 'tier_3') {
        return t('admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsTier3')
      }
      if (tierLower === 'aistudio_tier_2' || geminiUserLevel.value === 'tier_2') {
        return t('admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsTier2')
      }
      if (tierLower === 'aistudio_tier_1' || tierLower === 'aistudio_paid' || geminiUserLevel.value === 'tier_1') {
        return t('admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsPaid')
      }
      return t('admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsFree')
    }

    if (tierLower === 'aistudio_tier_3' || geminiUserLevel.value === 'tier_3') {
      return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsTier3')
    }
    if (tierLower === 'aistudio_tier_2' || geminiUserLevel.value === 'tier_2') {
      return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsTier2')
    }
    if (tierLower === 'aistudio_tier_1' || tierLower === 'aistudio_paid' || geminiUserLevel.value === 'tier_1') {
      return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsTier1')
    }
    return t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsFree')
  })

  const geminiQuotaPolicyDocsUrl = computed(() => {
    if (geminiChannel.value === 'google_one' || geminiChannel.value === 'gcp') {
      return 'https://developers.google.com/gemini-code-assist/resources/quotas'
    }
    if (geminiChannel.value === 'vertex_ai') {
      return 'https://cloud.google.com/vertex-ai/generative-ai/docs/quotas'
    }
    return 'https://ai.google.dev/pricing'
  })

  const geminiUsesSharedDaily = computed(() => {
    if (getRuntimePlatform(account.value) !== 'gemini') return false
    return (
      !!usageInfo.value?.gemini_shared_daily ||
      !!usageInfo.value?.gemini_shared_minute ||
      geminiOAuthType.value === 'google_one' ||
      isGeminiCodeAssist.value
    )
  })

  const geminiRows = computed(() => {
    if (getRuntimePlatform(account.value) !== 'gemini' || !usageInfo.value) return []

    if (geminiUsesSharedDaily.value) {
      return buildRows(buildProgressRow('gemini-shared-daily', '1d', usageInfo.value.gemini_shared_daily, 'indigo'))
    }

    return buildRows(
      buildProgressRow('gemini-pro-daily', 'pro', usageInfo.value.gemini_pro_daily, 'indigo'),
      buildProgressRow('gemini-flash-daily', 'flash', usageInfo.value.gemini_flash_daily, 'emerald'),
    )
  })

  const hasAccountQuota = computed(() => {
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
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return []
    return openAIResolvedRows.value
  })

  const openAIEarliestResetAt = computed(() => {
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return null

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

  const loadUsage = async (options: LoadUsageOptions = {}) => {
    const currentAccount = account.value
    if (!shouldFetchUsage.value) return
    await performUsageLoad(currentAccount, options).catch(() => {
      const entry = getUsageCacheEntry(currentAccount.id)
      if (!entry.error) {
        entry.error = t('common.error')
      }
    })
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
    if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return
    if (!isActiveOpenAIRateLimited.value && hasCompleteCodexUsage.value && !isOpenAICodexSnapshotStale.value) return

    loadUsage().catch((error) => {
      console.error('Failed to refresh OpenAI usage:', error)
    })
  })

  watch(
    () => [account.value.id, openAIEarliestResetAt.value, nowMs.value] as const,
    ([accountID, earliestResetAt, currentNow]) => {
      if (getRuntimePlatform(account.value) !== 'openai' || account.value.type !== 'oauth') return
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

    if (getRuntimePlatform(account.value) === 'openai' && account.value.type === 'oauth') {
      meta.snapshotUpdatedAtText = shouldPreferFetchedOpenAIMeta.value
        ? fetchedSnapshotUpdatedAtText.value || openAISnapshotUpdatedAtText.value || undefined
        : openAISnapshotUpdatedAtText.value || fetchedSnapshotUpdatedAtText.value || undefined
      meta.snapshotUpdatedAtTooltip = shouldPreferFetchedOpenAIMeta.value
        ? fetchedSnapshotUpdatedAtTooltip.value || openAISnapshotUpdatedAtTooltip.value || undefined
        : openAISnapshotUpdatedAtTooltip.value || fetchedSnapshotUpdatedAtTooltip.value || undefined

      if (shouldUseFetchedOpenAIUsage.value && openAIResolvedRows.value.length > 0) {
        state = 'bars'
        windowRows = openAIResolvedRows.value
      } else if (isActiveOpenAIRateLimited.value && currentState.loading && openAIFetchedRows.value.length === 0) {
        state = 'loading'
      } else if (openAIResolvedRows.value.length > 0) {
        state = 'bars'
        windowRows = openAIResolvedRows.value
      } else if (currentState.loading) {
        state = 'loading'
      } else if (openAIFetchedRows.value.length > 0) {
        state = 'bars'
        windowRows = openAIFetchedRows.value
      }
    } else if (getRuntimePlatform(account.value) === 'anthropic' && (account.value.type === 'oauth' || account.value.type === 'setup-token')) {
      if (currentState.loading) {
        state = 'loading'
      } else if (currentState.error) {
        state = 'error'
      } else if (anthropicRows.value.length > 0) {
        state = 'bars'
        windowRows = anthropicRows.value
        meta.snapshotUpdatedAtText = fetchedSnapshotUpdatedAtText.value || undefined
        meta.snapshotUpdatedAtTooltip = fetchedSnapshotUpdatedAtTooltip.value || undefined
        if (usageInfo.value?.source === 'passive') {
          meta.sampledBadgeLabel = t('admin.accounts.usageWindow.sampledBadge')
          meta.sampledBadgeTooltip = t('admin.accounts.usageWindow.passiveSampled')
        }
      }
    } else if (getRuntimePlatform(account.value) === 'antigravity' && account.value.type === 'oauth') {
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
    } else if (getRuntimePlatform(account.value) === 'gemini') {
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
    } else if (hasAccountQuota.value) {
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
