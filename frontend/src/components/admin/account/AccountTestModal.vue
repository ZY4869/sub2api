<template>
  <AccountTestModalContent :ctx="accountTestContext" />
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import type {
  BlacklistAdvicePayload,
  BlacklistFeedbackPayload
} from '@/api/admin/accounts'
import AccountTestModalContent from './AccountTestModalContent.vue'
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
const aiResponseHeaderPrinted = ref(false)
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
const supportsPromptInput = computed(() => {
  if (isGrokAccount.value || isBaiduDocumentAIAccount.value) {
    return false
  }
  return Boolean(effectiveSelectedModelId.value.trim())
})
const promptInputLabel = computed(() =>
  supportsImageTest.value
    ? t('admin.accounts.imageTestPromptLabel')
    : t('admin.accounts.textTestPromptLabel')
)
const promptInputPlaceholder = computed(() =>
  supportsImageTest.value
    ? t('admin.accounts.imageTestPromptPlaceholder')
    : t('admin.accounts.textTestPromptPlaceholder')
)
const promptInputHint = computed(() =>
  supportsImageTest.value
    ? t('admin.accounts.imageTestHint')
    : t('admin.accounts.textTestHint')
)
const defaultTestPrompt = computed(() =>
  supportsImageTest.value
    ? t('admin.accounts.imageTestPromptDefault')
    : t('admin.accounts.textTestPromptDefault')
)

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
  if (blacklistAdvice.value.reason_code === 'credentials_need_reauth') {
    return t('admin.accounts.testBlacklist.needsReauth')
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
  if (supportsPromptInput.value && !testPrompt.value.trim()) {
    testPrompt.value = defaultTestPrompt.value
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
  aiResponseHeaderPrinted.value = false
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

const ensureAiResponseHeader = () => {
  if (aiResponseHeaderPrinted.value) {
    return
  }
  aiResponseHeaderPrinted.value = true
  addLine(t('admin.accounts.aiResponseHeader'), 'text-green-300')
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
  const prompt = supportsPromptInput.value ? testPrompt.value.trim() : ''
  if (modelInputMode.value === 'manual') {
    return {
      model_input_mode: 'manual' as const,
      manual_model_id: manualModelId.value.trim(),
      request_alias: manualRequestAlias.value.trim() || undefined,
      test_mode: effectiveTestMode.value,
      source_protocol: effectiveSelectedSourceProtocol.value || undefined,
      prompt
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
    prompt
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
        ensureAiResponseHeader()
        if (isGrokAccount.value || effectiveTestMode.value === 'health_check') {
          addLine(event.text, 'text-green-400')
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

const accountTestContext = {
  t,
  show: computed(() => props.show),
  account: computed(() => props.account),
  supportsTestModes,
  testModeOptions,
  selectedTestMode,
  status,
  selectTestMode,
  loadingModels,
  loadAvailableModels,
  modelInputMode,
  selectedModelKey,
  manualModelId,
  manualSourceProtocol,
  availableModels,
  isProtocolGatewayAccount,
  manualRequestAlias,
  isKiroAccount,
  isGrokAccount,
  grokTestHintKey,
  supportsPromptInput,
  testPrompt,
  promptInputLabel,
  promptInputPlaceholder,
  promptInputHint,
  terminalRef,
  outputLines,
  streamingContent,
  errorMessage,
  copyOutput,
  generatedImages,
  runtimeContextItems,
  visibleBlacklistAdvice,
  blacklistAdviceClasses,
  blacklistAdviceTitle,
  blacklistAdviceMessage,
  blacklistAdviceBadgeClasses,
  blacklistAdviceBadge,
  supportsImageTest,
  handleClose,
  handleBlacklist,
  blacklistButtonDisabled,
  blacklistAdvice,
  blacklistButtonLabel,
  startTest,
  effectiveSelectedModelId
}

onBeforeUnmount(() => {
  abortActiveRequest()
})
</script>
