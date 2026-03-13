<template>
  <TablePageLayout>
    <template #actions>
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          {{ t('admin.models.pages.available.description') }}
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button class="btn btn-secondary" :disabled="loading" @click="loadList">
            {{ t('common.refresh') }}
          </button>
          <button class="btn btn-primary" :disabled="loading" @click="openActivateDialog">
            {{ t('admin.models.available.addAction') }}
          </button>
        </div>
      </div>
    </template>

    <template #filters>
      <div class="flex flex-wrap items-end gap-3">
        <input v-model.trim="filters.search" type="text" class="input min-w-[220px] flex-1" :placeholder="t('admin.models.registry.searchPlaceholder')" @keyup.enter="applyFilters" />
        <input v-model.trim="filters.provider" type="text" class="input min-w-[160px]" :placeholder="t('admin.models.registry.providerPlaceholder')" @keyup.enter="applyFilters" />
        <input v-model.trim="filters.platform" type="text" class="input min-w-[160px]" :placeholder="t('admin.models.registry.platformPlaceholder')" @keyup.enter="applyFilters" />
        <button class="btn btn-secondary" :disabled="loading" @click="applyFilters">{{ t('common.search') }}</button>
      </div>
    </template>

    <template #table>
      <DataTable :columns="columns" :data="items" :loading="loading" row-key="id">
        <template #cell-model="{ row }">
          <div class="min-w-[240px]">
            <p class="font-medium text-gray-900 dark:text-white">{{ row.id }}</p>
            <p v-if="row.display_name" class="text-xs text-gray-500 dark:text-gray-400">{{ row.display_name }}</p>
          </div>
        </template>

        <template #cell-provider="{ value }">
          <span class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">
            {{ value || '-' }}
          </span>
        </template>

        <template #cell-platforms="{ row }">
          <div class="flex flex-wrap gap-2">
            <span v-for="platform in row.platforms" :key="`${row.id}-${platform}`" class="inline-flex rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
              {{ platform }}
            </span>
          </div>
        </template>

        <template #cell-actions="{ row }">
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
    :items="unavailableItems"
    :submitting="submitting"
    @close="activateDialogOpen = false"
    @submit="activateSelected"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import ActivateAvailableModelsDialog from '@/components/admin/models/ActivateAvailableModelsDialog.vue'
import {
  activateModelRegistryEntries,
  deactivateModelRegistryEntries,
  listModelRegistry,
  type ModelRegistryDetail
} from '@/api/admin/modelRegistry'
import { useAppStore } from '@/stores/app'
import { ensureModelRegistryFresh, invalidateModelRegistry } from '@/stores/modelRegistry'
import { useModelInventoryStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()
const modelInventoryStore = useModelInventoryStore()

const loading = ref(false)
const submitting = ref(false)
const activateDialogOpen = ref(false)
const items = ref<ModelRegistryDetail[]>([])
const unavailableItems = ref<ModelRegistryDetail[]>([])

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
  { key: 'actions', label: t('common.actions') }
])

onMounted(() => {
  void loadList()
})

async function loadList() {
  loading.value = true
  try {
    const response = await listModelRegistry({
      search: filters.search || undefined,
      provider: filters.provider || undefined,
      platform: filters.platform || undefined,
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
  submitting.value = true
  try {
    const response = await listModelRegistry({
      availability: 'unavailable',
      include_hidden: false,
      include_tombstoned: false,
      page: 1,
      page_size: 1000
    })
    unavailableItems.value = response.items
    activateDialogOpen.value = true
  } catch (error) {
    console.error('[AvailableModelsView] load unavailable failed', error)
    appStore.showError(t('admin.models.registry.loadFailed'))
  } finally {
    submitting.value = false
  }
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
</script>
