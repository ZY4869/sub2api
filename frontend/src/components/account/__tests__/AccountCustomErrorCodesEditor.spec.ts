import { mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountCustomErrorCodesEditor from '../AccountCustomErrorCodesEditor.vue'
import { createDefaultAccountCustomErrorCodesState } from '@/utils/accountApiKeyAdvancedSettings'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountCustomErrorCodesEditor', () => {
  const showError = vi.fn()
  const showInfo = vi.fn()

  beforeEach(() => {
    showError.mockReset()
    showInfo.mockReset()
    vi.stubGlobal('confirm', vi.fn(() => true))
  })

  it('toggles common error codes and supports manual input', async () => {
    const state = reactive(createDefaultAccountCustomErrorCodesState())
    const wrapper = mount(AccountCustomErrorCodesEditor, {
      props: {
        state,
        errorCodeOptions: [
          { value: 401, label: 'Unauthorized' },
          { value: 429, label: 'Rate Limit' }
        ],
        showError,
        showInfo
      }
    })

    await wrapper.find('button').trigger('click')
    expect(state.enabled).toBe(true)

    const toggle429 = wrapper.findAll('button').find((button) => button.text().includes('429'))
    expect(toggle429).toBeTruthy()
    await toggle429?.trigger('click')
    expect(state.selectedCodes).toContain(429)

    const input = wrapper.find('input')
    await input.setValue('500')
    const addButton = wrapper.findAll('button').find((button) =>
      (button.attributes('class') || '').includes('btn-secondary')
    )
    await addButton?.trigger('click')
    expect(state.selectedCodes).toContain(500)
  })

  it('shows validation feedback for invalid or duplicate input', async () => {
    const state = reactive(createDefaultAccountCustomErrorCodesState())
    state.enabled = true
    state.selectedCodes = [500]

    const wrapper = mount(AccountCustomErrorCodesEditor, {
      props: {
        state,
        errorCodeOptions: [{ value: 500, label: 'Server Error' }],
        showError,
        showInfo
      }
    })

    await wrapper.find('input').setValue('99')
    await wrapper.findAll('button').find((button) =>
      (button.attributes('class') || '').includes('btn-secondary')
    )?.trigger('click')
    expect(showError).toHaveBeenCalledWith('admin.accounts.invalidErrorCode')

    await wrapper.find('input').setValue('500')
    await wrapper.findAll('button').find((button) =>
      (button.attributes('class') || '').includes('btn-secondary')
    )?.trigger('click')
    expect(showInfo).toHaveBeenCalledWith('admin.accounts.errorCodeExists')
  })
})
