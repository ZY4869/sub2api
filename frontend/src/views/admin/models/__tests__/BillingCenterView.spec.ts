import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import BillingCenterView from '../BillingCenterView.vue'

const apiMocks = vi.hoisted(() => ({
  getBillingCenter: vi.fn(),
  updateBillingSheet: vi.fn(),
  deleteBillingSheet: vi.fn(),
  copyBillingSheetOfficialToSale: vi.fn(),
  updateBillingRule: vi.fn(),
  deleteBillingRule: vi.fn(),
  simulateBilling: vi.fn()
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string, params?: Record<string, string>) => {
        if (!params) return key
        return `${key}:${JSON.stringify(params)}`
      }
    })
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: storeMocks.showError,
    showSuccess: storeMocks.showSuccess
  })
}))

vi.mock('@/api/admin/models', () => ({
  getBillingCenter: apiMocks.getBillingCenter,
  updateBillingSheet: apiMocks.updateBillingSheet,
  deleteBillingSheet: apiMocks.deleteBillingSheet,
  copyBillingSheetOfficialToSale: apiMocks.copyBillingSheetOfficialToSale,
  updateBillingRule: apiMocks.updateBillingRule,
  deleteBillingRule: apiMocks.deleteBillingRule,
  simulateBilling: apiMocks.simulateBilling
}))

function mountView() {
  return mount(BillingCenterView)
}

describe('BillingCenterView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.getBillingCenter.mockResolvedValue({
      sheets: [
        {
          id: 'gemini-2.5-pro',
          provider: 'gemini',
          model: 'gemini-2.5-pro',
          display_name: 'Gemini 2.5 Pro',
          official_matrix: {
            surfaces: ['native'],
            service_tiers: ['standard'],
            charge_slots: ['text_input', 'text_output'],
            rows: [
              {
                surface: 'native',
                service_tier: 'standard',
                slots: {
                  text_input: { price: 0.1, rule_id: 'rule-official-input' },
                  text_output: { price: 0.2, rule_id: 'rule-official-output' }
                }
              }
            ]
          },
          sale_matrix: {
            surfaces: ['native'],
            service_tiers: ['standard'],
            charge_slots: ['text_input', 'text_output'],
            rows: [
              {
                surface: 'native',
                service_tier: 'standard',
                slots: {
                  text_input: { price: 0.2, rule_id: 'rule-sale-input' },
                  text_output: { price: 0.3, rule_id: 'rule-sale-output' }
                }
              }
            ]
          },
          supports_service_tier: true
        }
      ],
      rules: [
        {
          id: 'rule-1',
          provider: 'gemini',
          layer: 'sale',
          surface: 'native',
          operation_type: 'generate_content',
          service_tier: '',
          batch_mode: 'any',
          matchers: {},
          unit: 'input_token',
          price: 0.2,
          priority: 1,
          enabled: true
        }
      ]
    })
    apiMocks.simulateBilling.mockResolvedValue({
      classification: {
        surface: 'native',
        operation_type: 'generate_content',
        service_tier: 'standard',
        batch_mode: 'realtime',
        input_modality: 'text',
        output_modality: 'text'
      },
      matched_rules: [
        {
          id: 'rule-1',
          provider: 'gemini',
          layer: 'sale',
          surface: 'native',
          operation_type: 'generate_content',
          service_tier: 'standard',
          batch_mode: 'realtime',
          unit: 'input_token',
          price: 0.0012,
          priority: 1,
          matchers: {}
        }
      ],
      total_cost: 1.23,
      actual_cost: 1.23,
      lines: [
        {
          charge_slot: 'text_input',
          unit: 'input_token',
          units: 1024,
          price: 0.0012,
          cost: 1.23,
          actual_cost: 1.23
        }
      ],
      unmatched_demands: [
        {
          charge_slot: 'grounding_search_request',
          unit: 'grounding_search_request',
          units: 1,
          reason: 'grounding_kind_miss',
          missing_dimensions: ['grounding_kind']
        }
      ],
      fallback: {
        policy: 'legacy_model_pricing',
        applied: true,
        reason: 'no_billing_rule_match',
        derived_from: 'billing_service_pricing',
        cost_lines: [
          {
            charge_slot: 'text_input',
            unit: 'input_token',
            units: 1024,
            price: 0.0012,
            cost: 1.23,
            actual_cost: 1.23
          }
        ]
      }
    })
  })

  it('loads billing center data on mount', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(apiMocks.getBillingCenter).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Gemini 2.5 Pro')
    expect(wrapper.text()).toContain('Generate Content')
    expect(wrapper.text()).toContain('admin.models.pages.billing.matrixBadge')
  })

  it('runs the billing simulator and renders structured result sections', async () => {
    const wrapper = mountView()
    await flushPromises()

    const searchButton = wrapper.findAll('button').find((button) => button.text() === 'common.search')
    expect(searchButton).toBeTruthy()

    await searchButton!.trigger('click')
    await flushPromises()

    expect(apiMocks.simulateBilling).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('1.2300')
    expect(wrapper.text()).toContain('admin.models.pages.billing.classificationResultTitle')
    expect(wrapper.text()).toContain('admin.models.pages.billing.matchedRulesTitle')
    expect(wrapper.text()).toContain('admin.models.pages.billing.unmatchedDemandsTitle')
    expect(wrapper.text()).toContain('admin.models.pages.billing.fallbackTitle')
  })
})
