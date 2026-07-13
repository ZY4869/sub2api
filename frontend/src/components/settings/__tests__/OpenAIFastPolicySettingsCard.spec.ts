import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpenAIFastPolicySettingsCard from '../OpenAIFastPolicySettingsCard.vue'
import type { OpenAIFastPolicySettings } from '@/api/admin/settings'

const mockState = vi.hoisted(() => ({
  searchUsers: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      searchUsers: mockState.searchUsers
    }
  }
}))

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
  beforeEach(() => {
    mockState.searchUsers.mockReset()
  })

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

  it('merges searched users with manual user IDs', async () => {
    vi.useFakeTimers()
    mockState.searchUsers.mockResolvedValue([
      { id: 35, email: 'picked@example.com' },
      { id: 44, email: 'fresh@example.com' }
    ])

    const policy: OpenAIFastPolicySettings = {
      rules: [
        {
          service_tier: 'flex',
          action: 'pass',
          scope: 'all',
          fallback_action: 'filter',
          model_whitelist: [],
          user_ids: [12]
        }
      ]
    }
    const wrapper = mountCard(policy)

    const searchInput = wrapper.find('input[type="text"]')
    await searchInput.trigger('focus')
    await searchInput.setValue('picked')
    vi.advanceTimersByTime(300)
    await flushPromises()
    await nextTick()

    const userButtons = wrapper.findAll('button').filter((button) => button.text().includes('picked@example.com'))
    expect(userButtons.length).toBeGreaterThan(0)
    await userButtons[0].trigger('click')

    expect(policy.rules[0].user_ids).toEqual([12, 35])

    const userIdsInput = wrapper.findAll('textarea')[1]
    await userIdsInput.setValue('12\n35\n44\n44\n-1')

    expect(policy.rules[0].user_ids).toEqual([12, 35, 44])
    vi.useRealTimers()
  })
})
