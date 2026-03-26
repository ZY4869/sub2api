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
        probedModels: []
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const probeButton = wrapper.find('button')
    await probeButton.trigger('click')
    await flushPromises()

    expect(probeProtocolGatewayModels).toHaveBeenCalledWith({
      gateway_protocol: 'openai',
      base_url: 'https://gateway.example.com',
      api_key: 'sk-test',
      proxy_id: 12
    })
    expect(wrapper.emitted('update:allowedModels')?.[0]?.[0]).toEqual(['gpt-4.1', 'custom-upstream-model'])
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
        probedModels: []
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
})
