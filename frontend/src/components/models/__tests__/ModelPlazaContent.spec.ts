import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ModelPlazaContent from '../ModelPlazaContent.vue'

const mocks = vi.hoisted(() => ({
  getModelPlaza: vi.fn(),
  showError: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/modelPlaza', () => ({
  getModelPlaza: mocks.getModelPlaza
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mocks.showError
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: mocks.copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const labels: Record<string, string> = {
    'ui.modelPlaza.eyebrow': 'Model plaza',
    'ui.modelPlaza.title': 'Available models',
    'ui.modelPlaza.description': 'Projected from local availability',
    'ui.modelPlaza.groups': 'Groups {count}',
    'ui.modelPlaza.models': 'Models {count}',
    'ui.modelPlaza.refreshing': 'Refreshing',
    'ui.modelPlaza.refresh': 'Refresh',
    'ui.modelPlaza.searchPlaceholder': 'Search models',
    'ui.modelPlaza.allAccess': 'All access',
    'ui.modelPlaza.publicAccess': 'Public',
    'ui.modelPlaza.exclusiveAccess': 'Exclusive',
    'ui.modelPlaza.allPlatforms': 'All platforms',
    'ui.modelPlaza.empty': 'No models',
    'ui.modelPlaza.exclusive': 'Exclusive group',
    'ui.modelPlaza.public': 'Public group',
    'ui.modelPlaza.subscription': 'Subscription',
    'ui.modelPlaza.rate': 'Rate',
    'ui.modelPlaza.peak': 'Peak',
    'ui.modelPlaza.independentImages': 'Independent images',
    'ui.modelPlaza.webSearch': 'Web search',
    'ui.modelPlaza.copyModelId': 'Copy model ID',
    'ui.modelPlaza.copySuccess': 'Copied {model}',
    'ui.modelPlaza.pricingUnavailable': 'Pricing unavailable',
    'ui.modelPlaza.input': 'Input',
    'ui.modelPlaza.output': 'Output',
    'ui.modelPlaza.cacheWrite': 'Cache write',
    'ui.modelPlaza.cacheRead': 'Cache read',
    'ui.modelPlaza.imageOutput': 'Image output',
    'ui.modelPlaza.request': 'Request',
    'ui.modelPlaza.perMillion': 'per million',
    'ui.modelPlaza.loadFailed': 'Load failed',
    'common.unknownError': 'Unknown error'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, values?: Record<string, unknown>) => {
        let text = labels[key] || key
        for (const [name, value] of Object.entries(values || {})) {
          text = text.replace(`{${name}}`, String(value))
        }
        return text
      }
    })
  }
})
vi.mock('@/components/common/ModelIcon.vue', () => ({
  default: { template: '<span data-testid="model-icon" />' }
}))

vi.mock('@/components/common/ModelPlatformIcon.vue', () => ({
  default: { template: '<span data-testid="platform-icon" />' }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { props: ['name', 'size'], template: '<span />' }
}))

describe('ModelPlazaContent', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getModelPlaza.mockResolvedValue({
      description: 'snapshot',
      groups: [
        {
          id: 1,
          name: 'OpenAI public',
          description: 'Public group',
          platform: 'openai',
          subscription_type: 'standard',
          rate_multiplier: 1,
          peak_rate_enabled: false,
          peak_start: '',
          peak_end: '',
          peak_rate_multiplier: 1,
          is_exclusive: false,
          image_rate_independent: true,
          image_rate_multiplier: 1,
          image_price_1k: 0.02,
          image_price_2k: null,
          image_price_4k: null,
          web_search_price_per_call: 0.03,
          models: [
            {
              display_model_id: 'gpt-5.6-luna',
              platform: 'openai',
              pricing: { billing_mode: 'token', input_price: 0.000001, output_price: 0.000002 },
              official_pricing: { billing_mode: 'token', input_price: 0.000001, output_price: 0.000002 }
            }
          ]
        },
        {
          id: 2,
          name: 'Anthropic exclusive',
          description: 'Private group',
          platform: 'anthropic',
          subscription_type: 'subscription',
          rate_multiplier: 1.2,
          peak_rate_enabled: true,
          peak_start: '09:00',
          peak_end: '18:00',
          peak_rate_multiplier: 1.5,
          is_exclusive: true,
          image_rate_independent: false,
          image_rate_multiplier: 1,
          image_price_1k: null,
          image_price_2k: null,
          image_price_4k: null,
          web_search_price_per_call: null,
          models: [
            {
              display_model_id: 'claude-sonnet-5',
              platform: 'anthropic',
              pricing: null,
              official_pricing: null
            }
          ]
        }
      ]
    })
  })

  it('renders display model IDs without target model IDs and supports copy', async () => {
    const wrapper = mount(ModelPlazaContent)
    await flushPromises()

    expect(mocks.getModelPlaza).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('gpt-5.6-luna')
    expect(wrapper.text()).toContain('claude-sonnet-5')
    expect(wrapper.html()).not.toContain('target_model_id')

    const copyButton = wrapper.findAll('button')
      .find((button) => button.attributes('title') === 'Copy model ID')
    expect(copyButton).toBeTruthy()
    await copyButton!.trigger('click')

    expect(mocks.copyToClipboard).toHaveBeenCalledWith('gpt-5.6-luna', 'Copied gpt-5.6-luna')
  })

  it('filters by access mode and search text', async () => {
    const wrapper = mount(ModelPlazaContent)
    await flushPromises()

    const exclusiveButton = wrapper.findAll('button')
      .find((button) => button.text() === 'Exclusive')
    expect(exclusiveButton).toBeTruthy()
    await exclusiveButton!.trigger('click')

    expect(wrapper.text()).toContain('claude-sonnet-5')
    expect(wrapper.text()).not.toContain('gpt-5.6-luna')

    await wrapper.get('input[type="search"]').setValue('luna')
    expect(wrapper.text()).toContain('No models')
  })

  it('shows a soft error when loading fails', async () => {
    mocks.getModelPlaza.mockRejectedValueOnce(new Error('network down'))
    const wrapper = mount(ModelPlazaContent)
    await flushPromises()

    expect(wrapper.text()).toContain('Load failed: network down')
    expect(mocks.showError).toHaveBeenCalledWith('Load failed: network down')
  })
})
