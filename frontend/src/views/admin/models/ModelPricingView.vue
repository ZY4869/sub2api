<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center gap-2 rounded-2xl border border-gray-200 bg-white p-2 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <button
        v-for="item in tabs"
        :key="item.value"
        type="button"
        class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
        :class="activeTab === item.value ? 'bg-primary-600 text-white shadow-sm' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
        @click="activeTab = item.value"
      >
        {{ item.label }}
      </button>
    </div>

    <ModelCatalogListView :key="activeTab" :pricing-layer="activeTab" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelCatalogListView from './ModelCatalogListView.vue'

const { t } = useI18n()
const activeTab = ref<'official' | 'sale'>('official')

const tabs = computed(() => [
  { value: 'official' as const, label: t('admin.models.pages.official.nav') },
  { value: 'sale' as const, label: t('admin.models.pages.sale.nav') }
])
</script>
