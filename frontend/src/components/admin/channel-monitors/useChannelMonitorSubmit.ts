import { ref, type ComputedRef, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import type {
  AdminChannelMonitor,
  CreateChannelMonitorRequest,
  UpdateChannelMonitorRequest
} from '@/api/admin/channelMonitors'
import type { ChannelMonitorFormState } from './channelMonitorFormTypes'
import {
  normalizeMonitorModels,
  parseHeaderRecord,
  parseJsonRecord,
  resolveChannelMonitorSaveErrorMessage
} from './channelMonitorFormHelpers'

interface SubmitOptions {
  form: ChannelMonitorFormState
  monitor: ComputedRef<AdminChannelMonitor | null>
  additionalModels: Ref<string[]>
  availableModels: Ref<Array<{ id: string; source_protocol?: string }>>
  additionalModelsText: Ref<string>
  apiKey: Ref<string>
  clearApiKey: Ref<boolean>
  extraHeadersText: Ref<string>
  bodyOverrideText: Ref<string>
  isAccountMode: ComputedRef<boolean>
  isDirectMode: ComputedRef<boolean>
  isEditMode: ComputedRef<boolean>
  onSaved: () => void
}

export function useChannelMonitorSubmit(options: SubmitOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()
  const submitting = ref(false)

  async function handleSubmit() {
    if (submitting.value) return
    const { form } = options
    if (!form.name.trim()) return appStore.showError(t('admin.channelMonitors.validation.nameRequired'))
    if (!form.provider) return appStore.showError(t('admin.channelMonitors.validation.providerRequired'))
    if (options.isDirectMode.value && !form.endpoint.trim()) return appStore.showError(t('admin.channelMonitors.validation.endpointRequired'))
    if (options.isAccountMode.value && form.account_ids.length === 0) return appStore.showError(t('admin.channelMonitors.validation.accountsRequired'))
    if (!form.primary_model_id.trim()) return appStore.showError(t('admin.channelMonitors.validation.primaryModelRequired'))

    const monitor = options.monitor.value
    const needsKey =
      options.isDirectMode.value &&
      ((!options.isEditMode.value && form.enabled) ||
        (options.isEditMode.value && (monitor?.api_key_decrypt_failed || (!monitor?.api_key_configured && form.enabled))))
    if (needsKey && !options.apiKey.value.trim() && !options.clearApiKey.value) {
      return appStore.showError(t('admin.channelMonitors.validation.apiKeyRequired'))
    }

    let extraHeaders: Record<string, string> = {}
    let bodyOverride: Record<string, any> = {}
    try {
      extraHeaders = parseHeaderRecord(options.extraHeadersText.value)
      bodyOverride = parseJsonRecord(options.bodyOverrideText.value, {})
    } catch (err: any) {
      return appStore.showError(err?.message === 'invalid_header_value'
        ? t('admin.channelMonitors.validation.invalidHeaders')
        : t('admin.channelMonitors.validation.invalidJson'))
    }
    if (form.body_override_mode === 'replace' && Object.keys(bodyOverride).length === 0) {
      return appStore.showError(t('admin.channelMonitors.validation.bodyOverrideRequired'))
    }

    const extraModels = options.isAccountMode.value
      ? options.additionalModels.value
      : normalizeMonitorModels(options.additionalModelsText.value)
    const selectedModelIDs = [form.primary_model_id.trim(), ...extraModels]
    const modelSourceProtocols = buildModelSourceProtocols(
      selectedModelIDs,
      options.availableModels.value,
      options.monitor.value?.model_source_protocols || {}
    )
    const basePayload = {
      name: form.name.trim(),
      provider: form.provider,
      probe_mode: form.probe_mode,
      request_protocol: form.request_protocol,
      endpoint: options.isDirectMode.value ? form.endpoint.trim() : '',
      interval_seconds: form.interval_seconds,
      enabled: form.enabled,
      account_ids: options.isAccountMode.value ? form.account_ids : [],
      primary_model_id: form.primary_model_id.trim(),
      additional_model_ids: extraModels,
      model_source_protocols: options.isAccountMode.value ? modelSourceProtocols : {},
      model_probe_strategy: options.isAccountMode.value ? form.model_probe_strategy : 'all_selected',
      test_prompt_template: form.test_prompt_template.trim(),
      template_id: form.template_id,
      extra_headers: extraHeaders,
      body_override_mode: form.body_override_mode,
      body_override: bodyOverride,
      openai_api_mode: form.request_protocol === 'openai' ? form.openai_api_mode : 'chat_completions'
    }

    submitting.value = true
    try {
      if (!options.isEditMode.value) {
        const payload: CreateChannelMonitorRequest = {
          ...basePayload,
          api_key: options.isDirectMode.value ? (options.apiKey.value.trim() || undefined) : undefined,
          save_as_template: form.save_as_template,
          template_name: form.template_name.trim() || undefined
        }
        await adminAPI.channelMonitors.createMonitor(payload)
      } else {
        const payload: UpdateChannelMonitorRequest = { ...basePayload }
        if (options.isDirectMode.value) {
          if (options.clearApiKey.value) payload.api_key = null
          else if (options.apiKey.value.trim()) payload.api_key = options.apiKey.value.trim()
        }
        await adminAPI.channelMonitors.updateMonitor(monitor!.id, payload)
      }
      appStore.showSuccess(t('admin.channelMonitors.messages.saved'))
      options.onSaved()
    } catch (err: any) {
      appStore.showError(resolveChannelMonitorSaveErrorMessage(err, t('admin.channelMonitors.messages.saveFailed')))
    } finally {
      submitting.value = false
    }
  }

  return {
    handleSubmit,
    submitting
  }
}

function buildModelSourceProtocols(
  modelIDs: string[],
  availableModels: Array<{ id: string; source_protocol?: string }>,
  fallback: Record<string, string>
): Record<string, string> {
  const byID = new Map(availableModels.map((model) => [model.id, model.source_protocol || '']))
  const out: Record<string, string> = {}
  for (const id of modelIDs) {
    const protocol = byID.get(id) || fallback[id]
    if (protocol === 'openai' || protocol === 'anthropic' || protocol === 'gemini') {
      out[id] = protocol
    }
  }
  return out
}
