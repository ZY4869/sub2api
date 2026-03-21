import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  activateModelRegistryEntries,
  listModelRegistry,
  listModelRegistryProviders,
  type ModelRegistryDetail
} from '@/api/admin/modelRegistry'
import { useAppStore } from '@/stores/app'
import { useModelInventoryStore } from '@/stores'
import { ensureModelRegistryFresh, invalidateModelRegistry } from '@/stores/modelRegistry'
import { formatModelCatalogProvider } from '@/utils/modelCatalogPresentation'

const PROVIDER_SUMMARY_PAGE_SIZE = 24
const PROVIDER_MODELS_PAGE_SIZE = 50

type ProviderModelsState = {
  items: ModelRegistryDetail[]
  total: number
  page: number
  pageSize: number
  pages: number
  loading: boolean
  initialized: boolean
}

export type AdminModelRegistryProviderGroup = {
  provider: string
  label: string
  totalCount: number
  availableCount: number
}

export function useAdminModelRegistryProviders() {
  const { t } = useI18n()
  const appStore = useAppStore()
  const modelInventoryStore = useModelInventoryStore()

  const loading = ref(false)
  const loadingMore = ref(false)
  const activatingIds = ref<Set<string>>(new Set())
  const items = ref<AdminModelRegistryProviderGroup[]>([])
  const pagination = reactive({
    page: 1,
    page_size: PROVIDER_SUMMARY_PAGE_SIZE,
    total: 0,
    pages: 0
  })
  const providerModelStates = reactive<Record<string, ProviderModelsState>>({})

  const providerGroups = computed<AdminModelRegistryProviderGroup[]>(() => {
    return items.value.map((item) => ({
      provider: item.provider,
      label: formatModelCatalogProvider(item.provider),
      totalCount: item.totalCount,
      availableCount: item.availableCount
    }))
  })

  const isActivating = (modelId: string) => activatingIds.value.has(modelId)
  const hasMoreProviders = computed(() => pagination.page < pagination.pages)

  function getProviderKey(provider: string) {
    return String(provider || '').trim().toLowerCase() || 'unknown'
  }

  function createProviderModelsState(): ProviderModelsState {
    return {
      items: [],
      total: 0,
      page: 0,
      pageSize: PROVIDER_MODELS_PAGE_SIZE,
      pages: 0,
      loading: false,
      initialized: false
    }
  }

  function ensureProviderModelsState(provider: string): ProviderModelsState {
    const key = getProviderKey(provider)
    if (!providerModelStates[key]) {
      providerModelStates[key] = createProviderModelsState()
    }
    return providerModelStates[key]
  }

  function sortProviderModels(models: ModelRegistryDetail[]) {
    return [...models].sort((left, right) => {
      if (left.available !== right.available) return left.available ? -1 : 1
      return (left.ui_priority - right.ui_priority) || left.id.localeCompare(right.id)
    })
  }

  async function loadProviderSummaries(reset = false) {
    if (reset) {
      loading.value = true
    } else {
      if (loadingMore.value || !hasMoreProviders.value) {
        return
      }
      loadingMore.value = true
    }
    try {
      const targetPage = reset ? 1 : pagination.page + 1
      const response = await listModelRegistryProviders({
        page: targetPage,
        page_size: pagination.page_size
      })
      const nextItems = response.items.map((item) => ({
        provider: getProviderKey(item.provider),
        label: formatModelCatalogProvider(item.provider),
        totalCount: item.total_count,
        availableCount: item.available_count
      }))
      items.value = reset ? nextItems : dedupeProviderGroups([...items.value, ...nextItems])
      pagination.total = response.total
      pagination.page = response.page
      pagination.page_size = response.page_size
      pagination.pages = response.pages
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] loadProviderSummaries failed', error)
      appStore.showError(t('admin.models.registry.loadFailed'))
    } finally {
      loading.value = false
      loadingMore.value = false
    }
  }

  async function loadAll() {
    await loadProviderSummaries(true)
  }

  async function loadMoreProviders() {
    await loadProviderSummaries(false)
  }

  async function loadProviderModels(provider: string, reset = false) {
    const normalizedProvider = getProviderKey(provider)
    const state = ensureProviderModelsState(normalizedProvider)
    if (state.loading) {
      return
    }
    const nextPage = reset || !state.initialized ? 1 : state.page + 1
    if (!reset && state.initialized && state.pages > 0 && state.page >= state.pages) {
      return
    }
    state.loading = true
    try {
      const response = await listModelRegistry({
        provider: normalizedProvider,
        availability: 'all',
        include_hidden: false,
        include_tombstoned: false,
        page: nextPage,
        page_size: state.pageSize
      })
      const mergedItems = reset || nextPage === 1
        ? response.items
        : mergeProviderModels(state.items, response.items)
      state.items = sortProviderModels(mergedItems)
      state.total = response.total
      state.page = response.page
      state.pageSize = response.page_size
      state.pages = response.pages
      state.initialized = true
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] loadProviderModels failed', error)
      appStore.showError(t('admin.models.registry.loadFailed'))
    } finally {
      state.loading = false
    }
  }

  async function ensureProviderModels(provider: string) {
    const state = ensureProviderModelsState(provider)
    if (state.initialized) {
      return
    }
    await loadProviderModels(provider, true)
  }

  async function loadMoreProviderModels(provider: string) {
    await loadProviderModels(provider, false)
  }

  async function refreshAll() {
    items.value = []
    pagination.page = 1
    pagination.total = 0
    pagination.pages = 0
    for (const key of Object.keys(providerModelStates)) {
      delete providerModelStates[key]
    }
    await loadAll()
  }

  async function activateModel(modelId: string) {
    if (!modelId || isActivating(modelId)) {
      return
    }
    activatingIds.value = new Set([...activatingIds.value, modelId])
    try {
      await activateModelRegistryEntries({ models: [modelId] })
      let updatedProvider = ''
      for (const [provider, state] of Object.entries(providerModelStates)) {
        let providerChanged = false
        state.items = state.items.map((entry) => {
          if (entry.id !== modelId || entry.available) {
            return entry
          }
          providerChanged = true
          updatedProvider = provider
          return {
            ...entry,
            available: true
          }
        })
        if (providerChanged) {
          break
        }
      }
      if (updatedProvider) {
        items.value = items.value.map((entry) => entry.provider === updatedProvider
          ? { ...entry, availableCount: Math.min(entry.totalCount, entry.availableCount + 1) }
          : entry)
      }
      appStore.showSuccess(t('admin.models.registry.activateSuccess'))
      invalidateModelRegistry()
      modelInventoryStore.invalidate()
      await Promise.allSettled([ensureModelRegistryFresh(true)])
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] activate failed', error)
      appStore.showError(t('admin.models.registry.availabilityFailed'))
    } finally {
      const next = new Set(activatingIds.value)
      next.delete(modelId)
      activatingIds.value = next
    }
  }

  return {
    loading,
    loadingMore,
    items,
    pagination,
    providerGroups,
    hasMoreProviders,
    isActivating,
    loadAll,
    loadMoreProviders,
    refreshAll,
    ensureProviderModels,
    loadMoreProviderModels,
    getProviderModels: (provider: string) => ensureProviderModelsState(provider).items,
    isProviderLoading: (provider: string) => ensureProviderModelsState(provider).loading,
    hasProviderModels: (provider: string) => ensureProviderModelsState(provider).initialized,
    providerHasMoreModels: (provider: string) => {
      const state = ensureProviderModelsState(provider)
      return state.initialized ? state.page < state.pages : false
    },
    activateModel
  }
}

function dedupeProviderGroups(groups: AdminModelRegistryProviderGroup[]) {
  const map = new Map<string, AdminModelRegistryProviderGroup>()
  for (const group of groups) {
    map.set(group.provider, group)
  }
  return [...map.values()]
}

function mergeProviderModels(current: ModelRegistryDetail[], incoming: ModelRegistryDetail[]) {
  if (incoming.length === 0) {
    return current
  }
  const seen = new Set(current.map((item) => item.id))
  const merged = [...current]
  for (const item of incoming) {
    if (seen.has(item.id)) {
      continue
    }
    merged.push(item)
    seen.add(item.id)
  }
  return merged
}
