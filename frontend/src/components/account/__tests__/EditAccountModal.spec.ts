import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { resolveAccountApiKeyDefaultBaseUrl } from '@/utils/accountApiKeyBasicSettings'

const { updateAccountMock, checkMixedChannelRiskMock, modelRegistrySnapshot } = vi.hoisted(() => ({
  updateAccountMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
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
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: true
  })
}))

vi.mock('@/stores/modelRegistry', () => ({
  ensureModelRegistryFresh: vi.fn().mockResolvedValue(modelRegistrySnapshot),
  getModelRegistrySnapshot: vi.fn(() => modelRegistrySnapshot),
  invalidateModelRegistry: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      update: updateAccountMock,
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

import EditAccountModal from '../EditAccountModal.vue'

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

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean, Object],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    },
    valueKey: {
      type: String,
      default: 'value'
    },
    labelKey: {
      type: String,
      default: 'label'
    }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      class="input"
      data-testid="select-stub"
      :value="String(modelValue ?? '')"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option
        v-for="option in options"
        :key="String(typeof option === 'object' ? option[valueKey] ?? option[labelKey] : option)"
        :value="typeof option === 'object' ? option[valueKey] : option"
      >
        {{ typeof option === 'object' ? option[labelKey] : option }}
      </option>
    </select>
  `
})

const AccountApiKeyBasicSettingsEditorStub = defineComponent({
  name: 'AccountApiKeyBasicSettingsEditor',
  props: {
    baseUrl: {
      type: String,
      default: ''
    },
    actualModelLocked: {
      type: Boolean,
      default: true
    },
    modelMappings: {
      type: Array,
      default: () => []
    },
    allowedModels: {
      type: Array,
      default: () => []
    },
    showGeminiTier: {
      type: Boolean,
      default: false
    }
  },
  emits: ['update:allowedModels', 'update:baseUrl'],
  template: `
    <div>
      <input
        type="text"
        data-testid="account-base-url-input"
        :value="baseUrl"
        @input="$emit('update:baseUrl', $event.target.value)"
      />
      <span data-testid="actual-model-locked-prop">{{ actualModelLocked }}</span>
      <button
        type="button"
        data-testid="rewrite-to-snapshot"
        @click="$emit('update:allowedModels', ['gpt-5.2-2025-12-11'])"
      >
        rewrite
      </button>
      <span data-testid="model-whitelist-value">
        {{ Array.isArray(allowedModels) ? allowedModels.join(',') : '' }}
      </span>
      <span data-testid="model-mappings-prop">
        {{ Array.isArray(modelMappings) ? JSON.stringify(modelMappings) : '' }}
      </span>
      <span data-testid="show-gemini-tier-prop">{{ showGeminiTier }}</span>
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
          $emit('update:allowedModels', ['friendly-gpt']);
          $emit('update:modelMappings', [{ from: 'friendly-gpt', to: 'gpt-5.4' }])
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
    allowedModels: {
      type: Array,
      default: () => []
    },
    modelMappings: {
      type: Array,
      default: () => []
    },
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
    'update:gatewayTestProvider',
    'update:gatewayTestModelId',
    'update:gateway-test-provider',
    'update:gateway-test-model-id'
  ],
  template: `
    <div>
      <span data-testid="gateway-allowed-models-prop">
        {{ Array.isArray(allowedModels) ? allowedModels.join(',') : '' }}
      </span>
      <span data-testid="gateway-model-mappings-prop">
        {{ Array.isArray(modelMappings) ? JSON.stringify(modelMappings) : '' }}
      </span>
      <span data-testid="gateway-test-provider-prop">{{ gatewayTestProvider }}</span>
      <span data-testid="gateway-test-model-prop">{{ gatewayTestModelId }}</span>
      <button
        type="button"
        data-testid="set-gateway-selection"
        @click="
          $emit('update:allowedModels', ['gpt-5.4']);
          $emit('update:modelMappings', [{ from: 'friendly-gateway-model', to: 'gpt-5.4' }])
        "
      >
        set gateway selection
      </button>
      <button
        type="button"
        data-testid="set-gateway-test-defaults"
        @click="
          $emit('update:gatewayTestProvider', 'openai');
          $emit('update:gateway-test-provider', 'openai');
          $emit('update:gatewayTestModelId', 'gpt-5.4');
          $emit('update:gateway-test-model-id', 'gpt-5.4')
        "
      >
        set defaults
      </button>
      <button
        type="button"
        data-testid="clear-gateway-selection-and-defaults"
        @click="
          $emit('update:allowedModels', []);
          $emit('update:modelMappings', []);
          $emit('update:gatewayTestProvider', '');
          $emit('update:gateway-test-provider', '');
          $emit('update:gatewayTestModelId', '');
          $emit('update:gateway-test-model-id', '')
        "
      >
        clear defaults
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

const AccountBaiduDocumentAICredentialsEditorStub = defineComponent({
  name: 'AccountBaiduDocumentAICredentialsEditor',
  props: {
    asyncBaseUrl: {
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
    <div>
      <span data-testid="baidu-async-base-url-prop">{{ asyncBaseUrl }}</span>
      <span data-testid="baidu-direct-api-urls-prop">{{ directApiUrlsText }}</span>
      <button
        type="button"
        data-testid="set-baidu-async-token"
        @click="$emit('update:async-bearer-token', 'new-async-token')"
      >
        set baidu async token
      </button>
    </div>
  `
})

