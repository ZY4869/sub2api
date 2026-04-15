import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import type { BillingPricingLayerForm, BillingPricingSheetDetail } from '@/api/admin/billing'
import BillingBulkDiscountPanel from '../BillingBulkDiscountPanel.vue'
import BillingPricingEditorDialog from '../BillingPricingEditorDialog.vue'

function createForm(overrides: Partial<BillingPricingLayerForm> = {}): BillingPricingLayerForm {
  return {
    input_price: 1,
    output_price: 2,
    cache_price: undefined,
    special_enabled: false,
    special: {
      ...(overrides.special || {}),
    },
    tiered_enabled: false,
    tier_threshold_tokens: undefined,
    input_price_above_threshold: undefined,
    output_price_above_threshold: undefined,
    ...overrides,
  }
}

function createDetail(overrides: Partial<BillingPricingSheetDetail> = {}): BillingPricingSheetDetail {
  return {
    model: 'gpt-5.4',
    display_name: 'GPT-5.4',
    provider: 'openai',
    mode: 'chat',
    input_supported: true,
    output_charge_slot: 'text_output',
    supports_prompt_caching: true,
    supports_service_tier: false,
    long_context_input_token_threshold: 200000,
    long_context_input_cost_multiplier: 2,
    long_context_output_cost_multiplier: 2,
    capabilities: {
      supports_tiered_pricing: true,
      supports_batch_pricing: true,
      supports_service_tier: false,
      supports_prompt_caching: true,
      supports_provider_special: true,
    },
    official_form: createForm({
      input_price: 1,
      output_price: 2,
      cache_price: 0.1,
    }),
    sale_form: createForm({
      input_price: 1.5,
      output_price: 2.5,
      cache_price: 0.2,
    }),
    ...overrides,
  }
}

function mountDialog(details: BillingPricingSheetDetail[]) {
  return mount(BillingPricingEditorDialog, {
    props: {
      show: true,
      details,
      activeModel: details[0]?.model || '',
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /></div>',
        },
      },
    },
  })
}

describe('BillingPricingEditorDialog', () => {
  it('renders simplified sections and removes the old raw advanced controls', () => {
    const wrapper = mountDialog([createDetail()])
    const officialColumn = wrapper.get('[data-testid="official-column"]')

    expect(officialColumn.find('[data-testid="pricing-field-input_price"]').exists()).toBe(true)
    expect(officialColumn.find('[data-testid="pricing-field-output_price"]').exists()).toBe(true)
    expect(officialColumn.find('[data-testid="pricing-field-cache_price"]').exists()).toBe(true)
    expect(officialColumn.find('[data-testid="pricing-special-toggle"]').exists()).toBe(true)
    expect(officialColumn.find('[data-testid="pricing-tier-toggle"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Surface')
    expect(wrapper.text()).not.toContain('Provider Special')
  })

  it('uses dynamic output mapping and hides input and tier fields for image models', () => {
    const wrapper = mountDialog([
      createDetail({
        model: 'gpt-image-1',
        display_name: 'GPT Image 1',
        mode: 'image',
        input_supported: false,
        output_charge_slot: 'image_output',
        capabilities: {
          supports_tiered_pricing: true,
          supports_batch_pricing: false,
          supports_service_tier: false,
          supports_prompt_caching: false,
          supports_provider_special: false,
        },
        official_form: createForm({
          input_price: undefined,
          output_price: 0.08,
          cache_price: undefined,
        }),
        sale_form: createForm({
          input_price: undefined,
          output_price: 0.1,
          cache_price: undefined,
        }),
      }),
    ])

    const officialColumn = wrapper.get('[data-testid="official-column"]')
    expect(officialColumn.text()).toContain('图片输出定价')
    expect(officialColumn.find('[data-testid="pricing-field-input_price"]').exists()).toBe(false)
    expect(officialColumn.find('[data-testid="pricing-tier-toggle"]').exists()).toBe(false)
  })

  it('emits simplified form payloads when saving a layer', async () => {
    const wrapper = mountDialog([createDetail()])
    const officialColumn = wrapper.get('[data-testid="official-column"]')

    await officialColumn.get('[data-testid="pricing-field-input_price"]').setValue('1.8')
    await officialColumn.get('[data-testid="pricing-special-toggle"]').trigger('click')
    await nextTick()
    await officialColumn.get('[data-testid="pricing-field-batch_input_price"]').setValue('0.9')
    await wrapper.get('[data-testid="save-layer-official"]').trigger('click')

    expect(wrapper.emitted('save-layer')).toEqual([
      [
        {
          model: 'gpt-5.4',
          layer: 'official',
          form: expect.objectContaining({
            input_price: 1.8,
            output_price: 2,
            cache_price: 0.1,
            special_enabled: true,
            special: expect.objectContaining({
              batch_input_price: 0.9,
            }),
          }),
        },
      ],
    ])
  })

  it('emits workset discount payloads with selected sale field ids', async () => {
    const wrapper = mountDialog([
      createDetail(),
      createDetail({
        model: 'claude-sonnet-4.5',
        display_name: 'Claude Sonnet 4.5',
        provider: 'anthropic',
        sale_form: createForm({
          input_price: 2,
          output_price: 3,
          cache_price: 0.4,
        }),
      }),
    ])

    wrapper.getComponent(BillingBulkDiscountPanel).vm.$emit('update:scope', 'workset')
    await nextTick()
    await wrapper.get('[data-testid="sale-column"]').get('[data-testid="field-select-input_price"] input').setValue(true)
    wrapper.getComponent(BillingBulkDiscountPanel).vm.$emit('apply-selected')
    await nextTick()

    expect(wrapper.emitted('apply-discount')).toEqual([
      [
        {
          models: ['gpt-5.4', 'claude-sonnet-4.5'],
          itemIds: ['input_price'],
          discountRatio: 0.9,
        },
      ],
    ])
  })
})
