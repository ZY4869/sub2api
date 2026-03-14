import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountAutoPauseToggle from '../AccountAutoPauseToggle.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountAutoPauseToggle', () => {
  it('emits enabled update when toggled', async () => {
    const wrapper = mount(AccountAutoPauseToggle, {
      props: {
        enabled: false
      }
    })

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('update:enabled')).toEqual([[true]])
  })
})
