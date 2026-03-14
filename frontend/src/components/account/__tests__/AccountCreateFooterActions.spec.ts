import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountCreateFooterActions from '../AccountCreateFooterActions.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountCreateFooterActions', () => {
  it('renders step 1 actions and emits close plus model updates', async () => {
    const wrapper = mount(AccountCreateFooterActions, {
      props: {
        step: 1,
        submitting: false,
        isOAuthFlow: true,
        isManualInputMethod: true,
        currentOAuthLoading: false,
        canExchangeCode: false,
        autoImportModels: false
      }
    })

    expect(wrapper.text()).toContain('common.next')

    await wrapper.get('input[type="checkbox"]').setValue(true)
    await wrapper.get('.btn-secondary').trigger('click')

    expect(wrapper.emitted('update:autoImportModels')).toEqual([[true]])
    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('renders step 2 actions and emits back/exchange', async () => {
    const wrapper = mount(AccountCreateFooterActions, {
      props: {
        step: 2,
        submitting: false,
        isOAuthFlow: true,
        isManualInputMethod: true,
        currentOAuthLoading: false,
        canExchangeCode: true,
        autoImportModels: true
      }
    })

    expect(wrapper.text()).toContain('common.back')
    expect(wrapper.text()).toContain('admin.accounts.oauth.completeAuth')

    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    await buttons[1].trigger('click')

    expect(wrapper.emitted('back')).toEqual([[]])
    expect(wrapper.emitted('exchangeCode')).toEqual([[]])
  })
})
