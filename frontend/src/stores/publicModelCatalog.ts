import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  getModelCatalog,
  getModelCatalogStatus,
  getUSDCNYExchangeRate,
  type PublicModelCatalogItem,
  type PublicModelCatalogSnapshot,
  type PublicModelCatalogStatusItem,
  type PublicModelCatalogStatusSnapshot
} from '@/api/meta'
const PUBLIC_MODEL_CATALOG_STORAGE_KEY = 'public-model-catalog:snapshot'

interface PersistedPublicModelCatalogState {
  snapshot: PublicModelCatalogSnapshot | null
  etag: string | null
  loadedAt: number
  usdToCnyRate: number | null
  exchangeRateLoadedAt: number
}

const snapshotState = ref<PublicModelCatalogSnapshot | null>(null)
const statusSnapshotState = ref<PublicModelCatalogStatusSnapshot | null>(null)
const etagState = ref<string | null>(null)
const loadedAtState = ref(0)
const loadingState = ref(false)
const statusLoadingState = ref(false)
const hardErrorState = ref('')
const statusErrorState = ref('')
const softStaleState = ref(false)
const exchangeRateWarningState = ref(false)
const usdToCnyRateState = ref<number | null>(null)
const exchangeRateLoadedAtState = ref(0)

let hydrated = false
let pendingRequest: Promise<PublicModelCatalogSnapshot | null> | null = null
let pendingStatusRequest: Promise<PublicModelCatalogStatusSnapshot | null> | null = null

function cloneSnapshot(snapshot: PublicModelCatalogSnapshot | null): PublicModelCatalogSnapshot | null {
  if (!snapshot) {
    return null
  }
  return {
    etag: snapshot.etag,
    updated_at: snapshot.updated_at,
    page_size: snapshot.page_size,
    catalog_source: snapshot.catalog_source,
    items: snapshot.items.map(cloneItem)
  }
}

function cloneItem(item: PublicModelCatalogItem): PublicModelCatalogItem {
  return {
    ...item,
    request_protocols: [...(item.request_protocols || [])],
    modalities: [...(item.modalities || [])],
    capabilities: [...(item.capabilities || [])],
    price_display: {
      primary: item.price_display.primary.map((entry) => ({ ...entry })),
      secondary: item.price_display.secondary?.map((entry) => ({ ...entry }))
    },
    multiplier_summary: { ...item.multiplier_summary }
  }
}

function cloneStatusSnapshot(snapshot: PublicModelCatalogStatusSnapshot | null): PublicModelCatalogStatusSnapshot | null {
  if (!snapshot) {
    return null
  }
  return {
    updated_at: snapshot.updated_at,
    items: snapshot.items.map((item) => ({
      ...item,
      daily: [...(item.daily || [])],
      trend: [...(item.trend || [])],
      rate_limit: item.rate_limit ? { ...item.rate_limit } : undefined
    }))
  }
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (
    typeof error === 'object' &&
    error &&
    'message' in error &&
    typeof (error as { message?: unknown }).message === 'string'
  ) {
    return String((error as { message: string }).message)
  }
  return fallback
}

function hydrateFromStorage() {
  if (hydrated || typeof window === 'undefined') {
    return
  }
  hydrated = true

  try {
    const raw = window.localStorage.getItem(PUBLIC_MODEL_CATALOG_STORAGE_KEY)
    if (!raw) {
      return
    }
    const parsed = JSON.parse(raw) as Partial<PersistedPublicModelCatalogState>
    snapshotState.value = cloneSnapshot(parsed.snapshot || null)
    etagState.value = typeof parsed.etag === 'string' ? parsed.etag : null
    loadedAtState.value = Number(parsed.loadedAt || 0)
    usdToCnyRateState.value =
      parsed.usdToCnyRate == null ? null : Number(parsed.usdToCnyRate)
    exchangeRateLoadedAtState.value = Number(parsed.exchangeRateLoadedAt || 0)
  } catch {
    // Ignore invalid cache payloads and keep using live fetches.
  }
}

function persistToStorage() {
  if (typeof window === 'undefined') {
    return
  }
  try {
    const payload: PersistedPublicModelCatalogState = {
      snapshot: cloneSnapshot(snapshotState.value),
      etag: etagState.value,
      loadedAt: loadedAtState.value,
      usdToCnyRate: usdToCnyRateState.value,
      exchangeRateLoadedAt: exchangeRateLoadedAtState.value
    }
    window.localStorage.setItem(PUBLIC_MODEL_CATALOG_STORAGE_KEY, JSON.stringify(payload))
  } catch {
    // Ignore localStorage failures so the catalog remains usable.
  }
}

function catalogHasCNYPricing() {
  return Boolean(snapshotState.value?.items.some((item) => item.currency === 'CNY'))
}

function shouldLoadExchangeRate(force = false) {
  if (force) {
    return catalogHasCNYPricing()
  }
  if (!catalogHasCNYPricing()) {
    return false
  }
  return usdToCnyRateState.value == null
}

async function loadExchangeRate(force = false) {
  if (!shouldLoadExchangeRate(force)) {
    exchangeRateWarningState.value = false
    return usdToCnyRateState.value
  }

  try {
    const rate = await getUSDCNYExchangeRate(force)
    usdToCnyRateState.value = Number(rate.rate)
    exchangeRateLoadedAtState.value = Date.now()
    exchangeRateWarningState.value = false
    persistToStorage()
  } catch {
    if (snapshotState.value) {
      exchangeRateWarningState.value = true
    }
  }

  return usdToCnyRateState.value
}

