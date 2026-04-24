<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.testAccountConnection')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Account Info Card -->
      <div
        v-if="account"
        class="flex items-center justify-between rounded-xl border border-gray-200 bg-gradient-to-r from-gray-50 to-gray-100 p-3 dark:border-dark-500 dark:from-dark-700 dark:to-dark-600"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-primary-600"
          >
            <Icon name="play" size="md" class="text-white" :stroke-width="2" />
          </div>
          <div>
            <div class="font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
            <div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
              <span
                class="rounded bg-gray-200 px-1.5 py-0.5 text-[10px] font-medium uppercase dark:bg-dark-500"
              >
                {{ account.type }}
              </span>
              <span>{{ t('admin.accounts.account') }}</span>
            </div>
          </div>
        </div>
        <span
          :class="[
            'rounded-full px-2.5 py-1 text-xs font-semibold',
            account.status === 'active'
              ? 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-400'
              : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
          ]"
        >
          {{ account.status }}
        </span>
      </div>

      <div v-if="supportsTestModes" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.testModeLabel') }}
        </label>
        <div class="grid gap-2 sm:grid-cols-2">
          <button
            v-for="option in testModeOptions"
            :key="option.value"
            :data-test="`test-mode-${option.value}`"
            type="button"
            :disabled="status === 'connecting'"
            :class="[
              'rounded-xl border px-4 py-3 text-left transition-all',
              selectedTestMode === option.value
                ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
                : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200 dark:hover:border-primary-500/60',
              status === 'connecting' ? 'cursor-not-allowed opacity-70' : ''
            ]"
            @click="selectTestMode(option.value)"
          >
            <div class="text-sm font-semibold">
              {{ option.label }}
            </div>
            <p class="mt-1 text-xs leading-5 opacity-80">
              {{ option.description }}
            </p>
          </button>
        </div>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.selectTestModel') }}
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="loadingModels || status === 'connecting' || !account"
            @click="loadAvailableModels(true)"
          >
            <Icon
              v-if="loadingModels"
              name="refresh"
              size="sm"
              class="animate-spin"
              :stroke-width="2"
            />
            <Icon v-else name="refresh" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.refreshTestModels') }}</span>
          </button>
        </div>

        <AccountTestModelSelectionFields
          v-model:model-input-mode="modelInputMode"
          v-model:selected-model-key="selectedModelKey"
          v-model:manual-model-id="manualModelId"
          v-model:manual-source-protocol="manualSourceProtocol"
          :available-models="availableModels"
          :loading-models="loadingModels"
          :disabled="status === 'connecting'"
          :show-manual-source-protocol-field="isProtocolGatewayAccount"
        />

        <label v-if="modelInputMode === 'manual'" class="space-y-1.5">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.probeFinalize.manualRequestAlias') }}
          </span>
          <input
            v-model="manualRequestAlias"
            type="text"
            class="input"
            :disabled="status === 'connecting'"
            :placeholder="manualModelId || t('admin.accounts.probeFinalize.manualRequestAliasPlaceholder')"
          />
        </label>
      </div>
      <div
        v-if="isKiroAccount"
        class="rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-700 dark:border-sky-700 dark:bg-sky-900/20 dark:text-sky-300"
      >
        {{ t('admin.accounts.kiroTestModelSourceHint') }}
      </div>
      <div
        v-else-if="isGrokAccount"
        class="rounded-lg border border-violet-200 bg-violet-50 px-3 py-2 text-xs text-violet-700 dark:border-violet-700 dark:bg-violet-900/20 dark:text-violet-300"
      >
        {{ t(grokTestHintKey) }}
      </div>

      <div v-if="supportsImageTest" class="space-y-1.5">
        <TextArea
          v-model="testPrompt"
          :label="t('admin.accounts.imageTestPromptLabel')"
          :placeholder="t('admin.accounts.imageTestPromptPlaceholder')"
          :hint="t('admin.accounts.imageTestHint')"
          :disabled="status === 'connecting'"
          rows="3"
        />
      </div>

      <!-- Terminal Output -->
      <div class="group relative">
        <div
          ref="terminalRef"
          class="max-h-[240px] min-h-[120px] overflow-y-auto rounded-xl border border-gray-700 bg-gray-900 p-4 font-mono text-sm dark:border-gray-800 dark:bg-black"
        >
          <!-- Status Line -->
          <div v-if="status === 'idle'" class="flex items-center gap-2 text-gray-500">
            <Icon name="play" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.readyToTest') }}</span>
          </div>
          <div v-else-if="status === 'connecting'" class="flex items-center gap-2 text-yellow-400">
            <Icon name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <span>{{ t('admin.accounts.connectingToApi') }}</span>
          </div>

          <!-- Output Lines -->
          <div v-for="(line, index) in outputLines" :key="index" :class="line.class">
            {{ line.text }}
          </div>

          <!-- Streaming Content -->
          <div v-if="streamingContent" class="text-green-400">
            {{ streamingContent }}<span class="animate-pulse">_</span>
          </div>

          <!-- Result Status -->
          <div
            v-if="status === 'success'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3 text-green-400"
          >
            <Icon name="check" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.testCompleted') }}</span>
          </div>
          <div
            v-else-if="status === 'error'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3 text-red-400"
          >
            <Icon name="x" size="sm" :stroke-width="2" />
            <span>{{ errorMessage }}</span>
          </div>
        </div>

        <!-- Copy Button -->
        <button
          v-if="outputLines.length > 0"
          @click="copyOutput"
          class="absolute right-2 top-2 rounded-lg bg-gray-800/80 p-1.5 text-gray-400 opacity-0 transition-all hover:bg-gray-700 hover:text-white group-hover:opacity-100"
          :title="t('admin.accounts.copyOutput')"
        >
          <Icon name="link" size="sm" :stroke-width="2" />
        </button>
      </div>

      <div v-if="generatedImages.length > 0" class="space-y-2">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.imageTestPreview') }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <a
            v-for="(image, index) in generatedImages"
            :key="`${image.url}-${index}`"
            :href="image.url"
            target="_blank"
            rel="noopener noreferrer"
            class="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:border-primary-300 hover:shadow-md dark:border-dark-500 dark:bg-dark-700"
          >
            <img :src="image.url" :alt="`account-test-image-${index + 1}`" class="h-48 w-full object-cover" />
            <div class="border-t border-gray-100 px-3 py-2 text-xs text-gray-500 dark:border-dark-500 dark:text-gray-300">
              {{ image.mimeType || 'image/*' }}
            </div>
          </a>
        </div>
      </div>

      <div
        v-if="runtimeContextItems.length > 0"
        class="rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-xs text-sky-900 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-100"
      >
        <div class="text-sm font-semibold">
          {{ t('admin.accounts.testRuntimeContextTitle') }}
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <span
            v-for="item in runtimeContextItems"
            :key="item.key"
            class="inline-flex items-center rounded-full bg-white/80 px-2.5 py-1 font-medium text-sky-700 dark:bg-white/10 dark:text-sky-200"
          >
            {{ item.label }}
          </span>
        </div>
      </div>

      <div
        v-if="visibleBlacklistAdvice"
        class="rounded-xl border px-4 py-3"
        :class="blacklistAdviceClasses"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="text-sm font-semibold">
              {{ blacklistAdviceTitle }}
            </div>
            <p class="mt-1 whitespace-pre-wrap text-xs leading-5 opacity-90">
              {{ blacklistAdviceMessage }}
            </p>
          </div>
          <span
            class="shrink-0 rounded-full px-2.5 py-1 text-[11px] font-semibold"
            :class="blacklistAdviceBadgeClasses"
          >
            {{ blacklistAdviceBadge }}
          </span>
        </div>
      </div>

      <!-- Test Info -->
      <div class="flex items-center justify-between px-1 text-xs text-gray-500 dark:text-gray-400">
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1">
            <Icon name="grid" size="sm" :stroke-width="2" />
            {{ t('admin.accounts.testModel') }}
          </span>
        </div>
        <span class="flex items-center gap-1">
          <Icon name="chat" size="sm" :stroke-width="2" />
          {{
            supportsImageTest
              ? t('admin.accounts.imageTestMode')
              : t('admin.accounts.testPrompt')
          }}
        </span>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          @click="handleClose"
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        >
          {{ t('common.close') }}
        </button>
        <button
          @click="handleBlacklist"
          :disabled="blacklistButtonDisabled"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            blacklistButtonDisabled
              ? 'cursor-not-allowed bg-rose-200 text-rose-500 dark:bg-rose-950/40 dark:text-rose-300/60'
              : blacklistAdvice?.decision === 'not_recommended'
                ? 'bg-amber-500 text-white hover:bg-amber-600'
                : 'bg-rose-500 text-white hover:bg-rose-600'
          ]"
        >
          <Icon name="ban" size="sm" :stroke-width="2" />
          <span>{{ blacklistButtonLabel }}</span>
        </button>
        <button
          @click="startTest"
          :disabled="status === 'connecting' || !effectiveSelectedModelId"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            status === 'connecting' || !effectiveSelectedModelId
              ? 'cursor-not-allowed bg-primary-400 text-white'
              : status === 'success'
                ? 'bg-green-500 text-white hover:bg-green-600'
                : status === 'error'
                  ? 'bg-orange-500 text-white hover:bg-orange-600'
                  : 'bg-primary-500 text-white hover:bg-primary-600'
          ]"
        >
          <Icon
            v-if="status === 'connecting'"
            name="refresh"
            size="sm"
            class="animate-spin"
            :stroke-width="2"
          />
          <Icon v-else-if="status === 'idle'" name="play" size="sm" :stroke-width="2" />
          <Icon v-else name="refresh" size="sm" :stroke-width="2" />
          <span>
            {{
              status === 'connecting'
                ? t('admin.accounts.testing')
                : status === 'idle'
                  ? t('admin.accounts.startTest')
                  : t('admin.accounts.retry')
            }}
          </span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TextArea from '@/components/common/TextArea.vue'
