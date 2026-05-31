<template>
  <TablePageLayout>
    <template #actions>
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          {{ t('admin.models.pages.available.description') }}
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button class="btn btn-secondary" :disabled="loading" @click="refreshAll">
            {{ t('common.refresh') }}
          </button>
          <button class="btn btn-secondary" :disabled="loading" @click="openActivateDialog">
            {{ t('admin.models.available.addAction') }}
          </button>
          <button class="btn btn-primary" :disabled="loading" @click="manualAddDialogOpen = true">
            {{ t('admin.models.available.manualAddAction') }}
          </button>
        </div>
      </div>
    </template>

    <template #filters>
      <div class="flex flex-nowrap items-end gap-3 overflow-x-auto pb-1">
        <input
          v-model.trim="filters.search"
          type="text"
          class="input min-w-[220px] flex-1"
          :placeholder="t('admin.models.registry.searchPlaceholder')"
          @keyup.enter="applyFilters"
        />
        <div class="min-w-[160px] shrink-0">
          <input
            v-model.trim="filters.provider"
            type="text"
            class="input w-full"
            :placeholder="t('admin.models.registry.providerPlaceholder')"
            list="available-models-provider-options"
            @keyup.enter="applyFilters"
          />
          <datalist id="available-models-provider-options">
            <option v-for="option in providerSuggestions" :key="option" :value="option" />
          </datalist>
        </div>
        <div class="min-w-[160px] shrink-0">
          <input
            v-model.trim="filters.platform"
            type="text"
            class="input w-full"
            :placeholder="t('admin.models.registry.platformPlaceholder')"
            list="available-models-platform-options"
            @keyup.enter="applyFilters"
          />
          <datalist id="available-models-platform-options">
            <option v-for="option in platformSuggestions" :key="option" :value="option" />
          </datalist>
        </div>
        <button class="btn btn-secondary shrink-0" :disabled="loading" @click="applyFilters">{{ t('common.search') }}</button>
      </div>
    </template>

    <template #table>
      <DataTable :columns="columns" :data="items" :loading="loading" row-key="id">
        <template #cell-model="{ row }">
          <div class="flex min-w-[240px] items-start gap-3">
            <ModelIcon :model="row.id" :provider="row.provider" :display-name="row.display_name" size="18px" />
            <div class="min-w-0">
              <p class="font-medium text-gray-900 dark:text-white">{{ row.id }}</p>
              <p v-if="row.display_name" class="text-xs text-gray-500 dark:text-gray-400">{{ row.display_name }}</p>
            </div>
          </div>
        </template>

        <template #cell-provider="{ value }">
          <span class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">
            {{ value || '-' }}
          </span>
        </template>

        <template #cell-platforms="{ row }">
          <ModelPlatformsInline :platforms="row.platforms" />
        </template>

        <template #cell-schedule="{ row }">
          <span
            v-if="row.schedule_status && row.schedule_status !== 'active'"
            class="inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
            :class="scheduleBadgeClass(row.schedule_status)"
          >
            {{ scheduleStatusLabel(row.schedule_status) }}
          </span>
          <span v-else class="text-xs text-gray-400">-</span>
        </template>

        <template #cell-actions="{ row }">
          <button class="btn btn-secondary btn-sm" :disabled="submitting" @click="openScheduleDialog(row)">
            {{ t('admin.models.registry.actions.editSchedule') }}
          </button>
          <button class="btn btn-secondary btn-sm" :disabled="submitting" @click="deactivateOne(row.id)">
            {{ t('admin.models.registry.actions.deactivate') }}
          </button>
        </template>

        <template #empty>
          <EmptyState :title="t('admin.models.available.emptyTitle')" :description="t('admin.models.available.emptyDescription')" />
        </template>
      </DataTable>
    </template>

    <template #pagination>
      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </template>
  </TablePageLayout>

  <ActivateAvailableModelsDialog
    :show="activateDialogOpen"
    :submitting="submitting"
    @close="activateDialogOpen = false"
    @submit="activateSelected"
  />
  <ManualAddModelDialog
    :show="manualAddDialogOpen"
    :submitting="submitting"
    @close="manualAddDialogOpen = false"
    @submit="manualAddModel"
  />
  <ModelRegistryScheduleDialog
    :show="scheduleDialogOpen"
    :model="scheduleModel"
    :submitting="submitting"
    @close="scheduleDialogOpen = false"
    @submit="saveSchedule"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformsInline from '@/components/common/ModelPlatformsInline.vue'
import Pagination from '@/components/common/Pagination.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import ActivateAvailableModelsDialog from '@/components/admin/models/ActivateAvailableModelsDialog.vue'
import ManualAddModelDialog from '@/components/admin/models/ManualAddModelDialog.vue'
import {
  activateModelRegistryEntries,
  deactivateModelRegistryEntries,
  listModelRegistry,
  manualAddModelRegistryEntry,
  upsertModelRegistryEntry,
  type ManualAddModelRegistryEntryPayload,
  type ModelRegistryDetail
} from '@/api/admin/modelRegistry'
import { useAppStore } from '@/stores/app'
import { ensureModelRegistryFresh, getModelRegistrySnapshot, invalidateModelRegistry } from '@/stores/modelRegistry'
import { useModelInventoryStore } from '@/stores'
import ModelRegistryScheduleDialog from '@/components/admin/models/ModelRegistryScheduleDialog.vue'
import type { TimeAccessPolicy } from '@/types/api-key-groups'
import { buildModelRegistryScheduleUpsertPayload } from '@/utils/modelRegistrySchedule'

