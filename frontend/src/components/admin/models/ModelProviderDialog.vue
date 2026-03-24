<template>
  <BaseDialog
    :show="show"
    :title="t('admin.models.pages.all.title')"
    width="full"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="flex flex-col gap-4 md:flex-row">
      <div class="w-full md:w-72">
        <div class="mb-3">
          <SearchInput
            :model-value="providerQuery"
            :placeholder="t('common.search')"
            @update:model-value="providerQuery = $event"
          />
        </div>

        <div class="max-h-[60vh] space-y-1 overflow-y-auto rounded-xl border border-gray-200 bg-white p-2 dark:border-dark-700 dark:bg-dark-800">
          <button
            v-for="group in filteredProviders"
            :key="group.provider"
            type="button"
            class="flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left transition-colors"
            :class="group.provider === activeProvider ? 'bg-primary-50 text-primary-700 dark:bg-primary-500/10 dark:text-primary-300' : 'hover:bg-gray-50 dark:hover:bg-dark-700'"
            @click="emit('select-provider', group.provider)"
          >
            <span class="flex min-w-0 items-center gap-3">
              <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gray-100 dark:bg-dark-700">
                <ModelPlatformIcon :platform="group.provider" size="md" />
              </span>
              <span class="min-w-0">
                <span class="block truncate text-sm font-semibold">{{ group.label }}</span>
                <span class="block truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ group.availableCount }}/{{ group.totalCount }} {{ t('admin.models.registry.availableStatus') }}
                </span>
              </span>
            </span>
            <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400" />
          </button>
        </div>
      </div>

      <div class="min-w-0 flex-1">
        <div v-if="activeGroup" class="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div class="flex items-center gap-3">
            <span class="flex h-10 w-10 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-700">
              <ModelPlatformIcon :platform="activeGroup.provider" size="lg" />
            </span>
            <div class="min-w-0">
              <div class="truncate text-base font-semibold text-gray-900 dark:text-white">
                {{ activeGroup.label }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ activeGroup.availableCount }}/{{ activeGroup.totalCount }} {{ t('admin.models.registry.availableStatus') }}
              </div>
            </div>
          </div>

          <button class="btn btn-secondary btn-sm" :disabled="loading" @click="emit('refresh')">
            {{ t('common.refresh') }}
          </button>
        </div>

        <div class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <ModelProviderModelsPanel
            :provider="activeProvider"
            :models="activeModels"
            :search-value="activeSearchValue"
            :exposure-filter="activeExposureFilter"
            :status-filter="activeStatusFilter"
            :selected-ids="activeSelectedIds"
            :total-count="activeGroup?.totalCount"
            :available-count="activeGroup?.availableCount"
            :loading="loading"
            :has-more="hasMore"
            :is-activating="isActivating"
            :is-deactivating="isDeactivating"
            :is-deleting="isDeleting"
            :is-syncing-test-exposure="isSyncingTestExposure"
            @update:search="emit('update:search', activeProvider, $event)"
            @search="emit('search', activeProvider, $event)"
            @update:exposure="emit('update:exposure', activeProvider, $event)"
            @update:status="emit('update:status', activeProvider, $event)"
            @toggle-selected="emit('toggle-selected', activeProvider, $event)"
            @toggle-all-selected="emit('toggle-all-selected', activeProvider, $event)"
            @clear-selection="emit('clear-selection', activeProvider)"
            @add-to-test="emit('add-to-test', activeProvider, $event)"
            @remove-from-test="emit('remove-from-test', activeProvider, $event)"
            @activate="emit('activate', activeProvider, $event)"
            @deactivate="emit('deactivate', activeProvider, $event)"
            @hard-delete="emit('hard-delete', activeProvider, $event)"
            @load-more="emit('load-more', activeProvider)"
          />
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminModelRegistryProviderGroup } from '@/composables/useAdminModelRegistryProviders'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import ModelProviderModelsPanel from '@/components/admin/models/ModelProviderModelsPanel.vue'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'

const props = defineProps<{
  show: boolean
  loading: boolean
  providers: AdminModelRegistryProviderGroup[]
  activeProvider: string
  activeModels: ModelRegistryDetail[]
  activeSearchValue: string
  activeExposureFilter: 'all' | 'test'
  activeStatusFilter: 'all' | 'stable' | 'beta' | 'deprecated'
  activeSelectedIds: string[]
  hasMore: boolean
  isActivating: (modelId: string) => boolean
  isDeactivating: (modelId: string) => boolean
  isDeleting: (modelId: string) => boolean
  isSyncingTestExposure: (modelId: string) => boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'refresh'): void
  (e: 'select-provider', provider: string): void
  (e: 'update:search', provider: string, value: string): void
  (e: 'search', provider: string, value: string): void
  (e: 'update:exposure', provider: string, value: 'all' | 'test'): void
  (e: 'update:status', provider: string, value: 'all' | 'stable' | 'beta' | 'deprecated'): void
  (e: 'toggle-selected', provider: string, modelId: string): void
  (e: 'toggle-all-selected', provider: string, checked: boolean): void
  (e: 'clear-selection', provider: string): void
  (e: 'add-to-test', provider: string, modelIds: string[]): void
  (e: 'remove-from-test', provider: string, modelIds: string[]): void
  (e: 'activate', provider: string, modelId: string): void
  (e: 'deactivate', provider: string, modelIds: string[]): void
  (e: 'hard-delete', provider: string, modelIds: string[]): void
  (e: 'load-more', provider: string): void
}>()

const { t } = useI18n()
const providerQuery = ref('')

const filteredProviders = computed(() => {
  const query = providerQuery.value.trim().toLowerCase()
  if (!query) return props.providers
  return props.providers.filter((item) => `${item.provider} ${item.label}`.toLowerCase().includes(query))
})

const activeGroup = computed(() => props.providers.find((item) => item.provider === props.activeProvider) || null)
const activeModels = computed(() => props.activeModels || [])
</script>
