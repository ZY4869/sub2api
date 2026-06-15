<template>
  <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.extraHeaders') }}</label>
      <textarea
        :value="extraHeadersText"
        rows="8"
        class="input font-mono"
        @input="emit('update:extraHeadersText', ($event.target as HTMLTextAreaElement).value)"
      ></textarea>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.channelMonitors.fields.jsonHint') }}
      </p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.bodyOverride') }}</label>
      <div class="mb-2">
        <Select
          :model-value="bodyOverrideMode"
          :options="bodyOverrideModeOptions"
          @update:model-value="emit('update:bodyOverrideMode', $event as ChannelMonitorBodyOverrideMode)"
        />
      </div>
      <textarea
        :value="bodyOverrideText"
        rows="8"
        class="input font-mono"
        @input="emit('update:bodyOverrideText', ($event.target as HTMLTextAreaElement).value)"
      ></textarea>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.channelMonitors.fields.bodyOverrideHint') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import type { ChannelMonitorBodyOverrideMode } from '@/api/admin/channelMonitors'
import type { SelectOptionItem } from '@/utils/channelMonitorPresentation'

defineProps<{
  bodyOverrideMode: ChannelMonitorBodyOverrideMode
  bodyOverrideModeOptions: SelectOptionItem[]
  bodyOverrideText: string
  extraHeadersText: string
}>()

const emit = defineEmits<{
  (e: 'update:bodyOverrideMode', value: ChannelMonitorBodyOverrideMode): void
  (e: 'update:bodyOverrideText', value: string): void
  (e: 'update:extraHeadersText', value: string): void
}>()

const { t } = useI18n()
</script>
