import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  activateModelRegistryEntries,
  listModelRegistry,
  type ModelRegistryDetail
} from '@/api/admin/modelRegistry'
import { useAppStore } from '@/stores/app'
import { useModelInventoryStore } from '@/stores'
import { ensureModelRegistryFresh, invalidateModelRegistry } from '@/stores/modelRegistry'
import { formatModelCatalogProvider } from '@/utils/modelCatalogPresentation'

export type AdminModelRegistryProviderGroup = {
  provider: string
  label: string
  models: ModelRegistryDetail[]
  totalCount: number
  availableCount: number
}

export function useAdminModelRegistryProviders() {
  const { t } = useI18n()
  const appStore = useAppStore()
  const modelInventoryStore = useModelInventoryStore()

  const loading = ref(false)
  const activatingIds = ref<Set<string>>(new Set())
  const items = ref<ModelRegistryDetail[]>([])

  const providerGroups = computed<AdminModelRegistryProviderGroup[]>(() => {
    const groups = new Map<string, ModelRegistryDetail[]>()
    for (const entry of items.value) {
      const provider = String(entry.provider || 'unknown').trim().toLowerCase() || 'unknown'
      const current = groups.get(provider) || []
      current.push(entry)
      groups.set(provider, current)
    }

    return [...groups.entries()]
      .map(([provider, models]) => {
        const sortedModels = [...models].sort((left, right) => {
          if (left.available !== right.available) return left.available ? -1 : 1
          return (left.ui_priority - right.ui_priority) || left.id.localeCompare(right.id)
        })
        return {
          provider,
          label: formatModelCatalogProvider(provider),
          models: sortedModels,
          totalCount: sortedModels.length,
          availableCount: sortedModels.filter((m) => m.available).length
        }
      })
      .sort((left, right) => right.totalCount - left.totalCount || left.label.localeCompare(right.label))
  })

  const isActivating = (modelId: string) => activatingIds.value.has(modelId)

  async function loadAll() {
    loading.value = true
    try {
      const collected: ModelRegistryDetail[] = []
      let page = 1
      const pageSize = 200
      let pages = 1

      do {
        const response = await listModelRegistry({
          availability: 'all',
          include_hidden: false,
          include_tombstoned: false,
          page,
          page_size: pageSize
        })
        collected.push(...response.items)
        pages = response.pages || 1
        page += 1
      } while (page <= pages)

      items.value = collected
    } catch (error) {
      console.error('[useAdminModelRegistryProviders] loadAll failed', error)
      appStore.showError(t('admin.models.registry.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  async function refreshAll() {
    await loadAll()
  }

  async function activateModel(modelId: string) {
    if (!modelId || isActivating(modelId)) {
      return
    }
    activatingIds.value = new Set([...activatingIds.value, modelId])
    try {
      await activateModelRegistryEntries({ models: [modelId] })
      items.value = items.value.map((entry) => (entry.id === modelId ? { ...entry, available: true } : entry))
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
    items,
    providerGroups,
    isActivating,
    loadAll,
    refreshAll,
    activateModel
  }
}
