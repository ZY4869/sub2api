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
  it('renders step 1 actions without the auto-import toggle and emits close', async () => {
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
    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(false)

    await wrapper.get('.btn-secondary').trigger('click')

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

  it('supports the generic complete authorization action', async () => {
    const disabledWrapper = mount(AccountCreateFooterActions, {
      props: {
        step: 2,
        submitting: false,
        isOAuthFlow: true,
        isManualInputMethod: false,
        currentOAuthLoading: false,
        canExchangeCode: false,
        showCompleteAuthAction: true,
        canCompleteAuth: false,
        autoImportModels: false
      }
    })

    const disabledButton = disabledWrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.oauth.completeAuth'))

    expect(disabledButton?.exists()).toBe(true)
    expect(disabledButton?.attributes('disabled')).toBeDefined()

    const wrapper = mount(AccountCreateFooterActions, {
      props: {
        step: 2,
        submitting: false,
        isOAuthFlow: true,
        isManualInputMethod: false,
        currentOAuthLoading: false,
        canExchangeCode: false,
        showCompleteAuthAction: true,
        canCompleteAuth: true,
        autoImportModels: false
      }
    })

    const completeButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.oauth.completeAuth'))

    await completeButton?.trigger('click')

    expect(wrapper.emitted('completeAuth')).toEqual([[]])
    expect(wrapper.emitted('exchangeCode')).toBeUndefined()
  })
})
