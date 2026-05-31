<template>
  <section class="rounded-2xl border border-slate-200 bg-white px-5 py-4 shadow-sm ring-1 ring-slate-900/5 dark:border-dark-700 dark:bg-dark-800 dark:ring-white/5">
    <div class="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
      <div class="min-w-0">
        <div class="inline-flex items-center gap-2 rounded-full bg-emerald-50 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-emerald-700 ring-1 ring-emerald-100 dark:bg-emerald-500/10 dark:text-emerald-200 dark:ring-emerald-500/20">
          <Icon name="sparkles" size="xs" />
          {{ t('admin.billing.publicCatalog.header.eyebrow') }}
        </div>
        <h2 class="mt-3 text-2xl font-bold text-slate-900 dark:text-white">
          {{ t('admin.billing.publicCatalog.header.title') }}
        </h2>
        <p class="mt-2 max-w-4xl text-sm leading-6 text-slate-500 dark:text-slate-300">
          {{ t('admin.billing.publicCatalog.header.description') }}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-600 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:bg-dark-700 dark:text-slate-200 dark:hover:bg-dark-600"
          :disabled="busy"
          data-testid="billing-public-catalog-revalidate"
          @click="emit('revalidate')"
        >
          <Icon name="sync" size="sm" :class="revalidating ? 'animate-spin' : ''" />
          {{ revalidating ? t('admin.billing.publicCatalog.header.revalidating') : t('admin.billing.publicCatalog.header.revalidate') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-600 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:bg-dark-700 dark:text-slate-200 dark:hover:bg-dark-600"
          :disabled="busy"
          @click="emit('update:revalidationAutoEnabled', !revalidationAutoEnabled)"
        >
          <Icon :name="revalidationAutoEnabled ? 'checkCircle' : 'xCircle'" size="sm" />
          {{ t('admin.billing.publicCatalog.header.autoRevalidation') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-600 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:bg-dark-700 dark:text-slate-200 dark:hover:bg-dark-600"
          :disabled="busy"
          @click="emit('refresh')"
        >
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ loading ? t('admin.billing.publicCatalog.header.loading') : t('admin.billing.publicCatalog.header.refresh') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-600 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:bg-dark-700 dark:text-slate-200 dark:hover:bg-dark-600"
          :disabled="selectedCount === 0"
          data-testid="billing-public-catalog-export"
          @click="emit('export')"
        >
          <Icon name="download" size="sm" />
          {{ t('admin.billing.publicCatalog.controls.export') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-emerald-200 bg-white px-3 py-2 text-sm font-medium text-emerald-700 shadow-sm transition hover:border-emerald-300 hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-500/30 dark:bg-dark-700 dark:text-emerald-200 dark:hover:bg-emerald-500/10"
          :disabled="busy"
          data-testid="billing-public-catalog-save"
          @click="emit('save')"
        >
          <Icon name="check" size="sm" />
          {{ saving ? t('admin.billing.publicCatalog.header.saving') : t('admin.billing.publicCatalog.header.save') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg bg-slate-900 px-3 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-300 disabled:text-slate-500 dark:bg-emerald-600 dark:hover:bg-emerald-500 dark:disabled:bg-dark-600"
          :disabled="busy || selectedCount === 0"
          data-testid="billing-public-catalog-publish"
          @click="emit('publish')"
        >
          <Icon name="upload" size="sm" />
          {{ publishing ? t('admin.billing.publicCatalog.header.publishing') : t('admin.billing.publicCatalog.header.publish') }}
        </button>
      </div>
    </div>

    <div class="mt-4 flex flex-wrap gap-2">
      <div
        v-for="item in statItems"
        :key="item.title"
        class="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-slate-50/80 px-3 py-2 text-xs text-slate-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-slate-300"
      >
        <span class="font-semibold text-slate-800 dark:text-white">{{ item.value }}</span>
        <span>{{ item.title }}</span>
      </div>
    </div>

    <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-slate-400">
      <span>{{ t('admin.billing.publicCatalog.header.draftSavedAt', { time: draftUpdatedAtLabel }) }}</span>
      <span>{{ t('admin.billing.publicCatalog.header.availableUpdatedAt', { time: availableUpdatedAtLabel }) }}</span>
      <span>{{ t('admin.billing.publicCatalog.header.publishedAt', { time: publishedAtLabel }) }}</span>
      <span>{{ t('admin.billing.publicCatalog.header.lastRevalidatedAt', { time: lastRevalidatedAtLabel }) }}</span>
      <span>{{ t('admin.billing.publicCatalog.header.revalidationAutoState', { state: revalidationAutoStateLabel }) }}</span>
      <span v-if="staleReasonSummary" class="text-amber-600 dark:text-amber-300">
        {{ t('admin.billing.publicCatalog.header.staleReason', { reason: staleReasonSummary }) }}
      </span>
      <span>{{ t('admin.billing.publicCatalog.header.sourceDescription', { source: availableSourceLabel }) }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  busy: boolean
  loading: boolean
  saving: boolean
  publishing: boolean
  revalidating: boolean
  revalidationAutoEnabled: boolean
  selectedCount: number
  availableCount: number
  accountAliasCount: number
  pageSize: number
  draftUpdatedAtLabel: string
  availableUpdatedAtLabel: string
  publishedCount: number
  publishedPageSize: number
  publishedUpdatedAtLabel: string
  publishedAtLabel: string
  lastRevalidatedAtLabel: string
  staleReasonSummary: string
  availableSourceLabel: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'save'): void
  (e: 'publish'): void
  (e: 'revalidate'): void
  (e: 'update:revalidationAutoEnabled', value: boolean): void
  (e: 'export'): void
}>()

const { t } = useI18n()

const statItems = computed(() => [
  {
    title: t('admin.billing.publicCatalog.header.draftTitle'),
    value: t('admin.billing.publicCatalog.header.draftValue', { count: props.selectedCount }),
  },
  {
    title: t('admin.billing.publicCatalog.header.availableTitle'),
    value: t('admin.billing.publicCatalog.header.availableValue', { count: props.availableCount }),
  },
  {
    title: t('admin.billing.publicCatalog.header.accountAliasTitle'),
    value: t('admin.billing.publicCatalog.header.accountAliasValue', { count: props.accountAliasCount }),
  },
  {
    title: t('admin.billing.publicCatalog.header.pageSizeTitle'),
    value: t('admin.billing.publicCatalog.header.pageSizeValue', { count: props.pageSize }),
  },
  {
    title: t('admin.billing.publicCatalog.header.publishedTitle'),
    value: t('admin.billing.publicCatalog.header.publishedValue', { count: props.publishedCount }),
  },
  {
    title: t('admin.billing.publicCatalog.header.publishedPageSizeTitle'),
    value: t('admin.billing.publicCatalog.header.pageSizeValue', { count: props.publishedPageSize }),
  },
])

const revalidationAutoStateLabel = computed(() =>
  props.revalidationAutoEnabled
    ? t('admin.billing.publicCatalog.header.autoEnabled')
    : t('admin.billing.publicCatalog.header.autoDisabled'),
)
</script>
