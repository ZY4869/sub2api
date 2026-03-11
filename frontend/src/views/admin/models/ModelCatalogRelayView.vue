<template>
  <section class="rounded-2xl border border-dashed border-gray-300 bg-white p-8 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="max-w-2xl space-y-4">
      <span class="inline-flex rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
        {{ t('admin.models.pages.relay.badge') }}
      </span>
      <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
        {{ t('admin.models.pages.relay.title') }}
      </h2>
      <p class="text-sm leading-6 text-gray-600 dark:text-gray-300">
        {{ t('admin.models.pages.relay.description') }}
      </p>
      <div class="rounded-2xl bg-gray-50 p-4 text-sm text-gray-500 dark:bg-dark-900 dark:text-gray-400">
        {{ t('admin.models.pages.relay.placeholder') }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
