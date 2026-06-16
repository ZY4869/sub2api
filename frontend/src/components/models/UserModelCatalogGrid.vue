<template>
  <section class="space-y-4">
    <div
      class="gap-4"
      :class="items.length > 1 ? 'grid md:grid-cols-2 2xl:grid-cols-3' : 'grid'"
      data-testid="user-model-results"
    >
      <PublicModelCard
        v-for="item in displayItems"
        :key="item.raw.model"
        :item="item"
        :provider-label="providerLabel(item.raw)"
        :detail-label="t('ui.modelCatalog.card.detail')"
        :detail-title="t('ui.modelCatalog.detailButton')"
        :copy-title="t('ui.modelCatalog.card.copyModelId')"
        :today-label="t('ui.modelCatalog.card.todaySuccess')"
        :week-label="t('ui.modelCatalog.card.weekSuccess')"
        :latency-label="t('ui.modelCatalog.card.latency')"
        :matrix-label="t('ui.modelCatalog.card.weekMatrix')"
        :pricing-label="t('ui.modelCatalog.card.pricing')"
        :t="t"
        :price-entry-label="priceEntryLabel"
        :format-catalog-price="formatCatalogPrice"
        @copy="copyModelID"
        @open-detail="openDetail"
      />
    </div>

    <div
      v-if="items.length === 0"
      class="rounded-3xl border border-dashed border-slate-300 bg-white/80 px-6 py-12 text-center text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-slate-400"
    >
      {{ t('ui.modelCatalog.emptyPublished') }}
    </div>

    <PublicModelCatalogDetailDialog
      :show="showDetailDialog"
      :model="selectedItem?.model || null"
      :catalog-item="selectedItem"
      @close="showDetailDialog = false"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PublicModelCatalogItem, PublicModelCatalogPriceEntry } from '@/api/meta'
import PublicModelCatalogDetailDialog from '@/components/models/PublicModelCatalogDetailDialog.vue'
import PublicModelCard from '@/components/models/public-catalog/PublicModelCard.vue'
import { useAppStore } from '@/stores/app'
import { formatProviderLabel } from '@/utils/providerLabels'
import {
  buildPublicModelCatalogDisplayItem,
  formatCatalogPrice as renderCatalogPrice,
  priceEntryLabel as renderPriceEntryLabel,
} from '@/utils/publicModelCatalog'

const props = defineProps<{
  items: PublicModelCatalogItem[]
}>()

const { t } = useI18n()
const appStore = useAppStore()
const showDetailDialog = ref(false)
const selectedItem = ref<PublicModelCatalogItem | null>(null)
const displayItems = computed(() => props.items.map(buildPublicModelCatalogDisplayItem))

function providerLabel(item: PublicModelCatalogItem): string {
  return formatProviderLabel(item.provider || item.provider_icon_key || '')
}

function priceEntryLabel(fieldID: string): string {
  return renderPriceEntryLabel(t, fieldID)
}

function formatCatalogPrice(entry: PublicModelCatalogPriceEntry, currency: string): string {
  return renderCatalogPrice(t, entry, currency, null)
}

async function copyModelID(item: PublicModelCatalogItem) {
  const modelID = String(item.model || '').trim()
  if (!modelID) {
    return
  }
  try {
    await navigator.clipboard.writeText(modelID)
    appStore.showSuccess(t('ui.modelCatalog.copySuccess', { model: modelID }))
  } catch {
    appStore.showError(t('ui.modelCatalog.copyFailed'))
  }
}

function openDetail(item: PublicModelCatalogItem) {
  selectedItem.value = item
  showDetailDialog.value = true
}
</script>
