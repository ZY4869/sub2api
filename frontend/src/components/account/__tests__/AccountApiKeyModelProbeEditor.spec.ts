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
      ...overrides
    },
    global: {
      stubs: {
        Icon: {
          template: '<span />'
        }
      }
    }
  })

const findProbeButton = (wrapper: ReturnType<typeof createWrapper>) =>
  wrapper.findAll('button').find((button) =>
    button.text().includes('admin.accounts.apiKeyProbe.action')
  )

describe('AccountApiKeyModelProbeEditor', () => {
  beforeEach(() => {
    probeModels.mockReset()
    showError.mockReset()
  })

  it('uses Vertex-prefixed aliases and only auto-selects callable models', async () => {
    probeModels.mockResolvedValue({
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

    const wrapper = createWrapper()
    const probeButton = findProbeButton(wrapper)

    expect(probeButton).toBeTruthy()
    await probeButton?.trigger('click')
    await flushPromises()

    expect(probeModels).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.sourceOfficial')
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.sourceVerifiedExtra')
    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.availabilityUncallable')

    const allowedModelsUpdates = wrapper.emitted('update:allowedModels') || []
    expect(allowedModelsUpdates.at(-1)).toEqual([['gemini-2.0-flash']])

    const modelMappingsUpdates = wrapper.emitted('update:modelMappings') || []
    expect(modelMappingsUpdates.at(-1)).toEqual([
      [{ from: 'Vertex-gemini-2.0-flash', to: 'gemini-2.0-flash' }]
    ])
  })

  it('keeps manually selected uncallable models selected after re-probing and shows a warning', async () => {
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

    const uncallableCard = wrapper.find('button[title="gemini-3.1-pro-preview"]')
    expect(uncallableCard.exists()).toBe(true)

    await uncallableCard.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.apiKeyProbe.selectedUncallableWarning')

    await probeButton?.trigger('click')
    await flushPromises()

    const allowedModelsUpdates = wrapper.emitted('update:allowedModels') || []
    expect(allowedModelsUpdates.at(-1)).toEqual([
      ['gemini-2.0-flash', 'gemini-3.1-pro-preview']
    ])
  })

  it('keeps an empty alias blank instead of auto-filling the Vertex prefix', async () => {
    const wrapper = createWrapper({
      allowedModels: ['gemini-2.0-flash'],
      modelMappings: [{ from: '', to: 'gemini-2.0-flash' }],
      probedModels: [
        {
          id: 'gemini-2.0-flash',
          display_name: 'Gemini 2.0 Flash',
          registry_state: 'existing'
        }
      ]
    })

    const aliasInput = wrapper.find('input[placeholder="gemini-2.0-flash"]')
    expect(aliasInput.exists()).toBe(true)
    expect((aliasInput.element as HTMLInputElement).value).toBe('')
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
})
