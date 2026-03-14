<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { AccountPoolModeState } from '@/utils/accountFormShared'

defineProps<{
  defaultRetryCount: number
  maxRetryCount: number
}>()

const state = defineModel<AccountPoolModeState>('state', { required: true })

const { t } = useI18n()
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <div>
        <label class="input-label mb-0">{{ t('admin.accounts.poolMode') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.poolModeHint') }}
        </p>
      </div>
      <button
        type="button"
        @click="state.enabled = !state.enabled"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          state.enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            state.enabled ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>

    <div v-if="state.enabled" class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
      <p class="text-xs text-blue-700 dark:text-blue-400">
        <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
        {{ t('admin.accounts.poolModeInfo') }}
      </p>
    </div>

    <div v-if="state.enabled" class="mt-3">
      <label class="input-label">{{ t('admin.accounts.poolModeRetryCount') }}</label>
      <input
        v-model.number="state.retryCount"
        type="number"
        min="0"
        :max="maxRetryCount"
        step="1"
        class="input"
      />
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{
          t('admin.accounts.poolModeRetryCountHint', {
            default: defaultRetryCount,
            max: maxRetryCount
          })
        }}
      </p>
    </div>
  </div>
</template>
