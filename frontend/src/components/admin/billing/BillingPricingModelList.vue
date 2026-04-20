<template>
  <section class="rounded-3xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <div>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">全部模型定价</h3>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">默认按列表分页展示，支持快速切换到任意模型编辑。</p>
      </div>
      <div class="text-sm text-gray-500 dark:text-gray-400">
        共 {{ total }} 条
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead class="bg-gray-50/80 dark:bg-dark-900/60">
          <tr>
            <th class="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-300">模型</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-300">供应商</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-300">模式</th>
            <th class="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-300">能力</th>
            <th class="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-300">官方项</th>
            <th class="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-300">出售项</th>
            <th class="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-300"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="item in items"
            :key="item.model"
            class="border-t border-gray-100 dark:border-dark-700"
          >
            <td class="px-4 py-3">
              <div class="flex flex-wrap items-center gap-2">
                <div class="font-medium text-gray-900 dark:text-white">{{ item.display_name || item.model }}</div>
                <span
                  v-if="item.pricing_status !== 'ok'"
                  class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium"
                  :class="pricingStatusClass(item.pricing_status)"
                >
                  {{ pricingStatusLabel(item.pricing_status) }}
                </span>
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ item.model }}</div>
              <div
                v-if="item.pricing_warnings?.length"
                class="mt-1 text-xs"
                :class="item.pricing_status === 'conflict' || item.pricing_status === 'missing' ? 'text-rose-600 dark:text-rose-300' : 'text-amber-600 dark:text-amber-300'"
              >
                {{ item.pricing_warnings[0] }}
              </div>
            </td>
            <td class="px-4 py-3 text-gray-700 dark:text-gray-200">{{ item.provider || '-' }}</td>
            <td class="px-4 py-3 text-gray-700 dark:text-gray-200">{{ item.mode || '-' }}</td>
            <td class="px-4 py-3">
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="capability in capabilityBadges(item)"
                  :key="capability"
                  class="inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-700 dark:bg-dark-700 dark:text-gray-200"
                >
                  {{ capability }}
                </span>
              </div>
            </td>
            <td class="px-4 py-3 text-right text-gray-700 dark:text-gray-200">{{ item.official_count }}</td>
            <td class="px-4 py-3 text-right text-gray-700 dark:text-gray-200">{{ item.sale_count }}</td>
            <td class="px-4 py-3 text-right">
              <button type="button" class="btn btn-primary btn-sm" @click="emit('open', item.model)">编辑定价</button>
            </td>
          </tr>
          <tr v-if="items.length === 0">
            <td colspan="7" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-gray-400">当前筛选下没有模型。</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 px-5 py-4 text-sm dark:border-dark-700">
      <div class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
        <span>每页</span>
        <select :value="pageSize" class="input w-24" @change="emit('update:pageSize', Number(($event.target as HTMLSelectElement).value))">
          <option :value="20">20</option>
          <option :value="50">50</option>
          <option :value="100">100</option>
        </select>
      </div>

      <div class="flex items-center gap-2">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="emit('update:page', page - 1)">上一页</button>
        <span class="text-gray-600 dark:text-gray-300">第 {{ page }} / {{ totalPages }} 页</span>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="page >= totalPages" @click="emit('update:page', page + 1)">下一页</button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { BillingPricingListItem } from '@/api/admin/billing'

const props = defineProps<{
  items: BillingPricingListItem[]
  total: number
  page: number
  pageSize: number
}>()

const emit = defineEmits<{
  (e: 'open', model: string): void
  (e: 'update:page', value: number): void
  (e: 'update:pageSize', value: number): void
}>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))

function capabilityBadges(item: BillingPricingListItem): string[] {
  const badges: string[] = []
  if (item.capabilities.supports_tiered_pricing) badges.push('Tiered')
  if (item.capabilities.supports_batch_pricing) badges.push('Batch')
  if (item.capabilities.supports_prompt_caching) badges.push('Caching')
  return badges
}

function pricingStatusLabel(status: BillingPricingListItem['pricing_status']): string {
  switch (status) {
    case 'conflict':
      return '冲突'
    case 'missing':
      return '缺价'
    case 'fallback':
      return '回退'
    default:
      return '正常'
  }
}

function pricingStatusClass(status: BillingPricingListItem['pricing_status']): string {
  switch (status) {
    case 'conflict':
    case 'missing':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-200'
    case 'fallback':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-200'
    default:
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200'
  }
}
</script>
