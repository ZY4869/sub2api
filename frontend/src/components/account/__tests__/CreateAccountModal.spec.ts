import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'),
  'utf-8'
)

const {
  createMock,
  checkMixedChannelRiskMock,
  invalidateModelRegistryMock,
  invalidateInventoryMock,
  modelRegistrySnapshot
} = vi.hoisted(() => ({
  createMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  invalidateModelRegistryMock: vi.fn(),
  invalidateInventoryMock: vi.fn(),
  modelRegistrySnapshot: {
    etag: 'test-etag',
    updated_at: '2026-04-08T00:00:00Z',
    provider_labels: {
      anthropic: 'Anthropic',
      openai: 'OpenAI'
    },
    models: [
      {
        id: 'claude-sonnet-4.5',
        provider: 'anthropic',
        display_name: 'Claude Sonnet 4.5',
        platforms: ['anthropic'],
        protocol_ids: ['claude-sonnet-4-5-20250929'],
        aliases: ['claude-sonnet-4-5-20250929'],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: ['text'],
        exposed_in: ['runtime', 'test', 'whitelist'],
        ui_priority: 1
      },
      {
        id: 'gpt-5.4',
        provider: 'openai',
        display_name: 'GPT-5.4',
        platforms: ['openai'],
        protocol_ids: ['gpt-5.4'],
        aliases: [],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: ['text'],
        exposed_in: ['runtime', 'test', 'whitelist'],
        ui_priority: 1
      }
    ],
    presets: []
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores', () => ({
  useModelInventoryStore: () => ({
    invalidate: invalidateInventoryMock
  })
}))

vi.mock('@/stores/modelRegistry', () => ({
  ensureModelRegistryFresh: vi.fn().mockResolvedValue(modelRegistrySnapshot),
  getModelRegistrySnapshot: vi.fn(() => modelRegistrySnapshot),
  invalidateModelRegistry: invalidateModelRegistryMock
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: true
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createMock,
      checkMixedChannelRisk: checkMixedChannelRiskMock
    }
  }
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import CreateAccountModal from '../CreateAccountModal.vue'
import { BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL } from '@/utils/baiduDocumentAI'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    },
    width: {
      type: String,
      default: 'normal'
    }
  },
  template: '<div v-if="show" :data-width="width"><slot /><slot name="footer" /></div>'
})

const AccountCreatePlatformSelectorStub = defineComponent({
  name: 'AccountCreatePlatformSelector',
  emits: ['update:platform'],
  template: `
    <div>
      <button
        type="button"
        data-testid="select-grok"
        @click="$emit('update:platform', 'grok')"
      >
        select grok
      </button>
      <button
        type="button"
        data-testid="select-protocol-gateway"
        @click="$emit('update:platform', 'protocol_gateway')"
      >
        select protocol gateway
      </button>
      <button
        type="button"
        data-testid="select-baidu-document-ai"
        @click="$emit('update:platform', 'baidu_document_ai')"
      >
        select baidu document ai
      </button>
      <button
        type="button"
        data-testid="select-baidu-legacy"
        @click="$emit('update:platform', 'baidu')"
      >
        select baidu legacy
      </button>
      <button
        type="button"
        data-testid="select-openai"
        @click="$emit('update:platform', 'openai')"
      >
        select openai
      </button>
      <button
        type="button"
        data-testid="select-anthropic"
        @click="$emit('update:platform', 'anthropic')"
      >
        select anthropic
      </button>
    </div>
  `
})

