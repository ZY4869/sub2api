import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGeminiHelpDialog from '../AccountGeminiHelpDialog.vue'

vi.mock('@/api/admin/settings', () => ({
  getGeminiRateCatalog: vi.fn().mockResolvedValue({
    effective_date: '2026-03-31',
    remaining_quota_api_supported: false,
    ai_studio_tiers: [
      {
        tier_id: 'aistudio_free',
        display_name: 'Free',
        qualification: 'Default',
        billing_tier_cap: '$0',
        model_families: []
      }
    ],
    batch_limits: {
      concurrent_batch_requests: 2,
      input_file_size_limit_bytes: 1000,
      file_storage_limit_bytes: 2000,
      by_tier: []
    },
    links: [
      { label: 'Pricing', url: 'https://ai.google.dev/pricing' }
    ],
    notes: []
  })
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

const baseDialogStub = {
  name: 'BaseDialog',
  props: ['show', 'title', 'maxWidth'],
  template: `
    <div data-testid="base-dialog">
      <span data-testid="dialog-title">{{ title }}</span>
      <slot />
      <slot name="footer" />
    </div>
  `
}

describe('AccountGeminiHelpDialog', () => {
  it('renders help content and emits close from footer button', async () => {
    const wrapper = mount(AccountGeminiHelpDialog, {
      props: {
        show: true
      },
      global: {
        stubs: {
          BaseDialog: baseDialogStub
        }
      }
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="dialog-title"]').text()).toBe('admin.accounts.gemini.helpDialog.title')
    expect(wrapper.text()).toContain('admin.accounts.gemini.quotaPolicy.title')

    const links = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(links).toContain('https://aistudio.google.com/app/apikey')
    expect(links).toContain('https://ai.google.dev/pricing')

    await wrapper.get('.btn-primary').trigger('click')
    expect(wrapper.emitted('close')).toEqual([[]])
  })
})
