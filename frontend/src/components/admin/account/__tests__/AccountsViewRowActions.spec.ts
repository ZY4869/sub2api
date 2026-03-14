import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountsViewRowActions from '../AccountsViewRowActions.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountsViewRowActions', () => {
  it('emits edit, delete and more actions', async () => {
    const wrapper = mount(AccountsViewRowActions)
    const buttons = wrapper.findAll('button')

    await buttons[0].trigger('click')
    await buttons[1].trigger('click')
    await buttons[2].trigger('click')

    expect(wrapper.emitted('edit')).toEqual([[]])
    expect(wrapper.emitted('delete')).toEqual([[]])
    expect(wrapper.emitted('more')).toHaveLength(1)
  })
})
