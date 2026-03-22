<template>
  <div class="space-y-4">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('common.total') }}: {{ resolvedTotalCount }}
        <span class="mx-2 text-gray-300 dark:text-dark-600">|</span>
        {{ t('admin.models.registry.availableStatus') }}: {{ resolvedAvailableCount }}
        <span class="mx-2 text-gray-300 dark:text-dark-600">|</span>
        {{ loadedCount }}/{{ resolvedTotalCount }} {{ t('admin.models.pages.all.loadMore') }}
      </div>

      <div class="w-full lg:max-w-sm">
        <SearchInput
          :model-value="searchValue"
          :placeholder="t('admin.models.pages.all.filterPlaceholder')"
          @update:model-value="emit('update:search', $event)"
          @search="emit('search', $event)"
        />
      </div>
    </div>

    <div
      v-if="selectedCount > 0"
      class="flex flex-col gap-3 rounded-2xl border border-primary-200 bg-primary-50 px-4 py-3 text-sm dark:border-primary-500/20 dark:bg-primary-500/10"
    >
      <div class="flex flex-wrap items-center gap-3">
        <span class="font-medium text-primary-800 dark:text-primary-200">
          {{ t('admin.models.pages.all.bulk.selected', { count: selectedCount }) }}
        </span>
        <button
          type="button"
          class="text-primary-700 transition hover:text-primary-900 dark:text-primary-300 dark:hover:text-primary-100"
          @click="emit('toggle-all-selected', true)"
        >
          {{ t('admin.models.pages.all.bulk.selectLoaded') }}
        </button>
        <button
          type="button"
          class="text-primary-700 transition hover:text-primary-900 dark:text-primary-300 dark:hover:text-primary-100"
          @click="emit('clear-selection')"
        >
          {{ t('admin.models.pages.all.bulk.clear') }}
        </button>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :disabled="selectedAvailableCount === 0 || selectedMutating"
          @click="handleBulkDeactivate"
        >
          {{ t('admin.models.pages.all.bulk.deactivate') }}
        </button>
        <button
          type="button"
          class="btn btn-danger btn-sm"
          :disabled="selectedCount === 0 || selectedMutating"
          @click="handleBulkHardDelete"
        >
          {{ t('admin.models.pages.all.bulk.hardDelete') }}
        </button>
      </div>
    </div>

    <div v-if="loading && models.length === 0" class="flex items-center justify-center py-8">
      <LoadingSpinner />
    </div>

    <div v-else-if="models.length > 0" class="space-y-4">
      <section
        v-for="group in groupedModels"
        :key="group.category"
        class="space-y-2"
      >
        <div class="flex items-center justify-between gap-3 border-b border-gray-100 pb-2 dark:border-dark-700">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t(`admin.models.pages.all.categories.${group.category}`) }}
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ group.items.length }} {{ t('admin.models.registry.columns.model') }}
          </span>
        </div>

        <div class="divide-y divide-gray-100 dark:divide-dark-700">
          <div
            v-for="model in group.items"
            :key="model.id"
            class="flex flex-col gap-3 py-3 sm:flex-row sm:items-start sm:justify-between"
          >
            <div class="min-w-0 flex-1">
              <div class="flex items-start gap-3">
                <input
                  type="checkbox"
                  class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600"
                  :checked="selectedIdSet.has(model.id)"
                  :disabled="isModelMutating(model.id)"
                  @change="emit('toggle-selected', model.id)"
                />

                <ModelIcon :model="model.id" :provider="model.provider" size="18px" />

                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="font-medium text-gray-900 dark:text-white">{{ model.id }}</p>
                    <p v-if="model.display_name" class="text-xs text-gray-500 dark:text-gray-400">
                      {{ model.display_name }}
                    </p>
                    <span
                      class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium"
                      :class="model.available ? availableBadgeClass : unavailableBadgeClass"
                    >
                      {{ model.available ? t('admin.models.registry.availableStatus') : t('admin.models.registry.unavailableStatus') }}
                    </span>
                  </div>
                  <div class="mt-2">
                    <ModelPlatformsInline :platforms="model.platforms" />
                  </div>
                </div>
              </div>
            </div>

            <div class="flex shrink-0 flex-wrap items-center gap-2">
              <button
                v-if="!model.available"
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="isModelMutating(model.id)"
                @click="emit('activate', model.id)"
              >
                {{ t('admin.models.registry.actions.activate') }}
              </button>
              <button
                v-else
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="isModelMutating(model.id)"
                @click="emit('deactivate', [model.id])"
              >
                {{ t('admin.models.registry.actions.deactivate') }}
              </button>
              <button
                type="button"
                class="btn btn-danger btn-sm"
                :disabled="isModelMutating(model.id)"
                @click="handleSingleHardDelete(model.id)"
              >
                {{ t('admin.models.registry.actions.hardDelete') }}
              </button>
            </div>
          </div>
        </div>
      </section>
    </div>

    <div v-else class="py-8">
      <EmptyState :title="t('admin.models.registry.emptyTitle')" :description="t('admin.models.registry.emptyDescription')" />
    </div>

    <div v-if="hasMore" class="flex justify-center pt-2">
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="loading"
        @click="emit('load-more')"
      >
        {{ loading ? t('common.loading') : t('admin.models.pages.all.loadMore') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformsInline from '@/components/common/ModelPlatformsInline.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import { groupModelRegistryModels } from '@/utils/modelRegistryCategories'

const props = withDefaults(defineProps<{
  provider: string
  models: ModelRegistryDetail[]
  searchValue?: string
  selectedIds?: string[]
  totalCount?: number
  availableCount?: number
  loading?: boolean
  hasMore?: boolean
  isActivating: (modelId: string) => boolean
  isDeactivating: (modelId: string) => boolean
  isDeleting: (modelId: string) => boolean
}>(), {
  searchValue: '',
  selectedIds: () => []
})

const emit = defineEmits<{
  (e: 'update:search', value: string): void
  (e: 'search', value: string): void
  (e: 'toggle-selected', modelId: string): void
  (e: 'toggle-all-selected', checked: boolean): void
  (e: 'clear-selection'): void
  (e: 'activate', modelId: string): void
  (e: 'deactivate', modelIds: string[]): void
  (e: 'hard-delete', modelIds: string[]): void
  (e: 'load-more'): void
}>()

const { t } = useI18n()

const selectedIdSet = computed(() => new Set(props.selectedIds || []))
const loadedCount = computed(() => props.models.length)
const resolvedAvailableCount = computed(() => props.availableCount ?? props.models.filter((m) => m.available).length)
const resolvedTotalCount = computed(() => props.totalCount ?? props.models.length)
const selectedCount = computed(() => props.selectedIds.length)
const selectedModels = computed(() => props.models.filter((model) => selectedIdSet.value.has(model.id)))
const selectedAvailableCount = computed(() => selectedModels.value.filter((model) => model.available).length)
const selectedMutating = computed(() => selectedModels.value.some((model) => isModelMutating(model.id)))
const groupedModels = computed(() => groupModelRegistryModels(props.models))
const availableBadgeClass = 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
const unavailableBadgeClass = 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'

function isModelMutating(modelId: string) {
  return props.isActivating(modelId) || props.isDeactivating(modelId) || props.isDeleting(modelId)
}

function handleSingleHardDelete(modelId: string) {
  if (!window.confirm(t('admin.models.pages.all.hardDeleteSingleConfirm', { model: modelId }))) {
    return
  }
  emit('hard-delete', [modelId])
}

function handleBulkDeactivate() {
  if (selectedAvailableCount.value === 0) {
    return
  }
  if (!window.confirm(t('admin.models.pages.all.bulk.deactivateConfirm', { count: selectedAvailableCount.value }))) {
    return
  }
  emit('deactivate', selectedModels.value.filter((model) => model.available).map((model) => model.id))
}

function handleBulkHardDelete() {
  if (selectedCount.value === 0) {
    return
  }
  if (!window.confirm(t('admin.models.pages.all.bulk.hardDeleteConfirm', { count: selectedCount.value }))) {
    return
  }
  emit('hard-delete', [...props.selectedIds])
}
</script>
