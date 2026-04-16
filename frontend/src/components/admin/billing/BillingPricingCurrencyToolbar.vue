<template>
  <section class="rounded-3xl border border-amber-200 bg-amber-50/80 p-4 dark:border-amber-500/20 dark:bg-amber-500/10">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-1">
        <h3 class="text-sm font-semibold text-amber-950 dark:text-amber-100">模型主货币</h3>
        <p class="text-xs text-amber-900 dark:text-amber-200">
          官方价和售价共用同一主货币，保存时仍会自动换回 USD 基准值。
        </p>
      </div>

      <label class="space-y-1 text-xs text-amber-950 dark:text-amber-100">
        <span>编辑货币</span>
        <select
          class="input min-w-[160px]"
          :value="currency"
          data-testid="pricing-currency-select"
          @change="emit('update:currency', ($event.target as HTMLSelectElement).value as BillingPricingCurrency)"
        >
          <option value="USD">USD ($)</option>
          <option value="CNY" :disabled="!cnyEnabled">CNY (￥)</option>
        </select>
      </label>
    </div>

    <div class="mt-3 flex flex-wrap gap-3 text-xs text-amber-900 dark:text-amber-200">
      <span data-testid="pricing-currency-rate">
        USD/CNY {{ usdToCnyRate ? usdToCnyRate.toFixed(4) : '不可用' }}
      </span>
      <span v-if="currency === 'CNY' && usdToCnyRate">
        当前输入按人民币展示，保存时会自动换回 USD。
      </span>
      <span v-else>
        当前输入按美元展示，不影响底层计费口径。
      </span>
    </div>

    <p
      v-if="saveBlocked"
      class="mt-3 rounded-2xl border border-amber-300 bg-white/80 px-3 py-2 text-xs text-amber-950 dark:border-amber-400/30 dark:bg-dark-900/40 dark:text-amber-100"
      data-testid="pricing-currency-alert"
    >
      当前选择的是人民币展示，但本次会话没有可用的 USD/CNY 汇率，已阻止保存。请刷新汇率后再继续编辑。
    </p>
  </section>
</template>

<script setup lang="ts">
import type { BillingPricingCurrency } from '@/api/admin/billing'

defineProps<{
  currency: BillingPricingCurrency
  usdToCnyRate?: number | null
  cnyEnabled: boolean
  saveBlocked: boolean
}>()

const emit = defineEmits<{
  (e: 'update:currency', value: BillingPricingCurrency): void
}>()
</script>
