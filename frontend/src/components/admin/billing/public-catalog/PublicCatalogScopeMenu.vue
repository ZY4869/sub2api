<template>
  <div class="absolute left-0 top-full z-30 mt-2 w-64 rounded-2xl border border-slate-100 bg-white py-2 shadow-xl dark:border-dark-700 dark:bg-dark-800">
    <div class="px-3 py-1 text-[10px] font-bold uppercase tracking-[0.14em] text-slate-400">
      {{ t('admin.billing.publicCatalog.controls.scopeGroups.global') }}
    </div>
    <button
      v-for="option in baseOptions"
      :key="option.value"
      type="button"
      class="mx-1.5 flex w-[calc(100%-0.75rem)] items-center rounded-xl px-3 py-2 text-left text-sm transition"
      :class="batchScope === option.value ? activeClass : inactiveClass"
      :data-testid="`billing-public-catalog-scope-${option.value}`"
      @click="emit('select', option.value)"
    >
      {{ option.label }}
    </button>

    <template v-if="accountAliases.length > 0">
      <div class="mx-3 my-2 border-t border-slate-100 dark:border-dark-700" />
      <div class="px-3 py-1 text-[10px] font-bold uppercase tracking-[0.14em] text-slate-400">
        {{ t('admin.billing.publicCatalog.controls.scopeGroups.source') }}
      </div>
      <div class="max-h-52 overflow-y-auto px-1.5">
        <button
          v-for="alias in accountAliases"
          :key="`scope-${alias}`"
          type="button"
          class="w-full truncate rounded-xl px-3 py-2 text-left text-sm transition"
          :class="batchScope === `source:${alias}` ? activeClass : inactiveClass"
          :title="sourceLabel(alias)"
          :data-testid="`billing-public-catalog-scope-source-${alias}`"
          @click="emit('select', `source:${alias}`)"
        >
          {{ sourceLabel(alias) }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  batchScope: string
  accountAliases: string[]
  baseOptions: Array<{ value: string; label: string }>
}>()

const emit = defineEmits<{
  (e: 'select', value: string): void
}>()

const { t } = useI18n()
const activeClass = 'bg-emerald-50 font-bold text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200'
const inactiveClass = 'text-slate-700 hover:bg-slate-50 dark:text-slate-200 dark:hover:bg-dark-700'

function sourceLabel(alias: string): string {
  return t('admin.billing.publicCatalog.controls.scopes.source', { alias })
}
</script>
