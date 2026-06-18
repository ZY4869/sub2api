<template>
  <div v-if="presentation.state === 'loading'" class="space-y-1.5">
    <div v-for="index in presentation.meta.loadingRows" :key="index" class="flex items-center gap-2">
      <div class="h-3 w-[32px] animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    </div>
  </div>

  <div v-else-if="presentation.state === 'error'" class="text-xs text-red-500">
    {{ presentation.error }}
  </div>

  <div v-else-if="presentation.resetRows.length > 0" class="max-w-full space-y-1.5 overflow-visible">
    <div
      v-for="row in presentation.resetRows"
      :key="row.key"
      class="grid min-w-0 max-w-full grid-cols-[minmax(42px,auto)_1fr] items-start gap-x-2 gap-y-1 text-[10px] tabular-nums"
    >
      <span
        :title="row.label"
        data-testid="account-usage-reset-window-label"
        :class="[
          'min-w-[42px] max-w-[82px] shrink-0 rounded-full border px-1.5 py-1 text-center font-bold leading-none',
          resetLabelClass(row.label)
        ]"
      >
        {{ row.label }}
      </span>

      <span
        v-if="formatResetValue(row.resetsAt, row.remainingSeconds)"
        class="flex min-w-0 flex-1 flex-wrap items-center gap-x-1.5 gap-y-1 overflow-visible text-gray-700 dark:text-gray-300"
        :title="formatResetValue(row.resetsAt, row.remainingSeconds)?.tooltip || undefined"
      >
        <Icon
          name="clock"
          size="xs"
          class="shrink-0 text-gray-400 dark:text-gray-500"
        />
        <span
          class="shrink-0 whitespace-nowrap rounded-full bg-gray-100 px-1.5 py-0.5 font-medium leading-none text-gray-700 dark:bg-gray-700 dark:text-gray-200"
        >
          {{ formatResetValue(row.resetsAt, row.remainingSeconds)?.countdown }}
        </span>
        <span class="shrink-0 text-gray-400 dark:text-gray-500">·</span>
        <span class="min-w-[112px] flex-1 whitespace-normal break-words font-semibold leading-snug text-gray-700 dark:text-gray-200">
          {{ formatResetValue(row.resetsAt, row.remainingSeconds)?.absolute }}
        </span>
      </span>

      <span v-else class="text-gray-400 dark:text-gray-500">-</span>
    </div>
    <div
      v-if="canResetOpenAIQuota"
      class="flex flex-wrap items-center gap-1.5"
    >
      <button
        type="button"
        class="inline-flex items-center gap-1 rounded-md border border-gray-200 px-2 py-1 text-[10px] font-medium text-gray-600 transition hover:border-primary-300 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300"
        :disabled="resetButtonDisabled"
        @click="resetOpenAIQuota"
      >
        <Icon
          name="refresh"
          size="xs"
          :class="resetting ? 'animate-spin' : ''"
        />
        {{ resetting ? t('admin.accounts.usageWindow.resettingQuota') : t('admin.accounts.usageWindow.resetQuota') }}
      </button>
      <span
        v-if="canResetOpenAIQuota"
        :class="[
          'inline-flex items-center rounded-full border px-2 py-1 text-[10px] font-semibold leading-none',
          resetCreditsUnsupported
            ? 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'
            : 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-400/25 dark:bg-amber-400/10 dark:text-amber-100'
        ]"
        data-testid="account-usage-reset-quota-remaining"
      >
        {{ resetCreditsStatusLabel }}
      </span>
    </div>
  </div>

  <div v-else class="text-xs text-gray-400">-</div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
  useAccountUsagePresentation,
} from '@/composables/useAccountUsagePresentation'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt,
} from '@/utils/usageResetTime'
import type { Account } from '@/types'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { resolveUsageWindowCapsuleClass } from '@/utils/accountUsageWindowDisplay'
import { adminAPI } from '@/api'
import { useAppStore } from '@/stores'
import { getRuntimePlatform } from '@/composables/accountUsagePresentation/support'

