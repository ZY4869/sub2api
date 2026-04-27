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
          <Select v-model="form.provider" :options="providerOptions" />
        </div>

        <div class="md:col-span-2">
          <label class="input-label">{{ t('admin.channelMonitors.templateFields.description') }}</label>
          <input v-model="form.description" type="text" class="input" />
        </div>
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
import type {
  AdminChannelMonitorTemplate,
  ChannelMonitorBodyOverrideMode,
  CreateChannelMonitorTemplateRequest,
  UpdateChannelMonitorTemplateRequest
} from '@/api/admin/channelMonitors'

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
  description: '',
  body_override_mode: 'off' as ChannelMonitorBodyOverrideMode
})

const extraHeadersText = ref('{}')
const bodyOverrideText = ref('{}')

const dialogTitle = computed(() =>
  isEditMode.value
    ? t('admin.channelMonitors.actions.editTemplate')
    : t('admin.channelMonitors.actions.createTemplate')
)

const providerOptions = computed(() => [
  { value: 'openai', label: 'openai' },
  { value: 'anthropic', label: 'anthropic' },
  { value: 'gemini', label: 'gemini' },
  { value: 'grok', label: 'grok' },
  { value: 'antigravity', label: 'antigravity' }
])

const bodyOverrideModeOptions = computed(() => [
  { value: 'off', label: 'off' },
  { value: 'merge', label: 'merge' },
  { value: 'replace', label: 'replace' }
])

function parseJsonRecord(text: string, fallback: Record<string, any> = {}): Record<string, any> {
  const trimmed = String(text || '').trim()
  if (!trimmed) return fallback
  const parsed = JSON.parse(trimmed)
  if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
    return parsed as Record<string, any>
  }
  throw new Error('invalid_json_object')
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
  form.description = ''
  form.body_override_mode = 'off'
  extraHeadersText.value = '{}'
  bodyOverrideText.value = '{}'
}

function hydrateFromTemplate(tpl: AdminChannelMonitorTemplate) {
  form.name = tpl.name
  form.provider = tpl.provider
  form.description = tpl.description || ''
  form.body_override_mode = (tpl.body_override_mode || 'off') as ChannelMonitorBodyOverrideMode
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

  let extraHeaders: Record<string, any> = {}
  let bodyOverride: Record<string, any> = {}
  try {
    extraHeaders = parseJsonRecord(extraHeadersText.value, {})
    bodyOverride = parseJsonRecord(bodyOverrideText.value, {})
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.validation.invalidJson'))
    return
  }

  submitting.value = true
  try {
    if (!isEditMode.value) {
      const payload: CreateChannelMonitorTemplateRequest = {
        name: form.name.trim(),
        provider: form.provider,
        description: form.description.trim() || null,
        extra_headers: extraHeaders,
        body_override_mode: form.body_override_mode,
        body_override: bodyOverride
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
      description: form.description.trim() || null,
      extra_headers: extraHeaders,
      body_override_mode: form.body_override_mode,
      body_override: bodyOverride
    }

    await adminAPI.channelMonitors.updateTemplate(id, payload)
    appStore.showSuccess(t('admin.channelMonitors.messages.saved'))
    emit('saved')
  } catch (err: any) {
    appStore.showError(resolveSaveErrorMessage(err))
  } finally {
    submitting.value = false
  }
}
</script>
