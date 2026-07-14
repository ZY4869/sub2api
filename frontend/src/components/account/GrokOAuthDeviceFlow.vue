<template>
  <div class="space-y-4 rounded-lg border border-emerald-200 bg-white/80 p-4 dark:border-emerald-900/40 dark:bg-slate-900/60">
    <div v-if="message" :class="messageClass">
      {{ message }}
    </div>

    <div class="flex flex-wrap gap-3">
      <button type="button" class="btn btn-secondary" :disabled="loading || submitting" data-testid="grok-device-start" @click="startDeviceFlow">
        {{ deviceState ? t('admin.accounts.grokOauth.deviceRegenerate') : t('admin.accounts.grokOauth.deviceStart') }}
      </button>
      <button type="button" class="btn btn-secondary" :disabled="!deviceState?.verification_uri_complete && !deviceState?.verification_uri" @click="openVerificationUrl">
        {{ t('admin.accounts.grokOauth.openDeviceUrl') }}
      </button>
      <button type="button" class="btn btn-secondary" :disabled="!deviceState?.user_code" @click="copyUserCode">
        {{ t('admin.accounts.grokOauth.copyDeviceCode') }}
      </button>
    </div>

    <div v-if="deviceState" class="grid gap-3 md:grid-cols-2">
      <div class="rounded-md bg-slate-100 px-3 py-3 dark:bg-slate-800">
        <div class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400">
          {{ t('admin.accounts.grokOauth.deviceUserCode') }}
        </div>
        <div class="mt-2 break-all font-mono text-lg font-black tracking-wide text-slate-800 dark:text-slate-100" data-testid="grok-device-user-code">
          {{ deviceState.user_code }}
        </div>
      </div>
      <div class="rounded-md bg-slate-100 px-3 py-3 dark:bg-slate-800">
        <div class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400">
          {{ t('admin.accounts.grokOauth.deviceExpiresIn') }}
        </div>
        <div class="mt-2 text-sm font-semibold text-slate-700 dark:text-slate-200" data-testid="grok-device-countdown">
          {{ countdownText }}
        </div>
      </div>
      <div class="md:col-span-2">
        <div class="mb-2 text-xs font-semibold uppercase text-emerald-700 dark:text-emerald-300">
          {{ t('admin.accounts.grokOauth.deviceUrl') }}
        </div>
        <div class="break-all rounded-md bg-slate-100 px-3 py-3 text-sm text-slate-700 dark:bg-slate-800 dark:text-slate-200">
          {{ deviceState.verification_uri_complete || deviceState.verification_uri }}
        </div>
      </div>
    </div>

    <div v-if="deviceState" class="flex flex-wrap items-center gap-3">
      <button
        type="button"
        class="btn btn-primary"
        :disabled="loading || submitting || terminalStatus"
        data-testid="grok-device-poll"
        @click="pollDeviceToken('manual')"
      >
        {{ loading ? t('common.loading') : submitLabel }}
      </button>
      <span class="text-xs text-emerald-700 dark:text-emerald-300">
        {{ t('admin.accounts.grokOauth.devicePollingHint', { seconds: pollInterval }) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useClipboard } from '@/composables/useClipboard'
import {
  buildGrokOAuthPayload,
  type GrokDeviceFlowStartResult,
  type ParsedGrokOAuthPayload
} from '@/utils/grokOAuth'

const props = withDefaults(defineProps<{
  proxyId?: number | null
  submitLabel: string
  submitting?: boolean
  initialMessage?: string
}>(), {
  proxyId: null,
  submitting: false,
  initialMessage: ''
})

const emit = defineEmits<{
  submit: [payload: ParsedGrokOAuthPayload]
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const deviceState = ref<GrokDeviceFlowStartResult | null>(null)
const message = ref(props.initialMessage)
const messageTone = ref<'info' | 'warning' | 'error'>('info')
const loading = ref(false)
const pollInterval = ref(5)
const pollTimer = ref<number | null>(null)
const countdownTimer = ref<number | null>(null)
const nowSeconds = ref(Math.floor(Date.now() / 1000))
const terminalStatus = ref(false)

const countdownText = computed(() => {
  if (!deviceState.value?.expires_at) return '--'
  const remaining = Math.max(0, deviceState.value.expires_at - nowSeconds.value)
  const minutes = Math.floor(remaining / 60)
  const seconds = remaining % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
})

const messageClass = computed(() => [
  'rounded-lg border p-3 text-sm',
  messageTone.value === 'error'
    ? 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300'
    : messageTone.value === 'warning'
      ? 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-200'
      : 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/20 dark:text-emerald-200'
])

watch(
  () => props.initialMessage,
  (value) => {
    if (value && !deviceState.value) {
      message.value = value
      messageTone.value = 'warning'
    }
  }
)

async function startDeviceFlow() {
  stopPolling()
  terminalStatus.value = false
  loading.value = true
  message.value = ''
  try {
    deviceState.value = await adminAPI.accounts.startGrokDeviceFlow({ proxy_id: props.proxyId })
    pollInterval.value = Math.max(1, deviceState.value.interval || 5)
    message.value = t('admin.accounts.grokOauth.deviceStarted')
    messageTone.value = 'info'
    startCountdown()
    schedulePoll()
  } catch (error: any) {
    message.value = error?.message || t('admin.accounts.grokOauth.deviceStartFailed')
    messageTone.value = 'error'
  } finally {
    loading.value = false
  }
}

async function pollDeviceToken(source: 'auto' | 'manual') {
  if (!deviceState.value?.session_id || terminalStatus.value || loading.value) return
  if (source === 'manual') stopPolling()
  loading.value = true
  try {
    const result = await adminAPI.accounts.pollGrokDeviceToken({
      session_id: deviceState.value.session_id,
      proxy_id: props.proxyId
    })
    pollInterval.value = Math.max(1, result.interval || pollInterval.value)
    if (result.expires_at) deviceState.value.expires_at = result.expires_at

    if (result.status === 'authorized' && result.token_info) {
      terminalStatus.value = true
      stopPolling()
      emit('submit', buildGrokOAuthPayload(result.token_info))
      return
    }
    if (result.status === 'denied' || result.status === 'expired') {
      terminalStatus.value = true
      stopPolling()
      message.value = result.error_message || t(`admin.accounts.grokOauth.device${result.status === 'denied' ? 'Denied' : 'Expired'}`)
      messageTone.value = 'error'
      return
    }
    if (result.status === 'slow_down') {
      message.value = result.error_message || t('admin.accounts.grokOauth.deviceSlowDown')
      messageTone.value = 'warning'
      schedulePoll()
      return
    }
    message.value = result.error_message || t('admin.accounts.grokOauth.devicePending')
    messageTone.value = 'info'
    schedulePoll()
  } catch (error: any) {
    message.value = error?.message || t('admin.accounts.grokOauth.devicePollFailed')
    messageTone.value = 'error'
  } finally {
    loading.value = false
  }
}

function schedulePoll() {
  stopPolling()
  if (!deviceState.value || terminalStatus.value) return
  pollTimer.value = window.setTimeout(() => {
    void pollDeviceToken('auto')
  }, pollInterval.value * 1000)
}

function stopPolling() {
  if (pollTimer.value) {
    window.clearTimeout(pollTimer.value)
    pollTimer.value = null
  }
}

function startCountdown() {
  if (countdownTimer.value) window.clearInterval(countdownTimer.value)
  nowSeconds.value = Math.floor(Date.now() / 1000)
  countdownTimer.value = window.setInterval(() => {
    nowSeconds.value = Math.floor(Date.now() / 1000)
    if (deviceState.value?.expires_at && nowSeconds.value >= deviceState.value.expires_at) {
      terminalStatus.value = true
      stopPolling()
      message.value = t('admin.accounts.grokOauth.deviceExpired')
      messageTone.value = 'error'
    }
  }, 1000)
}

function openVerificationUrl() {
  const url = deviceState.value?.verification_uri_complete || deviceState.value?.verification_uri
  if (url) window.open(url, '_blank', 'noopener,noreferrer')
}

function copyUserCode() {
  if (deviceState.value?.user_code) void copyToClipboard(deviceState.value.user_code)
}

function showExternalCodeHint() {
  message.value = t('admin.accounts.grokOauth.externalDeviceCodeHint')
  messageTone.value = 'warning'
}

function reset() {
  stopPolling()
  if (countdownTimer.value) {
    window.clearInterval(countdownTimer.value)
    countdownTimer.value = null
  }
  deviceState.value = null
  message.value = ''
  messageTone.value = 'info'
  loading.value = false
  pollInterval.value = 5
  terminalStatus.value = false
}

onBeforeUnmount(reset)

defineExpose({ reset, showExternalCodeHint })
</script>
