import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OpenAIFastPolicySettingsCard from '../OpenAIFastPolicySettingsCard.vue'
import type { OpenAIFastPolicySettings } from '@/api/admin/settings'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function mountCard(policy: OpenAIFastPolicySettings) {
  return mount(OpenAIFastPolicySettingsCard, {
    props: {
      modelValue: policy,
      'onUpdate:modelValue': vi.fn(),
      enableInjection: false,
      'onUpdate:enableInjection': vi.fn()
    },
    global: {
      stubs: {
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template:
            '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />'
        },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template:
            '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>'
        }
      }
    }
  })
}

describe('OpenAIFastPolicySettingsCard', () => {
  it('normalizes user-scoped Fast/Flex rule user IDs', async () => {
    const policy: OpenAIFastPolicySettings = {
      rules: [
        {
          service_tier: 'fast',
          action: 'filter',
          scope: 'all',
          fallback_action: 'pass',
          model_whitelist: [],
          user_ids: []
        }
      ]
    }
    const wrapper = mountCard(policy)

    const userIdsInput = wrapper.findAll('textarea')[1]
    await userIdsInput.setValue('12, 0\n12\n-4\n35\nabc')

    expect(policy.rules[0].user_ids).toEqual([12, 35])
  })
})
