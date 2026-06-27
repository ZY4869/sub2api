<template>
  <div class="inline-flex max-w-full w-max flex-nowrap items-center gap-1 whitespace-nowrap">
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
}>(), {
  resetZero: false,
  resetUnknown: false,
  resetCreditsLow: false,
  showRefresh: true,
  showReset: true,
  showRemaining: true,
})

const emit = defineEmits<{
  refresh: []
  reset: []
}>()

const { t } = useI18n()

const refreshDisabled = computed(() => props.refreshing || props.resetting)
</script>
