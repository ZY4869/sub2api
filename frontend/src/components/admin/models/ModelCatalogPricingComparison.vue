<template>
  <div class="overflow-x-auto rounded-2xl border border-gray-200 dark:border-dark-700">
    <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
      <thead class="bg-gray-50 dark:bg-dark-800">
        <tr>
          <th v-for="column in columns" :key="column.key" class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
            {{ t(column.labelKey) }}
          </th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
        <tr v-for="field in MODEL_CATALOG_PRICING_FIELDS" :key="field.key">
          <td class="bg-violet-50 px-4 py-3 text-sm font-medium text-gray-900 dark:bg-violet-500/10 dark:text-white">
            {{ t(field.labelKey) }}
          </td>
          <td v-for="column in columns.slice(1)" :key="`${field.key}-${column.key}`" :class="column.className">
            <ModelCatalogPriceValue
              :value="column.resolve(field.key)"
              :unit="field.unit"
              :exchange-rate="exchangeRate"
              :display-mode="priceDisplayMode"
            />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelCatalogDetail, ModelCatalogExchangeRate } from '@/api/admin/models'
import type { ModelCatalogPricingKey } from '@/utils/modelCatalogPricing'
import { MODEL_CATALOG_PRICING_FIELDS } from '@/utils/modelCatalogPricing'
import type { ModelCatalogPricingLayer } from '@/composables/useModelCatalogPage'
import type { ModelCatalogPriceDisplayMode } from '@/utils/modelCatalogPresentation'
import ModelCatalogPriceValue from './ModelCatalogPriceValue.vue'

const props = withDefaults(
  defineProps<{
    detail: ModelCatalogDetail
    view: ModelCatalogPricingLayer
    exchangeRate?: ModelCatalogExchangeRate | null
    priceDisplayMode?: ModelCatalogPriceDisplayMode
  }>(),
  {
    exchangeRate: null,
    priceDisplayMode: 'usd'
  }
)

const { t } = useI18n()

const columns = computed(() => {
  const shared = [{
    key: 'field',
    labelKey: 'admin.models.pricing.field',
    className: '',
    resolve: (_key: ModelCatalogPricingKey) => undefined
  }]

  if (props.view === 'official') {
    return [
      ...shared,
      { key: 'upstream', labelKey: 'admin.models.pricing.upstream', className: 'bg-gray-50 px-4 py-3 dark:bg-dark-900/80', resolve: (key: ModelCatalogPricingKey) => props.detail.upstream_pricing?.[key] },
      { key: 'officialOverride', labelKey: 'admin.models.pricing.officialOverride', className: 'bg-sky-50 px-4 py-3 dark:bg-sky-500/10', resolve: (key: ModelCatalogPricingKey) => props.detail.official_override_pricing?.[key] },
      { key: 'official', labelKey: 'admin.models.pricing.official', className: 'bg-blue-50 px-4 py-3 dark:bg-blue-500/10', resolve: (key: ModelCatalogPricingKey) => props.detail.official_pricing?.[key] }
    ]
  }

  return [
    ...shared,
    { key: 'official', labelKey: 'admin.models.pricing.official', className: 'bg-blue-50 px-4 py-3 dark:bg-blue-500/10', resolve: (key: ModelCatalogPricingKey) => props.detail.official_pricing?.[key] },
    { key: 'saleOverride', labelKey: 'admin.models.pricing.saleOverride', className: 'bg-emerald-50 px-4 py-3 dark:bg-emerald-500/10', resolve: (key: ModelCatalogPricingKey) => props.detail.sale_override_pricing?.[key] },
    { key: 'sale', labelKey: 'admin.models.pricing.sale', className: 'bg-amber-50 px-4 py-3 dark:bg-amber-500/10', resolve: (key: ModelCatalogPricingKey) => props.detail.sale_pricing?.[key] }
  ]
})
</script>
