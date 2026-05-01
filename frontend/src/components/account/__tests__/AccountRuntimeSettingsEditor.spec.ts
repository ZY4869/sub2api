import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountRuntimeSettingsEditor from '../AccountRuntimeSettingsEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const proxySelectorStub = {
  name: 'ProxySelector',
  props: ['modelValue', 'proxies'],
  emits: ['update:modelValue'],
  template:
    '<button type="button" data-testid="proxy-selector" @click="$emit(\'update:modelValue\', 3)">proxy</button>'
}

describe('AccountRuntimeSettingsEditor', () => {
  it('updates models and normalizes numeric inputs', async () => {
    const wrapper = mount(AccountRuntimeSettingsEditor, {
      props: {
        proxies: [],
        proxyId: null,
        concurrency: 1,
        loadFactor: null,
        priority: 1,
        rateMultiplier: 1,
        expiresAtInput: ''
      },
      global: {
        stubs: {
          ProxySelector: proxySelectorStub
        }
      }
    })

    await wrapper.get('[data-testid="proxy-selector"]').trigger('click')

    const inputs = wrapper.findAll('input[type="number"]')
    await inputs[0].setValue('0')
    await inputs[1].setValue('0')
    await inputs[2].setValue('4')
    await inputs[3].setValue('1.5')

    const checkbox = wrapper.find('input[type="checkbox"]')
    await checkbox.setValue(true)

    const datetimeInput = wrapper.find('input[type="datetime-local"]')
    await datetimeInput.setValue('2026-03-14T12:00')

    expect(wrapper.emitted('update:proxyId')?.[0]).toEqual([3])
    expect(wrapper.emitted('update:concurrency')?.at(-1)).toEqual([1])
    expect(wrapper.emitted('update:loadFactor')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:priority')?.[0]).toEqual([4])
    expect(wrapper.emitted('update:rateMultiplier')?.[0]).toEqual([1.5])
    expect(wrapper.emitted('update:expiresAtInput')?.at(-1)).toEqual(['2026-03-14T12:00'])
  })

  it('enables expiration with a default one month value and can switch to one year', async () => {
    const wrapper = mount(AccountRuntimeSettingsEditor, {
      props: {
        proxies: [],
        proxyId: null,
        concurrency: 1,
        loadFactor: null,
        priority: 1,
        rateMultiplier: 1,
        expiresAtInput: ''
      },
      global: {
        stubs: {
          ProxySelector: proxySelectorStub
        }
      }
    })

    const checkbox = wrapper.find('input[type="checkbox"]')
    await checkbox.setValue(true)

    const initialExpiry = wrapper.emitted('update:expiresAtInput')?.at(-1)?.[0]
    expect(typeof initialExpiry).toBe('string')
    expect(String(initialExpiry)).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/)

    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    const monthExpiry = wrapper.emitted('update:expiresAtInput')?.at(-1)?.[0]

    await buttons[1].trigger('click')
    const yearExpiry = wrapper.emitted('update:expiresAtInput')?.at(-1)?.[0]

    expect(String(monthExpiry)).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/)
    expect(String(yearExpiry)).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/)
    expect(wrapper.text()).toContain('admin.accounts.expiresAtPreview')
  })

  it('clears expiration when toggled off from an existing value', async () => {
    const wrapper = mount(AccountRuntimeSettingsEditor, {
      props: {
        proxies: [],
        proxyId: null,
        concurrency: 1,
        loadFactor: null,
        priority: 1,
        rateMultiplier: 1,
        expiresAtInput: '2026-06-01T12:30'
      },
      global: {
        stubs: {
          ProxySelector: proxySelectorStub
        }
      }
    })

    const checkbox = wrapper.find('input[type="checkbox"]')
    await checkbox.setValue(false)

    expect(wrapper.emitted('update:expiresAtInput')?.at(-1)).toEqual([''])
  })
})
