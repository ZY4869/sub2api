<template>
  <DataTable :columns="columns" :data="items" :loading="loading">
    <template #cell-model="{ row }">
      <span class="font-medium text-gray-900 dark:text-white">{{ row.model }}</span>
    </template>

    <template #cell-provider="{ row }">{{ row.provider || '-' }}</template>

    <template #cell-mode="{ row }">{{ formatMode(row.mode) }}</template>

    <template #cell-default_available="{ row }">
      <div class="flex flex-col gap-1">
        <span :class="availabilityClass(row.default_available)">{{ row.default_available ? t('common.available') : t('admin.models.unavailable') }}</span>
        <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatPlatforms(row.default_platforms) }}</span>
      </div>
    </template>

    <template #cell-pricing_source="{ value }">
      <span :class="sourceClass(String(value))">{{ t(`admin.models.sources.${value}`) }}</span>
    </template>

    <template #cell-input_cost_per_token="{ row }">{{ formatPrice(row, 'input_cost_per_token') }}</template>
    <template #cell-output_cost_per_token="{ row }">{{ formatPrice(row, 'output_cost_per_token') }}</template>
    <template #cell-cache_creation_input_token_cost="{ row }">{{ formatPrice(row, 'cache_creation_input_token_cost') }}</template>
    <template #cell-cache_read_input_token_cost="{ row }">{{ formatPrice(row, 'cache_read_input_token_cost') }}</template>
    <template #cell-output_cost_per_image="{ row }">{{ formatPrice(row, 'output_cost_per_image', 'image') }}</template>

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
import type { ModelCatalogItem, ModelCatalogPricing } from '@/api/admin/models'
import { formatModelCatalogPrice } from '@/utils/modelCatalogPricing'

defineProps<{
  items: ModelCatalogItem[]
  loading: boolean
}>()

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

function formatPrice(
  row: ModelCatalogItem,
  key: keyof ModelCatalogPricing,
  unit: 'token' | 'image' = 'token'
) {
  return formatModelCatalogPrice(row.effective_pricing?.[key], unit)
}

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
