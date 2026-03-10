<template>
  <div class="space-y-2 rounded-xl border border-gray-200/70 bg-white/80 px-3 py-2 dark:border-dark-700 dark:bg-dark-900/60">
    <div>
      <p class="text-[11px] font-semibold uppercase tracking-wide text-emerald-600 dark:text-emerald-300">
        {{ t('admin.models.salePrice') }}
      </p>
      <ModelCatalogPriceValue :value="saleValue" :unit="unit" :exchange-rate="exchangeRate" />
    </div>
    <div class="border-t border-gray-200/70 pt-2 dark:border-dark-700">
      <p class="text-[11px] font-semibold uppercase tracking-wide text-sky-600 dark:text-sky-300">
        {{ t('admin.models.officialPrice') }}
      </p>
      <ModelCatalogPriceValue :value="officialValue" :unit="unit" :exchange-rate="exchangeRate" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ModelCatalogExchangeRate } from '@/api/admin/models'
import type { ModelCatalogPricingUnit } from '@/utils/modelCatalogPricing'
import ModelCatalogPriceValue from './ModelCatalogPriceValue.vue'

withDefaults(
  defineProps<{
    saleValue?: number
    officialValue?: number
    unit?: ModelCatalogPricingUnit
    exchangeRate?: ModelCatalogExchangeRate | null
  }>(),
  {
    unit: 'token',
    exchangeRate: null
  }
)

const { t } = useI18n()
</script>
