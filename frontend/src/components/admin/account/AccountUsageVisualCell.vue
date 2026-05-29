<template>
  <div
    ref="rootRef"
    :class="[
      'flex min-w-0 max-w-full flex-col justify-center overflow-hidden',
      compact ? 'gap-1' : 'gap-1.5'
    ]"
    data-testid="account-usage-visual-cell"
  >
    <div v-if="tierBadges.length > 0" class="flex min-w-0 flex-wrap items-center gap-1">
      <span
        v-for="badge in tierBadges"
        :key="badge.label"
        :class="[
          'inline-flex max-w-[132px] items-center rounded-full px-1.5 py-0.5 text-[9px] font-bold',
          badge.className
        ]"
        :title="badge.label"
      >
        <span class="min-w-0 truncate">{{ badge.label }}</span>
      </span>
    </div>

    <div v-if="presentation.state === 'loading'" class="space-y-1.5">
      <div
        v-for="index in skeletonRows"
        :key="index"
        class="flex items-center gap-2 text-[11px]"
      >
        <div class="h-4 w-7 animate-pulse rounded border border-slate-200 bg-slate-100 dark:border-slate-700 dark:bg-slate-800" />
        <div class="h-2 w-[72px] animate-pulse rounded-full border border-slate-200 bg-slate-100 dark:border-slate-700 dark:bg-slate-800" />
        <div class="h-3 w-9 animate-pulse rounded bg-slate-100 dark:bg-slate-800" />
      </div>
    </div>

    <div v-else-if="presentation.state === 'error'" class="text-xs font-medium text-rose-600 dark:text-rose-200">
      {{ presentation.error || t('common.error') }}
    </div>

    <div v-else-if="displayRows.length > 0" class="space-y-1.5">
      <div
        v-for="row in displayRows"
        :key="row.key"
        :class="[
          'group flex min-w-0 items-center text-[11px]',
          compact ? 'gap-1.5' : 'gap-2'
        ]"
        :title="rowTitle(row)"
      >
        <span :class="['w-7 shrink-0 rounded border px-0.5 py-[1px] text-center text-[9px] font-bold', rowTagClass(row)]">
          {{ row.shortLabel }}
        </span>
        <div :class="[
          'h-2 shrink-0 overflow-hidden rounded-full border border-slate-200 bg-slate-100 dark:border-slate-700 dark:bg-slate-800/70',
          compact ? 'w-[56px]' : 'w-[72px]'
        ]">
          <div
            :class="['h-full rounded-full bg-gradient-to-r transition-all duration-700 ease-out', rowFillClass(row.usedPercent)]"
            :style="{ width: `${row.displayPercent}%` }"
          />
        </div>
        <span :class="[compact ? 'w-8' : 'w-9', 'shrink-0 text-right font-black', rowTextClass(row.usedPercent)]">
          {{ Math.round(row.displayPercent) }}%
        </span>
      </div>
    </div>

    <div v-else-if="presentation.state === 'unlimited'" class="text-xs font-semibold text-slate-500 dark:text-slate-300">
      {{ t('admin.accounts.gemini.rateLimit.unlimited') }}
    </div>

    <div v-else class="text-xs text-slate-400">-</div>

    <p
      v-if="presentation.meta.snapshotUpdatedAtText"
      class="text-[9px] font-medium leading-tight text-slate-500 dark:text-slate-300"
      :title="presentation.meta.snapshotUpdatedAtTooltip || undefined"
    >
      {{ t('admin.accounts.usageWindow.snapshotUpdatedAt', { time: presentation.meta.snapshotUpdatedAtText }) }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account, AccountVisualStyle, WindowStats } from '@/types'
import { useAccountUsagePresentation } from '@/composables/useAccountUsagePresentation'
import { useAccountUsageDisplayMode } from '@/composables/useAccountUsageDisplayMode'
import { useViewportAutoLoadGate } from '@/composables/useViewportAutoLoadGate'
import {
  createVisualUsageRows,
  rowFillClass,
  rowTagClass,
  rowTextClass,
  type VisualUsageRow,
} from './accountUsageVisualRows'

const props = withDefaults(defineProps<{
  account: Account
  todayStats?: WindowStats | null
  todayStatsLoading?: boolean
  manualRefreshToken?: number
  visualStyle?: AccountVisualStyle
  whiteSurfaceEnabled?: boolean
  compact?: boolean
}>(), {
  todayStats: null,
  todayStatsLoading: false,
  manualRefreshToken: 0,
  visualStyle: 'airy',
  whiteSurfaceEnabled: false,
  compact: false
})

const { t } = useI18n()
const { rootRef, autoLoadEnabled } = useViewportAutoLoadGate()
const { accountUsageDisplayMode } = useAccountUsageDisplayMode()
const { presentation, requestAutoLoad, shouldFetchUsage } =
  useAccountUsagePresentation(() => props.account, {
    autoLoadEnabled,
  })

const skeletonRows = computed(() =>
  Array.from(
    { length: Math.min(Math.max(presentation.value.meta.loadingRows, 2), 4) },
    (_, index) => index + 1,
  )
)

const displayRows = computed<VisualUsageRow[]>(() => {
  if (presentation.value.state !== 'bars') return []
  return createVisualUsageRows(
    presentation.value.windowRows,
    accountUsageDisplayMode.value,
  )
})

const rowTitle = (row: VisualUsageRow) => {
  const mode = accountUsageDisplayMode.value === 'remaining'
    ? t('admin.accounts.usageWindow.displayMode.remaining')
    : t('admin.accounts.usageWindow.displayMode.used')
  return `${row.label} ${mode}: ${Math.round(row.displayPercent)}%`
}

const tierBadges = computed(() => {
  const badges: Array<{ label: string; className: string }> = []
  const meta = presentation.value.meta
  if (meta.antigravityTierLabel) {
    badges.push({ label: meta.antigravityTierLabel, className: meta.antigravityTierClass || '' })
  }
  if (meta.protocolGatewayBadgeLabel) {
    badges.push({ label: meta.protocolGatewayBadgeLabel, className: meta.protocolGatewayBadgeClass || '' })
  }
  if (meta.geminiAuthTypeLabel) {
    badges.push({ label: meta.geminiAuthTypeLabel, className: meta.geminiTierClass || '' })
  }
  if (meta.sampledBadgeLabel) {
    badges.push({
      label: meta.sampledBadgeLabel,
      className: 'border border-amber-200/75 bg-amber-50 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-100'
    })
  }
  return badges
})

watch(
  () => props.manualRefreshToken,
  (nextToken, prevToken) => {
    if (nextToken === prevToken) return
    if (!shouldFetchUsage.value) return
    requestAutoLoad()
  },
)
</script>
