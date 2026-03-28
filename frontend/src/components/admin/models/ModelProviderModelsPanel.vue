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

      <div class="flex w-full flex-col gap-3 lg:max-w-3xl lg:flex-row lg:items-center">
        <SearchInput
          :model-value="searchValue"
          :placeholder="t('admin.models.pages.all.filterPlaceholder')"
          @update:model-value="emit('update:search', $event)"
          @search="emit('search', $event)"
        />
        <select
          class="input w-full lg:w-44"
          :value="exposureFilter"
          @change="handleExposureChange"
        >
          <option value="all">{{ t('common.all') }}</option>
          <option value="test">{{ t('admin.models.pages.all.testOnly') }}</option>
        </select>
        <select
          class="input w-full lg:w-44"
          :value="statusFilter"
          @change="handleStatusChange"
        >
          <option value="all">{{ t('common.all') }}</option>
          <option value="stable">{{ t('admin.models.registry.lifecycleLabels.stable') }}</option>
          <option value="beta">{{ t('admin.models.registry.lifecycleLabels.beta') }}</option>
          <option value="deprecated">{{ t('admin.models.registry.lifecycleLabels.deprecated') }}</option>
        </select>
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
          class="btn btn-primary btn-sm"
          :disabled="selectedNonTestCount === 0 || selectedMutating"
          @click="handleBulkAddToTest"
        >
          {{ t('admin.models.pages.all.bulk.addToTest') }}
        </button>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :disabled="selectedTestCount === 0 || selectedMutating"
          @click="handleBulkRemoveFromTest"
        >
          {{ t('admin.models.pages.all.bulk.removeFromTest') }}
        </button>
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
        <select
          v-model="selectedMoveTarget"
          data-test="bulk-move-provider-target"
          class="input min-w-[12rem] text-sm"
          :disabled="selectedCount === 0 || selectedMutating || availableMoveTargets.length === 0"
        >
          <option value="">{{ t('admin.models.pages.all.bulk.moveProviderPlaceholder') }}</option>
          <option
            v-for="option in availableMoveTargets"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </option>
        </select>
        <button
          type="button"
          data-test="bulk-move-provider-button"
          class="btn btn-secondary btn-sm"
          :disabled="selectedCount === 0 || selectedMutating || availableMoveTargets.length === 0"
          @click="handleBulkMoveProvider"
        >
          {{ t('admin.models.pages.all.bulk.moveProvider') }}
        </button>
      </div>

      <div
        v-if="availableMoveTargets.length > 0"
        data-test="bulk-move-provider-hint"
        class="text-xs text-primary-700 dark:text-primary-300"
      >
        {{ t('admin.models.pages.all.bulk.moveProviderHint') }}
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
                    <p class="font-medium text-gray-900 dark:text-white">
                      {{ model.display_name || model.id }}
                    </p>
                    <span
                      class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium"
                      :class="model.available ? availableBadgeClass : unavailableBadgeClass"
                    >
                      {{ model.available ? t('admin.models.registry.availableStatus') : t('admin.models.registry.unavailableStatus') }}
                    </span>
                    <span
                      v-if="hasTestExposure(model)"
                      class="inline-flex rounded-full bg-sky-100 px-2 py-0.5 text-[11px] font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300"
                    >
                      {{ t('admin.models.pages.all.testBadge') }}
                    </span>
                    <span
                      v-if="model.status === 'deprecated'"
                      class="inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
                    >
                      {{ t('admin.models.registry.lifecycleLabels.deprecated') }}
                    </span>
                  </div>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ model.id }}</p>
                  <p
                    v-if="model.replaced_by"
                    class="mt-1 text-xs text-amber-600 dark:text-amber-300"
                  >
                    {{ t('admin.models.registry.replacedByHint', { model: model.replaced_by }) }}
                  </p>
                  <div class="mt-2">
                    <ModelPlatformsInline :platforms="model.platforms" />
                  </div>
                </div>
              </div>
            </div>

            <div class="flex shrink-0 flex-wrap items-center gap-2">
              <button
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="isModelMutating(model.id)"
                @click="handleRowTestExposure(model)"
              >
                {{
                  hasTestExposure(model)
                    ? t('admin.models.pages.all.removeFromTest')
                    : t('admin.models.pages.all.addToTest')
                }}
              </button>
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
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformsInline from '@/components/common/ModelPlatformsInline.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import { useAppStore } from '@/stores/app'
import { groupModelRegistryModels } from '@/utils/modelRegistryCategories'

