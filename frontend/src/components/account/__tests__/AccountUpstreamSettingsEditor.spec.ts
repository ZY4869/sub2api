import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountUpstreamSettingsEditor from '../AccountUpstreamSettingsEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountUpstreamSettingsEditor', () => {
  it('renders required fields in create mode', async () => {
    const wrapper = mount(AccountUpstreamSettingsEditor, {
      props: {
        mode: 'create',
        baseUrl: '',
        apiKey: ''
      }
    })

    const [baseUrlInput, apiKeyInput] = wrapper.findAll('input')
    expect(baseUrlInput.attributes('required')).toBeDefined()
    expect(apiKeyInput.attributes('required')).toBeDefined()
    expect(wrapper.text()).toContain('admin.accounts.upstream.apiKeyHint')

    await baseUrlInput.setValue('https://example.com')
    await apiKeyInput.setValue('sk-test')

    expect(wrapper.emitted('update:baseUrl')?.[0]).toEqual(['https://example.com'])
    expect(wrapper.emitted('update:apiKey')?.[0]).toEqual(['sk-test'])
  })

  it('renders keep-existing hint in edit mode', () => {
    const wrapper = mount(AccountUpstreamSettingsEditor, {
      props: {
        mode: 'edit',
        baseUrl: '',
        apiKey: ''
      }
    })

    const [baseUrlInput, apiKeyInput] = wrapper.findAll('input')
    expect(baseUrlInput.attributes('required')).toBeUndefined()
    expect(apiKeyInput.attributes('required')).toBeUndefined()
    expect(wrapper.text()).toContain('admin.accounts.leaveEmptyToKeep')
  })
})
