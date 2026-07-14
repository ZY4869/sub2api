<template>
  <div class="inline-flex max-w-full w-max flex-col items-start gap-1">
    <div class="inline-flex max-w-full w-max flex-nowrap items-center gap-1 whitespace-nowrap" data-testid="account-usage-reset-credits-row">
      <button
        v-if="showRefresh"
        type="button"
        class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-gray-200 text-gray-500 transition hover:border-primary-300 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300"
        :disabled="refreshDisabled"
        :title="t('admin.accounts.usageWindow.refreshResetCreditsTitle')"
        :aria-label="t('admin.accounts.usageWindow.refreshResetCreditsTitle')"
        data-testid="account-usage-reset-credits-refresh"
        @click="emit('refresh')"
      >
        <Icon
          name="refresh"
          size="xs"
          :class="refreshing ? 'animate-spin' : ''"
        />
      </button>

      <span
        v-if="showRemaining"
        :class="[
          'inline-flex shrink-0 items-center rounded-full border px-1.5 py-1 text-[10px] font-semibold leading-none',
          resetUnsupported || resetUnknown
            ? 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'
            : resetZero
              ? 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-400/25 dark:bg-rose-400/10 dark:text-rose-100'
            : resetCreditsLow
              ? 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-400/25 dark:bg-amber-400/10 dark:text-amber-100'
            : 'border-teal-200 bg-teal-50 text-teal-700 dark:border-teal-400/25 dark:bg-teal-400/10 dark:text-teal-100'
        ]"
        data-testid="account-usage-reset-quota-remaining"
      >
        {{ resetStatusLabel }}
      </span>

      <button
        v-if="showReset"
        type="button"
        class="inline-flex shrink-0 items-center gap-1 rounded-md border border-gray-200 px-1.5 py-1 text-[10px] font-medium text-gray-600 transition hover:border-primary-300 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300"
        :disabled="resetDisabled"
        data-testid="account-usage-reset-quota-button"
        @click="emit('reset')"
      >
        <Icon
          name="refresh"
          size="xs"
          :class="resetting ? 'animate-spin' : ''"
        />
        {{
          resetting
            ? t('admin.accounts.usageWindow.resettingQuota')
            : t('admin.accounts.usageWindow.resetQuota')
        }}
      </button>

      <button
        v-if="showExpiry && earliestExpiryText"
        type="button"
        class="inline-flex shrink-0 items-center gap-1 rounded-full border border-sky-200 bg-sky-50 px-1.5 py-1 text-[10px] font-semibold leading-none text-sky-700 transition hover:border-sky-300 hover:text-sky-800 disabled:cursor-default dark:border-sky-400/25 dark:bg-sky-400/10 dark:text-sky-100"
        :disabled="expiryExtraCount <= 0"
        data-testid="account-usage-reset-quota-expiry"
        @click="emit('toggle-details')"
      >
        {{ t('admin.accounts.usageWindow.resetCreditExpiresAt', { time: earliestExpiryText }) }}
        <span v-if="expiryExtraCount > 0" data-testid="account-usage-reset-quota-expiry-extra">
          +{{ expiryExtraCount }}
        </span>
      </button>
    </div>

    <div
      v-if="detailsExpanded && detailRows.length > 0"
      class="max-w-full rounded-md border border-gray-200 bg-white p-2 text-[10px] leading-4 text-gray-600 shadow-sm dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
      data-testid="account-usage-reset-quota-expiry-details"
    >
      <div
        v-for="row in detailRows"
        :key="row.id || row.expiresAt"
        class="flex max-w-full items-center justify-between gap-3"
      >
        <span class="truncate">{{ row.status || t('admin.accounts.usageWindow.resetCreditDetail') }}</span>
        <span class="shrink-0 tabular-nums">{{ row.expiresAtText }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  resetStatusLabel: string
  resetUnsupported: boolean
  resetUnknown?: boolean
  resetZero?: boolean
  resetCreditsLow?: boolean
  resetting: boolean
  refreshing: boolean
  resetDisabled: boolean
  showRefresh?: boolean
  showReset?: boolean
  showRemaining?: boolean
  showExpiry?: boolean
  earliestExpiryText?: string
  expiryExtraCount?: number
  detailRows?: Array<{
    id: string
    status: string
    expiresAt: string
    expiresAtText: string
  }>
  detailsExpanded?: boolean
}>(), {
  resetZero: false,
  resetUnknown: false,
  resetCreditsLow: false,
  showRefresh: true,
  showReset: true,
  showRemaining: true,
  showExpiry: true,
  earliestExpiryText: '',
  expiryExtraCount: 0,
  detailRows: () => [],
  detailsExpanded: false,
})

const emit = defineEmits<{
  refresh: []
  reset: []
  'toggle-details': []
}>()

const { t } = useI18n()

const refreshDisabled = computed(() => props.refreshing || props.resetting)
</script>
