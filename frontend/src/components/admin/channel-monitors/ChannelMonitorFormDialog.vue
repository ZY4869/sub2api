<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="extra-wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <form id="channel-monitor-form" class="space-y-6" @submit.prevent="handleSubmit">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.name') }} <span class="text-red-500">*</span></label>
          <input v-model="form.name" type="text" class="input" />
        </div>

        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.provider') }} <span class="text-red-500">*</span></label>
          <Select
            v-model="form.provider"
            :options="providerOptions"
            :placeholder="t('common.selectOption')"
          />
        </div>

        <div class="md:col-span-2">
          <label class="input-label">{{ t('admin.channelMonitors.fields.endpoint') }} <span class="text-red-500">*</span></label>
          <input
            v-model="form.endpoint"
            type="text"
            class="input font-mono"
            :placeholder="t('admin.channelMonitors.fields.endpointPlaceholder')"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.intervalSeconds') }}</label>
          <input
            v-model.number="form.interval_seconds"
            type="number"
            class="input"
            min="15"
            max="3600"
          />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.channelMonitors.fields.intervalHint') }}
          </p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.enabled') }}</label>
          <div class="flex items-center gap-2">
            <Toggle v-model="form.enabled" />
            <span class="text-sm text-gray-600 dark:text-gray-300">
              {{ form.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.template') }}</label>
          <Select v-model="form.template_id" :options="templateOptions" />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.primaryModel') }} <span class="text-red-500">*</span></label>
          <input v-model="form.primary_model_id" type="text" class="input font-mono" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.additionalModels') }}</label>
          <input
            v-model="additionalModelsText"
            type="text"
            class="input font-mono"
            :placeholder="t('admin.channelMonitors.fields.additionalModelsPlaceholder')"
          />
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.channelMonitors.fields.apiKey') }}</label>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <input
            v-model="apiKey"
            type="password"
            class="input font-mono"
            :placeholder="apiKeyPlaceholder"
          />
          <div v-if="isEditMode" class="flex items-center gap-2">
            <Toggle v-model="clearApiKey" />
            <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.channelMonitors.fields.clearApiKey') }}</span>
          </div>
        </div>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ apiKeyHintText }}
        </p>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.extraHeaders') }}</label>
          <textarea v-model="extraHeadersText" rows="8" class="input font-mono"></textarea>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.channelMonitors.fields.jsonHint') }}
          </p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.channelMonitors.fields.bodyOverride') }}</label>
          <div class="mb-2">
            <Select v-model="form.body_override_mode" :options="bodyOverrideModeOptions" />
          </div>
          <textarea v-model="bodyOverrideText" rows="8" class="input font-mono"></textarea>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.channelMonitors.fields.bodyOverrideHint') }}
          </p>
        </div>
      </div>
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
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type {
  AdminChannelMonitor,
  AdminChannelMonitorTemplate,
  ChannelMonitorBodyOverrideMode,
  CreateChannelMonitorRequest,
  UpdateChannelMonitorRequest
} from '@/api/admin/channelMonitors'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  monitor: AdminChannelMonitor | null
  templates: AdminChannelMonitorTemplate[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
}>()

const isEditMode = computed(() => props.monitor != null)

const form = reactive({
  name: '',
  provider: 'openai',
  endpoint: '',
  interval_seconds: 60,
  enabled: false,
  primary_model_id: '',
  template_id: null as number | null,
  body_override_mode: 'off' as ChannelMonitorBodyOverrideMode
})

const additionalModelsText = ref('')
const apiKey = ref('')
const clearApiKey = ref(false)
const extraHeadersText = ref('{}')
const bodyOverrideText = ref('{}')
const submitting = ref(false)

const dialogTitle = computed(() =>
  isEditMode.value
    ? t('admin.channelMonitors.actions.editMonitor')
    : t('admin.channelMonitors.actions.createMonitor')
)

const providerOptions = computed(() => [
  { value: 'openai', label: 'openai' },
  { value: 'anthropic', label: 'anthropic' },
  { value: 'gemini', label: 'gemini' },
  { value: 'grok', label: 'grok' },
  { value: 'antigravity', label: 'antigravity' }
])

const templateOptions = computed(() => {
  const templates = Array.isArray(props.templates) ? props.templates : []
  return [
    { value: null, label: t('admin.channelMonitors.fields.templateNone') },
    ...templates.map((tpl) => ({ value: tpl.id, label: `${tpl.name} (${tpl.provider})` }))
  ]
})

const bodyOverrideModeOptions = computed(() => [
  { value: 'off', label: 'off' },
  { value: 'merge', label: 'merge' },
  { value: 'replace', label: 'replace' }
])

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

