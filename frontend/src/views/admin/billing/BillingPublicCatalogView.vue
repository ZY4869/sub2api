<template>
  <div class="space-y-6">
    <PublicCatalogHeader
      :busy="busy"
      :loading="loading"
      :saving="saving"
      :publishing="publishing"
      :revalidating="revalidating"
      :revalidation-auto-enabled="revalidationAutoEnabled"
      :selected-count="selectedEntries.length"
      :available-count="availableEntries.length"
      :account-alias-count="accountAliases.length"
      :page-size="pageSize"
      :draft-updated-at-label="draftUpdatedAtLabel"
      :available-updated-at-label="availableUpdatedAtLabel"
      :published-count="published?.model_count ?? 0"
      :published-page-size="published?.page_size ?? 10"
      :published-updated-at-label="publishedUpdatedAtLabel"
      :published-at-label="publishedAtLabel"
      :last-revalidated-at-label="lastRevalidatedAtLabel"
      :stale-reason-summary="staleReasonSummary"
      :available-source-label="availableSourceLabel"
      @refresh="loadDraft(true)"
      @save="saveDraftAction"
      @publish="publishAction"
      @revalidate="revalidateAction"
      @update:revalidation-auto-enabled="updateRevalidationAutoEnabled"
      @export="exportDraftSnapshot"
    />

    <PublicCatalogControls
      v-model:search="search"
      v-model:provider-filter="providerFilter"
      v-model:account-filter="accountFilter"
      v-model:page-size="pageSize"
      v-model:batch-ratio="batchRatio"
      v-model:batch-scope="batchScope"
      :providers="providers"
      :account-aliases="accountAliases"
      :filtered-count="filteredAvailableEntries.length"
      :selected-count="selectedEntries.length"
      :duplicate-public-i-ds="duplicatePublicIDs"
      @add-filtered="addFilteredEntries"
      @apply-batch-ratio="applyBatchRatio"
    />

    <PublicCatalogDiagnosticsPanel
      :diagnostics="diagnostics"
      :loading="diagnosticsLoading"
      @refresh="loadDiagnostics"
    />

    <PublicCatalogColumns
      :available-entries="filteredAvailableEntries"
      :selected-entries="selectedCatalogItems"
      :duplicate-public-i-d-set="duplicatePublicIDSet"
      @add="addEntry"
      @clear="clearSelection"
      @remove="removeEntry"
      @move="moveEntry"
      @reorder="reorderEntries"
      @update-entry="updateSelectedEntry"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import PublicCatalogColumns from '@/components/admin/billing/public-catalog/PublicCatalogColumns.vue'
import PublicCatalogControls from '@/components/admin/billing/public-catalog/PublicCatalogControls.vue'
import PublicCatalogDiagnosticsPanel from '@/components/admin/billing/public-catalog/PublicCatalogDiagnosticsPanel.vue'
import PublicCatalogHeader from '@/components/admin/billing/public-catalog/PublicCatalogHeader.vue'
import { useBillingPublicCatalog } from './useBillingPublicCatalog'

const {
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
  availableEntries,
  published,
  diagnostics,
  busy,
  providers,
  accountAliases,
  filteredAvailableEntries,
  selectedCatalogItems,
  duplicatePublicIDs,
  duplicatePublicIDSet,
  availableSourceLabel,
  draftUpdatedAtLabel,
  availableUpdatedAtLabel,
  publishedUpdatedAtLabel,
  publishedAtLabel,
  lastRevalidatedAtLabel,
  staleReasonSummary,
  loadDraft,
  loadDiagnostics,
  loadRevalidationState,
  saveDraftAction,
  publishAction,
  updateRevalidationAutoEnabled,
  revalidateAction,
  addEntry,
  addFilteredEntries,
  removeEntry,
  clearSelection,
  moveEntry,
  reorderEntries,
  updateSelectedEntry,
  applyBatchRatio,
  exportDraftSnapshot,
} = useBillingPublicCatalog()

onMounted(async () => {
  await loadDraft()
  await loadRevalidationState()
})
</script>
