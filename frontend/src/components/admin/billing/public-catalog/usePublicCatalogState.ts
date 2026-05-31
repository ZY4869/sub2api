import { computed, ref } from 'vue'
import type { ComposerTranslation } from 'vue-i18n'
import type {
  BillingPublicCatalogAdminEntry,
  BillingPublicCatalogCapacityDiagnosticsSnapshot,
  BillingPublicCatalogEntryDraft,
  BillingPublicCatalogPublishedSummary,
} from '@/api/admin/billing'
import {
  entryKey,
  uniqueSorted,
} from './publicCatalogDraft'
import {
  findDuplicatePublicIDs,
  matchesPublicCatalogFilters,
  sourceLabelKey,
} from './publicCatalogViewHelpers'

export function usePublicCatalogState(t: ComposerTranslation) {
  const loading = ref(false)
  const saving = ref(false)
  const publishing = ref(false)
  const revalidating = ref(false)
  const diagnosticsLoading = ref(false)
  const revalidationAutoEnabled = ref(false)
  const search = ref('')
  const providerFilter = ref('')
  const accountFilter = ref('')
  const batchRatio = ref('120')
  const batchScope = ref('selected')
  const selectedEntries = ref<BillingPublicCatalogEntryDraft[]>([])
  const pageSize = ref(10)
  const draftUpdatedAt = ref('')
  const availableUpdatedAt = ref('')
  const availableSource = ref('')
  const availableEntries = ref<BillingPublicCatalogAdminEntry[]>([])
  const published = ref<BillingPublicCatalogPublishedSummary | null>(null)
  const diagnostics = ref<BillingPublicCatalogCapacityDiagnosticsSnapshot | null>(null)

  const busy = computed(() => loading.value || saving.value || publishing.value || revalidating.value || diagnosticsLoading.value)
  const availableEntryMap = computed(() => new Map(availableEntries.value.map((item) => [entryKey(item), item] as const)))
  const providers = computed(() => uniqueSorted(availableEntries.value.map((item) => item.provider || item.source_protocol || '').filter(Boolean)))
  const accountAliases = computed(() => uniqueSorted(availableEntries.value.map((item) => item.source_alias || '').filter(Boolean)))
  const filteredAvailableEntries = computed(() => {
    const keyword = search.value.trim().toLowerCase()
    return availableEntries.value.filter((item) =>
      matchesPublicCatalogFilters(item, keyword, providerFilter.value, accountFilter.value),
    )
  })
  const duplicatePublicIDs = computed(() => findDuplicatePublicIDs(selectedEntries.value))
  const duplicatePublicIDSet = computed(() => new Set(duplicatePublicIDs.value))
  const availableSourceLabel = computed(() => t(sourceLabelKey(availableSource.value)))

  return {
    loading,
    saving,
    publishing,
    revalidating,
    diagnosticsLoading,
    revalidationAutoEnabled,
    search,
    providerFilter,
    accountFilter,
    batchRatio,
    batchScope,
    selectedEntries,
    pageSize,
    draftUpdatedAt,
    availableUpdatedAt,
    availableSource,
    availableEntries,
    published,
    diagnostics,
    busy,
    availableEntryMap,
    providers,
    accountAliases,
    filteredAvailableEntries,
    duplicatePublicIDs,
    duplicatePublicIDSet,
    availableSourceLabel,
  }
}
