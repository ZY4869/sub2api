<template>
  <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
    <div>
      <label class="input-label">
        {{ t('admin.channelMonitors.fields.name') }} <span class="text-red-500">*</span>
      </label>
      <input
        :value="form.name"
        type="text"
        class="input"
        @input="emitFormField('name', ($event.target as HTMLInputElement).value)"
      />
    </div>

    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.probeMode') }}</label>
      <Select
        :model-value="form.probe_mode"
        :options="probeModeOptions"
        @update:model-value="emitFormField('probe_mode', $event as ChannelMonitorFormState['probe_mode'])"
      >
        <template #selected="{ option }">
          <span>{{ option?.label || t('common.selectOption') }}</span>
        </template>
      </Select>
    </div>

    <div>
      <label class="input-label">
        {{ t('admin.channelMonitors.fields.provider') }} <span class="text-red-500">*</span>
      </label>
      <Select
        :model-value="form.provider"
        :options="providerOptions"
        searchable
        @update:model-value="emitFormField('provider', String($event || ''))"
      >
        <template #selected="{ option }">
          <ChannelMonitorProviderOption :option="option || undefined" />
        </template>
        <template #option="{ option }">
          <ChannelMonitorProviderOption :option="option || undefined" />
        </template>
      </Select>
    </div>

    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.requestProtocol') }}</label>
      <Select
        :model-value="form.request_protocol"
        :options="requestProtocolOptions"
        @update:model-value="emitFormField('request_protocol', $event as ChannelMonitorFormState['request_protocol'])"
      />
    </div>

    <div v-if="isDirectMode" class="md:col-span-2">
      <label class="input-label">
        {{ t('admin.channelMonitors.fields.endpoint') }} <span class="text-red-500">*</span>
      </label>
      <input
        :value="form.endpoint"
        type="text"
        class="input font-mono"
        :placeholder="t('admin.channelMonitors.fields.endpointPlaceholder')"
        @input="emitFormField('endpoint', ($event.target as HTMLInputElement).value)"
      />
    </div>
  </div>

  <div v-if="isAccountMode" class="space-y-3">
    <ChannelMonitorAccountPicker
      :search="accountSearch"
      :accounts="filteredAccounts"
      :selected-ids="form.account_ids"
      :loading="loadingAccounts"
      @update:search="emit('update:accountSearch', $event)"
      @toggle="emit('toggleAccount', $event)"
    />

    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.modelProbeStrategy') }}</label>
      <Select
        :model-value="form.model_probe_strategy"
        :options="modelProbeStrategyOptions"
        @update:model-value="emitFormField('model_probe_strategy', $event as ChannelMonitorFormState['model_probe_strategy'])"
      />
    </div>
  </div>

  <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.intervalSeconds') }}</label>
      <input
        :value="form.interval_seconds"
        type="number"
        class="input"
        min="15"
        max="3600"
        @input="emitFormField('interval_seconds', Number(($event.target as HTMLInputElement).value))"
      />
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.channelMonitors.fields.intervalHint') }}
      </p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.enabled') }}</label>
      <div class="flex items-center gap-2">
        <Toggle
          :model-value="form.enabled"
          @update:model-value="emitFormField('enabled', Boolean($event))"
        />
        <span class="text-sm text-gray-600 dark:text-gray-300">
          {{ form.enabled ? t('common.enabled') : t('common.disabled') }}
        </span>
      </div>
    </div>

    <div>
      <label class="input-label">{{ t('admin.channelMonitors.fields.template') }}</label>
      <Select
        :model-value="form.template_id"
        :options="templateOptions"
        @update:model-value="emitFormField('template_id', $event as number | null)"
      />
    </div>

    <div v-if="form.request_protocol === 'openai'">
      <label class="input-label">{{ t('admin.channelMonitors.fields.openaiApiMode') }}</label>
      <Select
        :model-value="form.openai_api_mode"
        :options="openAIApiModeOptions"
        @update:model-value="emitFormField('openai_api_mode', $event as ChannelMonitorFormState['openai_api_mode'])"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { Account } from '@/types'
import type { SelectOptionItem } from '@/utils/channelMonitorPresentation'
import ChannelMonitorAccountPicker from './ChannelMonitorAccountPicker.vue'
import ChannelMonitorProviderOption from './ChannelMonitorProviderOption.vue'
import type { ChannelMonitorFormState } from './channelMonitorFormTypes'

defineProps<{
  accountSearch: string
  filteredAccounts: Account[]
  form: ChannelMonitorFormState
  isAccountMode: boolean
  isDirectMode: boolean
  loadingAccounts: boolean
  modelProbeStrategyOptions: SelectOptionItem[]
  openAIApiModeOptions: SelectOptionItem[]
  probeModeOptions: SelectOptionItem[]
  providerOptions: SelectOptionItem[]
  requestProtocolOptions: SelectOptionItem[]
  templateOptions: SelectOptionItem[]
}>()

const emit = defineEmits<{
  (e: 'toggleAccount', id: number): void
  (e: 'update:accountSearch', value: string): void
  <K extends keyof ChannelMonitorFormState>(e: 'update:formField', field: K, value: ChannelMonitorFormState[K]): void
}>()

const { t } = useI18n()

function emitFormField<K extends keyof ChannelMonitorFormState>(field: K, value: ChannelMonitorFormState[K]) {
  emit('update:formField', field, value)
}
</script>
