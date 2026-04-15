<template>
  <section class="space-y-4">
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <button
        v-for="group in providers"
        :key="group.provider"
        type="button"
        class="rounded-3xl border p-5 text-left shadow-sm transition"
        :class="expandedProvider === group.provider ? 'border-primary-400 bg-primary-50 dark:border-primary-500/40 dark:bg-primary-500/10' : 'border-gray-200 bg-white hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800'"
        @click="emit('toggle-provider', group.provider)"
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
          </div>
        </div>
      </button>
    </div>

    <div v-if="expandedProvider" class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ expandedProviderLabel }}</h3>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">点击模型即可直接打开二级弹窗编辑定价。</p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="emit('toggle-provider', '')">收起</button>
      </div>

      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <button
          v-for="item in expandedModels"
          :key="item.model"
          type="button"
          class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 text-left transition hover:border-primary-300 dark:border-dark-700 dark:bg-dark-900/40"
          @click="emit('open-model', item.model)"
        >
          <div class="font-medium text-gray-900 dark:text-white">{{ item.display_name || item.model }}</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ item.model }}</div>
          <div class="mt-3 flex items-center justify-between text-xs text-gray-600 dark:text-gray-300">
            <span>官方 {{ item.official_count }}</span>
            <span>出售 {{ item.sale_count }}</span>
          </div>
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { BillingPricingListItem, BillingPricingProviderGroup } from '@/api/admin/billing'

const props = defineProps<{
  providers: BillingPricingProviderGroup[]
  providerModels: Record<string, BillingPricingListItem[]>
  expandedProvider: string
}>()

const emit = defineEmits<{
  (e: 'toggle-provider', provider: string): void
  (e: 'open-model', model: string): void
}>()

const expandedModels = computed(() => props.providerModels[props.expandedProvider] || [])
const expandedProviderLabel = computed(() => props.providers.find((item) => item.provider === props.expandedProvider)?.label || props.expandedProvider)
</script>
