<template>
  <div class="grid gap-6 xl:grid-cols-2">
    <section class="flex h-[720px] min-h-0 flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm ring-1 ring-slate-900/5 dark:border-dark-700 dark:bg-dark-800 dark:ring-white/5">
      <div class="border-b border-slate-100 px-5 py-4 dark:border-dark-700">
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <h3 class="flex items-center gap-2 text-base font-bold text-slate-900 dark:text-white">
              <Icon name="grid" size="sm" class="text-emerald-500" />
              {{ t('admin.billing.publicCatalog.columns.availableTitle') }}
              <span class="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-500 dark:bg-dark-700 dark:text-slate-300">
                {{ availableEntries.length }}
              </span>
            </h3>
            <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-300">
              {{ t('admin.billing.publicCatalog.columns.availableDescription') }}
            </p>
          </div>
          <span class="shrink-0 text-xs font-medium text-slate-400">
            {{ t('admin.billing.publicCatalog.columns.currentCount', { count: availableEntries.length }) }}
          </span>
        </div>
      </div>

      <div class="min-h-0 flex-1 space-y-3 overflow-y-auto bg-slate-50/60 p-4 dark:bg-dark-900/20">
        <PublicCatalogEntryCard
          v-for="item in availableEntries"
          :key="entryKey(item)"
          :item="item"
          mode="available"
          :already-selected="selectedEntryIDSet.has(entryKey(item))"
          @add="emit('add', item)"
        />
        <div
          v-if="availableEntries.length === 0"
          class="flex h-48 flex-col items-center justify-center rounded-2xl border-2 border-dashed border-slate-200 bg-white/70 px-4 text-center text-sm text-slate-400 dark:border-dark-700 dark:bg-dark-800/60 dark:text-slate-500"
        >
          <Icon name="search" size="lg" class="mb-3 text-slate-300 dark:text-slate-600" />
          {{ t('admin.billing.publicCatalog.columns.emptyAvailable') }}
        </div>
      </div>
    </section>

    <section class="flex h-[720px] min-h-0 flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm ring-1 ring-slate-900/5 dark:border-dark-700 dark:bg-dark-800 dark:ring-white/5">
      <div class="border-b border-slate-100 px-5 py-4 dark:border-dark-700">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 class="flex items-center gap-2 text-base font-bold text-slate-900 dark:text-white">
              <Icon name="externalLink" size="sm" class="text-blue-500" />
              {{ t('admin.billing.publicCatalog.columns.selectedTitle') }}
              <span class="rounded-full bg-blue-50 px-2 py-0.5 text-xs font-semibold text-blue-600 dark:bg-blue-500/15 dark:text-blue-200">
                {{ selectedEntries.length }}
              </span>
            </h3>
            <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-300">
              {{ t('admin.billing.publicCatalog.columns.selectedDescription') }}
            </p>
          </div>
          <button
            type="button"
            class="shrink-0 rounded-lg px-2 py-1 text-xs font-medium text-slate-400 transition hover:bg-rose-50 hover:text-rose-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-rose-500/10 dark:hover:text-rose-200"
            :disabled="selectedEntries.length === 0"
            data-testid="billing-public-catalog-clear"
            @click="emit('clear')"
          >
            {{ t('admin.billing.publicCatalog.columns.clear') }}
          </button>
        </div>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto bg-slate-50/60 p-4 dark:bg-dark-900/20">
        <VueDraggable
          v-model="draggableSelectedEntries"
          :animation="180"
          handle=".public-catalog-drag-handle"
          class="space-y-3"
          data-testid="public-catalog-selected-draggable"
        >
          <PublicCatalogEntryCard
            v-for="(item, index) in draggableSelectedEntries"
            :key="entryKey(item)"
            :item="item"
            mode="selected"
            :index="index"
            :missing="Boolean(item.missing)"
            :duplicate="duplicatePublicIDSet.has(normalizeModelID(item.public_model_id || item.model))"
            :move-up-disabled="index === 0"
            :move-down-disabled="index === draggableSelectedEntries.length - 1"
            @edit="openEditDialog(item)"
            @remove="emit('remove', entryKey(item))"
            @move-up="emit('move', index, -1)"
            @move-down="emit('move', index, 1)"
            @update-public-id="emit('update-entry', entryKey(item), { public_model_id: $event })"
            @update-source-alias="emit('update-entry', entryKey(item), { source_alias: $event })"
            @update-sale-price="emit('update-entry', entryKey(item), { sale_price_display: $event })"
          />
        </VueDraggable>
        <div
          v-if="selectedEntries.length === 0"
          class="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border-2 border-dashed border-slate-200 bg-white/70 px-4 text-center text-sm text-slate-400 dark:border-dark-700 dark:bg-dark-800/60 dark:text-slate-500"
        >
          <div class="mb-4 rounded-full border border-slate-100 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <Icon name="plus" size="lg" class="text-slate-300 dark:text-slate-600" />
          </div>
          <span class="font-medium text-slate-600 dark:text-slate-300">{{ t('admin.billing.publicCatalog.columns.emptySelectedTitle') }}</span>
          <span class="mt-1 text-xs">{{ t('admin.billing.publicCatalog.columns.emptySelected') }}</span>
        </div>
      </div>
    </section>

    <PublicCatalogEntryEditDialog
      :show="Boolean(editingEntry)"
      :item="editingEntry"
      @close="editingEntry = null"
      @save="handleDialogSave"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import type { BillingPublicCatalogAdminEntry, BillingPublicCatalogEntryDraft } from '@/api/admin/billing'
import type { PublicModelCatalogPriceDisplay } from '@/api/meta'
import Icon from '@/components/icons/Icon.vue'
import PublicCatalogEntryCard from './PublicCatalogEntryCard.vue'
import PublicCatalogEntryEditDialog from './PublicCatalogEntryEditDialog.vue'
import {
  entryKey,
  normalizeModelID,
  type SelectedCatalogItem,
} from './publicCatalogDraft'

const props = defineProps<{
  availableEntries: BillingPublicCatalogAdminEntry[]
  selectedEntries: SelectedCatalogItem[]
  duplicatePublicIDSet: Set<string>
}>()

const emit = defineEmits<{
  (e: 'add', item: BillingPublicCatalogAdminEntry): void
  (e: 'clear'): void
  (e: 'remove', entryID: string): void
  (e: 'move', index: number, delta: number): void
  (e: 'reorder', entryIDs: string[]): void
  (e: 'update-entry', entryID: string, patch: Partial<BillingPublicCatalogEntryDraft> & { sale_price_display?: PublicModelCatalogPriceDisplay }): void
}>()

const editingEntry = ref<SelectedCatalogItem | null>(null)
const { t } = useI18n()

const selectedEntryIDSet = computed(() => new Set(props.selectedEntries.map(entryKey)))
const draggableSelectedEntries = computed({
  get: () => props.selectedEntries,
  set: (items: SelectedCatalogItem[]) => {
    emit('reorder', items.map(entryKey))
  },
})

function openEditDialog(item: SelectedCatalogItem) {
  editingEntry.value = item
}

function handleDialogSave(
  entryID: string,
  patch: Partial<BillingPublicCatalogEntryDraft> & { sale_price_display?: PublicModelCatalogPriceDisplay },
) {
  emit('update-entry', entryID, patch)
}
</script>
