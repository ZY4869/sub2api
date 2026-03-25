<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AddMethod } from '@/composables/useAccountOAuth'
import type { AccountPlatform, GatewayProtocol } from '@/types'
import {
  PROTOCOL_GATEWAY_PROTOCOLS,
  resolveGatewayProtocolDescriptor
} from '@/utils/accountProtocolGateway'
import AccountCreateAddMethodSelector from './AccountCreateAddMethodSelector.vue'
import AccountCreateTypeCardGroup from './AccountCreateTypeCardGroup.vue'
import AccountGeminiAccountTypeEditor from './AccountGeminiAccountTypeEditor.vue'
import AccountUpstreamSettingsEditor from './AccountUpstreamSettingsEditor.vue'

type AccountCategory = 'oauth-based' | 'apikey'
type SoraAccountType = 'oauth' | 'apikey'
type AntigravityAccountType = 'oauth' | 'upstream'
type GeminiOAuthType = 'code_assist' | 'google_one' | 'ai_studio'
type GeminiGoogleOneTier = 'google_one_free' | 'google_ai_pro' | 'google_ai_ultra'
type GeminiGcpTier = 'gcp_standard' | 'gcp_enterprise'
type GeminiAiStudioTier = 'aistudio_free' | 'aistudio_paid'
type Accent = 'orange' | 'purple' | 'green' | 'rose'

interface TypeOption {
  key: string
  title: string
  description: string
  icon: 'sparkles' | 'key' | 'link' | 'cloud'
  accent: Accent
  active: boolean
}

defineProps<{
  aiStudioOAuthEnabled: boolean
  apiKeyHelpLink: string
  gcpProjectHelpLink: string
}>()

defineEmits<{
  openGeminiHelp: []
}>()

const platform = defineModel<AccountPlatform>('platform', { required: true })
const accountCategory = defineModel<AccountCategory>('accountCategory', { required: true })
const addMethod = defineModel<AddMethod>('addMethod', { required: true })
const soraAccountType = defineModel<SoraAccountType>('soraAccountType', { required: true })
const antigravityAccountType = defineModel<AntigravityAccountType>('antigravityAccountType', { required: true })
const geminiOAuthType = defineModel<GeminiOAuthType>('geminiOAuthType', { required: true })
const showAdvanced = defineModel<boolean>('showAdvanced', { required: true })
const geminiTierGoogleOne = defineModel<GeminiGoogleOneTier>('geminiTierGoogleOne', { required: true })
const geminiTierGcp = defineModel<GeminiGcpTier>('geminiTierGcp', { required: true })
const geminiTierAiStudio = defineModel<GeminiAiStudioTier>('geminiTierAiStudio', { required: true })
const upstreamBaseUrl = defineModel<string>('upstreamBaseUrl', { required: true })
const upstreamApiKey = defineModel<string>('upstreamApiKey', { required: true })
const gatewayProtocol = defineModel<GatewayProtocol>('gatewayProtocol', { required: true })
const { t } = useI18n()

function selectAccountCategory(next: AccountCategory) {
  accountCategory.value = next
}

function selectSoraAccountType(next: SoraAccountType) {
  soraAccountType.value = next
  if (next === 'oauth') {
    accountCategory.value = 'oauth-based'
    addMethod.value = 'oauth'
    return
  }
  accountCategory.value = 'apikey'
}

function selectAntigravityAccountType(next: AntigravityAccountType) {
  antigravityAccountType.value = next
}

const soraOptions = computed<TypeOption[]>(() => [
  {
    key: 'oauth',
    title: 'OAuth',
    description: t('admin.accounts.types.chatgptOauth'),
    icon: 'key',
    accent: 'rose',
    active: soraAccountType.value === 'oauth'
  },
  {
    key: 'apikey',
    title: t('admin.accounts.types.soraApiKey'),
    description: t('admin.accounts.types.soraApiKeyHint'),
    icon: 'link',
    accent: 'rose',
    active: soraAccountType.value === 'apikey'
  }
])

const anthropicOptions = computed<TypeOption[]>(() => [
  {
    key: 'oauth-based',
    title: t('admin.accounts.claudeCode'),
    description: t('admin.accounts.oauthSetupToken'),
    icon: 'sparkles',
    accent: 'orange',
    active: accountCategory.value === 'oauth-based'
  },
  {
    key: 'apikey',
    title: t('admin.accounts.claudeConsole'),
    description: t('admin.accounts.apiKey'),
    icon: 'key',
    accent: 'purple',
    active: accountCategory.value === 'apikey'
  }
])

