<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <button
      type="button"
      class="inline-flex items-center gap-1 rounded-md border border-gray-200 px-2 py-1 text-[10px] font-medium text-gray-600 transition hover:border-primary-300 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300"
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
      type="button"
      class="inline-flex items-center gap-1 rounded-md border border-gray-200 px-2 py-1 text-[10px] font-medium text-gray-600 transition hover:border-primary-300 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300"
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
      :class="[
        'inline-flex items-center rounded-full border px-2 py-1 text-[10px] font-semibold leading-none',
        resetUnsupported
          ? 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'
          : 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-400/25 dark:bg-amber-400/10 dark:text-amber-100'
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

const props = defineProps<{
  resetStatusLabel: string
  resetUnsupported: boolean
  resetting: boolean
  refreshing: boolean
  resetDisabled: boolean
}>()

const emit = defineEmits<{
  refresh: []
  reset: []
}>()

const { t } = useI18n()

const refreshDisabled = computed(() => props.refreshing || props.resetting)
</script>