import AccountTestModelSelectionFields from './AccountTestModelSelectionFields.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import type {
  BlacklistAdvicePayload,
  BlacklistFeedbackPayload
} from '@/api/admin/accounts'
import type { Account, AdminAccountModelOption } from '@/types'
import {
  DEFAULT_ACCOUNT_TEST_MODE,
  type AccountTestMode,
  loadAccountTestModePreference,
  normalizeAccountTestMode,
  saveAccountTestModePreference
} from '@/utils/accountTestMode'
import {
  normalizeGatewayAcceptedProtocol,
  resolveEffectiveAccountPlatformFromAccount,
  resolveGatewayProtocolLabel
} from '@/utils/accountProtocolGateway'
import {
  findAccountTestModelByKey
} from '@/utils/accountTestModelOptions'
import {
  resolveCatalogTargetFromModel,
  resolveGatewayTestSelectedModelKey
} from '@/utils/accountGatewayTestDefaults'
import { isBaiduDocumentAIPlatform } from '@/utils/baiduDocumentAI'

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

interface OutputLine {
  text: string
  class: string
}

interface PreviewImage {
  url: string
  mimeType?: string
}

const props = defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (
    e: 'blacklist',
    payload: { account: Account; source: 'test_modal'; feedback?: BlacklistFeedbackPayload }
  ): void
}>()

