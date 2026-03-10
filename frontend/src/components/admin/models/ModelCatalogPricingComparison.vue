<template>
  <div class="overflow-x-auto rounded-2xl border border-gray-200 dark:border-dark-700">
    <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
      <thead class="bg-gray-50 dark:bg-dark-800">
        <tr>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.field') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.upstream') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.officialOverride') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.official') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.saleOverride') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.sale') }}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
        <tr v-for="field in MODEL_CATALOG_PRICING_FIELDS" :key="field.key">
          <td class="bg-violet-50 px-4 py-3 text-sm font-medium text-gray-900 dark:bg-violet-500/10 dark:text-white">{{ t(field.labelKey) }}</td>
          <td class="bg-gray-50 px-4 py-3 dark:bg-dark-900/80">
            <ModelCatalogPriceValue :value="detail.upstream_pricing?.[field.key]" :unit="field.unit" :exchange-rate="exchangeRate" />
          </td>
          <td class="bg-sky-50 px-4 py-3 dark:bg-sky-500/10">
            <ModelCatalogPriceValue :value="detail.official_override_pricing?.[field.key]" :unit="field.unit" :exchange-rate="exchangeRate" />
          </td>
          <td class="bg-blue-50 px-4 py-3 dark:bg-blue-500/10">
            <ModelCatalogPriceValue :value="detail.official_pricing?.[field.key]" :unit="field.unit" :exchange-rate="exchangeRate" />
          </td>
          <td class="bg-emerald-50 px-4 py-3 dark:bg-emerald-500/10">
            <ModelCatalogPriceValue :value="detail.sale_override_pricing?.[field.key]" :unit="field.unit" :exchange-rate="exchangeRate" />
          </td>
          <td class="bg-amber-50 px-4 py-3 dark:bg-amber-500/10">
            <ModelCatalogPriceValue :value="detail.sale_pricing?.[field.key]" :unit="field.unit" :exchange-rate="exchangeRate" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ModelCatalogDetail, ModelCatalogExchangeRate } from '@/api/admin/models'
import { MODEL_CATALOG_PRICING_FIELDS } from '@/utils/modelCatalogPricing'
import ModelCatalogPriceValue from './ModelCatalogPriceValue.vue'

withDefaults(
  defineProps<{
    detail: ModelCatalogDetail
    exchangeRate?: ModelCatalogExchangeRate | null
  }>(),
  {
    exchangeRate: null
  }
)

const { t } = useI18n()
</script>
