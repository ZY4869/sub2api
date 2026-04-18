import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountApiKeyModelProbeEditor from '../AccountApiKeyModelProbeEditor.vue'

const { probeModels, showError } = vi.hoisted(() => ({
  probeModels: vi.fn(),
  showError: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        if (key === 'admin.accounts.modelImportErrorHints.openai_api_model_read') {
          return 'OpenAI model list permission hint'
        }
        return key
      }
    })
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      probeModels
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError
  })
}))

const createWrapper = (overrides: Record<string, unknown> = {}) =>
  mount(AccountApiKeyModelProbeEditor, {
    props: {
      platform: 'gemini',
      accountType: 'apikey',
      credentials: {
        gemini_api_variant: 'vertex_express',
        api_key: 'vertex-express-key'
      },
      probeReady: true,
      allowedModels: [],
      modelMappings: [],
      probedModels: [],
      manualModels: [],
      resolvedUpstream: null,
      ...overrides
    },
    global: {
      stubs: {
        Icon: {
          template: '<span />'
        },
        ModelIcon: {
          template: '<span data-testid="model-icon" />'
        },
        ModelPlatformIcon: {
          template: '<span data-testid="provider-icon" />'
        }
      }
    }
  })

const findProbeButton = (wrapper: ReturnType<typeof createWrapper>) =>
  wrapper.findAll('button').find((button) =>
    button.text().includes('admin.accounts.apiKeyProbe.action')
  )

const findButtonByText = (wrapper: ReturnType<typeof createWrapper>, text: string) =>
  wrapper.findAll('button').find((button) => button.text().includes(text))

