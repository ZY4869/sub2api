import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGeminiAccountTypeEditor from '../AccountGeminiAccountTypeEditor.vue'

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
  mount(AccountGeminiAccountTypeEditor, {
    props: {
      accountCategory: 'oauth-based',
      oauthType: 'google_one',
      showAdvanced: false,
      tierGoogleOne: 'google_one_free',
      tierGcp: 'gcp_standard',
      tierAiStudio: 'aistudio_free',
      aiStudioOAuthEnabled: false,
      apiKeyHelpLink: 'https://example.com/api-key',
      gcpProjectHelpLink: 'https://example.com/project',
      ...overrides
    }
  })

describe('AccountGeminiAccountTypeEditor', () => {
  it('emits help and account category updates', async () => {
    const wrapper = createWrapper()

    const helpButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.gemini.helpButton'))
    const apiKeyButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.gemini.accountType.apiKeyTitle'))

    expect(helpButton).toBeTruthy()
    expect(apiKeyButton).toBeTruthy()

    await helpButton?.trigger('click')
    await apiKeyButton?.trigger('click')

    expect(wrapper.emitted('openHelp')).toEqual([[]])
    expect(wrapper.emitted('update:accountCategory')).toContainEqual(['apikey'])
  })

  it('toggles advanced options and blocks ai studio selection when disabled', async () => {
    const wrapper = createWrapper()
    const advancedToggle = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.gemini.oauthType.advancedToggleShow')
    )

    expect(advancedToggle).toBeTruthy()
    await advancedToggle?.trigger('click')
    expect(wrapper.emitted('update:showAdvanced')).toEqual([[true]])

    const withAdvancedWrapper = createWrapper({ showAdvanced: true, aiStudioOAuthEnabled: false })
    const aiStudioButton = withAdvancedWrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.gemini.oauthType.customTitle')
    )

    expect(aiStudioButton).toBeTruthy()
    await aiStudioButton?.trigger('click')
    expect(withAdvancedWrapper.emitted('update:oauthType')).toBeUndefined()
  })

  it('updates oauth type and tier selections', async () => {
    const wrapper = createWrapper({ showAdvanced: true, aiStudioOAuthEnabled: true })
    const codeAssistButton = wrapper.findAll('button').find((button) => button.text().includes('GCP Code Assist'))

    expect(codeAssistButton).toBeTruthy()
    await codeAssistButton?.trigger('click')
    expect(wrapper.emitted('update:oauthType')).toContainEqual(['code_assist'])

    const codeAssistSelectWrapper = createWrapper({ oauthType: 'code_assist' })
    await codeAssistSelectWrapper.find('select').setValue('gcp_enterprise')
    expect(codeAssistSelectWrapper.emitted('update:tierGcp')).toEqual([['gcp_enterprise']])
  })

  it('shows and selects the vertex ai branch without tier selector', async () => {
    const wrapper = createWrapper({ showAdvanced: true, aiStudioOAuthEnabled: true, oauthType: 'code_assist' })
    const vertexButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.gemini.oauthType.vertexTitle')
    )

    expect(vertexButton).toBeTruthy()
    await vertexButton?.trigger('click')

    expect(wrapper.emitted('update:oauthType')).toContainEqual(['vertex_ai'])

    const vertexWrapper = createWrapper({
      showAdvanced: true,
      aiStudioOAuthEnabled: true,
      oauthType: 'vertex_ai'
    })
    expect(vertexWrapper.text()).toContain('admin.accounts.gemini.vertex.formInlineHint')
    expect(vertexWrapper.find('select').exists()).toBe(false)
  })
})