function buildAccount() {
  return {
    id: 1,
    name: 'OpenAI Key',
    notes: '',
    platform: 'openai',
    type: 'apikey',
    credentials: {
      api_key: 'sk-test',
      base_url: 'https://api.openai.com',
      model_mapping: {
        'gpt-5.2': 'gpt-5.2'
      }
    },
    extra: {},
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildGrokSsoAccount() {
  return {
    id: 2,
    name: 'Grok SSO',
    notes: '',
    platform: 'grok',
    type: 'sso',
    credentials: {
      sso_token: 'Bearer old-token',
      model_mapping: {
        'grok-3-beta': 'grok-3-beta'
      }
    },
    extra: {
      grok_tier: 'super'
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildGrokAPIKeyAccount() {
  return {
    id: 6,
    name: 'Grok API Key',
    notes: '',
    platform: 'grok',
    type: 'apikey',
    credentials: {
      api_key: 'xai-test',
      base_url: 'https://api.x.ai',
      model_mapping: {
        'grok-4': 'grok-4'
      }
    },
    extra: {
      grok_tier: 'heavy',
      grok_capabilities: ['grok-4']
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildOpenAIOAuthAccount() {
  return {
    id: 5,
    name: 'OpenAI OAuth',
    notes: '',
    platform: 'openai',
    type: 'oauth',
    credentials: {
      access_token: 'access-token',
      model_mapping: {
        'friendly-gpt': 'gpt-5.4'
      }
    },
    extra: {
      model_probe_snapshot: {
        models: ['gpt-5.4', 'gpt-4.1-mini'],
        updated_at: '2026-04-01T10:00:00Z',
        source: 'manual_probe',
        probe_source: 'upstream'
      }
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildVertexExpressAccount() {
  return {
    id: 3,
    name: 'Vertex Express',
    notes: '',
    platform: 'gemini',
    type: 'apikey',
    credentials: {
      api_key: 'vertex-key',
      base_url: 'https://aiplatform.googleapis.com',
      gemini_api_variant: 'vertex_express'
    },
    extra: {
      quota_limit: 120,
      quota_daily_limit: 12,
      quota_weekly_limit: 50,
      quota_daily_reset_mode: 'rolling',
      quota_weekly_reset_mode: 'rolling'
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildAnthropicWhitelistAccountWithSelectedModelIDs() {
  return {
    id: 9,
    name: 'Anthropic Key',
    notes: '',
    platform: 'anthropic',
    type: 'apikey',
    credentials: {
      api_key: 'sk-anthropic',
      base_url: 'https://api.anthropic.com'
    },
    extra: {
      model_scope_v2: {
        supported_providers: ['anthropic'],
        supported_models_by_provider: {
          anthropic: ['claude-sonnet-4.5']
        },
        selected_model_ids: ['claude-sonnet-4-5-20250929', 'claude-sonnet-4.5'],
        advanced_provider_override: false,
        manual_mapping_rows: [],
        manual_mappings: {}
      }
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildProtocolGatewayGeminiAccount() {
  return {
    id: 4,
    name: 'Gemini Gateway',
    notes: '',
    platform: 'protocol_gateway',
    gateway_protocol: 'mixed',
    gateway_batch_enabled: true,
    type: 'apikey',
    credentials: {
      api_key: 'gateway-key',
      base_url: 'https://gateway.example.com',
      model_mapping: {
        'friendly-gateway-model': 'gpt-5.4'
      }
    },
    extra: {
      gateway_protocol: 'mixed',
      gateway_accepted_protocols: ['gemini'],
      gateway_batch_enabled: true,
      model_scope_v2: {
        supported_providers: ['gemini'],
        supported_models_by_provider: {
          gemini: ['gpt-5.4']
        },
        advanced_provider_override: false,
        manual_mapping_rows: [
          {
            from: 'friendly-gateway-model',
            to: 'gpt-5.4'
          }
        ],
        manual_mappings: {
          'friendly-gateway-model': 'gpt-5.4'
        }
      }
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildProtocolGatewayOpenAIAccount() {
  return {
    id: 7,
    name: 'OpenAI Gateway',
    notes: '',
    platform: 'protocol_gateway',
    gateway_protocol: 'openai',
    type: 'apikey',
    credentials: {
      api_key: 'gateway-key',
      base_url: 'https://gateway.example.com'
    },
    extra: {
      gateway_protocol: 'openai',
      gateway_accepted_protocols: ['openai'],
      gateway_openai_request_format: '/v1/chat/completions'
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function buildBaiduDocumentAIAccount(platform: string = 'baidu_document_ai') {
  return {
    id: 8,
    name: 'Baidu OCR',
    notes: '',
    platform,
    type: 'apikey',
    credentials: {
      async_bearer_token: 'existing-async-token',
      async_base_url: 'https://aistudio.baidu.com/async',
      direct_token: 'existing-direct-token',
      direct_api_urls: {
        'pp-ocrv5-server': 'https://direct.baidu.com/ocr'
      }
    },
    extra: {},
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  } as any
}

function mountModal(account = buildAccount()) {
  return mount(EditAccountModal, {
    props: {
      show: true,
      loading: false,
      account,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        AccountApiKeyBasicSettingsEditor: AccountApiKeyBasicSettingsEditorStub,
        AccountApiKeyModelProbeEditor: AccountApiKeyModelProbeEditorStub,
        AccountBaiduDocumentAICredentialsEditor: AccountBaiduDocumentAICredentialsEditorStub,
        AccountProtocolGatewayModelProbeEditor: AccountProtocolGatewayModelProbeEditorStub,
        AccountProtocolGatewayOpenAIRequestFormatEditor: AccountProtocolGatewayOpenAIRequestFormatEditorStub,
        AccountProtocolGatewayBatchEditor: true,
        AccountGeminiVertexCredentialsEditor: true,
        AccountModelScopeEditor: true,
        AccountRuntimeSettingsEditor: true,
        AccountGatewaySettingsEditor: true,
        AccountGroupSettingsEditor: true,
        AccountAutoPauseToggle: true,
        QuotaLimitCard: true,
        Select: SelectStub,
        Icon: true,
        ProxySelector: true,
        GroupSelector: true,
      }
    }
  })
}

describe('EditAccountModal', () => {
  it('uses the dedicated account-wide dialog width', () => {
    const wrapper = mountModal()
    expect(wrapper.find('[data-width="account-wide"]').exists()).toBe(true)
  })

  it('reopening the same account rehydrates the OpenAI whitelist from props', async () => {
    const account = buildAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('gpt-5.2')

    await wrapper.get('[data-testid="rewrite-to-snapshot"]').trigger('click')
    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('gpt-5.2-2025-12-11')

    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('gpt-5.2')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toEqual({
      'gpt-5.2': 'gpt-5.2'
    })
  })

  it('rehydrates whitelist selected_model_ids before canonical ids and keeps them on submit', async () => {
    const account = buildAnthropicWhitelistAccountWithSelectedModelIDs()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe(
      'claude-sonnet-4-5-20250929,claude-sonnet-4.5'
    )

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.model_scope_v2).toMatchObject({
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
    })
  })

  it('submits Grok SSO token replacement and grok_tier updates', async () => {
    const account = buildGrokSsoAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    const tokenInput = wrapper.find('textarea[placeholder="admin.accounts.leaveEmptyToKeep"]')
    expect(tokenInput.exists()).toBe(true)
    await tokenInput.setValue('Bearer new-token')

    const tierSelect = wrapper.find('select.input')
    expect(tierSelect.exists()).toBe(true)
    await tierSelect.setValue('heavy')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.sso_token).toBe('Bearer new-token')
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.grok_tier).toBe('heavy')
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toEqual({
      'grok-3-beta': 'grok-3-beta'
    })
  })

  it('renders the unified model probe editor for Grok SSO accounts', () => {
    const wrapper = mountModal(buildGrokSsoAccount())

    expect(wrapper.findComponent({ name: 'AccountApiKeyModelProbeEditor' }).exists()).toBe(true)
  })

  it('keeps Grok API key accounts on the unified editor and removes legacy tier fields on submit', async () => {
    const account = buildGrokAPIKeyAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.text()).not.toContain('admin.accounts.grokTier')
    expect(wrapper.find('textarea[placeholder="admin.accounts.leaveEmptyToKeep"]').exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'AccountApiKeyModelProbeEditor' }).exists()).toBe(true)

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.grok_tier).toBeUndefined()
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.grok_capabilities).toBeUndefined()
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toEqual({
      'grok-4': 'grok-4'
    })
  })

  it('renders the unified model probe editor for OpenAI OAuth accounts and rebuilds snapshot extra from the current model scope on submit', async () => {
    const account = buildOpenAIOAuthAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.findComponent({ name: 'AccountApiKeyModelProbeEditor' }).exists()).toBe(true)

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.model_probe_snapshot).toMatchObject({
      models: ['gpt-5.4'],
      source: 'model_scope_preview',
      probe_source: 'model_scope_preview'
    })
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.model_probe_snapshot?.updated_at).toMatch(/^\d{4}-\d{2}-\d{2}T/)
  })

  it('keeps upstream quota fields when editing a vertex express account', async () => {
    const account = buildVertexExpressAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toMatchObject({
      quota_limit: 120,
      quota_daily_limit: 12,
      quota_weekly_limit: 50,
      quota_daily_reset_mode: 'rolling',
      quota_weekly_reset_mode: 'rolling'
    })
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.gemini_api_variant).toBe('vertex_express')
  })

  it('keeps gateway_batch_enabled when editing a gemini protocol gateway account', async () => {
    const account = buildProtocolGatewayGeminiAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.findComponent({ name: 'AccountProtocolGatewayModelProbeEditor' }).exists()).toBe(true)
    expect(wrapper.get('[data-testid="show-gemini-tier-prop"]').text()).toBe('false')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.gateway_batch_enabled).toBe(true)
    expect(updateAccountMock.mock.calls[0]?.[1]?.gateway_protocol).toBe('mixed')
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.tier_id).toBeUndefined()
  })

  it('rehydrates protocol gateway scope rows without fabricating identity mappings', async () => {
    const account = buildProtocolGatewayGeminiAccount()

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="gateway-allowed-models-prop"]').text()).toBe('gpt-5.4')
    expect(wrapper.get('[data-testid="gateway-model-mappings-prop"]').text()).toContain('friendly-gateway-model')
  })

  it('rehydrates protocol gateway whitelist-only scope without fabricating manual mapping rows', async () => {
    const account = buildProtocolGatewayGeminiAccount()
    delete account.credentials.model_mapping
    account.extra.model_scope_v2 = {
      supported_providers: ['gemini'],
      supported_models_by_provider: {
        gemini: ['gemini-2.5-pro']
      },
      advanced_provider_override: false,
      manual_mapping_rows: [],
      manual_mappings: {}
    }

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="gateway-allowed-models-prop"]').text()).toBe('gemini-2.5-pro')
    expect(wrapper.get('[data-testid="gateway-model-mappings-prop"]').text()).toBe('[]')
  })

  it('persists protocol gateway selected targets and explicit aliases on edit submit', async () => {
    const account = buildProtocolGatewayGeminiAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    await wrapper.get('[data-testid="set-gateway-selection"]').trigger('click')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toEqual({
      'friendly-gateway-model': 'gpt-5.4'
    })
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.model_scope_v2).toMatchObject({
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

  it('rehydrates and persists gateway test provider/model defaults for protocol gateway accounts', async () => {
    const account = buildProtocolGatewayGeminiAccount()
    account.extra.gateway_test_provider = 'anthropic'
    account.extra.gateway_test_model_id = 'claude-sonnet-4.5'

    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="gateway-test-provider-prop"]').text()).toBe('anthropic')
    expect(wrapper.get('[data-testid="gateway-test-model-prop"]').text()).toBe('claude-sonnet-4.5')

    await wrapper.get('[data-testid="set-gateway-test-defaults"]').trigger('click')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.gateway_test_provider).toBe('openai')
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.gateway_test_model_id).toBe('gpt-5.4')
  })

  it('drops gateway test defaults from edit payloads after the selected gateway models are cleared', async () => {
    const account = buildProtocolGatewayGeminiAccount()
    account.extra.gateway_test_provider = 'anthropic'
    account.extra.gateway_test_model_id = 'claude-sonnet-4.5'

    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="gateway-test-provider-prop"]').text()).toBe('anthropic')
    expect(wrapper.get('[data-testid="gateway-test-model-prop"]').text()).toBe('claude-sonnet-4.5')

    await wrapper.get('[data-testid="clear-gateway-selection-and-defaults"]').trigger('click')
    expect(wrapper.get('[data-testid="gateway-test-provider-prop"]').text()).toBe('')
    expect(wrapper.get('[data-testid="gateway-test-model-prop"]').text()).toBe('')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toBeUndefined()
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.model_scope_v2).toBeUndefined()
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.gateway_test_provider).toBeUndefined()
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.gateway_test_model_id).toBeUndefined()
  })

  it('rehydrates and persists the protocol gateway OpenAI request format', async () => {
    const account = buildProtocolGatewayOpenAIAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="gateway-openai-request-format-prop"]').text()).toBe('/v1/chat/completions')

    await wrapper.get('[data-testid="set-gateway-openai-request-format"]').trigger('click')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.gateway_openai_request_format).toBe('/v1/responses')
  })

  it('preserves a custom protocol gateway base_url on reopen and only resets it after a manual protocol switch', async () => {
    const wrapper = mountModal(buildProtocolGatewayOpenAIAccount())

    expect((wrapper.get('[data-testid="account-base-url-input"]').element as HTMLInputElement).value).toBe('https://gateway.example.com')

    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })

    expect((wrapper.get('[data-testid="account-base-url-input"]').element as HTMLInputElement).value).toBe('https://gateway.example.com')

    const selects = wrapper.findAll('[data-testid="select-stub"]')
    await selects[0].setValue('gemini')

    expect((wrapper.get('[data-testid="account-base-url-input"]').element as HTMLInputElement).value).toBe(
      resolveAccountApiKeyDefaultBaseUrl('protocol_gateway', 'gemini')
    )
  })

  it('rehydrates baidu document ai editor fields and keeps existing tokens unless replaced', async () => {
    const account = buildBaiduDocumentAIAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="baidu-async-base-url-prop"]').text()).toBe('https://aistudio.baidu.com/async')
    expect(wrapper.get('[data-testid="baidu-direct-api-urls-prop"]').text()).toContain('pp-ocrv5-server')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials).toMatchObject({
      async_bearer_token: 'existing-async-token',
      async_base_url: 'https://aistudio.baidu.com/async',
      direct_token: 'existing-direct-token',
      direct_api_urls: {
        'pp-ocrv5-server': 'https://direct.baidu.com/ocr'
      }
    })

    updateAccountMock.mockReset()
    await wrapper.get('[data-testid="set-baidu-async-token"]').trigger('click')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.async_bearer_token).toBe('new-async-token')
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.direct_token).toBe('existing-direct-token')
  })

  it('renders the baidu document ai editor for legacy baidu platform values', async () => {
    const wrapper = mountModal(buildBaiduDocumentAIAccount('baidu'))
    expect(wrapper.find('[data-testid=\"baidu-async-base-url-prop\"]').exists()).toBe(true)
  })

  it('keeps mapping rows synchronized with probe selection changes for api key accounts', async () => {
    const account = buildAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('gpt-5.2')
    expect(wrapper.get('[data-testid="model-mappings-prop"]').text()).toBe('[]')

    await wrapper.get('[data-testid="probe-select-models"]').trigger('click')

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('friendly-gpt')
    expect(wrapper.get('[data-testid="model-mappings-prop"]').text()).toContain('friendly-gpt')
    expect(wrapper.get('[data-testid="actual-model-locked-prop"]').text()).toBe('true')

    await wrapper.get('[data-testid="probe-clear-models"]').trigger('click')

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('')
    expect(wrapper.get('[data-testid="model-mappings-prop"]').text()).toBe('[]')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toBeUndefined()
  })
})
