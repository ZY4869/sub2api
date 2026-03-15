<template>
  <div class="flex flex-wrap items-center gap-3">
    <div class="min-w-[220px] flex-1">
      <SearchInput
        :model-value="search"
        :placeholder="t('admin.models.searchPlaceholder')"
        @update:model-value="emit('update:search', $event)"
        @search="emit('search')"
      />
    </div>
    <div class="w-full sm:w-40">
      <Select :model-value="provider" :options="providerOptions" @update:model-value="emit('update:provider', String($event ?? ''))" />
    </div>
    <div class="w-full sm:w-40">
      <Select :model-value="mode" :options="modeOptions" @update:model-value="emit('update:mode', String($event ?? ''))" />
    </div>
    <div class="w-full sm:w-44">
      <Select :model-value="pricingSource" :options="pricingSourceOptions" @update:model-value="emit('update:pricingSource', String($event ?? ''))" />
    </div>

    <div class="inline-flex items-center gap-3 rounded-full border border-gray-200 bg-white px-4 py-2 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
      <span>{{ t('admin.models.filters.onlyAvailable') }}</span>
      <Toggle
        :model-value="availability === 'available'"
        @update:model-value="emit('update:availability', $event ? 'available' : '')"
      />
    </div>
    <div class="ml-auto inline-flex items-center gap-3 rounded-full border border-gray-200 bg-white px-4 py-2 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
      <span>{{ t('admin.models.filters.showCnyReference') }}</span>
      <Toggle
        :model-value="priceDisplayMode === 'dual'"
        @update:model-value="emit('update:priceDisplayMode', $event ? 'dual' : 'usd')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SearchInput from '@/components/common/SearchInput.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { ModelCatalogPriceDisplayMode } from '@/utils/modelCatalogPresentation'

defineProps<{
  search: string
  provider: string
  mode: string
  availability: string
  pricingSource: string
  providerOptions: SelectOption[]
  modeOptions: SelectOption[]
  availabilityOptions: SelectOption[]
  pricingSourceOptions: SelectOption[]
  priceDisplayMode: ModelCatalogPriceDisplayMode
}>()

const emit = defineEmits<{
  (e: 'update:search', value: string): void
  (e: 'update:provider', value: string): void
  (e: 'update:mode', value: string): void
  (e: 'update:availability', value: string): void
  (e: 'update:pricingSource', value: string): void
  (e: 'update:priceDisplayMode', value: ModelCatalogPriceDisplayMode): void
  (e: 'search'): void
}>()

const { t } = useI18n()
</script>
