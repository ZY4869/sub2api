<template>
  <div
    class="rounded-xl border border-primary-200 bg-primary-50/50 p-4 dark:border-primary-800 dark:bg-primary-900/20"
  >
    <div class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ t('admin.scheduledTests.addPlan') }}
    </div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.modelInputMode') }}
        </label>
        <Select
          :model-value="form.model_input_mode"
          :options="modelInputModeOptions"
          @update:model-value="(value) => updateFormField('model_input_mode', value)"
        />
      </div>
      <div v-if="form.model_input_mode === 'catalog'">
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.model') }}
        </label>
        <Select
          :model-value="form.selected_model_key"
          :options="modelOptions"
          :placeholder="t('admin.scheduledTests.model')"
          :searchable="modelOptions.length > 5"
          @update:model-value="(value) => updateFormField('selected_model_key', value)"
        />
      </div>
      <div v-else class="space-y-3 sm:col-span-2">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.manualModelId') }}
            </label>
            <Input
              :model-value="form.manual_model_id"
              :placeholder="t('admin.scheduledTests.manualModelIdPlaceholder')"
              @update:model-value="(value) => updateFormField('manual_model_id', value)"
            />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.requestAlias') }}
            </label>
            <Input
              :model-value="form.request_alias"
              :placeholder="form.manual_model_id || t('admin.scheduledTests.requestAliasPlaceholder')"
              @update:model-value="(value) => updateFormField('request_alias', value)"
            />
          </div>
        </div>
        <div v-if="showManualSourceProtocolField">
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.scheduledTests.sourceProtocol') }}
          </label>
          <Select
            :model-value="form.source_protocol"
            :options="sourceProtocolOptions"
            :placeholder="t('admin.scheduledTests.sourceProtocol')"
            @update:model-value="(value) => updateFormField('source_protocol', value)"
          />
        </div>
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.frequency') }}
        </label>
        <Select
          :model-value="form.frequency_preset"
          :options="frequencyOptions"
          @update:model-value="updateFrequencyPreset"
        />
      </div>
      <div>
        <label class="mb-1 flex items-center gap-1 text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.cronExpression') }}
          <HelpTooltip>
            <template #trigger>
              <span class="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-gray-400/70 text-[10px] font-semibold text-gray-400 transition-colors hover:border-primary-500 hover:text-primary-600 dark:border-gray-500 dark:text-gray-500 dark:hover:border-primary-400 dark:hover:text-primary-400">
                ?
              </span>
            </template>
            <div class="space-y-1.5">
              <p class="font-medium">{{ t('admin.scheduledTests.cronTooltipTitle') }}</p>
              <p>{{ t('admin.scheduledTests.cronTooltipMeaning') }}</p>
              <p>{{ t('admin.scheduledTests.cronTooltipExampleEvery30Min') }}</p>
              <p>{{ t('admin.scheduledTests.cronTooltipExampleHourly') }}</p>
              <p>{{ t('admin.scheduledTests.cronTooltipExampleDaily') }}</p>
              <p>{{ t('admin.scheduledTests.cronTooltipExampleWeekly') }}</p>
              <p>{{ t('admin.scheduledTests.cronTooltipRange') }}</p>
            </div>
          </HelpTooltip>
        </label>
        <Input
          :model-value="form.cron_expression"
          :placeholder="'*/30 * * * *'"
          :hint="t('admin.scheduledTests.cronHelp')"
          :disabled="form.frequency_preset !== 'custom'"
          @update:model-value="(value) => updateFormField('cron_expression', value)"
        />
      </div>
      <div>
        <label class="mb-1 flex items-center gap-1 text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.maxResults') }}
          <HelpTooltip>
            <template #trigger>
              <span class="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-gray-400/70 text-[10px] font-semibold text-gray-400 transition-colors hover:border-primary-500 hover:text-primary-600 dark:border-gray-500 dark:text-gray-500 dark:hover:border-primary-400 dark:hover:text-primary-400">
                ?
              </span>
            </template>
            <div class="space-y-1.5">
              <p class="font-medium">{{ t('admin.scheduledTests.maxResultsTooltipTitle') }}</p>
              <p>{{ t('admin.scheduledTests.maxResultsTooltipMeaning') }}</p>
              <p>{{ t('admin.scheduledTests.maxResultsTooltipBody') }}</p>
              <p>{{ t('admin.scheduledTests.maxResultsTooltipExample') }}</p>
              <p>{{ t('admin.scheduledTests.maxResultsTooltipRange') }}</p>
            </div>
          </HelpTooltip>
        </label>
        <Input
          :model-value="form.max_results"
          type="number"
          placeholder="100"
          @update:model-value="(value) => updateFormField('max_results', value)"
        />
      </div>
      <div class="flex items-end">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <Toggle
            :model-value="form.enabled"
            @update:model-value="(value) => updateFormField('enabled', value)"
          />
          {{ t('admin.scheduledTests.enabled') }}
        </label>
      </div>
      <div class="flex items-end">
        <div>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <Toggle
              :model-value="form.auto_recover"
              @update:model-value="(value) => updateFormField('auto_recover', value)"
            />
            {{ t('admin.scheduledTests.autoRecover') }}
          </label>
          <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
            {{ t('admin.scheduledTests.autoRecoverHelp') }}
          </p>
        </div>
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.notifyPolicy') }}
        </label>
        <Select
          :model-value="form.notify_policy"
          :options="notifyPolicyOptions"
          @update:model-value="(value) => updateFormField('notify_policy', value)"
        />
      </div>
      <div v-if="form.notify_policy === 'failure_only'">
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.notifyFailureThreshold') }}
        </label>
        <Select
          :model-value="form.notify_failure_threshold"
          :options="failureThresholdOptions"
          @update:model-value="(value) => updateFormField('notify_failure_threshold', value)"
        />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.retryInterval') }}
        </label>
        <Select
          :model-value="form.retry_interval_minutes"
          :options="retryIntervalOptions"
          @update:model-value="(value) => updateFormField('retry_interval_minutes', value)"
        />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.maxRetries') }}
        </label>
        <Select
          :model-value="form.max_retries"
          :options="maxRetryOptions"
          @update:model-value="(value) => updateFormField('max_retries', value)"
        />
      </div>
    </div>
    <div class="mt-3 flex justify-end gap-2">
      <button
        class="rounded-lg bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        @click="$emit('cancel')"
      >
        {{ t('common.cancel') }}
      </button>
      <button
        :disabled="submitDisabled"
        class="flex items-center gap-1.5 rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
        @click="$emit('submit')"
      >
        <Icon v-if="submitting" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
        {{ t('common.save') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Select from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import Toggle from '@/components/common/Toggle.vue'
import { Icon } from '@/components/icons'

const props = defineProps<{
  ctx: any
  form: any
  submitting?: boolean
  submitDisabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'submit'): void
  (e: 'update:form', value: any): void
}>()

const {
  t,
  modelInputModeOptions,
  modelOptions,
  showManualSourceProtocolField,
  sourceProtocolOptions,
  frequencyOptions,
  handleFrequencyPresetChange,
  notifyPolicyOptions,
  failureThresholdOptions,
  retryIntervalOptions,
  maxRetryOptions
} = props.ctx

const updateFormField = (key: string, value: unknown) => {
  emit('update:form', { ...props.form, [key]: value })
}

const updateFrequencyPreset = (value: string | number | boolean | null) => {
  const next = { ...props.form }
  handleFrequencyPresetChange(next, value)
  emit('update:form', next)
}
</script>