function parseJsonRecord(text: string, fallback: Record<string, any> = {}): Record<string, any> {
  const trimmed = String(text || '').trim()
  if (!trimmed) return fallback
  const parsed = JSON.parse(trimmed)
  if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
    return parsed as Record<string, any>
  }
  throw new Error('invalid_json_object')
}

function normalizeModels(text: string): string[] {
  return String(text || '')
    .split(/[\s,]+/)
    .map(s => s.trim())
    .filter(Boolean)
}

function resolveSaveErrorMessage(err: any): string {
  return err?.response?.data?.detail ||
    err?.response?.data?.message ||
    err?.message ||
    t('admin.channelMonitors.messages.saveFailed')
}

function resetForm() {
  form.name = ''
  form.provider = 'openai'
  form.endpoint = ''
  form.interval_seconds = 60
  form.enabled = false
  form.primary_model_id = ''
  form.template_id = null
  form.body_override_mode = 'off'
  additionalModelsText.value = ''
  apiKey.value = ''
  clearApiKey.value = false
  extraHeadersText.value = '{}'
  bodyOverrideText.value = '{}'
}

function hydrateFromMonitor(m: AdminChannelMonitor) {
  form.name = m.name
  form.provider = m.provider
  form.endpoint = m.endpoint
  form.interval_seconds = m.interval_seconds
  form.enabled = m.enabled
  form.primary_model_id = m.primary_model_id
  form.template_id = m.template_id ?? null
  form.body_override_mode = (m.body_override_mode || 'off') as ChannelMonitorBodyOverrideMode
  additionalModelsText.value = (m.additional_model_ids || []).join(', ')
  apiKey.value = ''
  clearApiKey.value = false
  extraHeadersText.value = JSON.stringify(m.extra_headers || {}, null, 2)
  bodyOverrideText.value = JSON.stringify(m.body_override || {}, null, 2)
}

watch(
  () => [props.show, props.monitor] as const,
  ([show, monitor]) => {
    if (!show) return
    if (!monitor) {
      resetForm()
      return
    }
    hydrateFromMonitor(monitor)
  },
  { immediate: true }
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
  if (!form.endpoint.trim()) {
    appStore.showError(t('admin.channelMonitors.validation.endpointRequired'))
    return
  }
  if (!form.primary_model_id.trim()) {
    appStore.showError(t('admin.channelMonitors.validation.primaryModelRequired'))
    return
  }

  const needsKey =
    (!isEditMode.value && form.enabled) ||
    (isEditMode.value && (props.monitor?.api_key_decrypt_failed || (!props.monitor?.api_key_configured && form.enabled)))

  if (needsKey && !apiKey.value.trim() && !clearApiKey.value) {
    appStore.showError(t('admin.channelMonitors.validation.apiKeyRequired'))
    return
  }

  let extraHeaders: Record<string, any> = {}
  let bodyOverride: Record<string, any> = {}
  try {
    extraHeaders = parseJsonRecord(extraHeadersText.value, {})
    bodyOverride = parseJsonRecord(bodyOverrideText.value, {})
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.validation.invalidJson'))
    return
  }

  const additionalModels = normalizeModels(additionalModelsText.value)

  submitting.value = true
  try {
    if (!isEditMode.value) {
      const payload: CreateChannelMonitorRequest = {
        name: form.name.trim(),
        provider: form.provider,
        endpoint: form.endpoint.trim(),
        api_key: apiKey.value.trim() || undefined,
        interval_seconds: form.interval_seconds,
        enabled: form.enabled,
        primary_model_id: form.primary_model_id.trim(),
        additional_model_ids: additionalModels,
        template_id: form.template_id,
        extra_headers: extraHeaders,
        body_override_mode: form.body_override_mode,
        body_override: bodyOverride
      }
      await adminAPI.channelMonitors.createMonitor(payload)
      appStore.showSuccess(t('admin.channelMonitors.messages.saved'))
      emit('saved')
      return
    }

    const id = props.monitor!.id
    const payload: UpdateChannelMonitorRequest = {
      name: form.name.trim(),
      provider: form.provider,
      endpoint: form.endpoint.trim(),
      interval_seconds: form.interval_seconds,
      enabled: form.enabled,
      primary_model_id: form.primary_model_id.trim(),
      additional_model_ids: additionalModels,
      template_id: form.template_id,
      extra_headers: extraHeaders,
      body_override_mode: form.body_override_mode,
      body_override: bodyOverride
    }

    if (clearApiKey.value) {
      payload.api_key = null
    } else if (apiKey.value.trim()) {
      payload.api_key = apiKey.value.trim()
    }

    await adminAPI.channelMonitors.updateMonitor(id, payload)
    appStore.showSuccess(t('admin.channelMonitors.messages.saved'))
    emit('saved')
  } catch (err: any) {
    appStore.showError(resolveSaveErrorMessage(err))
  } finally {
    submitting.value = false
  }
}
</script>