const terminalRef = ref<HTMLElement | null>(null)
const status = ref<'idle' | 'connecting' | 'success' | 'error'>('idle')
const outputLines = ref<OutputLine[]>([])
const streamingContent = ref('')
const errorMessage = ref('')
const availableModels = ref<AdminAccountModelOption[]>([])
const selectedModelKey = ref('')
const modelInputMode = ref<'catalog' | 'manual'>('catalog')
const manualModelId = ref('')
const manualRequestAlias = ref('')
const manualSourceProtocol = ref<'openai' | 'anthropic' | 'gemini' | ''>('')
const testPrompt = ref('')
const loadingModels = ref(false)
let activeAbortController: AbortController | null = null
const blacklistAdvice = ref<BlacklistAdvicePayload | null>(null)
const selectedTestMode = ref<AccountTestMode>(DEFAULT_ACCOUNT_TEST_MODE)
const runtimePlatform = computed(() =>
  props.account ? resolveEffectiveAccountPlatformFromAccount(props.account) : null
)
const selectedModelOption = computed(() =>
  findAccountTestModelByKey(availableModels.value, selectedModelKey.value)
)
const selectedModelId = computed(() => selectedModelOption.value?.id || '')
const selectedSourceProtocol = computed(() =>
  normalizeGatewayAcceptedProtocol(selectedModelOption.value?.source_protocol)
)
const effectiveSelectedModelId = computed(() =>
  modelInputMode.value === 'manual' ? manualModelId.value.trim() : selectedModelId.value
)
const effectiveSelectedSourceProtocol = computed(() =>
  modelInputMode.value === 'manual'
    ? normalizeGatewayAcceptedProtocol(manualSourceProtocol.value)
    : selectedSourceProtocol.value
)
const isGrokAccount = computed(() => props.account?.platform === 'grok')
const isBaiduDocumentAIAccount = computed(() => isBaiduDocumentAIPlatform(props.account?.platform))
const isKiroAccount = computed(() => props.account?.platform === 'kiro')
const isProtocolGatewayAccount = computed(() => props.account?.platform === 'protocol_gateway')
const supportsTestModes = computed(() => !isGrokAccount.value)
const generatedImages = ref<PreviewImage[]>([])
const runtimeContext = ref<{
  testMode: '' | AccountTestMode
  platform: string
  sourceProtocol: '' | 'openai' | 'anthropic' | 'gemini'
  simulatedClient: string
}>({
  testMode: '',
  platform: '',
  sourceProtocol: '',
  simulatedClient: ''
})
const availableTestModeValues = computed<AccountTestMode[]>(() => {
  if (isGrokAccount.value) {
    return []
  }
  if (isBaiduDocumentAIAccount.value) {
    return ['health_check']
  }
  return ['real_forward', 'health_check']
})
const effectiveTestMode = computed<AccountTestMode>(() =>
  isBaiduDocumentAIAccount.value ? 'health_check' : normalizeAccountTestMode(selectedTestMode.value)
)
const testModeOptions = computed(() =>
  availableTestModeValues.value.map((mode) => ({
    value: mode,
    label: t(mode === 'health_check'
      ? 'admin.accounts.testModes.healthCheck'
      : 'admin.accounts.testModes.realForward'),
    description: t(mode === 'health_check'
      ? 'admin.accounts.testModes.healthCheckHint'
      : 'admin.accounts.testModes.realForwardHint')
  }))
)
const effectiveTestPlatform = computed(() =>
  effectiveSelectedSourceProtocol.value || runtimePlatform.value
)
const supportsImageTest = computed(() => {
  if (isGrokAccount.value || isBaiduDocumentAIAccount.value) {
    return false
  }

  const optionMode = String(selectedModelOption.value?.mode || '').trim()
  const modelID = effectiveSelectedModelId.value.trim().toLowerCase()
  const inferredByID =
    modelID === 'chatgpt-image-latest' ||
    modelID.startsWith('gpt-image-') ||
    (modelID.startsWith('gemini-') && modelID.includes('-image'))
  const isImageModel = modelInputMode.value === 'manual'
    ? inferredByID
    : optionMode === 'image' || (optionMode === '' && inferredByID)
  if (!isImageModel) {
    return false
  }

  return (
    effectiveTestPlatform.value === 'openai' ||
    effectiveTestPlatform.value === 'gemini' ||
    (props.account?.platform === 'antigravity' && props.account?.type === 'apikey')
  )
})

