import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BillingPricingLayerForm, BillingPricingSheetDetail } from '@/api/admin/billing'
import BillingBulkDiscountPanel from '../BillingBulkDiscountPanel.vue'
import BillingPricingEditorDialog from '../BillingPricingEditorDialog.vue'

const exchangeRateStoreMock = vi.hoisted(() => ({
  exchangeRate: {
    base: 'USD',
    quote: 'CNY',
    rate: 7.2,
    date: '2026-04-16',
    updated_at: '2026-04-16T00:00:00Z',
    cached: true,
  } as {
    base: string
    quote: string
    rate: number
    date: string
    updated_at: string
    cached: boolean
  } | null,
  loading: false,
  fetchExchangeRate: vi.fn(),
}))

vi.mock('@/stores/exchangeRate', () => ({
  useExchangeRateStore: () => exchangeRateStoreMock,
}))

function createForm(overrides: Partial<BillingPricingLayerForm> = {}): BillingPricingLayerForm {
  return {
    input_price: 2.8e-7,
    output_price: 6e-7,
    cache_price: undefined,
    special_enabled: false,
    special: {
      ...(overrides.special || {}),
    },
    tiered_enabled: false,
    tier_threshold_tokens: undefined,
    input_price_above_threshold: undefined,
    output_price_above_threshold: undefined,
    multiplier_enabled: false,
    multiplier_mode: undefined,
    shared_multiplier: undefined,
    item_multipliers: {},
    ...overrides,
  }
}