describe('AccountApiKeyModelProbeEditor', () => {
  beforeEach(() => {
    probeModels.mockReset()
    showError.mockReset()
  })

  it('keeps probe results unselected until callable models are explicitly selected', async () => {
    probeModels.mockResolvedValue({
      probe_source: 'vertex_express_catalog',
      probe_notice: '',
      models: [
        {
          id: 'gemini-2.0-flash',
          display_name: 'Gemini 2.0 Flash',
          provider: 'gemini',
          provider_label: 'Google-Gemini',
          registry_state: 'existing',
          upstream_source: 'official',
          availability: 'callable'
        },
        {
          id: 'gemini-3.1-pro-preview',
          display_name: 'Gemini 3.1 Pro Preview',
          registry_state: 'missing',
          upstream_source: 'verified_extra',
          availability: 'uncallable',
          availability_reason: 'status 403 PERMISSION_DENIED'
        }
      ]
    })

    const wrapper = createWrapper()
    const probeButton = findProbeButton(wrapper)

    expect(probeButton).toBeTruthy()
    await probeButton?.trigger('click')
    await flushPromises()

    expect(probeModels).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.sourceOfficial')
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.sourceVerifiedExtra')
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.availabilityUncallable')
    expect(wrapper.get('[data-testid="probe-model-display-name"]').text()).toBe('Gemini 2.0 Flash')
    expect(wrapper.get('[data-testid="probe-model-id"]').text()).toContain('gemini-2.0-flash')
    expect(wrapper.findAll('[data-testid="probe-model-icon"]')).not.toHaveLength(0)
    expect(wrapper.findAll('[data-testid="probe-provider-icon"]')).not.toHaveLength(0)

    const allowedModelsUpdates = wrapper.emitted('update:allowedModels') || []
    expect(allowedModelsUpdates.at(-1)).toEqual([[]])

    const modelMappingsUpdates = wrapper.emitted('update:modelMappings') || []
    expect(modelMappingsUpdates.at(-1)).toEqual([[]])
    expect(wrapper.find('input[placeholder="gemini-2.0-flash"]').exists()).toBe(false)

    const selectCallableButton = findButtonByText(
      wrapper,
      'admin.accounts.apiKeyProbe.selectCallableModels'
    )
    expect(selectCallableButton).toBeTruthy()

    await selectCallableButton?.trigger('click')
    await flushPromises()

    expect(allowedModelsUpdates.at(-1)).toEqual([['gemini-2.0-flash']])
    expect(modelMappingsUpdates.at(-1)).toEqual([
      [{ from: 'Vertex-gemini-2.0-flash', to: 'gemini-2.0-flash' }]
    ])
  })

  it('keeps only previously selected models after re-probing and preserves custom aliases from existing mappings', async () => {
    probeModels
      .mockResolvedValueOnce({
        probe_source: 'vertex_express_catalog',
        probe_notice: '',
        models: [
          {
            id: 'gemini-2.0-flash',
            display_name: 'Gemini 2.0 Flash',
            registry_state: 'existing',
            upstream_source: 'official',
            availability: 'callable'
          },
          {
            id: 'gemini-3.1-pro-preview',
            display_name: 'Gemini 3.1 Pro Preview',
            registry_state: 'missing',
            upstream_source: 'verified_extra',
            availability: 'uncallable',
            availability_reason: 'status 403 PERMISSION_DENIED'
          }
        ]
      })
      .mockResolvedValueOnce({
        probe_source: 'vertex_express_catalog',
        probe_notice: '',
        models: [
          {
            id: 'gemini-2.5-pro',
            display_name: 'Gemini 2.5 Pro',
            registry_state: 'existing',
            upstream_source: 'official',
            availability: 'callable'
          },
          {
            id: 'gemini-2.0-flash',
            display_name: 'Gemini 2.0 Flash',
            registry_state: 'existing',
            upstream_source: 'official',
            availability: 'callable'
          },
          {
            id: 'gemini-3.1-pro-preview',
            display_name: 'Gemini 3.1 Pro Preview',
            registry_state: 'missing',
            upstream_source: 'verified_extra',
            availability: 'uncallable',
            availability_reason: 'status 403 PERMISSION_DENIED'
          }
        ]
      })

    const wrapper = createWrapper()
    const probeButton = findProbeButton(wrapper)

    await probeButton?.trigger('click')
    await flushPromises()

    const selectCallableButton = findButtonByText(
      wrapper,
      'admin.accounts.apiKeyProbe.selectCallableModels'
    )
    expect(selectCallableButton).toBeTruthy()
    await selectCallableButton?.trigger('click')
    await flushPromises()

    const uncallableCard = wrapper.find('button[title="gemini-3.1-pro-preview"]')
    expect(uncallableCard.exists()).toBe(true)

    await uncallableCard.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.selectedUncallableWarning')
    await wrapper.setProps({
      allowedModels: ['gemini-2.0-flash', 'gemini-3.1-pro-preview'],
      modelMappings: [
        { from: 'Flash Alias', to: 'gemini-2.0-flash' },
        { from: 'Vertex-gemini-3.1-pro-preview', to: 'gemini-3.1-pro-preview' }
      ]
    })

    await probeButton?.trigger('click')
    await flushPromises()

    const allowedModelsUpdates = wrapper.emitted('update:allowedModels') || []
    expect(allowedModelsUpdates.at(-1)).toEqual([
      ['gemini-2.0-flash', 'gemini-3.1-pro-preview']
    ])

    const modelMappingsUpdates = wrapper.emitted('update:modelMappings') || []
    expect(modelMappingsUpdates.at(-1)).toEqual([
      [
        { from: 'Flash Alias', to: 'gemini-2.0-flash' },
        { from: 'Vertex-gemini-3.1-pro-preview', to: 'gemini-3.1-pro-preview' }
      ]
    ])
    expect(wrapper.find('input[placeholder="gemini-2.0-flash"]').exists()).toBe(false)
  })

  it('hydrates probe cards from existing mappings without rendering inline alias inputs', async () => {
    const wrapper = createWrapper({
      allowedModels: ['gemini-2.0-flash'],
      modelMappings: [{ from: '', to: 'gemini-2.0-flash' }],
      probedModels: []
    })

    await flushPromises()

    expect(wrapper.find('button[title="gemini-2.0-flash"]').exists()).toBe(true)
    expect(wrapper.find('input[placeholder="gemini-2.0-flash"]').exists()).toBe(false)
  })

  it('renders long model ids alongside source and status badges without truncating the text content', () => {
    const longModelId = 'gemini-3.1-flash-image-preview-with-a-very-long-suffix-for-layout-testing'
    const wrapper = createWrapper({
      allowedModels: [longModelId],
      modelMappings: [{ from: `Vertex-${longModelId}`, to: longModelId }],
      probedModels: [
        {
          id: longModelId,
          display_name: longModelId,
          registry_state: 'existing',
          upstream_source: 'verified_extra',
          availability: 'callable'
        }
      ]
    })

    expect(wrapper.text()).toContain(longModelId)
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.sourceVerifiedExtra')
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.availabilityCallable')
    expect(wrapper.html()).toContain('lg:grid-cols-2')
    expect(wrapper.html()).toContain('flex-wrap')
  })

  it('shows structured 403 repair guidance from backend metadata', async () => {
    probeModels.mockRejectedValue({
      message: 'upstream model listing failed with status 403',
      metadata: {
        hint_key: 'openai_api_model_read'
      }
    })

    const wrapper = createWrapper({
      platform: 'openai',
      credentials: {
        api_key: 'sk-test',
        base_url: 'https://api.openai.com'
      }
    })
    const probeButton = findProbeButton(wrapper)

    await probeButton?.trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(
      'upstream model listing failed with status 403\nOpenAI model list permission hint'
    )
  })

  it('passes manual model provider metadata through probe requests', async () => {
    probeModels.mockResolvedValue({
      probe_source: 'vertex_express_catalog',
      probe_notice: '',
      models: []
    })

    const wrapper = createWrapper({
      manualModels: [
        {
          model_id: 'custom-model',
          request_alias: 'Custom Alias',
          provider: 'grok'
        }
      ]
    })

    const probeButton = findProbeButton(wrapper)
    await probeButton?.trigger('click')
    await flushPromises()

    expect(probeModels).toHaveBeenCalledWith(
      expect.objectContaining({
        manual_models: [
          {
            model_id: 'custom-model',
            request_alias: 'Custom Alias',
            provider: 'grok'
          }
        ]
      })
    )
  })
})
