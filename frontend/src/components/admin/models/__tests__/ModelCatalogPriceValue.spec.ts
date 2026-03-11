import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelCatalogPriceValue from '../ModelCatalogPriceValue.vue'

describe('ModelCatalogPriceValue', () => {
  it('shows only USD by default', () => {
    const wrapper = mount(ModelCatalogPriceValue, {
      props: {
        value: 0.000001,
        unit: 'token',
        exchangeRate: { rate: 7.2 }
      }
    })

    expect(wrapper.text()).toContain('$1.0000')
    expect(wrapper.text()).not.toContain('≈')
  })

  it('shows USD and CNY in dual mode', () => {
    const wrapper = mount(ModelCatalogPriceValue, {
      props: {
        value: 0.000001,
        unit: 'token',
        exchangeRate: { rate: 7.2 },
        displayMode: 'dual'
      }
    })

    expect(wrapper.text()).toContain('$1.0000')
    expect(wrapper.text()).toContain('≈ ¥7.20')
  })
})
