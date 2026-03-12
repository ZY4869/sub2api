import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { metaAPI } from '@/api/meta'
import {
  generatedModelRegistrySnapshot,
  type ModelRegistryEntry,
  type ModelRegistryPreset,
  type ModelRegistrySnapshot
} from '@/generated/modelRegistry'

const MODEL_REGISTRY_MAX_AGE_MS = 30 * 1000

const snapshotState = ref<ModelRegistrySnapshot>(cloneSnapshot(generatedModelRegistrySnapshot))
const etagState = ref<string | null>(generatedModelRegistrySnapshot.etag || null)
const loadedAtState = ref(0)
const loadingState = ref(false)
let pendingRequest: Promise<ModelRegistrySnapshot> | null = null
let listenersAttached = false

function cloneSnapshot(snapshot: ModelRegistrySnapshot): ModelRegistrySnapshot {
  return {
    etag: snapshot.etag,
    updated_at: snapshot.updated_at,
    models: snapshot.models.map((model) => ({
      ...model,
      platforms: [...model.platforms],
      protocol_ids: [...model.protocol_ids],
      aliases: [...model.aliases],
      pricing_lookup_ids: [...model.pricing_lookup_ids],
      modalities: [...model.modalities],
      capabilities: [...model.capabilities],
      exposed_in: [...model.exposed_in]
    })),
    presets: snapshot.presets.map((preset) => ({ ...preset }))
  }
}

function attachListeners() {
  if (listenersAttached || typeof window === 'undefined') {
    return
  }
  listenersAttached = true

  const revalidate = () => {
    if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
      return
    }
    void ensureModelRegistryFresh()
  }

  window.addEventListener('focus', revalidate)
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', revalidate)
  }
}

export function getModelRegistrySnapshot(): ModelRegistrySnapshot {
  attachListeners()
  if (loadedAtState.value === 0) {
    void fetchModelRegistry()
  } else if (Date.now() - loadedAtState.value > MODEL_REGISTRY_MAX_AGE_MS && !loadingState.value) {
    void fetchModelRegistry()
  }
  return snapshotState.value
}

export async function fetchModelRegistry(force = false): Promise<ModelRegistrySnapshot> {
  attachListeners()
  if (!force && loadingState.value && pendingRequest) {
    return pendingRequest
  }

  pendingRequest = (async () => {
    loadingState.value = true
    try {
      const result = await metaAPI.getModelRegistry(force ? null : etagState.value)
      if (!result.notModified && result.data) {
        snapshotState.value = cloneSnapshot(result.data)
        etagState.value = result.etag || result.data.etag || null
      } else if (result.etag) {
        etagState.value = result.etag
      }
      loadedAtState.value = Date.now()
      return snapshotState.value
    } catch {
      return snapshotState.value
    } finally {
      loadingState.value = false
      pendingRequest = null
    }
  })()

  return pendingRequest
}

export async function ensureModelRegistryFresh(force = false): Promise<ModelRegistrySnapshot> {
  if (force || loadedAtState.value === 0 || Date.now() - loadedAtState.value > MODEL_REGISTRY_MAX_AGE_MS) {
    return fetchModelRegistry(force)
  }
  return snapshotState.value
}

export function invalidateModelRegistry(): void {
  loadedAtState.value = 0
}

export const useModelRegistryStore = defineStore('modelRegistry', () => {
  const snapshot = computed(() => snapshotState.value)
  const models = computed<ModelRegistryEntry[]>(() => snapshotState.value.models)
  const presets = computed<ModelRegistryPreset[]>(() => snapshotState.value.presets)
  const etag = computed(() => etagState.value)
  const loading = computed(() => loadingState.value)
  const hasFreshSnapshot = computed(
    () => loadedAtState.value > 0 && Date.now() - loadedAtState.value < MODEL_REGISTRY_MAX_AGE_MS
  )

  async function fetchRegistry(force = false) {
    return ensureModelRegistryFresh(force)
  }

  async function refetch() {
    return ensureModelRegistryFresh(true)
  }

  function invalidate() {
    invalidateModelRegistry()
  }

  return {
    snapshot,
    models,
    presets,
    etag,
    loading,
    hasFreshSnapshot,
    fetchRegistry,
    refetch,
    invalidate
  }
})
