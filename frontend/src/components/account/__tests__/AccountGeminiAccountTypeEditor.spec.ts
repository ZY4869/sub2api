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
  it('renders the three Google secondary entry cards and emits help/category updates', async () => {
    const wrapper = createWrapper()

    const helpButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.gemini.helpButton'))
    const apiKeyButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.gemini.accountType.apiKeyTitle'))
    const oauthButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.gemini.accountType.oauthTitle'))
    const vertexButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.gemini.oauthType.vertexTitle'))

    expect(helpButton).toBeTruthy()
    expect(oauthButton).toBeTruthy()
    expect(apiKeyButton).toBeTruthy()
    expect(vertexButton).toBeTruthy()

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

  it('keeps the advanced panel open when selecting ai studio', async () => {
    const wrapper = createWrapper({ showAdvanced: true, aiStudioOAuthEnabled: true })
    const aiStudioButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.gemini.oauthType.customTitle')
    )

    expect(aiStudioButton).toBeTruthy()
    await aiStudioButton?.trigger('click')

    expect(wrapper.emitted('update:oauthType')).toContainEqual(['ai_studio'])
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

  it('selects the vertex ai branch as a top-level Google entry', async () => {
    const wrapper = createWrapper({ showAdvanced: false, aiStudioOAuthEnabled: true, oauthType: 'code_assist' })
    const vertexButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.gemini.oauthType.vertexTitle')
    )

    expect(vertexButton).toBeTruthy()
    await vertexButton?.trigger('click')

    expect(wrapper.emitted('update:accountCategory')).toContainEqual(['vertex_ai'])
    expect(wrapper.emitted('update:oauthType')).toContainEqual(['vertex_ai'])
  })

  it('shows the vertex credentials branch without oauth subtype or tier controls', () => {
    const wrapper = createWrapper({
      accountCategory: 'vertex_ai',
      showAdvanced: false,
      aiStudioOAuthEnabled: true,
      oauthType: 'vertex_ai'
    })

    expect(wrapper.text()).toContain('admin.accounts.gemini.vertex.formInlineHint')
    expect(wrapper.text()).not.toContain('admin.accounts.oauth.gemini.oauthTypeLabel')
    expect(wrapper.text()).not.toContain('admin.accounts.gemini.oauthType.customTitle')
    expect(wrapper.find('select').exists()).toBe(false)
  })

  it('shows the AI Studio API key helper note in api key mode', () => {
    const wrapper = createWrapper({ accountCategory: 'apikey' })
    const apiKeyLink = wrapper.find('a')

    expect(wrapper.text()).toContain('admin.accounts.gemini.accountType.apiKeyNote')
    expect(wrapper.text()).toContain('admin.accounts.gemini.accountType.apiKeyLink')
    expect(apiKeyLink.attributes('href')).toBe('https://example.com/api-key')
  })
})
