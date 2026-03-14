<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryPreset } from '@/generated/modelRegistry'
import type { AccountPlatform } from '@/types'
import type { ModelMapping } from '@/utils/accountFormShared'
import {
  resolveAccountApiKeyBaseUrlHintKey,
  resolveAccountApiKeyDefaultBaseUrl,
  resolveAccountApiKeyHintKey,
  resolveAccountApiKeyPlaceholder,
  type AccountApiKeySettingsMode
} from '@/utils/accountApiKeyBasicSettings'
import AccountModelScopeEditor from './AccountModelScopeEditor.vue'

type GeminiAiStudioTier = 'aistudio_free' | 'aistudio_paid'

const props = withDefaults(defineProps<{
  platform: AccountPlatform
  mode: AccountApiKeySettingsMode
  modelScopeDisabled?: boolean
  modelMappings: ModelMapping[]
  presetMappings: ModelRegistryPreset[]
  getMappingKey: (mapping: ModelMapping) => string
  showGeminiTier?: boolean
}>(), {
  modelScopeDisabled: false,
  showGeminiTier: false
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
const geminiTierAiStudio = defineModel<GeminiAiStudioTier>('geminiTierAiStudio')

const { t } = useI18n()

const baseUrlHint = computed(() => t(resolveAccountApiKeyBaseUrlHintKey(props.platform, props.mode)))
const baseUrlPlaceholder = computed(() => resolveAccountApiKeyDefaultBaseUrl(props.platform))
const apiKeyHint = computed(() => t(resolveAccountApiKeyHintKey(props.platform, props.mode)))
const apiKeyLabel = computed(() =>
  props.mode === 'create' ? t('admin.accounts.apiKeyRequired') : t('admin.accounts.apiKey')
)
const apiKeyPlaceholder = computed(() => resolveAccountApiKeyPlaceholder(props.platform))
const showModelScopeEditor = computed(() => props.platform !== 'antigravity')
</script>

<template>
  <div class="space-y-4">
    <div>
      <label class="input-label">{{ t('admin.accounts.baseUrl') }}</label>
      <input
        v-model="baseUrl"
        type="text"
        class="input"
        :placeholder="baseUrlPlaceholder"
      />
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
        <option value="aistudio_paid">{{ t('admin.accounts.gemini.tier.aiStudio.paid') }}</option>
      </select>
      <p class="input-hint">{{ t('admin.accounts.gemini.tier.aiStudioHint') }}</p>
    </div>

    <AccountModelScopeEditor
      v-if="showModelScopeEditor"
      :disabled="modelScopeDisabled"
      :platform="platform"
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
