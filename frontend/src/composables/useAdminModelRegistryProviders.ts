import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  activateModelRegistryEntries,
  deactivateModelRegistryEntries,
  hardDeleteModelRegistryEntries,
  listModelRegistry,
  listModelRegistryProviders,
  moveModelRegistryProvider,
  syncModelRegistryExposures,
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
  search: string
  appliedSearch: string
  exposure: 'all' | 'test'
  status: 'all' | 'stable' | 'beta' | 'deprecated'
  selectedIds: string[]
  requestId: number
}

type MoveProviderPayload = {
  targetProvider: string
  modelIds: string[]
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
  const deactivatingIds = ref<Set<string>>(new Set())
  const deletingIds = ref<Set<string>>(new Set())
  const movingIds = ref<Set<string>>(new Set())
  const syncingTestExposureIds = ref<Set<string>>(new Set())
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
  const isDeactivating = (modelId: string) => deactivatingIds.value.has(modelId)
  const isDeleting = (modelId: string) => deletingIds.value.has(modelId)
  const isMoving = (modelId: string) => movingIds.value.has(modelId)
  const isSyncingTestExposure = (modelId: string) => syncingTestExposureIds.value.has(modelId)
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
      initialized: false,
      search: '',
      appliedSearch: '',
      exposure: 'all',
      status: 'all',
      selectedIds: [],
      requestId: 0
    }
  }

  function ensureProviderModelsState(provider: string): ProviderModelsState {
    const key = getProviderKey(provider)
    if (!providerModelStates[key]) {
      providerModelStates[key] = createProviderModelsState()
    }
    return providerModelStates[key]
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
    const nextPage = reset || !state.initialized ? 1 : state.page + 1
    if (!reset && state.loading) {
      return
    }
    if (!reset && state.initialized && state.pages > 0 && state.page >= state.pages) {
      return
    }
    const requestId = state.requestId + 1
    state.requestId = requestId
    state.loading = true
    if (reset) {
      state.selectedIds = []
    }
    try {
      const response = await listModelRegistry({
        provider: normalizedProvider,
        search: state.appliedSearch || undefined,
        exposure: state.exposure === 'test' ? 'test' : undefined,
        status: state.status === 'all' ? undefined : state.status,
        availability: 'all',
        sort_mode: 'category_latest',
        include_hidden: false,
        include_tombstoned: false,
        page: nextPage,
        page_size: state.pageSize
      })
      if (requestId !== state.requestId) {
        return
      }
      state.items = reset || nextPage === 1
        ? response.items
        : mergeProviderModels(state.items, response.items)
      state.total = response.total
      state.page = response.page
      state.pageSize = response.page_size
      state.pages = response.pages
      state.initialized = true
    } catch (error) {
      if (requestId !== state.requestId) {
        return
      }
      console.error('[useAdminModelRegistryProviders] loadProviderModels failed', error)
      appStore.showError(t('admin.models.registry.loadFailed'))
    } finally {
      if (requestId === state.requestId) {
        state.loading = false
      }
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

  function updateProviderSearch(provider: string, search: string) {
    const state = ensureProviderModelsState(provider)
    const nextSearch = String(search || '').trim()
    if (state.search !== nextSearch) {
      state.selectedIds = []
    }
    state.search = nextSearch
  }

  function setProviderSearch(provider: string, search: string) {
    const state = ensureProviderModelsState(provider)
    updateProviderSearch(provider, search)
    if (state.appliedSearch === state.search && state.initialized) {
      return
    }
    state.appliedSearch = state.search
    void loadProviderModels(provider, true)
  }

  function setProviderExposure(provider: string, exposure: 'all' | 'test') {
    const state = ensureProviderModelsState(provider)
    if (state.exposure === exposure) {
      return
    }
    state.exposure = exposure
    state.selectedIds = []
    if (state.initialized) {
      void loadProviderModels(provider, true)
    }
  }

  function setProviderStatus(provider: string, status: 'all' | 'stable' | 'beta' | 'deprecated') {
    const state = ensureProviderModelsState(provider)
    if (state.status === status) {
      return
    }
    state.status = status
    state.selectedIds = []
    if (state.initialized) {
      void loadProviderModels(provider, true)
    }
  }

  function setProviderSelectedIds(provider: string, modelIds: string[]) {
    const state = ensureProviderModelsState(provider)
    const visibleIds = new Set(state.items.map((item) => item.id))
    state.selectedIds = normalizeModelIds(modelIds).filter((modelId) => visibleIds.has(modelId))
  }

  function toggleProviderModelSelected(provider: string, modelId: string) {
    const state = ensureProviderModelsState(provider)
    if (state.selectedIds.includes(modelId)) {
      state.selectedIds = state.selectedIds.filter((current) => current !== modelId)
      return
    }
    state.selectedIds = [...state.selectedIds, modelId]
  }

  function toggleAllProviderModelsSelected(provider: string, checked: boolean) {
    const state = ensureProviderModelsState(provider)
    state.selectedIds = checked ? state.items.map((item) => item.id) : []
  }

  function clearProviderSelection(provider: string) {
    ensureProviderModelsState(provider).selectedIds = []
  }

  async function activateModel(provider: string, modelId: string) {
    if (!modelId || isActivating(modelId)) {
      return
    }
    activatingIds.value = addPendingId(activatingIds.value, modelId)
    try {
      await activateModelRegistryEntries({ models: [modelId] })
      appStore.showSuccess(t('admin.models.registry.activateSuccess'))
      await refreshProviderAfterMutation(provider)
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] activate failed', error)
      appStore.showError(t('admin.models.registry.availabilityFailed'))
    } finally {
      activatingIds.value = removePendingId(activatingIds.value, modelId)
    }
  }

  async function deactivateModels(provider: string, modelIds: string[]) {
    const normalizedIds = normalizeModelIds(modelIds)
    const pendingIds = normalizedIds.filter((modelId) => !isDeactivating(modelId))
    if (pendingIds.length === 0) {
      return
    }
    deactivatingIds.value = addPendingIds(deactivatingIds.value, pendingIds)
    try {
      await deactivateModelRegistryEntries({ models: pendingIds })
      appStore.showSuccess(
        pendingIds.length > 1
          ? t('admin.models.pages.all.bulk.deactivateSuccess', { count: pendingIds.length })
          : t('admin.models.registry.deactivateSuccess')
      )
      await refreshProviderAfterMutation(provider)
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] deactivate failed', error)
      appStore.showError(t('admin.models.registry.availabilityFailed'))
    } finally {
      deactivatingIds.value = removePendingIds(deactivatingIds.value, pendingIds)
    }
  }

  async function hardDeleteModels(provider: string, modelIds: string[]) {
    const normalizedIds = normalizeModelIds(modelIds)
    const pendingIds = normalizedIds.filter((modelId) => !isDeleting(modelId))
    if (pendingIds.length === 0) {
      return
    }
    deletingIds.value = addPendingIds(deletingIds.value, pendingIds)
    try {
      await hardDeleteModelRegistryEntries({ models: pendingIds })
      appStore.showSuccess(
        pendingIds.length > 1
          ? t('admin.models.pages.all.bulk.hardDeleteSuccess', { count: pendingIds.length })
          : t('admin.models.registry.deleteSuccess')
      )
      await refreshProviderAfterMutation(provider)
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] hard delete failed', error)
      appStore.showError(t('admin.models.registry.deleteFailed'))
    } finally {
      deletingIds.value = removePendingIds(deletingIds.value, pendingIds)
    }
  }

  async function updateTestExposure(provider: string, modelIds: string[], mode: 'add' | 'remove') {
    const normalizedIds = normalizeModelIds(modelIds)
    const pendingIds = normalizedIds.filter((modelId) => !isSyncingTestExposure(modelId))
    if (pendingIds.length === 0) {
      return
    }
    syncingTestExposureIds.value = addPendingIds(syncingTestExposureIds.value, pendingIds)
    try {
      await syncModelRegistryExposures({
        models: pendingIds,
        exposures: ['test'],
        mode
      })
      appStore.showSuccess(
        mode === 'add'
          ? pendingIds.length > 1
            ? t('admin.models.pages.all.bulk.addToTestSuccess', { count: pendingIds.length })
            : t('admin.models.pages.all.addToTestSuccess')
          : pendingIds.length > 1
            ? t('admin.models.pages.all.bulk.removeFromTestSuccess', { count: pendingIds.length })
            : t('admin.models.pages.all.removeFromTestSuccess')
      )
      await refreshProviderAfterMutation(provider)
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] updateTestExposure failed', error)
      appStore.showError(t('admin.models.pages.all.testExposureUpdateFailed'))
    } finally {
      syncingTestExposureIds.value = removePendingIds(syncingTestExposureIds.value, pendingIds)
    }
  }

  async function moveModelsToProvider(
    provider: string,
    targetProviderOrPayload: string | MoveProviderPayload,
    modelIds: string[] = []
  ) {
    const moveInput = resolveMoveProviderInput(targetProviderOrPayload, modelIds)
    const sourceProvider = getProviderKey(provider)
    const nextProvider = getProviderKey(moveInput.targetProvider)
    const pendingIds = normalizeModelIds(moveInput.modelIds).filter((modelId) => !isMoving(modelId))
    if (pendingIds.length === 0 || !nextProvider || nextProvider === sourceProvider) {
      return
    }
    movingIds.value = addPendingIds(movingIds.value, pendingIds)
    try {
      const result = await moveModelRegistryProvider({
        models: pendingIds,
        target_provider: nextProvider
      })

      if (result.failed_count === 0 && result.updated_count > 0) {
        appStore.showSuccess(
          t('admin.models.pages.all.bulk.moveProviderSuccess', {
            count: result.updated_count,
            provider: formatModelCatalogProvider(nextProvider)
          })
        )
      } else if (result.updated_count === 0 && result.failed_count === 0) {
        appStore.showWarning(t('admin.models.pages.all.bulk.moveProviderNoop'))
      } else if (result.updated_count > 0) {
        appStore.showWarning(
          t('admin.models.pages.all.bulk.moveProviderPartial', {
            updated: result.updated_count,
            failed: result.failed_count
          }),
          {
            details: result.failed_models?.map((item) => `${item.model}: ${item.error}`),
            persistent: true
          }
        )
      } else {
        appStore.showError(
          t('admin.models.pages.all.bulk.moveProviderFailed', {
            provider: formatModelCatalogProvider(nextProvider)
          }),
          {
            details: result.failed_models?.map((item) => `${item.model}: ${item.error}`),
            persistent: true
          }
        )
      }

      await refreshProvidersAfterMutation([sourceProvider, nextProvider])
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] move provider failed', error)
      appStore.showError(t('admin.models.pages.all.bulk.moveProviderRequestFailed'))
    } finally {
      movingIds.value = removePendingIds(movingIds.value, pendingIds)
    }
  }

  async function refreshProviderAfterMutation(provider: string) {
    await refreshProvidersAfterMutation([provider])
  }

  async function refreshProvidersAfterMutation(providers: string[]) {
    const normalizedProviders = [...new Set(
      providers
        .map((provider) => getProviderKey(provider))
        .filter((provider) => provider.length > 0)
    )]
    for (const provider of normalizedProviders) {
      clearProviderSelection(provider)
    }
    invalidateModelRegistry()
    modelInventoryStore.invalidate()
    const tasks: Array<Promise<unknown>> = [
      loadProviderSummaries(true),
      ensureModelRegistryFresh(true)
    ]
    for (const provider of normalizedProviders) {
      if (ensureProviderModelsState(provider).initialized) {
        tasks.push(loadProviderModels(provider, true))
      }
    }
    await Promise.allSettled(tasks)
  }

  return {
    loading,
    loadingMore,
    items,
    pagination,
    providerGroups,
    hasMoreProviders,
    isActivating,
    isDeactivating,
    isDeleting,
    isMoving,
    isSyncingTestExposure,
    loadAll,
    loadMoreProviders,
    refreshAll,
    ensureProviderModels,
    loadMoreProviderModels,
    getProviderModels: (provider: string) => provider ? ensureProviderModelsState(provider).items : [],
    getProviderSearch: (provider: string) => provider ? ensureProviderModelsState(provider).search : '',
    getProviderExposure: (provider: string) => provider ? ensureProviderModelsState(provider).exposure : 'all',
    getProviderStatus: (provider: string) => provider ? ensureProviderModelsState(provider).status : 'all',
    getProviderSelectedIds: (provider: string) => provider ? ensureProviderModelsState(provider).selectedIds : [],
    isProviderModelSelected: (provider: string, modelId: string) => provider ? ensureProviderModelsState(provider).selectedIds.includes(modelId) : false,
    isProviderLoading: (provider: string) => provider ? ensureProviderModelsState(provider).loading : false,
    hasProviderModels: (provider: string) => provider ? ensureProviderModelsState(provider).initialized : false,
    providerHasMoreModels: (provider: string) => {
      if (!provider) {
        return false
      }
      const state = ensureProviderModelsState(provider)
      return state.initialized ? state.page < state.pages : false
    },
    updateProviderSearch,
    setProviderSearch,
    setProviderExposure,
    setProviderStatus,
    setProviderSelectedIds,
    toggleProviderModelSelected,
    toggleAllProviderModelsSelected,
    clearProviderSelection,
    activateModel,
    deactivateModels,
    hardDeleteModels,
    moveModelsToProvider,
    addModelsToTest: (provider: string, modelIds: string[]) => updateTestExposure(provider, modelIds, 'add'),
    removeModelsFromTest: (provider: string, modelIds: string[]) => updateTestExposure(provider, modelIds, 'remove')
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

function normalizeModelIds(modelIds: string[]) {
  return [...new Set(
    modelIds
      .map((modelId) => String(modelId || '').trim())
      .filter((modelId) => modelId.length > 0)
  )]
}

function resolveMoveProviderInput(
  targetProviderOrPayload: string | MoveProviderPayload,
  modelIds: string[]
): MoveProviderPayload {
  if (typeof targetProviderOrPayload === 'string') {
    return {
      targetProvider: targetProviderOrPayload,
      modelIds
    }
  }
  return {
    targetProvider: String(targetProviderOrPayload?.targetProvider || ''),
    modelIds: Array.isArray(targetProviderOrPayload?.modelIds) ? targetProviderOrPayload.modelIds : []
  }
}

function addPendingId(current: Set<string>, modelId: string) {
  return addPendingIds(current, [modelId])
}

function addPendingIds(current: Set<string>, modelIds: string[]) {
  return new Set([...current, ...modelIds])
}

function removePendingId(current: Set<string>, modelId: string) {
  return removePendingIds(current, [modelId])
}

function removePendingIds(current: Set<string>, modelIds: string[]) {
  const next = new Set(current)
  for (const modelId of modelIds) {
    next.delete(modelId)
  }
  return next
}
