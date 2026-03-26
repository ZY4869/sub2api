<template>
  <div class="flex items-center gap-2">
    <div v-if="isRateLimited" class="flex flex-col items-center gap-1">
      <span class="badge text-xs badge-warning">{{ rateLimitStatusLabel }}</span>
      <span class="text-[11px] text-gray-400 dark:text-gray-500">{{ rateLimitResumeText }}</span>
    </div>

    <div v-else-if="isOverloaded" class="flex flex-col items-center gap-1">
      <span class="badge text-xs badge-danger">{{ t('admin.accounts.status.overloaded') }}</span>
      <span class="text-[11px] text-gray-400 dark:text-gray-500">{{ overloadCountdown }}</span>
    </div>

    <template v-else>
      <button
        v-if="isTempUnschedulable"
        type="button"
        :class="['badge text-xs', statusClass, 'cursor-pointer']"
        :title="t('admin.accounts.status.viewTempUnschedDetails')"
        @click="handleTempUnschedClick"
      >
        {{ statusText }}
      </button>
      <span v-else :class="['badge text-xs', statusClass]">
        {{ statusText }}
      </span>
    </template>

    <div v-if="hasError && account.error_message" class="relative">
      <button
        ref="errorTooltipTriggerRef"
        type="button"
        class="error-info-trigger inline-flex text-red-500 transition-colors hover:text-red-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-400/60 dark:text-red-400 dark:hover:text-red-300"
        :aria-label="t('admin.accounts.status.error')"
        @mouseenter="showErrorTooltip"
        @mouseleave="hideErrorTooltip"
        @focus="showErrorTooltip"
        @blur="hideErrorTooltip"
      >
        <Icon name="questionCircle" size="sm" :stroke-width="2" />
      </button>
      <Teleport to="body">
        <div
          v-if="errorTooltipVisible"
          ref="errorTooltipRef"
          class="error-info-tooltip pointer-events-none fixed z-[2100] rounded-lg bg-gray-800 px-3 py-2 text-xs text-white shadow-xl dark:bg-gray-900"
          :style="errorTooltipStyle"
          role="tooltip"
        >
          <div class="whitespace-pre-wrap break-words leading-relaxed text-gray-300">
            {{ account.error_message }}
          </div>
          <div
            class="absolute h-3 w-3 rotate-45 bg-gray-800 dark:bg-gray-900"
            :style="errorTooltipArrowStyle"
          ></div>
        </div>
      </Teleport>
    </div>

    <div v-if="isRateLimited" class="group relative">
      <span
        class="inline-flex items-center gap-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
      >
        <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
        {{ rateLimitBadgeText }}
      </span>
      <div
        class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-56 -translate-x-1/2 whitespace-normal rounded bg-gray-900 px-3 py-2 text-center text-xs leading-relaxed text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
      >
        {{ rateLimitTooltipText }}
        <div
          class="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-gray-700"
        ></div>
      </div>
    </div>

    <div
      v-if="activeModelStatuses.length > 0"
      :class="[
        activeModelStatuses.length <= 4
          ? 'flex flex-col gap-1'
          : activeModelStatuses.length <= 8
            ? 'columns-2 gap-x-2'
            : 'columns-3 gap-x-2'
      ]"
    >
      <div
        v-for="item in activeModelStatuses"
        :key="`${item.kind}-${item.model}`"
        class="group relative mb-1 break-inside-avoid"
      >
        <span
          v-if="item.kind === 'credits_exhausted'"
          class="inline-flex items-center gap-1 rounded bg-red-100 px-1.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400"
        >
          <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
          {{ t('admin.accounts.status.creditsExhausted') }}
          <span class="text-[10px] opacity-70">{{ formatModelResetTime(item.reset_at) }}</span>
        </span>
        <span
          v-else-if="item.kind === 'credits_active'"
          class="inline-flex items-center gap-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
        >
          <Icon name="sparkles" size="xs" :stroke-width="2" />
          {{ formatScopeName(item.model) }}
          <span class="text-[10px] opacity-70">{{ formatModelResetTime(item.reset_at) }}</span>
        </span>
        <span
          v-else
          class="inline-flex items-center gap-1 rounded bg-purple-100 px-1.5 py-0.5 text-xs font-medium text-purple-700 dark:bg-purple-900/30 dark:text-purple-400"
        >
          <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
          {{ formatScopeName(item.model) }}
          <span class="text-[10px] opacity-70">{{ formatModelResetTime(item.reset_at) }}</span>
        </span>
        <div
          class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-56 -translate-x-1/2 whitespace-normal rounded bg-gray-900 px-3 py-2 text-center text-xs leading-relaxed text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
        >
          {{
            item.kind === 'credits_exhausted'
              ? t('admin.accounts.status.creditsExhaustedUntil', { time: formatTime(item.reset_at) })
              : item.kind === 'credits_active'
                ? t('admin.accounts.status.modelCreditOveragesUntil', { model: formatScopeName(item.model), time: formatTime(item.reset_at) })
                : t('admin.accounts.status.modelRateLimitedUntil', { model: formatScopeName(item.model), time: formatTime(item.reset_at) })
          }}
          <div
            class="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-gray-700"
          ></div>
        </div>
      </div>
    </div>

    <div v-if="isOverloaded" class="group relative">
      <span
        class="inline-flex items-center gap-1 rounded bg-red-100 px-1.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400"
      >
        <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
        529
      </span>
      <div
        class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-56 -translate-x-1/2 whitespace-normal rounded bg-gray-900 px-3 py-2 text-center text-xs leading-relaxed text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
      >
        {{ t('admin.accounts.status.overloadedUntil', { time: formatTime(account.overload_until) }) }}
        <div
          class="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-gray-700"
        ></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useUiNow } from '@/composables/useUiNow'
