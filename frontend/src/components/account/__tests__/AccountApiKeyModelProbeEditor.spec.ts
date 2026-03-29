import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import AccountApiKeyModelProbeEditor from '../AccountApiKeyModelProbeEditor.vue'

const { probeModels } = vi.hoisted(() => ({
  probeModels: vi.fn()
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

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      probeModels
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
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

describe('AccountApiKeyModelProbeEditor', () => {
  beforeEach(() => {
    probeModels.mockReset()
  })

  it('uses Vertex-prefixed aliases for Vertex-sourced probe results', async () => {
    probeModels.mockResolvedValue({
      probe_source: 'vertex_express_catalog',
      probe_notice: '',
      models: [
        {
          id: 'gemini-2.0-flash',
          display_name: 'Gemini 2.0 Flash',
          registry_state: 'existing'
        }
      ]
    })

    const wrapper = createWrapper()
    const probeButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.apiKeyProbe.action')
    )

    expect(probeButton).toBeTruthy()
    await probeButton?.trigger('click')
    await flushPromises()

    expect(probeModels).toHaveBeenCalledTimes(1)
    const modelMappingsUpdates = wrapper.emitted('update:modelMappings') || []
    expect(modelMappingsUpdates.at(-1)).toEqual([
      [{ from: 'Vertex-gemini-2.0-flash', to: 'gemini-2.0-flash' }]
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
})
