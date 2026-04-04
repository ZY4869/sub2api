import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountTestModal from '../AccountTestModal.vue'

const { getAvailableModels, copyToClipboard, testGrokAccount } = vi.hoisted(() => ({
  getAvailableModels: vi.fn(),
  copyToClipboard: vi.fn(),
  testGrokAccount: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels,
      testGrokAccount
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.geminiImagePromptDefault': 'Generate a cute orange cat astronaut sticker on a clean pastel background.'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.geminiImageReceived' && params?.count) {
          return `received-${params.count}`
        }
        if (key === 'admin.models.registry.replacedByHint' && params?.model) {
          return `replaced-by-${params.model}`
        }
        return messages[key] || key
      }
    })
  }
})

function createStreamResponse(lines: string[]) {
  const encoder = new TextEncoder()
  const chunks = lines.map((line) => encoder.encode(line))
  let index = 0

  return {
    ok: true,
    body: {
      getReader: () => ({
        read: vi.fn().mockImplementation(async () => {
          if (index < chunks.length) {
            return { done: false, value: chunks[index++] }
          }
          return { done: true, value: undefined }
        })
      })
    }
  } as Response
}

function mountModal() {
  return mount(AccountTestModal, {
    props: {
      show: false,
      account: {
        id: 42,
        name: 'Gemini Image Test',
        platform: 'gemini',
        type: 'apikey',
        status: 'active'
      }
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: {
          props: ['modelValue', 'options'],
          template: `
        <div class="select-stub">
          <div data-test="selected-option">
                <slot name="selected" :option="options.find((opt) => (opt.key || opt.id) === modelValue) || null" />
          </div>
          <div
                v-for="option in options"
                :key="option.key || option.id"
                class="select-option-stub"
              >
                <slot name="option" :option="option" :selected="(option.key || option.id) === modelValue" />
              </div>
        </div>
          `
        },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        Icon: true
      }
    }
  })
}