const AccountCreatePlatformTypeEditorStub = defineComponent({
  name: 'AccountCreatePlatformTypeEditor',
  emits: ['update:account-category', 'update:gateway-protocol'],
  template: `
    <div>
      <button
        type="button"
        data-testid="set-gateway-mixed"
        @click="
          $emit('update:account-category', 'apikey');
          $emit('update:gateway-protocol', 'mixed')
        "
      >
        set mixed gateway
      </button>
      <button
        type="button"
        data-testid="set-oauth-based-mode"
        @click="$emit('update:account-category', 'oauth-based')"
      >
        set oauth based mode
      </button>
      <button
        type="button"
        data-testid="set-gateway-gemini"
        @click="
          $emit('update:account-category', 'apikey');
          $emit('update:gateway-protocol', 'gemini')
        "
      >
        set gemini gateway
      </button>
      <button
        type="button"
        data-testid="set-apikey-mode"
        @click="$emit('update:account-category', 'apikey')"
      >
        set apikey mode
      </button>
    </div>
  `
})

const AccountApiKeyBasicSettingsEditorStub = defineComponent({
  name: 'AccountApiKeyBasicSettingsEditor',
  props: {
    actualModelLocked: {
      type: Boolean,
      default: true
    },
    modelScopeMode: {
      type: String,
      default: 'whitelist'
    },
    allowedModels: {
      type: Array,
      default: () => []
    },
    modelMappings: {
      type: Array,
      default: () => []
    },
    showGeminiTier: {
      type: Boolean,
      default: false
    },
    modelScopeEnabled: {
      type: Boolean,
      default: false
    },
    skipModelScopeEditor: {
      type: Boolean,
      default: false
    }
  },
  emits: ['update:api-key', 'update:base-url', 'update:allowedModels', 'update:modelScopeEnabled'],
  template: `
    <div>
      <span data-testid="actual-model-locked-prop">{{ actualModelLocked }}</span>
      <span data-testid="model-scope-mode-prop">{{ modelScopeMode }}</span>
      <span data-testid="model-scope-enabled-prop">{{ modelScopeEnabled }}</span>
      <span data-testid="skip-model-scope-editor-prop">{{ skipModelScopeEditor }}</span>
      <span data-testid="allowed-models-prop">
        {{ Array.isArray(allowedModels) ? allowedModels.join(',') : '' }}
      </span>
      <span data-testid="model-mappings-prop">
        {{ Array.isArray(modelMappings) ? JSON.stringify(modelMappings) : '' }}
      </span>
      <span data-testid="show-gemini-tier-prop">{{ showGeminiTier }}</span>
      <button
        type="button"
        data-testid="set-api-key"
        @click="
          $emit('update:api-key', 'gateway-key');
          $emit('update:base-url', 'https://gateway.example.com')
        "
      >
        set api key
      </button>
      <button
        type="button"
        data-testid="enable-model-restriction"
        @click="$emit('update:modelScopeEnabled', true)"
      >
        enable restriction
      </button>
      <button
        type="button"
        data-testid="set-whitelist-selection"
        @click="$emit('update:allowedModels', ['claude-sonnet-4-5-20250929', 'claude-sonnet-4.5'])"
      >
        set whitelist selection
      </button>
    </div>
  `
})

const AccountApiKeyModelProbeEditorStub = defineComponent({
  name: 'AccountApiKeyModelProbeEditor',
  emits: ['update:allowedModels', 'update:modelMappings'],
  template: `
    <div>
      <button
        type="button"
        data-testid="probe-select-models"
        @click="
          $emit('update:allowedModels', ['gemini-2.0-flash']);
          $emit('update:modelMappings', [{ from: 'friendly-flash', to: 'gemini-2.0-flash' }])
        "
      >
        select models
      </button>
      <button
        type="button"
        data-testid="probe-clear-models"
        @click="
          $emit('update:allowedModels', []);
          $emit('update:modelMappings', [])
        "
      >
        clear models
      </button>
    </div>
  `
})

