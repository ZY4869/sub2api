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

      <div v-if="!isSoraAccount" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.selectTestModel') }}
        </label>
        <Select
          v-model="selectedModelId"
          :options="availableModelOptions"
          :disabled="loadingModels || status === 'connecting'"
          searchable
          value-key="id"
          label-key="display_name"
          :placeholder="loadingModels ? t('common.loading') + '...' : t('admin.accounts.selectTestModel')"
        >
          <template #selected="{ option }">
            <div v-if="option" class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium text-gray-900 dark:text-white">
                  {{ option.display_name || option.id }}
                </span>
                <span
                  v-if="isDeprecatedModel(option)"
                  class="inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
                >
                  {{ t('admin.models.registry.lifecycleLabels.deprecated') }}
                </span>
              </div>
              <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ option.id }}</div>
            </div>
            <span v-else>
              {{ loadingModels ? `${t('common.loading')}...` : t('admin.accounts.selectTestModel') }}
            </span>
          </template>

          <template #option="{ option, selected }">
            <div class="flex min-w-0 flex-1 items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="truncate font-medium text-gray-900 dark:text-white">
                    {{ option.display_name || option.id }}
                  </span>
                  <span
                    v-if="isDeprecatedModel(option)"
                    class="inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
                  >
                    {{ t('admin.models.registry.lifecycleLabels.deprecated') }}
                  </span>
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ option.id }}</div>
                <div
                  v-if="option.replaced_by"
                  class="truncate text-[11px] text-amber-600 dark:text-amber-300"
                >
                  {{ t('admin.models.registry.replacedByHint', { model: option.replaced_by }) }}
                </div>
              </div>
              <Icon
                v-if="selected"
                name="check"
                size="sm"
                class="mt-0.5 shrink-0 text-primary-500"
                :stroke-width="2"
              />
            </div>
          </template>
        </Select>
      </div>
      <div
        v-if="isKiroAccount"
        class="rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-700 dark:border-sky-700 dark:bg-sky-900/20 dark:text-sky-300"
      >
        {{ t('admin.accounts.kiroTestModelSourceHint') }}
      </div>
      <div
        v-else-if="isSoraAccount"
        class="rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:border-blue-700 dark:bg-blue-900/20 dark:text-blue-300"
      >
        {{ t('admin.accounts.soraTestHint') }}
      </div>

      <div v-if="supportsGeminiImageTest" class="space-y-1.5">
        <TextArea
          v-model="testPrompt"
          :label="t('admin.accounts.geminiImagePromptLabel')"
          :placeholder="t('admin.accounts.geminiImagePromptPlaceholder')"
          :hint="t('admin.accounts.geminiImageTestHint')"
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
          {{ t('admin.accounts.geminiImagePreview') }}
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
            <img :src="image.url" :alt="`gemini-test-image-${index + 1}`" class="h-48 w-full object-cover" />
            <div class="border-t border-gray-100 px-3 py-2 text-xs text-gray-500 dark:border-dark-500 dark:text-gray-300">
              {{ image.mimeType || 'image/*' }}
            </div>
          </a>
        </div>
      </div>

      <div
        v-if="blacklistAdvice"
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
            {{ isSoraAccount ? t('admin.accounts.soraTestTarget') : t('admin.accounts.testModel') }}
          </span>
        </div>
        <span class="flex items-center gap-1">
          <Icon name="chat" size="sm" :stroke-width="2" />
          {{
            isSoraAccount
              ? t('admin.accounts.soraTestMode')
              : supportsGeminiImageTest
                ? t('admin.accounts.geminiImageTestMode')
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
          :disabled="status === 'connecting'"
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
          :disabled="status === 'connecting' || (!isSoraAccount && !selectedModelId)"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            status === 'connecting' || (!isSoraAccount && !selectedModelId)
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
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import type {
  BlacklistAdvicePayload,
  BlacklistFeedbackPayload
} from '@/api/admin/accounts'
import type { Account, ClaudeModel } from '@/types'
import { resolveEffectiveAccountPlatformFromAccount } from '@/utils/accountProtocolGateway'

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

type AccountTestModelOption = ClaudeModel & {
  description: string
  [key: string]: unknown
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
const availableModels = ref<ClaudeModel[]>([])
const availableModelOptions = computed<AccountTestModelOption[]>(() =>
  availableModels.value.map((model) => ({
    ...model,
    description: model.id
  }))
)
const selectedModelId = ref('')
const testPrompt = ref('')
const loadingModels = ref(false)
let eventSource: EventSource | null = null
const blacklistAdvice = ref<BlacklistAdvicePayload | null>(null)
const runtimePlatform = computed(() =>
  props.account ? resolveEffectiveAccountPlatformFromAccount(props.account) : null
)
const isSoraAccount = computed(() => props.account?.platform === 'sora')
const isKiroAccount = computed(() => props.account?.platform === 'kiro')
const generatedImages = ref<PreviewImage[]>([])
const supportsGeminiImageTest = computed(() => {
  if (isSoraAccount.value) return false
  const modelID = selectedModelId.value.toLowerCase()
  if (!modelID.startsWith('gemini-') || !modelID.includes('-image')) return false

  return runtimePlatform.value === 'gemini' || (props.account?.platform === 'antigravity' && props.account?.type === 'apikey')
})

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

watch(
  () => props.show,
  async (newVal) => {
    if (newVal && props.account) {
      testPrompt.value = ''
      resetState()
      await loadAvailableModels()
    } else {
      closeEventSource()
    }
  }
)

watch(selectedModelId, () => {
  if (supportsGeminiImageTest.value && !testPrompt.value.trim()) {
    testPrompt.value = t('admin.accounts.geminiImagePromptDefault')
  }
})

const loadAvailableModels = async () => {
  if (!props.account) return
  if (props.account.platform === 'sora') {
    availableModels.value = []
    selectedModelId.value = ''
    loadingModels.value = false
    return
  }

  loadingModels.value = true
  selectedModelId.value = ''
  try {
    const models = await adminAPI.accounts.getAvailableModels(props.account.id)
    availableModels.value = models
    selectedModelId.value = models[0]?.id || ''
  } catch (error) {
    console.error('Failed to load available models:', error)
    availableModels.value = []
    selectedModelId.value = ''
  } finally {
    loadingModels.value = false
  }
}

const resetState = () => {
  status.value = 'idle'
  outputLines.value = []
  streamingContent.value = ''
  errorMessage.value = ''
  generatedImages.value = []
  blacklistAdvice.value = null
}

const handleClose = () => {
  // 防止在连接测试进行中关闭对话框
  if (status.value === 'connecting') {
    return
  }
  closeEventSource()
  emit('close')
}

const closeEventSource = () => {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
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

const startTest = async () => {
  if (!props.account || (!isSoraAccount.value && !selectedModelId.value)) return

  resetState()
  status.value = 'connecting'
  addLine(t('admin.accounts.startingTestForAccount', { name: props.account.name }), 'text-blue-400')
  addLine(t('admin.accounts.testAccountTypeLabel', { type: props.account.type }), 'text-gray-400')
  addLine('', 'text-gray-300')

  closeEventSource()

  try {
    // Create EventSource for SSE
    const url = `/api/v1/admin/accounts/${props.account.id}/test`

    // Use fetch with streaming for SSE since EventSource doesn't support POST
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token')}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(
        isSoraAccount.value
          ? {}
          : {
              model_id: selectedModelId.value,
              model: selectedModelId.value,
              prompt: supportsGeminiImageTest.value ? testPrompt.value.trim() : ''
            }
      )
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('No response body')
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
    status.value = 'error'
    errorMessage.value = error.message || 'Unknown error'
    addLine(`Error: ${errorMessage.value}`, 'text-red-400')
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
        isSoraAccount.value
          ? t('admin.accounts.soraTestingFlow')
          : supportsGeminiImageTest.value
            ? t('admin.accounts.sendingGeminiImageRequest')
            : t('admin.accounts.sendingTestMessage'),
        'text-gray-400'
      )
      addLine('', 'text-gray-300')
      addLine(t('admin.accounts.response'), 'text-yellow-400')
      break

    case 'content':
      if (event.data?.kind === 'runtime_meta') {
        if (event.text) {
          addLine(event.text, 'text-sky-300')
        }
        break
      }
      if (event.text) {
        streamingContent.value += event.text
        scrollToBottom()
      }
      break

    case 'image':
      if (event.image_url) {
        generatedImages.value.push({
          url: event.image_url,
          mimeType: event.mime_type
        })
        addLine(t('admin.accounts.geminiImageReceived', { count: generatedImages.value.length }), 'text-purple-300')
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

const isDeprecatedModel = (model: Record<string, unknown> | null | undefined) => model?.status === 'deprecated'
</script>
