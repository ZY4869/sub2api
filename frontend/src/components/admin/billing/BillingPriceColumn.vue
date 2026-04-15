<template>
  <section class="flex min-h-0 flex-col rounded-3xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900/40">
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-4 py-4 dark:border-dark-700">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ title }}</h3>
        <p v-if="description" class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ description }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <slot name="actions" />
      </div>
    </div>

    <div class="flex-1 space-y-3 overflow-y-auto px-4 py-4">
      <article
        v-for="item in items"
        :key="item.id"
        class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex flex-wrap items-center gap-2">
          <label v-if="selectable" class="inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" :checked="selectedIds.includes(item.id)" @change="emit('toggle-select', item.id)" />
            选中
          </label>
          <span class="inline-flex rounded-full bg-white px-2 py-1 text-[11px] font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ item.mode }}</span>
          <span v-if="item.rule_id" class="inline-flex rounded-full bg-sky-100 px-2 py-1 text-[11px] text-sky-700 dark:bg-sky-500/15 dark:text-sky-200">{{ item.rule_id }}</span>
          <button type="button" class="ml-auto text-xs text-rose-600 hover:text-rose-700 dark:text-rose-300" @click="emit('remove-item', item.id)">删除</button>
        </div>

        <div class="mt-3 grid gap-3 md:grid-cols-2">
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>计费项</span>
            <select class="input" :value="item.charge_slot" @change="update(item.id, 'charge_slot', ($event.target as HTMLSelectElement).value)">
              <option v-for="option in billingChargeSlotOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>模式</span>
            <select class="input" :value="item.mode" @change="update(item.id, 'mode', ($event.target as HTMLSelectElement).value)">
              <option v-for="option in billingModeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>基础价格</span>
            <input class="input" type="number" step="0.0000001" :value="item.price" @input="update(item.id, 'price', Number(($event.target as HTMLInputElement).value))" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>单位</span>
            <input class="input" type="text" :value="item.unit" @input="update(item.id, 'unit', ($event.target as HTMLInputElement).value)" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>服务层级</span>
            <select class="input" :value="item.service_tier || ''" @change="update(item.id, 'service_tier', ($event.target as HTMLSelectElement).value)">
              <option v-for="option in billingServiceTierOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>Batch 模式</span>
            <select class="input" :value="item.batch_mode || ''" @change="update(item.id, 'batch_mode', ($event.target as HTMLSelectElement).value)">
              <option v-for="option in billingBatchModeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>阈值 Token</span>
            <input class="input" type="number" step="1" :value="item.threshold_tokens ?? ''" @input="updateOptionalNumber(item.id, 'threshold_tokens', ($event.target as HTMLInputElement).value)" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>阈值后价格</span>
            <input class="input" type="number" step="0.0000001" :value="item.price_above_threshold ?? ''" @input="updateOptionalNumber(item.id, 'price_above_threshold', ($event.target as HTMLInputElement).value)" />
          </label>
        </div>

        <div class="mt-3 grid gap-3 md:grid-cols-2">
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>Surface</span>
            <input class="input" type="text" :value="item.surface || ''" @input="update(item.id, 'surface', ($event.target as HTMLInputElement).value)" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>Operation</span>
            <input class="input" type="text" :value="item.operation_type || ''" @input="update(item.id, 'operation_type', ($event.target as HTMLInputElement).value)" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>Input Modality</span>
            <input class="input" type="text" :value="item.input_modality || ''" @input="update(item.id, 'input_modality', ($event.target as HTMLInputElement).value)" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>Output Modality</span>
            <input class="input" type="text" :value="item.output_modality || ''" @input="update(item.id, 'output_modality', ($event.target as HTMLInputElement).value)" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>Cache Phase</span>
            <input class="input" type="text" :value="item.cache_phase || ''" @input="update(item.id, 'cache_phase', ($event.target as HTMLInputElement).value)" />
          </label>
          <label class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <span>Grounding</span>
            <input class="input" type="text" :value="item.grounding_kind || ''" @input="update(item.id, 'grounding_kind', ($event.target as HTMLInputElement).value)" />
          </label>
        </div>

        <label class="mt-3 inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
          <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" :checked="item.enabled" @change="update(item.id, 'enabled', ($event.target as HTMLInputElement).checked)" />
          启用该价格项
        </label>
      </article>

      <div v-if="items.length === 0" class="rounded-2xl border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
        当前层还没有价格项，可以从预设按钮快速生成，或者手动新增。
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { BillingPriceItem } from '@/api/admin/billing'
import {
  billingBatchModeOptions,
  billingChargeSlotOptions,
  billingModeOptions,
  billingServiceTierOptions,
  defaultUnitForChargeSlot,
} from './pricingOptions'

const props = withDefaults(defineProps<{
  title: string
  description?: string
  items: BillingPriceItem[]
  selectedIds?: string[]
  selectable?: boolean
}>(), {
  description: '',
  selectedIds: () => [],
  selectable: false,
})

const emit = defineEmits<{
  (e: 'update-item', value: BillingPriceItem): void
  (e: 'remove-item', id: string): void
  (e: 'toggle-select', id: string): void
}>()

function update(id: string, field: keyof BillingPriceItem, value: BillingPriceItem[keyof BillingPriceItem]) {
  const target = props.items.find((item) => item.id === id)
  if (!target) return
  const next: BillingPriceItem = { ...target, [field]: value }
  if (field === 'charge_slot') {
    next.unit = defaultUnitForChargeSlot(String(value || ''))
  }
  emit('update-item', next)
}

function updateOptionalNumber(id: string, field: 'threshold_tokens' | 'price_above_threshold', raw: string) {
  const normalized = raw.trim()
  update(id, field, normalized === '' ? undefined : Number(normalized))
}
</script>