const openAIOptions = computed<TypeOption[]>(() => [
  {
    key: 'oauth-based',
    title: 'OAuth',
    description: t('admin.accounts.types.chatgptOauth'),
    icon: 'key',
    accent: 'green',
    active: accountCategory.value === 'oauth-based'
  },
  {
    key: 'apikey',
    title: 'API Key',
    description: t('admin.accounts.types.responsesApi'),
    icon: 'key',
    accent: 'purple',
    active: accountCategory.value === 'apikey'
  }
])

const protocolGatewayOptions = computed<TypeOption[]>(() => [
  {
    key: 'apikey',
    title: 'API Key',
    description: t('admin.accounts.protocolGateway.apiKeyOnly'),
    icon: 'key',
    accent: 'green',
    active: true
  }
])

const gatewayProtocolOptions = computed(() =>
  PROTOCOL_GATEWAY_PROTOCOLS.map((id) => ({
    value: id,
    label: resolveGatewayProtocolDescriptor(id)?.displayName || id
  }))
)

const antigravityOptions = computed<TypeOption[]>(() => [
  {
    key: 'oauth',
    title: 'OAuth',
    description: t('admin.accounts.types.antigravityOauth'),
    icon: 'key',
    accent: 'purple',
    active: antigravityAccountType.value === 'oauth'
  },
  {
    key: 'upstream',
    title: 'API Key',
    description: t('admin.accounts.types.antigravityApikey'),
    icon: 'cloud',
    accent: 'purple',
    active: antigravityAccountType.value === 'upstream'
  }
])

const showStepOneTypeCards = computed(() => platform.value !== 'kiro' && platform.value !== 'copilot')

const showAnthropicAddMethod = computed(() => platform.value === 'anthropic' && accountCategory.value === 'oauth-based')

function handleAnthropicSelect(key: string) {
  selectAccountCategory(key as AccountCategory)
}

function handleOpenAISelect(key: string) {
  selectAccountCategory(key as AccountCategory)
}

function handleSoraSelect(key: string) {
  selectSoraAccountType(key as SoraAccountType)
}

function handleAntigravitySelect(key: string) {
  selectAntigravityAccountType(key as AntigravityAccountType)
}
</script>

<template>
  <template v-if="showStepOneTypeCards">
    <AccountCreateTypeCardGroup
      v-if="platform === 'sora'"
      :label="t('admin.accounts.accountType')"
      :options="soraOptions"
      tour="account-form-type"
      @select="handleSoraSelect"
    />

    <AccountCreateTypeCardGroup
      v-else-if="platform === 'anthropic'"
      :label="t('admin.accounts.accountType')"
      :options="anthropicOptions"
      tour="account-form-type"
      @select="handleAnthropicSelect"
    />

    <AccountCreateTypeCardGroup
      v-else-if="platform === 'openai'"
      :label="t('admin.accounts.accountType')"
      :options="openAIOptions"
      tour="account-form-type"
      @select="handleOpenAISelect"
    />

    <div v-else-if="platform === 'protocol_gateway'" class="space-y-4">
      <AccountCreateTypeCardGroup
        :label="t('admin.accounts.accountType')"
        :options="protocolGatewayOptions"
      />

      <div>
        <label class="input-label">{{ t('admin.accounts.protocolGateway.protocolLabel') }}</label>
        <select v-model="gatewayProtocol" class="input">
          <option
            v-for="option in gatewayProtocolOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </option>
        </select>
        <p class="input-hint">{{ t('admin.accounts.protocolGateway.protocolHint') }}</p>
      </div>
    </div>

    <AccountGeminiAccountTypeEditor
      v-else-if="platform === 'gemini'"
      v-model:account-category="accountCategory"
      v-model:oauth-type="geminiOAuthType"
      v-model:show-advanced="showAdvanced"
      v-model:tier-google-one="geminiTierGoogleOne"
      v-model:tier-gcp="geminiTierGcp"
      v-model:tier-ai-studio="geminiTierAiStudio"
      :ai-studio-o-auth-enabled="aiStudioOAuthEnabled"
      :api-key-help-link="apiKeyHelpLink"
      :gcp-project-help-link="gcpProjectHelpLink"
      @open-help="$emit('openGeminiHelp')"
    />

    <div v-else-if="platform === 'antigravity'" class="space-y-4">
      <AccountCreateTypeCardGroup
        :label="t('admin.accounts.accountType')"
        :options="antigravityOptions"
        @select="handleAntigravitySelect"
      />

      <div v-if="antigravityAccountType === 'upstream'">
        <AccountUpstreamSettingsEditor
          v-model:base-url="upstreamBaseUrl"
          v-model:api-key="upstreamApiKey"
          mode="create"
        />
      </div>
    </div>
  </template>

  <AccountCreateAddMethodSelector
    v-if="showAnthropicAddMethod"
    v-model:add-method="addMethod"
  />
</template>
