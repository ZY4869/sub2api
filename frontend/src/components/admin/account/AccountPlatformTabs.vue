<template>
  <div class="overflow-x-auto">
    <div class="inline-flex min-w-full items-center gap-2 border-b border-gray-200 pb-1 dark:border-dark-700">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        type="button"
        class="inline-flex items-center gap-2 whitespace-nowrap rounded-t-xl border-b-2 px-4 py-2 text-sm font-medium transition-colors"
        :class="modelValue === tab.value ? 'border-primary-600 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
        @click="emit('update:modelValue', tab.value)"
      >
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

const props = defineProps<{
  modelValue: string
  platformCounts?: Partial<Record<AccountPlatform, number>>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()

const tabs = computed(() => [
  { value: '', countKey: undefined, label: t('admin.accounts.platformTabs.all') },
  { value: 'anthropic', countKey: 'anthropic' as const, label: t('admin.accounts.platforms.anthropic') },
  { value: 'kiro', countKey: 'kiro' as const, label: t('admin.accounts.platforms.kiro') },
  { value: 'openai', countKey: 'openai' as const, label: t('admin.accounts.platforms.openai') },
  { value: 'copilot', countKey: 'copilot' as const, label: t('admin.accounts.platforms.copilot') },
  { value: 'gemini', countKey: 'gemini' as const, label: t('admin.accounts.platforms.gemini') },
  { value: 'antigravity', countKey: 'antigravity' as const, label: t('admin.accounts.platforms.antigravity') },
  { value: 'sora', countKey: 'sora' as const, label: t('admin.accounts.platforms.sora') }
])

const resolveCount = (platform: AccountPlatform) => props.platformCounts?.[platform] ?? 0

void props
</script>