const { t } = useI18n()
const appStore = useAppStore()
const modelInventoryStore = useModelInventoryStore()

const loading = ref(false)
const submitting = ref(false)
const activateDialogOpen = ref(false)
const manualAddDialogOpen = ref(false)
const scheduleDialogOpen = ref(false)
const scheduleModel = ref<ModelRegistryDetail | null>(null)
const items = ref<ModelRegistryDetail[]>([])

const filters = reactive({
  search: '',
  provider: '',
  platform: ''
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 0
})

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('admin.models.registry.columns.model') },
  { key: 'provider', label: t('admin.models.registry.columns.provider') },
  { key: 'platforms', label: t('admin.models.registry.columns.platforms') },
  { key: 'schedule', label: t('admin.models.registry.columns.schedule') },
  { key: 'actions', label: t('common.actions') }
])

const providerSuggestions = computed(() => {
  const snapshot = getModelRegistrySnapshot()
  const values = snapshot.models
    .map((entry) => String(entry.provider || '').trim().toLowerCase())
    .filter((value) => value.length > 0)
  return [...new Set(values)].sort()
})

const platformSuggestions = computed(() => {
  const snapshot = getModelRegistrySnapshot()
  const values = snapshot.models
    .flatMap((entry) => (entry.platforms || []).map((value) => String(value || '').trim().toLowerCase()))
    .filter((value) => value.length > 0)
  return [...new Set(values)].sort()
})

function scheduleStatusLabel(status: string) {
  const keyMap: Record<string, string> = {
    scheduled: 'admin.models.registry.scheduleStatuses.scheduled',
    expired: 'admin.models.registry.scheduleStatuses.expired',
    out_of_window: 'admin.models.registry.scheduleStatuses.outOfWindow',
    invalid: 'admin.models.registry.scheduleStatuses.invalid'
  }
  return t(keyMap[status] || 'admin.models.registry.scheduleStatuses.invalid')
}

function scheduleBadgeClass(status: string) {
  if (status === 'scheduled') return 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300'
  if (status === 'out_of_window') return 'bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300'
  return 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300'
}

onMounted(() => {
  void Promise.allSettled([ensureModelRegistryFresh(), loadList()])
})

async function loadList() {
  loading.value = true
  try {
    const response = await listModelRegistry({
      search: filters.search || undefined,
      provider: filters.provider ? filters.provider.trim().toLowerCase() : undefined,
      platform: filters.platform ? filters.platform.trim().toLowerCase() : undefined,
      availability: 'available',
      include_hidden: false,
      include_tombstoned: false,
      page: pagination.page,
      page_size: pagination.page_size
    })
    items.value = response.items
    pagination.total = response.total
    pagination.page = response.page
    pagination.page_size = response.page_size
    pagination.pages = response.pages
  } catch (error) {
    console.error('[AvailableModelsView] load failed', error)
    appStore.showError(t('admin.models.registry.loadFailed'))
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  pagination.page = 1
  void loadList()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadList()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadList()
}

async function refreshAll() {
  invalidateModelRegistry()
  modelInventoryStore.invalidate()
  await Promise.allSettled([ensureModelRegistryFresh(true), loadList()])
}

async function openActivateDialog() {
  activateDialogOpen.value = true
}

async function activateSelected(modelIds: string[]) {
  submitting.value = true
  try {
    await activateModelRegistryEntries({ models: modelIds })
    activateDialogOpen.value = false
    appStore.showSuccess(t('admin.models.registry.activateSuccess'))
    await refreshAll()
  } catch (error) {
    console.error('[AvailableModelsView] activate failed', error)
    appStore.showError(t('admin.models.registry.availabilityFailed'))
  } finally {
    submitting.value = false
  }
}

async function deactivateOne(modelId: string) {
  submitting.value = true
  try {
    await deactivateModelRegistryEntries({ models: [modelId] })
    appStore.showSuccess(t('admin.models.registry.deactivateSuccess'))
    await refreshAll()
  } catch (error) {
    console.error('[AvailableModelsView] deactivate failed', error)
    appStore.showError(t('admin.models.registry.availabilityFailed'))
  } finally {
    submitting.value = false
  }
}

async function manualAddModel(payload: ManualAddModelRegistryEntryPayload) {
  submitting.value = true
  try {
    await manualAddModelRegistryEntry(payload)
    manualAddDialogOpen.value = false
    appStore.showSuccess(t('admin.models.available.manualAddSuccess'))
    await refreshAll()
  } catch (error) {
    console.error('[AvailableModelsView] manual add failed', error)
    appStore.showError(t('admin.models.available.manualAddFailed'))
  } finally {
    submitting.value = false
  }
}

function openScheduleDialog(model: ModelRegistryDetail) {
  scheduleModel.value = model
  scheduleDialogOpen.value = true
}

async function saveSchedule(
  model: ModelRegistryDetail,
  patch: {
    available_from?: string
    available_until?: string
    access_time_policy?: TimeAccessPolicy | null
  }
) {
  submitting.value = true
  try {
    await upsertModelRegistryEntry(buildModelRegistryScheduleUpsertPayload(model, patch))
    scheduleDialogOpen.value = false
    appStore.showSuccess(t('admin.models.registry.scheduleDialog.saveSuccess'))
    await refreshAll()
  } catch (error) {
    console.error('[AvailableModelsView] schedule save failed', error)
    appStore.showError(t('admin.models.registry.scheduleDialog.saveFailed'))
  } finally {
    submitting.value = false
  }
}
</script>