const props = withDefaults(defineProps<{
  provider: string
  models: ModelRegistryDetail[]
  searchValue?: string
  exposureFilter?: 'all' | 'test'
  statusFilter?: 'all' | 'stable' | 'beta' | 'deprecated'
  selectedIds?: string[]
  moveTargetOptions?: Array<{ value: string; label: string }>
  totalCount?: number
  availableCount?: number
  loading?: boolean
  hasMore?: boolean
  isActivating: (modelId: string) => boolean
  isDeactivating: (modelId: string) => boolean
  isDeleting: (modelId: string) => boolean
  isMoving: (modelId: string) => boolean
  isSyncingTestExposure: (modelId: string) => boolean
}>(), {
  searchValue: '',
  exposureFilter: 'all',
  statusFilter: 'all',
  selectedIds: () => [],
  moveTargetOptions: () => []
})

const emit = defineEmits<{
  (e: 'update:search', value: string): void
  (e: 'search', value: string): void
  (e: 'update:exposure', value: 'all' | 'test'): void
  (e: 'update:status', value: 'all' | 'stable' | 'beta' | 'deprecated'): void
  (e: 'toggle-selected', modelId: string): void
  (e: 'toggle-all-selected', checked: boolean): void
  (e: 'clear-selection'): void
  (e: 'add-to-test', modelIds: string[]): void
  (e: 'remove-from-test', modelIds: string[]): void
  (e: 'activate', modelId: string): void
  (e: 'deactivate', modelIds: string[]): void
  (e: 'hard-delete', modelIds: string[]): void
  (e: 'move-provider', payload: { targetProvider: string; modelIds: string[] }): void
  (e: 'load-more'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const selectedMoveTarget = ref('')

const selectedIdSet = computed(() => new Set(props.selectedIds || []))
const loadedCount = computed(() => props.models.length)
const resolvedAvailableCount = computed(() => props.availableCount ?? props.models.filter((m) => m.available).length)
const resolvedTotalCount = computed(() => props.totalCount ?? props.models.length)
const selectedCount = computed(() => props.selectedIds.length)
const selectedModels = computed(() => props.models.filter((model) => selectedIdSet.value.has(model.id)))
const selectedAvailableCount = computed(() => selectedModels.value.filter((model) => model.available).length)
const selectedTestCount = computed(() => selectedModels.value.filter((model) => hasTestExposure(model)).length)
const selectedNonTestCount = computed(() => selectedModels.value.filter((model) => !hasTestExposure(model)).length)
const selectedMutating = computed(() => selectedModels.value.some((model) => isModelMutating(model.id)))
const groupedModels = computed(() => groupModelRegistryModels(props.models))
const availableMoveTargets = computed(() =>
  (props.moveTargetOptions || []).filter((option) => option.value !== props.provider)
)
const availableBadgeClass = 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
const unavailableBadgeClass = 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'

watch(
  () => props.provider,
  () => {
    selectedMoveTarget.value = ''
  }
)

watch(selectedCount, (count) => {
  if (count === 0) {
    selectedMoveTarget.value = ''
  }
})

function isModelMutating(modelId: string) {
  return props.isActivating(modelId) ||
    props.isDeactivating(modelId) ||
    props.isDeleting(modelId) ||
    props.isMoving(modelId) ||
    props.isSyncingTestExposure(modelId)
}

function hasTestExposure(model: ModelRegistryDetail) {
  return Array.isArray(model.exposed_in) && model.exposed_in.includes('test')
}

function handleExposureChange(event: Event) {
  const value = (event.target as HTMLSelectElement | null)?.value
  emit('update:exposure', value === 'test' ? 'test' : 'all')
}

function handleStatusChange(event: Event) {
  const value = (event.target as HTMLSelectElement | null)?.value
  if (value === 'stable' || value === 'beta' || value === 'deprecated') {
    emit('update:status', value)
    return
  }
  emit('update:status', 'all')
}

function handleBulkAddToTest() {
  if (selectedNonTestCount.value === 0) {
    return
  }
  emit('add-to-test', selectedModels.value.filter((model) => !hasTestExposure(model)).map((model) => model.id))
}

function handleBulkRemoveFromTest() {
  if (selectedTestCount.value === 0) {
    return
  }
  emit('remove-from-test', selectedModels.value.filter((model) => hasTestExposure(model)).map((model) => model.id))
}

function handleRowTestExposure(model: ModelRegistryDetail) {
  if (hasTestExposure(model)) {
    emit('remove-from-test', [model.id])
    return
  }
  emit('add-to-test', [model.id])
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

function handleBulkMoveProvider() {
  if (selectedCount.value === 0) {
    return
  }
  if (!selectedMoveTarget.value) {
    appStore.showWarning(t('admin.models.pages.all.bulk.moveProviderSelectRequired'))
    return
  }
  if (!window.confirm(t('admin.models.pages.all.bulk.moveProviderConfirm', {
    count: selectedCount.value,
    provider: availableMoveTargets.value.find((option) => option.value === selectedMoveTarget.value)?.label || selectedMoveTarget.value
  }))) {
    return
  }
  emit('move-provider', {
    targetProvider: selectedMoveTarget.value,
    modelIds: [...props.selectedIds]
  })
}
</script>
