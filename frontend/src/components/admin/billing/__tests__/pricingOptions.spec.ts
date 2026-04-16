import { describe, expect, it } from 'vitest'
import {
  billingLayerHasSpecialValues,
  billingLayerHasValues,
  cloneBillingPricingLayerForm,
  countConfiguredBillingFields,
  outputPriceLabel,
} from '../pricingOptions'
import { pricingFieldUnitLabelForField } from '../pricingFieldPresentation'

describe('pricingOptions', () => {
  it('clones layer forms and preserves nested special values', () => {
    const form = cloneBillingPricingLayerForm({
      input_price: 1.2,
      special_enabled: true,
      special: {
        batch_input_price: 0.5,
      },
      tiered_enabled: true,
      tier_threshold_tokens: 200000,
    })

    expect(form).toEqual({
      input_price: 1.2,
      output_price: undefined,
      cache_price: undefined,
      special_enabled: true,
      special: {
        batch_input_price: 0.5,
      },
      tiered_enabled: true,
      tier_threshold_tokens: 200000,
      input_price_above_threshold: undefined,
      output_price_above_threshold: undefined,
    })

    form.special.batch_input_price = 0.8

    const clonedAgain = cloneBillingPricingLayerForm({
      special_enabled: true,
      special: {
        batch_input_price: 0.5,
      },
    })
    expect(clonedAgain.special.batch_input_price).toBe(0.5)
  })

  it('counts configured fields across base, tier and special sections', () => {
    expect(countConfiguredBillingFields({
      input_price: 1,
      output_price: 2,
      special: {
        grounding_search: 0.01,
      },
      input_price_above_threshold: 3,
    })).toBe(4)

    expect(billingLayerHasValues({
      special: {},
    })).toBe(false)

    expect(billingLayerHasSpecialValues({
      special: {
        grounding_search: 0.01,
      },
    })).toBe(true)
  })

  it('maps output labels by model charge slot', () => {
    expect(outputPriceLabel()).toBe('输出定价')
    expect(outputPriceLabel('image_output')).toBe('图片输出定价')
    expect(outputPriceLabel('video_request')).toBe('视频请求定价')
  })

  it('maps pricing field units by field semantics', () => {
    expect(pricingFieldUnitLabelForField('input_price')).toBe('$ / M Tokens')
    expect(pricingFieldUnitLabelForField('batch_cache_price')).toBe('$ / M Tokens')
    expect(pricingFieldUnitLabelForField('grounding_search')).toBe('$ / 次')
    expect(pricingFieldUnitLabelForField('grounding_maps')).toBe('$ / 次')
    expect(pricingFieldUnitLabelForField('output_price', 'image_output')).toBe('$ / 张')
    expect(pricingFieldUnitLabelForField('batch_output_price', 'video_request')).toBe('$ / 次')
  })
})
