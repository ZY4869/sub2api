<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const enabled = defineModel<boolean>('enabled', { required: true })
const baseUrl = defineModel<string>('baseUrl', { required: true })

const { t } = useI18n()
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <label
        id="bulk-edit-base-url-label"
        class="input-label mb-0"
        for="bulk-edit-base-url-enabled"
      >
        {{ t('admin.accounts.baseUrl') }}
      </label>
      <input
        v-model="enabled"
        id="bulk-edit-base-url-enabled"
        type="checkbox"
        aria-controls="bulk-edit-base-url"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>
    <input
      v-model="baseUrl"
      id="bulk-edit-base-url"
      type="text"
      :disabled="!enabled"
      class="input"
      :class="!enabled && 'cursor-not-allowed opacity-50'"
      :placeholder="t('admin.accounts.bulkEdit.baseUrlPlaceholder')"
      aria-labelledby="bulk-edit-base-url-label"
    />
    <p class="input-hint">
      {{ t('admin.accounts.bulkEdit.baseUrlNotice') }}
    </p>
  </div>
</template>

