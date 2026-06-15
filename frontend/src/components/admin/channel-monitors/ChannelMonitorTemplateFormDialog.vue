<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <form class="space-y-6" @submit.prevent="handleSubmit">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.channelMonitors.templateFields.name') }} <span class="text-red-500">*</span></label>
          <input v-model="form.name" type="text" class="input" />
        </div>

        <div>
          <label class="input-label">{{ t('admin.channelMonitors.templateFields.provider') }} <span class="text-red-500">*</span></label>
          <Select v-model="form.provider" :options="providerOptions" searchable>
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
          <Select v-model="form.request_protocol" :options="requestProtocolOptions" />
        </div>

        <div v-if="form.request_protocol === 'openai'">
          <label class="input-label">{{ t('admin.channelMonitors.fields.openaiApiMode') }}</label>
          <Select v-model="form.openai_api_mode" :options="openAIApiModeOptions" />
        </div>

        <div class="md:col-span-2">
          <label class="input-label">{{ t('admin.channelMonitors.templateFields.description') }}</label>
          <input v-model="form.description" type="text" class="input" />
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.channelMonitors.fields.testPromptTemplate') }}</label>
        <textarea
          v-model="form.test_prompt_template"
          rows="3"
          class="input"
          :placeholder="t('admin.channelMonitors.fields.testPromptTemplatePlaceholder')"
        ></textarea>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.channelMonitors.fields.testPromptTemplateHint') }}
        </p>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.channelMonitors.templateFields.extraHeaders') }}</label>
          <textarea v-model="extraHeadersText" rows="10" class="input font-mono"></textarea>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.channelMonitors.fields.jsonHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.channelMonitors.templateFields.bodyOverride') }}</label>
          <div class="mb-2">
            <Select v-model="form.body_override_mode" :options="bodyOverrideModeOptions" />
          </div>
          <textarea v-model="bodyOverrideText" rows="10" class="input font-mono"></textarea>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.channelMonitors.fields.bodyOverrideHint') }}</p>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="submitting" @click="handleSubmit">
          <Icon v-if="submitting" name="refresh" size="md" class="mr-2 animate-spin" />
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import ChannelMonitorProviderOption from './ChannelMonitorProviderOption.vue'
import type {
  AdminChannelMonitorTemplate,
  ChannelMonitorBodyOverrideMode,
  ChannelMonitorOpenAIAPIMode,
  CreateChannelMonitorTemplateRequest,
  UpdateChannelMonitorTemplateRequest
} from '@/api/admin/channelMonitors'
import {
  getChannelMonitorBodyOverrideModeOptions,
  getChannelMonitorProviderOptions,
  getChannelMonitorRequestProtocolOptions
} from '@/utils/channelMonitorPresentation'
import {
  inferChannelMonitorProtocol,
  parseHeaderRecord,
  parseJsonRecord,
  resolveChannelMonitorSaveErrorMessage
} from './channelMonitorFormHelpers'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  template: AdminChannelMonitorTemplate | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
}>()

const isEditMode = computed(() => props.template != null)
const submitting = ref(false)

const form = reactive({
  name: '',
  provider: 'openai',
  request_protocol: 'openai' as 'openai' | 'anthropic' | 'gemini',
  description: '',
  test_prompt_template: '',
  body_override_mode: 'off' as ChannelMonitorBodyOverrideMode,
  openai_api_mode: 'chat_completions' as ChannelMonitorOpenAIAPIMode
})

const extraHeadersText = ref('{}')
const bodyOverrideText = ref('{}')

const dialogTitle = computed(() =>
  isEditMode.value
    ? t('admin.channelMonitors.actions.editTemplate')
    : t('admin.channelMonitors.actions.createTemplate')
)

const providerOptions = computed(() => getChannelMonitorProviderOptions())
const requestProtocolOptions = computed(() => getChannelMonitorRequestProtocolOptions())

const bodyOverrideModeOptions = computed(() => getChannelMonitorBodyOverrideModeOptions())

const openAIApiModeOptions = computed(() => [
  { value: 'chat_completions', label: 'Chat Completions' },
  { value: 'responses', label: 'Responses' }
])

