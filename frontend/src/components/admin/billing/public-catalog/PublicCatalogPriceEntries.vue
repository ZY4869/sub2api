<template>
  <div class="space-y-2">
    <component
      :is="editable ? 'label' : 'div'"
      v-for="entry in featuredEntries"
      :key="`featured-${entry.section}-${entry.index}-${entry.entry.id}`"
      class="block rounded-lg border bg-white px-2.5 py-2 shadow-sm dark:bg-dark-800"
      :class="accent === 'sale' ? 'border-emerald-100 dark:border-emerald-500/20' : 'border-white/80 dark:border-dark-700'"
    >
      <span class="block truncate text-[11px] font-medium text-slate-400 dark:text-slate-500">
        {{ priceLabel(entry.entry.id) }}
      </span>
      <span class="mt-1 block truncate text-[10px] font-medium text-slate-400 dark:text-slate-500">
        {{ formatUnit(entry.entry) }}
      </span>
      <input
        v-if="editable"
        :value="formatInputValue(entry.entry.value)"
        type="number"
        min="0"
        step="0.00000001"
        class="mt-1 w-full rounded border border-emerald-200 bg-white px-2 py-1 text-right font-mono text-sm font-semibold text-emerald-700 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 dark:border-emerald-500/30 dark:bg-dark-900 dark:text-emerald-200"
        :data-testid="`${testidPrefix}-${entry.entry.id}`"
        :aria-label="priceLabel(entry.entry.id)"
        @input="emitUpdate(entry.section, entry.index, ($event.target as HTMLInputElement).value)"
      />
      <span
        v-else-if="isUnpriced(entry.entry)"
        class="mt-1 block text-xs font-semibold text-amber-600 dark:text-amber-300"
      >
        {{ unpricedLabel }}
      </span>
      <span
        v-else
        class="mt-1 block truncate font-mono text-sm font-semibold"
        :class="accent === 'sale' ? 'text-emerald-700 dark:text-emerald-200' : 'text-slate-800 dark:text-slate-100'"
      >
        {{ formatPrice(entry.entry) }}
      </span>
    </component>

    <div v-if="compactEntries.length > 0" class="space-y-1.5">
      <component
        :is="editable ? 'label' : 'div'"
        v-for="entry in compactEntries"
        :key="`compact-${entry.section}-${entry.index}-${entry.entry.id}`"
        class="flex items-center justify-between gap-2 text-xs"
      >
        <span class="min-w-0">
          <span class="block truncate text-slate-500 dark:text-slate-400">{{ priceLabel(entry.entry.id) }}</span>
          <span class="block truncate text-[10px] text-slate-400 dark:text-slate-500">{{ formatUnit(entry.entry) }}</span>
        </span>
        <input
          v-if="editable"
          :value="formatInputValue(entry.entry.value)"
          type="number"
          min="0"
          step="0.00000001"
          class="w-24 rounded border border-emerald-200 bg-white px-2 py-0.5 text-right font-mono text-emerald-700 outline-none transition focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 dark:border-emerald-500/30 dark:bg-dark-800 dark:text-emerald-200"
          :data-testid="`${testidPrefix}-${entry.entry.id}`"
          :aria-label="priceLabel(entry.entry.id)"
          @input="emitUpdate(entry.section, entry.index, ($event.target as HTMLInputElement).value)"
        />
        <span
          v-else-if="isUnpriced(entry.entry)"
          class="shrink-0 text-right font-sans text-xs font-semibold text-amber-600 dark:text-amber-300"
        >
          {{ unpricedLabel }}
        </span>
        <span
          v-else
          class="shrink-0 font-mono"
          :class="accent === 'sale' ? 'text-emerald-700 dark:text-emerald-200' : 'text-slate-700 dark:text-slate-100'"
        >
          {{ formatPrice(entry.entry) }}
        </span>
      </component>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { PublicModelCatalogPriceEntry } from '@/api/meta'

type PriceSection = 'primary' | 'secondary'
export type PublicCatalogFlatPriceEntry = {
  section: PriceSection
  index: number
  entry: PublicModelCatalogPriceEntry
}

const props = defineProps<{
  entries: PublicCatalogFlatPriceEntry[]
  editable: boolean
  accent: 'official' | 'sale'
  testidPrefix: string
  priceLabel: (id: string) => string
  formatPrice: (entry: PublicModelCatalogPriceEntry) => string
  formatUnit: (entry: PublicModelCatalogPriceEntry) => string
  formatInputValue: (value: number) => string
  unpricedLabel: string
}>()

const emit = defineEmits<{
  (e: 'update-entry', section: PriceSection, index: number, value: string): void
}>()

const featuredEntries = computed(() => props.entries.slice(0, 2))
const compactEntries = computed(() => props.entries.slice(2))

function isUnpriced(entry: PublicModelCatalogPriceEntry): boolean {
  return entry.supported_unpriced || entry.configured === false
}

function emitUpdate(section: PriceSection, index: number, value: string) {
  emit('update-entry', section, index, value)
}
</script>
