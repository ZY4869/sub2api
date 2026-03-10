<template>
  <div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700">
    <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
      <thead class="bg-gray-50 dark:bg-dark-800">
        <tr>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.field') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.base') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.override') }}</th>
          <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.pricing.effective') }}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
        <tr v-for="field in MODEL_CATALOG_PRICING_FIELDS" :key="field.key">
          <td class="px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ t(field.labelKey) }}</td>
          <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ renderPrice(detail.base_pricing?.[field.key], field.unit) }}</td>
          <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ renderPrice(detail.override_pricing?.[field.key], field.unit) }}</td>
          <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ renderPrice(detail.effective_pricing?.[field.key], field.unit) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ModelCatalogDetail } from '@/api/admin/models'
import { MODEL_CATALOG_PRICING_FIELDS, formatModelCatalogPrice } from '@/utils/modelCatalogPricing'

defineProps<{
  detail: ModelCatalogDetail
}>()

const { t } = useI18n()

function renderPrice(value?: number, unit: 'token' | 'image' = 'token') {
  return formatModelCatalogPrice(value, unit)
}
</script>
