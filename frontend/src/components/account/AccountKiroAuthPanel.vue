<template>
  <div class="space-y-4 rounded-xl border border-amber-200 bg-amber-50/80 p-5 dark:border-amber-900/40 dark:bg-amber-950/20">
    <div class="space-y-1">
      <h3 class="text-base font-semibold text-amber-950 dark:text-amber-100">
        {{ t('admin.accounts.kiroAuth.title') }}
      </h3>
      <p class="text-sm text-amber-800 dark:text-amber-300">
        {{ t('admin.accounts.kiroAuth.description') }}
      </p>
    </div>

    <div class="inline-flex rounded-lg bg-white/70 p-1 dark:bg-slate-900/60">
      <button
        type="button"
        :class="tabClass('oauth')"
        @click="activeTab = 'oauth'"
      >
        {{ t('admin.accounts.kiroAuth.tabs.oauth') }}
      </button>
      <button
        type="button"
        :class="tabClass('import')"
        @click="activeTab = 'import'"
      >
        {{ t('admin.accounts.kiroAuth.tabs.import') }}
      </button>
    </div>

    <AccountKiroTokenImportPanel
      v-if="activeTab === 'import'"
      ref="importPanelRef"
      :submit-label="submitLabel"
      :submitting="submitting"
      :initial-extra="initialExtra"
      @submit="emit('submit', $event)"
    />

    <template v-else>
      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <button
          v-for="method in methods"
          :key="method.value"
          type="button"
          :class="methodClass(method.value)"
          @click="selectMethod(method.value)"
        >
          <span class="block text-sm font-semibold">{{ method.title }}</span>
          <span class="mt-1 block text-left text-xs opacity-80">{{ method.description }}</span>
        </button>
      </div>

      <div v-if="selectedMethod === 'idc'" class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.accounts.kiroAuth.startUrl') }}</label>
          <input
            v-model="startUrl"
            type="text"
            class="input"
            :placeholder="t('admin.accounts.kiroAuth.startUrlPlaceholder')"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.kiroAuth.region') }}</label>
          <input
            v-model="region"
            type="text"
            class="input"
            :placeholder="t('admin.accounts.kiroAuth.regionPlaceholder')"
          />
        </div>
      </div>

      <div
        v-if="isSocialMethod"
        class="rounded-lg border border-amber-300 bg-amber-100/70 p-3 text-sm text-amber-900 dark:border-amber-700/60 dark:bg-amber-950/30 dark:text-amber-200"
      >
        {{ t('admin.accounts.kiroAuth.socialWarning') }}
      </div>

      <AccountKiroMembershipFields
        v-model:member-level="memberLevel"
        v-model:member-credits="memberCredits"
      />

      <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300">
        {{ errorMessage }}
      </div>

      <div class="rounded-lg border border-amber-200 bg-white/80 p-4 dark:border-amber-900/40 dark:bg-slate-900/60">
        <div class="flex flex-wrap gap-3">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="generateAuthUrl">
            {{ authState ? t('admin.accounts.kiroAuth.regenerate') : t('admin.accounts.kiroAuth.generate') }}
          </button>
          <button type="button" class="btn btn-secondary" :disabled="!authState?.auth_url" @click="openAuthUrl">
            {{ t('admin.accounts.kiroAuth.openAuthUrl') }}
          </button>
          <button type="button" class="btn btn-secondary" :disabled="!authState?.auth_url" @click="copyAuthUrl">
            {{ t('admin.accounts.kiroAuth.copyAuthUrl') }}
          </button>
        </div>

        <div v-if="authState" class="mt-4 space-y-3">
          <div>
            <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
              {{ t('admin.accounts.kiroAuth.authUrl') }}
            </div>
            <div class="break-all rounded-md bg-slate-100 px-3 py-3 text-sm text-slate-700 dark:bg-slate-800 dark:text-slate-200">
              {{ authState.auth_url }}
            </div>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div>
              <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
                {{ t('admin.accounts.kiroAuth.redirectUri') }}
              </div>
              <div class="break-all rounded-md bg-slate-100 px-3 py-3 text-sm text-slate-700 dark:bg-slate-800 dark:text-slate-200">
                {{ authState.redirect_uri }}
              </div>
            </div>
            <div>
              <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
                {{ t('admin.accounts.kiroAuth.oauthState') }}
              </div>
              <div class="break-all rounded-md bg-slate-100 px-3 py-3 font-mono text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-200">
                {{ authState.state }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="rounded-lg border border-amber-200 bg-white/80 p-4 dark:border-amber-900/40 dark:bg-slate-900/60">
        <label class="input-label">{{ t('admin.accounts.kiroAuth.callbackUrl') }}</label>
        <textarea
          v-model="callbackUrl"
          rows="4"
          class="input w-full resize-y font-mono text-sm"
          :placeholder="t('admin.accounts.kiroAuth.callbackPlaceholder')"
        ></textarea>
        <p class="mt-2 text-xs text-amber-700 dark:text-amber-300">
          {{ t('admin.accounts.kiroAuth.callbackHint') }}
        </p>

        <div v-if="parsedCode" class="mt-4 grid gap-3 md:grid-cols-2">
          <div class="rounded-md bg-slate-100 px-3 py-3 dark:bg-slate-800">
            <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
              {{ t('admin.accounts.kiroAuth.parsedCode') }}
            </div>
            <div class="mt-2 break-all font-mono text-xs text-slate-700 dark:text-slate-200">
              {{ parsedCode }}
            </div>
          </div>
          <div class="rounded-md bg-slate-100 px-3 py-3 dark:bg-slate-800">
            <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
              {{ t('admin.accounts.kiroAuth.parsedState') }}
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
          @click="submitOAuth"
        >
          {{ submitting || loading ? t('common.loading') : submitLabel }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import AccountKiroMembershipFields from '@/components/account/AccountKiroMembershipFields.vue'
import AccountKiroTokenImportPanel from '@/components/account/AccountKiroTokenImportPanel.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { ParsedKiroTokenImport } from '@/utils/kiroTokenImport'
import {
  buildKiroOAuthPayload,
  parseKiroOAuthCallback,
  type KiroAuthUrlResult,
  type KiroExchangeCodeResult,
  type KiroOAuthMethod
} from '@/utils/kiroOAuth'
import type { KiroMemberLevel } from '@/utils/kiroMembership'
import {
  buildKiroMembershipExtra,
  parseKiroMemberCredits,
  readKiroMembershipFromExtra
} from '@/utils/kiroMembership'

interface Props {
  proxyId?: number | null
  submitLabel: string
  submitting?: boolean
  initialExtra?: Record<string, unknown> | null
}

const props = withDefaults(defineProps<Props>(), {
  proxyId: null,
  submitting: false,
  initialExtra: null
})

const emit = defineEmits<{
  submit: [payload: ParsedKiroTokenImport]
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const importPanelRef = ref<{ reset: () => void } | null>(null)
const activeTab = ref<'oauth' | 'import'>('oauth')
const selectedMethod = ref<KiroOAuthMethod>('builder_id')
const startUrl = ref('https://view.awsapps.com/start')
const region = ref('us-east-1')
const authState = ref<KiroAuthUrlResult | null>(null)
const callbackUrl = ref('')
const parsedCode = ref('')
const parsedState = ref('')
const errorMessage = ref('')
const loading = ref(false)
const memberLevel = ref<KiroMemberLevel>('kiro_free')
const memberCredits = ref('50')

const methods = computed(() => [
  {
    value: 'builder_id' as const,
    title: t('admin.accounts.kiroAuth.methods.builderId.title'),
    description: t('admin.accounts.kiroAuth.methods.builderId.description')
  },
  {
    value: 'idc' as const,
    title: t('admin.accounts.kiroAuth.methods.idc.title'),
    description: t('admin.accounts.kiroAuth.methods.idc.description')
  },
  {
    value: 'github' as const,
    title: t('admin.accounts.kiroAuth.methods.github.title'),
    description: t('admin.accounts.kiroAuth.methods.github.description')
  },
  {
    value: 'google' as const,
    title: t('admin.accounts.kiroAuth.methods.google.title'),
    description: t('admin.accounts.kiroAuth.methods.google.description')
  }
])

const isSocialMethod = computed(() => selectedMethod.value === 'github' || selectedMethod.value === 'google')
const resolvedState = computed(() => parsedState.value || authState.value?.state || '')
const canSubmit = computed(() => Boolean(authState.value?.session_id && parsedCode.value && resolvedState.value))

watch(callbackUrl, (value) => {
  const parsed = parseKiroOAuthCallback(value)
  parsedCode.value = parsed.code
  parsedState.value = parsed.state || ''
  errorMessage.value = ''
})

watch(selectedMethod, () => {
  resetOAuthState()
})

watch(activeTab, (tab) => {
  errorMessage.value = ''
  if (tab === 'oauth') {
    importPanelRef.value?.reset()
  } else {
    resetOAuthState()
  }
})

watch(
  () => props.initialExtra,
  () => {
    resetMembership()
  },
  { immediate: true }
)

function tabClass(tab: 'oauth' | 'import') {
  return tab === activeTab.value
    ? 'rounded-md bg-amber-500 px-3 py-2 text-sm font-semibold text-white'
    : 'rounded-md px-3 py-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'
}

function methodClass(method: KiroOAuthMethod) {
  return method === selectedMethod.value
    ? 'rounded-lg border border-amber-500 bg-amber-500 px-4 py-3 text-left text-white shadow-sm'
    : 'rounded-lg border border-amber-200 bg-white/70 px-4 py-3 text-left text-amber-950 hover:border-amber-400 dark:border-amber-900/40 dark:bg-slate-900/50 dark:text-amber-100'
}

function selectMethod(method: KiroOAuthMethod) {
  selectedMethod.value = method
}

async function generateAuthUrl() {
  loading.value = true
  errorMessage.value = ''
  try {
    authState.value = await adminAPI.accounts.generateKiroAuthUrl({
      proxy_id: props.proxyId,
      method: selectedMethod.value,
      start_url: selectedMethod.value === 'idc' ? startUrl.value.trim() || undefined : undefined,
      region: selectedMethod.value === 'idc' || selectedMethod.value === 'builder_id' ? region.value.trim() || undefined : undefined
    })
    callbackUrl.value = ''
    parsedCode.value = ''
    parsedState.value = ''
  } catch (error: any) {
    errorMessage.value = error?.message || t('admin.accounts.kiroAuth.generateFailed')
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
    errorMessage.value = t('admin.accounts.kiroAuth.sessionMissing')
    return
  }
  if (!parsedCode.value) {
    errorMessage.value = t('admin.accounts.kiroAuth.codeMissing')
    return
  }

  loading.value = true
  errorMessage.value = ''
  try {
    const credits = parseKiroMemberCredits(memberCredits.value)
    if (credits === null) {
      errorMessage.value = t('admin.accounts.kiroMembership.invalidCredits')
      return
    }
    const tokenInfo = await adminAPI.accounts.exchangeKiroAuthCode({
      session_id: authState.value.session_id,
      code: parsedCode.value,
      state: resolvedState.value,
      proxy_id: props.proxyId
    })
    const payload = buildKiroOAuthPayload(tokenInfo as KiroExchangeCodeResult)
    payload.extra = buildKiroMembershipExtra(memberLevel.value, credits, payload.extra)
    emit('submit', payload)
  } catch (error: any) {
    errorMessage.value = error?.message || t('admin.accounts.kiroAuth.exchangeFailed')
  } finally {
    loading.value = false
  }
}

function resetOAuthState() {
  authState.value = null
  callbackUrl.value = ''
  parsedCode.value = ''
  parsedState.value = ''
  errorMessage.value = ''
  loading.value = false
}

function resetMembership() {
  const membership = readKiroMembershipFromExtra(props.initialExtra)
  memberLevel.value = membership.level
  memberCredits.value = String(membership.credits)
}

function reset() {
  activeTab.value = 'oauth'
  selectedMethod.value = 'builder_id'
  startUrl.value = 'https://view.awsapps.com/start'
  region.value = 'us-east-1'
  importPanelRef.value?.reset()
  resetOAuthState()
  resetMembership()
}

defineExpose({ reset })
</script>
