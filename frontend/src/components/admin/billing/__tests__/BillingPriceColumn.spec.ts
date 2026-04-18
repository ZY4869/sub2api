import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import type { BillingPricingLayerForm } from '@/api/admin/billing'
import BillingPriceColumn from '../BillingPriceColumn.vue'

function createForm(overrides: Partial<BillingPricingLayerForm> = {}): BillingPricingLayerForm {
  return {
    input_price: 3e-7,
    output_price: 8e-7,
    cache_price: 1.2e-7,
    special_enabled: false,
    special: {
      ...(overrides.special || {}),
    },
    tiered_enabled: false,
    tier_threshold_tokens: undefined,
    input_price_above_threshold: undefined,
    output_price_above_threshold: undefined,
    multiplier_enabled: true,
    multiplier_mode: 'shared',
    shared_multiplier: 0.12,
    item_multipliers: {},
    ...overrides,
  }
}

function mountColumn(initialForm: BillingPricingLayerForm) {
  return mount(defineComponent({
    components: {
      BillingPriceColumn,
    },
    setup() {
      const form = ref(createForm(initialForm))
      const handleUpdate = (value: BillingPricingLayerForm) => {
        form.value = value
      }
      return {
        form,
        handleUpdate,
        capabilities: {
          supports_tiered_pricing: true,
          supports_batch_pricing: true,
          supports_service_tier: false,
          supports_prompt_caching: true,
          supports_provider_special: true,
        },
      }
    },
    template: `
      <BillingPriceColumn
        title="Sale"
        :form="form"
        currency="USD"
        :input-supported="true"
        output-charge-slot="text_output"
        :supports-prompt-caching="true"
        :capabilities="capabilities"
        :show-multiplier-controls="true"
        @update-form="handleUpdate"
      />
    `,
  }))
}

function currentForm(wrapper: ReturnType<typeof mount>) {
  return wrapper.getComponent(BillingPriceColumn).props('form') as BillingPricingLayerForm
}

describe('BillingPriceColumn', () => {
  it('shows shared multiplier inline preview and keeps the shared mode editable', async () => {
    const wrapper = mountColumn(createForm())

    expect(wrapper.get('[data-testid="pricing-multiplier-inline-input_price"]').text()).toContain('0.3')
    expect(wrapper.get('[data-testid="pricing-multiplier-inline-input_price"]').text()).toContain('0.12')
    expect(wrapper.get('[data-testid="pricing-multiplier-inline-input_price"]').text()).toContain('0.036')

    await wrapper.get('[data-testid="pricing-shared-multiplier"]').setValue('0.2')

    expect(currentForm(wrapper).multiplier_mode).toBe('shared')
    expect(currentForm(wrapper).shared_multiplier).toBe(0.2)
    expect(wrapper.get('[data-testid="pricing-multiplier-inline-input_price"]').text()).toContain('0.06')
  })

  it('switches to item mode and saves per-item multipliers', async () => {
    const wrapper = mountColumn(createForm())

    await wrapper.get('[data-testid="pricing-multiplier-mode-item"]').trigger('click')
    await wrapper.get('[data-testid="pricing-item-multiplier-input_price"]').setValue('0.3')

    expect(currentForm(wrapper).multiplier_mode).toBe('item')
    expect(currentForm(wrapper).item_multipliers?.input_price).toBe(0.3)
    expect(wrapper.get('[data-testid="pricing-multiplier-inline-input_price"]').text()).toContain('0.09')
  })

  it('applies the shared multiplier to all populated price fields', async () => {
    const wrapper = mountColumn(createForm())

    await wrapper.get('[data-testid="pricing-apply-shared-multiplier"]').trigger('click')

    expect(currentForm(wrapper).multiplier_mode).toBe('item')
    expect(currentForm(wrapper).item_multipliers).toEqual({
      input_price: 0.12,
      output_price: 0.12,
      cache_price: 0.12,
    })
  })
})