export async function fetchPublicModelCatalogStatus(force = false): Promise<PublicModelCatalogStatusSnapshot | null> {
  hydrateFromStorage()
  if (!force && statusLoadingState.value && pendingStatusRequest) {
    return pendingStatusRequest
  }

  pendingStatusRequest = (async () => {
    statusLoadingState.value = true
    statusErrorState.value = ''
    try {
      const snapshot = await getModelCatalogStatus()
      statusSnapshotState.value = cloneStatusSnapshot(snapshot)
    } catch (error) {
      statusErrorState.value = resolveErrorMessage(error, 'Model health status is temporarily unavailable.')
      return null
    }
    return statusSnapshotState.value
  })()

  try {
    return await pendingStatusRequest
  } finally {
    statusLoadingState.value = false
    pendingStatusRequest = null
  }
}

export async function fetchPublicModelCatalog(force = false): Promise<PublicModelCatalogSnapshot | null> {
  hydrateFromStorage()
  if (!force && loadingState.value && pendingRequest) {
    return pendingRequest
  }

  pendingRequest = (async () => {
    loadingState.value = true
    hardErrorState.value = ''
    try {
      const result = await getModelCatalog(etagState.value)
      if (!result.notModified && result.data) {
        snapshotState.value = cloneSnapshot(result.data)
        etagState.value = result.etag || result.data.etag || null
      } else if (result.etag) {
        etagState.value = result.etag
      }

      if (snapshotState.value) {
        loadedAtState.value = Date.now()
        softStaleState.value = false
        persistToStorage()
      }
    } catch (error) {
      const message = resolveErrorMessage(error, 'Network error. Please check your connection.')
      if (snapshotState.value) {
        softStaleState.value = true
      } else {
        hardErrorState.value = message
        return null
      }
    }

    if (snapshotState.value) {
      await loadExchangeRate(force)
      void fetchPublicModelCatalogStatus(force)
    }

    return snapshotState.value
  })()

  try {
    return await pendingRequest
  } finally {
    loadingState.value = false
    pendingRequest = null
  }
}

export async function ensurePublicModelCatalogReady(force = false): Promise<PublicModelCatalogSnapshot | null> {
  hydrateFromStorage()

  if (force || !snapshotState.value) {
    return fetchPublicModelCatalog(force)
  }

  hardErrorState.value = ''
  if (shouldLoadExchangeRate()) {
    void loadExchangeRate()
  }
  void fetchPublicModelCatalogStatus(false)
  void fetchPublicModelCatalog(false)
  return snapshotState.value
}

export function invalidatePublicModelCatalog() {
  loadedAtState.value = 0
}

export function resetPublicModelCatalogStoreForTests() {
  snapshotState.value = null
  statusSnapshotState.value = null
  etagState.value = null
  loadedAtState.value = 0
  loadingState.value = false
  statusLoadingState.value = false
  hardErrorState.value = ''
  statusErrorState.value = ''
  softStaleState.value = false
  exchangeRateWarningState.value = false
  usdToCnyRateState.value = null
  exchangeRateLoadedAtState.value = 0
  hydrated = false
  pendingRequest = null
  pendingStatusRequest = null
}

export const usePublicModelCatalogStore = defineStore('publicModelCatalog', () => {
  const snapshot = computed(() => snapshotState.value)
  const statusSnapshot = computed(() => statusSnapshotState.value)
  const statusByModel = computed<Record<string, PublicModelCatalogStatusItem>>(() => {
    const items: Record<string, PublicModelCatalogStatusItem> = {}
    for (const item of statusSnapshotState.value?.items || []) {
      const model = String(item.model || '').trim()
      if (model) {
        items[model] = item
      }
    }
    return items
  })
  const etag = computed(() => etagState.value)
  const loadedAt = computed(() => loadedAtState.value)
  const loading = computed(() => loadingState.value)
  const statusLoading = computed(() => statusLoadingState.value)
  const hardError = computed(() => hardErrorState.value)
  const statusError = computed(() => statusErrorState.value)
  const softStale = computed(() => softStaleState.value)
  const exchangeRateWarning = computed(() => exchangeRateWarningState.value)
  const usdToCnyRate = computed(() => usdToCnyRateState.value)
  const hasSnapshot = computed(() => Boolean(snapshotState.value))
  const hasFreshSnapshot = computed(() => Boolean(snapshotState.value))

  async function initialize(force = false) {
    return ensurePublicModelCatalogReady(force)
  }

  async function refresh() {
    return fetchPublicModelCatalog(true)
  }

  async function fetchCatalog(force = false) {
    return fetchPublicModelCatalog(force)
  }

  async function fetchStatus(force = false) {
    return fetchPublicModelCatalogStatus(force)
  }

  function invalidate() {
    invalidatePublicModelCatalog()
  }

  return {
    snapshot,
    statusSnapshot,
    statusByModel,
    etag,
    loadedAt,
    loading,
    statusLoading,
    hardError,
    statusError,
    softStale,
    exchangeRateWarning,
    usdToCnyRate,
    hasSnapshot,
    hasFreshSnapshot,
    initialize,
    refresh,
    fetchCatalog,
    fetchStatus,
    invalidate
  }
})
