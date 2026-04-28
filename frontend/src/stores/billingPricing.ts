import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  listBillingPricingModels,
  listBillingPricingProviders,
  refreshBillingPricingCatalog,
  type BillingPricingListItem,
  type BillingPricingListParams,
  type BillingPricingProviderGroup,
  type BillingPricingRefreshResult,
  type BillingPricingSortBy,
  type BillingPricingSortOrder,
  type BillingPricingStatus,
} from '@/api/admin/billing'
import type { PaginatedResponse } from '@/types'

const PAGE_SIZE_STORAGE_KEY = 'admin.billing.pricing.page_size'

const providersState = ref<BillingPricingProviderGroup[]>([])
const providersLoadedState = ref(false)
const itemsState = ref<BillingPricingListItem[]>([])
const totalState = ref(0)
const providerModelsState = ref<Record<string, BillingPricingListItem[]>>({})
const listCacheState = ref<Record<string, PaginatedResponse<BillingPricingListItem>>>({})
const providerModelCacheState = ref<Record<string, BillingPricingListItem[]>>({})
const providerModelScopeState = ref('')

const viewModeState = ref<'list' | 'grid'>('list')
const searchState = ref('')
const providerFilterState = ref('')
const modeFilterState = ref('')
const pricingStatusFilterState = ref<BillingPricingStatus | ''>('')
const groupPreviewIdState = ref<number | null>(null)
const sortByState = ref<BillingPricingSortBy>('display_name')
const sortOrderState = ref<BillingPricingSortOrder>('asc')
const pageState = ref(1)
const pageSizeState = ref(readPageSize())
const expandedProviderState = ref('')

function readPageSize(): number {
  if (typeof window === 'undefined') {
    return 20
  }
  const raw = window.localStorage.getItem(PAGE_SIZE_STORAGE_KEY)
  const parsed = Number(raw)
  return parsed === 50 || parsed === 100 ? parsed : 20
}

function buildCurrentListParams(): BillingPricingListParams {
  return {
    search: searchState.value || undefined,
    provider: providerFilterState.value || undefined,
    mode: modeFilterState.value || undefined,
    pricing_status: pricingStatusFilterState.value || undefined,
    group_id: groupPreviewIdState.value || undefined,
    sort_by: sortByState.value,
    sort_order: sortOrderState.value,
    page: pageState.value,
    page_size: pageSizeState.value,
  }
}

function serializeListParams(params: BillingPricingListParams): string {
  return JSON.stringify({
    search: params.search || '',
    provider: params.provider || '',
    mode: params.mode || '',
    pricing_status: params.pricing_status || '',
    group_id: params.group_id || null,
    sort_by: params.sort_by || 'display_name',
    sort_order: params.sort_order || 'asc',
    page: params.page || 1,
    page_size: params.page_size || 20,
  })
}

function buildProviderModelScope(): string {
  return JSON.stringify({
    search: searchState.value || '',
    mode: modeFilterState.value || '',
    pricing_status: pricingStatusFilterState.value || '',
    group_id: groupPreviewIdState.value || null,
    sort_by: sortByState.value,
    sort_order: sortOrderState.value,
  })
}

function setVisibleProviderModelsScope(scope: string) {
  if (providerModelScopeState.value === scope) {
    return
  }
  providerModelScopeState.value = scope
  providerModelsState.value = {}
}

