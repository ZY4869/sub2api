<template>
  <DataTable :columns="columns" :data="items" :loading="loading">
    <template #cell-model="{ row }">
      <ModelCatalogModelLabel
        :model="row.model"
        :display-name="row.display_name"
        :icon-key="row.icon_key"
        :show-tier-badge="hasTieredPricing(pricingFor(row))"
        :tier-badge-label="t('admin.models.tierBadge')"
      />
    </template>

    <template #cell-provider="{ row }">
      <span class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">
        {{ formatProvider(row.provider) }}
      </span>
    </template>

    <template #cell-mode="{ row }">
      <span class="inline-flex rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300">
        {{ formatMode(row.mode) }}
      </span>
    </template>

    <template #cell-default_available="{ row }">
      <div class="flex flex-wrap gap-2">
        <template v-if="platformLabels(row.default_platforms).length">
          <span
            v-for="platform in platformLabels(row.default_platforms)"
            :key="`${row.model}-${platform}`"
            class="inline-flex rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300"
          >
            {{ platform }}
          </span>
        </template>
        <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
      </div>
    </template>

    <template #cell-pricing_source="{ value }">
      <span :class="sourceClass(String(value))">{{ t(`admin.models.sources.${value}`) }}</span>
    </template>

    <template #cell-input_cost_per_token="{ row }">
      <ModelCatalogPriceValue
        :value="pricingFor(row)?.input_cost_per_token"
        unit="token"
        :exchange-rate="exchangeRate"
        :display-mode="priceDisplayMode"
      />
    </template>

    <template #cell-output_cost_per_token="{ row }">
      <ModelCatalogPriceValue
        :value="pricingFor(row)?.output_cost_per_token"
        unit="token"
        :exchange-rate="exchangeRate"
        :display-mode="priceDisplayMode"
      />
    </template>

    <template #cell-cache_creation_input_token_cost="{ row }">
      <ModelCatalogPriceValue
        :value="pricingFor(row)?.cache_creation_input_token_cost"
        unit="token"
        :exchange-rate="exchangeRate"
        :display-mode="priceDisplayMode"
      />
    </template>

    <template #cell-cache_read_input_token_cost="{ row }">
      <ModelCatalogPriceValue
        :value="pricingFor(row)?.cache_read_input_token_cost"
        unit="token"
        :exchange-rate="exchangeRate"
        :display-mode="priceDisplayMode"
      />
    </template>

    <template #cell-output_cost_per_image="{ row }">
      <ModelCatalogPriceValue
        :value="pricingFor(row)?.output_cost_per_image"
        unit="image"
        :exchange-rate="exchangeRate"
        :display-mode="priceDisplayMode"
      />
    </template>

    <template #cell-actions="{ row }">
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-secondary btn-sm" @click="emit('inspect', row.model)">
          {{ t('admin.models.viewDetails') }}
        </button>
        <button class="btn btn-danger btn-sm" @click="emit('delete', row.model)">
          {{ t('common.delete') }}
        </button>
      </div>
    </template>

    <template #empty>
      <EmptyState :title="t('admin.models.emptyTitle')" :description="t('admin.models.emptyDescription')" />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import type { ModelCatalogExchangeRate, ModelCatalogItem, ModelCatalogPricing } from '@/api/admin/models'
import type { ModelCatalogPricingLayer } from '@/composables/useModelCatalogPage'
import { hasTieredPricing } from '@/utils/modelCatalogPricing'
import {
  formatModelCatalogPlatforms,
  formatModelCatalogProvider,
  type ModelCatalogPriceDisplayMode
} from '@/utils/modelCatalogPresentation'
import ModelCatalogModelLabel from './ModelCatalogModelLabel.vue'
import ModelCatalogPriceValue from './ModelCatalogPriceValue.vue'

const props = withDefaults(
  defineProps<{
    items: ModelCatalogItem[]
    loading: boolean
    pricingLayer: ModelCatalogPricingLayer
    exchangeRate?: ModelCatalogExchangeRate | null
    priceDisplayMode?: ModelCatalogPriceDisplayMode
  }>(),
  {
    exchangeRate: null,
    priceDisplayMode: 'usd'
  }
)

const emit = defineEmits<{
  (e: 'inspect', model: string): void
  (e: 'delete', model: string): void
}>()

const { t } = useI18n()

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('admin.models.columns.model') },
  { key: 'provider', label: t('admin.models.columns.provider') },
  { key: 'mode', label: t('admin.models.columns.mode') },
  { key: 'default_available', label: t('admin.models.columns.defaultProtocol') },
  { key: 'pricing_source', label: t('admin.models.columns.pricingSource') },
  { key: 'input_cost_per_token', label: t('admin.models.columns.inputCost') },
  { key: 'output_cost_per_token', label: t('admin.models.columns.outputCost') },
  { key: 'cache_creation_input_token_cost', label: t('admin.models.columns.cacheCreationCost') },
  { key: 'cache_read_input_token_cost', label: t('admin.models.columns.cacheReadCost') },
  { key: 'output_cost_per_image', label: t('admin.models.columns.imageCost') },
  { key: 'actions', label: t('common.actions') }
])

function pricingFor(row: ModelCatalogItem): ModelCatalogPricing | undefined {
  return props.pricingLayer === 'official' ? row.official_pricing : row.sale_pricing
}

function sourceClass(source: string) {
  const classes: Record<string, string> = {
    override: 'inline-flex rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-500/15 dark:text-primary-300',
    dynamic: 'inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300',
    fallback: 'inline-flex rounded-full bg-amber-100 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    none: 'inline-flex rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
  return classes[source] || classes.none
}

function platformLabels(platforms?: string[]) {
  return formatModelCatalogPlatforms(platforms)
}

function formatProvider(provider?: string) {
  return formatModelCatalogProvider(provider)
}

function formatMode(mode?: string) {
  const labels: Record<string, string> = {
    chat: t('admin.models.modes.chat'),
    image: t('admin.models.modes.image'),
    video: t('admin.models.modes.video'),
    prompt_enhance: t('admin.models.modes.promptEnhance')
  }
  return mode ? labels[mode] || mode : '-'
}
</script>
