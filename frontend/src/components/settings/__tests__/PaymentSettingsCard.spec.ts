import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PaymentSettingsCard from '../PaymentSettingsCard.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.n ? `${key} ${params.n}` : key
    })
  }
})

function mountCard() {
  return mount(PaymentSettingsCard, {
    props: {
      enabled: true,
      'onUpdate:enabled': vi.fn(),
      purchaseUrl: '',
      'onUpdate:purchaseUrl': vi.fn(),
      airwallexEnabled: false,
      'onUpdate:airwallexEnabled': vi.fn(),
      airwallexEnv: 'demo',
      'onUpdate:airwallexEnv': vi.fn(),
      airwallexClientId: '',
      'onUpdate:airwallexClientId': vi.fn(),
      airwallexApiKey: '',
      'onUpdate:airwallexApiKey': vi.fn(),
      airwallexWebhookSecret: '',
      'onUpdate:airwallexWebhookSecret': vi.fn(),
      mobileForceQrcodeEnabled: false,
      'onUpdate:mobileForceQrcodeEnabled': vi.fn(),
      allowedCurrencies: ['USD', 'CNY'],
      'onUpdate:allowedCurrencies': vi.fn(),
      defaultCurrency: 'USD',
      'onUpdate:defaultCurrency': vi.fn(),
      minTopupAmount: 1,
      'onUpdate:minTopupAmount': vi.fn(),
      maxTopupAmount: 5000,
      'onUpdate:maxTopupAmount': vi.fn(),
      subscriptionPlans: [],
      'onUpdate:subscriptionPlans': vi.fn(),
      antigravityUserAgentVersion: '',
      'onUpdate:antigravityUserAgentVersion': vi.fn(),
      codexOauthUserAgentMode: 'default',
      'onUpdate:codexOauthUserAgentMode': vi.fn(),
      codexOauthUserAgentOverride: '',
      'onUpdate:codexOauthUserAgentOverride': vi.fn(),
      apiKeyConfigured: true,
      webhookSecretConfigured: true,
      effectiveEnabled: false
    },
    global: {
      stubs: {
        Icon: { template: '<span />' },
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />'
        },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>'
        }
      }
    }
  })
}

describe('PaymentSettingsCard', () => {
  it('normalizes allowed currencies from comma separated input', async () => {
    const wrapper = mountCard()
    const currencyInput = wrapper.findAll('input').find((input) => input.element.value === 'USD, CNY')

    await currencyInput!.setValue('usd, hkd, eur')

    expect(wrapper.emitted('update:allowedCurrencies')?.at(-1)).toEqual([['USD', 'HKD', 'EUR']])
  })

  it('shows the configured effective public status', async () => {
    const wrapper = mountCard()

    expect(wrapper.text()).toContain('admin.settings.purchase.effectiveDisabled')

    await wrapper.setProps({ effectiveEnabled: true })
    expect(wrapper.text()).toContain('admin.settings.purchase.effectiveEnabled')
  })

  it('adds a subscription plan and parses price pairs', async () => {
    const wrapper = mountCard()

    await wrapper.find('button.btn-sm').trigger('click')
    await wrapper.setProps({ subscriptionPlans: [{
      plan_id: '',
      name: '',
      group_id: 0,
      validity_days: 30,
      prices_by_currency: {},
      enabled: true
    }] })

    const priceInput = wrapper.findAll('input').find((input) =>
      input.attributes('placeholder') === 'admin.settings.purchase.pricesPlaceholder'
    )
    await priceInput!.setValue('usd:12.5, hkd:98')

    const plan = (wrapper.props('subscriptionPlans') as Array<{ prices_by_currency: Record<string, number> }>)[0]
    expect(plan.prices_by_currency).toEqual({ USD: 12.5, HKD: 98 })
  })
})
