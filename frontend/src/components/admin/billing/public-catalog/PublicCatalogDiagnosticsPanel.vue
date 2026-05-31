<template>
  <section class="rounded-2xl border border-slate-200 bg-white px-5 py-4 shadow-sm ring-1 ring-slate-900/5 dark:border-dark-700 dark:bg-dark-800 dark:ring-white/5">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <h3 class="flex items-center gap-2 text-base font-bold text-slate-900 dark:text-white">
          <Icon name="chart" size="sm" class="text-sky-500" />
          {{ t('admin.billing.publicCatalog.diagnostics.title') }}
        </h3>
        <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-300">
          {{ t('admin.billing.publicCatalog.diagnostics.description') }}
        </p>
      </div>
      <button
        type="button"
        class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-600 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:bg-dark-700 dark:text-slate-200 dark:hover:bg-dark-600"
        :disabled="loading"
        data-testid="billing-public-catalog-diagnostics-refresh"
        @click="emit('refresh')"
      >
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        {{ loading ? t('admin.billing.publicCatalog.diagnostics.loading') : t('admin.billing.publicCatalog.diagnostics.refresh') }}
      </button>
    </div>

    <div class="mt-4 grid gap-2 sm:grid-cols-4">
      <div
        v-for="item in summaryCards"
        :key="item.key"
        class="rounded-xl border border-slate-200 bg-slate-50/80 px-3 py-2 dark:border-dark-700 dark:bg-dark-900/40"
      >
        <div class="text-[11px] font-medium text-slate-400">{{ item.label }}</div>
        <div class="mt-1 font-mono text-lg font-black text-slate-900 dark:text-white">{{ item.value }}</div>
      </div>
    </div>

    <div v-if="topRestrictions.length > 0" class="mt-4 flex flex-wrap gap-2">
      <span
        v-for="[kind, count] in topRestrictions"
        :key="kind"
        class="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200"
      >
        {{ restrictionLabel(kind) }} · {{ count }}
      </span>
    </div>

    <div class="mt-4 overflow-hidden rounded-xl border border-slate-200 dark:border-dark-700">
      <div class="grid grid-cols-12 gap-3 border-b border-slate-100 bg-slate-50 px-4 py-2 text-[10px] font-black uppercase tracking-widest text-slate-500 dark:border-dark-700 dark:bg-dark-900/60">
        <div class="col-span-4">{{ t('admin.billing.publicCatalog.diagnostics.model') }}</div>
        <div class="col-span-2">{{ t('admin.billing.publicCatalog.diagnostics.availability') }}</div>
        <div class="col-span-3">{{ t('admin.billing.publicCatalog.diagnostics.effectiveLimit') }}</div>
        <div class="col-span-3">{{ t('admin.billing.publicCatalog.diagnostics.sources') }}</div>
      </div>
      <div
        v-for="item in visibleItems"
        :key="item.public_model_id"
        class="grid grid-cols-12 gap-3 border-b border-slate-100 px-4 py-3 text-xs last:border-b-0 dark:border-dark-700"
      >
        <div class="col-span-4 min-w-0">
          <div class="truncate font-semibold text-slate-800 dark:text-slate-100">{{ item.public_model_id }}</div>
          <div class="mt-0.5 truncate text-slate-400">
            {{ item.provider || item.source_protocol || '-' }}
            <span v-if="item.source_account_id">#{{ item.source_account_id }}</span>
          </div>
        </div>
        <div class="col-span-2">
          <span class="rounded-full px-2 py-1 text-[11px] font-semibold" :class="availabilityClass(item.availability)">
            {{ availabilityLabel(item.availability) }}
          </span>
        </div>
        <div class="col-span-3 font-mono text-slate-600 dark:text-slate-300">{{ limitLabel(item) }}</div>
        <div class="col-span-3 min-w-0">
          <div class="truncate text-slate-600 dark:text-slate-300">{{ sourceLabel(item) }}</div>
          <div v-if="restrictionSummary(item)" class="mt-0.5 truncate text-amber-600 dark:text-amber-300">
            {{ restrictionSummary(item) }}
          </div>
        </div>
      </div>
      <div v-if="visibleItems.length === 0" class="px-4 py-8 text-center text-sm text-slate-400">
        {{ t('admin.billing.publicCatalog.diagnostics.empty') }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  BillingPublicCatalogCapacityDiagnosticItem,
  BillingPublicCatalogCapacityDiagnosticsSnapshot,
} from '@/api/admin/billing'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  diagnostics: BillingPublicCatalogCapacityDiagnosticsSnapshot | null
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
}>()

const { t } = useI18n()
const summary = computed(() => props.diagnostics?.summary)
const visibleItems = computed(() => [...(props.diagnostics?.items || [])].slice(0, 8))
const summaryCards = computed(() => [
  { key: 'total', label: t('admin.billing.publicCatalog.diagnostics.total'), value: summary.value?.model_count || 0 },
  { key: 'available', label: t('admin.billing.publicCatalog.diagnostics.available'), value: summary.value?.available_count || 0 },
  { key: 'limited', label: t('admin.billing.publicCatalog.diagnostics.limited'), value: summary.value?.limited_count || 0 },
  { key: 'unschedulable', label: t('admin.billing.publicCatalog.diagnostics.unschedulable'), value: summary.value?.unschedulable_count || 0 },
])
const topRestrictions = computed(() =>
  Object.entries(summary.value?.restriction_counts || {})
    .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0]))
    .slice(0, 6),
)

function availabilityLabel(value: string): string {
  const key = `admin.billing.publicCatalog.diagnostics.availabilityLabels.${value}`
  const label = t(key)
  return label === key ? value : label
}

function restrictionLabel(value: string): string {
  const key = `admin.billing.publicCatalog.diagnostics.restrictions.${value}`
  const label = t(key)
  return label === key ? value.replace(/_/g, ' ') : label
}

function limitLabel(item: BillingPublicCatalogCapacityDiagnosticItem): string {
  const limit = item.effective_rate_limit
  if (!limit) return '-'
  return [
    limit.rpm != null ? `RPM ${limit.rpm}` : '',
    limit.tpm != null ? `TPM ${limit.tpm}` : '',
    limit.rpd != null ? `RPD ${limit.rpd}` : '',
  ].filter(Boolean).join(' / ') || '-'
}

function sourceLabel(item: BillingPublicCatalogCapacityDiagnosticItem): string {
  return (item.sources || []).map((source) => source.source).filter(Boolean).join(' / ') || '-'
}

function restrictionSummary(item: BillingPublicCatalogCapacityDiagnosticItem): string {
  return (item.restrictions || []).map((restriction) => restrictionLabel(restriction.kind)).join(' / ')
}

function availabilityClass(value: string): string {
  switch (value) {
    case 'available':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200'
    case 'limited':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200'
    case 'unschedulable':
      return 'bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-200'
    default:
      return 'bg-slate-100 text-slate-500 dark:bg-dark-700 dark:text-slate-300'
  }
}
</script>
