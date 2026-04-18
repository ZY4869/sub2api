<template>
  <section class="space-y-4">
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <button
        v-for="group in providers"
        :key="group.provider"
        type="button"
        class="rounded-3xl border border-gray-200 bg-white p-5 text-left shadow-sm transition hover:border-primary-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-800"
        :data-testid="`provider-grid-${group.provider}`"
        @click="emit('open-provider', group.provider)"
      >
        <div class="flex items-start gap-4">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-700">
            <ModelPlatformIcon :platform="group.provider" size="lg" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="truncate text-lg font-semibold text-gray-900 dark:text-white">{{ group.label }}</div>
            <div class="mt-2 flex flex-wrap gap-2 text-xs">
              <span class="inline-flex rounded-full bg-gray-100 px-2 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">模型 {{ group.total_count }}</span>
              <span class="inline-flex rounded-full bg-sky-100 px-2 py-1 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200">官方 {{ group.official_count }}</span>
              <span class="inline-flex rounded-full bg-emerald-100 px-2 py-1 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200">出售 {{ group.sale_count }}</span>
            </div>
            <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">
              点击卡片直接打开该供应商的模型定价工作集。
            </p>
          </div>
        </div>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { BillingPricingProviderGroup } from '@/api/admin/billing'

defineProps<{
  providers: BillingPricingProviderGroup[]
}>()

const emit = defineEmits<{
  (e: 'open-provider', provider: string): void
}>()
</script>