const AccountProtocolGatewayModelProbeEditorStub = defineComponent({
  name: 'AccountProtocolGatewayModelProbeEditor',
  props: {
    gatewayTestProvider: {
      type: String,
      default: ''
    },
    gatewayTestModelId: {
      type: String,
      default: ''
    }
  },
  emits: [
    'update:allowedModels',
    'update:modelMappings',
    'update:gateway-test-provider',
    'update:gateway-test-model-id'
  ],
  template: `
    <div>
      <span data-testid="gateway-test-provider-prop">{{ gatewayTestProvider }}</span>
      <span data-testid="gateway-test-model-prop">{{ gatewayTestModelId }}</span>
      <button
        type="button"
        data-testid="set-gateway-defaults"
        @click="
          $emit('update:gateway-test-provider', 'anthropic');
          $emit('update:gateway-test-model-id', 'claude-sonnet-4.5')
        "
      >
        set defaults
      </button>
      <button
        type="button"
        data-testid="gateway-probe-select-models"
        @click="
          $emit('update:allowedModels', ['gpt-5.4']);
          $emit('update:modelMappings', [{ from: 'friendly-gateway-model', to: 'gpt-5.4' }])
        "
      >
        select gateway models
      </button>
      <button
        type="button"
        data-testid="gateway-probe-select-models-with-defaults"
        @click="
          $emit('update:allowedModels', ['gpt-5.4']);
          $emit('update:modelMappings', [{ from: 'friendly-gateway-model', to: 'gpt-5.4' }]);
          $emit('update:gateway-test-provider', 'openai');
          $emit('update:gateway-test-model-id', 'gpt-5.4')
        "
      >
        select gateway models with defaults
      </button>
      <button
        type="button"
        data-testid="gateway-probe-clear-models"
        @click="
          $emit('update:allowedModels', []);
          $emit('update:modelMappings', [])
        "
      >
        clear gateway models
      </button>
      <button
        type="button"
        data-testid="gateway-probe-clear-models-and-defaults"
        @click="
          $emit('update:allowedModels', []);
          $emit('update:modelMappings', []);
          $emit('update:gateway-test-provider', '');
          $emit('update:gateway-test-model-id', '')
        "
      >
        clear gateway models and defaults
      </button>
    </div>
  `
})

const AccountProtocolGatewayOpenAIRequestFormatEditorStub = defineComponent({
  name: 'AccountProtocolGatewayOpenAIRequestFormatEditor',
  props: {
    value: {
      type: String,
      default: '/v1/chat/completions'
    }
  },
  emits: ['update:value'],
  template: `
    <div>
      <span data-testid="gateway-openai-request-format-prop">{{ value }}</span>
      <button
        type="button"
        data-testid="set-gateway-openai-request-format"
        @click="$emit('update:value', '/v1/responses')"
      >
        set openai request format
      </button>
    </div>
  `
})

