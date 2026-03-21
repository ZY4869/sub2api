<template>
  <div class="space-y-3 p-4">
    <div
      v-for="group in providers"
      :key="group.provider"
      class="rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
    >
      <button
        type="button"
        class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left"
        @click="toggle(group.provider)"
      >
        <span class="flex min-w-0 items-center gap-3">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-700">
            <ModelPlatformIcon :platform="group.provider" size="lg" />
          </span>
          <span class="min-w-0">
            <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">
              {{ group.label }}
            </span>
            <span class="mt-1 block truncate text-xs text-gray-500 dark:text-gray-400">
              {{ group.availableCount }}/{{ group.totalCount }} {{ t('admin.models.registry.availableStatus') }}
            </span>
          </span>
        </span>

        <Icon
          name="chevronDown"
          size="md"
          :class="['transition-transform', expandedProviders.has(group.provider) && 'rotate-180']"
        />
      </button>

      <div v-if="expandedProviders.has(group.provider)" class="border-t border-gray-100 px-4 py-4 dark:border-dark-700">
        <ModelProviderModelsPanel
          :models="getModels(group.provider)"
          :total-count="group.totalCount"
          :available-count="group.availableCount"
          :loading="isProviderLoading(group.provider)"
          :has-more="providerHasMoreModels(group.provider)"
          :is-activating="isActivating"
          @activate="emit('activate', $event)"
          @load-more="emit('load-more', group.provider)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminModelRegistryProviderGroup } from '@/composables/useAdminModelRegistryProviders'
import Icon from '@/components/icons/Icon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import ModelProviderModelsPanel from '@/components/admin/models/ModelProviderModelsPanel.vue'

defineProps<{
  providers: AdminModelRegistryProviderGroup[]
  getModels: (provider: string) => import('@/api/admin/modelRegistry').ModelRegistryDetail[]
  isProviderLoading: (provider: string) => boolean
  providerHasMoreModels: (provider: string) => boolean
  isActivating: (modelId: string) => boolean
}>()

const emit = defineEmits<{
  (e: 'activate', modelId: string): void
  (e: 'expand', provider: string): void
  (e: 'load-more', provider: string): void
}>()

const { t } = useI18n()
const expandedProviders = ref<Set<string>>(new Set())

function toggle(provider: string) {
  const next = new Set(expandedProviders.value)
  if (next.has(provider)) {
    next.delete(provider)
  } else {
    next.add(provider)
    emit('expand', provider)
  }
  expandedProviders.value = next
}
</script>
