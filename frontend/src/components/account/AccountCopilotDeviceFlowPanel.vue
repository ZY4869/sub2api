<template>
  <div class="space-y-4 rounded-xl border border-cyan-200 bg-cyan-50/80 p-5 dark:border-cyan-900/40 dark:bg-cyan-950/20">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <h3 class="text-base font-semibold text-cyan-950 dark:text-cyan-100">
          {{ t('admin.accounts.copilotDeviceFlow.title') }}
        </h3>
        <p class="text-sm text-cyan-800 dark:text-cyan-300">
          {{ t('admin.accounts.copilotDeviceFlow.description') }}
        </p>
      </div>
      <span :class="statusClass">
        {{ statusLabel }}
      </span>
    </div>

    <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300">
      {{ errorMessage }}
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="rounded-lg border border-cyan-200 bg-white/80 p-4 dark:border-cyan-900/40 dark:bg-slate-900/60">
        <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-cyan-700 dark:text-cyan-300">
          {{ t('admin.accounts.copilotDeviceFlow.userCode') }}
        </div>
        <div class="rounded-md bg-slate-950 px-3 py-3 font-mono text-lg font-semibold tracking-[0.3em] text-cyan-300">
          {{ state?.user_code || '......' }}
        </div>
        <button
          type="button"
          class="btn btn-secondary mt-3 w-full"
          :disabled="!state?.user_code"
          @click="copyCode"
        >
          {{ t('admin.accounts.copilotDeviceFlow.copyCode') }}
        </button>
      </div>

      <div class="rounded-lg border border-cyan-200 bg-white/80 p-4 dark:border-cyan-900/40 dark:bg-slate-900/60">
        <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-cyan-700 dark:text-cyan-300">
          {{ t('admin.accounts.copilotDeviceFlow.verificationUrl') }}
        </div>
        <div class="break-all rounded-md bg-slate-100 px-3 py-3 text-sm text-slate-700 dark:bg-slate-800 dark:text-slate-200">
          {{ verificationUrl || t('admin.accounts.copilotDeviceFlow.notStarted') }}
        </div>
        <div class="mt-3 flex gap-2">
          <button
            type="button"
            class="btn btn-secondary flex-1"
            :disabled="!verificationUrl"
            @click="openVerification"
          >
            {{ t('admin.accounts.copilotDeviceFlow.openGitHub') }}
          </button>
          <button
            type="button"
            class="btn btn-secondary flex-1"
            :disabled="!verificationUrl"
            @click="copyUrl"
          >
            {{ t('admin.accounts.copilotDeviceFlow.copyUrl') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="state" class="rounded-lg border border-cyan-200 bg-white/80 p-4 text-sm text-slate-700 dark:border-cyan-900/40 dark:bg-slate-900/60 dark:text-slate-200">
      <p>{{ t('admin.accounts.copilotDeviceFlow.pollHint', { seconds: state.interval }) }}</p>
      <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
        {{ t('admin.accounts.copilotDeviceFlow.expiresHint', { seconds: expiresIn }) }}
      </p>
      <div v-if="userSummary" class="mt-3 rounded-md bg-emerald-50 px-3 py-2 text-emerald-700 dark:bg-emerald-950/20 dark:text-emerald-300">
        {{ t('admin.accounts.copilotDeviceFlow.authorizedAs', { user: userSummary }) }}
      </div>
    </div>

    <div class="flex flex-wrap gap-3">
      <button type="button" class="btn btn-secondary" :disabled="loading" @click="startFlow">
        {{ state ? t('admin.accounts.copilotDeviceFlow.restart') : t('admin.accounts.copilotDeviceFlow.start') }}
      </button>
      <button
        v-if="state && status !== 'completed'"
        type="button"
        class="btn btn-secondary"
        :disabled="loading"
        @click="pollNow"
      >
        {{ t('admin.accounts.copilotDeviceFlow.checkStatus') }}
      </button>
      <button
        v-if="status === 'completed'"
        type="button"
        class="btn btn-primary"
        :disabled="submitLoading"
        @click="emitSubmit"
      >
        {{ submitLoading ? t('common.loading') : submitLabel }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  CopilotDeviceFlowPollResult,
  CopilotDeviceFlowStartResult
} from '@/api/admin/accounts'
import { useClipboard } from '@/composables/useClipboard'

interface Props {
  proxyId?: number | null
  submitLabel: string
  submitLoading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  proxyId: null,
  submitLoading: false
})

const emit = defineEmits<{
  submit: [payload: { sessionId: string; user?: CopilotDeviceFlowPollResult['user'] }]
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const state = ref<CopilotDeviceFlowStartResult | null>(null)
const status = ref<'idle' | 'pending' | 'completed'>('idle')
const user = ref<CopilotDeviceFlowPollResult['user']>()
const errorMessage = ref('')
const loading = ref(false)
let pollTimer: ReturnType<typeof setTimeout> | null = null

const verificationUrl = computed(() => state.value?.verification_uri_complete || state.value?.verification_uri || '')
const expiresIn = computed(() => state.value?.expires_in || 0)
const userSummary = computed(() => user.value?.email || user.value?.login || user.value?.name || '')
const statusLabel = computed(() => t(`admin.accounts.copilotDeviceFlow.status.${status.value}`))
const statusClass = computed(() =>
  status.value === 'completed'
    ? 'rounded-full bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300'
    : status.value === 'pending'
      ? 'rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-700 dark:bg-amber-950/30 dark:text-amber-300'
      : 'rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-300'
)

function clearTimer() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function schedulePoll(seconds: number) {
  clearTimer()
  pollTimer = setTimeout(() => {
    void pollNow()
  }, Math.max(seconds, 2) * 1000)
}

async function startFlow() {
  clearTimer()
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await adminAPI.accounts.startCopilotDeviceFlow({ proxy_id: props.proxyId })
    state.value = result
    status.value = 'pending'
    user.value = undefined
    schedulePoll(result.interval)
  } catch (error: any) {
    errorMessage.value = error?.message || t('admin.accounts.copilotDeviceFlow.startFailed')
  } finally {
    loading.value = false
  }
}

async function pollNow() {
  if (!state.value || loading.value || status.value === 'completed') return
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await adminAPI.accounts.pollCopilotDeviceFlow(state.value.session_id)
    status.value = result.status
    user.value = result.user
    if (result.status === 'completed') {
      clearTimer()
    } else {
      schedulePoll(result.interval)
    }
  } catch (error: any) {
    clearTimer()
    errorMessage.value = error?.message || t('admin.accounts.copilotDeviceFlow.pollFailed')
  } finally {
    loading.value = false
  }
}

function openVerification() {
  if (verificationUrl.value) {
    window.open(verificationUrl.value, '_blank', 'noopener,noreferrer')
  }
}

function copyCode() {
  if (state.value?.user_code) {
    void copyToClipboard(state.value.user_code)
  }
}

function copyUrl() {
  if (verificationUrl.value) {
    void copyToClipboard(verificationUrl.value)
  }
}

function emitSubmit() {
  if (state.value && status.value === 'completed') {
    emit('submit', {
      sessionId: state.value.session_id,
      user: user.value
    })
  }
}

function reset() {
  clearTimer()
  state.value = null
  status.value = 'idle'
  user.value = undefined
  errorMessage.value = ''
  loading.value = false
}

onBeforeUnmount(clearTimer)

defineExpose({ reset })
</script>
