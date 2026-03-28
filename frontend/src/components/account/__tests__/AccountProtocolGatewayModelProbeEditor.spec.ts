import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountProtocolGatewayModelProbeEditor from '../AccountProtocolGatewayModelProbeEditor.vue'

const { probeProtocolGatewayModels, showError, showWarning } = vi.hoisted(() => ({
  probeProtocolGatewayModels: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      probeProtocolGatewayModels
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showWarning
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) =>
        params?.count ? `${key}:${params.count}` : key
    })
  }
})

describe('AccountProtocolGatewayModelProbeEditor', () => {
  beforeEach(() => {
    probeProtocolGatewayModels.mockReset()
    probeProtocolGatewayModels.mockResolvedValue({
      probe_source: 'upstream',
      probe_notice: '2 models detected',
      models: [
        {
          id: 'gpt-4.1',
          display_name: 'GPT-4.1',
          registry_state: 'existing',
          registry_model_id: 'gpt-4.1'
        },
        {
          id: 'custom-upstream-model',
          display_name: 'Custom Upstream Model',
          registry_state: 'missing'
        }
      ]
    })
    showError.mockReset()
    showWarning.mockReset()
  })

  it('probes upstream models and selects all returned models by default', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'https://gateway.example.com',
        apiKey: 'sk-test',
        proxyId: 12,
        allowedModels: [],
        modelMappings: [],
        probedModels: [],
        acceptedProtocols: ['openai'],
        clientProfiles: [],
        clientRoutes: []
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const probeButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.accounts.protocolGateway.probeAction')
    expect(probeButton).toBeTruthy()
    await probeButton!.trigger('click')
    await flushPromises()

    expect(probeProtocolGatewayModels).toHaveBeenCalledWith({
      gateway_protocol: 'openai',
      accepted_protocols: ['openai'],
      base_url: 'https://gateway.example.com',
      api_key: 'sk-test',
      proxy_id: 12
    })
    expect(wrapper.emitted('update:allowedModels')?.[0]?.[0]).toEqual(['gpt-4.1', 'custom-upstream-model'])
    expect(wrapper.emitted('update:modelMappings')?.[0]?.[0]).toEqual([
      { from: 'gpt-4.1', to: 'gpt-4.1' },
      { from: 'custom-upstream-model', to: 'custom-upstream-model' }
    ])
    expect(wrapper.emitted('update:probedModels')?.[0]?.[0]).toHaveLength(2)
    expect(wrapper.text()).toContain('admin.accounts.protocolGateway.registryExisting')
    expect(wrapper.text()).toContain('admin.accounts.protocolGateway.registryMissing')
  })

  it('disables probing when api key is missing', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'anthropic',
        baseUrl: '',
        apiKey: '',
        allowedModels: [],
        modelMappings: [],
        probedModels: [],
        acceptedProtocols: ['anthropic'],
        clientProfiles: [],
        clientRoutes: []
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const probeButton = wrapper.find('button')
    expect((probeButton.element as HTMLButtonElement).disabled).toBe(true)
    await probeButton.trigger('click')

    expect(probeProtocolGatewayModels).not.toHaveBeenCalled()
    expect(showWarning).not.toHaveBeenCalled()
  })

  it('groups mixed probe results by source protocol and applies profile batches by compatibility', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'mixed',
        baseUrl: 'https://gateway.example.com',
        apiKey: 'sk-test',
        allowedModels: ['gpt-4.1', 'gemini-2.5-pro'],
        modelMappings: [
          { from: 'gpt-4.1', to: 'gpt-4.1' },
          { from: 'gemini-2.5-pro', to: 'gemini-2.5-pro' }
        ],
        probedModels: [
          {
            id: 'gpt-4.1',
            display_name: 'GPT-4.1',
            registry_state: 'existing',
            registry_model_id: 'gpt-4.1',
            source_protocol: 'openai'
          },
          {
            id: 'gemini-2.5-pro',
            display_name: 'Gemini 2.5 Pro',
            registry_state: 'missing',
            source_protocol: 'gemini'
          }
        ],
        acceptedProtocols: ['openai', 'gemini'],
        clientProfiles: ['codex', 'gemini_cli'],
        clientRoutes: [
          {
            protocol: 'openai',
            match_type: 'exact',
            match_value: 'gpt-4.1',
            client_profile: 'codex'
          }
        ]
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('OpenAI')
    expect(wrapper.text()).toContain('Gemini')
    expect(wrapper.text()).toContain('/v1/chat/completions')
    expect(wrapper.text()).toContain('/v1beta/models/{model}:generateContent')
    expect(wrapper.text()).toContain('admin.accounts.protocolGateway.clientProfileCodex')

    const applyAllGeminiCliButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.accounts.protocolGateway.applyAllGeminiCli')
    expect(applyAllGeminiCliButton).toBeTruthy()

    await applyAllGeminiCliButton!.trigger('click')

    expect(wrapper.emitted('update:clientRoutes')?.[0]?.[0]).toEqual([
      {
        protocol: 'gemini',
        match_type: 'exact',
        match_value: 'gemini-2.5-pro',
        client_profile: 'gemini_cli'
      }
    ])
  })

  it('supports editing chinese request model aliases and removes mappings when deselected', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'https://gateway.example.com',
        apiKey: 'sk-test',
        allowedModels: ['gpt-4.1'],
        modelMappings: [{ from: 'gpt-4.1', to: 'gpt-4.1' }],
        probedModels: [
          {
            id: 'gpt-4.1',
            display_name: 'GPT-4.1',
            registry_state: 'existing',
            registry_model_id: 'gpt-4.1'
          }
        ],
        acceptedProtocols: ['openai'],
        clientProfiles: [],
        clientRoutes: []
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const aliasInput = wrapper.find('input[placeholder="gpt-4.1"]')
    expect(aliasInput.exists()).toBe(true)

    await aliasInput.setValue('中文别名')
    await aliasInput.trigger('blur')

    const mappingEvents = wrapper.emitted('update:modelMappings') || []
    expect(mappingEvents.at(-1)?.[0]).toEqual([{ from: '中文别名', to: 'gpt-4.1' }])

    const modelCard = wrapper.find('button[title="gpt-4.1"]')
    await modelCard.trigger('click')

    expect(wrapper.emitted('update:allowedModels')?.at(-1)?.[0]).toEqual([])
    expect(wrapper.emitted('update:modelMappings')?.at(-1)?.[0]).toEqual([])
  })
})