function resetForm() {
  form.name = ''
  form.provider = 'openai'
  form.request_protocol = 'openai'
  form.description = ''
  form.test_prompt_template = ''
  form.body_override_mode = 'off'
  form.openai_api_mode = 'chat_completions'
  extraHeadersText.value = '{}'
  bodyOverrideText.value = '{}'
}

function hydrateFromTemplate(tpl: AdminChannelMonitorTemplate) {
  form.name = tpl.name
  form.provider = tpl.provider
  form.request_protocol = (tpl.request_protocol || inferChannelMonitorProtocol(tpl.provider)) as 'openai' | 'anthropic' | 'gemini'
  form.description = tpl.description || ''
  form.test_prompt_template = tpl.test_prompt_template || ''
  form.body_override_mode = (tpl.body_override_mode || 'off') as ChannelMonitorBodyOverrideMode
  form.openai_api_mode = (tpl.openai_api_mode || 'chat_completions') as ChannelMonitorOpenAIAPIMode
  extraHeadersText.value = JSON.stringify(tpl.extra_headers || {}, null, 2)
  bodyOverrideText.value = JSON.stringify(tpl.body_override || {}, null, 2)
}

watch(
  () => [props.show, props.template] as const,
  ([show, tpl]) => {
    if (!show) return
    if (!tpl) {
      resetForm()
      return
    }
    hydrateFromTemplate(tpl)
  },
  { immediate: true }
)

watch(
  () => form.provider,
  (provider, previous) => {
    if (!previous || form.request_protocol === inferChannelMonitorProtocol(previous)) {
      form.request_protocol = inferChannelMonitorProtocol(provider)
    }
  }
)

async function handleSubmit() {
  if (submitting.value) return
  if (!form.name.trim()) {
    appStore.showError(t('admin.channelMonitors.validation.nameRequired'))
    return
  }
  if (!form.provider) {
    appStore.showError(t('admin.channelMonitors.validation.providerRequired'))
    return
  }

  let extraHeaders: Record<string, string> = {}
  let bodyOverride: Record<string, any> = {}
  try {
    extraHeaders = parseHeaderRecord(extraHeadersText.value)
    bodyOverride = parseJsonRecord(bodyOverrideText.value, {})
  } catch (err: any) {
    appStore.showError(
      err?.message === 'invalid_header_value'
        ? t('admin.channelMonitors.validation.invalidHeaders')
        : t('admin.channelMonitors.validation.invalidJson')
    )
    return
  }

  if (form.body_override_mode === 'replace' && Object.keys(bodyOverride).length === 0) {
    appStore.showError(t('admin.channelMonitors.validation.bodyOverrideRequired'))
    return
  }

  const openAIApiMode = form.request_protocol === 'openai' ? form.openai_api_mode : 'chat_completions'
  submitting.value = true
  try {
    if (!isEditMode.value) {
      const payload: CreateChannelMonitorTemplateRequest = {
        name: form.name.trim(),
        provider: form.provider,
        request_protocol: form.request_protocol,
        description: form.description.trim() || null,
        test_prompt_template: form.test_prompt_template.trim(),
        extra_headers: extraHeaders,
        body_override_mode: form.body_override_mode,
        body_override: bodyOverride,
        openai_api_mode: openAIApiMode
      }
      await adminAPI.channelMonitors.createTemplate(payload)
      appStore.showSuccess(t('admin.channelMonitors.messages.saved'))
      emit('saved')
      return
    }

    const id = props.template!.id
    const payload: UpdateChannelMonitorTemplateRequest = {
      name: form.name.trim(),
      provider: form.provider,
      request_protocol: form.request_protocol,
      description: form.description.trim() || null,
      test_prompt_template: form.test_prompt_template.trim(),
      extra_headers: extraHeaders,
      body_override_mode: form.body_override_mode,
      body_override: bodyOverride,
      openai_api_mode: openAIApiMode
    }

    await adminAPI.channelMonitors.updateTemplate(id, payload)
    appStore.showSuccess(t('admin.channelMonitors.messages.saved'))
    emit('saved')
  } catch (err: any) {
    appStore.showError(resolveChannelMonitorSaveErrorMessage(err, t('admin.channelMonitors.messages.saveFailed')))
  } finally {
    submitting.value = false
  }
}
</script>
