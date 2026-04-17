<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type { OpenAIWSMode } from '@/utils/openaiWsMode'

defineProps<{
  visible: boolean
  modeOptions: SelectOption[]
  concurrencyHintKey: string
}>()

const enabled = defineModel<boolean>('enabled', { required: true })
const mode = defineModel<OpenAIWSMode>('mode', { required: true })

const { t } = useI18n()
</script>

<template>
  <div
    v-if="visible"
    class="border-t border-gray-200 pt-4 dark:border-dark-600"
  >
    <div class="mb-3 flex items-center justify-between">
      <div>
        <label
          id="bulk-edit-openai-ws-mode-label"
          class="input-label mb-0"
          for="bulk-edit-openai-ws-mode-enabled"
        >
          {{ t('admin.accounts.openai.wsMode') }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.openai.wsModeDesc') }}
        </p>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t(concurrencyHintKey) }}
        </p>
      </div>
      <input
        v-model="enabled"
        id="bulk-edit-openai-ws-mode-enabled"
        type="checkbox"
        aria-controls="bulk-edit-openai-ws-mode-body"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>

    <div
      id="bulk-edit-openai-ws-mode-body"
      :class="!enabled && 'pointer-events-none opacity-50'"
    >
      <div class="w-full md:w-64">
        <Select
          :model-value="mode"
          :options="modeOptions"
          aria-labelledby="bulk-edit-openai-ws-mode-label"
          @update:model-value="mode = $event as OpenAIWSMode"
        />
      </div>
    </div>
  </div>
</template>
