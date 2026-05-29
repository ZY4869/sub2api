<template>
  <div class="flex flex-col items-center">
    <Icon
      name="inbox"
      size="xl"
      class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500"
    />
    <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
      {{ t('empty.noData') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
</script>
