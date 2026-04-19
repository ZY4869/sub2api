<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import RequestDetailsSubjectTab from './components/RequestDetailsSubjectTab.vue'
import RequestDetailsTraceTab from './components/RequestDetailsTraceTab.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const activeTab = computed<'trace' | 'subject'>(() =>
  route.query.tab === 'subject' ? 'subject' : 'trace'
)

async function switchTab(tab: 'trace' | 'subject') {
  await router.replace({
    query: {
      ...route.query,
      tab,
    }
  })
}
</script>

<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ t('admin.requestDetails.title') }}
        </h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.requestDetails.description') }}
        </p>
      </div>

      <div class="inline-flex rounded-full border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <button
          class="rounded-full px-4 py-2 text-sm font-medium transition-colors"
          :class="activeTab === 'trace' ? 'bg-primary-600 text-white' : 'text-gray-600 dark:text-gray-300'"
          type="button"
          @click="switchTab('trace')"
        >
          {{ t('admin.requestDetails.pageTabs.trace') }}
        </button>
        <button
          class="rounded-full px-4 py-2 text-sm font-medium transition-colors"
          :class="activeTab === 'subject' ? 'bg-primary-600 text-white' : 'text-gray-600 dark:text-gray-300'"
          type="button"
          @click="switchTab('subject')"
        >
          {{ t('admin.requestDetails.pageTabs.subject') }}
        </button>
      </div>

      <RequestDetailsTraceTab v-if="activeTab === 'trace'" />
      <RequestDetailsSubjectTab v-else />
    </div>
  </AppLayout>
</template>