describe('AccountTestModal', () => {
  beforeEach(() => {
    getAvailableModels.mockResolvedValue([
      { id: 'gemini-3.1-flash-image', display_name: 'Gemini 3.1 Flash Image' },
      { id: 'gemini-2.0-flash', display_name: 'Gemini 2.0 Flash' },
      { id: 'gemini-2.5-flash-image', display_name: 'Gemini 2.5 Flash Image' }
    ])
    testGrokAccount.mockReset()
    copyToClipboard.mockReset()
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => (key === 'auth_token' ? 'test-token' : null)),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn()
      },
      configurable: true
    })
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gemini-3.1-flash-image"}\n',
        'data: {"type":"image","image_url":"data:image/png;base64,QUJD","mime_type":"image/png"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('gemini 图片模型测试会携带提示词并渲染图片预览', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect(promptInput.exists()).toBe(true)
    await promptInput.setValue('draw a tiny orange cat astronaut')

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model: 'gemini-3.1-flash-image',
      model_id: 'gemini-3.1-flash-image',
      test_mode: 'real_forward',
      prompt: 'draw a tiny orange cat astronaut'
    })

    const preview = wrapper.find('img[alt="gemini-test-image-1"]')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('data:image/png;base64,QUJD')
  })

  it('renders display name above model id and shows deprecated metadata in the selector content', async () => {
    getAvailableModels.mockResolvedValueOnce([
      {
        id: 'claude-sonnet-legacy',
        display_name: 'Claude Sonnet Legacy',
        status: 'deprecated',
        replaced_by: 'claude-sonnet-4.5'
      }
    ])

    const wrapper = mount(AccountTestModal, {
      props: {
        show: false,
        account: {
          id: 7,
          name: 'Legacy Test',
          platform: 'openai',
          type: 'oauth',
          status: 'active'
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: {
            props: ['modelValue', 'options'],
            template: `
              <div>
                <div data-test="selected-option">
                  <slot name="selected" :option="options.find((opt) => (opt.key || opt.id) === modelValue) || null" />
                </div>
                <div v-for="option in options" :key="option.key || option.id" data-test="option">
                  <slot name="option" :option="option" :selected="(option.key || option.id) === modelValue" />
                </div>
              </div>
            `
          },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('Claude Sonnet Legacy')
    expect(text).toContain('claude-sonnet-legacy')
    expect(text).toContain('admin.models.registry.lifecycleLabels.deprecated')
    expect(text).toContain('replaced-by-claude-sonnet-4.5')
  })

  it('prefers canonical_id when rendering the model identifier in the selector', async () => {
    getAvailableModels.mockResolvedValueOnce([
      {
        id: 'claude-sonnet-4-5-20250929',
        canonical_id: 'claude-sonnet-4.5',
        display_name: 'Claude Sonnet 4.5'
      }
    ])

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('Claude Sonnet 4.5')
    expect(text).toContain('claude-sonnet-4.5')
    expect(text).not.toContain('claude-sonnet-4-5-20250929')
  })

  it('shows blacklist advice and emits direct blacklist feedback from the test modal', async () => {
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gemini-3.1-flash-image"}\n',
        'data: {"type":"blacklist_advice","data":{"decision":"recommend_blacklist","reason_code":"invalid_api_key","reason_message":"invalid api key","feedback_fingerprint":"fp-123","collect_feedback":true,"platform":"gemini","status_code":401,"error_code":"invalid_api_key","message_keywords":["invalid","key"]}}\n',
        'data: {"type":"error","error":"API returned 401: invalid api key"}\n'
      ])
    ) as any

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.testBlacklist.recommendTitle')
    expect(wrapper.text()).toContain('invalid api key')

    const blacklistButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.testBlacklist.button')
    )
    expect(blacklistButton).toBeTruthy()

    await blacklistButton!.trigger('click')

    expect(wrapper.emitted('blacklist')).toEqual([[
      {
        account: expect.objectContaining({ id: 42, name: 'Gemini Image Test' }),
        source: 'test_modal',
        feedback: {
          fingerprint: 'fp-123',
          advice_decision: 'recommend_blacklist',
          action: 'blacklist',
          platform: 'gemini',
          status_code: 401,
          error_code: 'invalid_api_key',
          message_keywords: ['invalid', 'key']
        }
      }
    ]])
  })

  it('persists the selected test mode and submits health_check when switched', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const healthCheckButton = wrapper.find('[data-test="test-mode-health_check"]')
    expect(healthCheckButton.exists()).toBe(true)

    await healthCheckButton.trigger('click')

    expect(globalThis.localStorage.setItem).toHaveBeenCalledWith(
      'sub2api.admin.accounts.test_mode',
      'health_check'
    )

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model: 'gemini-3.1-flash-image',
      model_id: 'gemini-3.1-flash-image',
      test_mode: 'health_check',
      prompt: 'Generate a cute orange cat astronaut sticker on a clean pastel background.'
    })
  })

  it('submits source_protocol for protocol gateway models and renders runtime context', async () => {
    getAvailableModels.mockResolvedValueOnce([
      {
        id: 'claude-sonnet-4-5',
        display_name: 'Claude Sonnet 4.5',
        source_protocol: 'anthropic'
      }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"claude-sonnet-4-5"}\n',
        'data: {"type":"content","text":"Gateway source protocol: Anthropic","data":{"kind":"runtime_meta","key":"resolved_protocol","value":"anthropic","label":"Anthropic","source":"protocol_gateway_test"}}\n',
        'data: {"type":"content","text":"Gateway simulated client: Claude Client Mimic","data":{"kind":"runtime_meta","key":"simulated_client","value":"claude_client_mimic","label":"Claude Client Mimic","source":"protocol_gateway_test"}}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mount(AccountTestModal, {
      props: {
        show: false,
        account: {
          id: 88,
          name: 'Gateway Anthropic',
          platform: 'protocol_gateway',
          gateway_protocol: 'mixed',
          extra: {
            gateway_accepted_protocols: ['anthropic']
          },
          type: 'apikey',
          status: 'active'
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: {
            props: ['modelValue', 'options'],
            template: `
              <div>
                <div data-test="selected-option">
                  <slot name="selected" :option="options.find((opt) => (opt.key || opt.id) === modelValue) || null" />
                </div>
                <div v-for="option in options" :key="option.key || option.id" data-test="option">
                  <slot name="option" :option="option" :selected="(option.key || option.id) === modelValue" />
                </div>
              </div>
            `
          },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model: 'claude-sonnet-4-5',
      model_id: 'claude-sonnet-4-5',
      test_mode: 'real_forward',
      source_protocol: 'anthropic',
      prompt: ''
    })
    expect(wrapper.text()).toContain('admin.accounts.testRuntimeContextTitle')
    expect(wrapper.text()).toContain('admin.accounts.testRuntimeContextProtocol')
    expect(wrapper.text()).toContain('admin.accounts.testRuntimeContextClient')
  })

  it('disables the blacklist button when generic unauthorized responses are auto blacklisted', async () => {
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gemini-3.1-flash-image"}\n',
        'data: {"type":"blacklist_advice","data":{"decision":"auto_blacklisted","reason_code":"credentials_likely_invalid","reason_message":"Unauthorized","already_blacklisted":true,"collect_feedback":false,"status_code":401}}\n',
        'data: {"type":"error","error":"API returned 401: {\\"detail\\":\\"Unauthorized\\"}"}\n'
      ])
    ) as any

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    const blacklistButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.testBlacklist.buttonDone')
    )
    expect(blacklistButton).toBeTruthy()
    expect(blacklistButton!.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).not.toContain('admin.accounts.testBlacklist.autoTitle')
    expect(wrapper.text()).not.toContain('admin.accounts.testBlacklist.autoBadge')
    expect(wrapper.text()).toContain('Unauthorized')
  })

  it('uses the Grok-specific test endpoint and renders probe lines as terminal output', async () => {
    getAvailableModels.mockResolvedValueOnce([
      { id: 'grok-3-beta', display_name: 'Grok 3 Beta' }
    ])
    testGrokAccount.mockResolvedValueOnce(
      createStreamResponse([
        'data: {"type":"test_start","model":"grok-3-beta"}\n',
        'data: {"type":"content","text":"Reverse runtime connectivity probe started"}\n',
        'data: {"type":"content","text":"Visible models after model_mapping: grok-3-beta"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    )

    const wrapper = mount(AccountTestModal, {
      props: {
        show: false,
        account: {
          id: 9,
          name: 'Grok Reverse',
          platform: 'grok',
          type: 'sso',
          status: 'active'
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: {
            props: ['modelValue', 'options'],
            template: `
              <div>
                <div data-test="selected-option">
                  <slot name="selected" :option="options.find((opt) => (opt.key || opt.id) === modelValue) || null" />
                </div>
                <div v-for="option in options" :key="option.key || option.id" data-test="option">
                  <slot name="option" :option="option" :selected="(option.key || option.id) === modelValue" />
                </div>
              </div>
            `
          },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.find('[data-test="test-mode-health_check"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.accounts.grokTestSsoHint')

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(testGrokAccount).toHaveBeenCalledWith(9, {
      model: 'grok-3-beta',
      model_id: 'grok-3-beta'
    })
    expect(wrapper.text()).toContain('Reverse runtime connectivity probe started')
    expect(wrapper.text()).toContain('Visible models after model_mapping: grok-3-beta')
  })
})
