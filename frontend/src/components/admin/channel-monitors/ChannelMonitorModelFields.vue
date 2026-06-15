<template>
  <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
    <div>
      <label class="input-label">
        {{ t('admin.channelMonitors.fields.primaryModel') }} <span class="text-red-500">*</span>
      </label>
      <Select
        v-if="isAccountMode"
        :model-value="primaryModelId"
        :options="modelOptions"
        searchable
        :disabled="loadingModels || modelOptions.length === 0"
        @update:model-value="emit('update:primaryModelId', String($event || ''))"
      >
        <template #selected="{ option }">
          <ChannelMonitorModelOption :option="option || undefined" />
        </template>
        <template #option="{ option }">
          <ChannelMonitorModelOption :option="option || undefined" />
        </template>
      </Select>
      <input
        v-else
        :value="primaryModelId"
        type="text"
        class="input font-mono"
        @input="emit('update:primaryModelId', ($event.target as HTMLInputElement).value)"
      />
      <p v-if="isAccountMode" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ loadingModels ? t('admin.channelMonitors.fields.loadingModels') : modelSelectHint }}
      </p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.additionalModels') }}</label>
      <div v-if="isAccountMode" class="space-y-2">
        <Select
          :model-value="additionalModelToAdd"
          :options="additionalModelOptions"
          searchable
          :disabled="loadingModels || additionalModelOptions.length === 0"
          @change="emit('addAdditionalModel', $event)"
          @update:model-value="emit('update:additionalModelToAdd', $event)"
        >
          <template #selected="{ option }">
            <ChannelMonitorModelOption :option="option || undefined" />
          </template>
          <template #option="{ option }">
            <ChannelMonitorModelOption :option="option || undefined" />
          </template>
        </Select>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="model in additionalModels"
            :key="model"
            type="button"
            class="inline-flex items-center gap-1 rounded-lg border border-gray-200 px-2 py-1 text-xs text-gray-700 dark:border-dark-600 dark:text-gray-200"
            @click="emit('removeAdditionalModel', model)"
          >
            <ModelIcon :model="model" :provider="provider" size="12px" />
            <span class="font-mono">{{ model }}</span>
            <Icon name="x" size="xs" />
          </button>
        </div>
      </div>
      <input
        v-else
        :value="additionalModelsText"
        type="text"
        class="input font-mono"
        :placeholder="t('admin.channelMonitors.fields.additionalModelsPlaceholder')"
        @input="emit('update:additionalModelsText', ($event.target as HTMLInputElement).value)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import ChannelMonitorModelOption from './ChannelMonitorModelOption.vue'
import type { MonitorModelSelectOption } from './channelMonitorFormTypes'

defineProps<{
  additionalModelOptions: MonitorModelSelectOption[]
  additionalModelToAdd: string | number | boolean | null
  additionalModels: string[]
  additionalModelsText: string
  isAccountMode: boolean
  loadingModels: boolean
  modelOptions: MonitorModelSelectOption[]
  modelSelectHint: string
  primaryModelId: string
  provider: string
}>()

const emit = defineEmits<{
  (e: 'addAdditionalModel', value: string | number | boolean | null): void
  (e: 'removeAdditionalModel', model: string): void
  (e: 'update:additionalModelToAdd', value: string | number | boolean | null): void
  (e: 'update:additionalModelsText', value: string): void
  (e: 'update:primaryModelId', value: string): void
}>()

const { t } = useI18n()
</script>
