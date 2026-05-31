import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PublicCatalogEntryEditDialog from '../PublicCatalogEntryEditDialog.vue'
import type { BillingPublicCatalogAdminEntry } from '@/api/admin/billing'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      te: () => true,
    }),
  }
})

const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><slot /><footer><slot name="footer" /></footer></section>',
}

const PassthroughStub = {
  template: '<div><slot /></div>',
}

function catalogEntry(overrides: Partial<BillingPublicCatalogAdminEntry> = {}): BillingPublicCatalogAdminEntry {
  return {
    entry_id: 'entry-gpt-54',
    public_model_id: 'gpt-5.4',
    model: 'gpt-5.4',
    base_model: 'gpt-5.4',
    source_model_id: 'gpt-5.4',
    source_protocol: 'openai',
    source_alias: 'Primary',
    source_account_id: 42,
    display_name: 'GPT-5.4',
    provider: 'openai',
    currency: 'USD',
    price_display: { primary: [{ id: 'output_price', unit: 'output_token', value: 2e-6, configured: true }] },
    sale_price_display: { primary: [{ id: 'output_price', unit: 'output_token', value: 3e-6, configured: true }] },
    official_price_display: { primary: [{ id: 'output_price', unit: 'output_token', value: 2e-6, configured: true }] },
    multiplier_summary: { enabled: false, kind: 'disabled' },
    ...overrides,
  }
}

describe('PublicCatalogEntryEditDialog', () => {
  it('emits timed discount payload from the editor controls', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-01T10:20:30.000Z'))
    try {
      const wrapper = mount(PublicCatalogEntryEditDialog, {
        props: {
          show: true,
          item: catalogEntry(),
        },
        global: {
          stubs: {
            BaseDialog: BaseDialogStub,
            ModelIcon: PassthroughStub,
            ModelPlatformIcon: PassthroughStub,
            Icon: PassthroughStub,
            PublicCatalogPriceEditor: PassthroughStub,
            TimeAccessPolicyEditor: PassthroughStub,
          },
        },
      })

      await wrapper.get('[data-testid="catalog-dialog-discount-enabled"]').setValue(true)
      await wrapper.get('[data-testid="catalog-dialog-discount-percent"]').setValue('35')
      await wrapper.get('[data-testid="catalog-dialog-discount-timezone"]').setValue('Asia/Tokyo')

      const dailyWindow = wrapper.vm.$data
      expect(dailyWindow).toBeDefined()

      await wrapper.find('select').setValue('once')
      await wrapper.get('[data-testid="catalog-dialog-save"]').trigger('click')

      const saveEvent = wrapper.emitted('save')?.[0]
      expect(saveEvent?.[0]).toBe('entry-gpt-54')
      expect(saveEvent?.[1]).toMatchObject({
        discount_policy: {
          enabled: true,
          reduction_percent: 35,
          timezone: 'Asia/Tokyo',
          windows: [{
            type: 'once',
            start_at: '2026-06-01T10:20:30.000Z',
            end_at: '2026-06-02T10:20:30.000Z',
          }],
        },
      })
      expect(wrapper.emitted('close')).toHaveLength(1)
    } finally {
      vi.useRealTimers()
    }
  })
})
