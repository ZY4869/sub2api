<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { AccountPlatform } from '@/types'

const platform = defineModel<AccountPlatform>('platform', { required: true })

const { t } = useI18n()

const platformOptions: Array<{
  value: AccountPlatform
  label: string
  activeClass: string
}> = [
  { value: 'anthropic', label: 'Anthropic', activeClass: 'text-orange-600 dark:text-orange-400' },
  { value: 'openai', label: 'OpenAI', activeClass: 'text-green-600 dark:text-green-400' },
  { value: 'sora', label: 'Sora', activeClass: 'text-rose-600 dark:text-rose-400' },
  { value: 'gemini', label: 'Gemini', activeClass: 'text-blue-600 dark:text-blue-400' },
  { value: 'antigravity', label: 'Antigravity', activeClass: 'text-purple-600 dark:text-purple-400' }
]
</script>

<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.platform') }}</label>
    <div class="mt-2 flex rounded-lg bg-gray-100 p-1 dark:bg-dark-700" data-tour="account-form-platform">
      <button
        v-for="option in platformOptions"
        :key="option.value"
        type="button"
        @click="platform = option.value"
        :class="[
          'flex flex-1 items-center justify-center gap-2 rounded-md px-4 py-2.5 text-sm font-medium transition-all',
          platform === option.value
            ? `bg-white shadow-sm dark:bg-dark-600 ${option.activeClass}`
            : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
        ]"
      >
        <PlatformIcon :platform="option.value" size="md" />
        {{ option.label }}
      </button>
    </div>
  </div>
</template>