import type { Account } from '@/types'
import { formatCountdown, formatCountdownWithSuffix, formatDateTime, formatTime } from '@/utils/format'

const { t } = useI18n()
const { nowMs, nowDate } = useUiNow()

const props = defineProps<{
  account: Account
}>()

const emit = defineEmits<{
  (e: 'show-temp-unsched', account: Account): void
}>()

type AccountModelStatusItem = {
  kind: 'rate_limit' | 'credits_exhausted' | 'credits_active'
  model: string
  reset_at: string
}

const errorTooltipVisible = ref(false)
const errorTooltipTriggerRef = ref<HTMLElement | null>(null)
const errorTooltipRef = ref<HTMLElement | null>(null)
const errorTooltipStyle = ref<Record<string, string>>({})
const errorTooltipArrowStyle = ref<Record<string, string>>({})

const isRateLimited = computed(() => {
  if (!props.account.rate_limit_reset_at) return false
  const resetAtMs = new Date(props.account.rate_limit_reset_at).getTime()
  return !Number.isNaN(resetAtMs) && resetAtMs > nowMs.value
})

const activeModelStatuses = computed<AccountModelStatusItem[]>(() => {
  const extra = props.account.extra as Record<string, unknown> | undefined
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

const formatModelResetTime = (resetAt: string): string => {
  const date = new Date(resetAt)
  const diffMs = date.getTime() - nowMs.value
  if (diffMs <= 0) return ''
  const totalSecs = Math.floor(diffMs / 1000)
  const hours = Math.floor(totalSecs / 3600)
  const minutes = Math.floor((totalSecs % 3600) / 60)
  const seconds = totalSecs % 60
  if (hours > 0) return `${hours}h${minutes}m`
  if (minutes > 0) return `${minutes}m${seconds}s`
  return `${seconds}s`
}

const isOverloaded = computed(() => {
  if (!props.account.overload_until) return false
  const untilMs = new Date(props.account.overload_until).getTime()
  return !Number.isNaN(untilMs) && untilMs > nowMs.value
})

const isTempUnschedulable = computed(() => {
  if (!props.account.temp_unschedulable_until) return false
  const untilMs = new Date(props.account.temp_unschedulable_until).getTime()
  return !Number.isNaN(untilMs) && untilMs > nowMs.value
})

const hasError = computed(() => props.account.status === 'error')

const rateLimitCountdown = computed(() => {
  if (!props.account.rate_limit_reset_at) return null
  void nowMs.value
  return formatCountdown(props.account.rate_limit_reset_at)
})

const rateLimitResumeText = computed(() => {
  if (!rateLimitCountdown.value) return ''
  switch (props.account.rate_limit_reason) {
    case 'usage_5h':
      return t('admin.accounts.status.usage5hAutoResume', { time: rateLimitCountdown.value })
    case 'usage_7d':
      return t('admin.accounts.status.usage7dAutoResume', { time: rateLimitCountdown.value })
    default:
      return t('admin.accounts.status.rateLimitedAutoResume', { time: rateLimitCountdown.value })
  }
})

const rateLimitStatusLabel = computed(() => {
  switch (props.account.rate_limit_reason) {
    case 'usage_5h':
      return t('admin.accounts.status.usage5h')
    case 'usage_7d':
      return t('admin.accounts.status.usage7d')
    default:
      return t('admin.accounts.status.rateLimited')
  }
})

const rateLimitBadgeText = computed(() => {
  switch (props.account.rate_limit_reason) {
    case 'usage_5h':
      return '5h'
    case 'usage_7d':
      return '7d'
    default:
      return '429'
  }
})

const rateLimitTooltipText = computed(() => {
  const time = formatDateTime(props.account.rate_limit_reset_at)
  switch (props.account.rate_limit_reason) {
    case 'usage_5h':
      return t('admin.accounts.status.usage5hUntil', { time })
    case 'usage_7d':
      return t('admin.accounts.status.usage7dUntil', { time })
    default:
      return t('admin.accounts.status.rateLimitedUntil', { time })
  }
})

const overloadCountdown = computed(() => {
  if (!props.account.overload_until) return null
  void nowMs.value
  return formatCountdownWithSuffix(props.account.overload_until)
})

const statusClass = computed(() => {
  if (hasError.value) return 'badge-danger'
  if (isTempUnschedulable.value) return 'badge-warning'
  if (!props.account.schedulable) return 'badge-gray'

  switch (props.account.status) {
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
  if (!props.account.schedulable) return t('admin.accounts.status.paused')
  return t(`admin.accounts.status.${props.account.status}`)
})

const handleTempUnschedClick = () => {
  if (!isTempUnschedulable.value) return
  emit('show-temp-unsched', props.account)
}

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
  if (!hasError.value || !props.account.error_message) return

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
</script>
