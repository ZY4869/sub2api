<template>
  <div class="space-y-4 rounded-lg border border-emerald-200 bg-emerald-50/70 p-4 dark:border-emerald-900/40 dark:bg-emerald-950/20">
    <div class="space-y-1">
      <h3 class="text-sm font-semibold text-emerald-950 dark:text-emerald-100">
        {{ t('admin.accounts.grokOauth.title') }}
      </h3>
      <p class="text-xs leading-5 text-emerald-800 dark:text-emerald-300">
        {{ t('admin.accounts.grokOauth.description') }}
      </p>
    </div>

    <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300">
      {{ errorMessage }}
    </div>

    <div class="rounded-lg border border-emerald-200 bg-white/80 p-4 dark:border-emerald-900/40 dark:bg-slate-900/60">
      <div class="flex flex-wrap gap-3">
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="loading"
          @click="generateAuthUrl"
        >
          {{ authState ? t('admin.accounts.grokOauth.regenerate') : t('admin.accounts.grokOauth.generate') }}
        </button>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="!authState?.auth_url"
          @click="openAuthUrl"
        >
          {{ t('admin.accounts.grokOauth.openAuthUrl') }}
        </button>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="!authState?.auth_url"
          @click="copyAuthUrl"
        >
          {{ t('admin.accounts.grokOauth.copyAuthUrl') }}
        </button>
      </div>

      <div v-if="authState" class="mt-4 space-y-3">
        <div>
          <div class="mb-2 text-xs font-semibold uppercase text-emerald-700 dark:text-emerald-300">
            {{ t('admin.accounts.grokOauth.authUrl') }}
          </div>
          <div class="break-all rounded-md bg-slate-100 px-3 py-3 text-sm text-slate-700 dark:bg-slate-800 dark:text-slate-200">
            {{ authState.auth_url }}
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div>
            <div class="mb-2 text-xs font-semibold uppercase text-emerald-700 dark:text-emerald-300">
              {{ t('admin.accounts.grokOauth.redirectUri') }}
            </div>
            <div class="break-all rounded-md bg-slate-100 px-3 py-3 text-sm text-slate-700 dark:bg-slate-800 dark:text-slate-200">
              {{ authState.redirect_uri }}
            </div>
          </div>
          <div>
            <div class="mb-2 text-xs font-semibold uppercase text-emerald-700 dark:text-emerald-300">
              {{ t('admin.accounts.grokOauth.oauthState') }}
            </div>
            <div class="break-all rounded-md bg-slate-100 px-3 py-3 font-mono text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-200">
              {{ authState.state }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="rounded-lg border border-emerald-200 bg-white/80 p-4 dark:border-emerald-900/40 dark:bg-slate-900/60">
      <label class="input-label">{{ t('admin.accounts.grokOauth.callbackUrl') }}</label>
      <textarea
        v-model="callbackUrl"
        rows="4"
        class="input w-full resize-y font-mono text-sm"
        :placeholder="t('admin.accounts.grokOauth.callbackPlaceholder')"
      ></textarea>
      <p class="mt-2 text-xs text-emerald-700 dark:text-emerald-300">
        {{ t('admin.accounts.grokOauth.callbackHint') }}
      </p>

      <div v-if="parsedCode" class="mt-4 grid gap-3 md:grid-cols-2">
        <div class="rounded-md bg-slate-100 px-3 py-3 dark:bg-slate-800">
          <div class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400">
            {{ t('admin.accounts.grokOauth.parsedCode') }}
          </div>
          <div class="mt-2 break-all font-mono text-xs text-slate-700 dark:text-slate-200">
            {{ parsedCode }}
          </div>
        </div>
        <div class="rounded-md bg-slate-100 px-3 py-3 dark:bg-slate-800">
          <div class="text-xs font-semibold uppercase text-slate-500 dark:text-slate-400">
            {{ t('admin.accounts.grokOauth.parsedState') }}
          </div>
          <div class="mt-2 break-all font-mono text-xs text-slate-700 dark:text-slate-200">
            {{ resolvedState || '-' }}
          </div>
        </div>
      </div>

      <button
        type="button"
        class="btn btn-primary mt-4"
        :disabled="submitting || loading || !canSubmit"
        data-testid="grok-oauth-submit"
        @click="submitOAuth"
      >
        {{ submitting || loading ? t('common.loading') : submitLabel }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useClipboard } from '@/composables/useClipboard'
import {
  buildGrokOAuthPayload,
  parseGrokOAuthCallback,
  type GrokAuthUrlResult,
  type GrokExchangeCodeResult,
  type ParsedGrokOAuthPayload
} from '@/utils/grokOAuth'

const props = withDefaults(defineProps<{
  proxyId?: number | null
  submitLabel: string
  submitting?: boolean
}>(), {
  proxyId: null,
  submitting: false
})

const emit = defineEmits<{
  submit: [payload: ParsedGrokOAuthPayload]
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const authState = ref<GrokAuthUrlResult | null>(null)
const callbackUrl = ref('')
const parsedCode = ref('')
const parsedState = ref('')
const errorMessage = ref('')
const loading = ref(false)

const resolvedState = computed(() => parsedState.value || authState.value?.state || '')
const canSubmit = computed(() => Boolean(authState.value?.session_id && parsedCode.value && resolvedState.value))

watch(callbackUrl, (value) => {
  const parsed = parseGrokOAuthCallback(value)
  parsedCode.value = parsed.code
  parsedState.value = parsed.state || ''
  errorMessage.value = ''
})

async function generateAuthUrl() {
  loading.value = true
  errorMessage.value = ''
  try {
    authState.value = await adminAPI.accounts.generateGrokAuthUrl({
      proxy_id: props.proxyId
    })
    callbackUrl.value = ''
    parsedCode.value = ''
    parsedState.value = ''
  } catch (error: any) {
    errorMessage.value = error?.message || t('admin.accounts.grokOauth.generateFailed')
  } finally {
    loading.value = false
  }
}

function openAuthUrl() {
  if (authState.value?.auth_url) {
    window.open(authState.value.auth_url, '_blank', 'noopener,noreferrer')
  }
}

function copyAuthUrl() {
  if (authState.value?.auth_url) {
    void copyToClipboard(authState.value.auth_url)
  }
}

async function submitOAuth() {
  if (!authState.value?.session_id) {
    errorMessage.value = t('admin.accounts.grokOauth.sessionMissing')
    return
  }
  if (!parsedCode.value) {
    errorMessage.value = t('admin.accounts.grokOauth.codeMissing')
    return
  }

  loading.value = true
  errorMessage.value = ''
  try {
    const tokenInfo = await adminAPI.accounts.exchangeGrokAuthCode({
      session_id: authState.value.session_id,
      code: parsedCode.value,
      state: resolvedState.value,
      proxy_id: props.proxyId
    })
    emit('submit', buildGrokOAuthPayload(tokenInfo as GrokExchangeCodeResult))
  } catch (error: any) {
    errorMessage.value = error?.message || t('admin.accounts.grokOauth.exchangeFailed')
  } finally {
    loading.value = false
  }
}

function reset() {
  authState.value = null
  callbackUrl.value = ''
  parsedCode.value = ''
  parsedState.value = ''
  errorMessage.value = ''
  loading.value = false
}

defineExpose({ reset })
</script>
