<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="extra-wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <form id="channel-monitor-form" class="space-y-6" @submit.prevent="handleSubmit">
      <ChannelMonitorGeneralFields
        v-model:account-search="accountSearch"
        :filtered-accounts="filteredAccounts"
        :form="form"
        :is-account-mode="isAccountMode"
        :is-direct-mode="isDirectMode"
        :loading-accounts="loadingAccounts"
        :model-probe-strategy-options="modelProbeStrategyOptions"
        :open-a-i-api-mode-options="openAIApiModeOptions"
        :probe-mode-options="probeModeOptions"
        :provider-options="providerOptions"
        :request-protocol-options="requestProtocolOptions"
        :template-options="templateOptions"
        @toggle-account="toggleAccount"
        @update:form-field="updateFormField"
      />

      <ChannelMonitorModelFields
        v-model:additional-model-to-add="additionalModelToAdd"
        v-model:additional-models-text="additionalModelsText"
        v-model:primary-model-id="form.primary_model_id"
        :additional-model-options="additionalModelOptions"
        :additional-models="additionalModels"
        :is-account-mode="isAccountMode"
        :loading-models="loadingModels"
        :model-options="modelOptions"
        :model-select-hint="modelSelectHint"
        :provider="form.provider"
        @add-additional-model="addAdditionalModel"
        @remove-additional-model="removeAdditionalModel"
      />

      <ChannelMonitorSecretPromptFields
        v-model:api-key="apiKey"
        v-model:clear-api-key="clearApiKey"
        v-model:save-as-template="form.save_as_template"
        v-model:template-name="form.template_name"
        v-model:test-prompt-template="form.test_prompt_template"
        :api-key-hint-text="apiKeyHintText"
        :api-key-placeholder="apiKeyPlaceholder"
        :is-direct-mode="isDirectMode"
        :is-edit-mode="isEditMode"
      />

      <ChannelMonitorRequestOverrideFields
        v-model:body-override-mode="form.body_override_mode"
        v-model:body-override-text="bodyOverrideText"
        v-model:extra-headers-text="extraHeadersText"
        :body-override-mode-options="bodyOverrideModeOptions"
      />
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="submit" form="channel-monitor-form" class="btn btn-primary" :disabled="submitting">
          <Icon v-if="submitting" name="refresh" size="md" class="mr-2 animate-spin" />
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import ChannelMonitorGeneralFields from './ChannelMonitorGeneralFields.vue'
import ChannelMonitorModelFields from './ChannelMonitorModelFields.vue'
import ChannelMonitorRequestOverrideFields from './ChannelMonitorRequestOverrideFields.vue'
import ChannelMonitorSecretPromptFields from './ChannelMonitorSecretPromptFields.vue'
import type { AdminChannelMonitor, AdminChannelMonitorTemplate } from '@/api/admin/channelMonitors'
import {
  getChannelMonitorBodyOverrideModeOptions,
  getChannelMonitorModelProbeStrategyOptions,
  getChannelMonitorProbeModeOptions,
  getChannelMonitorProviderOptions,
  getChannelMonitorRequestProtocolOptions
} from '@/utils/channelMonitorPresentation'
import { useChannelMonitorForm } from './useChannelMonitorForm'
import type { ChannelMonitorFormState } from './channelMonitorFormTypes'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  monitor: AdminChannelMonitor | null
  templates: AdminChannelMonitorTemplate[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
}>()

const dialogTitle = computed(() =>
  isEditMode.value ? t('admin.channelMonitors.actions.editMonitor') : t('admin.channelMonitors.actions.createMonitor')
)

const providerOptions = computed(() => getChannelMonitorProviderOptions())
const requestProtocolOptions = computed(() => getChannelMonitorRequestProtocolOptions())
const probeModeOptions = computed(() => getChannelMonitorProbeModeOptions())
const modelProbeStrategyOptions = computed(() => getChannelMonitorModelProbeStrategyOptions())
const bodyOverrideModeOptions = computed(() => getChannelMonitorBodyOverrideModeOptions())
const openAIApiModeOptions = computed(() => [
  { value: 'chat_completions', label: 'Chat Completions' },
  { value: 'responses', label: 'Responses' }
])

const {
  additionalModelOptions,
  additionalModelToAdd,
  additionalModels,
  additionalModelsText,
  addAdditionalModel,
  apiKey,
  apiKeyHintText,
  apiKeyPlaceholder,
  bodyOverrideText,
  clearApiKey,
  extraHeadersText,
  filteredAccounts,
  form,
  handleSubmit,
  isAccountMode,
  isDirectMode,
  isEditMode,
  loadingAccounts,
  loadingModels,
  modelOptions,
  modelSelectHint,
  removeAdditionalModel,
  submitting,
  templateOptions,
  toggleAccount,
  accountSearch
} = useChannelMonitorForm(props, () => emit('saved'))

function updateFormField<K extends keyof ChannelMonitorFormState>(field: K, value: ChannelMonitorFormState[K]) {
  form[field] = value
}
</script>
