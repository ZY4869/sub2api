<template>
  <div class="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm ring-1 ring-slate-900/5 dark:border-dark-700 dark:bg-dark-800 dark:ring-white/5">
    <div class="flex flex-wrap items-center gap-2">
      <div class="inline-flex items-center gap-1.5 px-2 text-sm font-bold text-slate-700 dark:text-slate-100">
        <Icon name="bolt" size="sm" class="text-amber-500" />
        {{ t('admin.billing.publicCatalog.controls.batchTitle') }}
      </div>
      <div class="h-5 w-px bg-slate-200 dark:bg-dark-600" />

      <label class="relative inline-flex items-center">
        <span class="pointer-events-none absolute left-3 text-xs font-medium text-slate-400">
          {{ t('admin.billing.publicCatalog.controls.ratio') }}
        </span>
        <input
          :value="batchRatio"
          type="number"
          min="0"
          step="1"
          class="h-9 w-28 rounded-lg border border-transparent bg-slate-50 pl-11 pr-6 text-right text-sm font-bold text-emerald-700 outline-none transition hover:bg-slate-100 focus:border-emerald-500 focus:bg-white focus:ring-2 focus:ring-emerald-500/20 dark:bg-dark-900 dark:text-emerald-200"
          data-testid="billing-public-catalog-batch-ratio"
          @input="emit('update:batchRatio', ($event.target as HTMLInputElement).value)"
        />
        <span class="pointer-events-none absolute right-2 text-xs font-bold text-emerald-600/60">%</span>
      </label>

      <div class="relative">
        <button
          type="button"
          class="inline-flex h-9 min-w-[220px] items-center justify-between gap-3 rounded-lg border px-3 text-sm font-medium transition"
          :class="scopeOpen ? 'border-emerald-500 bg-white text-slate-800 ring-2 ring-emerald-500/20 dark:bg-dark-900 dark:text-white' : 'border-transparent bg-slate-50 text-slate-700 hover:bg-slate-100 dark:bg-dark-900 dark:text-slate-200'"
          :aria-label="t('admin.billing.publicCatalog.controls.scopeAria')"
          data-testid="billing-public-catalog-scope"
          @click="scopeOpen = !scopeOpen"
        >
          <span class="flex min-w-0 items-center gap-2">
            <span class="text-xs font-medium text-slate-400">{{ t('admin.billing.publicCatalog.controls.scope') }}</span>
            <span class="truncate">{{ scopeLabel(batchScope) }}</span>
          </span>
          <Icon name="chevronDown" size="xs" :class="scopeOpen ? 'rotate-180 transition' : 'transition'" />
        </button>

        <PublicCatalogScopeMenu
          v-if="scopeOpen"
          :batch-scope="batchScope"
          :account-aliases="accountAliases"
          :base-options="baseScopeOptions"
          @select="selectScope"
        />
      </div>

      <button
        type="button"
        class="inline-flex h-9 items-center gap-1.5 rounded-lg bg-emerald-600 px-3 text-sm font-semibold text-white shadow-sm transition hover:bg-emerald-700"
        @click="emit('apply-batch-ratio')"
      >
        <Icon name="checkCircle" size="sm" />
        {{ t('admin.billing.publicCatalog.controls.applyOfficial') }}
      </button>
      <div class="min-w-[220px] flex-1 text-xs text-slate-500 dark:text-slate-400">
        {{ t('admin.billing.publicCatalog.controls.batchHint') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PublicCatalogScopeMenu from './PublicCatalogScopeMenu.vue'

defineProps<{
  batchRatio: string
  batchScope: string
  accountAliases: string[]
}>()

const emit = defineEmits<{
  (e: 'update:batchRatio', value: string): void
  (e: 'update:batchScope', value: string): void
  (e: 'apply-batch-ratio'): void
}>()

const { t } = useI18n()
const scopeOpen = ref(false)
const baseScopeOptions = computed(() => [
  { value: 'filtered', label: t('admin.billing.publicCatalog.controls.scopes.filtered') },
  { value: 'selected', label: t('admin.billing.publicCatalog.controls.scopes.selected') },
  { value: 'all', label: t('admin.billing.publicCatalog.controls.scopes.all') },
])

function selectScope(value: string) {
  emit('update:batchScope', value)
  scopeOpen.value = false
}

function scopeLabel(value: string): string {
  if (value.startsWith('source:')) {
    return t('admin.billing.publicCatalog.controls.scopes.source', { alias: value.slice('source:'.length) })
  }
  return baseScopeOptions.value.find((option) => option.value === value)?.label || value
}
</script>