watch(
  [modelInputMode, manualModelId, isProtocolGatewayAccount, manualSourceProtocol],
  ([inputMode, modelID, isGateway, selectedProtocol]) => {
    if (inputMode !== 'manual' || !isGateway || selectedProtocol) {
      return
    }
    const normalizedModelID = String(modelID || '').trim().toLowerCase()
    if (normalizedModelID === 'chatgpt-image-latest' || normalizedModelID.startsWith('gpt-image-')) {
      manualSourceProtocol.value = 'openai'
    }
  }
)
const runtimeContextItems = computed(() => {
  const items: Array<{ key: string; label: string }> = []
  const testModeLabel = runtimeTestModeLabel(runtimeContext.value.testMode)
  if (testModeLabel) {
    items.push({
      key: 'test_mode',
      label: t('admin.accounts.testRuntimeContextMode', {
        mode: testModeLabel
      })
    })
  }
  const platformLabel = runtimePlatformLabel(runtimeContext.value.platform)
  if (platformLabel) {
    items.push({
      key: 'resolved_platform',
      label: t('admin.accounts.testRuntimeContextPlatform', {
        platform: platformLabel
      })
    })
  }
  if (runtimeContext.value.sourceProtocol) {
    items.push({
      key: 'source_protocol',
      label: t('admin.accounts.testRuntimeContextProtocol', {
        protocol: protocolSourceLabel(runtimeContext.value.sourceProtocol)
      })
    })
  }
  const simulatedClientLabel = runtimeSimulatedClientLabel(runtimeContext.value.simulatedClient)
  if (simulatedClientLabel) {
    items.push({
      key: 'simulated_client',
      label: t('admin.accounts.testRuntimeContextClient', {
        client: simulatedClientLabel
      })
    })
  }
  return items
})
const grokTestHintKey = computed(() =>
  props.account?.type === 'sso'
    ? 'admin.accounts.grokTestSsoHint'
    : 'admin.accounts.grokTestApiKeyHint'
)
const visibleBlacklistAdvice = computed(() =>
  blacklistAdvice.value?.decision === 'auto_blacklisted' ? null : blacklistAdvice.value
)

