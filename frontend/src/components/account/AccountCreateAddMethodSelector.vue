<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AddMethod } from '@/composables/useAccountOAuth'

const addMethod = defineModel<AddMethod>('addMethod', { required: true })

const { t } = useI18n()
</script>

<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.addMethod') }}</label>
    <div class="mt-2 flex gap-4">
      <label class="flex cursor-pointer items-center">
        <input
          v-model="addMethod"
          type="radio"
          value="oauth"
          class="mr-2 text-primary-600 focus:ring-primary-500"
        />
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.types.oauth') }}</span>
      </label>
      <label class="flex cursor-pointer items-center">
        <input
          v-model="addMethod"
          type="radio"
          value="setup-token"
          class="mr-2 text-primary-600 focus:ring-primary-500"
        />
        <span class="text-sm text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.setupTokenLongLived') }}
        </span>
      </label>
    </div>
  </div>
</template>
