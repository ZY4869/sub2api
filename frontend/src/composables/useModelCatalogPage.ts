import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SelectOption } from '@/types'
import {
  modelsAPI,
  type ModelCatalogDetail,
  type ModelCatalogItem,
  type ModelCatalogPricingSource,
  type UpdatePricingOverridePayload
} from '@/api/admin/models'
import { useAppStore, useModelInventoryStore } from '@/stores'
import {
  getModelCatalogPriceDisplayMode,
  MODEL_CATALOG_PAGE_SIZE,
  setModelCatalogPriceDisplayMode,
  type ModelCatalogPriceDisplayMode
} from '@/utils/modelCatalogPresentation'

export type ModelCatalogPricingLayer = 'official' | 'sale'

export function useModelCatalogPage(pricingLayer: ModelCatalogPricingLayer) {
  const { t } = useI18n()
  const appStore = useAppStore()
  const modelInventoryStore = useModelInventoryStore()

  const loading = ref(false)
  const detailLoading = ref(false)
  const saving = ref(false)
  const dialogOpen = ref(false)
  const items = ref<ModelCatalogItem[]>([])
  const detail = ref<ModelCatalogDetail | null>(null)
  const priceDisplayMode = ref<ModelCatalogPriceDisplayMode>(getModelCatalogPriceDisplayMode())

  const filters = reactive({
    search: '',
    provider: '',
    mode: '',
    availability: 'available',
    pricingSource: ''
  })

  watch(priceDisplayMode, (mode) => {
    setModelCatalogPriceDisplayMode(mode)
  })

  watch(
    () => modelInventoryStore.revision,
    (revision, previous) => {
      if (!revision || revision === previous) {
        return
      }
      void loadModels()
      if (detail.value?.model) {
        void openDetail(detail.value.model)
      }
    }
  )

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

  async function loadModels(_resetPage = false) {
    loading.value = true
    try {
      const response = await modelsAPI.listModels({
        search: filters.search || undefined,
        provider: filters.provider || undefined,
        mode: filters.mode || undefined,
        availability: (filters.availability || undefined) as 'available' | 'unavailable' | undefined,
        pricing_source: (filters.pricingSource || undefined) as ModelCatalogPricingSource | undefined,
        page: 1,
        page_size: MODEL_CATALOG_PAGE_SIZE
      })
      items.value = response.items
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

  async function saveOverride(payload: UpdatePricingOverridePayload) {
    saving.value = true
    try {
      detail.value = pricingLayer === 'official'
        ? await modelsAPI.updateOfficialPricingOverride(payload)
        : await modelsAPI.updatePricingOverride(payload)
      appStore.showSuccess(t(pricingLayer === 'official' ? 'admin.models.officialSaveSuccess' : 'admin.models.saleSaveSuccess'))
      await loadModels()
    } catch (error) {
      appStore.showError(extractErrorMessage(error, t(pricingLayer === 'official' ? 'admin.models.officialSaveFailed' : 'admin.models.saleSaveFailed')))
    } finally {
      saving.value = false
    }
  }

  async function resetOverride(model: string) {
    saving.value = true
    try {
      if (pricingLayer === 'official') {
        await modelsAPI.deleteOfficialPricingOverride(model)
      } else {
        await modelsAPI.deletePricingOverride(model)
      }
      detail.value = await modelsAPI.getModelDetail(model)
      appStore.showSuccess(t(pricingLayer === 'official' ? 'admin.models.officialResetSuccess' : 'admin.models.saleResetSuccess'))
      await loadModels()
    } catch (error) {
      appStore.showError(extractErrorMessage(error, t(pricingLayer === 'official' ? 'admin.models.officialResetFailed' : 'admin.models.saleResetFailed')))
    } finally {
      saving.value = false
    }
  }


  async function copyOfficialToSale(model: string) {
    saving.value = true
    try {
      detail.value = await modelsAPI.copyOfficialPricingToSale(model)
      appStore.showSuccess(t('admin.models.copyToSaleSuccess'))
      await loadModels()
    } catch (error) {
      appStore.showError(extractErrorMessage(error, t('admin.models.copyToSaleFailed')))
    } finally {
      saving.value = false
    }
  }

  function updateFilter(key: keyof typeof filters, value: string, autoLoad = true) {
    filters[key] = value
    if (autoLoad) {
      loadModels(true)
    }
  }

  function closeDetail() {
    dialogOpen.value = false
    detail.value = null
  }

  return {
    loading,
    detailLoading,
    saving,
    dialogOpen,
    items,
    detail,
    priceDisplayMode,
    filters,
    providerOptions,
    modeOptions,
    availabilityOptions,
    pricingSourceOptions,
    loadModels,
    openDetail,
    saveOverride,
    resetOverride,
    copyOfficialToSale,
    updateFilter,
    closeDetail
  }
}

function extractErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}
