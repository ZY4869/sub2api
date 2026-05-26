<template>
  <section class="space-y-3">
    <PublicCatalogFilterBar
      :search="search"
      :provider-filter="providerFilter"
      :account-filter="accountFilter"
      :page-size="pageSize"
      :providers="providers"
      :account-aliases="accountAliases"
      :filtered-count="filteredCount"
      @update:search="emit('update:search', $event)"
      @update:provider-filter="emit('update:providerFilter', $event)"
      @update:account-filter="emit('update:accountFilter', $event)"
      @update:page-size="emit('update:pageSize', $event)"
      @add-filtered="emit('add-filtered')"
    />

    <PublicCatalogBatchToolbar
      :batch-ratio="batchRatio"
      :batch-scope="batchScope"
      :account-aliases="accountAliases"
      @update:batch-ratio="emit('update:batchRatio', $event)"
      @update:batch-scope="emit('update:batchScope', $event)"
      @apply-batch-ratio="emit('apply-batch-ratio')"
    />

    <div
      v-if="duplicatePublicIDs.length > 0"
      class="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-100"
    >
      {{ t('admin.billing.publicCatalog.controls.duplicate', { ids: duplicatePublicIDs.join(t('admin.billing.publicCatalog.controls.listSeparator')) }) }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PublicCatalogBatchToolbar from './PublicCatalogBatchToolbar.vue'
import PublicCatalogFilterBar from './PublicCatalogFilterBar.vue'

defineProps<{
  search: string
  providerFilter: string
  accountFilter: string
  pageSize: number
  batchRatio: string
  batchScope: string
  providers: string[]
  accountAliases: string[]
  filteredCount: number
  selectedCount: number
  duplicatePublicIDs: string[]
}>()

const emit = defineEmits<{
  (e: 'update:search', value: string): void
  (e: 'update:providerFilter', value: string): void
  (e: 'update:accountFilter', value: string): void
  (e: 'update:pageSize', value: number): void
  (e: 'update:batchRatio', value: string): void
  (e: 'update:batchScope', value: string): void
  (e: 'add-filtered'): void
  (e: 'apply-batch-ratio'): void
}>()

const { t } = useI18n()
</script>
