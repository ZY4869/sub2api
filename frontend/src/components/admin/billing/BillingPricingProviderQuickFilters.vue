<template>
  <div class="flex flex-wrap gap-3">
    <button
      type="button"
      class="min-w-[180px] rounded-2xl border px-4 py-3 text-left transition"
      :class="activeProvider ? 'border-gray-200 bg-white hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800' : 'border-primary-400 bg-primary-50 dark:border-primary-500/40 dark:bg-primary-500/10'"
      data-testid="provider-quick-filter-all"
      @click="emit('select', '')"
    >
      <div class="text-sm font-semibold text-gray-900 dark:text-white">全部供应商</div>
      <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">模型 {{ totalCount }}</div>
      <div class="mt-2 flex flex-wrap gap-2 text-[11px]">
        <span class="inline-flex rounded-full bg-gray-100 px-2 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">
          官方 {{ officialCount }}
        </span>
        <span class="inline-flex rounded-full bg-emerald-100 px-2 py-1 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200">
          售价 {{ saleCount }}
        </span>
      </div>
    </button>

    <button
      v-for="provider in providers"
      :key="provider.provider"
      type="button"
      class="min-w-[180px] rounded-2xl border px-4 py-3 text-left transition"
      :class="activeProvider === provider.provider ? 'border-primary-400 bg-primary-50 dark:border-primary-500/40 dark:bg-primary-500/10' : 'border-gray-200 bg-white hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800'"
      :data-testid="`provider-quick-filter-${provider.provider}`"
      @click="emit('select', provider.provider)"
    >
      <div class="flex items-center gap-3">
        <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-gray-100 dark:bg-dark-700">
          <ModelPlatformIcon :platform="provider.provider" size="sm" />
        </div>
        <div class="min-w-0">
          <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ provider.label }}</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">模型 {{ provider.total_count }}</div>
        </div>
      </div>
      <div class="mt-2 flex flex-wrap gap-2 text-[11px]">
        <span class="inline-flex rounded-full bg-sky-100 px-2 py-1 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200">
          官方 {{ provider.official_count }}
        </span>
        <span class="inline-flex rounded-full bg-emerald-100 px-2 py-1 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200">
          售价 {{ provider.sale_count }}
        </span>
      </div>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { BillingPricingProviderGroup } from '@/api/admin/billing'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'

const props = defineProps<{
  providers: BillingPricingProviderGroup[]
  activeProvider: string
}>()

const emit = defineEmits<{
  (e: 'select', provider: string): void
}>()

const totalCount = computed(() => props.providers.reduce((sum, item) => sum + item.total_count, 0))
const officialCount = computed(() => props.providers.reduce((sum, item) => sum + item.official_count, 0))
const saleCount = computed(() => props.providers.reduce((sum, item) => sum + item.sale_count, 0))
</script>
