<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { AccountPlatform } from '@/types'
import { ACCOUNT_PLATFORM_ORDER } from '@/utils/platformBranding'

const platform = defineModel<AccountPlatform>('platform', { required: true })

const { t } = useI18n()

const PLATFORM_ACTIVE_CLASSES: Record<AccountPlatform, string> = {
  anthropic: 'text-orange-600 dark:text-orange-400',
  antigravity: 'text-purple-600 dark:text-purple-400',
  baidu_document_ai: 'text-rose-600 dark:text-rose-400',
  deepseek: 'text-indigo-600 dark:text-indigo-400',
  gemini: 'text-blue-600 dark:text-blue-400',
  grok: 'text-slate-700 dark:text-slate-200',
  kiro: 'text-orange-600 dark:text-orange-400',
  openai: 'text-green-600 dark:text-green-400',
  protocol_gateway: 'text-slate-600 dark:text-slate-300'
}

const platformOptions = computed<
  Array<{
    value: AccountPlatform
    label: string
    activeClass: string
  }>
>(() =>
  ACCOUNT_PLATFORM_ORDER.map((value) => ({
    value,
    label: t(`admin.accounts.platforms.${value}`),
    activeClass: PLATFORM_ACTIVE_CLASSES[value]
  }))
)
</script>

<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.platform') }}</label>
    <div
      class="mt-2 grid grid-cols-2 gap-2 rounded-lg bg-gray-100 p-2 dark:bg-dark-700 md:grid-cols-3 xl:grid-cols-4"
      data-tour="account-form-platform"
    >
      <button
        v-for="option in platformOptions"
        :key="option.value"
        type="button"
        :data-testid="`select-${String(option.value).replace(/_/g, '-')}`"
        @click="platform = option.value"
        :class="[
          'flex min-w-0 w-full items-center justify-center gap-2 rounded-md px-3 py-3 text-center text-sm font-medium leading-snug transition-all whitespace-normal',
          platform === option.value
            ? `bg-white shadow-sm dark:bg-dark-600 ${option.activeClass}`
            : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
        ]"
      >
        <PlatformIcon :platform="option.value" size="md" class="shrink-0" />
        <span class="min-w-0 break-words">{{ option.label }}</span>
      </button>
    </div>
  </div>
</template>
