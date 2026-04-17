import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'),
  'utf-8'
)

const { createMock, checkMixedChannelRiskMock, invalidateModelRegistryMock, invalidateInventoryMock } = vi.hoisted(() => ({
  createMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  invalidateModelRegistryMock: vi.fn(),
  invalidateInventoryMock: vi.fn()
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
    <button
      type="button"
      data-testid="select-protocol-gateway"
      @click="$emit('update:platform', 'protocol_gateway')"
    >
      select protocol gateway
    </button>
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
        data-testid="set-gateway-gemini"
        @click="
          $emit('update:account-category', 'apikey');
          $emit('update:gateway-protocol', 'gemini')
        "
      >
        set gemini gateway
      </button>
    </div>
  `
})

const AccountApiKeyBasicSettingsEditorStub = defineComponent({
  name: 'AccountApiKeyBasicSettingsEditor',
  props: {
    showGeminiTier: {
      type: Boolean,
      default: false
    }
  },
  emits: ['update:api-key', 'update:base-url'],
  template: `
    <div>
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
  emits: ['update:gateway-test-provider', 'update:gateway-test-model-id'],
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

function mountModal() {
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
        AccountProtocolGatewayModelProbeEditor: AccountProtocolGatewayModelProbeEditorStub,
        AccountProtocolGatewayOpenAIRequestFormatEditor: AccountProtocolGatewayOpenAIRequestFormatEditorStub,
        AccountApiKeyModelProbeEditor: true,
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
        AccountModelScopeEditor: true,
        AccountPoolModeEditor: true,
        AccountProtocolGatewayClaudeMimicEditor: true,
        AccountProtocolGatewayBatchEditor: true,
        AccountQuotaControlEditor: true,
        AccountRuntimeSettingsEditor: true,
        AccountTempUnschedRulesEditor: true,
        QuotaLimitCard: true
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
    expect(source).toContain(":skip-model-scope-editor=\"form.platform === 'protocol_gateway'\"")
    expect(source).toContain(
      ":show-auto-import=\"form.platform !== 'protocol_gateway' && form.platform !== 'baidu_document_ai'\""
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
})
