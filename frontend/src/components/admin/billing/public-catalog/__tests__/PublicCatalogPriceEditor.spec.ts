import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PublicCatalogPriceEditor from '../PublicCatalogPriceEditor.vue'

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

describe('PublicCatalogPriceEditor', () => {
  it('emits fixed image pricing updates and validates always fixed prices', async () => {
    const wrapper = mount(PublicCatalogPriceEditor, {
      props: {
        editable: true,
        testidPrefix: 'price-test',
        sale: {
          primary: [{ id: 'output_price', unit: 'image', value: 0.5, configured: true }],
        },
        imageFixedPricing: {
          enabled: false,
          always_fixed: false,
          prices: { '1K': null, '2K': null, '4K': null },
        },
      },
      global: {
        stubs: {
          PublicCatalogPriceEntries: true,
        },
      },
    })

    await wrapper.get('[data-testid="price-test-image-fixed-enabled"]').setValue(true)
    expect(wrapper.emitted('update:imageFixedPricing')?.at(-1)?.[0]).toMatchObject({
      enabled: true,
      always_fixed: false,
    })

    await wrapper.setProps({
      imageFixedPricing: {
        enabled: true,
        always_fixed: true,
        prices: { '1K': 0.1, '2K': null, '4K': null },
      },
    })
    expect(wrapper.text()).toContain('admin.billing.publicCatalog.imageFixed.alwaysFixedError')

    await wrapper.get('[data-testid="price-test-image-fixed-2K"]').setValue('0.2')
    expect(wrapper.emitted('update:imageFixedPricing')?.at(-1)?.[0]).toMatchObject({
      prices: { '1K': 0.1, '2K': 0.2, '4K': null },
    })
  })
})
