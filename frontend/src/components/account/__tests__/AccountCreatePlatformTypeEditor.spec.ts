import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountCreatePlatformTypeEditor from '../AccountCreatePlatformTypeEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const geminiStub = {
  name: 'AccountGeminiAccountTypeEditor',
  props: ['aiStudioOAuthEnabled', 'apiKeyHelpLink', 'gcpProjectHelpLink'],
  emits: ['open-help'],
  template: `
    <div data-testid="gemini-stub">
      <span data-testid="gemini-ai-studio">{{ aiStudioOAuthEnabled }}</span>
      <span data-testid="gemini-api-key-link">{{ apiKeyHelpLink }}</span>
      <span data-testid="gemini-gcp-link">{{ gcpProjectHelpLink }}</span>
      <button type="button" data-testid="gemini-help" @click="$emit('open-help')" />
    </div>
  `
}

const upstreamStub = {
  name: 'AccountUpstreamSettingsEditor',
  props: ['baseUrl', 'apiKey', 'mode'],
  emits: ['update:baseUrl', 'update:apiKey'],
  template: `
    <div data-testid="upstream-stub">
      <span data-testid="upstream-mode">{{ mode }}</span>
      <button type="button" data-testid="emit-base" @click="$emit('update:baseUrl', 'https://example.com')" />
      <button type="button" data-testid="emit-key" @click="$emit('update:apiKey', 'sk-test')" />
    </div>
  `
}

const createWrapper = (overrides: Record<string, unknown> = {}) =>
  mount(AccountCreatePlatformTypeEditor, {
    props: {
      platform: 'anthropic',
      accountCategory: 'oauth-based',
      addMethod: 'oauth',
      soraAccountType: 'oauth',
      antigravityAccountType: 'oauth',
      geminiOAuthType: 'google_one',
      showAdvanced: false,
      geminiTierGoogleOne: 'google_one_free',
      geminiTierGcp: 'gcp_standard',
      geminiTierAiStudio: 'aistudio_free',
      gatewayProtocol: 'openai',
      upstreamBaseUrl: '',
      upstreamApiKey: '',
      aiStudioOAuthEnabled: false,
      apiKeyHelpLink: 'https://example.com/api-key',
      gcpProjectHelpLink: 'https://example.com/gcp-project',
      ...overrides
    },
    global: {
      stubs: {
        AccountGeminiAccountTypeEditor: geminiStub,
        AccountUpstreamSettingsEditor: upstreamStub,
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: `
            <div data-testid="select-stub">
              <div data-testid="selected-option">
                <slot name="selected" :option="options.find((item) => item.value === modelValue) || null" />
              </div>
              <button
                v-for="option in options"
                :key="option.value"
                type="button"
                class="select-option"
                @click="$emit('update:modelValue', option.value)"
              >
                <slot name="option" :option="option" />
              </button>
            </div>
          `
        }
      }
    }
  })

