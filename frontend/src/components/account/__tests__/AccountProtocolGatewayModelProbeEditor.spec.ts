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

const findButtonByText = (wrapper: ReturnType<typeof mount>, text: string) =>
  wrapper.findAll('button').find((button) => button.text() === text)

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

  it('probes upstream models without auto-selecting them, then selects all on demand', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'https://gateway.example.com',
        apiKey: 'sk-test',
        proxyId: 12,
        allowedModels: [],
        modelMappings: [],
        probedModels: [],
        manualModels: [],
        resolvedUpstream: null,
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

    const probeButton = findButtonByText(wrapper, 'admin.accounts.protocolGateway.probeAction')
    expect(probeButton).toBeTruthy()
    await probeButton!.trigger('click')
    await flushPromises()

    expect(probeProtocolGatewayModels).toHaveBeenCalledWith({
      gateway_protocol: 'openai',
      accepted_protocols: ['openai'],
      base_url: 'https://gateway.example.com',
      api_key: 'sk-test',
      manual_models: [],
      proxy_id: 12
    })
    expect(wrapper.emitted('update:allowedModels')?.at(-1)?.[0]).toEqual([])
    expect(wrapper.emitted('update:modelMappings')?.at(-1)?.[0]).toEqual([])
    expect(wrapper.emitted('update:probedModels')?.[0]?.[0]).toHaveLength(2)
    expect(wrapper.text()).toContain('admin.accounts.protocolGateway.registryExisting')
    expect(wrapper.text()).toContain('admin.accounts.protocolGateway.registryMissing')

    const selectAllButton = findButtonByText(
      wrapper,
      'admin.accounts.protocolGateway.selectAllCurrentResults'
    )
    expect(selectAllButton).toBeTruthy()
    await selectAllButton!.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:allowedModels')?.at(-1)?.[0]).toEqual(['gpt-4.1', 'custom-upstream-model'])
    expect(wrapper.emitted('update:modelMappings')?.at(-1)?.[0]).toEqual([
      { from: 'gpt-4.1', to: 'gpt-4.1' },
      { from: 'custom-upstream-model', to: 'custom-upstream-model' }
    ])
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
        manualModels: [],
        resolvedUpstream: null,
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
        manualModels: [],
        resolvedUpstream: null,
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

    const applyAllGeminiCliButton = findButtonByText(
      wrapper,
      'admin.accounts.protocolGateway.applyAllGeminiCli'
    )
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

  it('supports editing request model aliases, keeps empty drafts, and removes mappings when deselected', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'https://gateway.example.com',
        apiKey: 'sk-test',
        allowedModels: ['gpt-4.1'],
        modelMappings: [{ from: 'gpt-4.1', to: 'gpt-4.1' }],
        manualModels: [],
        resolvedUpstream: null,
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
    let mappingEvents = wrapper.emitted('update:modelMappings') || []
    expect(mappingEvents.at(-1)?.[0]).toEqual([{ from: '中文别名', to: 'gpt-4.1' }])

    await aliasInput.setValue('')
    mappingEvents = wrapper.emitted('update:modelMappings') || []
    expect(mappingEvents.at(-1)?.[0]).toEqual([{ from: '', to: 'gpt-4.1' }])

    const modelCard = wrapper.find('button[title="gpt-4.1"]')
    await modelCard.trigger('click')

    expect(wrapper.emitted('update:allowedModels')?.at(-1)?.[0]).toEqual([])
    expect(wrapper.emitted('update:modelMappings')?.at(-1)?.[0]).toEqual([])
  })

  it('renders long model details without truncating the content', () => {
    const longDisplayName = 'A Very Long Upstream Model Name That Should Stay Visible In Full'
    const longModelId = 'provider/collections/super-long-model-id-that-should-wrap-instead-of-truncating'
    const longRegistryId = 'registry/super-long-model-id-that-should-also-remain-visible'

    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'https://gateway.example.com',
        apiKey: 'sk-test',
        allowedModels: [longModelId],
        modelMappings: [{ from: longModelId, to: longModelId }],
        manualModels: [],
        resolvedUpstream: null,
        probedModels: [
          {
            id: longModelId,
            display_name: longDisplayName,
            registry_state: 'existing',
            registry_model_id: longRegistryId
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

    expect(wrapper.text()).toContain(longDisplayName)
    expect(wrapper.text()).toContain(longModelId)
    expect(wrapper.text()).toContain(longRegistryId)
    expect(wrapper.find(`button[title="${longModelId}"]`).exists()).toBe(true)
  })

  it('clears selected models and exact-match routes via the batch clear action', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'https://gateway.example.com',
        apiKey: 'sk-test',
        allowedModels: ['gpt-4.1', 'custom-upstream-model'],
        modelMappings: [
          { from: 'gpt-4.1', to: 'gpt-4.1' },
          { from: 'custom-upstream-model', to: 'custom-upstream-model' }
        ],
        manualModels: [],
        resolvedUpstream: null,
        probedModels: [
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
        ],
        acceptedProtocols: ['openai'],
        clientProfiles: ['codex'],
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

    const clearSelectionButton = findButtonByText(
      wrapper,
      'admin.accounts.protocolGateway.clearSelection'
    )
    expect(clearSelectionButton).toBeTruthy()

    await clearSelectionButton!.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:allowedModels')?.at(-1)?.[0]).toEqual([])
    expect(wrapper.emitted('update:modelMappings')?.at(-1)?.[0]).toEqual([])
    expect(wrapper.emitted('update:clientRoutes')?.at(-1)?.[0]).toEqual([])
  })

  it('blocks probing on loopback base urls and shows docker guidance', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'http://127.0.0.1:8082',
        apiKey: 'sk-test',
        allowedModels: [],
        modelMappings: [],
        probedModels: [],
        manualModels: [],
        resolvedUpstream: null,
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

    const probeButton = findButtonByText(wrapper, 'admin.accounts.protocolGateway.probeAction')
    expect(probeButton).toBeTruthy()
    await probeButton!.trigger('click')

    expect(probeProtocolGatewayModels).not.toHaveBeenCalled()
    expect(showWarning).toHaveBeenCalledWith(
      'admin.accounts.protocolGateway.baseUrlLoopbackWarning'
    )
  })

  it('blocks probing on invalid base urls before sending any request', async () => {
    const wrapper = mount(AccountProtocolGatewayModelProbeEditor, {
      props: {
        gatewayProtocol: 'openai',
        baseUrl: 'http://127.0.0.1.8082',
        apiKey: 'sk-test',
        allowedModels: [],
        modelMappings: [],
        probedModels: [],
        manualModels: [],
        resolvedUpstream: null,
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

    const probeButton = findButtonByText(wrapper, 'admin.accounts.protocolGateway.probeAction')
    expect(probeButton).toBeTruthy()
    await probeButton!.trigger('click')

    expect(probeProtocolGatewayModels).not.toHaveBeenCalled()
    expect(showWarning).toHaveBeenCalledWith(
      'admin.accounts.protocolGateway.baseUrlInvalidWarning'
    )
  })
})
