import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import type { BillingPricingSheetDetail } from '@/api/admin/billing'
import BillingBulkDiscountPanel from '../BillingBulkDiscountPanel.vue'
import BillingPriceColumn from '../BillingPriceColumn.vue'
import BillingPricingEditorDialog from '../BillingPricingEditorDialog.vue'

function createDetail(overrides: Partial<BillingPricingSheetDetail> = {}): BillingPricingSheetDetail {
  return {
    model: 'gpt-5.4',
    display_name: 'GPT-5.4',
    provider: 'openai',
    mode: 'chat',
    supports_prompt_caching: false,
    supports_service_tier: true,
    long_context_input_token_threshold: 200000,
    long_context_input_cost_multiplier: 2,
    long_context_output_cost_multiplier: 2,
    capabilities: {
      supports_tiered_pricing: true,
      supports_batch_pricing: true,
      supports_service_tier: true,
      supports_prompt_caching: false,
      supports_provider_special: true,
    },
    official_items: [
      {
        id: 'official-input',
        charge_slot: 'text_input',
        unit: 'input_token',
        layer: 'official',
        mode: 'base',
        price: 1,
        enabled: true,
      },
      {
        id: 'official-output',
        charge_slot: 'text_output',
        unit: 'output_token',
        layer: 'official',
        mode: 'base',
        price: 2,
        enabled: true,
      },
    ],
    sale_items: [
      {
        id: 'sale-1',
        charge_slot: 'text_input',
        unit: 'input_token',
        layer: 'sale',
        mode: 'base',
        price: 1.5,
        enabled: true,
      },
    ],
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
  it('shows preset actions only for the active model capabilities', () => {
    const wrapper = mountDialog([
      createDetail({
        capabilities: {
          supports_tiered_pricing: true,
          supports_batch_pricing: false,
          supports_service_tier: true,
          supports_prompt_caching: false,
          supports_provider_special: true,
        },
      }),
    ])

    expect(wrapper.text()).toContain('启用阶梯')
    expect(wrapper.text()).toContain('启用层级')
    expect(wrapper.text()).toContain('Provider Special')
    expect(wrapper.text()).not.toContain('启用 Batch')
    expect(wrapper.text()).not.toContain('启用缓存')
  })

  it('dedupes generated preset rows when batch preset is applied repeatedly', async () => {
    const wrapper = mountDialog([createDetail()])

    const batchButton = wrapper.findAll('button').find((button) => button.text() === '启用 Batch')
    expect(batchButton).toBeTruthy()
    const officialColumn = wrapper.findAllComponents(BillingPriceColumn)[0]

    await batchButton!.trigger('click')
    await nextTick()
    const firstPassCount = officialColumn.findAll('article').length

    await batchButton!.trigger('click')
    await nextTick()

    expect(firstPassCount).toBe(4)
    expect(officialColumn.findAll('article')).toHaveLength(4)
  })

  it('hides advanced matcher inputs for simple base pricing rows', () => {
    const wrapper = mountDialog([createDetail()])

    const officialColumn = wrapper.findAllComponents(BillingPriceColumn)[0]
    expect(officialColumn.text()).not.toContain('Surface')
    expect(officialColumn.text()).not.toContain('Operation')
    expect(officialColumn.text()).not.toContain('Input Modality')
  })

  it('emits workset discount payloads with the selected sale item ids', async () => {
    const wrapper = mountDialog([
      createDetail(),
      createDetail({
        model: 'claude-sonnet-4.5',
        display_name: 'Claude Sonnet 4.5',
        provider: 'anthropic',
        sale_items: [
          {
            id: 'sale-2',
            charge_slot: 'text_output',
            unit: 'output_token',
            layer: 'sale',
            mode: 'base',
            price: 2.5,
            enabled: true,
          },
        ],
      }),
    ])

    wrapper.getComponent(BillingBulkDiscountPanel).vm.$emit('update:scope', 'workset')
    await nextTick()
    wrapper.findAllComponents(BillingPriceColumn)[1].vm.$emit('toggle-select', 'sale-1')
    await nextTick()
    wrapper.getComponent(BillingBulkDiscountPanel).vm.$emit('apply-selected')
    await nextTick()

    expect(wrapper.emitted('apply-discount')).toEqual([
      [
        {
          models: ['gpt-5.4', 'claude-sonnet-4.5'],
          itemIds: ['sale-1'],
          discountRatio: 0.9,
        },
      ],
    ])
  })
})