describe('AccountCreatePlatformTypeEditor', () => {
  it('updates sora type and syncs oauth defaults', async () => {
    const wrapper = createWrapper({
      platform: 'sora',
      accountCategory: 'apikey',
      addMethod: 'setup-token',
      soraAccountType: 'apikey'
    })

    const oauthButton = wrapper.findAll('button').find((button) => button.text().includes('OAuth'))
    expect(oauthButton).toBeTruthy()

    await oauthButton?.trigger('click')

    expect(wrapper.emitted('update:soraAccountType')).toContainEqual(['oauth'])
    expect(wrapper.emitted('update:accountCategory')).toContainEqual(['oauth-based'])
    expect(wrapper.emitted('update:addMethod')).toContainEqual(['oauth'])
  })

  it('renders anthropic add method selector and emits category changes', async () => {
    const wrapper = createWrapper()

    expect(wrapper.text()).toContain('admin.accounts.setupTokenLongLived')

    await wrapper.find('input[value="setup-token"]').setValue()
    expect(wrapper.emitted('update:addMethod')).toContainEqual(['setup-token'])

    const apiKeyButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.claudeConsole')
    )

    expect(apiKeyButton).toBeTruthy()
    await apiKeyButton?.trigger('click')
    expect(wrapper.emitted('update:accountCategory')).toContainEqual(['apikey'])
  })

  it('switches grok account types between sso and apikey', async () => {
    const wrapper = createWrapper({
      platform: 'grok',
      accountCategory: 'oauth-based'
    })

    expect(wrapper.text()).toContain('admin.accounts.types.grokSso')

    const apiKeyButton = wrapper.findAll('button').find((button) =>
      button.text().includes('API Key')
    )
    expect(apiKeyButton).toBeTruthy()

    await apiKeyButton!.trigger('click')
    expect(wrapper.emitted('update:accountCategory')).toContainEqual(['apikey'])
  })

  it('bridges gemini help events to the parent', async () => {
    const wrapper = createWrapper({
      platform: 'gemini',
      aiStudioOAuthEnabled: true
    })

    expect(wrapper.get('[data-testid="gemini-ai-studio"]').text()).toBe('true')
    expect(wrapper.get('[data-testid="gemini-api-key-link"]').text()).toBe('https://example.com/api-key')
    expect(wrapper.get('[data-testid="gemini-gcp-link"]').text()).toBe('https://example.com/gcp-project')

    await wrapper.get('[data-testid="gemini-help"]').trigger('click')
    expect(wrapper.emitted('openGeminiHelp')).toEqual([[]])
  })

  it('shows upstream settings for antigravity upstream accounts and forwards updates', async () => {
    const wrapper = createWrapper({
      platform: 'antigravity',
      antigravityAccountType: 'upstream'
    })

    expect(wrapper.get('[data-testid="upstream-mode"]').text()).toBe('create')

    await wrapper.get('[data-testid="emit-base"]').trigger('click')
    await wrapper.get('[data-testid="emit-key"]').trigger('click')

    expect(wrapper.emitted('update:upstreamBaseUrl')).toContainEqual(['https://example.com'])
    expect(wrapper.emitted('update:upstreamApiKey')).toContainEqual(['sk-test'])

    const oauthWrapper = createWrapper({
      platform: 'antigravity',
      antigravityAccountType: 'oauth'
    })
    const apiKeyButton = oauthWrapper.findAll('button').find((button) => button.text().includes('API Key'))

    expect(apiKeyButton).toBeTruthy()
    await apiKeyButton?.trigger('click')
    expect(oauthWrapper.emitted('update:antigravityAccountType')).toContainEqual(['upstream'])
  })

  it('skips Step 1 type cards for kiro and copilot', () => {
    for (const platform of ['kiro', 'copilot'] as const) {
      const wrapper = createWrapper({
        platform
      })

      expect(wrapper.text()).not.toContain('admin.accounts.accountType')
      expect(wrapper.findAll('button')).toHaveLength(0)
      expect(wrapper.find('[data-testid="gemini-stub"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="upstream-stub"]').exists()).toBe(false)
    }
  })

  it('shows protocol gateway request formats on the same row', async () => {
    const wrapper = createWrapper({
      platform: 'protocol_gateway',
      accountCategory: 'apikey',
      gatewayProtocol: 'openai'
    })

    expect(wrapper.get('[data-testid="selected-option"]').text()).toContain('OpenAI')
    expect(wrapper.get('[data-testid="selected-option"]').text()).toContain('/v1/chat/completions')
    expect(wrapper.text()).toContain('/v1/responses')
    expect(wrapper.text()).toContain('/v1/messages')

    const anthropicButton = wrapper.findAll('.select-option').find((button) =>
      button.text().includes('Anthropic')
    )
    expect(anthropicButton).toBeTruthy()

    await anthropicButton!.trigger('click')
    expect(wrapper.emitted('update:gatewayProtocol')).toContainEqual(['anthropic'])
  })
})
