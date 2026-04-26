<template>
  <BaseDialog
    :show="show"
    :title="t('admin.channelMonitors.templateApply.title')"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="rounded-2xl border border-gray-200 bg-white p-4 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200">
        <div class="font-medium">{{ template?.name || '-' }}</div>
        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ template?.provider || '-' }}</div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.channelMonitors.templateApply.selectMonitor') }}</label>
        <Select
          v-model="selectedMonitorId"
          :options="monitorOptions"
          :placeholder="t('common.selectOption')"
          :disabled="monitorOptions.length === 0"
        />
        <p v-if="monitorOptions.length === 0" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.channelMonitors.templateApply.noMonitors') }}
        </p>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="submitting || !template || selectedMonitorId == null"
          @click="applyNow"
        >
          <Icon v-if="submitting" name="refresh" size="md" class="mr-2 animate-spin" />
          {{ t('admin.channelMonitors.actions.apply') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import type { AdminChannelMonitor, AdminChannelMonitorTemplate } from '@/api/admin/channelMonitors'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  template: AdminChannelMonitorTemplate | null
  monitors: AdminChannelMonitor[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'applied'): void
}>()

const selectedMonitorId = ref<number | null>(null)
const submitting = ref(false)

const monitorOptions = computed(() => {
  const monitors = Array.isArray(props.monitors) ? props.monitors : []
  return monitors.map((m) => ({ value: m.id, label: m.name }))
})

watch(
  () => [props.show, props.template?.id, Array.isArray(props.monitors) ? props.monitors.length : 0],
  ([show]) => {
    if (!show) return
    const monitors = Array.isArray(props.monitors) ? props.monitors : []
    selectedMonitorId.value = monitors.length > 0 ? monitors[0].id : null
  },
  { immediate: true }
)

async function applyNow() {
  if (!props.template || selectedMonitorId.value == null) return
  submitting.value = true
  try {
    await adminAPI.channelMonitors.applyTemplate(props.template.id, selectedMonitorId.value)
    appStore.showSuccess(t('admin.channelMonitors.messages.applied'))
    emit('applied')
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.channelMonitors.messages.applyFailed'))
  } finally {
    submitting.value = false
  }
}
</script>
