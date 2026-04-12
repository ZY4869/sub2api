<template>
  <div v-if="matrix" class="overflow-auto rounded-2xl border border-gray-200 dark:border-dark-700">
    <table class="min-w-full text-xs">
      <thead class="bg-gray-100/80 dark:bg-dark-900/60">
        <tr>
          <th class="sticky left-0 z-10 border-b border-r border-gray-200 bg-gray-100/95 px-3 py-2 text-left font-semibold text-gray-700 dark:border-dark-700 dark:bg-dark-900/95 dark:text-gray-200">
            {{ leftHeader }}
          </th>
          <th
            v-for="slot in matrix.charge_slots"
            :key="slot"
            class="border-b border-gray-200 px-3 py-2 text-left font-semibold text-gray-700 dark:border-dark-700 dark:text-gray-200"
          >
            {{ humanize(slot) }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, rowIndex) in matrix.rows"
          :key="`${row.surface}:${row.service_tier}`"
          class="border-t border-gray-200 dark:border-dark-700"
        >
          <td class="sticky left-0 z-10 border-r border-gray-200 bg-white px-3 py-3 align-top dark:border-dark-700 dark:bg-dark-800">
            <div class="font-medium text-gray-900 dark:text-white">{{ humanize(row.surface) }}</div>
            <div class="mt-1 text-[11px] text-gray-500 dark:text-gray-400">{{ humanize(row.service_tier) }}</div>
          </td>
          <td
            v-for="slot in matrix.charge_slots"
            :key="`${row.surface}:${row.service_tier}:${slot}`"
            class="min-w-[148px] px-3 py-3 align-top"
          >
            <input
              :value="displayPrice(row.slots?.[slot]?.price)"
              type="number"
              step="any"
              class="input h-10 w-full text-xs"
              @input="updateCellPrice(rowIndex, slot, ($event.target as HTMLInputElement).value)"
            />
            <div class="mt-2 space-y-1 text-[11px] text-gray-500 dark:text-gray-400">
              <div v-if="row.slots?.[slot]?.rule_id" class="break-all">
                {{ ruleLabel }}{{ row.slots[slot].rule_id }}
              </div>
              <div v-if="row.slots?.[slot]?.derived_via" class="break-all">
                {{ derivedLabel }}{{ humanize(row.slots[slot].derived_via || '') }}
              </div>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { GeminiBillingMatrix } from '@/api/admin/models'

const props = withDefaults(
  defineProps<{
    modelValue?: GeminiBillingMatrix | null
    leftHeader?: string
    ruleLabel?: string
    derivedLabel?: string
  }>(),
  {
    modelValue: null,
    leftHeader: 'Surface / Tier',
    ruleLabel: 'Rule: ',
    derivedLabel: 'Derived: '
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: GeminiBillingMatrix | null]
}>()

const matrix = computed(() => props.modelValue ?? null)

function cloneMatrix(source: GeminiBillingMatrix): GeminiBillingMatrix {
  return {
    surfaces: [...(source.surfaces || [])],
    service_tiers: [...(source.service_tiers || [])],
    charge_slots: [...(source.charge_slots || [])],
    rows: (source.rows || []).map((row) => ({
      surface: row.surface,
      service_tier: row.service_tier,
      slots: Object.fromEntries(
        Object.entries(row.slots || {}).map(([slot, cell]) => [
          slot,
          {
            price: cell?.price,
            rule_id: cell?.rule_id,
            derived_via: cell?.derived_via
          }
        ])
      )
    }))
  }
}

function updateCellPrice(rowIndex: number, slot: string, rawValue: string) {
  if (!props.modelValue) {
    return
  }
  const next = cloneMatrix(props.modelValue)
  const row = next.rows[rowIndex]
  if (!row) {
    return
  }
  if (!row.slots) {
    row.slots = {}
  }
  if (!row.slots[slot]) {
    row.slots[slot] = {}
  }
  const trimmed = String(rawValue || '').trim()
  if (!trimmed) {
    row.slots[slot].price = undefined
    emit('update:modelValue', next)
    return
  }
  const parsed = Number(trimmed)
  if (Number.isNaN(parsed)) {
    return
  }
  row.slots[slot].price = parsed
  emit('update:modelValue', next)
}

function humanize(value: string): string {
  const normalized = String(value || '').trim()
  if (!normalized) {
    return '-'
  }
  return normalized.replace(/_/g, ' ').replace(/\b\w/g, (letter) => letter.toUpperCase())
}

function displayPrice(value?: number): string {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return ''
  }
  return String(value)
}
</script>
