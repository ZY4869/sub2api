import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UseKeyModal from '../UseKeyModal.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('@/composables/useModelWhitelist', () => ({
  getModelsByPlatform: vi.fn(() => ['gpt-5.4-mini', 'gpt-5.4-nano']),
  getModelCapabilities: vi.fn(() => ({
    name: 'GPT-5.4 Mini',
    limit: { context: 400000, output: 128000 }
  }))
}))

const BaseDialogStub = {
  name: 'BaseDialogStub',
  props: ['show', 'title'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
}

describe('UseKeyModal', () => {
  it('uses gpt-5.4-mini as the default OpenAI Codex config model', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai',
        allowMessagesDispatch: false
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: true
        }
      }
    })

    const text = wrapper.text()
    expect(text).toContain('model_provider = "OpenAI"')
    expect(text).toContain('model = "gpt-5.4-mini"')
    expect(text).toContain('review_model = "gpt-5.4-mini"')
    expect(text).not.toContain('review_model = "gpt-5.4"')
  })

  it('keeps the explicit Grok image example path for Grok keys', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'grok',
        allowMessagesDispatch: false
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('/grok/v1/images/generations')
  })
})
