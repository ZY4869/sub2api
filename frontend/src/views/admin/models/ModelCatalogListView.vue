<template>
  <TablePageLayout>
    <template #actions>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          {{ t(pageDescriptionKey) }}
        </div>
        <button class="btn btn-primary" @click="createOpen = true">{{ t('admin.models.catalog.addModel') }}</button>
      </div>
    </template>

    <template #filters>
      <ModelCatalogFilters
        :search="filters.search"
        :provider="filters.provider"
        :mode="filters.mode"
        :availability="filters.availability"
        :pricing-source="filters.pricingSource"
        :provider-options="providerOptions"
        :mode-options="modeOptions"
        :availability-options="availabilityOptions"
        :pricing-source-options="pricingSourceOptions"
        :price-display-mode="priceDisplayMode"
        @update:search="updateFilter('search', $event, false)"
        @update:provider="updateFilter('provider', $event)"
        @update:mode="updateFilter('mode', $event)"
        @update:availability="updateFilter('availability', $event)"
        @update:pricing-source="updateFilter('pricingSource', $event)"
        @update:price-display-mode="priceDisplayMode = $event"
        @search="loadModels(true)"
      />
    </template>

    <template #table>
      <ModelCatalogTable
        :items="items"
        :loading="loading"
        :pricing-layer="pricingLayer"
        :exchange-rate="exchangeRate"
        :price-display-mode="priceDisplayMode"
        @inspect="openDetail"
        @delete="deleteModel"
      />
    </template>
  </TablePageLayout>

  <ModelCatalogAddDialog :show="createOpen" @close="createOpen = false" @confirm="createModel" />

  <ModelCatalogDetailDialog
    :show="dialogOpen"
    :detail="detail"
    :loading="detailLoading"
    :saving="saving"
    :view="pricingLayer"
    :exchange-rate="exchangeRate"
    :price-display-mode="priceDisplayMode"
    @close="closeDetail"
    @save-official="saveOverride"
    @reset-official="resetOverride"
    @save-sale="saveOverride"
    @reset-sale="resetOverride"
    @copy-official-to-sale="copyOfficialToSale"
  />
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import ModelCatalogAddDialog from '@/components/admin/models/ModelCatalogAddDialog.vue'
import ModelCatalogDetailDialog from '@/components/admin/models/ModelCatalogDetailDialog.vue'
import ModelCatalogFilters from '@/components/admin/models/ModelCatalogFilters.vue'
import ModelCatalogTable from '@/components/admin/models/ModelCatalogTable.vue'
import { useExchangeRateStore } from '@/stores/exchangeRate'
import { useModelCatalogPage, type ModelCatalogPricingLayer } from '@/composables/useModelCatalogPage'

const props = defineProps<{
  pricingLayer: ModelCatalogPricingLayer
}>()

const { t } = useI18n()
const exchangeRateStore = useExchangeRateStore()
const exchangeRate = computed(() => exchangeRateStore.exchangeRate)
const pageDescriptionKey = computed(() => `admin.models.pages.${props.pricingLayer}.description`)

const {
  loading,
  detailLoading,
  saving,
  createOpen,
  dialogOpen,
  items,
  detail,
  priceDisplayMode,
  filters,
  providerOptions,
  modeOptions,
  availabilityOptions,
  pricingSourceOptions,
  loadModels,
  openDetail,
  saveOverride,
  resetOverride,
  createModel,
  deleteModel,
  copyOfficialToSale,
  updateFilter,
  closeDetail
} = useModelCatalogPage(props.pricingLayer)

onMounted(() => {
  loadModels()
  exchangeRateStore.fetchExchangeRate()
})
</script>
