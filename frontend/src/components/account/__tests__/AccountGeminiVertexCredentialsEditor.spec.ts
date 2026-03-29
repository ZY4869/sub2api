import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGeminiVertexCredentialsEditor from '../AccountGeminiVertexCredentialsEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const createWrapper = (overrides: Record<string, unknown> = {}) =>
  mount(AccountGeminiVertexCredentialsEditor, {
    props: {
      authMode: 'service_account',
      projectId: '',
      location: 'global',
      serviceAccountJson: '',
      apiKey: '',
      legacyAccessToken: '',
      legacyExpiresAtInput: '',
      baseUrl: '',
      ...overrides
    },
    global: {
      stubs: {
        Select: {
          template: '<div data-test="select-stub"></div>'
        }
      }
    }
  })

describe('AccountGeminiVertexCredentialsEditor', () => {
  it('switches from service account to express mode', async () => {
    const wrapper = createWrapper()

    expect(wrapper.text()).toContain('admin.accounts.gemini.vertex.projectId')
    expect(wrapper.text()).not.toContain('admin.accounts.gemini.vertex.expressApiKey')

    const expressButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.gemini.vertex.authModes.expressTitle')
    )

    expect(expressButton).toBeTruthy()
    await expressButton?.trigger('click')

    expect(wrapper.emitted('update:authMode')).toContainEqual(['express_api_key'])
  })

  it('renders express mode fields without project and location inputs', () => {
    const wrapper = createWrapper({ authMode: 'express_api_key' })

    expect(wrapper.text()).toContain('admin.accounts.gemini.vertex.expressApiKey')
    expect(wrapper.text()).not.toContain('admin.accounts.gemini.vertex.projectId')
    expect(wrapper.text()).not.toContain('admin.accounts.gemini.vertex.location')
  })
})