const AccountModelScopeEditorStub = defineComponent({
  name: 'AccountModelScopeEditor',
  props: {
    enabled: {
      type: Boolean,
      default: false
    },
    allowedModels: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:allowedModels', 'update:enabled'],
  template: `
    <div>
      <span data-testid="oauth-model-scope-enabled-prop">{{ enabled }}</span>
      <span data-testid="oauth-allowed-models-prop">
        {{ Array.isArray(allowedModels) ? allowedModels.join(',') : '' }}
      </span>
      <button
        type="button"
        data-testid="enable-oauth-model-restriction"
        @click="$emit('update:enabled', true)"
      >
        enable oauth restriction
      </button>
      <button
        type="button"
        data-testid="set-openai-custom-whitelist"
        @click="$emit('update:allowedModels', ['gpt-5.4'])"
      >
        set openai custom whitelist
      </button>
    </div>
  `
})

const AccountBaiduDocumentAICredentialsEditorStub = defineComponent({
  name: 'AccountBaiduDocumentAICredentialsEditor',
  props: {
    asyncBearerToken: {
      type: String,
      default: ''
    },
    asyncBaseUrl: {
      type: String,
      default: ''
    },
    directToken: {
      type: String,
      default: ''
    },
    directApiUrlsText: {
      type: String,
      default: ''
    }
  },
  emits: [
    'update:async-bearer-token',
    'update:async-base-url',
    'update:direct-token',
    'update:direct-api-urls-text'
  ],
  template: `
    <div data-testid="baidu-document-ai-credentials-editor">
      <span data-testid="baidu-async-bearer-token-prop">{{ asyncBearerToken }}</span>
      <span data-testid="baidu-async-base-url-prop">{{ asyncBaseUrl }}</span>
      <span data-testid="baidu-direct-token-prop">{{ directToken }}</span>
      <span data-testid="baidu-direct-api-urls-prop">{{ directApiUrlsText }}</span>
      <button
        type="button"
        data-testid="set-baidu-document-ai-credentials"
        @click="
          $emit('update:async-bearer-token', 'async-token');
          $emit('update:async-base-url', 'https://aistudio.baidu.com/async');
          $emit('update:direct-token', 'direct-token');
          $emit('update:direct-api-urls-text', '{&quot;pp-ocrv5-server&quot;:&quot;https://direct.baidu.com/ocr&quot;}')
        "
      >
        set baidu credentials
      </button>
    </div>
  `
})

function mountModal(stubOverrides: Record<string, any> = {}) {
  return mount(CreateAccountModal, {
    props: {
      show: true,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        AccountCreatePlatformSelector: AccountCreatePlatformSelectorStub,
        AccountCreatePlatformTypeEditor: AccountCreatePlatformTypeEditorStub,
        AccountApiKeyBasicSettingsEditor: AccountApiKeyBasicSettingsEditorStub,
        AccountBaiduDocumentAICredentialsEditor: AccountBaiduDocumentAICredentialsEditorStub,
        AccountProtocolGatewayModelProbeEditor: AccountProtocolGatewayModelProbeEditorStub,
        AccountProtocolGatewayOpenAIRequestFormatEditor: AccountProtocolGatewayOpenAIRequestFormatEditorStub,
        AccountApiKeyModelProbeEditor: AccountApiKeyModelProbeEditorStub,
        AccountAntigravityModelMappingEditor: true,
        AccountAutoPauseToggle: true,
        AccountCopilotDeviceFlowPanel: true,
        AccountCreateFooterActions: true,
        AccountCreateOAuthStep: true,
        AccountCustomErrorCodesEditor: true,
        AccountGatewaySettingsEditor: true,
        AccountGoogleBatchArchiveEditor: true,
        AccountGeminiHelpDialog: true,
        AccountGeminiVertexCredentialsEditor: true,
        AccountGrokImportPanel: true,
        AccountGroupSettingsEditor: true,
        AccountKiroAuthPanel: true,
        AccountMixedChannelWarningDialog: true,
        AccountModelScopeEditor: AccountModelScopeEditorStub,
        AccountPoolModeEditor: true,
        AccountProtocolGatewayClaudeMimicEditor: true,
        AccountProtocolGatewayBatchEditor: true,
        AccountQuotaControlEditor: true,
        AccountRuntimeSettingsEditor: true,
        AccountTempUnschedRulesEditor: true,
        QuotaLimitCard: true,
        ...stubOverrides
      }
    }
  })
}

