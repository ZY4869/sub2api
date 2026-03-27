<template>
  <div class="overflow-x-auto">
    <div class="inline-flex min-w-full items-center gap-1 border-b border-gray-200 pb-0 dark:border-dark-700">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        type="button"
        class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-t-xl border-b-2 px-3 py-1.5 text-sm font-medium transition-colors"
        :class="modelValue === tab.value ? 'border-primary-600 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
        @click="emit('update:modelValue', tab.value)"
      >
        <LobeStaticIcon
          v-if="tab.iconSources"
          :sources="tab.iconSources"
          :badge-text="tab.badgeText || ''"
          size="16px"
          variant="platform"
        />
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
import { buildLobeIconSources } from '@/utils/lobeIconResolver'
import LobeStaticIcon from '@/components/common/LobeStaticIcon.vue'
import type { AccountPlatform } from '@/types'

const PLATFORM_ICON_MAP: Record<string, { slug: string; badge: string }> = {
  anthropic: { slug: 'anthropic', badge: 'An' },
  kiro: { slug: 'kiro', badge: 'Ki' },
  openai: { slug: 'openai', badge: 'OA' },
  copilot: { slug: 'githubcopilot', badge: 'GH' },
  grok: { slug: 'xai', badge: 'Gr' },
  protocol_gateway: { slug: 'openrouter', badge: 'PG' },
  gemini: { slug: 'google', badge: 'Ge' },
  antigravity: { slug: 'antigravity', badge: 'AG' },
  sora: { slug: 'sora', badge: 'So' }
}

const props = defineProps<{
  modelValue: string
  platformCounts?: Partial<Record<AccountPlatform, number>>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()

const tabs = computed(() => [
  { value: '', countKey: undefined, label: t('admin.accounts.platformTabs.all'), iconSources: null, badgeText: null },
  ...Object.entries(PLATFORM_ICON_MAP).map(([value, icon]) => ({
    value,
    countKey: value as AccountPlatform,
    label: t(`admin.accounts.platforms.${value}`),
    iconSources: buildLobeIconSources([icon.slug]),
    badgeText: icon.badge
  }))
])

const resolveCount = (platform: AccountPlatform) => props.platformCounts?.[platform] ?? 0

void props
</script>
