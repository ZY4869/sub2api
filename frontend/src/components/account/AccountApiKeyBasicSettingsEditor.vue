<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryPreset } from '@/generated/modelRegistry'
import type { AccountPlatform, GatewayProtocol } from '@/types'
import type { ModelMapping } from '@/utils/accountFormShared'
import { type GeminiAIStudioTier } from '@/utils/geminiAccount'
import {
  resolveAccountApiKeyBaseUrlHintKey,
  resolveAccountApiKeyDefaultBaseUrl,
  resolveAccountApiKeyHintKey,
  resolveAccountApiKeyPlaceholder,
  shouldSuggestProtocolGateway,
  type AccountApiKeySettingsMode
} from '@/utils/accountApiKeyBasicSettings'
import { checkProtocolGatewayBaseUrl } from '@/utils/protocolGatewayBaseUrl'
import AccountModelScopeEditor from './AccountModelScopeEditor.vue'

const props = withDefaults(defineProps<{
  platform: AccountPlatform
  gatewayProtocol?: GatewayProtocol
  effectivePlatform?: AccountPlatform
  mode: AccountApiKeySettingsMode
  modelScopeDisabled?: boolean
  skipModelScopeEditor?: boolean
  modelMappings: ModelMapping[]
  presetMappings: ModelRegistryPreset[]
  getMappingKey: (mapping: ModelMapping) => string
  showGeminiTier?: boolean
}>(), {
  modelScopeDisabled: false,
  skipModelScopeEditor: false,
  showGeminiTier: false,
  gatewayProtocol: undefined,
  effectivePlatform: undefined
})

const emit = defineEmits<{
  'add-mapping': []
  'remove-mapping': [index: number]
  'add-preset': [payload: { from: string; to: string }]
}>()

const baseUrl = defineModel<string>('baseUrl', { required: true })
const apiKey = defineModel<string>('apiKey', { required: true })
const modelScopeMode = defineModel<'whitelist' | 'mapping'>('modelScopeMode', { required: true })
const allowedModels = defineModel<string[]>('allowedModels', { required: true })
const geminiTierAiStudio = defineModel<GeminiAIStudioTier>('geminiTierAiStudio')

const { t } = useI18n()

const resolvedEffectivePlatform = computed(() => props.effectivePlatform || props.platform)
const baseUrlHint = computed(() =>
  t(resolveAccountApiKeyBaseUrlHintKey(props.platform, props.mode, props.gatewayProtocol))
)
const baseUrlPlaceholder = computed(() =>
  resolveAccountApiKeyDefaultBaseUrl(props.platform, props.gatewayProtocol)
)
const apiKeyHint = computed(() =>
  t(resolveAccountApiKeyHintKey(props.platform, props.mode, props.gatewayProtocol))
)
const apiKeyLabel = computed(() =>
  props.mode === 'create' ? t('admin.accounts.apiKeyRequired') : t('admin.accounts.apiKey')
)
const apiKeyPlaceholder = computed(() =>
  resolveAccountApiKeyPlaceholder(props.platform, props.gatewayProtocol)
)
const showModelScopeEditor = computed(() =>
  !props.skipModelScopeEditor && resolvedEffectivePlatform.value !== 'antigravity'
)
const showProtocolGatewaySuggestion = computed(() =>
  shouldSuggestProtocolGateway(props.platform, baseUrl.value)
)
const protocolGatewayBaseUrlWarning = computed(() => {
  if (props.platform !== 'protocol_gateway') {
    return ''
  }
  const result = checkProtocolGatewayBaseUrl(baseUrl.value)
  if (result.status === 'invalid') {
    return t('admin.accounts.protocolGateway.baseUrlInvalidWarning')
  }
  if (result.status === 'loopback') {
    return t('admin.accounts.protocolGateway.baseUrlLoopbackWarning', {
      host: result.displayHost || result.input
    })
  }
  return ''
})
</script>

<template>
  <div class="space-y-4">
    <div
      v-if="showProtocolGatewaySuggestion"
      class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-300"
    >
      {{ t('admin.accounts.protocolGateway.migrationSuggestion') }}
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.baseUrl') }}</label>
      <input
        v-model="baseUrl"
        type="text"
        class="input"
        :placeholder="baseUrlPlaceholder"
      />
      <p
        v-if="protocolGatewayBaseUrlWarning"
        class="mt-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-300"
      >
        {{ protocolGatewayBaseUrlWarning }}
      </p>
      <p class="input-hint">{{ baseUrlHint }}</p>
    </div>

    <div>
      <label class="input-label">{{ apiKeyLabel }}</label>
      <input
        v-model="apiKey"
        type="password"
        class="input font-mono"
        :required="mode === 'create'"
        :placeholder="apiKeyPlaceholder"
      />
      <p class="input-hint">{{ apiKeyHint }}</p>
    </div>

    <div v-if="showGeminiTier">
      <label class="input-label">{{ t('admin.accounts.gemini.tier.label') }}</label>
      <select v-model="geminiTierAiStudio" class="input" data-testid="gemini-api-key-tier">
        <option value="aistudio_free">{{ t('admin.accounts.gemini.tier.aiStudio.free') }}</option>
        <option value="aistudio_tier_1">{{ t('admin.accounts.gemini.tier.aiStudio.tier1') }}</option>
        <option value="aistudio_tier_2">{{ t('admin.accounts.gemini.tier.aiStudio.tier2') }}</option>
        <option value="aistudio_tier_3">{{ t('admin.accounts.gemini.tier.aiStudio.tier3') }}</option>
      </select>
      <p class="input-hint">{{ t('admin.accounts.gemini.tier.aiStudioHint') }}</p>
    </div>

    <AccountModelScopeEditor
      v-if="showModelScopeEditor"
      :disabled="modelScopeDisabled"
      :platform="resolvedEffectivePlatform"
      :mode="modelScopeMode"
      :allowed-models="allowedModels"
      :model-mappings="modelMappings"
      :preset-mappings="presetMappings"
      :get-mapping-key="getMappingKey"
      @update:mode="modelScopeMode = $event"
      @update:allowedModels="allowedModels = $event"
      @add-mapping="emit('add-mapping')"
      @remove-mapping="emit('remove-mapping', $event)"
      @add-preset="emit('add-preset', $event)"
    />
  </div>
</template>
