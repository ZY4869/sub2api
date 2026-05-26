import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getBillingPublicCatalogDraft,
  publishBillingPublicCatalog,
  saveBillingPublicCatalogDraft,
  type BillingPublicCatalogDraft,
  type BillingPublicCatalogAdminEntry,
  type BillingPublicCatalogEntryDraft,
} from '@/api/admin/billing'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import {
  draftEntryToMissingItem,
  entryKey,
  mergeDraftEntryWithItem,
  normalizeAvailableEntries,
  normalizeDraftEntries,
  normalizePageSize,
} from '@/components/admin/billing/public-catalog/publicCatalogDraft'
import {
  addCatalogEntry,
  addFilteredCatalogEntries,
  applyCatalogBatchRatio,
  buildPublicCatalogDraftPayload,
  moveCatalogEntry,
  patchCatalogEntry,
  reorderCatalogEntries,
} from '@/components/admin/billing/public-catalog/publicCatalogSelection'
import {
  downloadPublicCatalogDraft,
  resolveErrorMessage,
} from '@/components/admin/billing/public-catalog/publicCatalogViewHelpers'
import { usePublicCatalogState } from '@/components/admin/billing/public-catalog/usePublicCatalogState'

export function useBillingPublicCatalog() {
  const appStore = useAppStore()
  const { t } = useI18n()
  const state = usePublicCatalogState(t)

  const selectedCatalogItems = computed(() =>
    state.selectedEntries.value.map((entry) => {
      const source = state.availableEntryMap.value.get(entry.entry_id)
      return source ? mergeDraftEntryWithItem(entry, source) : draftEntryToMissingItem(entry)
    }),
  )
  const draftUpdatedAtLabel = computed(() => formatTimestamp(state.draftUpdatedAt.value))
  const availableUpdatedAtLabel = computed(() => formatTimestamp(state.availableUpdatedAt.value))
  const publishedUpdatedAtLabel = computed(() => formatTimestamp(state.published.value?.updated_at))

  async function loadDraft(force = false) {
    state.loading.value = true
    try {
      const payload = await getBillingPublicCatalogDraft({ force })
      state.availableEntries.value = normalizeAvailableEntries(payload.available_entries || payload.available_items || [])
      state.selectedEntries.value = normalizeDraftEntries(payload.draft || {}, state.availableEntries.value)
      state.pageSize.value = normalizePageSize(payload.draft?.page_size || 10)
      state.draftUpdatedAt.value = payload.draft?.updated_at || ''
      state.availableUpdatedAt.value = payload.available_updated_at || ''
      state.availableSource.value = payload.available_source || ''
      state.published.value = payload.published || null
    } catch (error) {
      appStore.showError(resolveErrorMessage(error, t('admin.billing.publicCatalog.messages.loadFailed')))
    } finally {
      state.loading.value = false
    }
  }

  async function saveDraftAction() {
    const payload = buildDraftPayload()
    if (!payload) return
    state.saving.value = true
    try {
      const result = await saveBillingPublicCatalogDraft(payload)
      state.selectedEntries.value = normalizeDraftEntries(result, state.availableEntries.value)
      state.pageSize.value = normalizePageSize(result.page_size || state.pageSize.value)
      state.draftUpdatedAt.value = result.updated_at || state.draftUpdatedAt.value
      appStore.showSuccess(t('admin.billing.publicCatalog.messages.draftSaved'))
    } catch (error) {
      appStore.showError(resolveErrorMessage(error, t('admin.billing.publicCatalog.messages.saveFailed')))
    } finally {
      state.saving.value = false
    }
  }

  async function publishAction() {
    const payload = buildDraftPayload()
    if (!payload) return
    state.publishing.value = true
    try {
      state.published.value = await publishBillingPublicCatalog(payload)
      state.draftUpdatedAt.value = state.published.value?.updated_at || state.draftUpdatedAt.value
      appStore.showSuccess(t('admin.billing.publicCatalog.messages.published'))
    } catch (error) {
      appStore.showError(resolveErrorMessage(error, t('admin.billing.publicCatalog.messages.publishFailed')))
    } finally {
      state.publishing.value = false
    }
  }

  function addEntry(item: BillingPublicCatalogAdminEntry) {
    state.selectedEntries.value = addCatalogEntry(state.selectedEntries.value, item)
  }

  function addFilteredEntries() {
    state.selectedEntries.value = addFilteredCatalogEntries(state.selectedEntries.value, state.filteredAvailableEntries.value)
  }

  function removeEntry(entryID: string) {
    state.selectedEntries.value = state.selectedEntries.value.filter((entry) => entry.entry_id !== entryID)
  }

  function clearSelection() {
    state.selectedEntries.value = []
  }

  function moveEntry(index: number, delta: number) {
    state.selectedEntries.value = moveCatalogEntry(state.selectedEntries.value, index, delta)
  }

  function reorderEntries(entryIDs: string[]) {
    state.selectedEntries.value = reorderCatalogEntries(state.selectedEntries.value, entryIDs)
  }

  function updateSelectedEntry(entryID: string, patch: Partial<BillingPublicCatalogEntryDraft>) {
    state.selectedEntries.value = patchCatalogEntry(state.selectedEntries.value, entryID, patch)
  }

  function applyBatchRatio() {
    const ratio = Number(state.batchRatio.value) / 100
    if (!Number.isFinite(ratio) || ratio < 0) {
      appStore.showError(t('admin.billing.publicCatalog.messages.invalidBatchRatio'))
      return
    }
    const targetEntryIDs = new Set(resolveBatchTargetEntryIDs())
    state.selectedEntries.value = applyCatalogBatchRatio(
      state.selectedEntries.value,
      targetEntryIDs,
      state.availableEntryMap.value,
      ratio,
    )
  }

  function buildDraftPayload(): BillingPublicCatalogDraft | null {
    if (state.duplicatePublicIDs.value.length > 0) {
      appStore.showError(t('admin.billing.publicCatalog.messages.duplicatePublicId', {
        ids: state.duplicatePublicIDs.value.join(t('admin.billing.publicCatalog.controls.listSeparator')),
      }))
      return null
    }
    const unavailableEntries = state.selectedEntries.value.filter((entry) => !state.availableEntryMap.value.has(entry.entry_id))
    if (unavailableEntries.length > 0) {
      appStore.showError(t('admin.billing.publicCatalog.messages.unavailableEntries'))
      return null
    }
    return buildPublicCatalogDraftPayload(
      state.selectedEntries.value,
      state.pageSize.value,
      state.draftUpdatedAt.value,
      state.availableEntryMap.value,
    )
  }

  function exportDraftSnapshot() {
    if (state.selectedEntries.value.length === 0) return
    const payload = buildDraftPayload()
    if (!payload) return
    downloadPublicCatalogDraft(payload)
  }

  function resolveBatchTargetEntryIDs(): string[] {
    if (state.batchScope.value === 'filtered') return state.filteredAvailableEntries.value.map(entryKey)
    if (state.batchScope.value === 'all') return state.availableEntries.value.map(entryKey)
    if (state.batchScope.value.startsWith('source:')) {
      const alias = state.batchScope.value.slice('source:'.length)
      return state.availableEntries.value.filter((item) => item.source_alias === alias).map(entryKey)
    }
    return state.selectedEntries.value.map((entry) => entry.entry_id)
  }

  function formatTimestamp(value?: string | null): string {
    return value ? formatDateTime(value) : t('admin.billing.publicCatalog.messages.unsaved')
  }

  return {
    ...state,
    selectedCatalogItems,
    draftUpdatedAtLabel,
    availableUpdatedAtLabel,
    publishedUpdatedAtLabel,
    loadDraft,
    saveDraftAction,
    publishAction,
    addEntry,
    addFilteredEntries,
    removeEntry,
    clearSelection,
    moveEntry,
    reorderEntries,
    updateSelectedEntry,
    applyBatchRatio,
    exportDraftSnapshot,
  }
}
