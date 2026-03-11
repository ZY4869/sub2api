<template>
  <div class="flex min-w-[6rem] flex-col gap-0.5">
    <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ price.usd }}</span>
    <span v-if="displayMode === 'dual' && price.cny" class="text-xs text-gray-500 dark:text-gray-400">{{ price.cny }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ModelCatalogExchangeRate } from '@/api/admin/models'
import {
  formatModelCatalogPricePair,
  type ModelCatalogPricingUnit
} from '@/utils/modelCatalogPricing'
import type { ModelCatalogPriceDisplayMode } from '@/utils/modelCatalogPresentation'

const props = withDefaults(
  defineProps<{
    value?: number
    unit?: ModelCatalogPricingUnit
    exchangeRate?: ModelCatalogExchangeRate | null
    displayMode?: ModelCatalogPriceDisplayMode
  }>(),
  {
    unit: 'token',
    exchangeRate: null,
    displayMode: 'usd'
  }
)

const price = computed(() => formatModelCatalogPricePair(props.value, props.unit, props.exchangeRate))
</script>
