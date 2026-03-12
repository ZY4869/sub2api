<template>
  <TablePageLayout>
    <template #actions>
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          <p>{{ t('admin.models.pages.registry.description') }}</p>
          <p v-if="lastLoadedAt" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.models.registry.lastSynced', { time: formatDateTime(lastLoadedAt) }) }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button class="btn btn-secondary" :disabled="loading" @click="handleRefresh">
            {{ t('common.refresh') }}
          </button>
          <button class="btn btn-primary" @click="openCreate">
            {{ t('admin.models.registry.addModel') }}
          </button>
        </div>
      </div>
    </template>

    <template #filters>
      <div class="flex flex-wrap items-end gap-3">
        <div class="min-w-[220px] flex-1">
          <label class="input-label" for="registry-search">{{ t('common.search') }}</label>
          <input
            id="registry-search"
            v-model.trim="filters.search"
            type="text"
            class="input"
            :placeholder="t('admin.models.registry.searchPlaceholder')"
            @keyup.enter="applyFilters"
          />
        </div>

        <div class="min-w-[160px]">
          <label class="input-label" for="registry-provider-filter">{{ t('admin.models.registry.fields.provider') }}</label>
          <input
            id="registry-provider-filter"
            v-model.trim="filters.provider"
            type="text"
            class="input"
            :placeholder="t('admin.models.registry.providerPlaceholder')"
            @keyup.enter="applyFilters"
          />
        </div>

        <div class="min-w-[160px]">
          <label class="input-label" for="registry-platform-filter">{{ t('admin.models.registry.fields.platforms') }}</label>
          <input
            id="registry-platform-filter"
            v-model.trim="filters.platform"
            type="text"
            class="input"
            :placeholder="t('admin.models.registry.platformPlaceholder')"
            @keyup.enter="applyFilters"
          />
        </div>

        <label class="flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          <input v-model="filters.includeHidden" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" @change="applyFilters" />
          {{ t('admin.models.registry.includeHidden') }}
        </label>

        <label class="flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          <input v-model="filters.includeTombstoned" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" @change="applyFilters" />
          {{ t('admin.models.registry.includeTombstoned') }}
        </label>

        <button class="btn btn-secondary" :disabled="loading" @click="applyFilters">
          {{ t('common.search') }}
        </button>
      </div>
    </template>

    <template #table>
      <DataTable :columns="columns" :data="items" :loading="loading">
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
          <div class="flex max-w-[220px] flex-wrap gap-2">
            <span
              v-for="platform in row.platforms"
              :key="`${row.id}-${platform}`"
              class="inline-flex rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200"
            >
              {{ platform }}
            </span>
            <span v-if="row.platforms.length === 0" class="text-sm text-gray-400 dark:text-gray-500">-</span>
          </div>
        </template>

        <template #cell-source="{ row }">
          <span :class="sourceClass(row.source)">
            {{ formatSourceLabel(row.source) }}
          </span>
        </template>

        <template #cell-status="{ row }">
          <div class="flex flex-wrap gap-2">
            <span :class="row.hidden ? statusClass('hidden') : statusClass('active')">
              {{ row.hidden ? t('admin.models.registry.statusLabels.hidden') : t('admin.models.registry.statusLabels.active') }}
            </span>
            <span v-if="row.tombstoned" :class="statusClass('tombstoned')">
              {{ t('admin.models.registry.statusLabels.tombstoned') }}
            </span>
          </div>
        </template>

        <template #cell-actions="{ row }">
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary btn-sm" @click="openEdit(row)">
              {{ t('common.edit') }}
            </button>
            <button class="btn btn-secondary btn-sm" @click="openDeleteDialog(row)">
              {{ row.hidden ? t('admin.models.registry.actions.show') : t('admin.models.registry.actions.hide') }}
            </button>
            <button class="btn btn-danger btn-sm" @click="openDeleteDialog(row)">
              {{ t('common.delete') }}
            </button>
          </div>
        </template>

        <template #empty>
          <EmptyState
            :title="t('admin.models.registry.emptyTitle')"
            :description="t('admin.models.registry.emptyDescription')"
          />
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

  <ModelRegistryEntryModal
    :show="entryModalOpen"
    :entry="activeEntry"
    :saving="actionLoading"
    @close="closeEntryModal"
    @submit="handleEntrySubmit"
  />

  <ModelRegistryDeleteDialog
    :show="deleteDialogOpen"
    :entry="activeEntry"
    :saving="actionLoading"
    @close="closeDeleteDialog"
    @toggle-visibility="handleToggleVisibility"
    @hard-delete="handleHardDelete"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import {
  listModelRegistry,
  upsertModelRegistryEntry,
  updateModelRegistryVisibility,
  deleteModelRegistryEntry,
  type ModelRegistryDetail,
  type UpsertModelRegistryEntryPayload
} from '@/api/admin/modelRegistry'
import ModelRegistryDeleteDialog from '@/components/admin/models/ModelRegistryDeleteDialog.vue'
import ModelRegistryEntryModal from '@/components/admin/models/ModelRegistryEntryModal.vue'
import { ensureModelRegistryFresh, invalidateModelRegistry } from '@/stores/modelRegistry'
import { useAppStore } from '@/stores/app'
import { useModelInventoryStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()
const modelInventoryStore = useModelInventoryStore()

const loading = ref(false)
const actionLoading = ref(false)
const items = ref<ModelRegistryDetail[]>([])
const activeEntry = ref<ModelRegistryDetail | null>(null)
const entryModalOpen = ref(false)
const deleteDialogOpen = ref(false)
const lastLoadedAt = ref('')

const filters = reactive({
  search: '',
  provider: '',
  platform: '',
  includeHidden: false,
  includeTombstoned: false
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
  { key: 'source', label: t('admin.models.registry.columns.source') },
  { key: 'status', label: t('admin.models.registry.columns.status') },
  { key: 'actions', label: t('common.actions') }
])

onMounted(() => {
  void handleRefresh()
})

watch(
  () => modelInventoryStore.revision,
  (revision, previous) => {
    if (!revision || revision === previous) {
      return
    }
    void handleRefresh()
  }
)

async function loadList() {
  loading.value = true
  try {
    const response = await listModelRegistry({
      search: filters.search || undefined,
      provider: filters.provider || undefined,
      platform: filters.platform || undefined,
      include_hidden: filters.includeHidden,
      include_tombstoned: filters.includeTombstoned,
      page: pagination.page,
      page_size: pagination.page_size
    })
    items.value = response.items
    pagination.total = response.total
    pagination.page = response.page
    pagination.page_size = response.page_size
    pagination.pages = response.pages
    lastLoadedAt.value = new Date().toISOString()
  } catch (error) {
    console.error('[ModelRegistryView] load failed', error)
    appStore.showError(t('admin.models.registry.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function refreshRegistryState() {
  invalidateModelRegistry()
  await Promise.allSettled([
    ensureModelRegistryFresh(true),
    loadList()
  ])
}

async function handleRefresh() {
  await refreshRegistryState()
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

function openCreate() {
  activeEntry.value = null
  entryModalOpen.value = true
}

function openEdit(entry: ModelRegistryDetail) {
  activeEntry.value = cloneEntry(entry)
  entryModalOpen.value = true
}

function closeEntryModal() {
  entryModalOpen.value = false
  activeEntry.value = null
}

function openDeleteDialog(entry: ModelRegistryDetail) {
  activeEntry.value = cloneEntry(entry)
  deleteDialogOpen.value = true
}

function closeDeleteDialog() {
  deleteDialogOpen.value = false
  activeEntry.value = null
}

async function handleEntrySubmit(payload: UpsertModelRegistryEntryPayload) {
  actionLoading.value = true
  try {
    await upsertModelRegistryEntry(payload)
    closeEntryModal()
    appStore.showSuccess(t('admin.models.registry.saveSuccess'))
    await refreshRegistryState()
    modelInventoryStore.invalidate()
  } catch (error) {
    console.error('[ModelRegistryView] save failed', error)
    appStore.showError(t('admin.models.registry.saveFailed'))
  } finally {
    actionLoading.value = false
  }
}

async function handleToggleVisibility() {
  if (!activeEntry.value) {
    return
  }
  const nextHidden = !activeEntry.value.hidden
  actionLoading.value = true
  try {
    await updateModelRegistryVisibility({
      model: activeEntry.value.id,
      hidden: nextHidden
    })
    closeDeleteDialog()
    appStore.showSuccess(
      t(nextHidden ? 'admin.models.registry.hideSuccess' : 'admin.models.registry.showSuccess')
    )
    await refreshRegistryState()
    modelInventoryStore.invalidate()
  } catch (error) {
    console.error('[ModelRegistryView] visibility update failed', error)
    appStore.showError(t('admin.models.registry.visibilityFailed'))
  } finally {
    actionLoading.value = false
  }
}

async function handleHardDelete() {
  if (!activeEntry.value) {
    return
  }
  actionLoading.value = true
  try {
    await deleteModelRegistryEntry(activeEntry.value.id)
    closeDeleteDialog()
    appStore.showSuccess(t('admin.models.registry.deleteSuccess'))
    await refreshRegistryState()
    modelInventoryStore.invalidate()
  } catch (error) {
    console.error('[ModelRegistryView] hard delete failed', error)
    appStore.showError(t('admin.models.registry.deleteFailed'))
  } finally {
    actionLoading.value = false
  }
}

function formatSourceLabel(source: string) {
  const key = `admin.models.registry.sourceLabels.${source}`
  const translated = t(key)
  return translated === key ? source : translated
}

function sourceClass(source: string) {
  const classes: Record<string, string> = {
    manual: 'inline-flex rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-500/15 dark:text-primary-300',
    seed: 'inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300',
    legacy: 'inline-flex rounded-full bg-amber-100 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  }
  return classes[source] || 'inline-flex rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}

function statusClass(status: 'active' | 'hidden' | 'tombstoned') {
  const classes: Record<typeof status, string> = {
    active: 'inline-flex rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300',
    hidden: 'inline-flex rounded-full bg-amber-100 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    tombstoned: 'inline-flex rounded-full bg-red-100 px-2.5 py-1 text-xs font-medium text-red-700 dark:bg-red-500/15 dark:text-red-300'
  }
  return classes[status]
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(new Date(value))
}

function cloneEntry(entry: ModelRegistryDetail): ModelRegistryDetail {
  return {
    ...entry,
    platforms: [...entry.platforms],
    protocol_ids: [...entry.protocol_ids],
    aliases: [...entry.aliases],
    pricing_lookup_ids: [...entry.pricing_lookup_ids],
    modalities: [...entry.modalities],
    capabilities: [...entry.capabilities],
    exposed_in: [...entry.exposed_in]
  }
}
</script>
