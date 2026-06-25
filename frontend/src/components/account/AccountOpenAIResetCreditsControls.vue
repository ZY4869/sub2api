<template>
  <div class="inline-flex max-w-full w-max flex-nowrap items-center gap-1 whitespace-nowrap">
    <button
      v-if="showRefresh"
      type="button"
      class="inline-flex shrink-0 items-center gap-1 rounded-md border border-gray-200 px-1.5 py-1 text-[10px] font-medium text-gray-600 transition hover:border-primary-300 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300"
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
      {{
        refreshing
          ? t('admin.accounts.usageWindow.refreshingResetCredits')
          : t('admin.accounts.usageWindow.refreshResetCredits')
      }}
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
            ? 'border-orange-200 bg-orange-50 text-orange-700 dark:border-orange-400/25 dark:bg-orange-400/10 dark:text-orange-100'
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
  resetting: boolean
  refreshing: boolean
  resetDisabled: boolean
  showRefresh?: boolean
  showReset?: boolean
  showRemaining?: boolean
}>(), {
  resetZero: false,
  resetUnknown: false,
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