const blacklistAdviceTitle = computed(() => {
  switch (blacklistAdvice.value?.decision) {
    case 'auto_blacklisted':
      return t('admin.accounts.testBlacklist.autoTitle')
    case 'recommend_blacklist':
      return t('admin.accounts.testBlacklist.recommendTitle')
    case 'not_recommended':
      return t('admin.accounts.testBlacklist.notRecommendedTitle')
    default:
      return ''
  }
})

const blacklistAdviceMessage = computed(() => {
  if (!blacklistAdvice.value) {
    return ''
  }
  const reason = blacklistAdvice.value.reason_message || t('admin.accounts.testBlacklist.noReason')
  if (blacklistAdvice.value.decision === 'not_recommended') {
    return `${reason}\n${t('admin.accounts.testBlacklist.manualOverrideHint')}`
  }
  return reason
})

const blacklistAdviceBadge = computed(() => {
  switch (blacklistAdvice.value?.decision) {
    case 'auto_blacklisted':
      return t('admin.accounts.testBlacklist.autoBadge')
    case 'recommend_blacklist':
      return t('admin.accounts.testBlacklist.recommendBadge')
    case 'not_recommended':
      return t('admin.accounts.testBlacklist.notRecommendedBadge')
    default:
      return ''
  }
})

const blacklistAdviceClasses = computed(() => {
  switch (blacklistAdvice.value?.decision) {
    case 'auto_blacklisted':
      return 'border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-100'
    case 'recommend_blacklist':
      return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-100'
    default:
      return 'border-sky-200 bg-sky-50 text-sky-900 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-100'
  }
})

const blacklistAdviceBadgeClasses = computed(() => {
  switch (blacklistAdvice.value?.decision) {
    case 'auto_blacklisted':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-200'
    case 'recommend_blacklist':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-200'
    default:
      return 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-200'
  }
})

const blacklistButtonDisabled = computed(() =>
  !props.account ||
  status.value === 'connecting' ||
  blacklistAdvice.value?.decision === 'auto_blacklisted'
)

const blacklistButtonLabel = computed(() =>
  blacklistAdvice.value?.decision === 'auto_blacklisted'
    ? t('admin.accounts.testBlacklist.buttonDone')
    : t('admin.accounts.testBlacklist.button')
)

function protocolSourceLabel(sourceProtocol?: unknown) {
  return resolveGatewayProtocolLabel(sourceProtocol) || String(sourceProtocol || '').trim()
}

function runtimeSimulatedClientLabel(simulatedClient?: string | null) {
  switch (String(simulatedClient || '').trim()) {
    case 'codex':
      return t('admin.accounts.protocolGateway.clientProfileCodex')
    case 'gemini_cli':
      return t('admin.accounts.protocolGateway.clientProfileGeminiCli')
    case 'claude_client_mimic':
      return t('admin.accounts.protocolGateway.clientProfileClaudeMimic')
    default:
      return ''
  }
}

function runtimeTestModeLabel(mode?: AccountTestMode | string | null) {
  switch (normalizeAccountTestMode(mode)) {
    case 'health_check':
      return t('admin.accounts.testModes.healthCheck')
    default:
      return t('admin.accounts.testModes.realForward')
  }
}

