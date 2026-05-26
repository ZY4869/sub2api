<template>
  <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm ring-1 ring-slate-900/5 dark:border-dark-700 dark:bg-dark-800 dark:ring-white/5">
    <div class="grid gap-3 xl:grid-cols-[minmax(260px,1fr)_auto] xl:items-center">
      <label class="relative block">
        <span class="sr-only">{{ t('admin.billing.publicCatalog.controls.search') }}</span>
        <Icon
          name="search"
          size="sm"
          class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        />
        <input
          :value="search"
          type="search"
          class="w-full rounded-xl border border-slate-200 bg-slate-50 py-2.5 pl-9 pr-3 text-sm text-slate-800 outline-none transition placeholder:text-slate-400 hover:border-slate-300 focus:border-emerald-500 focus:bg-white focus:ring-2 focus:ring-emerald-500/20 dark:border-dark-600 dark:bg-dark-900/50 dark:text-slate-100"
          :placeholder="t('admin.billing.publicCatalog.controls.searchPlaceholder')"
          data-testid="billing-public-catalog-search"
          @input="emit('update:search', ($event.target as HTMLInputElement).value.trim())"
        />
      </label>

      <div class="flex flex-wrap items-center gap-2">
        <label class="inline-flex items-center gap-2 text-xs font-medium text-slate-500 dark:text-slate-300">
          <span>{{ t('admin.billing.publicCatalog.controls.accountSource') }}</span>
          <select
            :value="accountFilter"
            class="h-9 rounded-lg border border-slate-200 bg-white px-2 text-sm text-slate-700 outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20 dark:border-dark-600 dark:bg-dark-900 dark:text-slate-100"
            data-testid="billing-public-catalog-account-filter"
            @change="emit('update:accountFilter', ($event.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t('admin.billing.publicCatalog.controls.allSources') }}</option>
            <option v-for="alias in accountAliases" :key="alias" :value="alias">{{ alias }}</option>
          </select>
        </label>
        <label class="inline-flex items-center gap-2 text-xs font-medium text-slate-500 dark:text-slate-300">
          <span>{{ t('admin.billing.publicCatalog.controls.pageSize') }}</span>
          <input
            :value="pageSize"
            type="number"
            min="1"
            max="100"
            class="h-9 w-20 rounded-lg border border-slate-200 bg-white px-2 text-right text-sm text-slate-700 outline-none focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20 dark:border-dark-600 dark:bg-dark-900 dark:text-slate-100"
            data-testid="billing-public-catalog-page-size"
            @input="emit('update:pageSize', Number(($event.target as HTMLInputElement).value))"
          />
        </label>
        <button
          type="button"
          class="inline-flex h-9 items-center gap-1.5 rounded-lg border border-emerald-200 bg-white px-3 text-sm font-medium text-emerald-700 shadow-sm transition hover:border-emerald-300 hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-emerald-500/30 dark:bg-dark-700 dark:text-emerald-200 dark:hover:bg-emerald-500/10"
          :disabled="filteredCount === 0"
          @click="emit('add-filtered')"
        >
          <Icon name="plus" size="sm" />
          {{ t('admin.billing.publicCatalog.controls.addAll') }}
        </button>
      </div>
    </div>

    <div class="mt-3 flex items-center gap-2 overflow-x-auto pb-1">
      <button
        type="button"
        class="inline-flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition"
        :class="providerFilter === '' ? activeChipClass : inactiveChipClass"
        @click="emit('update:providerFilter', '')"
      >
        <Icon name="grid" size="xs" />
        {{ t('admin.billing.publicCatalog.controls.allProviders') }}
      </button>
      <button
        v-for="provider in providers"
        :key="provider"
        type="button"
        class="inline-flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition"
        :class="providerFilter === provider ? activeChipClass : inactiveChipClass"
        :aria-label="providerLabel(provider)"
        @click="emit('update:providerFilter', provider)"
      >
        <ModelPlatformIcon :platform="provider" size="xs" />
        {{ providerLabel(provider) }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  search: string
  providerFilter: string
  accountFilter: string
  pageSize: number
  providers: string[]
  accountAliases: string[]
  filteredCount: number
}>()

const emit = defineEmits<{
  (e: 'update:search', value: string): void
  (e: 'update:providerFilter', value: string): void
  (e: 'update:accountFilter', value: string): void
  (e: 'update:pageSize', value: number): void
  (e: 'add-filtered'): void
}>()

const { t } = useI18n()
const activeChipClass = 'bg-emerald-100 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-200 dark:ring-emerald-500/30'
const inactiveChipClass = 'border border-slate-200 bg-slate-50 text-slate-600 hover:bg-slate-100 dark:border-dark-700 dark:bg-dark-900 dark:text-slate-300 dark:hover:bg-dark-700'

function providerLabel(provider: string): string {
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    google: 'Google',
    deepseek: 'DeepSeek',
  }
  return labels[provider.toLowerCase()] || provider
}
</script>
