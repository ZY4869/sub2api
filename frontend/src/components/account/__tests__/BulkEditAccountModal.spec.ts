import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BulkEditAccountModal from '../BulkEditAccountModal.vue'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      bulkUpdate: vi.fn(),
      checkMixedChannelRisk: vi.fn()
    }
  }
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
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

function mountModal() {
  return mount(BulkEditAccountModal, {
    props: {
      show: true,
      accountIds: [1, 2],
      selectedPlatforms: ['antigravity'],
      selectedTypes: [],
      proxies: [],
      groups: []
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: true,
        ProxySelector: true,
        GroupSelector: true,
        Icon: true
      }
    }
  })
}

describe('BulkEditAccountModal', () => {
  it('shows Claude 4.6 models as independent antigravity whitelist options', () => {
    const wrapper = mountModal()

    expect(wrapper.text()).toContain('claude-opus-4.1')
    expect(wrapper.text()).toContain('claude-opus-4-6')
    expect(wrapper.text()).toContain('claude-sonnet-4.5')
    expect(wrapper.text()).toContain('claude-sonnet-4-6')
    expect(wrapper.text()).toContain('claude-haiku-4.5')
    expect(wrapper.text()).toContain('gemini-2.5-flash-image')
    expect(wrapper.text()).toContain('gemini-3.1-flash-image')
  })

  it('removes legacy 4.6 presets from antigravity mappings', async () => {
    const wrapper = mountModal()

    const mappingTab = wrapper.findAll('button').find((btn) => btn.text().includes('admin.accounts.modelMapping'))
    expect(mappingTab).toBeTruthy()
    await mappingTab!.trigger('click')

    expect(wrapper.text()).toContain('2.5-Flash-Image')
    expect(wrapper.text()).toContain('3.1-Flash-Image')
    expect(wrapper.text()).toContain('Gemini 3->Flash')
    expect(wrapper.text()).toContain('Sonnet 4.5')
    expect(wrapper.text()).toContain('Opus 4.1')
    expect(wrapper.text()).not.toContain('4.6')
  })
})