function createDetail(overrides: Partial<BillingPricingSheetDetail> = {}): BillingPricingSheetDetail {
  return {
    model: 'gpt-5.4',
    display_name: 'GPT-5.4',
    provider: 'openai',
    mode: 'chat',
    currency: 'USD',
    pricing_status: 'ok',
    pricing_warnings: [],
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
      cache_price: 1e-7,
    }),
    sale_form: createForm({
      input_price: 3e-7,
      output_price: 8e-7,
      cache_price: 1.2e-7,
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
  beforeEach(() => {
    exchangeRateStoreMock.exchangeRate = {
      base: 'USD',
      quote: 'CNY',
      rate: 7.2,
      date: '2026-04-16',
      updated_at: '2026-04-16T00:00:00Z',
      cached: true,
    }
    exchangeRateStoreMock.loading = false
    exchangeRateStoreMock.fetchExchangeRate.mockReset()
    exchangeRateStoreMock.fetchExchangeRate.mockResolvedValue(exchangeRateStoreMock.exchangeRate)
  })

  it('renders compact single-line sections, defaults to USD, and shows alternate currency text', () => {
    const wrapper = mountDialog([createDetail()])
    const officialColumn = wrapper.get('[data-testid="official-column"]')

    expect(wrapper.get('[data-testid="pricing-currency-select"]').element.value).toBe('USD')
    expect(officialColumn.get('[data-testid="pricing-field-unit-input_price"]').text()).toBe('$ / M Tokens')
    expect(officialColumn.get('[data-testid="pricing-field-input_price"]').element).toHaveProperty('value', '0.28')
    expect(officialColumn.get('[data-testid="pricing-field-secondary-input_price"]').text()).toContain('￥')
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
    expect(officialColumn.get('[data-testid="pricing-field-unit-output_price"]').text()).toBe('$ / 张')
    expect(officialColumn.find('[data-testid="pricing-field-input_price"]').exists()).toBe(false)
    expect(officialColumn.find('[data-testid="pricing-tier-toggle"]').exists()).toBe(false)
  })

  it('shows sale special fields when official special pricing is enabled without mutating sale payloads', async () => {
    const wrapper = mountDialog([
      createDetail({
        official_form: createForm({
          special_enabled: true,
          special: {
            grounding_search: 0.12,
          },
        }),
        sale_form: createForm({
          special_enabled: false,
          special: {},
        }),
      }),
    ])

    const saleColumn = wrapper.get('[data-testid="sale-column"]')
    expect(saleColumn.find('[data-testid="pricing-field-grounding_search"]').exists()).toBe(true)
    expect(saleColumn.get('[data-testid="pricing-field-unit-grounding_search"]').text()).toBe('$ / 次')

    await wrapper.get('[data-testid="save-layer-sale"]').trigger('click')

    expect(wrapper.emitted('save-layer')?.[0]).toEqual([
      {
        model: 'gpt-5.4',
        layer: 'sale',
        currency: 'USD',
        form: expect.objectContaining({
          special_enabled: false,
          special: {},
        }),
      },
    ])
  })

  it('shares currency state and saves converted source currency values', async () => {
    const wrapper = mountDialog([createDetail()])

    await wrapper.get('[data-testid="pricing-currency-select"]').setValue('CNY')

    expect(wrapper.get('[data-testid="official-column"]').get('[data-testid="pricing-field-unit-input_price"]').text()).toBe('￥ / M Tokens')
    expect(wrapper.get('[data-testid="sale-column"]').get('[data-testid="pricing-field-unit-input_price"]').text()).toBe('￥ / M Tokens')

    await wrapper.get('[data-testid="save-layer-official"]').trigger('click')

    const payload = wrapper.emitted('save-layer')?.[0]?.[0]
    expect(payload).toEqual(expect.objectContaining({
      model: 'gpt-5.4',
      layer: 'official',
      currency: 'CNY',
    }))
    expect(payload?.form.input_price).toBeCloseTo(2.016e-6)
    expect(payload?.form.output_price).toBeCloseTo(4.32e-6)
  })

  it('marks sale special pricing enabled after editing mirrored special fields', async () => {
    const wrapper = mountDialog([
      createDetail({
        official_form: createForm({
          special_enabled: true,
          special: {
            grounding_search: 0.12,
          },
        }),
        sale_form: createForm({
          special_enabled: false,
          special: {},
        }),
      }),
    ])

    const saleColumn = wrapper.get('[data-testid="sale-column"]')
    await saleColumn.get('[data-testid="pricing-field-grounding_search"]').setValue('0.18')
    await wrapper.get('[data-testid="save-layer-sale"]').trigger('click')

    expect(wrapper.emitted('save-layer')?.[0]).toEqual([
      {
        model: 'gpt-5.4',
        layer: 'sale',
        currency: 'USD',
        form: expect.objectContaining({
          special_enabled: true,
          special: expect.objectContaining({
            grounding_search: 0.18,
          }),
        }),
      },
    ])
  })

  it('preserves sale multiplier config when saving edited item multipliers', async () => {
    const wrapper = mountDialog([
      createDetail({
        sale_form: createForm({
          input_price: 3e-7,
          output_price: 8e-7,
          multiplier_enabled: true,
          multiplier_mode: 'item',
          item_multipliers: {
            input_price: 0.12,
            output_price: 0.15,
          },
        }),
      }),
    ])

    await wrapper.get('[data-testid="pricing-item-multiplier-input_price"]').setValue('0.2')
    await wrapper.get('[data-testid="save-layer-sale"]').trigger('click')

    expect(wrapper.emitted('save-layer')?.[0]).toEqual([
      {
        model: 'gpt-5.4',
        layer: 'sale',
        currency: 'USD',
        form: expect.objectContaining({
          multiplier_enabled: true,
          multiplier_mode: 'item',
          item_multipliers: expect.objectContaining({
            input_price: 0.2,
            output_price: 0.15,
          }),
        }),
      },
    ])
  })

  it('blocks cny save when exchange rate is unavailable', async () => {
    exchangeRateStoreMock.exchangeRate = null
    const wrapper = mountDialog([
      createDetail({
        currency: 'CNY',
      }),
    ])

    await nextTick()

    expect(wrapper.find('[data-testid="pricing-currency-alert"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="save-layer-official"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="save-layer-sale"]').attributes('disabled')).toBeDefined()
  })

  it('shows conflict badges and audit warnings for non-ok models', () => {
    const wrapper = mountDialog([
      createDetail({
        pricing_status: 'conflict',
        pricing_warnings: ['aliases identifier "gpt-5.4" collides with 2 models'],
      }),
    ])

    expect(wrapper.text()).toContain('冲突')
    expect(wrapper.text()).toContain('当前模型定价审计存在提示')
    expect(wrapper.text()).toContain('collides with 2 models')
  })

  it('emits workset discount payloads with selected sale field ids', async () => {
    const wrapper = mountDialog([
      createDetail(),
      createDetail({
        model: 'claude-sonnet-4.5',
        display_name: 'Claude Sonnet 4.5',
        provider: 'anthropic',
        sale_form: createForm({
          input_price: 2e-7,
          output_price: 3e-7,
          cache_price: 4e-8,
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
