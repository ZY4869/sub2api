import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (key === 'admin.models.editor.tierHint') {
        return `Low tier ${params?.low ?? ''} · High tier ${params?.high ?? ''}`
      }
      return key
    },
  }),
}))

import ModelCatalogPricingEditorSection from '../ModelCatalogPricingEditorSection.vue'

describe('ModelCatalogPricingEditorSection', () => {
  it('prefills missing thresholds with 200000 and shows tier hint', () => {
    const wrapper = mount(ModelCatalogPricingEditorSection, {
      props: {
        detail: {
          model: 'gpt-4o-mini',
          official_pricing: { input_cost_per_token: 0.0000015 },
          sale_pricing: { output_cost_per_token: 0.0000035 },
        },
        layer: 'official',
        saving: false,
      },
    })

    const inputs = wrapper.findAll('input')
    expect(inputs[0].element.value).toBe('200000')
    expect(inputs[5].element.value).toBe('200000')
    expect(wrapper.text()).toContain('Low tier <= 200,000 · High tier >= 200,001')
  })

  it('emits virtual threshold when above-threshold price is edited', async () => {
    const wrapper = mount(ModelCatalogPricingEditorSection, {
      props: {
        detail: {
          model: 'gpt-4o-mini',
          official_pricing: {},
          sale_pricing: {},
        },
        layer: 'official',
        saving: false,
      },
    })

    const inputs = wrapper.findAll('input')
    await inputs[2].setValue('3.5')
    await wrapper.findAll('button')[1].trigger('click')

    expect(wrapper.emitted('save')).toBeTruthy()
    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({
      model: 'gpt-4o-mini',
      input_token_threshold: 200000,
      input_cost_per_token_above_threshold: 0.0000035,
    })
  })
})
