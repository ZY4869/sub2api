<template>
  <BaseDialog
    :show="show"
    :title="t('admin.channelMonitors.actions.associated')"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="rounded-2xl border border-gray-200 bg-white p-4 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200">
        <div class="font-medium">{{ template?.name || '-' }}</div>
        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ template?.provider || '-' }}</div>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-10">
        <LoadingSpinner />
      </div>

      <div v-else-if="items.length === 0" class="p-6">
        <EmptyState
          :title="t('admin.channelMonitors.templateApply.associatedEmptyTitle')"
          :description="t('admin.channelMonitors.templateApply.associatedEmptyDescription')"
        />
      </div>

      <ul v-else class="space-y-2">
        <li
          v-for="m in items"
          :key="m.id"
          class="flex items-center justify-between rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm dark:border-dark-700 dark:bg-dark-800"
        >
          <span class="font-medium text-gray-900 dark:text-white">{{ m.name }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">#{{ m.id }}</span>
        </li>
      </ul>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { AdminChannelMonitorTemplate } from '@/api/admin/channelMonitors'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  template: AdminChannelMonitorTemplate | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(false)
const items = ref<Array<{ id: number; name: string }>>([])

async function load() {
  if (!props.template) return
  loading.value = true
  try {
    items.value = await adminAPI.channelMonitors.listAssociatedMonitors(props.template.id)
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.loadFailed'))
    items.value = []
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.show, props.template?.id],
  ([show]) => {
    if (!show) return
    load()
  },
  { immediate: true }
)
</script>