function runtimePlatformLabel(platform?: string | null) {
  const normalized = String(platform || '').trim()
  if (!normalized) {
    return ''
  }
  const translationKey = `admin.accounts.platforms.${normalized}`
  const translated = t(translationKey)
  return translated === translationKey ? normalized : translated
}

watch(
  () => props.show,
  async (newVal) => {
    if (newVal && props.account) {
      testPrompt.value = ''
      modelInputMode.value = 'catalog'
      manualModelId.value = ''
      manualRequestAlias.value = ''
      manualSourceProtocol.value = ''
      selectedTestMode.value = supportsTestModes.value
        ? (isBaiduDocumentAIAccount.value ? 'health_check' : loadAccountTestModePreference())
        : DEFAULT_ACCOUNT_TEST_MODE
      resetState()
      await loadAvailableModels()
    } else {
      abortActiveRequest()
      resetState()
    }
  }
)

watch([effectiveSelectedModelId, effectiveTestPlatform], () => {
  if (supportsImageTest.value && !testPrompt.value.trim()) {
    testPrompt.value = t('admin.accounts.imageTestPromptDefault')
  }
})

watch([selectedModelKey, modelInputMode, manualModelId, manualSourceProtocol], () => {
  runtimeContext.value = {
    testMode: '',
    platform: '',
    sourceProtocol: '',
    simulatedClient: ''
  }
})

const loadAvailableModels = async (forceRefresh = false) => {
  if (!props.account) return

  loadingModels.value = true
  selectedModelKey.value = ''
  try {
    const models = await adminAPI.accounts.getAvailableModels(props.account.id, {
      refresh: forceRefresh
    })
    availableModels.value = models
    selectedModelKey.value = resolveGatewayTestSelectedModelKey(props.account ? [props.account] : [], models)
  } catch (error) {
    console.error('Failed to load available models:', error)
    availableModels.value = []
    selectedModelKey.value = ''
  } finally {
    loadingModels.value = false
  }
}

const resetRuntimeContext = () => {
  runtimeContext.value = {
    testMode: '',
    platform: '',
    sourceProtocol: '',
    simulatedClient: ''
  }
}

const resetState = () => {
  status.value = 'idle'
  outputLines.value = []
  streamingContent.value = ''
  errorMessage.value = ''
  generatedImages.value = []
  blacklistAdvice.value = null
  resetRuntimeContext()
}

/*
const handleClose = () => {
  // 防止在连接测试进行中关闭对话框
  if (status.value === 'connecting') {
  abortActiveRequest()
  resetState()
  emit('close')
}

*/

const handleClose = () => {
  abortActiveRequest()
  resetState()
  emit('close')
}

const abortActiveRequest = () => {
  if (activeAbortController) {
    activeAbortController.abort()
    activeAbortController = null
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') {
    return false
  }
  return String((error as { name?: unknown }).name || '').trim() === 'AbortError'
}

const addLine = (text: string, className: string = 'text-gray-300') => {
  outputLines.value.push({ text, class: className })
  scrollToBottom()
}

const scrollToBottom = async () => {
  await nextTick()
  if (terminalRef.value) {
    terminalRef.value.scrollTop = terminalRef.value.scrollHeight
  }
}

const selectTestMode = (mode: AccountTestMode) => {
  if (status.value === 'connecting') {
    return
  }
  const normalized = normalizeAccountTestMode(mode)
  selectedTestMode.value = normalized
  saveAccountTestModePreference(normalized)
}

const resolveTestRequestBody = () => {
  if (modelInputMode.value === 'manual') {
    return {
      model_input_mode: 'manual' as const,
      manual_model_id: manualModelId.value.trim(),
      request_alias: manualRequestAlias.value.trim() || undefined,
      test_mode: effectiveTestMode.value,
      source_protocol: effectiveSelectedSourceProtocol.value || undefined,
      prompt: supportsImageTest.value ? testPrompt.value.trim() : ''
    }
  }
  const catalogTarget = resolveCatalogTargetFromModel(selectedModelOption.value)
  if (isGrokAccount.value) {
    return {
      model_id: selectedModelId.value,
      model: selectedModelId.value,
      source_protocol: catalogTarget.sourceProtocol,
      target_provider: catalogTarget.targetProvider,
      target_model_id: catalogTarget.targetModelId
    }
  }
  return {
    model_id: selectedModelId.value,
    model: selectedModelId.value,
    test_mode: effectiveTestMode.value,
    source_protocol: catalogTarget.sourceProtocol || effectiveSelectedSourceProtocol.value || undefined,
    target_provider: catalogTarget.targetProvider,
    target_model_id: catalogTarget.targetModelId,
    prompt: supportsImageTest.value ? testPrompt.value.trim() : ''
  }
}