export const useBillingPricingStore = defineStore('billingPricing', () => {
  const providers = computed(() => providersState.value)
  const items = computed(() => itemsState.value)
  const total = computed(() => totalState.value)
  const providerModels = computed(() => providerModelsState.value)
  const viewMode = computed({
    get: () => viewModeState.value,
    set: (value: 'list' | 'grid') => { viewModeState.value = value },
  })
  const search = computed({
    get: () => searchState.value,
    set: (value: string) => { searchState.value = value },
  })
  const providerFilter = computed({
    get: () => providerFilterState.value,
    set: (value: string) => { providerFilterState.value = value },
  })
  const modeFilter = computed({
    get: () => modeFilterState.value,
    set: (value: string) => { modeFilterState.value = value },
  })
  const pricingStatusFilter = computed({
    get: () => pricingStatusFilterState.value,
    set: (value: BillingPricingStatus | '') => { pricingStatusFilterState.value = value },
  })
  const sortBy = computed({
    get: () => sortByState.value,
    set: (value: BillingPricingSortBy) => { sortByState.value = value },
  })
  const groupPreviewId = computed({
    get: () => groupPreviewIdState.value,
    set: (value: number | null) => { groupPreviewIdState.value = value },
  })
  const sortOrder = computed({
    get: () => sortOrderState.value,
    set: (value: BillingPricingSortOrder) => { sortOrderState.value = value },
  })
  const page = computed({
    get: () => pageState.value,
    set: (value: number) => { pageState.value = value },
  })
  const pageSize = computed({
    get: () => pageSizeState.value,
    set: (value: number) => { pageSizeState.value = value },
  })
  const expandedProvider = computed({
    get: () => expandedProviderState.value,
    set: (value: string) => { expandedProviderState.value = value },
  })

  async function loadProviders(force = false): Promise<BillingPricingProviderGroup[]> {
    if (!force && providersLoadedState.value) {
      return providersState.value
    }
    const data = await listBillingPricingProviders()
    providersState.value = data || []
    providersLoadedState.value = true
    return providersState.value
  }

  async function loadModels(force = false): Promise<PaginatedResponse<BillingPricingListItem>> {
    const params = buildCurrentListParams()
    const cacheKey = serializeListParams(params)
    if (!force && listCacheState.value[cacheKey]) {
      const cached = listCacheState.value[cacheKey]
      itemsState.value = cached.items || []
      totalState.value = cached.total || 0
      return cached
    }
    const data = await listBillingPricingModels(params)
    listCacheState.value = {
      ...listCacheState.value,
      [cacheKey]: data,
    }
    itemsState.value = data.items || []
    totalState.value = data.total || 0
    return data
  }

  async function loadProviderModels(provider: string, force = false): Promise<BillingPricingListItem[]> {
    const normalizedProvider = String(provider || '').trim()
    if (!normalizedProvider) {
      return []
    }
    const scopeKey = buildProviderModelScope()
    setVisibleProviderModelsScope(scopeKey)
    const cacheKey = `${scopeKey}::${normalizedProvider}`
    if (!force && providerModelCacheState.value[cacheKey]) {
      providerModelsState.value = {
        ...providerModelsState.value,
        [normalizedProvider]: providerModelCacheState.value[cacheKey],
      }
      return providerModelCacheState.value[cacheKey]
    }
    const data = await listBillingPricingModels({
      search: searchState.value || undefined,
      provider: normalizedProvider,
      mode: modeFilterState.value || undefined,
      pricing_status: pricingStatusFilterState.value || undefined,
      group_id: groupPreviewIdState.value || undefined,
      sort_by: sortByState.value,
      sort_order: sortOrderState.value,
      page: 1,
      page_size: 100,
    })
    const nextItems = data.items || []
    providerModelCacheState.value = {
      ...providerModelCacheState.value,
      [cacheKey]: nextItems,
    }
    providerModelsState.value = {
      ...providerModelsState.value,
      [normalizedProvider]: nextItems,
    }
    return nextItems
  }

  async function refreshCatalog(): Promise<BillingPricingRefreshResult> {
    const result = await refreshBillingPricingCatalog()
    invalidate()
    return result
  }

  function resetPricingStatusForEntry() {
    if (pricingStatusFilterState.value === '') {
      return
    }
    pricingStatusFilterState.value = ''
    pageState.value = 1
    itemsState.value = []
    totalState.value = 0
    providerModelsState.value = {}
    listCacheState.value = {}
    providerModelCacheState.value = {}
    providerModelScopeState.value = ''
  }

  function invalidate() {
    providersState.value = []
    providersLoadedState.value = false
    itemsState.value = []
    totalState.value = 0
    providerModelsState.value = {}
    listCacheState.value = {}
    providerModelCacheState.value = {}
    providerModelScopeState.value = ''
  }

  return {
    providers,
    items,
    total,
    providerModels,
    viewMode,
    search,
    providerFilter,
    modeFilter,
    pricingStatusFilter,
    sortBy,
    groupPreviewId,
    sortOrder,
    page,
    pageSize,
    expandedProvider,
    loadProviders,
    loadModels,
    loadProviderModels,
    refreshCatalog,
    resetPricingStatusForEntry,
    invalidate,
  }
})
