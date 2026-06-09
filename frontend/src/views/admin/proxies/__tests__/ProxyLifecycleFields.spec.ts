import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ProxyLifecycleFields from '../ProxyLifecycleFields.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('ProxyLifecycleFields', () => {
  it('renders expiry inputs and emits lifecycle updates', async () => {
    const wrapper = mount(ProxyLifecycleFields, {
      props: {
        form: {
          expires_at: '',
          expiry_remind_days: 0,
          fallback_proxy_id: null
        },
        fallbackProxyOptions: [
          { value: null, label: 'No fallback' },
          { value: 12, label: 'fallback-proxy' }
        ]
      },
      global: {
        stubs: {
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            template: `
              <select
                data-testid="fallback-select"
                :value="modelValue ?? ''"
                @change="$emit('update:modelValue', $event.target.value)"
              >
                <option
                  v-for="option in options"
                  :key="String(option.value)"
                  :value="option.value ?? ''"
                >
                  {{ option.label }}
                </option>
              </select>
            `
          }
        }
      }
    })

    expect(wrapper.text()).toContain('admin.proxies.expiresAt')
    expect(wrapper.text()).toContain('admin.proxies.expiryRemindDays')
    expect(wrapper.text()).toContain('admin.proxies.fallbackProxy')
    expect(wrapper.text()).toContain('fallback-proxy')

    await wrapper.get('input[type="datetime-local"]').setValue('2026-07-01T09:30')
    await wrapper.get('input[type="number"]').setValue('5')
    await wrapper.get('[data-testid="fallback-select"]').setValue('12')

    expect(wrapper.emitted('update')).toEqual([
      [{ expires_at: '2026-07-01T09:30', expiry_remind_days: 0, fallback_proxy_id: null }],
      [{ expires_at: '', expiry_remind_days: 5, fallback_proxy_id: null }],
      [{ expires_at: '', expiry_remind_days: 0, fallback_proxy_id: 12 }]
    ])
  })
})