const resolveResponseErrorMessage = async (response: Response) => {
  try {
    const contentType = response.headers?.get?.('content-type') || ''
    if (contentType.includes('application/json')) {
      const payload = await response.json()
      const message = String(
        payload?.message ||
        payload?.error ||
        payload?.detail ||
        payload?.msg ||
        ''
      ).trim()
      if (message) {
        return message
      }
    } else {
      const text = (await response.text()).trim()
      if (text) {
        return text
      }
    }
  } catch (error) {
    console.error('Failed to parse test error response:', error)
  }
  return `HTTP error! status: ${response.status}`
}

const startTest = async () => {
  if (!props.account || !effectiveSelectedModelId.value) return

  resetState()
  status.value = 'connecting'
  addLine(t('admin.accounts.startingTestForAccount', { name: props.account.name }), 'text-blue-400')
  addLine(t('admin.accounts.testAccountTypeLabel', { type: props.account.type }), 'text-gray-400')
  if (supportsTestModes.value) {
    addLine(
      t('admin.accounts.testModeLine', {
        mode: runtimeTestModeLabel(effectiveTestMode.value)
      }),
      'text-purple-300'
    )
  }
  addLine('', 'text-gray-300')

  abortActiveRequest()
  const abortController = new AbortController()
  activeAbortController = abortController

  try {
    const response = isGrokAccount.value
      ? await adminAPI.accounts.testGrokAccount(props.account.id, resolveTestRequestBody(), {
          signal: abortController.signal
        })
      : await fetch(`/api/v1/admin/accounts/${props.account.id}/test`, {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${localStorage.getItem('auth_token')}`,
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(resolveTestRequestBody()),
          signal: abortController.signal
        })

    if (!response.ok) {
      throw new Error(await resolveResponseErrorMessage(response))
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error(t('admin.accounts.testNoResponseBody'))
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const jsonStr = line.slice(6).trim()
          if (jsonStr) {
            try {
              const event = JSON.parse(jsonStr)
              handleEvent(event)
            } catch (e) {
              console.error('Failed to parse SSE event:', e)
            }
          }
        }
      }
    }
  } catch (error: any) {
    if (isAbortError(error)) {
      status.value = 'idle'
      return
    }
    status.value = 'error'
    errorMessage.value = error.message || 'Unknown error'
    addLine(`Error: ${errorMessage.value}`, 'text-red-400')
  } finally {
    if (activeAbortController === abortController) {
      activeAbortController = null
    }
  }
}

const normalizeBlacklistAdvicePayload = (payload: unknown): BlacklistAdvicePayload | null => {
  if (!payload || typeof payload !== 'object') {
    return null
  }
  const data = payload as Record<string, unknown>
  const decision = String(data.decision || '').trim() as BlacklistAdvicePayload['decision']
  if (!decision) {
    return null
  }
  return {
    decision,
    reason_code: typeof data.reason_code === 'string' ? data.reason_code : undefined,
    reason_message: typeof data.reason_message === 'string' ? data.reason_message : undefined,
    already_blacklisted: Boolean(data.already_blacklisted),
    feedback_fingerprint:
      typeof data.feedback_fingerprint === 'string' ? data.feedback_fingerprint : undefined,
    collect_feedback: Boolean(data.collect_feedback),
    platform: typeof data.platform === 'string' ? data.platform : undefined,
    status_code: typeof data.status_code === 'number' ? data.status_code : undefined,
    error_code: typeof data.error_code === 'string' ? data.error_code : undefined,
    message_keywords: Array.isArray(data.message_keywords)
      ? data.message_keywords.filter((item): item is string => typeof item === 'string')
      : undefined
  }
}

const handleBlacklist = () => {
  if (!props.account || blacklistButtonDisabled.value) {
    return
  }
  const feedback =
    blacklistAdvice.value?.feedback_fingerprint
      ? {
          fingerprint: blacklistAdvice.value.feedback_fingerprint,
          advice_decision: blacklistAdvice.value.decision,
          action: 'blacklist' as const,
          platform: blacklistAdvice.value.platform,
          status_code: blacklistAdvice.value.status_code,
          error_code: blacklistAdvice.value.error_code,
          message_keywords: blacklistAdvice.value.message_keywords
        }
      : undefined
  emit('blacklist', {
    account: props.account,
    source: 'test_modal',
    feedback
  })
}

const handleEvent = (event: {
  type: string
  text?: string
  model?: string
  success?: boolean
  error?: string
  image_url?: string
  mime_type?: string
  data?: {
    kind?: string
    [key: string]: unknown
  }
}) => {
  switch (event.type) {
    case 'test_start':
      addLine(t('admin.accounts.connectedToApi'), 'text-green-400')
      if (event.model) {
        addLine(t('admin.accounts.usingModel', { model: event.model }), 'text-cyan-400')
      }
      addLine(
        supportsImageTest.value
          ? t('admin.accounts.sendingImageTestRequest')
          : t('admin.accounts.sendingTestMessage'),
        'text-gray-400'
      )
      addLine('', 'text-gray-300')
      addLine(t('admin.accounts.response'), 'text-yellow-400')
      break

    case 'content':
      if (event.data?.kind === 'runtime_meta') {
        const runtimeKey = String(event.data.key || '').trim()
        const runtimeValue = String(event.data.value || '').trim()
        if (runtimeKey === 'resolved_protocol') {
          runtimeContext.value = {
            ...runtimeContext.value,
            sourceProtocol: normalizeGatewayAcceptedProtocol(runtimeValue)
          }
        }
        if (runtimeKey === 'resolved_platform') {
          runtimeContext.value = {
            ...runtimeContext.value,
            platform: runtimeValue
          }
        }
        if (runtimeKey === 'test_mode') {
          runtimeContext.value = {
            ...runtimeContext.value,
            testMode: normalizeAccountTestMode(runtimeValue)
          }
        }
        if (runtimeKey === 'simulated_client') {
          runtimeContext.value = {
            ...runtimeContext.value,
            simulatedClient: runtimeValue
          }
        }
        if (event.text) {
          addLine(event.text, 'text-sky-300')
        }
        break
      }
      if (event.text) {
        if (isGrokAccount.value || effectiveTestMode.value === 'health_check') {
          addLine(event.text, 'text-sky-300')
        } else {
          streamingContent.value += event.text
          scrollToBottom()
        }
      }
      break

    case 'image':
      if (event.image_url) {
        generatedImages.value.push({
          url: event.image_url,
          mimeType: event.mime_type
        })
        addLine(t('admin.accounts.imageTestReceived', { count: generatedImages.value.length }), 'text-purple-300')
      }
      break

    case 'blacklist_advice': {
      const advice = normalizeBlacklistAdvicePayload(event.data)
      if (!advice) {
        break
      }
      blacklistAdvice.value = advice
      addLine(
        advice.reason_message || blacklistAdviceTitle.value,
        advice.decision === 'not_recommended' ? 'text-sky-300' : 'text-amber-300'
      )
      break
    }

    case 'test_complete':
      // Move streaming content to output lines
      if (streamingContent.value) {
        addLine(streamingContent.value, 'text-green-300')
        streamingContent.value = ''
      }
      if (event.success) {
        status.value = 'success'
      } else {
        status.value = 'error'
        errorMessage.value = event.error || 'Test failed'
      }
      break

    case 'error':
      status.value = 'error'
      errorMessage.value = event.error || 'Unknown error'
      if (streamingContent.value) {
        addLine(streamingContent.value, 'text-green-300')
        streamingContent.value = ''
      }
      break
  }
}

const copyOutput = () => {
  const text = outputLines.value.map((l) => l.text).join('\n')
  copyToClipboard(text, t('admin.accounts.outputCopied'))
}

onBeforeUnmount(() => {
  abortActiveRequest()
})
</script>
