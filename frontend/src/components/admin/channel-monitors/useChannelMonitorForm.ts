import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminChannelMonitor, AdminChannelMonitorTemplate } from '@/api/admin/channelMonitors'
import { inferChannelMonitorProtocol } from './channelMonitorFormHelpers'
import type { ChannelMonitorFormState } from './channelMonitorFormTypes'
import { useChannelMonitorAccountModels } from './useChannelMonitorAccountModels'
import { useChannelMonitorSubmit } from './useChannelMonitorSubmit'

interface UseChannelMonitorFormProps {
  show: boolean
  monitor: AdminChannelMonitor | null
  templates: AdminChannelMonitorTemplate[]
}

type EmitFn = (event: 'saved') => void

export function useChannelMonitorForm(props: Readonly<UseChannelMonitorFormProps>, emit: EmitFn) {
  const { t } = useI18n()

  const form = reactive<ChannelMonitorFormState>({
    name: '',
    provider: 'openai',
    probe_mode: 'direct',
    request_protocol: 'openai',
    endpoint: '',
    interval_seconds: 60,
    jitter_seconds: 0,
    enabled: false,
    account_ids: [] as number[],
    primary_model_id: '',
    template_id: null as number | null,
    model_probe_strategy: 'primary_only',
    test_prompt_template: '',
    body_override_mode: 'off',
    openai_api_mode: 'chat_completions',
    save_as_template: false,
    template_name: ''
  })

  const additionalModelsText = ref('')
  const apiKey = ref('')
  const clearApiKey = ref(false)
  const extraHeadersText = ref('{}')
  const bodyOverrideText = ref('{}')

  const monitorRef = computed(() => props.monitor)
  const isEditMode = computed(() => props.monitor != null)
  const isDirectMode = computed(() => form.probe_mode === 'direct')
  const isAccountMode = computed(() => form.probe_mode === 'account_pool')

  const {
    additionalModelOptions,
    additionalModelToAdd,
    additionalModels,
    addAdditionalModel,
    accountSearch,
    availableModels,
    filteredAccounts,
    loadAccounts,
    loadSharedModels,
    loadingAccounts,
    loadingModels,
    modelOptions,
    modelSelectHint,
    removeAdditionalModel,
    toggleAccount
  } = useChannelMonitorAccountModels(form, isAccountMode)

  const { handleSubmit, submitting } = useChannelMonitorSubmit({
    form,
    monitor: monitorRef,
    additionalModels,
    availableModels,
    additionalModelsText,
    apiKey,
    clearApiKey,
    extraHeadersText,
    bodyOverrideText,
    isAccountMode,
    isDirectMode,
    isEditMode,
    onSaved: () => emit('saved')
  })

  const templateOptions = computed(() => {
    const templates = Array.isArray(props.templates) ? props.templates : []
    return [
      { value: null, label: t('admin.channelMonitors.fields.templateNone') },
      ...templates.map((tpl) => ({ value: tpl.id, label: `${tpl.name} (${tpl.provider})` }))
    ]
  })

  const apiKeyPlaceholder = computed(() => {
    if (!isEditMode.value) return t('admin.channelMonitors.fields.apiKeyPlaceholder')
    if (props.monitor?.api_key_decrypt_failed) return t('admin.channelMonitors.fields.apiKeyDecryptFailedPlaceholder')
    if (props.monitor?.api_key_configured) return t('admin.channelMonitors.fields.apiKeyConfiguredPlaceholder')
    return t('admin.channelMonitors.fields.apiKeyPlaceholder')
  })

  const apiKeyHintText = computed(() => {
    if (props.monitor?.api_key_decrypt_failed) return t('admin.channelMonitors.fields.apiKeyDecryptFailedHint')
    return t('admin.channelMonitors.fields.apiKeyHint')
  })

  function resetForm() {
    form.name = ''
    form.provider = 'openai'
    form.probe_mode = 'direct'
    form.request_protocol = 'openai'
    form.endpoint = ''
    form.interval_seconds = 60
    form.jitter_seconds = 0
    form.enabled = false
    form.account_ids = []
    form.primary_model_id = ''
    form.template_id = null
    form.model_probe_strategy = 'primary_only'
    form.test_prompt_template = ''
    form.body_override_mode = 'off'
    form.openai_api_mode = 'chat_completions'
    form.save_as_template = false
    form.template_name = ''
    additionalModelsText.value = ''
    additionalModels.value = []
    apiKey.value = ''
    clearApiKey.value = false
    extraHeadersText.value = '{}'
    bodyOverrideText.value = '{}'
  }

  function hydrateFromMonitor(monitor: AdminChannelMonitor) {
    form.name = monitor.name
    form.provider = monitor.provider || 'openai'
    form.probe_mode = (monitor.probe_mode || 'direct') as ChannelMonitorFormState['probe_mode']
    form.request_protocol = (monitor.request_protocol || inferChannelMonitorProtocol(form.provider)) as ChannelMonitorFormState['request_protocol']
    form.endpoint = monitor.endpoint || ''
    form.interval_seconds = monitor.interval_seconds
    form.jitter_seconds = monitor.jitter_seconds || 0
    form.enabled = monitor.enabled
    form.account_ids = Array.isArray(monitor.account_ids) ? [...monitor.account_ids] : []
    form.primary_model_id = monitor.primary_model_id
    form.template_id = monitor.template_id ?? null
    form.model_probe_strategy = (monitor.model_probe_strategy || 'primary_only') as ChannelMonitorFormState['model_probe_strategy']
    form.test_prompt_template = monitor.test_prompt_template || ''
    form.body_override_mode = (monitor.body_override_mode || 'off') as ChannelMonitorFormState['body_override_mode']
    form.openai_api_mode = (monitor.openai_api_mode || 'chat_completions') as ChannelMonitorFormState['openai_api_mode']
    form.save_as_template = false
    form.template_name = ''
    additionalModels.value = [...(monitor.additional_model_ids || [])]
    additionalModelsText.value = (monitor.additional_model_ids || []).join(', ')
    apiKey.value = ''
    clearApiKey.value = false
    extraHeadersText.value = JSON.stringify(monitor.extra_headers || {}, null, 2)
    bodyOverrideText.value = JSON.stringify(monitor.body_override || {}, null, 2)
  }

  watch(
    () => [props.show, props.monitor] as const,
    ([show, monitor]) => {
      if (!show) return
      if (!monitor) resetForm()
      else hydrateFromMonitor(monitor)
      if (monitor?.probe_mode === 'account_pool' || form.probe_mode === 'account_pool') loadAccounts()
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

  watch(
    () => form.probe_mode,
    (mode, previous) => {
      if (mode !== 'account_pool') return
      if (previous !== 'account_pool') form.model_probe_strategy = 'primary_only'
      loadAccounts()
      loadSharedModels()
    }
  )

  watch(
    () => form.account_ids.join(','),
    () => loadSharedModels()
  )

  return {
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
  }
}