describe('CreateAccountModal', () => {
  it('uses the dedicated account-wide dialog with local horizontal overflow protection', () => {
    expect(source).toContain('width="account-wide"')
    expect(source).toContain('class="min-w-0 overflow-x-hidden"')
  })

  it('keeps the OAuth step indicator responsive', () => {
    expect(source).toContain('flex-col items-center gap-3 sm:w-auto sm:flex-row sm:gap-4')
    expect(source).toContain('hidden h-0.5 w-8 bg-gray-300 dark:bg-dark-600 sm:block')
    expect(source).toContain('min-w-0 break-words text-sm font-medium text-gray-700 dark:text-gray-300')
  })

  it('uses the protocol gateway probe editor and hides the generic auto-import toggle for that platform', () => {
    expect(source).toContain('AccountProtocolGatewayModelProbeEditor')
    expect(source).toContain(':skip-model-scope-editor="!showApiKeyModelScopeEditor"')
    expect(source).toContain(
      ":show-auto-import=\"form.platform !== 'protocol_gateway' && !isBaiduDocumentAISelected\""
    )
  })

  it('embeds the Grok batch import panel alongside the single-account Grok fields', () => {
    expect(source).toContain('AccountGrokImportPanel')
    expect(source).toContain("@imported=\"handleGrokImportCompleted\"")
  })

  it('defaults Grok to API Key mode and only persists grok_tier for SSO submissions', () => {
    expect(source).toContain("if (newPlatform === 'grok')")
    expect(source).toContain("accountCategory.value = 'apikey'")
    expect(source).toContain("form.type = 'apikey'")
    expect(source).toContain("form.platform === 'grok' && form.type === 'sso'")
  })

  it('shows model scope controls for Grok API Key and Grok SSO account creation', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-grok"]').trigger('click')

    expect(wrapper.find('[data-testid="set-api-key"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="skip-model-scope-editor-prop"]').text()).toBe('false')
    expect(wrapper.find('[data-testid="oauth-allowed-models-prop"]').exists()).toBe(false)

    await wrapper.get('[data-testid="set-oauth-based-mode"]').trigger('click')

    expect(wrapper.find('[data-testid="set-api-key"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="oauth-allowed-models-prop"]').exists()).toBe(true)
  })

  it('submits Grok SSO model scope through extra.model_scope_v2 while preserving grok_tier', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 18,
      name: 'Grok SSO Account',
      platform: 'grok',
      type: 'sso',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-grok"]').trigger('click')
    await wrapper.get('[data-testid="set-oauth-based-mode"]').trigger('click')
    await wrapper.get('textarea[placeholder="admin.accounts.grokTokenPlaceholder"]').setValue('grok-sso-token')
    await wrapper.get('[data-testid="enable-oauth-model-restriction"]').trigger('click')
    await wrapper.get('[data-testid="set-openai-custom-whitelist"]').trigger('click')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Grok SSO Account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createMock).toHaveBeenCalledTimes(1)
    const payload = createMock.mock.calls[0]?.[0] as any
    expect(payload).toMatchObject({
      name: 'Grok SSO Account',
      platform: 'grok',
      type: 'sso',
      credentials: {
        sso_token: 'grok-sso-token'
      },
      extra: {
        grok_tier: 'basic',
        model_scope_v2: {
          policy_mode: 'whitelist'
        }
      }
    })
    expect(payload.extra.model_scope_v2.entries).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          display_model_id: 'gpt-5.4',
          target_model_id: 'gpt-5.4'
        })
      ])
    )
    expect(payload.credentials.model_mapping).toEqual({
      'gpt-5.4': 'gpt-5.4'
    })
  })

  it('shows generic quota controls and protocol gateway batch controls in account creation', () => {
    expect(source).toContain('const showQuotaLimitSection = computed(() =>')
    expect(source).toContain('const showQuotaLimitSection = computed(() => true)')
    expect(source).toContain('AccountProtocolGatewayBatchEditor')
    expect(source).toContain('const showProtocolGatewayBatchEditor = computed(() =>')
  })

  it('submits protocol gateway default provider/model fields from the modal flow', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 9,
      name: 'Gateway Account',
      platform: 'protocol_gateway',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')
    await wrapper.get('[data-testid="set-gateway-mixed"]').trigger('click')
    await wrapper.get('[data-testid="set-api-key"]').trigger('click')
    await wrapper.get('[data-testid="set-gateway-defaults"]').trigger('click')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Gateway Account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]).toMatchObject({
      name: 'Gateway Account',
      platform: 'protocol_gateway',
      gateway_protocol: 'mixed',
      type: 'apikey',
      credentials: {
        api_key: 'gateway-key',
        base_url: 'https://gateway.example.com'
      },
      extra: {
        gateway_protocol: 'mixed',
        gateway_test_provider: 'anthropic',
        gateway_test_model_id: 'claude-sonnet-4.5'
      }
    })
  })

  it('submits protocol gateway model scope with selected target models and explicit aliases only', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 14,
      name: 'Gateway Scoped Account',
      platform: 'protocol_gateway',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')
    await wrapper.get('[data-testid="set-api-key"]').trigger('click')
    await wrapper.get('[data-testid="enable-model-restriction"]').trigger('click')
    await wrapper.get('[data-testid="gateway-probe-select-models"]').trigger('click')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Gateway Scoped Account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]?.credentials?.model_mapping).toEqual({
      'friendly-gateway-model': 'gpt-5.4'
    })
    expect(createMock.mock.calls[0]?.[0]?.extra?.model_scope_v2).toMatchObject({
      policy_mode: 'mapping',
      entries: [{
        display_model_id: 'friendly-gateway-model',
        target_model_id: 'gpt-5.4',
        provider: 'openai',
        source_protocol: 'openai',
        visibility_mode: 'alias'
      }]
    })
  })

  it('submits whitelist selected_model_ids alongside canonical supported models', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 15,
      name: 'Anthropic Scoped Account',
      platform: 'anthropic',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-anthropic"]').trigger('click')
    await wrapper.get('[data-testid="set-apikey-mode"]').trigger('click')
    await wrapper.get('[data-testid="set-api-key"]').trigger('click')
    await wrapper.get('[data-testid="enable-model-restriction"]').trigger('click')
    await wrapper.get('[data-testid="set-whitelist-selection"]').trigger('click')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Anthropic Scoped Account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]).toMatchObject({
      platform: 'anthropic',
      type: 'apikey',
      extra: {
        model_scope_v2: {
          policy_mode: 'whitelist',
          entries: [
            {
              display_model_id: 'claude-sonnet-4-5-20250929',
              target_model_id: 'claude-sonnet-4-5-20250929',
              provider: 'anthropic',
              source_protocol: 'anthropic',
              visibility_mode: 'direct'
            },
            {
              display_model_id: 'claude-sonnet-4.5',
              target_model_id: 'claude-sonnet-4.5',
              provider: 'anthropic',
              source_protocol: 'anthropic',
              visibility_mode: 'direct'
            }
          ]
        }
      }
    })
  })

  it('keeps the openai login flow whitelist empty until models are explicitly selected', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-openai"]').trigger('click')

    expect(wrapper.get('[data-testid="oauth-model-scope-enabled-prop"]').text()).toBe('true')
    expect(wrapper.get('[data-testid="oauth-allowed-models-prop"]').text()).toBe('')
  })

  it('persists the protocol gateway OpenAI request format selection', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 10,
      name: 'Gateway OpenAI Account',
      platform: 'protocol_gateway',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')
    await wrapper.get('[data-testid="set-api-key"]').trigger('click')
    await wrapper.get('[data-testid="set-gateway-openai-request-format"]').trigger('click')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Gateway OpenAI Account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]?.extra?.gateway_openai_request_format).toBe('/v1/responses')
  })

  it('does not send gemini tier_id for protocol gateway gemini accounts', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 11,
      name: 'Gateway Gemini Account',
      platform: 'protocol_gateway',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')
    await wrapper.get('[data-testid="set-gateway-gemini"]').trigger('click')
    expect(wrapper.get('[data-testid="show-gemini-tier-prop"]').text()).toBe('false')

    await wrapper.get('[data-testid="set-api-key"]').trigger('click')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Gateway Gemini Account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]).toMatchObject({
      name: 'Gateway Gemini Account',
      platform: 'protocol_gateway',
      gateway_protocol: 'gemini',
      type: 'apikey'
    })
    expect(createMock.mock.calls[0]?.[0]?.credentials?.tier_id).toBeUndefined()
  })

  it('submits baidu document ai credentials from the dedicated editor', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 12,
      name: 'Baidu OCR',
      platform: 'baidu_document_ai',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-baidu-document-ai"]').trigger('click')
    await wrapper.get('[data-testid="set-baidu-document-ai-credentials"]').trigger('click')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Baidu OCR')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]).toMatchObject({
      name: 'Baidu OCR',
      platform: 'baidu_document_ai',
      type: 'apikey',
      credentials: {
        async_bearer_token: 'async-token',
        async_base_url: 'https://aistudio.baidu.com/async',
        direct_token: 'direct-token',
        direct_api_urls: {
          'pp-ocrv5-server': 'https://direct.baidu.com/ocr'
        }
      }
    })
  })

  it('renders the real baidu document ai credential inputs instead of the generic api key editor', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 16,
      name: 'Baidu Real Editor',
      platform: 'baidu_document_ai',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal({
      AccountBaiduDocumentAICredentialsEditor: false
    })

    await wrapper.get('[data-testid="select-baidu-document-ai"]').trigger('click')

    expect(wrapper.find('[data-testid="baidu-document-ai-async-bearer-token"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="baidu-document-ai-direct-token"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="baidu-document-ai-direct-api-urls"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="set-api-key"]').exists()).toBe(false)

    await wrapper.get('[data-testid="baidu-document-ai-async-bearer-token"]').setValue('async-token')
    await wrapper.get('[data-testid="baidu-document-ai-direct-token"]').setValue('direct-token')
    await wrapper
      .get('[data-testid="baidu-document-ai-direct-api-urls"]')
      .setValue('{"pp-ocrv5-server":"https://direct.baidu.com/ocr"}')
    await wrapper.get('input[data-tour="account-form-name"]').setValue('Baidu Real Editor')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]).toMatchObject({
      name: 'Baidu Real Editor',
      platform: 'baidu_document_ai',
      type: 'apikey',
      credentials: {
        async_bearer_token: 'async-token',
        async_base_url: BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL,
        direct_token: 'direct-token',
        direct_api_urls: {
          'pp-ocrv5-server': 'https://direct.baidu.com/ocr'
        }
      }
    })
  })

  it('resets baidu document ai fields and model scope state after switching away and back', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')
    await wrapper.get('[data-testid="gateway-probe-select-models"]').trigger('click')

    expect(wrapper.get('[data-testid="allowed-models-prop"]').text()).toBe('gpt-5.4')
    expect(wrapper.get('[data-testid="model-mappings-prop"]').text()).toContain('friendly-gateway-model')

    await wrapper.get('[data-testid="select-baidu-document-ai"]').trigger('click')
    await wrapper.get('[data-testid="set-baidu-document-ai-credentials"]').trigger('click')
    expect(wrapper.get('[data-testid="baidu-async-bearer-token-prop"]').text()).toBe('async-token')
    expect(wrapper.get('[data-testid="baidu-direct-token-prop"]').text()).toBe('direct-token')
    expect(wrapper.get('[data-testid="baidu-direct-api-urls-prop"]').text()).toContain('pp-ocrv5-server')

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')

    expect(wrapper.get('[data-testid="actual-model-locked-prop"]').text()).toBe('true')
    expect(wrapper.get('[data-testid="allowed-models-prop"]').text()).toBe('')
    expect(wrapper.get('[data-testid="model-mappings-prop"]').text()).toBe('[]')

    await wrapper.get('[data-testid="select-baidu-document-ai"]').trigger('click')

    expect(wrapper.get('[data-testid="baidu-async-bearer-token-prop"]').text()).toBe('')
    expect(wrapper.get('[data-testid="baidu-async-base-url-prop"]').text()).toBe(
      BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
    )
    expect(wrapper.get('[data-testid="baidu-direct-token-prop"]').text()).toBe('')
    expect(wrapper.get('[data-testid="baidu-direct-api-urls-prop"]').text()).toBe('')
  })

  it('mounts the baidu document ai credential editor only when platform is selected', async () => {
    const wrapper = mountModal()

    expect(wrapper.find('[data-testid="baidu-document-ai-credentials-editor"]').exists()).toBe(false)

    await wrapper.get('[data-testid="select-baidu-document-ai"]').trigger('click')

    expect(wrapper.find('[data-testid="baidu-document-ai-credentials-editor"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="oauth-model-scope-enabled-prop"]').text()).toBe('true')

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')

    expect(wrapper.find('[data-testid="baidu-document-ai-credentials-editor"]').exists()).toBe(false)
  })

  it('returns to the basic info step when switching from OAuth flow platforms to baidu document ai', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-openai"]').trigger('click')

    ;(wrapper.vm as any).step = 2
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="baidu-document-ai-credentials-editor"]').exists()).toBe(false)

    ;(wrapper.vm as any).form.platform = 'baidu_document_ai'
    await flushPromises()

    expect((wrapper.vm as any).step).toBe(1)
    expect(wrapper.find('[data-testid="baidu-document-ai-selected-hint"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="baidu-document-ai-credentials-editor"]').exists()).toBe(true)
  })

  it('shows the baidu document ai credential editor for legacy baidu platform values', async () => {
    const wrapper = mountModal()

    expect(wrapper.find('[data-testid="baidu-document-ai-credentials-editor"]').exists()).toBe(false)

    await wrapper.get('[data-testid="select-baidu-legacy"]').trigger('click')

    expect(wrapper.find('[data-testid="baidu-document-ai-credentials-editor"]').exists()).toBe(true)
  })

  it('keeps the top mapping state in sync when probe selections are added and removed', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')
    await wrapper.get('[data-testid="gateway-probe-select-models"]').trigger('click')

    expect(wrapper.get('[data-testid="allowed-models-prop"]').text()).toBe('gpt-5.4')
    expect(wrapper.get('[data-testid="model-mappings-prop"]').text()).toContain('friendly-gateway-model')

    await wrapper.get('[data-testid="gateway-probe-clear-models"]').trigger('click')

    expect(wrapper.get('[data-testid="allowed-models-prop"]').text()).toBe('')
    expect(wrapper.get('[data-testid="model-mappings-prop"]').text()).toBe('[]')
  })

  it('clears gateway test defaults from the submitted payload when the selected gateway models are reset', async () => {
    createMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    invalidateModelRegistryMock.mockReset()
    invalidateInventoryMock.mockReset()

    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    createMock.mockResolvedValue({
      id: 13,
      name: 'Gateway Reset Account',
      platform: 'protocol_gateway',
      type: 'apikey',
      extra: {}
    })

    const wrapper = mountModal()

    await wrapper.get('[data-testid="select-protocol-gateway"]').trigger('click')
    await wrapper.get('[data-testid="set-api-key"]').trigger('click')
    await wrapper.get('[data-testid="gateway-probe-select-models-with-defaults"]').trigger('click')
    expect(wrapper.get('[data-testid="gateway-test-provider-prop"]').text()).toBe('openai')
    expect(wrapper.get('[data-testid="gateway-test-model-prop"]').text()).toBe('gpt-5.4')

    await wrapper.get('[data-testid="gateway-probe-clear-models-and-defaults"]').trigger('click')
    expect(wrapper.get('[data-testid="gateway-test-provider-prop"]').text()).toBe('')
    expect(wrapper.get('[data-testid="gateway-test-model-prop"]').text()).toBe('')

    await wrapper.get('input[data-tour="account-form-name"]').setValue('Gateway Reset Account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]?.credentials?.model_mapping).toBeUndefined()
    expect(createMock.mock.calls[0]?.[0]?.extra?.model_scope_v2).toBeUndefined()
    expect(createMock.mock.calls[0]?.[0]?.extra?.gateway_test_provider).toBeUndefined()
    expect(createMock.mock.calls[0]?.[0]?.extra?.gateway_test_model_id).toBeUndefined()
  })
})
