import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountTestModelSelectionFields from '../AccountTestModelSelectionFields.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.testModelTargetRelation') {
          return `-> ${params?.target || ''}`
        }
        if (key === 'admin.accounts.testModelAvailability.verified') {
          return 'Verified'
        }
        if (key === 'admin.accounts.testModelAvailability.unavailable') {
          return 'Unavailable'
        }
        if (key === 'admin.accounts.testModelAvailability.unknown') {
          return 'Unknown'
        }
        if (key === 'admin.accounts.testModelStale.fresh') {
          return 'Fresh'
        }
        if (key === 'admin.accounts.testModelStale.stale') {
          return 'Stale'
        }
        if (key === 'admin.accounts.testModelStale.unverified') {
          return 'Unverified'
        }
        return key
      }
    })
  }
})

function mountFields(availableModels: Array<Record<string, unknown>>, selectedModelKey: string) {
  return mount(AccountTestModelSelectionFields, {
    props: {
      availableModels,
      modelInputMode: 'catalog',
      selectedModelKey,
      manualModelId: ''
    } as any,
    global: {
      stubs: {
        Select: {
          props: ['modelValue', 'options'],
          template: `
            <div>
              <div data-test="selected-option">
                <slot name="selected" :option="options.find((opt) => opt.key === modelValue) || null" />
              </div>
              <div v-for="option in options" :key="option.key" data-test="option">
                <slot name="option" :option="option" :selected="option.key === modelValue" />
              </div>
            </div>
          `
        },
        ModelIcon: true,
        Icon: true
      }
    }
  })
}

describe('AccountTestModelSelectionFields', () => {
  it('shows alias display id, target relation, and separate availability/stale badges', () => {
    const wrapper = mountFields(
      [
        {
          id: 'friendly-sonnet',
          display_name: 'Friendly Sonnet',
          target_model_id: 'claude-sonnet-4.5',
          source_protocol: 'anthropic',
          availability_state: 'verified',
          stale_state: 'fresh'
        }
      ],
      'anthropic::friendly-sonnet'
    )

    expect(wrapper.text()).toContain('Friendly Sonnet')
    expect(wrapper.text()).toContain('friendly-sonnet')
    expect(wrapper.text()).toContain('-> claude-sonnet-4.5')

    const availabilityBadges = wrapper.findAll('[data-test="model-availability-badge"]')
    const staleBadges = wrapper.findAll('[data-test="model-stale-badge"]')
    expect(availabilityBadges[0]?.text()).toBe('Verified')
    expect(staleBadges[0]?.text()).toBe('Fresh')
  })

  it('does not render a fake target relation for direct models', () => {
    const wrapper = mountFields(
      [
        {
          id: 'gpt-5.4',
          display_name: 'GPT-5.4',
          target_model_id: 'gpt-5.4',
          availability_state: 'unknown',
          stale_state: 'unverified'
        }
      ],
      'default::gpt-5.4'
    )

    expect(wrapper.text()).toContain('gpt-5.4')
    expect(wrapper.find('[data-test="model-target-relation"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-test="model-availability-badge"]')[0]?.text()).toBe('Unknown')
    expect(wrapper.findAll('[data-test="model-stale-badge"]')[0]?.text()).toBe('Unverified')
  })
})