const props = defineProps<{
  account: Account
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { nowDate } = useRealtimeCountdownNow('accounts')
const { presentation } = useAccountUsagePresentation(() => props.account)
const resetting = ref(false)

const canResetOpenAIQuota = computed(() => {
  return getRuntimePlatform(props.account) === 'openai' && props.account.type === 'oauth'
})

const openAIQuotaResetRemaining = computed(() => {
  return presentation.value.meta.openAIResetCreditsAvailableCount ?? null
})

const resetCreditsUnsupported = computed(() => {
  return presentation.value.meta.openAIResetCreditsStatus === 'unsupported'
})

const resetButtonDisabled = computed(() => {
  return resetting.value || resetCreditsUnsupported.value
})

const resetCreditsStatusLabel = computed(() => {
  if (resetCreditsUnsupported.value) {
    return (
      presentation.value.meta.openAIResetCreditsUnsupportedReason ||
      t('admin.accounts.usageWindow.resetQuotaUnsupported')
    )
  }
  return t('admin.accounts.usageWindow.resetQuotaRemaining', {
    count: formatOpenAIQuotaResetRemaining(openAIQuotaResetRemaining.value),
  })
})

function formatOpenAIQuotaResetRemaining(value: number | null): string {
  if (value === null) return '--'
  const normalized = Number.isFinite(Number(value)) ? Math.max(0, Math.floor(Number(value))) : null
  if (normalized === null) return '--'
  return String(normalized).padStart(2, '0')
}

async function resetOpenAIQuota() {
  if (!canResetOpenAIQuota.value || resetButtonDisabled.value) return
  if (!window.confirm(t('admin.accounts.usageWindow.resetQuotaConfirm'))) return
  resetting.value = true
  try {
    await adminAPI.accounts.resetAccountQuota(props.account.id)
    invalidateAccountUsagePresentationCache([props.account.id])
    await refreshAccountUsagePresentation([props.account], { force: true, source: 'active' })
    appStore.showSuccess(t('admin.accounts.usageWindow.resetQuotaSuccess'))
  } catch (error: any) {
    invalidateAccountUsagePresentationCache([props.account.id])
    await refreshAccountUsagePresentation([props.account], { force: true, source: 'active' }).catch(() => {})
    appStore.showError(resolveResetQuotaErrorMessage(error))
  } finally {
    resetting.value = false
  }
}

function resolveResetQuotaErrorMessage(error: any): string {
  const responseData = error?.response?.data ?? {}
  const reason = String(
    responseData.reason ||
      responseData.error ||
      responseData.code ||
      responseData.error_code ||
      '',
  )

  if (reason === 'OPENAI_RESET_CREDITS_NO_CREDIT') {
    return t('admin.accounts.usageWindow.resetQuotaNoCredit')
  }
  if (reason === 'OPENAI_RESET_CREDITS_NOTHING_TO_RESET') {
    return t('admin.accounts.usageWindow.resetQuotaNothingToReset')
  }

  return (
    responseData.detail ||
    responseData.message ||
    error?.message ||
    t('admin.accounts.usageWindow.resetQuotaFailed')
  )
}

function formatResetValue(
  resetsAt: string | null,
  remainingSeconds?: number | null,
): {
  countdown: string
  absolute: string
  tooltip: string
} | null {
  const effectiveResetAt = parseEffectiveResetAt(
    resetsAt,
    remainingSeconds ?? null,
    nowDate.value,
  )
  if (!effectiveResetAt) return null

  return {
    countdown: formatResetCountdown(
      effectiveResetAt,
      nowDate.value,
      t('admin.accounts.usageWindow.now'),
    ),
    absolute: formatLocalAbsoluteTime(effectiveResetAt, nowDate.value, {
      today: t('dates.today'),
      tomorrow: t('dates.tomorrow'),
    }),
    tooltip: formatLocalTimestamp(effectiveResetAt),
  }
}

function resetLabelClass(label: string): string {
  return resolveUsageWindowCapsuleClass(label)
}
</script>
