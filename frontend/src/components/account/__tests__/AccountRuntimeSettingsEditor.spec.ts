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

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('0')
    await inputs[1].setValue('0')
    await inputs[2].setValue('4')
    await inputs[3].setValue('1.5')
    await inputs[4].setValue('2026-03-14T12:00')

    expect(wrapper.emitted('update:proxyId')?.[0]).toEqual([3])
    expect(wrapper.emitted('update:concurrency')?.at(-1)).toEqual([1])
    expect(wrapper.emitted('update:loadFactor')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:priority')?.[0]).toEqual([4])
    expect(wrapper.emitted('update:rateMultiplier')?.[0]).toEqual([1.5])
    expect(wrapper.emitted('update:expiresAtInput')?.[0]).toEqual(['2026-03-14T12:00'])
  })
})
