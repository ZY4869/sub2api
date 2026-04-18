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
import Select from '@/components/common/Select.vue'
import PlatformLabel from '@/components/common/PlatformLabel.vue'
import type { GeminiAIStudioTier, GeminiOAuthType } from '@/utils/geminiAccount'

type AccountCategory = 'oauth-based' | 'apikey' | 'vertex_ai'
type AntigravityAccountType = 'oauth' | 'upstream'
type GeminiGoogleOneTier = 'google_one_free' | 'google_ai_pro' | 'google_ai_ultra'
type GeminiGcpTier = 'gcp_standard' | 'gcp_enterprise'
type Accent = 'orange' | 'purple' | 'green' | 'rose'

interface TypeOption {
  key: string
  title: string
  description: string
  icon: 'sparkles' | 'key' | 'link' | 'cloud'
  accent: Accent
  active: boolean
}

interface GatewayProtocolOption extends Record<string, unknown> {
  value: GatewayProtocol
  label: string
  requestFormatsText: string
  iconPlatform: AccountPlatform
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
const antigravityAccountType = defineModel<AntigravityAccountType>('antigravityAccountType', { required: true })
const geminiOAuthType = defineModel<GeminiOAuthType>('geminiOAuthType', { required: true })
const showAdvanced = defineModel<boolean>('showAdvanced', { required: true })
const geminiTierGoogleOne = defineModel<GeminiGoogleOneTier>('geminiTierGoogleOne', { required: true })
const geminiTierGcp = defineModel<GeminiGcpTier>('geminiTierGcp', { required: true })
const geminiTierAiStudio = defineModel<GeminiAIStudioTier>('geminiTierAiStudio', { required: true })
const upstreamBaseUrl = defineModel<string>('upstreamBaseUrl', { required: true })
const upstreamApiKey = defineModel<string>('upstreamApiKey', { required: true })
const gatewayProtocol = defineModel<GatewayProtocol>('gatewayProtocol', { required: true })
const { t } = useI18n()

function selectAccountCategory(next: AccountCategory) {
  accountCategory.value = next
}

function selectAntigravityAccountType(next: AntigravityAccountType) {
  antigravityAccountType.value = next
}

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

const baiduDocumentAIOptions = computed<TypeOption[]>(() => [
  {
    key: 'apikey',
    title: 'API Key',
    description: t('admin.accounts.types.baiduDocumentAIApikey'),
    icon: 'key',
    accent: 'rose',
    active: true
  }
])

const grokOptions = computed<TypeOption[]>(() => [
  {
    key: 'oauth-based',
    title: t('admin.accounts.types.grokSso'),
    description: t('admin.accounts.types.grokSsoHint'),
    icon: 'sparkles',
    accent: 'green',
    active: accountCategory.value === 'oauth-based'
  },
  {
    key: 'apikey',
    title: 'API Key',
    description: t('admin.accounts.grokDedicatedRouteHint'),
    icon: 'key',
    accent: 'purple',
    active: accountCategory.value === 'apikey'
  }
])

const gatewayProtocolOptions = computed<GatewayProtocolOption[]>(() =>
  PROTOCOL_GATEWAY_PROTOCOLS.map((id) => {
    const descriptor = resolveGatewayProtocolDescriptor(id)
    return {
      value: id,
      label: descriptor?.displayName || id,
      requestFormatsText: (descriptor?.requestFormats || []).join(', '),
      iconPlatform: descriptor?.targetGroupPlatform || 'protocol_gateway'
    }
  })
)

function isGatewayProtocolOption(option: unknown): option is GatewayProtocolOption {
  return typeof option === 'object' && option !== null && 'value' in option && 'label' in option
}

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

function handleGrokSelect(key: string) {
  selectAccountCategory(key as AccountCategory)
}

function handleAntigravitySelect(key: string) {
  selectAntigravityAccountType(key as AntigravityAccountType)
}
</script>

<template>
  <template v-if="showStepOneTypeCards">
    <AccountCreateTypeCardGroup
      v-if="platform === 'anthropic'"
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

    <AccountCreateTypeCardGroup
      v-else-if="platform === 'grok'"
      :label="t('admin.accounts.accountType')"
      :options="grokOptions"
      tour="account-form-type"
      @select="handleGrokSelect"
    />

    <div v-else-if="platform === 'protocol_gateway'" class="space-y-4">
      <AccountCreateTypeCardGroup
        :label="t('admin.accounts.accountType')"
        :options="protocolGatewayOptions"
      />

      <div>
        <label class="input-label">{{ t('admin.accounts.protocolGateway.protocolLabel') }}</label>
        <Select
          v-model="gatewayProtocol"
          :options="gatewayProtocolOptions"
          value-key="value"
          label-key="label"
        >
          <template #selected="{ option }">
            <PlatformLabel
              v-if="isGatewayProtocolOption(option)"
              :platform="option.iconPlatform"
              :label="option.label"
              :description="option.requestFormatsText"
            />
          </template>

          <template #option="{ option }">
            <PlatformLabel
              v-if="isGatewayProtocolOption(option)"
              :platform="option.iconPlatform"
              :label="option.label"
              :description="option.requestFormatsText"
            />
          </template>
        </Select>
        <p class="input-hint">{{ t('admin.accounts.protocolGateway.protocolHint') }}</p>
      </div>
    </div>

    <AccountCreateTypeCardGroup
      v-else-if="platform === 'baidu_document_ai'"
      :label="t('admin.accounts.accountType')"
      :options="baiduDocumentAIOptions"
    />

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
