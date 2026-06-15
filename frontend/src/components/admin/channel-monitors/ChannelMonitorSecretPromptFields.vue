<template>
  <div v-if="isDirectMode">
    <label class="input-label">{{ t('admin.channelMonitors.fields.apiKey') }}</label>
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
      <input
        :value="apiKey"
        type="password"
        class="input font-mono"
        :placeholder="apiKeyPlaceholder"
        @input="emit('update:apiKey', ($event.target as HTMLInputElement).value)"
      />
      <div v-if="isEditMode" class="flex items-center gap-2">
        <Toggle
          :model-value="clearApiKey"
          @update:model-value="emit('update:clearApiKey', Boolean($event))"
        />
        <span class="text-sm text-gray-600 dark:text-gray-300">
          {{ t('admin.channelMonitors.fields.clearApiKey') }}
        </span>
      </div>
    </div>
    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ apiKeyHintText }}</p>
  </div>

  <div>
    <label class="input-label">{{ t('admin.channelMonitors.fields.testPromptTemplate') }}</label>
    <textarea
      :value="testPromptTemplate"
      rows="3"
      class="input"
      :placeholder="t('admin.channelMonitors.fields.testPromptTemplatePlaceholder')"
      @input="emit('update:testPromptTemplate', ($event.target as HTMLTextAreaElement).value)"
    ></textarea>
    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {{ t('admin.channelMonitors.fields.testPromptTemplateHint') }}
    </p>
  </div>

  <div v-if="!isEditMode" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex items-center gap-2">
      <Toggle
        :model-value="saveAsTemplate"
        @update:model-value="emit('update:saveAsTemplate', Boolean($event))"
      />
      <span class="text-sm font-medium text-gray-800 dark:text-gray-100">
        {{ t('admin.channelMonitors.fields.saveAsTemplate') }}
      </span>
    </div>
    <input
      v-if="saveAsTemplate"
      :value="templateName"
      type="text"
      class="input mt-3"
      :placeholder="t('admin.channelMonitors.fields.templateNamePlaceholder')"
      @input="emit('update:templateName', ($event.target as HTMLInputElement).value)"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'

defineProps<{
  apiKey: string
  apiKeyHintText: string
  apiKeyPlaceholder: string
  clearApiKey: boolean
  isDirectMode: boolean
  isEditMode: boolean
  saveAsTemplate: boolean
  templateName: string
  testPromptTemplate: string
}>()

const emit = defineEmits<{
  (e: 'update:apiKey', value: string): void
  (e: 'update:clearApiKey', value: boolean): void
  (e: 'update:saveAsTemplate', value: boolean): void
  (e: 'update:templateName', value: string): void
  (e: 'update:testPromptTemplate', value: string): void
}>()

const { t } = useI18n()
</script>
