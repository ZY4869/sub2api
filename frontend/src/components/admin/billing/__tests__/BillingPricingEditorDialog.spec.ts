import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BillingPricingLayerForm, BillingPricingSheetDetail } from '@/api/admin/billing'
import BillingPricingEditorDialog from '../BillingPricingEditorDialog.vue'

const appStoreMock = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

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

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
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
    appStoreMock.showError.mockReset()
    appStoreMock.showSuccess.mockReset()
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

  it('does not expose sale editing controls in the official cost editor', () => {
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

    expect(wrapper.find('[data-testid="sale-column"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="save-layer-sale"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('保存售价')
  })

  it('shares currency state and saves converted source currency values', async () => {
    const wrapper = mountDialog([createDetail()])

    await wrapper.get('[data-testid="pricing-currency-select"]').setValue('CNY')

    expect(wrapper.get('[data-testid="official-column"]').get('[data-testid="pricing-field-unit-input_price"]').text()).toBe('￥ / M Tokens')
    expect(wrapper.find('[data-testid="sale-column"]').exists()).toBe(false)

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

  it('allows cny save when exchange rate is unavailable and keeps the warning visible', async () => {
    exchangeRateStoreMock.exchangeRate = null
    const wrapper = mountDialog([
      createDetail({
        currency: 'CNY',
      }),
    ])

    await nextTick()

    expect(wrapper.find('[data-testid="pricing-currency-alert"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="save-layer-official"]').attributes('disabled')).toBeUndefined()

    await wrapper.get('[data-testid="save-layer-official"]').trigger('click')
    expect(wrapper.emitted('save-layer')?.[0]?.[0]).toEqual(expect.objectContaining({
      currency: 'CNY',
    }))
  })

  it('blocks incomplete tiered pricing before save and shows field-level errors', async () => {
    const wrapper = mountDialog([
      createDetail({
        official_form: createForm({
          tiered_enabled: true,
        }),
      }),
    ])

    await wrapper.get('[data-testid="save-layer-official"]').trigger('click')

    expect(wrapper.emitted('save-layer')).toBeUndefined()
    expect(wrapper.get('[data-testid="pricing-field-error-tier_threshold_tokens"]').text()).toContain('共享阈值必须是正整数')
    expect(wrapper.get('[data-testid="pricing-field-error-input_price_above_threshold"]').text()).toContain('至少填写一个阈值后价格')
    expect(appStoreMock.showError).toHaveBeenCalled()
  })

  it('merges server field errors into the active layer and lets server text override the same field', async () => {
    const wrapper = mount(BillingPricingEditorDialog, {
      props: {
        show: true,
        details: [
          createDetail({
            official_form: createForm({
              tiered_enabled: true,
            }),
          }),
        ],
        activeModel: 'gpt-5.4',
        serverErrors: {
          official: {
            tier_threshold_tokens: '服务端：共享阈值必须是正整数',
            input_price_above_threshold: '服务端：至少填写一个阈值后价格',
          },
        },
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

    await nextTick()

    expect(wrapper.get('[data-testid="pricing-field-error-tier_threshold_tokens"]').text()).toContain('服务端：共享阈值必须是正整数')
    expect(wrapper.get('[data-testid="pricing-field-error-input_price_above_threshold"]').text()).toContain('服务端：至少填写一个阈值后价格')
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

})
