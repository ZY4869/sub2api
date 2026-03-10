<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <ModelCatalogFilters
          :search="filters.search"
          :provider="filters.provider"
          :mode="filters.mode"
          :availability="filters.availability"
          :pricing-source="filters.pricingSource"
          :provider-options="providerOptions"
          :mode-options="modeOptions"
          :availability-options="availabilityOptions"
          :pricing-source-options="pricingSourceOptions"
          @update:search="updateFilter('search', $event, false)"
          @update:provider="updateFilter('provider', $event)"
          @update:mode="updateFilter('mode', $event)"
          @update:availability="updateFilter('availability', $event)"
          @update:pricing-source="updateFilter('pricingSource', $event)"
          @search="loadModels(true)"
        />
      </template>

      <template #table>
        <ModelCatalogTable :items="items" :loading="loading" @inspect="openDetail" />
      </template>

      <template #pagination>
        <Pagination :total="total" :page="page" :page-size="pageSize" @update:page="handlePageChange" @update:page-size="handlePageSizeChange" />
      </template>
    </TablePageLayout>

    <ModelCatalogDetailDialog
      :show="dialogOpen"
      :detail="detail"
      :loading="detailLoading"
      :saving="saving"
      @close="closeDetail"
      @save="saveOverride"
      @reset="resetOverride"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import {
  modelsAPI,
  type ModelCatalogDetail,
  type ModelCatalogItem,
  type ModelCatalogPricingSource,
  type UpdatePricingOverridePayload
} from '@/api/admin/models'
import { useAppStore } from '@/stores'
import ModelCatalogFilters from '@/components/admin/models/ModelCatalogFilters.vue'
import ModelCatalogTable from '@/components/admin/models/ModelCatalogTable.vue'
import ModelCatalogDetailDialog from '@/components/admin/models/ModelCatalogDetailDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const detailLoading = ref(false)
const saving = ref(false)
const dialogOpen = ref(false)
const items = ref<ModelCatalogItem[]>([])
const detail = ref<ModelCatalogDetail | null>(null)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const filters = reactive({
  search: '',
  provider: '',
  mode: '',
  availability: '',
  pricingSource: ''
})

const providerOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.models.filters.allProviders') },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' }
])

const modeOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.models.filters.allModes') },
  { value: 'chat', label: t('admin.models.modes.chat') },
  { value: 'image', label: t('admin.models.modes.image') },
  { value: 'video', label: t('admin.models.modes.video') },
  { value: 'prompt_enhance', label: t('admin.models.modes.promptEnhance') }
])

const availabilityOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.models.filters.allAvailability') },
  { value: 'available', label: t('common.available') },
  { value: 'unavailable', label: t('admin.models.unavailable') }
])

const pricingSourceOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.models.filters.allPricingSources') },
  { value: 'dynamic', label: t('admin.models.sources.dynamic') },
  { value: 'fallback', label: t('admin.models.sources.fallback') },
  { value: 'override', label: t('admin.models.sources.override') },
  { value: 'none', label: t('admin.models.sources.none') }
])

async function loadModels(resetPage = false) {
  if (resetPage) page.value = 1
  loading.value = true
  try {
    const response = await modelsAPI.listModels({
      search: filters.search || undefined,
      provider: filters.provider || undefined,
      mode: filters.mode || undefined,
      availability: (filters.availability || undefined) as 'available' | 'unavailable' | undefined,
      pricing_source: (filters.pricingSource || undefined) as ModelCatalogPricingSource | undefined,
      page: page.value,
      page_size: pageSize.value
    })
    items.value = response.items
    total.value = response.total
  } catch (error) {
    appStore.showError(extractErrorMessage(error, t('admin.models.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function openDetail(model: string) {
  dialogOpen.value = true
  detailLoading.value = true
  try {
    detail.value = await modelsAPI.getModelDetail(model)
  } catch (error) {
    appStore.showError(extractErrorMessage(error, t('admin.models.detailLoadFailed')))
    dialogOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  dialogOpen.value = false
  detail.value = null
}

async function saveOverride(payload: UpdatePricingOverridePayload) {
  saving.value = true
  try {
    detail.value = await modelsAPI.updatePricingOverride(payload)
    appStore.showSuccess(t('admin.models.saveSuccess'))
    await loadModels()
  } catch (error) {
    appStore.showError(extractErrorMessage(error, t('admin.models.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function resetOverride(model: string) {
  saving.value = true
  try {
    await modelsAPI.deletePricingOverride(model)
    detail.value = await modelsAPI.getModelDetail(model)
    appStore.showSuccess(t('admin.models.resetSuccess'))
    await loadModels()
  } catch (error) {
    appStore.showError(extractErrorMessage(error, t('admin.models.resetFailed')))
  } finally {
    saving.value = false
  }
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  loadModels()
}

function handlePageSizeChange(nextPageSize: number) {
  pageSize.value = nextPageSize
  page.value = 1
  loadModels()
}

function updateFilter(key: keyof typeof filters, value: string, autoLoad = true) {
  filters[key] = value
  if (autoLoad) {
    loadModels(true)
  }
}

function extractErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}

onMounted(() => {
  loadModels()
})
</script>
