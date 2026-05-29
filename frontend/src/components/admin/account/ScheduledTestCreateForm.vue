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
        <Select v-model="form.model_input_mode" :options="modelInputModeOptions" />
      </div>
      <div v-if="form.model_input_mode === 'catalog'">
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.model') }}
        </label>
        <Select
          v-model="form.selected_model_key"
          :options="modelOptions"
          :placeholder="t('admin.scheduledTests.model')"
          :searchable="modelOptions.length > 5"
        />
      </div>
      <div v-else class="space-y-3 sm:col-span-2">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.manualModelId') }}
            </label>
            <Input
              v-model="form.manual_model_id"
              :placeholder="t('admin.scheduledTests.manualModelIdPlaceholder')"
            />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.requestAlias') }}
            </label>
            <Input
              v-model="form.request_alias"
              :placeholder="form.manual_model_id || t('admin.scheduledTests.requestAliasPlaceholder')"
            />
          </div>
        </div>
        <div v-if="showManualSourceProtocolField">
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.scheduledTests.sourceProtocol') }}
          </label>
          <Select
            v-model="form.source_protocol"
            :options="sourceProtocolOptions"
            :placeholder="t('admin.scheduledTests.sourceProtocol')"
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
          @update:model-value="(value) => handleFrequencyPresetChange(form, value)"
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
          v-model="form.cron_expression"
          :placeholder="'*/30 * * * *'"
          :hint="t('admin.scheduledTests.cronHelp')"
          :disabled="form.frequency_preset !== 'custom'"
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
        <Input v-model="form.max_results" type="number" placeholder="100" />
      </div>
      <div class="flex items-end">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <Toggle v-model="form.enabled" />
          {{ t('admin.scheduledTests.enabled') }}
        </label>
      </div>
      <div class="flex items-end">
        <div>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <Toggle v-model="form.auto_recover" />
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
        <Select v-model="form.notify_policy" :options="notifyPolicyOptions" />
      </div>
      <div v-if="form.notify_policy === 'failure_only'">
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.notifyFailureThreshold') }}
        </label>
        <Select v-model="form.notify_failure_threshold" :options="failureThresholdOptions" />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.retryInterval') }}
        </label>
        <Select v-model="form.retry_interval_minutes" :options="retryIntervalOptions" />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
          {{ t('admin.scheduledTests.maxRetries') }}
        </label>
        <Select v-model="form.max_retries" :options="maxRetryOptions" />
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

defineEmits<{
  (e: 'cancel'): void
  (e: 'submit'): void
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
</script>
