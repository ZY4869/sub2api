<template>
  <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-700/50 dark:bg-amber-950/30">
    <div class="flex items-start gap-3">
      <Icon name="exclamationTriangle" size="md" class="mt-0.5 shrink-0 text-amber-500" />
      <div class="space-y-1">
        <p class="text-sm font-semibold text-amber-900 dark:text-amber-100">
          {{ t('auth.maintenanceModeTitle') }}
        </p>
        <p class="text-sm text-amber-800 dark:text-amber-200">
          {{ t('auth.maintenanceModeMessage') }}
        </p>
        <p
          v-if="showAdminLoginHint"
          class="text-xs font-medium text-amber-700 dark:text-amber-300"
        >
          {{ t('auth.maintenanceModeAdminOnly') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

withDefaults(defineProps<{
  showAdminLoginHint?: boolean
}>(), {
  showAdminLoginHint: false
})

const { t } = useI18n()
</script>
