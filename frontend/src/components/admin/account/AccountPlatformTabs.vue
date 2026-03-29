<template>
  <div class="overflow-x-auto">
    <div class="inline-flex min-w-full items-center gap-1 border-b border-gray-200 pb-0 dark:border-dark-700">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        type="button"
        :data-tab-value="tab.dataValue"
        class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-t-xl border-b-2 px-3 py-1.5 text-sm font-medium transition-colors"
        :class="modelValue === tab.value ? 'border-primary-600 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
        @click="emit('update:modelValue', tab.value)"
      >
        <PlatformIcon v-if="tab.platform" :platform="tab.platform" size="sm" />
        <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
        </svg>
        <span>{{ tab.label }}</span>
        <span
          v-if="tab.countKey"
          class="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-200"
        >
          {{ resolveCount(tab.countKey) }}
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountPlatform } from '@/types'
import { ACCOUNT_PLATFORM_ORDER } from '@/utils/platformBranding'
import PlatformIcon from '@/components/common/PlatformIcon.vue'

const props = defineProps<{
  modelValue: string
  platformCounts?: Partial<Record<AccountPlatform, number>>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()

const resolveCount = (platform: AccountPlatform) => props.platformCounts?.[platform] ?? 0

const tabs = computed(() => [
  {
    value: '',
    dataValue: 'all',
    countKey: undefined,
    label: t('admin.accounts.platformTabs.all'),
    platform: null
  },
  ...ACCOUNT_PLATFORM_ORDER.map((platform) => ({
    value: platform,
    dataValue: platform,
    countKey: platform,
    label: t(`admin.accounts.platforms.${platform}`),
    platform
  }))
])

void props
</script>
