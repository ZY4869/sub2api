import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'

const { updateAccountMock, checkMixedChannelRiskMock } = vi.hoisted(() => ({
  updateAccountMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn()
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
  ensureModelRegistryFresh: vi.fn().mockResolvedValue({
    etag: 'test-etag',
    updated_at: '2026-04-08T00:00:00Z',
    models: [],
    presets: []
  }),
  getModelRegistrySnapshot: vi.fn(() => ({
    etag: 'test-etag',
    updated_at: '2026-04-08T00:00:00Z',
    models: [],
    presets: []
  })),
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

const AccountApiKeyBasicSettingsEditorStub = defineComponent({
  name: 'AccountApiKeyBasicSettingsEditor',
  props: {
    allowedModels: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:allowedModels'],
  template: `
    <div>
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
  emits: ['update:gatewayTestProvider', 'update:gatewayTestModelId', 'update:gateway-test-provider', 'update:gateway-test-model-id'],
  template: `
    <div>
      <span data-testid="gateway-test-provider-prop">{{ gatewayTestProvider }}</span>
      <span data-testid="gateway-test-model-prop">{{ gatewayTestModelId }}</span>
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
      base_url: 'https://gateway.example.com'
    },
    extra: {
      gateway_protocol: 'mixed',
      gateway_accepted_protocols: ['gemini'],
      gateway_batch_enabled: true
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
        AccountApiKeyModelProbeEditor: true,
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
        Select: true,
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

  it('renders the unified model probe editor for OpenAI OAuth accounts and keeps snapshot extra on submit', async () => {
    const account = buildOpenAIOAuthAccount()
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.findComponent({ name: 'AccountApiKeyModelProbeEditor' }).exists()).toBe(true)

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.model_probe_snapshot).toEqual({
      models: ['gpt-5.4', 'gpt-4.1-mini'],
      updated_at: '2026-04-01T10:00:00Z',
      source: 'manual_probe',
      probe_source: 'upstream'
    })
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

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.gateway_batch_enabled).toBe(true)
    expect(updateAccountMock.mock.calls[0]?.[1]?.gateway_protocol).toBe('mixed')
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
})
