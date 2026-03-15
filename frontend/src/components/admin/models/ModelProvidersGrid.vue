<template>
  <div class="grid gap-4 p-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
    <button
      v-for="group in providers"
      :key="group.provider"
      type="button"
      class="group rounded-2xl border border-gray-200 bg-white p-4 text-left shadow-sm transition hover:border-primary-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-500/50"
      @click="emit('open', group.provider)"
    >
      <div class="flex items-start gap-3">
        <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-700">
          <ModelPlatformIcon :platform="group.provider" size="lg" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">
            {{ group.label }}
          </div>
          <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <span class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 dark:bg-dark-700">
              {{ group.totalCount }} {{ t('admin.models.registry.columns.model') }}
            </span>
            <span class="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">
              {{ group.availableCount }}/{{ group.totalCount }} {{ t('admin.models.registry.availableStatus') }}
            </span>
          </div>
        </div>
      </div>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AdminModelRegistryProviderGroup } from '@/composables/useAdminModelRegistryProviders'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'

defineProps<{
  providers: AdminModelRegistryProviderGroup[]
}>()

const emit = defineEmits<{
  (e: 'open', provider: string): void
}>()

const { t } = useI18n()
</script>

