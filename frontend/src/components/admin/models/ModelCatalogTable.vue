<template>
  <DataTable :columns="columns" :data="items" :loading="loading">
    <template #cell-model="{ row }">
      <div class="space-y-2">
        <ModelCatalogModelLabel
          :model="row.model"
          :display-name="row.display_name"
          :icon-key="row.icon_key"
        />
        <span
          v-if="hasTieredPricing(row.sale_pricing || row.official_pricing)"
          class="inline-flex rounded-full bg-violet-100 px-2 py-0.5 text-xs font-medium text-violet-700 dark:bg-violet-500/15 dark:text-violet-300"
        >
          {{ t('admin.models.tieredPricing') }}
        </span>
      </div>
    </template>

    <template #cell-provider="{ row }">
      <div class="rounded-xl bg-sky-50 px-3 py-2 text-sm font-medium text-sky-800 dark:bg-sky-500/10 dark:text-sky-200">
        {{ row.provider || '-' }}
      </div>
    </template>

    <template #cell-mode="{ row }">
      <div class="rounded-xl bg-indigo-50 px-3 py-2 text-sm font-medium text-indigo-800 dark:bg-indigo-500/10 dark:text-indigo-200">
        {{ formatMode(row.mode) }}
      </div>
    </template>

    <template #cell-default_available="{ row }">
      <div class="rounded-xl bg-emerald-50 px-3 py-2 dark:bg-emerald-500/10">
        <span :class="availabilityClass(row.default_available)">{{ row.default_available ? t('common.available') : t('admin.models.unavailable') }}</span>
        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ formatPlatforms(row.default_platforms) }}</div>
      </div>
    </template>

    <template #cell-pricing_source="{ value }">
      <div class="rounded-xl bg-amber-50 px-3 py-2 dark:bg-amber-500/10">
        <span :class="sourceClass(String(value))">{{ t(`admin.models.sources.${value}`) }}</span>
      </div>
    </template>

    <template #cell-input_cost_per_token="{ row }">
      <ModelCatalogLayeredPriceCell
        :sale-value="row.sale_pricing?.input_cost_per_token"
        :official-value="row.official_pricing?.input_cost_per_token"
        unit="token"
        :exchange-rate="exchangeRate"
      />
    </template>

    <template #cell-output_cost_per_token="{ row }">
      <ModelCatalogLayeredPriceCell
        :sale-value="row.sale_pricing?.output_cost_per_token"
        :official-value="row.official_pricing?.output_cost_per_token"
        unit="token"
        :exchange-rate="exchangeRate"
      />
    </template>

    <template #cell-cache_creation_input_token_cost="{ row }">
      <ModelCatalogLayeredPriceCell
        :sale-value="row.sale_pricing?.cache_creation_input_token_cost"
        :official-value="row.official_pricing?.cache_creation_input_token_cost"
        unit="token"
        :exchange-rate="exchangeRate"
      />
    </template>

    <template #cell-cache_read_input_token_cost="{ row }">
      <ModelCatalogLayeredPriceCell
        :sale-value="row.sale_pricing?.cache_read_input_token_cost"
        :official-value="row.official_pricing?.cache_read_input_token_cost"
        unit="token"
        :exchange-rate="exchangeRate"
      />
    </template>

    <template #cell-output_cost_per_image="{ row }">
      <ModelCatalogLayeredPriceCell
        :sale-value="row.sale_pricing?.output_cost_per_image"
        :official-value="row.official_pricing?.output_cost_per_image"
        unit="image"
        :exchange-rate="exchangeRate"
      />
    </template>

    <template #cell-actions="{ row }">
      <button class="btn btn-secondary btn-sm" @click="emit('inspect', row.model)">
        {{ t('admin.models.viewDetails') }}
      </button>
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
import type { ModelCatalogExchangeRate, ModelCatalogItem } from '@/api/admin/models'
import { hasTieredPricing } from '@/utils/modelCatalogPricing'
import ModelCatalogLayeredPriceCell from './ModelCatalogLayeredPriceCell.vue'
import ModelCatalogModelLabel from './ModelCatalogModelLabel.vue'

withDefaults(
  defineProps<{
    items: ModelCatalogItem[]
    loading: boolean
    exchangeRate?: ModelCatalogExchangeRate | null
  }>(),
  {
    exchangeRate: null
  }
)

const emit = defineEmits<{
  (e: 'inspect', model: string): void
}>()

const { t } = useI18n()

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('admin.models.columns.model') },
  { key: 'provider', label: t('admin.models.columns.provider') },
  { key: 'mode', label: t('admin.models.columns.mode') },
  { key: 'default_available', label: t('admin.models.columns.defaultAvailable') },
  { key: 'pricing_source', label: t('admin.models.columns.pricingSource') },
  { key: 'input_cost_per_token', label: t('admin.models.columns.inputCost') },
  { key: 'output_cost_per_token', label: t('admin.models.columns.outputCost') },
  { key: 'cache_creation_input_token_cost', label: t('admin.models.columns.cacheCreationCost') },
  { key: 'cache_read_input_token_cost', label: t('admin.models.columns.cacheReadCost') },
  { key: 'output_cost_per_image', label: t('admin.models.columns.imageCost') },
  { key: 'actions', label: t('common.actions') }
])

function availabilityClass(available: boolean) {
  return available
    ? 'inline-flex w-fit rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    : 'inline-flex w-fit rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}

function sourceClass(source: string) {
  const classes: Record<string, string> = {
    override: 'inline-flex rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-500/15 dark:text-primary-300',
    dynamic: 'inline-flex rounded-full bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300',
    fallback: 'inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    none: 'inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
  return classes[source] || classes.none
}

function formatPlatforms(platforms?: string[]) {
  if (!platforms || platforms.length === 0) {
    return '-'
  }
  return platforms.join(', ')
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
